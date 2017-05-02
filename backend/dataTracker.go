package backend

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"text/template"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/dgrijalva/jwt-go"
	"github.com/digitalrebar/digitalrebar/go/common/store"
)

var (
	ignoreBoot = &BootEnv{
		Name:        "ignore",
		Description: "The boot environment you should use to have unknown machines boot off their local hard drive",
		OS:          OsInfo{Name: "ignore"},
		OnlyUnknown: true,
		Templates: []TemplateInfo{
			{
				Name: "pxelinux",
				Path: "pxelinux.cfg/default",
				Contents: `DEFAULT local
PROMPT 0
TIMEOUT 10
LABEL local
localboot 0
`,
			},
			{
				Name:     "elilo",
				Path:     "elilo.conf",
				Contents: "exit",
			},
			{
				Name: "ipxe",
				Path: "default.ipxe",
				Contents: `#!ipxe
chain tftp://{{.ProvisionerAddress}}/${netX/ip}.ipxe || exit
`,
			},
		},
	}
)

type followUpSaver interface {
	followUpSave()
}

type followUpDeleter interface {
	followUpDelete()
}

// dtobjs is an in-memory cache of all the objects we could
// reference. The implementation of this may need to change from
// storing a slice of things to a more elaborate datastructure at some
// point in time.  Since that point in time is when the slices are
// forced out of CPU cache, I am not terribly concerned for now.
// Until that point is reached, sorting and searching slices is
// fantastically efficient.
type dtobjs struct {
	sync.Mutex
	d []store.KeySaver
}

// sort is only used when loading data from the backingStore at
// startup time.
func (dt *dtobjs) sort() {
	sort.Slice(dt.d, func(i, j int) bool { return dt.d[i].Key() < dt.d[j].Key() })
}

// find assumes that d is sorted.  It returns the index where
// something with the key should be inserted at, along with a flag
// indicating whether the object currently at that index has the
// passed-in key.
func (dt *dtobjs) find(key string) (int, bool) {
	idx := sort.Search(len(dt.d), func(i int) bool { return dt.d[i].Key() >= key })
	return idx, idx < len(dt.d) && dt.d[idx].Key() == key
}

// subset returns a copy of part of the cached data, based on the
// search functions passed into subset.
func (dt *dtobjs) subset(lower, upper func(string) bool) []store.KeySaver {
	i := sort.Search(len(dt.d), func(i int) bool { return lower(dt.d[i].Key()) })
	j := sort.Search(len(dt.d), func(i int) bool { return upper(dt.d[i].Key()) })
	if i == len(dt.d) {
		return []store.KeySaver{}
	}
	res := make([]store.KeySaver, j-i)
	copy(res, dt.d[i:])
	return res
}

// add adds a single object to the object cache in such a way that the
// underlying slice remains sorted without actually needing to resort
// the slice.
func (dt *dtobjs) add(obj store.KeySaver) {
	idx, found := dt.find(obj.Key())
	if found {
		dt.d[idx] = obj
		return
	}
	if idx == len(dt.d) {
		dt.d = append(dt.d, obj)
		return
	}
	// Grow by one, amortized by append()
	dt.d = append(dt.d, nil)
	copy(dt.d[idx+1:], dt.d[idx:])
	dt.d[idx] = obj
}

// remove removes the specified entries from the underlying slice
// while maintaining the overall sort order and minimizing the amount
// of data that needs to be moved.
func (dt *dtobjs) remove(idxs ...int) {
	if len(idxs) == 0 {
		return
	}
	sort.Ints(idxs)
	lastDT := len(dt.d)
	lastIdx := len(idxs) - 1
	// Progressively copy over slices to overwrite entries we are
	// deleting
	for i, idx := range idxs {
		if idx == lastDT-1 {
			continue
		}
		var srcend int
		if i != lastIdx {
			srcend = idxs[i+1]
		} else {
			srcend = lastDT
		}
		// copy(dest, src)
		copy(dt.d[idx-i:srcend], dt.d[idx+1:srcend])
	}
	// Nil out entries that we should garbage collect.  We do this
	// so that we don't wind up leaking items based on the
	// underlying arrays still pointing at things we no longer
	// care about.
	for i := range idxs {
		dt.d[lastDT-i-1] = nil
	}
	// Resize dt.d to forget about the last elements.  This does
	// not always resize the underlying array, hence the above
	// manual GC enablement.
	//
	// At some point we may want to manually switch to a smaller
	// underlying array based on len() vs. cap() for dt.d, but
	// probably only when we can potentially free a significant
	// amount of memory by doing so.
	dt.d = dt.d[:len(dt.d)-len(idxs)]
}

type dtSetter interface {
	store.KeySaver
	setDT(*DataTracker)
}

// DataTracker represents everything there is to know about acting as
// a dataTracker.
type DataTracker struct {
	FileRoot            string
	CommandURL          string
	OurAddress          string
	StaticPort, ApiPort int
	Logger              *log.Logger
	FS                  *FileSystem
	// Note the lack of mutexes for these maps.
	// We should be able to get away with not locking them
	// by only ever writing to them at DataTracker create time,
	// and only ever reading from them afterwards.
	backends          map[string]store.SimpleStore
	objs              map[string]*dtobjs
	defaultPrefs      map[string]string
	defaultBootEnv    string
	globalProfileName string
	tokenManager      *JwtManager
	rootTemplate      *template.Template
	tmplMux           *sync.Mutex
	thunks            []func()
	thunkMux          *sync.Mutex
}

// This grabs the requested dtobj locks in reverse alphabetical
// order to give everything a consistent locking order and allow
// for templates to lock bootenvs when a template change necessitates
// a bootenv template cache rebuild.
func (p *DataTracker) lockEnts(ents ...string) ([]*dtobjs, func()) {
	res := make([]*dtobjs, len(ents))
	s := sort.StringSlice(ents)
	sort.Sort(sort.Reverse(s))
	for i := range s {
		res[i] = p.lockFor(ents[i])
	}
	unlocker := func() {
		for i := len(res) - 1; i >= 0; i-- {
			res[i].Unlock()
		}
	}
	return res, unlocker
}

func (p *DataTracker) LocalIP(remote net.IP) string {
	if localIP := LocalFor(remote); localIP != nil {
		return localIP.String()
	}
	// Determining what this is needs to be made smarter, probably by
	// firguing out which interface the default route goes over for ipv4
	// then ipv6, and then figurig out the appropriate address on that
	// interface
	return p.OurAddress
}

func (p *DataTracker) urlFor(scheme string, remoteIP net.IP, port int) string {
	return fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(p.LocalIP(remoteIP), strconv.Itoa(port)))
}

func (p *DataTracker) FileURL(remoteIP net.IP) string {
	return p.urlFor("http", remoteIP, p.StaticPort)
}

func (p *DataTracker) ApiURL(remoteIP net.IP) string {
	return p.urlFor("https", remoteIP, p.ApiPort)
}

func (p *DataTracker) lockFor(prefix string) *dtobjs {
	res := p.objs[prefix]
	res.Lock()
	return res
}

func (p *DataTracker) makeBackends(backend store.SimpleStore, objs []store.KeySaver) {
	for _, o := range objs {
		prefix := o.Prefix()
		bk, err := backend.Sub(prefix)
		if err != nil {
			p.Logger.Fatalf("dataTracker: Error creating substore %s: %v", prefix, err)
		}
		p.backends[prefix] = bk
	}
}

func (p *DataTracker) loadData(refObj store.KeySaver) {
	prefix := refObj.Prefix()
	objs, err := store.List(refObj)
	if err != nil {
		p.Logger.Fatalf("dataTracker: Error loading data for %s: %v", prefix, err)
	}
	p.objs[prefix] = &dtobjs{d: objs}
	p.objs[prefix].sort()
}

// Create a new DataTracker that will use passed store to save all operational data
func NewDataTracker(backend store.SimpleStore,
	fileRoot, addr string,
	staticPort, apiPort int,
	logger *log.Logger,
	defaultPrefs map[string]string) *DataTracker {
	res := &DataTracker{
		FileRoot:          fileRoot,
		StaticPort:        staticPort,
		ApiPort:           apiPort,
		OurAddress:        addr,
		Logger:            logger,
		backends:          map[string]store.SimpleStore{},
		defaultPrefs:      defaultPrefs,
		FS:                NewFS(fileRoot, logger),
		tokenManager:      NewJwtManager([]byte(randString(32)), JwtConfig{Method: jwt.SigningMethodHS256}),
		tmplMux:           &sync.Mutex{},
		globalProfileName: "global",
		thunks:            make([]func(), 0),
		thunkMux:          &sync.Mutex{},
	}
	objs := []store.KeySaver{
		&Machine{p: res},
		&Profile{p: res},
		&User{p: res},
		&Template{p: res},
		&BootEnv{p: res},
		&Subnet{p: res},
		&Reservation{p: res},
		&Lease{p: res},
		&Pref{p: res},
	}
	res.makeBackends(backend, objs)
	res.objs = map[string]*dtobjs{}
	for _, obj := range objs {
		res.loadData(obj)
		if obj.Prefix() == "templates" {
			buf := &bytes.Buffer{}
			for _, thing := range res.objs["templates"].d {
				tmpl := AsTemplate(thing)
				fmt.Fprintf(buf, `{{define "%s"}}%s{{end}}`, tmpl.ID, tmpl.Contents)
			}
			root, err := template.New("").Parse(buf.String())
			if err != nil {
				logger.Fatalf("Unable to load root templates: %v", err)
			}
			res.rootTemplate = root
			res.rootTemplate.Option("missingkey=error")
		}
	}
	if _, ok := res.fetchOne(ignoreBoot, ignoreBoot.Name); !ok {
		res.Save(ignoreBoot)
	}
	if _, ok := res.fetchOne(res.NewProfile(), res.globalProfileName); !ok {
		gp := AsProfile(res.NewProfile())
		gp.Name = "global"
		res.Save(gp)
	}
	users := res.objs["users"]
	if len(users.d) == 0 {
		logger.Printf("Creating rocketskates user")
		user := &User{p: res, Name: "rocketskates"}
		if err := user.ChangePassword("r0cketsk8ts"); err != nil {
			logger.Fatalf("Failed to create rocketskates user: %v", err)
		}
		users.add(user)
	}
	res.defaultBootEnv = defaultPrefs["defaultBootEnv"]
	machines := res.lockFor("machines")
	for i := range machines.d {
		machine := AsMachine(machines.d[i])
		be, found := res.fetchOne(res.NewBootEnv(), machine.BootEnv)
		if !found {
			continue
		}
		err := &Error{o: machine}
		AsBootEnv(be).Render(machine, err).register(res.FS)
		if err.containsError {
			logger.Printf("Error rendering machine %s at startup:", machine.UUID())
			logger.Println(err.Error())
		}
	}
	machines.Unlock()
	return res
}

func (p *DataTracker) Pref(name string) (string, error) {
	prefIsh := p.load("preferences", name)
	if prefIsh == nil {
		val, ok := p.defaultPrefs[name]
		if ok {
			return val, nil
		}
		return "", fmt.Errorf("No such preference %s", name)
	}
	pref := AsPref(prefIsh)
	return pref.Val, nil
}

func (p *DataTracker) Prefs() map[string]string {
	vals := map[string]string{}
	for k, v := range p.defaultPrefs {
		vals[k] = v
	}
	prefs := p.lockFor("preferences")
	defer prefs.Unlock()
	for i := range prefs.d {
		pref := AsPref(prefs.d[i])
		vals[pref.Name] = pref.Val
	}
	return vals
}

func (p *DataTracker) SetPrefs(prefs map[string]string) error {
	err := &Error{}
	benvCheck := func(name, val string) *BootEnv {
		be := p.load("bootenvs", val)
		if be == nil {
			err.Errorf("%s: Bootenv %s does not exist", name, val)
			return nil
		}
		return AsBootEnv(be)
	}
	intCheck := func(name, val string) bool {
		_, e := strconv.Atoi(val)
		if e == nil {
			return true
		}
		err.Errorf("%s: %s", name, e.Error())
		return false
	}
	savePref := func(name, val string) bool {
		pref := &Pref{p: p, Name: name, Val: val}
		if _, saveErr := p.save(pref); saveErr != nil {
			err.Errorf("%s: Failed to save %s: %v", name, val, saveErr)
			return false
		}
		return true
	}
	for name, val := range prefs {
		switch name {
		case "defaultBootEnv":
			be := benvCheck(name, val)
			if be != nil && !be.OnlyUnknown {
				savePref(name, val)
				p.defaultBootEnv = val
			}
		case "unknownBootEnv":
			if benvCheck(name, val) != nil && savePref(name, val) {
				err.Merge(p.RenderUnknown())
			}
		case "unknownTokenTimeout":
			if intCheck(name, val) {
				savePref(name, val)
			}
			continue
		case "knownTokenTimeout":
			if intCheck(name, val) {
				savePref(name, val)
			}
			continue
		default:
			err.Errorf("Unknown preference %s", name)
		}
	}
	return err.OrNil()
}

func (p *DataTracker) RenderUnknown() error {
	pref, e := p.Pref("unknownBootEnv")
	if e != nil {
		return e
	}
	envIsh := p.load("bootenvs", pref)
	if envIsh == nil {
		return fmt.Errorf("No such BootEnv: %s", pref)
	}
	env := AsBootEnv(envIsh)
	err := &Error{o: env, Type: "StartupError"}
	if !env.Available {
		err.Messages = env.Errors
		err.containsError = true
		return err
	}
	if !env.OnlyUnknown {
		err.Errorf("BootEnv %s cannot be used for the unknownBootEnv", env.Name)
		return err
	}
	env.Render(nil, err).register(p.FS)
	return err.OrNil()
}

// Load should only be used by tests and initialization
func (p *DataTracker) load(prefix, key string) store.KeySaver {
	objs, idx, found := p.lockedGet(prefix, key)
	defer objs.Unlock()
	if found {
		return objs.d[idx]
	}
	return nil
}

func (p *DataTracker) getBackend(t store.KeySaver) store.SimpleStore {
	res, ok := p.backends[t.Prefix()]
	if !ok {
		p.Logger.Fatalf("%s: No registered storage backend!", t.Prefix())
	}
	return res
}

func (p *DataTracker) setDT(s store.KeySaver) {
	if tgt, ok := s.(dtSetter); ok {
		tgt.setDT(p)
	}
}

func (p *DataTracker) Clone(ref store.KeySaver) store.KeySaver {
	var res store.KeySaver
	switch ref.(type) {
	case *Machine:
		res = p.NewMachine()
	case *Profile:
		res = p.NewProfile()
	case *User:
		res = p.NewUser()
	case *Template:
		res = p.NewTemplate()
	case *BootEnv:
		res = p.NewBootEnv()
	case *Lease:
		res = p.NewLease()
	case *Reservation:
		res = p.NewReservation()
	case *Subnet:
		res = p.NewSubnet()
	case *Pref:
		res = p.NewPref()
	default:
		panic("Unknown type of KeySaver passed to Clone")
	}
	buf, err := json.Marshal(ref)
	if err != nil {
		panic(err.Error())
	}
	if err := json.Unmarshal(buf, &res); err != nil {
		panic(err.Error())
	}
	return res
}

// unlockedFetchAll gets all the objects with a given prefix without
// taking any locks.  It should only be called in hooks that need to
// check for object uniqueness based on something besides a Key()
func (p *DataTracker) unlockedFetchAll(prefix string) []store.KeySaver {
	return p.objs[prefix].d
}

// fetchSome returns all of the objects in one of our caches in
// between the point where lower starts to match its Search and upper
// starts to match its Search.  The lower and upper parameters must be
// functions that accept a Key() and return a yes or no decision about
// whether that particular entry is in range.
func (p *DataTracker) fetchSome(prefix string, lower, upper func(string) bool) []store.KeySaver {
	mux := p.lockFor(prefix)
	defer mux.Unlock()
	return mux.subset(lower, upper)
}

// fetchAll returns all the instances we know about, It differs from FetchAll in that
// it does not make a copy of the thing.
func (p *DataTracker) fetchAll(ref store.KeySaver) []store.KeySaver {
	prefix := ref.Prefix()
	mux := p.lockFor(prefix)
	defer mux.Unlock()
	res := make([]store.KeySaver, len(mux.d))
	copy(res, mux.d)
	return res
}

// FetchAll returns all of the cached objects of a given type.  It
// should be used instead of store.List.
func (p *DataTracker) FetchAll(ref store.KeySaver) []store.KeySaver {
	prefix := ref.Prefix()
	res := p.lockFor(prefix)
	ret := make([]store.KeySaver, len(res.d))
	for i := range res.d {
		ret[i] = p.Clone(res.d[i])
	}
	res.Unlock()
	return ret
}

func (p *DataTracker) lockedGet(prefix, key string) (*dtobjs, int, bool) {
	mux := p.lockFor(prefix)
	idx, found := mux.find(key)
	return mux, idx, found
}

func (p *DataTracker) fetchOne(ref store.KeySaver, key string) (store.KeySaver, bool) {
	prefix := ref.Prefix()
	mux, idx, found := p.lockedGet(prefix, key)
	defer mux.Unlock()
	if found {
		return mux.d[idx], found
	}
	return nil, found
}

// FetchOne returns a specific instance from the cached objects of
// that type.  It should be used instead of store.Load.
func (p *DataTracker) FetchOne(ref store.KeySaver, key string) (store.KeySaver, bool) {
	res, found := p.fetchOne(ref, key)
	if !found {
		return nil, found
	}
	return p.Clone(res), found
}

func (p *DataTracker) create(ref store.KeySaver) (bool, error) {
	p.setDT(ref)
	prefix := ref.Prefix()
	key := ref.Key()
	if key == "" {
		return false, fmt.Errorf("dataTracker create %s: Empty key not allowed", prefix)
	}
	mux, _, found := p.lockedGet(prefix, key)
	defer mux.Unlock()
	if found {
		return false, fmt.Errorf("dataTracker create %s: %s already exists", prefix, key)
	}
	saved, err := store.Create(ref)
	if saved {
		mux.add(ref)
	}
	return saved, err
}

// Create creates a new thing, caching it locally iff the create
// succeeds.  It should be used instead of store.Create
func (p *DataTracker) Create(ref store.KeySaver) (store.KeySaver, error) {
	created, err := p.create(ref)
	if created {
		if fs, ok := ref.(followUpSaver); ok {
			fs.followUpSave()
		}
		return p.Clone(ref), err
	}
	return ref, err
}

func (p *DataTracker) remove(ref store.KeySaver) (bool, error) {
	prefix := ref.Prefix()
	key := ref.Key()
	mux, idx, found := p.lockedGet(prefix, key)
	defer mux.Unlock()
	if !found {
		err := &Error{
			Code:  http.StatusNotFound,
			Key:   key,
			Model: prefix,
		}
		err.Errorf("%s: DELETE %s: Not Found", err.Model, err.Key)
		return false, err
	}
	removed, err := store.Remove(mux.d[idx])
	if removed {
		mux.remove(idx)
	}
	return removed, err
}

// Remove removes the thing from the backing store.  If the remove
// succeeds, it will also be removed from the local cache.  It should
// be used instead of store.Remove
func (p *DataTracker) Remove(ref store.KeySaver) (store.KeySaver, error) {
	removed, err := p.remove(ref)
	if !removed {
		return p.Clone(ref), err
	}
	if fs, ok := ref.(followUpDeleter); ok {
		fs.followUpDelete()
	}

	return ref, err
}

func (p *DataTracker) Patch(ref store.KeySaver, key string, patch jsonpatch2.Patch) (store.KeySaver, error) {
	prefix := ref.Prefix()
	mux, idx, found := p.lockedGet(prefix, key)
	defer mux.Unlock()
	if !found {
		err := &Error{
			Code:  http.StatusNotFound,
			Key:   key,
			Model: prefix,
		}
		err.Errorf("%s: PATCH %s: Not Found", err.Model, err.Key)
		return nil, err
	}
	target := mux.d[idx]
	buf, fatalErr := json.Marshal(target)
	if fatalErr != nil {
		p.Logger.Fatalf("Non-JSON encodable %v:%v stored in cache: %v", prefix, key, fatalErr)
	}

	resBuf, patchErr, loc := patch.Apply(buf)
	if patchErr == nil {
		toSave := ref.New()
		if err := json.Unmarshal(resBuf, &toSave); err != nil {
			return nil, err
		}
		p.setDT(toSave)
		saved, err := store.Update(toSave)
		if !saved {
			return toSave, err
		}
		mux.d[idx] = toSave
		if fs, ok := ref.(followUpSaver); ok {
			fs.followUpSave()
		}
		return p.Clone(toSave), nil
	}
	err := &Error{
		Code:  http.StatusNotAcceptable,
		Key:   key,
		Model: ref.Prefix(),
		Type:  "JsonPatchError",
	}
	err.Errorf("Patch error at line %d: %v", loc, patchErr)
	return nil, err
}

func (p *DataTracker) update(ref store.KeySaver) (bool, error) {
	prefix := ref.Prefix()
	key := ref.Key()
	mux, idx, found := p.lockedGet(prefix, key)
	defer mux.Unlock()
	if !found {
		err := &Error{
			Code:  http.StatusNotFound,
			Key:   key,
			Model: prefix,
		}
		err.Errorf("%s: PUT %s: Not Found", err.Model, err.Key)
		return false, err
	}
	p.setDT(ref)
	ok, err := store.Update(ref)
	if ok {
		mux.d[idx] = ref
	}
	return ok, err
}

// Update updates the passed thing, and updates the local cache iff
// the update succeeds.  It should be used in preference to
// store.Update
func (p *DataTracker) Update(ref store.KeySaver) (store.KeySaver, error) {
	updated, err := p.update(ref)
	if updated {
		if fs, ok := ref.(followUpSaver); ok {
			fs.followUpSave()
		}
		return p.Clone(ref), err
	}
	return ref, err
}

func (p *DataTracker) save(ref store.KeySaver) (bool, error) {
	prefix := ref.Prefix()
	key := ref.Key()
	mux, idx, found := p.lockedGet(prefix, key)
	defer mux.Unlock()
	p.setDT(ref)
	ok, err := store.Save(ref)
	if !ok {
		return ok, err
	}
	if found {
		mux.d[idx] = ref
	} else {
		mux.add(ref)
	}
	return ok, err
}

// Save saves the passed thing, updating the local cache iff the save
// succeeds.  It should be used instead of store.Save
func (p *DataTracker) Save(ref store.KeySaver) (store.KeySaver, error) {
	saved, err := p.save(ref)
	if saved {
		if fs, ok := ref.(followUpSaver); ok {
			fs.followUpSave()
		}
		return p.Clone(ref), err
	}
	return ref, err
}

func (p *DataTracker) GetToken(tokenString string) (*DrpCustomClaims, error) {
	return p.tokenManager.get(tokenString)
}

func (p *DataTracker) SealClaims(claims *DrpCustomClaims) (string, error) {
	return claims.Seal(p.tokenManager)
}

func (p *DataTracker) NewToken(id string, ttl int, scope, action, specific string) (string, error) {
	return NewClaim(id, ttl).Add(scope, action, specific).Seal(p.tokenManager)
}

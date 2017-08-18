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
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

var (
	ignoreBoot = &models.BootEnv{
		Name:        `ignore`,
		Description: "The boot environment you should use to have unknown machines boot off their local hard drive",
		OS: models.OsInfo{
			Name: `ignore`,
		},
		OnlyUnknown: true,
		Templates: []models.TemplateInfo{
			{
				Name: "pxelinux",
				Path: `pxelinux.cfg/default`,
				Contents: `DEFAULT local
PROMPT 0
TIMEOUT 10
LABEL local
localboot 0
`,
			},
			{
				Name:     `elilo`,
				Path:     `elilo.conf`,
				Contents: `exit`,
			},
			{
				Name: `ipxe`,
				Path: `default.ipxe`,
				Contents: `#!ipxe
exit
`,
			},
		},
	}

	localBoot = &BootEnv{
		Name:        "local",
		Description: "The boot environment you should use to have known machines boot off their local hard drive",
		OS:          OsInfo{Name: "local"},
		OnlyUnknown: false,
		Templates: []TemplateInfo{
			{
				Name: "pxelinux",
				Path: "pxelinux.cfg/{{.Machine.HexAddress}}",
				Contents: `DEFAULT local
PROMPT 0
TIMEOUT 10
LABEL local
localboot 0
`,
			},
			{
				Name:     "elilo",
				Path:     "{{.Machine.HexAddress}}.conf",
				Contents: "exit",
			},
			{
				Name: "ipxe",
				Path: "{{.Machine.Address}}.ipxe",
				Contents: `#!ipxe
exit
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

type AuthSaver interface {
	AuthKey() string
}

// dtobjs is an in-memory cache of all the objects we could
// reference. The implementation of this may need to change from
// storing a slice of things to a more elaborate datastructure at some
// point in time.  Since that point in time is when the slices are
// forced out of CPU cache, I am not terribly concerned for now.
// Until that point is reached, sorting and searching slices is
// fantastically efficient.
type Store struct {
	sync.Mutex
	index.Index
	backingStore store.Store
}

type ObjectValidator func(Stores, models.Model, models.Model) error

func (s *Store) getBackend(obj models.Model) store.Store {
	return s.backingStore
}

type dtSetter interface {
	models.Model
	setDT(*DataTracker)
}

func Fill(t store.KeySaver) {
	switch obj := t.(type) {
	case *BootEnv:
		obj.BootEnv = &models.BootEnv{}
	case *Job:
		obj.Job = &models.Job{}
	case *Lease:
		obj.Lease = &models.Lease{}
	case *Machine:
		obj.Machine = &models.Machine{}
	case *Param:
		obj.Param = &models.Param{}
	case *Plugin:
		obj.Plugin = &models.Plugin{}
	case *Profile:
		obj.Profile = &models.Profile{}
	case *Reservation:
		obj.Reservation = &models.Reservation{}
	case *Subnet:
		obj.Subnet = &models.Subnet{}
	case *Task:
		obj.Task = &models.Task{}
	case *Template:
		obj.Template = &models.Template{}
	case *User:
		obj.User = &models.User{}
	default:
		panic(fmt.Sprintf("Unknown backend model %T", t))
	}
}

func toBackend(p *DataTracker, s Stores, m models.Model) store.KeySaver {
	if res, ok := m.(store.KeySaver); ok {
		p.setDT(res)
		return res
	}
	var ours store.KeySaver
	if s != nil {
		backend := s(m.Prefix())
		if backend == nil {
			p.Logger.Panicf("No store for %T", m)
		}
		if this := backend.Find(m.Key()); this != nil {
			ours = this.(store.KeySaver)
		}
	}

	switch obj := m.(type) {
	case *models.BootEnv:
		var res BootEnv
		if ours != nil {
			res = *ours.(*BootEnv)
		} else {
			res = BootEnv{}
		}
		res.BootEnv = obj
		p.setDT(&res)
		return &res
	case *models.Job:
		var res Job
		if ours != nil {
			res = *ours.(*Job)
		} else {
			res = Job{}
		}
		res.Job = obj
		p.setDT(&res)
		return &res
	case *models.Lease:
		var res Lease
		if ours != nil {
			res = *ours.(*Lease)
		} else {
			res = Lease{}
		}
		res.Lease = obj
		p.setDT(&res)
		return &res
	case *models.Machine:
		var res Machine
		if ours != nil {
			res = *ours.(*Machine)
		} else {
			res = Machine{}
		}
		res.Machine = obj
		p.setDT(&res)
		return &res
	case *models.Param:
		var res Param
		if ours != nil {
			res = *ours.(*Param)
		} else {
			res = Param{}
		}
		res.Param = obj
		p.setDT(&res)
		return &res
	case *models.Plugin:
		var res Plugin
		if ours != nil {
			res = *ours.(*Plugin)
		} else {
			res = Plugin{}
		}
		res.Plugin = obj
		p.setDT(&res)
		return &res
	case *models.Pref:
		var res Pref
		if ours != nil {
			res = *ours.(*Pref)
		} else {
			res = Pref{}
		}
		res.Pref = obj
		p.setDT(&res)
		return &res
	case *models.Profile:
		var res Profile
		if ours != nil {
			res = *ours.(*Profile)
		} else {
			res = Profile{}
		}
		res.Profile = obj
		p.setDT(&res)
		return &res
	case *models.Reservation:
		var res Reservation
		if ours != nil {
			res = *ours.(*Reservation)
		} else {
			res = Reservation{}
		}
		res.Reservation = obj
		p.setDT(&res)
		return &res
	case *models.Subnet:
		var res Subnet
		if ours != nil {
			res = *ours.(*Subnet)
		} else {
			res = Subnet{}
		}
		res.Subnet = obj
		p.setDT(&res)
		return &res
	case *models.Task:
		var res Task
		if ours != nil {
			res = *ours.(*Task)
		} else {
			res = Task{}
		}
		res.Task = obj
		p.setDT(&res)
		return &res
	case *models.Template:
		var res Template
		if ours != nil {
			res = *ours.(*Template)
		} else {
			res = Template{}
		}
		res.Template = obj
		p.setDT(&res)
		return &res

	case *models.User:
		var res User
		if ours != nil {
			res = *ours.(*User)
		} else {
			res = User{}
		}
		res.User = obj
		p.setDT(&res)
		return &res

	default:
		p.Logger.Panicf("Unknown model %T", m)
	}
	return nil
}

// DataTracker represents everything there is to know about acting as
// a dataTracker.
type DataTracker struct {
	FileRoot            string
	LogRoot             string
	OurAddress          string
	StaticPort, ApiPort int
	Logger              *log.Logger
	FS                  *FileSystem
	Backend             store.Store
	objs                map[string]*Store
	defaultPrefs        map[string]string
	runningPrefs        map[string]string
	prefMux             *sync.Mutex
	allMux              *sync.RWMutex
	defaultBootEnv      string
	GlobalProfileName   string
	tokenManager        *JwtManager
	rootTemplate        *template.Template
	tmplMux             *sync.Mutex
	thunks              []func()
	thunkMux            *sync.Mutex
	publishers          *Publishers
}

type Stores func(string) *Store

func allKeySavers(res *DataTracker) []models.Model {
	return []models.Model{
		&Pref{p: res},
		&Param{p: res},
		&Profile{p: res},
		&User{p: res},
		&Template{p: res},
		&Task{p: res},
		&BootEnv{p: res},
		&Machine{p: res},
		&Subnet{p: res},
		&Reservation{p: res},
		&Lease{p: res},
		&Plugin{p: res},
		&Job{p: res},
	}
}

// LockEnts grabs the requested Store locks a consistent order.
// It returns a function to get an Index that was requested, and
// a function that unlocks the taken locks in the right order.
func (p *DataTracker) LockEnts(ents ...string) (stores Stores, unlocker func()) {
	p.allMux.RLock()
	sortedEnts := make([]string, len(ents))
	copy(sortedEnts, ents)
	s := sort.StringSlice(sortedEnts)
	sort.Sort(sort.Reverse(s))
	sortedRes := map[string]*Store{}
	for _, ent := range s {
		objs, ok := p.objs[ent]
		if !ok {
			log.Panicf("Tried to reference nonexistent object type '%s'", ent)
		}
		sortedRes[ent] = objs
	}
	for _, ent := range s {
		sortedRes[ent].Lock()
	}
	srMux := &sync.Mutex{}
	return func(ref string) *Store {
			srMux.Lock()
			idx, ok := sortedRes[ref]
			srMux.Unlock()
			if !ok {
				log.Panicf("Tried to access unlocked resource %s", ref)
			}
			return idx
		},
		func() {
			srMux.Lock()
			for i := len(s) - 1; i >= 0; i-- {
				sortedRes[s[i]].Unlock()
				delete(sortedRes, s[i])
			}
			srMux.Unlock()
			p.allMux.RUnlock()
		}
}

func (p *DataTracker) LockAll() (stores Stores, unlocker func()) {
	p.allMux.Lock()
	return func(ref string) *Store {
			return p.objs[ref]
		},
		func() {
			p.allMux.Unlock()
		}
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

func (p *DataTracker) rebuildCache() error {
	p.objs = map[string]*Store{}
	objs := allKeySavers(p)
	for _, obj := range objs {
		prefix := obj.Prefix()
		bk := p.Backend.GetSub(prefix)
		p.objs[prefix] = &Store{backingStore: bk}
		storeObjs, err := store.List(bk, toBackend(p, nil, obj))
		if err != nil {
			return fmt.Errorf("%s: %v", prefix, err)
		}
		res := make([]models.Model, len(storeObjs))
		for i := range storeObjs {
			p.setDT(storeObjs[i])
			res[i] = models.Model(storeObjs[i])
		}
		p.objs[prefix].Index = *index.Create(res)
		if obj.Prefix() == "templates" {
			buf := &bytes.Buffer{}
			for _, thing := range p.objs["templates"].Items() {
				tmpl := AsTemplate(thing)
				fmt.Fprintf(buf, `{{define "%s"}}%s{{end}}`, tmpl.ID, tmpl.Contents)
			}
			root, err := template.New("").Parse(buf.String())
			if err != nil {
				return fmt.Errorf("Unable to load root templates: %v", err)
			}
			p.rootTemplate = root
			p.rootTemplate.Option("missingkey=error")
		}
	}
	return nil
}

func ValidateDataTrackerStore(backend store.Store, logger *log.Logger) error {
	res := &DataTracker{
		Backend:           backend,
		FileRoot:          ".",
		LogRoot:           ".",
		StaticPort:        1,
		ApiPort:           2,
		OurAddress:        "1.1.1.1",
		Logger:            logger,
		defaultPrefs:      map[string]string{},
		runningPrefs:      map[string]string{},
		prefMux:           &sync.Mutex{},
		allMux:            &sync.RWMutex{},
		FS:                NewFS(".", logger),
		tokenManager:      NewJwtManager([]byte(randString(32)), JwtConfig{Method: jwt.SigningMethodHS256}),
		tmplMux:           &sync.Mutex{},
		GlobalProfileName: "global",
		thunks:            make([]func(), 0),
		thunkMux:          &sync.Mutex{},
		publishers:        &Publishers{},
	}

	// Load stores.
	err := res.rebuildCache()
	if err != nil {
		return models.NewError("LoadError", http.StatusInternalServerError, fmt.Sprintf("Failed to rebuild cache: %v", err))
	}

	keys := make([]string, len(res.objs))
	i := 0
	for k := range res.objs {
		keys[i] = k
		i++
	}

	d, unlocker := res.LockAll()
	defer unlocker()

	berr := &models.Error{Code: http.StatusUnprocessableEntity, Type: ValidationError}

	for _, k := range keys {

		for _, obj := range res.objs[k].Items() {
			if val, ok := obj.(Validator); ok {
				obj.(validator).setStores(d)
				val.ClearValidation()
				val.Validate()
				if !val.Useable() {
					berr.AddError(val.HasError())
				}
			}
		}
	}
	return berr.HasError()
}

// Create a new DataTracker that will use passed store to save all operational data
func NewDataTracker(backend store.Store,
	fileRoot, logRoot, addr string,
	staticPort, apiPort int,
	logger *log.Logger,
	defaultPrefs map[string]string,
	publishers *Publishers) *DataTracker {
	res := &DataTracker{
		Backend:           backend,
		FileRoot:          fileRoot,
		LogRoot:           logRoot,
		StaticPort:        staticPort,
		ApiPort:           apiPort,
		OurAddress:        addr,
		Logger:            logger,
		defaultPrefs:      defaultPrefs,
		runningPrefs:      map[string]string{},
		prefMux:           &sync.Mutex{},
		allMux:            &sync.RWMutex{},
		FS:                NewFS(fileRoot, logger),
		tokenManager:      NewJwtManager([]byte(randString(32)), JwtConfig{Method: jwt.SigningMethodHS256}),
		tmplMux:           &sync.Mutex{},
		GlobalProfileName: "global",
		thunks:            make([]func(), 0),
		thunkMux:          &sync.Mutex{},
		publishers:        publishers,
	}

	// Make sure incoming writable backend has all stores created
	objs := allKeySavers(res)
	for _, obj := range objs {
		prefix := obj.Prefix()
		_, err := backend.MakeSub(prefix)
		if err != nil {
			res.Logger.Fatalf("dataTracker: Error creating substore %s: %v", prefix, err)
		}
	}

	// Load stores.
	err := res.rebuildCache()
	if err != nil {
		res.Logger.Fatalf("dataTracker: Error loading data: %v", err)
	}

	// Create minimal content.
	d, unlocker := res.LockEnts("bootenvs", "preferences", "users", "machines", "profiles", "params")
	defer unlocker()
	if d("bootenvs").Find("ignore") == nil {
		res.Create(d, ignoreBoot)
	}
	if d("bootenvs").Find(localBoot.Key()) == nil {
		res.Create(d, localBoot, nil)
	}
	for _, prefIsh := range d("preferences").Items() {
		pref := AsPref(prefIsh)
		res.runningPrefs[pref.Name] = pref.Val
	}
	if d("preferences").Find(res.GlobalProfileName) == nil {
		res.Create(d, &models.Profile{
			Name:   res.GlobalProfileName,
			Params: map[string]interface{}{},
			Tasks:  []string{},
		})
	}
	users := d("users")
	if users.Count() == 0 {
		res.Infof("debugBootEnv", "Creating rocketskates user")
		user := &User{}
		Fill(user)
		user.Name = "rocketskates"
		if err := user.ChangePassword(d, "r0cketsk8ts"); err != nil {
			logger.Fatalf("Failed to create rocketskates user: %v", err)
		}
		res.Create(d, user)
	}
	res.defaultBootEnv = defaultPrefs["defaultBootEnv"]
	machines := d("machines")
	for _, obj := range machines.Items() {
		machine := AsMachine(obj)
		bootEnv := d("bootenvs").Find(machine.BootEnv)
		if bootEnv == nil {
			continue
		}
		err := &models.Error{}
		AsBootEnv(bootEnv).Render(d, machine, err).register(res.FS)
		if err.ContainsError() {
			logger.Printf("Error rendering machine %s at startup:", machine.UUID())
			logger.Println(err.Error())
		}
	}
	if err := res.RenderUnknown(d); err != nil {
		logger.Fatalf("Failed to render unknown bootenv: %v", err)
	}
	return res
}

func (p *DataTracker) Prefs() map[string]string {
	vals := map[string]string{}
	p.prefMux.Lock()
	for k, v := range p.defaultPrefs {
		vals[k] = v
	}
	for k, v := range p.runningPrefs {
		vals[k] = v
	}
	p.prefMux.Unlock()
	return vals
}

func (p *DataTracker) Pref(name string) (string, error) {
	res, ok := p.Prefs()[name]
	if !ok {
		return "", fmt.Errorf("No such preference %s", name)
	}
	return res, nil
}

func (p *DataTracker) pref(name string) string {
	return p.Prefs()[name]
}

func (p *DataTracker) SetPrefs(d Stores, prefs map[string]string) error {
	err := &models.Error{}
	bootenvs := d("bootenvs")
	benvCheck := func(name, val string) *BootEnv {
		be := bootenvs.Find(val)
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
		p.prefMux.Lock()
		defer p.prefMux.Unlock()
		pref := &models.Pref{}
		pref.Name = name
		pref.Val = val
		if _, saveErr := p.Save(d, pref); saveErr != nil {
			err.Errorf("%s: Failed to save %s: %v", name, val, saveErr)
			return false
		}
		p.runningPrefs[name] = val
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
				err.AddError(p.RenderUnknown(d))
			}
		case "unknownTokenTimeout",
			"knownTokenTimeout",
			"debugDhcp",
			"debugRenderer",
			"debugBootEnv",
			"debugFrontend",
			"debugPlugins":
			if intCheck(name, val) {
				savePref(name, val)
			}
			continue
		default:
			err.Errorf("Unknown preference %s", name)
		}
	}
	return err.HasError()
}

func (p *DataTracker) RenderUnknown(d Stores) error {
	pref, e := p.Pref("unknownBootEnv")
	if e != nil {
		return e
	}
	envIsh := d("bootenvs").Find(pref)
	if envIsh == nil {
		return fmt.Errorf("No such BootEnv: %s", pref)
	}
	env := AsBootEnv(envIsh)
	err := &models.Error{Object: env, Type: "StartupError"}
	if !env.Available {
		err.AddError(env)
		return err
	}
	if !env.OnlyUnknown {
		err.Errorf("BootEnv %s cannot be used for the unknownBootEnv", env.Name)
		return err
	}
	env.Render(d, nil, err).register(p.FS)
	return err.HasError()
}

func (p *DataTracker) getBackend(t models.Model) store.Store {
	res, ok := p.objs[t.Prefix()]
	if !ok {
		p.Logger.Fatalf("%s: No registered storage backend!", t.Prefix())
	}
	return res.backingStore
}

func (p *DataTracker) setDT(s models.Model) {
	if tgt, ok := s.(dtSetter); ok {
		tgt.setDT(p)
	}
}

func (p *DataTracker) Create(d Stores, obj models.Model) (saved bool, err error) {
	ref := toBackend(p, d, obj)
	prefix := ref.Prefix()
	key := ref.Key()
	backend := d(prefix).backingStore
	if key == "" {
		return false, fmt.Errorf("dataTracker create %s: Empty key not allowed", prefix)
	}
	if d(prefix).Find(key) != nil {
		return false, fmt.Errorf("dataTracker create %s: %s already exists", prefix, key)
	}
	ref.(validator).setStores(d)
	checker, checkOK := ref.(Validator)
	if checkOK {
		checker.ClearValidation()
	}
	saved, err = store.Create(backend, ref)
	if saved {
		ref.(validator).clearStores()
		d(prefix).Add(ref)

		p.publishers.Publish(prefix, "create", key, ref)
	}

	return saved, err
}

func (p *DataTracker) Remove(d Stores, obj models.Model) (removed bool, err error) {
	ref := toBackend(p, d, obj)
	prefix := ref.Prefix()
	key := ref.Key()
	backend := d(prefix).backingStore
	item := d(prefix).Find(key)
	if item == nil {
		err := &models.Error{
			Code:  http.StatusNotFound,
			Key:   key,
			Model: prefix,
		}
		err.Errorf("%s: DELETE %s: Not Found", err.Model, err.Key)
		return false, err
	}
	item.(validator).setStores(d)
	removed, err = store.Remove(backend, item.(store.KeySaver))
	if removed {
		d(prefix).Remove(item)
		p.publishers.Publish(prefix, "delete", key, item)
	}
	return removed, err
}

func (p *DataTracker) Patch(d Stores, obj models.Model, key string, patch jsonpatch2.Patch) (models.Model, error) {
	ref := toBackend(p, d, obj)
	prefix := ref.Prefix()
	backend := d(prefix).backingStore
	target := d(prefix).Find(key)
	if target == nil {
		err := &models.Error{
			Code:  http.StatusNotFound,
			Key:   key,
			Model: prefix,
		}
		err.Errorf("%s: PATCH %s: Not Found", err.Model, err.Key)
		return nil, err
	}
	buf, fatalErr := json.Marshal(target)
	if fatalErr != nil {
		p.Logger.Fatalf("Non-JSON encodable %v:%v stored in cache: %v", prefix, key, fatalErr)
	}
	resBuf, patchErr, loc := patch.Apply(buf)
	if patchErr != nil {
		err := &models.Error{
			Code:  http.StatusNotAcceptable,
			Key:   key,
			Model: ref.Prefix(),
			Type:  "JsonPatchError",
		}
		err.Errorf("Patch error at line %d: %v", loc, patchErr)
		return nil, err
	}
	toSave := ref.New()
	if err := json.Unmarshal(resBuf, &toSave); err != nil {
		return nil, err
	}
	p.setDT(toSave)
	toSave.(validator).setStores(d)
	checker, checkOK := toSave.(Validator)
	if checkOK {
		checker.ClearValidation()
	}
	saved, err := store.Update(backend, toSave)
	toSave.(validator).clearStores()
	if !saved {
		return toSave, err
	}
	d(prefix).Add(toSave)
	p.publishers.Publish(prefix, "update", key, toSave)
	return toSave, nil
}

func (p *DataTracker) Update(d Stores, obj models.Model) (saved bool, err error) {
	ref := toBackend(p, d, obj)
	prefix := ref.Prefix()
	key := ref.Key()
	backend := d(prefix).backingStore
	if target := d(prefix).Find(key); target == nil {
		err := &models.Error{
			Code:  http.StatusNotFound,
			Key:   key,
			Model: prefix,
		}
		err.Errorf("%s: PUT %s: Not Found", err.Model, err.Key)
		return false, err
	}

	p.setDT(ref)
	ref.(validator).setStores(d)
	checker, checkOK := ref.(Validator)
	if checkOK {
		checker.ClearValidation()
	}
	saved, err = store.Update(backend, ref)
	ref.(validator).clearStores()
	if saved {
		d(prefix).Add(ref)
		p.publishers.Publish(prefix, "update", key, ref)
	}
	return saved, err
}

func (p *DataTracker) Save(d Stores, obj models.Model) (saved bool, err error) {
	ref := toBackend(p, d, obj)
	prefix := ref.Prefix()
	backend := d(prefix).backingStore
	ref.(validator).setStores(d)
	checker, checkOK := ref.(Validator)
	if checkOK {
		checker.ClearValidation()
	}
	saved, err = store.Save(backend, ref)
	ref.(validator).clearStores()
	if saved {
		d(ref.Prefix()).Add(ref)
		p.publishers.Publish(ref.Prefix(), "save", ref.Key(), ref)
	}
	return saved, err
}

func (p *DataTracker) GetToken(tokenString string) (*DrpCustomClaims, error) {
	return p.tokenManager.get(tokenString)
}

func (p *DataTracker) SealClaims(claims *DrpCustomClaims) (string, error) {
	return claims.Seal(p.tokenManager)
}

func (p *DataTracker) Backup() ([]byte, error) {
	keys := make([]string, len(p.objs))
	for k := range p.objs {
		keys = append(keys, k)
	}
	_, unlocker := p.LockEnts(keys...)
	defer unlocker()
	res := map[string][]models.Model{}
	for _, k := range keys {
		res[k] = p.objs[k].Items()
	}
	return json.Marshal(res)
}

// Assumes that all locks are held
func (p *DataTracker) ReplaceBackend(st store.Store) error {
	p.Backend = st
	return p.rebuildCache()
}

func (p *DataTracker) NewToken(id string, ttl int, scope, action, specific string) (string, error) {
	return NewClaim(id, ttl).Add(scope, action, specific).Seal(p.tokenManager)
}

func (p *DataTracker) Printf(f string, args ...interface{}) {
	p.Logger.Printf(f, args...)
}

func (p *DataTracker) DebugLevel(pref string) int {
	debugLevel := 0
	d2, e := strconv.Atoi(p.pref(pref))
	if e == nil {
		debugLevel = d2
	}
	return debugLevel
}

func (p *DataTracker) printlevelf(pref string, level int, f string, args ...interface{}) {
	debugLevel := p.DebugLevel(pref)
	if debugLevel >= level {
		p.Logger.Printf(f, args...)
	}
}

func (p *DataTracker) Infof(pref, f string, args ...interface{}) {
	p.printlevelf(pref, 1, f, args...)
}
func (p *DataTracker) Debugf(pref, f string, args ...interface{}) {
	p.printlevelf(pref, 2, f, args...)
}

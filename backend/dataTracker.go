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
	"strings"
	"sync"
	"text/template"

	"github.com/VictorLowther/jsonpatch2"
	jwt "github.com/dgrijalva/jwt-go"
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

func BasicContent() store.Store {
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
chain tftp://{{.ProvisionerAddress}}/${netX/ip}.ipxe || exit
`,
				},
			},
		}

		localBoot = &models.BootEnv{
			Name:        "local",
			Description: "The boot environment you should use to have known machines boot off their local hard drive",
			OS: models.OsInfo{
				Name: "local",
			},
			OnlyUnknown: false,
			Templates: []models.TemplateInfo{
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
		noneStage = &models.Stage{
			Name: "none",
			MetaData: models.MetaData{Meta: map[string]string{
				"icon":  "circle thin",
				"color": "green",
				"title": "Digital Rebar Provision",
			}},
		}
	)
	res, _ := store.Open("memory:///")
	bootEnvs, _ := res.MakeSub("bootenvs")
	stages, _ := res.MakeSub("stages")
	localBoot.ClearValidation()
	ignoreBoot.ClearValidation()
	noneStage.ClearValidation()
	localBoot.Fill()
	ignoreBoot.Fill()
	noneStage.Fill()
	bootEnvs.Save("local", localBoot)
	bootEnvs.Save("ignore", ignoreBoot)
	stages.Save("none", noneStage)
	res.(*store.Memory).SetMetaData(map[string]string{
		"Name":        "BasicStore",
		"Description": "Default objects that must be present",
		"Version":     "Unversioned",
		"Type":        "default",
	})
	return res
}

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

func (s *Store) getBackend(obj models.Model) store.Store {
	return s.backingStore
}

type dtSetter interface {
	models.Model
	setDT(*DataTracker)
}

func Fill(t store.KeySaver) {
	switch obj := t.(type) {
	case *Stage:
		obj.Stage = &models.Stage{}
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
	case *Pref:
		obj.Pref = &models.Pref{}
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

func ModelToBackend(m models.Model) store.KeySaver {
	switch obj := m.(type) {
	case store.KeySaver:
		return obj
	case *models.Stage:
		return &Stage{Stage: obj}
	case *models.BootEnv:
		return &BootEnv{BootEnv: obj}
	case *models.Job:
		return &Job{Job: obj}
	case *models.Lease:
		return &Lease{Lease: obj}
	case *models.Machine:
		return &Machine{Machine: obj}
	case *models.Param:
		return &Param{Param: obj}
	case *models.Plugin:
		return &Plugin{Plugin: obj}
	case *models.Pref:
		return &Pref{Pref: obj}
	case *models.Profile:
		return &Profile{Profile: obj}
	case *models.Reservation:
		return &Reservation{Reservation: obj}
	case *models.Subnet:
		return &Subnet{Subnet: obj}
	case *models.Task:
		return &Task{Task: obj}
	case *models.Template:
		return &Template{Template: obj}
	case *models.User:
		return &User{User: obj}
	default:
		panic(fmt.Sprintf("Unknown model %T", m))
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
	case *models.Stage:
		var res Stage
		if ours != nil {
			res = *ours.(*Stage)
		} else {
			res = Stage{}
		}
		res.Stage = obj
		p.setDT(&res)
		return &res
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
	ForceOurAddress     bool
	StaticPort, ApiPort int
	Logger              *log.Logger
	FS                  *FileSystem
	Backend             store.Store
	objs                map[string]*Store
	defaultPrefs        map[string]string
	runningPrefs        map[string]string
	prefMux             *sync.Mutex
	allMux              *sync.RWMutex
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
		&User{p: res},
		&Template{p: res},
		&Task{p: res},
		&Profile{p: res},
		&BootEnv{p: res},
		&Stage{p: res},
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
	// If we are behind a NAT, always use Our Address
	if p.ForceOurAddress {
		return p.OurAddress
	}
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

func (p *DataTracker) rebuildCache() (hard, soft *models.Error) {
	hard = &models.Error{Code: 500, Type: "Failed to load backing objects from cache"}
	soft = &models.Error{Code: 422, Type: ValidationError}
	p.objs = map[string]*Store{}
	objs := allKeySavers(p)
	for _, obj := range objs {
		prefix := obj.Prefix()
		bk := p.Backend.GetSub(prefix)
		p.objs[prefix] = &Store{backingStore: bk}
		storeObjs, err := store.List(bk, toBackend(p, nil, obj))
		if err != nil {
			// Make fake index to keep others from failing and exploding.
			res := make([]models.Model, 0)
			p.objs[prefix].Index = *index.Create(res)
			hard.Errorf("Unable to load %s: %v", prefix, err)
			continue
		}
		res := make([]models.Model, len(storeObjs))
		for i := range storeObjs {
			res[i] = models.Model(storeObjs[i])
			if v, ok := res[i].(models.Validator); ok && v.Useable() {
				soft.AddError(v.HasError())
			}
		}
		if prefix == "tasks" {
			stack, ok := bk.(*store.StackedStore)
			if ok {
				subStore := stack.Subs()[prefix]
				if subStore != nil {
					sub := stack.Subs()[prefix].(*store.StackedStore)
					for i := range res {
						obj := AsTask(res[i])
						key := obj.Key()
						meta := sub.MetaFor(key)
						if flagStr, ok := meta["feature-flags"]; ok && len(flagStr) > 0 {
							obj.MergeFeatures(strings.Split(flagStr, ","))
						}
						if obj.HasFeature("original-exit-codes") {
							obj.RemoveFeature("sane-exit-codes")
						}
						if !obj.HasFeature("sane-exit-codes") {
							obj.AddFeature("original-exit-codes")
						}
						res[i] = obj
					}
				}
			}
		}

		p.objs[prefix].Index = *index.Create(res)

		if prefix == "templates" {
			buf := &bytes.Buffer{}
			for _, thing := range p.objs[prefix].Items() {
				tmpl := AsTemplate(thing)
				fmt.Fprintf(buf, `{{define "%s"}}%s{{end}}`, tmpl.ID, tmpl.Contents)
			}
			root, err := template.New("").Parse(buf.String())
			if err != nil {
				hard.Errorf("Unable to load root templates: %v", err)
				return
			}
			p.rootTemplate = root
			p.rootTemplate.Option("missingkey=error")
		}
	}
	return
}

// This must be locked with ALL locks from the caller.
func ValidateDataTrackerStore(backend store.Store, logger *log.Logger) (hard, soft error) {
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
		tokenManager:      NewJwtManager([]byte{}, JwtConfig{Method: jwt.SigningMethodHS256}),
		prefMux:           &sync.Mutex{},
		allMux:            &sync.RWMutex{},
		FS:                NewFS(".", logger),
		tmplMux:           &sync.Mutex{},
		GlobalProfileName: "global",
		thunks:            make([]func(), 0),
		thunkMux:          &sync.Mutex{},
		publishers:        &Publishers{},
	}

	// Load stores. - This must be All locked by the caller
	a, b := res.rebuildCache()
	return a.HasError(), b.HasError()
}

// Create a new DataTracker that will use passed store to save all operational data
func NewDataTracker(backend store.Store,
	fileRoot, logRoot, addr string, forceAddr bool,
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
		ForceOurAddress:   forceAddr,
		Logger:            logger,
		defaultPrefs:      defaultPrefs,
		runningPrefs:      map[string]string{},
		tokenManager:      NewJwtManager([]byte{}, JwtConfig{Method: jwt.SigningMethodHS256}),
		prefMux:           &sync.Mutex{},
		allMux:            &sync.RWMutex{},
		FS:                NewFS(fileRoot, logger),
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

	// Load stores. - This is implicitly locked because we are creating a new one.
	hard, _ := res.rebuildCache()
	if hard.HasError() != nil {
		res.Logger.Fatalf("dataTracker: Error loading data: %v", hard.HasError())
	}

	// Create minimal content.
	d, unlocker := res.LockEnts("stages", "bootenvs", "preferences", "users", "machines", "profiles", "params")
	defer unlocker()

	// Load the prefs - overriding defaults.
	for _, prefIsh := range d("preferences").Items() {
		pref := AsPref(prefIsh)
		res.runningPrefs[pref.Name] = pref.Val
	}

	// Set systemGrantorSecret and baseTokenSecret if unset and save it to backing store.
	prefs := res.Prefs()
	savePrefs := false
	for _, pref := range []string{"systemGrantorSecret", "baseTokenSecret"} {
		if val, ok := prefs[pref]; !ok || val == "" {
			prefs[pref] = randString(32)
			savePrefs = true
		}
	}
	if savePrefs {
		res.SetPrefs(d, prefs)
	}
	res.tokenManager.updateKey([]byte(res.pref("baseTokenSecret")))

	if d("profiles").Find(res.GlobalProfileName) == nil {
		res.Create(d, &models.Profile{
			Name:   res.GlobalProfileName,
			Params: map[string]interface{}{},
			MetaData: models.MetaData{Meta: map[string]string{
				"icon":  "world",
				"color": "blue",
				"title": "Digital Rebar Provision",
			}},
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
	stages := d("stages")
	lenCheck := func(name, val string) bool {
		if len(val) != 32 {
			err.Errorf("%s: Must be a string of length 32: %s", name, val)
			return false
		}
		return true
	}
	benvCheck := func(name, val string) *BootEnv {
		be := bootenvs.Find(val)
		if be == nil {
			err.Errorf("%s: Bootenv %s does not exist", name, val)
			return nil
		}
		return AsBootEnv(be)
	}
	stageCheck := func(name, val string) bool {
		stage := stages.Find(val)
		if stage == nil {
			err.Errorf("%s: Stage %s does not exist", name, val)
			return false
		}
		return true
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
		case "systemGrantorSecret":
			savePref(name, val)
		case "baseTokenSecret":
			if lenCheck(name, val) && savePref(name, val) {
				p.tokenManager.updateKey([]byte(val))
			}
		case "defaultBootEnv":
			be := benvCheck(name, val)
			if be != nil && !be.OnlyUnknown {
				savePref(name, val)
			}
		case "defaultStage":
			if stageCheck(name, val) {
				savePref(name, val)
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
	if ms, ok := obj.(models.Filler); ok {
		ms.Fill()
	}
	ref := toBackend(p, d, obj)
	prefix := ref.Prefix()
	key := ref.Key()
	backend := d(prefix).backingStore
	if key == "" {
		return false, &models.Error{
			Type:     "CREATE",
			Model:    prefix,
			Messages: []string{"Empty key not allowed"},
			Code:     http.StatusBadRequest,
		}
	}
	if d(prefix).Find(key) != nil {
		return false, &models.Error{
			Type:     "CREATE",
			Model:    prefix,
			Key:      key,
			Messages: []string{"already exists"},
			Code:     http.StatusConflict,
		}
	}
	ref.(validator).setStores(d)
	checker, checkOK := ref.(models.Validator)
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
		return false, &models.Error{
			Type:     "DELETE",
			Code:     http.StatusNotFound,
			Key:      key,
			Model:    prefix,
			Messages: []string{"Not Found"},
		}
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
		return nil, &models.Error{
			Type:     "PATCH",
			Code:     http.StatusNotFound,
			Key:      key,
			Model:    prefix,
			Messages: []string{"Not Found"},
		}
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
			Type:  "PATCH",
		}
		err.Errorf("Patch error at line %d: %v", loc, patchErr)
		buf, _ := json.Marshal(patch[loc])
		err.Errorf("Patch line: %v", string(buf))
		return nil, err
	}
	toSave := ref.New()
	if err := json.Unmarshal(resBuf, &toSave); err != nil {
		retErr := &models.Error{
			Code:  http.StatusConflict,
			Key:   key,
			Model: ref.Prefix(),
			Type:  "PATCH",
		}
		retErr.AddError(err)
		return nil, retErr
	}
	if ms, ok := toSave.(models.Filler); ok {
		ms.Fill()
	}
	p.setDT(toSave)
	toSave.(validator).setStores(d)
	checker, checkOK := toSave.(models.Validator)
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
		return false, &models.Error{
			Type:     "PUT",
			Code:     http.StatusNotFound,
			Key:      key,
			Model:    prefix,
			Messages: []string{"Not Found"},
		}
	}

	p.setDT(ref)
	ref.(validator).setStores(d)
	if ms, ok := ref.(models.Filler); ok {
		ms.Fill()
	}
	checker, checkOK := ref.(models.Validator)
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
	if ms, ok := ref.(models.Filler); ok {
		ms.Fill()
	}
	checker, checkOK := ref.(models.Validator)
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
func (p *DataTracker) ReplaceBackend(st store.Store) (hard, soft error) {
	p.Backend = st
	return p.rebuildCache()
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

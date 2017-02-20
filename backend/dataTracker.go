package backend

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/digitalrebar/go/rebar-api/api"
	"github.com/rackn/rocket-skates/embedded"
)

type keySavers struct {
	sync.Mutex
	d map[string]store.KeySaver
}

// DataTracker represents everything there is to know about acting as a provisioner.
type DataTracker struct {
	useProvisioner, useDHCP bool
	Logger                  *log.Logger
	Address                 net.IP
	FileRoot                string
	FileURL                 string
	CommandURL              string
	DefaultBootEnv          string
	RebarClient             *api.Client
	backends                map[string]store.SimpleStore
	backendMux              sync.Mutex
	objs                    map[string]*keySavers
	objTypeMux              sync.Mutex
}

func (p *DataTracker) makeBackends(backend store.SimpleStore, objs []store.KeySaver) {
	p.backendMux.Lock()
	defer p.backendMux.Unlock()
	for _, o := range objs {
		prefix := o.Prefix()
		bk, err := backend.Sub(prefix)
		if err != nil {
			p.Logger.Fatalf("provisioner: Error creating substore %s: %v", prefix, err)
		}
		p.backends[prefix] = bk
	}
}

func (p *DataTracker) loadData(refObjs []store.KeySaver) error {
	p.objs = map[string]*keySavers{}
	for _, refObj := range refObjs {
		prefix := refObj.Prefix()
		instances := &keySavers{d: map[string]store.KeySaver{}}
		objs, err := store.List(refObj)
		if err != nil {
			return err
		}
		for _, obj := range objs {
			instances.d[obj.Key()] = obj
		}
		p.objs[prefix] = instances
	}
	return nil
}

// Create a new DataTracker that will use passed store to save all operational data
func NewDataTracker(backend store.SimpleStore, useProvisioner, useDHCP bool) *DataTracker {
	res := &DataTracker{
		useDHCP:        useDHCP,
		useProvisioner: useProvisioner,
		backends:       map[string]store.SimpleStore{},
		backendMux:     sync.Mutex{},
	}
	objs := []store.KeySaver{&Machine{p: res}, &User{p: res}}

	if useProvisioner {
		objs = append(objs, &Template{p: res}, &BootEnv{p: res})
	}
	if useDHCP {
		objs = append(objs, &Lease{p: res}, &Reservation{p: res}, &Subnet{p: res})
	}
	res.makeBackends(backend, objs)
	res.loadData(objs)
	return res
}

// ExtractAssets is responsible for saving all the assets we need to act as a provisioner.
func (p *DataTracker) ExtractAssets() error {
	if !p.useProvisioner {
		return nil
	}
	if p.FileRoot == "" {
		return fmt.Errorf("ExtractAssets called before FileRoot was set")
	}
	assets := map[string]string{
		"assets/explode_iso.sh": "",
	}
	for src, dest := range assets {
		buf, err := embedded.Asset(src)
		if err != nil {
			return fmt.Errorf("No such embedded asset %s", src)
		}
		info, err := embedded.AssetInfo(src)
		if err != nil {
			return fmt.Errorf("No mode info for embedded asset %s", src)
		}
		destFile := path.Join(p.FileRoot, dest, path.Join(strings.Split(src, "/")[1:]...))
		destDir := path.Dir(destFile)
		log.Print(destFile)
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}
		if err := ioutil.WriteFile(destFile, buf, info.Mode()); err != nil {
			return err
		}
		if err := os.Chtimes(destFile, info.ModTime(), info.ModTime()); err != nil {
			return err
		}
	}
	return nil
}

func (p *DataTracker) getBackend(t store.KeySaver) store.SimpleStore {
	p.backendMux.Lock()
	defer p.backendMux.Unlock()
	res, ok := p.backends[t.Prefix()]
	if !ok {
		p.Logger.Fatalf("%s: No registered storage backend!", t.Prefix())
	}
	return res
}

// FetchAll returns all of the cached objects of a given type.  It
// should be used instead of store.List.
func (p *DataTracker) FetchAll(ref store.KeySaver) []store.KeySaver {
	prefix := ref.Prefix()
	p.objTypeMux.Lock()
	instances := p.objs[prefix]
	instances.Lock()
	p.objTypeMux.Unlock()
	res := make([]store.KeySaver, 0, len(instances.d))
	for _, v := range instances.d {
		res = append(res, v)
	}
	instances.Unlock()
	return res
}

// FetchOne returns a specific instance from the cached objects of
// that type.  It should be used instead of store.Load.
func (p *DataTracker) FetchOne(ref store.KeySaver, key string) (store.KeySaver, bool) {
	prefix := ref.Prefix()
	p.objTypeMux.Lock()
	p.objs[prefix].Lock()
	res, ok := p.objs[prefix].d[key]
	p.objs[prefix].Unlock()
	p.objTypeMux.Unlock()
	return res, ok
}

// Create creates a new thing, caching it locally iff the create
// succeeds.  It should be used instead of store.Create
func (p *DataTracker) Create(ref store.KeySaver) (bool, error) {
	prefix := ref.Prefix()
	p.objTypeMux.Lock()
	p.objs[prefix].Lock()
	instances := p.objs[prefix]
	p.objTypeMux.Unlock()
	defer instances.Unlock()
	key := ref.Key()
	if _, ok := instances.d[key]; ok {
		return false, fmt.Errorf("provisioner create %s: %s already exists", prefix, key)
	}
	saved, err := store.Create(ref)
	if saved {
		instances.d[key] = ref
	}
	return saved, err
}

// Remove removes the thing from the backing store.  If the remove
// succeeds, it will also be removed from the local cache.  It should
// be used instead of store.Remove
func (p *DataTracker) Remove(ref store.KeySaver) (bool, error) {
	prefix := ref.Prefix()
	p.objTypeMux.Lock()
	p.objs[prefix].Lock()
	instances := p.objs[prefix]
	p.objTypeMux.Unlock()
	defer instances.Unlock()
	key := ref.Key()
	if _, ok := instances.d[key]; !ok {
		return false, fmt.Errorf("provisioner remove %s: %s does not exist", prefix, key)
	}
	removed, err := store.Remove(ref)
	if removed {
		delete(instances.d, key)
	}
	return removed, err
}

// Update updates the passed thing, and updates the local cache iff
// the update succeeds.  It should be used in preference to
// store.Update
func (p *DataTracker) Update(ref store.KeySaver) (bool, error) {
	prefix := ref.Prefix()
	p.objTypeMux.Lock()
	p.objs[prefix].Lock()
	instances := p.objs[prefix]
	p.objTypeMux.Unlock()
	defer instances.Unlock()
	key := ref.Key()
	ok, err := store.Update(ref)
	if ok {
		instances.d[key] = ref
	}
	return ok, err
}

// Save saves the passed thing, updating the local cache iff the save
// succeeds.  It should be used instead of store.Save
func (p *DataTracker) Save(ref store.KeySaver) (bool, error) {
	prefix := ref.Prefix()
	p.objTypeMux.Lock()
	p.objs[prefix].Lock()
	instances := p.objs[prefix]
	p.objTypeMux.Unlock()
	defer instances.Unlock()
	key := ref.Key()
	ok, err := store.Save(ref)
	if ok {
		instances.d[key] = ref
	}
	return ok, err
}

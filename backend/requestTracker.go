package backend

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

type RequestTracker struct {
	logger.Logger
	dt    *DataTracker
	locks []string
	d     Stores
}

func (p *DataTracker) Request(l logger.Logger, locks ...string) *RequestTracker {
	return &RequestTracker{dt: p, Logger: l, locks: locks}
}

func (rt *RequestTracker) Find(prefix, key string) models.Model {
	var res models.Model
	if s := rt.d(prefix); s != nil {
		res = s.Find(key)
	}
	return res
}

func (rt *RequestTracker) Do(thunk func(Stores)) {
	d, unlocker := rt.dt.lockEnts(rt.locks...)
	rt.d = d
	defer unlocker()
	thunk(d)
}

func (rt *RequestTracker) With(s Stores) *RequestTracker {
	rt.d = s
	return rt
}

func (rt *RequestTracker) withFake() *RequestTracker {
	rt.d = func(s string) *Store { return rt.dt.objs[s] }
	return rt
}

func (rt *RequestTracker) backend(m models.Model) store.Store {
	return rt.dt.getBackend(m)
}

func (rt *RequestTracker) stores(s string) *Store {
	return rt.d(s)
}

func (rt *RequestTracker) spkibrt(obj models.Model) (
	s Stores,
	prefix, key string,
	idx *Store, bk store.Store,
	ref, target store.KeySaver) {
	if rt.d == nil {
		rt.Panicf("RequestTracker used outside of Do")
		return
	}
	s = rt.d
	prefix = obj.Prefix()
	idx = rt.d(prefix)
	bk = idx.backingStore
	if obj == nil {
		return
	}
	key = obj.Key()
	m := idx.Find(key)
	ref = ModelToBackend(obj)
	if m != nil {
		target = m.(store.KeySaver)
	}
	return
}

func (rt *RequestTracker) Create(obj models.Model) (saved bool, err error) {
	if ms, ok := obj.(models.Filler); ok {
		ms.Fill()
	}
	_, prefix, key, idx, backend, ref, target := rt.spkibrt(obj)
	if key == "" {
		return false, &models.Error{
			Type:     "CREATE",
			Model:    prefix,
			Messages: []string{"Empty key not allowed"},
			Code:     http.StatusBadRequest,
		}
	}
	if target != nil {
		return false, &models.Error{
			Type:     "CREATE",
			Model:    prefix,
			Key:      key,
			Messages: []string{"already exists"},
			Code:     http.StatusConflict,
		}
	}
	ref.(validator).setRT(rt)
	checker, checkOK := ref.(models.Validator)
	if checkOK {
		checker.ClearValidation()
	}
	saved, err = store.Create(backend, ref)
	if saved {
		ref.(validator).clearRT()
		idx.Add(ref)

		rt.dt.Publish(prefix, "create", key, ref)
	}

	return saved, err
}

func (rt *RequestTracker) Remove(obj models.Model) (removed bool, err error) {
	_, prefix, key, idx, backend, _, item := rt.spkibrt(obj)
	if item == nil {
		return false, &models.Error{
			Type:     "DELETE",
			Code:     http.StatusNotFound,
			Key:      key,
			Model:    prefix,
			Messages: []string{"Not Found"},
		}
	}
	item.(validator).setRT(rt)
	removed, err = store.Remove(backend, item.(store.KeySaver))
	if removed {
		idx.Remove(item)
		rt.dt.Publish(prefix, "delete", key, item)
	}
	return removed, err
}

func (rt *RequestTracker) Patch(obj models.Model, key string, patch jsonpatch2.Patch) (models.Model, error) {
	_, prefix, _, idx, backend, _, _ := rt.spkibrt(obj)
	ref := idx.Find(key)
	if ref == nil {
		return nil, &models.Error{
			Type:     "PATCH",
			Code:     http.StatusNotFound,
			Key:      key,
			Model:    prefix,
			Messages: []string{"Not Found"},
		}
	}
	target := ref.(store.KeySaver)
	buf, fatalErr := json.Marshal(target)
	if fatalErr != nil {
		rt.Fatalf("Non-JSON encodable %v:%v stored in cache: %v", obj.Prefix(), key, fatalErr)
	}
	resBuf, patchErr, loc := patch.Apply(buf)
	rt.Tracef("Patching %s", string(buf))
	rt.Tracef("Patched to: %s", string(resBuf))
	if patchErr != nil {
		err := &models.Error{
			Code:  http.StatusConflict,
			Key:   key,
			Model: prefix,
			Type:  "PATCH",
		}
		rt.Tracef("Patched to: %s", string(resBuf))
		err.Errorf("Patch error at line %d: %v", loc, patchErr)
		buf, _ := json.Marshal(patch[loc])
		err.Errorf("Patch line: %v", string(buf))
		return nil, err
	}
	toSave := target.New()
	if err := json.Unmarshal(resBuf, &toSave); err != nil {
		retErr := &models.Error{
			Code:  http.StatusNotAcceptable,
			Key:   key,
			Model: prefix,
			Type:  "PATCH",
		}
		retErr.AddError(err)
		return nil, retErr
	}
	if ms, ok := toSave.(models.Filler); ok {
		ms.Fill()
	}
	toSave.(validator).setRT(rt)
	checker, checkOK := toSave.(models.Validator)
	if checkOK {
		checker.ClearValidation()
	}
	if obj != nil {
		a, aok := obj.(models.ChangeForcer)
		if aok {
			rt.Tracef("obj: %#v", obj)
			rt.Tracef("a: %#v", a)
			if a != nil && a.ChangeForced() {
				rt.Tracef("Forcing change for %s:%s", prefix, key)
				toSave.(models.ChangeForcer).ForceChange()
			}
		}
	}
	saved, err := store.Update(backend, toSave)
	toSave.(validator).clearRT()
	if saved {
		idx.Add(toSave)
		rt.dt.Publish(prefix, "update", key, toSave)
	}
	return toSave, err
}

func (rt *RequestTracker) Update(obj models.Model) (saved bool, err error) {
	_, prefix, key, idx, backend, ref, target := rt.spkibrt(obj)
	if target == nil {
		return false, &models.Error{
			Type:     "PUT",
			Code:     http.StatusNotFound,
			Key:      key,
			Model:    prefix,
			Messages: []string{"Not Found"},
		}
	}
	if ms, ok := ref.(models.Filler); ok {
		ms.Fill()
	}
	ref.(validator).setRT(rt)
	checker, checkOK := ref.(models.Validator)
	if checkOK {
		checker.ClearValidation()
	}
	saved, err = store.Update(backend, ref)
	ref.(validator).clearRT()
	if saved {
		idx.Add(ref)
		rt.dt.Publish(prefix, "update", key, ref)
	}
	return saved, err
}

func (rt *RequestTracker) Save(obj models.Model) (saved bool, err error) {
	_, prefix, key, idx, backend, ref, _ := rt.spkibrt(obj)
	if ms, ok := ref.(models.Filler); ok {
		ms.Fill()
	}
	ref.(validator).setRT(rt)
	checker, checkOK := ref.(models.Validator)
	if checkOK {
		checker.ClearValidation()
	}
	saved, err = store.Save(backend, ref)
	ref.(validator).clearRT()
	if saved {
		idx.Add(ref)
		rt.dt.Publish(prefix, "save", key, ref)
	}
	return saved, err
}

func (rt *RequestTracker) GetParams(obj models.Paramer, aggregate bool) map[string]interface{} {
	res := obj.GetParams()
	if !aggregate {
		return res
	}
	subObjs := []models.Paramer{}
	var profiles []string
	var stage string
	switch ref := obj.(type) {
	case *rMachine:
		profiles, stage = ref.Profiles, ref.Stage
	case *models.Machine:
		profiles, stage = ref.Profiles, ref.Stage
	case *Machine:
		profiles, stage = ref.Profiles, ref.Stage
	}
	for _, pn := range profiles {
		if pobj := rt.Find("profiles", pn); pobj != nil {
			subObjs = append(subObjs, pobj.(models.Paramer))
		}
	}
	if sobj := rt.Find("stages", stage); sobj != nil {
		for _, pn := range AsStage(sobj).Profiles {
			if pobj := rt.Find("profiles", pn); pobj != nil {
				subObjs = append(subObjs, pobj.(models.Paramer))
			}
		}
	}
	if pobj := rt.Find("profiles", rt.dt.GlobalProfileName); pobj != nil {
		subObjs = append(subObjs, pobj.(models.Paramer))
	}
	for _, sub := range subObjs {
		subp := sub.GetParams()
		for k, v := range subp {
			if _, ok := res[k]; !ok {
				res[k] = v
			}
		}
	}
	return res
}

func (rt *RequestTracker) SetParams(obj models.Paramer, values map[string]interface{}) error {
	obj.SetParams(values)
	e := &models.Error{Code: 422, Type: ValidationError, Model: obj.Prefix(), Key: obj.Key()}
	_, e2 := rt.Save(obj)
	e.AddError(e2)
	return e.HasError()
}

func (rt *RequestTracker) GetParam(obj models.Paramer, key string, aggregate bool) (interface{}, bool) {
	v, ok := rt.GetParams(obj, aggregate)[key]
	if ok || !aggregate {
		return v, ok
	}
	if pobj := rt.Find("params", key); pobj != nil {
		rt.Tracef("Param %s not defined, falling back to default value")
		return AsParam(pobj).DefaultValue()
	}
	return nil, false
}

func (rt *RequestTracker) SetParam(obj models.Paramer, key string, val interface{}) error {
	p := obj.GetParams()
	p[key] = val
	return rt.SetParams(obj, p)
}

func (rt *RequestTracker) DelParam(obj models.Paramer, key string) (interface{}, error) {
	p := obj.GetParams()
	if val, ok := p[key]; !ok {
		return nil, &models.Error{
			Code:  http.StatusNotFound,
			Type:  "DELETE",
			Model: "params",
			Key:   key,
		}
	} else {
		delete(p, key)
		return val, rt.SetParams(obj, p)
	}
}
func (rt *RequestTracker) AddParam(obj models.Paramer, key string, val interface{}) error {
	p := obj.GetParams()
	if _, ok := p[key]; !ok {
		p[key] = val
		return rt.SetParams(obj, p)
	}
	return &models.Error{
		Code:  http.StatusConflict,
		Model: "params",
		Key:   key,
	}
}

func (rt *RequestTracker) urlFor(scheme string, remoteIP net.IP, port int) string {
	return fmt.Sprintf("%s://%s", scheme, net.JoinHostPort(rt.dt.LocalIP(remoteIP), strconv.Itoa(port)))
}

func (rt *RequestTracker) ApiURL(remoteIP net.IP) string {
	return rt.urlFor("https", remoteIP, rt.dt.ApiPort)
}

func (rt *RequestTracker) FileURL(remoteIP net.IP) string {
	return rt.urlFor("http", remoteIP, rt.dt.StaticPort)
}

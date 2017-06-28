package backend

import (
	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend/index"
)

// Profile represents a set of key/values to use in
// template expansion.
//
// There is one special profile named 'global' that acts
// as a global set of parameters for the system.
//
// These can be assigned to a machine's profile list.
// swagger:model
type Profile struct {
	Validation
	// The name of the profile.  This must be unique across all
	// profiles.
	//
	// required: true
	Name string
	// A description of this profile.  This can contain any reference
	// information for humans you want associated with the profile.
	Description string
	// Any additional parameters that may be needed to expand templates
	// for BootEnv, as documented by that boot environment's
	// RequiredParams and OptionalParams.
	Params map[string]interface{}
	// Profiles can also have an associated list of Tasks
	Tasks []string

	p *DataTracker
}

func (p *Profile) HasTask(s string) bool {
	for _, p := range p.Tasks {
		if p == s {
			return true
		}
	}
	return false
}

func (p *Profile) Indexes() map[string]index.Maker {
	fix := AsProfile
	return map[string]index.Maker{
		"Key": index.MakeKey(),
		"Name": index.Make(
			true,
			"string",
			func(i, j store.KeySaver) bool { return fix(i).Name < fix(j).Name },
			func(ref store.KeySaver) (gte, gt index.Test) {
				refName := fix(ref).Name
				return func(s store.KeySaver) bool {
						return fix(s).Name >= refName
					},
					func(s store.KeySaver) bool {
						return fix(s).Name > refName
					}
			},
			func(s string) (store.KeySaver, error) {
				return &Profile{Name: s}, nil
			}),
	}
}

func (p *Profile) Backend() store.SimpleStore {
	return p.p.getBackend(p)
}

func (p *Profile) Prefix() string {
	return "profiles"
}

func (p *Profile) Key() string {
	return p.Name
}

func (p *Profile) GetParams() map[string]interface{} {
	m := p.Params
	if m == nil {
		m = map[string]interface{}{}
	}
	return m
}

func (p *Profile) SetParams(values map[string]interface{}) error {
	p.Params = values
	e := &Error{Code: 409, Type: ValidationError, o: p}
	_, e2 := p.p.save(p)
	e.Merge(e2)
	return e.OrNil()
}

func (p *Profile) GetParam(key string, searchProfiles bool) (interface{}, bool) {
	mm := p.GetParams()
	if v, found := mm[key]; found {
		return v, true
	}
	return nil, false
}

func (p *Profile) New() store.KeySaver {
	res := &Profile{Name: p.Name, p: p.p}
	return store.KeySaver(res)
}

func (p *Profile) setDT(dp *DataTracker) {
	p.p = dp
}

func (p *Profile) OnCreate() error {
	e := &Error{Code: 409, Type: ValidationError, o: p}
	// We do not allow duplicate profile names
	profiles := AsProfiles(p.p.unlockedFetchAll(p.Prefix()))
	for _, pp := range profiles {
		if pp.Name == p.Name {
			e.Errorf("Profile %s is already exists", p.Name)
			return e
		}
	}
	return nil
}

func (p *Profile) BeforeDelete() error {
	e := &Error{Code: 422, Type: ValidationError, o: p}
	objs, unlocker := p.p.lockEnts("machines")
	defer unlocker()
	machines := objs[0]
	for i := range machines.d {
		m := AsMachine(machines.d[i])
		if m.HasProfile(p.Name) {
			e.Errorf("Machine %s is using profile %s", m.UUID(), p.Name)
		}
	}
	return e.OrNil()
}

func (p *Profile) OnLoad() error {
	if p.Params == nil {
		p.Params = map[string]interface{}{}
	}
	return nil
}

func (p *Profile) List() []*Profile {
	return AsProfiles(p.p.FetchAll(p))
}

func (p *DataTracker) NewProfile() *Profile {
	return &Profile{p: p, Params: map[string]interface{}{}}
}

func AsProfile(o store.KeySaver) *Profile {
	return o.(*Profile)
}

func AsProfiles(o []store.KeySaver) []*Profile {
	res := make([]*Profile, len(o))
	for i := range o {
		res[i] = AsProfile(o[i])
	}
	return res
}

func (p *Profile) BeforeSave() error {
	err := &Error{Code: 422, Type: ValidationError, o: p}
	err.Merge(index.CheckUnique(p, p.p.objs[p.Prefix()].d))
	objs, unlocker := p.p.lockEnts("params")
	defer unlocker()
	for k, v := range p.Params {
		pIdx, found := objs[0].find(k)
		if !found {
			continue
		}
		param := AsParam(objs[0].d[pIdx])
		err.Merge(param.Validate(v))
	}
	return err.OrNil()
}

func (p *Profile) AfterSave() {
	p.deferred(func() bool {
		if len(p.Params) == 0 {
			return true
		}
		objs, unlocker := p.p.lockEnts("tasks", "profiles")
		defer unlocker()
		err := &Error{o: p}
		for i, taskName := range p.Tasks {
			if _, found := objs[0].find(taskName); !found {
				err.Errorf("Task %s (at %d) does not exist", taskName, i)
			}
		}
		p.Available = !err.ContainsError()
		p.Errors = err.Messages
		return true
	})
}

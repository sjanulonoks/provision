package backend

import (
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

// Pref tracks a global DigitalRebar Provision preference -- things like the
// bootenv to use for unknown systems trying to PXE boot to us, the
// default bootenv for known systems, etc.
//
type Pref struct {
	*models.Pref
	validate
	p *DataTracker
}

func (p *Pref) Indexes() map[string]index.Maker {
	fix := AsPref
	return map[string]index.Maker{
		"Key": index.MakeKey(),
		"Name": index.Make(
			true,
			"string",
			func(i, j models.Model) bool { return fix(i).Name < fix(j).Name },
			func(ref models.Model) (gte, gt index.Test) {
				refName := fix(ref).Name
				return func(s models.Model) bool {
						return fix(s).Name >= refName
					},
					func(s models.Model) bool {
						return fix(s).Name > refName
					}
			},
			func(s string) (models.Model, error) {
				pref := fix(p.New())
				pref.Name = s
				return pref, nil
			}),
	}
}

func (p *Pref) Backend() store.Store {
	return p.p.getBackend(p)
}

func (p *Pref) New() store.KeySaver {
	return &Pref{Pref: &models.Pref{}}
}

func (p *Pref) setDT(dt *DataTracker) {
	p.p = dt
}

func AsPref(v models.Model) *Pref {
	return v.(*Pref)
}

var prefLockMap = map[string][]string{
	"get":    []string{"preferences"},
	"create": []string{"preferences", "bootenvs"},
	"update": []string{"preferences", "bootenvs"},
	"patch":  []string{"preferences", "bootenvs"},
	"delete": []string{"preferences", "bootenvs"},
}

func (p *Pref) Locks(action string) []string {
	return prefLockMap[action]
}

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
}

func (p *Pref) Indexes() map[string]index.Maker {
	fix := AsPref
	res := index.MakeBaseIndexes(p)
	res["Name"] = index.Make(
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
		})
	return res
}

func (p *Pref) New() store.KeySaver {
	res := &Pref{Pref: &models.Pref{}}
	res.rt = p.rt
	return res
}

func AsPref(v models.Model) *Pref {
	return v.(*Pref)
}

var prefLockMap = map[string][]string{
	"get":    []string{"preferences"},
	"create": []string{"preferences", "bootenvs", "stages"},
	"update": []string{"preferences", "bootenvs", "stages"},
	"patch":  []string{"preferences", "bootenvs", "stages"},
	"delete": []string{"preferences", "bootenvs", "stages"},
}

func (p *Pref) Locks(action string) []string {
	return prefLockMap[action]
}

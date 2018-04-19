package backend

import (
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

type Role struct {
	*models.Role
	validate
}

func (r *Role) SaveClean() store.KeySaver {
	mod := *r.Role
	mod.ClearValidation()
	return ModelToBackend(&mod)
}

func AsRole(r models.Model) *Role {
	return r.(*Role)
}

func AsRoles(o []models.Model) []*Role {
	res := make([]*Role, len(o))
	for i := range o {
		res[i] = AsRole(o[i])
	}
	return res
}

func (r *Role) New() store.KeySaver {
	res := &Role{Role: &models.Role{}}
	res.Fill()
	res.rt = r.rt
	return res
}

func (r *Role) Indexes() map[string]index.Maker {
	fix := AsRole
	res := index.MakeBaseIndexes(r)
	res["Name"] = index.Make(
		true,
		"string",
		func(i, j models.Model) bool {
			return fix(i).Name < fix(j).Name
		},
		func(ref models.Model) (gte, gt index.Test) {
			name := fix(ref).Name
			return func(s models.Model) bool {
					return fix(s).Name >= name
				},
				func(s models.Model) bool {
					return fix(s).Name > name
				}
		},
		func(s string) (models.Model, error) {
			res := fix(r.New())
			res.Name = s
			return res, nil
		})
	return res
}

var roleLockMap = map[string][]string{
	"get":     []string{"roles"},
	"create":  []string{"roles"},
	"update":  []string{"roles"},
	"patch":   []string{"roles"},
	"delete":  []string{"users", "roles"},
	"actions": []string{"roles"},
}

func (r *Role) Locks(action string) []string {
	return roleLockMap[action]
}

func (r *Role) Validate() {
	r.Role.Validate()
	r.AddError(index.CheckUnique(r, r.rt.stores("roles").Items()))
	r.SetValid()
	r.SetAvailable()
}

func (r *Role) BeforeSave() error {
	r.Validate()
	if !r.Validated {
		return r.MakeError(422, ValidationError, r)
	}
	return nil
}

func (r *Role) OnLoad() error {
	defer func() { r.rt = nil }()
	r.Fill()
	return r.BeforeSave()
}

func (r *Role) BeforeDelete() error {
	for _, u := range r.rt.d("users").Items() {
		user := AsUser(u)
		for _, name := range user.Roles {
			if name == r.Name {
				r.Errorf("In use by User %s", user.Name)
				break
			}
		}
	}
	return r.HasError()
}

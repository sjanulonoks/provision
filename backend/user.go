package backend

import (
	"strings"

	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
	sc "github.com/elithrar/simple-scrypt"
)

// User is an API user of DigitalRebar Provision
// swagger:model
type User struct {
	*models.User
	validate
	p *DataTracker
}

func (obj *User) SetReadOnly(b bool) {
	obj.ReadOnly = b
}

func (obj *User) SaveClean() store.KeySaver {
	mod := *obj.User
	mod.ClearValidation()
	return toBackend(obj.p, nil, &mod)
}

func (p *User) Indexes() map[string]index.Maker {
	fix := AsUser
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
			u := fix(p.New())
			u.Name = s
			return u, nil
		})
	return res
}

func (u *User) Backend() store.Store {
	return u.p.getBackend(u)
}

func (u *User) New() store.KeySaver {
	res := &User{User: &models.User{}}
	if u.User != nil && u.ChangeForced() {
		res.ForceChange()
	}
	res.p = u.p
	return res
}

func (u *User) setDT(p *DataTracker) {
	u.p = p
}

func AsUser(o models.Model) *User {
	return o.(*User)
}

func AsUsers(o []models.Model) []*User {
	res := make([]*User, len(o))
	for i := range o {
		res[i] = AsUser(o[i])
	}
	return res
}

func (u *User) ChangePassword(d Stores, newPass string) error {
	ph, err := sc.GenerateFromPassword([]byte(newPass), sc.DefaultParams)
	if err != nil {
		return err
	}
	u.PasswordHash = ph
	if u.p != nil {
		_, err = u.p.Save(d, u)
	}
	return err
}

func (u *User) Validate() {
	u.AddError(index.CheckUnique(u, u.stores("users").Items()))
	if strings.Contains(u.Name, "/") || strings.Contains(u.Name, "\\") {
		u.Errorf("Name must not contain a '/' or '\\'")
	}
	u.SetValid()
	u.SetAvailable()
}

func (u *User) BeforeSave() error {
	if u.Secret == "" {
		u.Secret = randString(16)
	}
	u.Validate()
	if !u.Useable() {
		return u.MakeError(422, ValidationError, u)
	}
	return nil
}

func (u *User) OnLoad() error {
	u.stores = func(ref string) *Store {
		return u.p.objs[ref]
	}
	defer func() { u.stores = nil }()

	// This mustSave part is just to keep us from resaving all the users on startup.
	mustSave := false
	if u.Secret == "" {
		mustSave = true
	}
	err := u.BeforeSave()
	if err == nil && mustSave {
		v := u.SaveValidation()
		u.ClearValidation()
		err = u.stores("machines").backingStore.Save(u.Key(), u)
		u.RestoreValidation(v)
	}
	return err
}

var userLockMap = map[string][]string{
	"get":    []string{"users"},
	"create": []string{"users"},
	"update": []string{"users"},
	"patch":  []string{"users"},
	"delete": []string{"users"},
}

func (u *User) Locks(action string) []string {
	return userLockMap[action]
}

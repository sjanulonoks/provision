package backend

import (
	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend/index"
	sc "github.com/elithrar/simple-scrypt"
)

// User is an API user of DigitalRebar Provision
// swagger:model
type User struct {
	// Name is the name of the user
	//
	// required: true
	Name string
	// PasswordHash is the scrypt-hashed version of the user's Password.
	//
	// swagger:strfmt password
	PasswordHash []byte `json:",omitempty"`
	p            *DataTracker
}

func (u *User) Prefix() string {
	return "users"
}

func (p *User) Indexes() map[string]index.Maker {
	fix := AsUser
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
				return &User{Name: s}, nil
			}),
	}
}

func (u *User) Key() string {
	return u.Name
}

func (u *User) Backend() store.SimpleStore {
	return u.p.getBackend(u)
}

func (u *User) New() store.KeySaver {
	return &User{p: u.p}
}

func (u *User) setDT(p *DataTracker) {
	u.p = p
}

func (u *User) CheckPassword(pass string) bool {
	if err := sc.CompareHashAndPassword(u.PasswordHash, []byte(pass)); err == nil {
		return true
	}
	return false
}

func (u *User) List() []*User {
	return AsUsers(u.p.FetchAll(u))
}

func AsUser(o store.KeySaver) *User {
	return o.(*User)
}

func AsUsers(o []store.KeySaver) []*User {
	res := make([]*User, len(o))
	for i := range o {
		res[i] = AsUser(o[i])
	}
	return res
}

func (u *User) Sanitize() {
	u.PasswordHash = []byte{}
}

func (u *User) ChangePassword(newPass string) error {
	ph, err := sc.GenerateFromPassword([]byte(newPass), sc.DefaultParams)
	if err != nil {
		return err
	}
	u.PasswordHash = ph
	if u.p != nil {
		_, err = u.p.save(u)
	}
	return err
}

func (p *DataTracker) NewUser() *User {
	return &User{p: p}
}

func (u *User) BeforeSave() error {
	return index.CheckUnique(u, u.p.objs[u.Prefix()].d)
}

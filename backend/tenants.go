package backend

import (
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

type Tenant struct {
	*models.Tenant
	validate
	cachedExpansion map[string]map[string]struct{}
	userAdd, userRm []string
}

func (t *Tenant) ExpandedMembers() map[string]map[string]struct{} {
	if t.cachedExpansion == nil {
		res := map[string]map[string]struct{}{}
		for k := range t.Members {
			res[k] = map[string]struct{}{}
			for idx := range t.Members[k] {
				res[k][t.Members[k][idx]] = struct{}{}
			}
		}
		t.cachedExpansion = res
	}
	return t.cachedExpansion
}

func (t *Tenant) SaveClean() store.KeySaver {
	mod := *t.Tenant
	mod.ClearValidation()
	return ModelToBackend(&mod)
}

func AsTenant(t models.Model) *Tenant {
	return t.(*Tenant)
}

func AsTenants(o []models.Model) []*Tenant {
	res := make([]*Tenant, len(o))
	for i := range o {
		res[i] = AsTenant(o[i])
	}
	return res
}

func (t *Tenant) New() store.KeySaver {
	res := &Tenant{Tenant: &models.Tenant{}}
	res.Fill()
	res.rt = t.rt
	return res
}

func (t *Tenant) Indexes() map[string]index.Maker {
	fix := AsTenant
	res := index.MakeBaseIndexes(t)
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
			res := fix(t.New())
			res.Name = s
			return res, nil
		})
	return res
}

var tenantLockMap = map[string][]string{
	"get":     []string{"tenants"},
	"create":  []string{"tenants", "users"},
	"update":  []string{"tenants", "users"},
	"delete":  []string{"users", "tenants"},
	"actions": []string{"tenants"},
}

func (t *Tenant) Locks(action string) []string {
	return tenantLockMap[action]
}

func (t *Tenant) Validate() {
	t.Tenant.Validate()
	t.AddError(index.CheckUnique(t, t.rt.stores("tenants").Items()))
	t.SetValid()
	t.SetAvailable()
}

// todo: Actually validate that all the items the Tenant references still exist.
func (t *Tenant) BeforeSave() error {
	t.Validate()
	if !t.Validated {
		return t.MakeError(422, ValidationError, t)
	}
	t.SetValid()
	if t.userAdd != nil && len(t.userAdd) > 0 {
		for _, name := range t.Users {
			if t.rt.find("users", name) == nil {
				t.Errorf("User %s does not exist")
			}
		}
		uMap := map[string]string{}
		for _, un := range t.Users {
			uMap[un] = t.Name
		}
		for _, t2 := range t.rt.d("tenants").Items() {
			if t2.Key() == t.Key() {
				continue
			}
			tm := AsTenant(t2)
			for _, u2 := range tm.Users {
				if _, ok := uMap[u2]; ok {
					t.Errorf("User %s already in tenant %s, and users can only be in one tenant at a time")
				}
			}
		}
	}
	t.SetAvailable()
	if !t.Available {
		return t.MakeError(422, ValidationError, t)
	}
	return nil
}

func (t *Tenant) AfterSave() {
	t.cachedExpansion = nil
	if t.userRm != nil && len(t.userRm) > 0 {
		for _, u := range t.userRm {
			if u2 := t.rt.find("users", u); u2 != nil {
				AsUser(u2).activeTenant = ""
			}
		}
	}
	if t.userAdd != nil && len(t.userAdd) > 0 {
		for _, u := range t.userAdd {
			if u2 := t.rt.find("users", u); u2 != nil {
				AsUser(u2).activeTenant = t.Name
			}
		}
	}
	t.userAdd, t.userRm = nil, nil
}

func (t *Tenant) OnLoad() error {
	defer func() { t.rt = nil }()
	t.Fill()
	t.userAdd = t.Users
	return t.BeforeSave()
}

func (t *Tenant) OnCreate() error {
	t.userAdd = t.Users
	return nil
}

func (t *Tenant) OnChange(t2 store.KeySaver) error {
	t.userAdd, t.userRm = []string{}, []string{}
	oldT := AsTenant(t2)
	newU, oldU := map[string]struct{}{}, map[string]struct{}{}
	for _, u := range oldT.Users {
		oldU[u] = struct{}{}
	}
	for _, u := range t.Users {
		if _, ok := oldU[u]; !ok {
			t.userAdd = append(t.userAdd, u)
		}
		newU[u] = struct{}{}
	}
	for u := range oldU {
		if _, ok := newU[u]; !ok {
			t.userRm = append(t.userRm, u)
		}
	}
	return nil
}

func (t *Tenant) BeforeDelete() error {
	e := models.Error{Code: 409, Type: StillInUseError, Model: t.Prefix(), Key: t.Key()}
	if len(t.Users) != 0 {
		e.Errorf("Constraining Users: %v", t.Users)
	}
	return e.HasError()
}

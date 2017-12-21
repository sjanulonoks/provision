package backend

import (
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
	"github.com/xeipuuv/gojsonschema"
)

type Paramer interface {
	models.Model
	GetParams(Stores, bool) map[string]interface{}
	SetParams(Stores, map[string]interface{}) error
	GetParam(Stores, string, bool) (interface{}, bool)
	SetParam(Stores, string, interface{}) error
}

// Param represents metadata about a Parameter or a Preference.
// Specifically, it contains a description of what the information
// is for, detailed documentation about the param, and a JSON schema that
// the param must match to be considered valid.
// swagger:model
type Param struct {
	*models.Param
	validate
	validator *gojsonschema.Schema
}

func (obj *Param) SetReadOnly(b bool) {
	obj.ReadOnly = b
}

func (obj *Param) SaveClean() store.KeySaver {
	mod := *obj.Param
	mod.ClearValidation()
	return toBackend(&mod, obj.rt)
}

func AsParam(o models.Model) *Param {
	return o.(*Param)
}

func AsParams(o []models.Model) []*Param {
	res := make([]*Param, len(o))
	for i := range o {
		res[i] = AsParam(o[i])
	}
	return res
}

func (p *Param) New() store.KeySaver {
	res := &Param{Param: &models.Param{}}
	if p.Param != nil && p.ChangeForced() {
		res.ForceChange()
	}
	res.rt = p.rt
	return res
}

func (p *Param) Indexes() map[string]index.Maker {
	fix := AsParam
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
			param := fix(p.New())
			param.Name = s
			return param, nil
		})
	return res
}

func (p *Param) Validate() {
	p.Param.Validate()
	p.SetValid()
	p.SetAvailable()
}

func (p *Param) BeforeSave() error {
	p.Validate()
	if !p.Useable() {
		return p.MakeError(422, ValidationError, p)
	}
	return nil
	// Arguably, we should also detect when an attempted schema update happens
	// and verify that it does not break validation, or at least report on what
	// previously-valid values would become invalid.
	// However, I don't feel like writing that code for now, so ignore the problem.
}

func (p *Param) OnLoad() error {
	defer func() { p.rt = nil }()
	return p.BeforeSave()
}

func (p *Param) ValidateValue(val interface{}) error {
	if !p.Useable() {
		return p.MakeError(422, ValidationError, p)
	}
	if p.Schema == nil {
		return nil
	}
	if p.validator == nil {
		p.validator, _ = gojsonschema.NewSchema(gojsonschema.NewGoLoader(p.Schema))
	}
	res, err := p.validator.Validate(gojsonschema.NewGoLoader(val))
	if err != nil {
		return err
	}
	if res.Valid() {
		return nil
	}
	e := &models.Error{}
	for _, i := range res.Errors() {
		e.Errorf(i.String())
	}
	return e
}

var paramLockMap = map[string][]string{
	"get":    []string{"params"},
	"create": []string{"params", "profiles"},
	"update": []string{"params", "profiles"},
	"patch":  []string{"params", "profiles"},
	"delete": []string{"params", "profiles"},
}

func (p *Param) Locks(action string) []string {
	return paramLockMap[action]
}

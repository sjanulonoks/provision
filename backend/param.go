package backend

import (
	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend/index"
	"github.com/xeipuuv/gojsonschema"
)

// Param represents metadata about a Parameter or a Preference.
// Specifically, it contains a description of what the information
// is for, detailed documentation about the param, and a JSON schema that
// the param must match to be considered valid.
// swagger:model
type Param struct {
	validate
	// Name is the name of the param.  Params must be uniquely named.
	//
	// required: true
	Name string
	// Description is a one-line description of the parameter.
	Description string
	// Documentation details what the parameter does, what values it can
	// take, what it is used for, etc.
	Documentation string
	// Schema must be a valid JSONSchema as of draft v4.
	//
	// required: true
	Schema    interface{}
	p         *DataTracker
	validator *gojsonschema.Schema
}

func AsParam(o store.KeySaver) *Param {
	return o.(*Param)
}

func AsParams(o []store.KeySaver) []*Param {
	res := make([]*Param, len(o))
	for i := range o {
		res[i] = AsParam(o[i])
	}
	return res
}

func (p *Param) Backend() store.SimpleStore {
	return p.p.getBackend(p)
}

func (p *Param) Prefix() string {
	return "params"
}

func (p *Param) Key() string {
	return p.Name
}

func (p *Param) New() store.KeySaver {
	res := &Param{Name: p.Name, p: p.p}
	return store.KeySaver(res)
}

func (d *DataTracker) NewParam() *Param {
	return &Param{p: d}
}

func (p *Param) setDT(dp *DataTracker) {
	p.p = dp
}

func (p *Param) Indexes() map[string]index.Maker {
	fix := AsParam
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
				return &Param{Name: s}, nil
			}),
	}
}

func (p *Param) BeforeSave() error {
	schema, err := gojsonschema.NewSchema(gojsonschema.NewGoLoader(p.Schema))
	if err != nil {
		return err
	}
	p.validator = schema
	return nil
	// Arguably, we should also detect when an attempted schema update happens
	// and verify that it does not break validation, or at least report on what
	// previously-valid values would become invalid.
	// However, I don't feel like writing that code for now, so ignore the problem.
}

func (p *Param) Validate(val interface{}) error {
	res, err := p.validator.Validate(gojsonschema.NewGoLoader(val))
	if err != nil {
		return err
	}
	if res.Valid() {
		return nil
	}
	e := &Error{}
	for _, i := range res.Errors() {
		e.Errorf(i.String())
	}
	return e
}

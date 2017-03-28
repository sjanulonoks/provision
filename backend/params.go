package backend

import "github.com/digitalrebar/digitalrebar/go/common/store"

// swagger:model
type Param struct {
	p *DataTracker
	// Key part of the Key/Value parameter space
	// required: true
	Name string
	// Value part of the Key/Value parameter space
	// Any arbirtary structure can be stored.
	Value interface{}
}

func (p *Param) Prefix() string {
	return "parameters"
}

func (p *Param) Key() string {
	return p.Name
}

func (p *Param) Backend() store.SimpleStore {
	return p.p.getBackend(p)
}

func (p *Param) New() store.KeySaver {
	return &Param{p: p.p}
}

func (p *Param) setDT(dt *DataTracker) {
	p.p = dt
}

func AsParam(v store.KeySaver) *Param {
	return v.(*Param)
}

func (p *DataTracker) NewParam() *Param {
	return &Param{p: p}
}

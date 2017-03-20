package backend

import "github.com/digitalrebar/digitalrebar/go/common/store"

type Param struct {
	p     *DataTracker
	Name  string
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

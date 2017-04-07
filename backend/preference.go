package backend

import "github.com/digitalrebar/digitalrebar/go/common/store"

// Pref tracks a global DigitalRebar Provision preference -- things like the
// bootenv to use for unknown systems trying to PXE boot to us, the
// default bootenv for known systems, etc.
//
type Pref struct {
	p    *DataTracker
	Name string
	Val  string
}

func (p *Pref) Prefix() string {
	return "preferences"
}

func (p *Pref) Key() string {
	return p.Name
}

func (p *Pref) Backend() store.SimpleStore {
	return p.p.getBackend(p)
}

func (p *Pref) New() store.KeySaver {
	return &Pref{p: p.p}
}

func (p *Pref) setDT(dt *DataTracker) {
	p.p = dt
}

func AsPref(v store.KeySaver) *Pref {
	return v.(*Pref)
}

func (p *DataTracker) NewPref() *Pref {
	return &Pref{p: p}
}

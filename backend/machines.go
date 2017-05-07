package backend

import (
	"fmt"
	"math/big"
	"net"
	"path"
	"strings"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend/index"
	"github.com/pborman/uuid"
)

// Machine represents a single bare-metal system that the provisioner
// should manage the boot environment for.
// swagger:model
type Machine struct {
	// The name of the machine.  THis must be unique across all
	// machines, and by convention it is the FQDN of the machine,
	// although nothing enforces that.
	//
	// required: true
	// swagger:strfmt hostname
	Name string
	// A description of this machine.  This can contain any reference
	// information for humans you want associated with the machine.
	Description string
	// The UUID of the machine.
	// This is auto-created at Create time, and cannot change afterwards.
	//
	// required: true
	// swagger:strfmt uuid
	Uuid uuid.UUID
	// The IPv4 address of the machine that should be used for PXE
	// purposes.  Note that this field does not directly tie into DHCP
	// leases or reservations -- the provisioner relies solely on this
	// address when determining what to render for a specific machine.
	//
	// swagger:strfmt ipv4
	Address net.IP
	// The boot environment that the machine should boot into.  This
	// must be the name of a boot environment present in the backend.
	// If this field is not present or blank, the global default bootenv
	// will be used instead.
	BootEnv string
	// An array of profiles to apply to this machine in order when looking
	// for a parameter during rendering.
	Profiles []string
	//
	// The Machine specific Profile Data - only used for the map (name and other
	// fields not used
	Profile Profile
	// Errors keeps hold of any errors that happen while writing out
	// rendered templates for the current BootEnv.  This field should be
	// checked any time the boot environment is changed to verify that
	// the boot environment change is valid.
	Errors []string
	p      *DataTracker

	// used during AfterSave() and AfterRemove() to handle boot environment changes.
	toRemove renderers
	toRender renderers
}

func (n *Machine) Indexes() map[string]index.Maker {
	fix := AsMachine
	return map[string]index.Maker{
		"Key": index.MakeKey(),
		"Uuid": index.Make(
			true,
			"UUID string",
			func(i, j store.KeySaver) bool { return fix(i).Uuid.String() < fix(j).Uuid.String() },
			func(ref store.KeySaver) (gte, gt index.Test) {
				refUuid := fix(ref).Uuid.String()
				return func(s store.KeySaver) bool {
						return fix(s).Uuid.String() >= refUuid
					},
					func(s store.KeySaver) bool {
						return fix(s).Uuid.String() > refUuid
					}
			},
			func(s string) (store.KeySaver, error) {
				id := uuid.Parse(s)
				if id == nil {
					return nil, fmt.Errorf("Invalid UUID: %s", s)
				}
				return &Machine{Uuid: id}, nil
			}),
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
				return &Machine{Name: s}, nil
			}),
		"BootEnv": index.Make(
			false,
			"string",
			func(i, j store.KeySaver) bool { return fix(i).BootEnv < fix(j).BootEnv },
			func(ref store.KeySaver) (gte, gt index.Test) {
				refBootEnv := fix(ref).BootEnv
				return func(s store.KeySaver) bool {
						return fix(s).BootEnv >= refBootEnv
					},
					func(s store.KeySaver) bool {
						return fix(s).BootEnv > refBootEnv
					}
			},
			func(s string) (store.KeySaver, error) {
				return &Machine{BootEnv: s}, nil
			}),
		"Address": index.Make(
			false,
			"IP Address",
			func(i, j store.KeySaver) bool {
				n, o := big.Int{}, big.Int{}
				n.SetBytes(fix(i).Address.To16())
				o.SetBytes(fix(j).Address.To16())
				return n.Cmp(&o) == -1
			},
			func(ref store.KeySaver) (gte, gt index.Test) {
				addr := &big.Int{}
				addr.SetBytes(fix(ref).Address.To16())
				return func(s store.KeySaver) bool {
						o := big.Int{}
						o.SetBytes(fix(s).Address.To16())
						return o.Cmp(addr) != -1
					},
					func(s store.KeySaver) bool {
						o := big.Int{}
						o.SetBytes(fix(s).Address.To16())
						return o.Cmp(addr) == 1
					}
			},
			func(s string) (store.KeySaver, error) {
				addr := net.ParseIP(s)
				if addr == nil {
					return nil, fmt.Errorf("Invalid address: %s", s)
				}
				return &Machine{Address: addr}, nil
			}),
	}
}

func (n *Machine) Backend() store.SimpleStore {
	return n.p.getBackend(n)
}

// HexAddress returns Address in raw hexadecimal format, suitable for
// pxelinux and elilo usage.
func (n *Machine) HexAddress() string {
	return Hexaddr(n.Address)
}

func (n *Machine) ShortName() string {
	idx := strings.Index(n.Name, ".")
	if idx == -1 {
		return n.Name
	}
	return n.Name[:idx]
}

func (n *Machine) UUID() string {
	return n.Uuid.String()
}

func (n *Machine) Prefix() string {
	return "machines"
}

func (n *Machine) Path() string {
	return path.Join(n.Prefix(), n.UUID())
}

func (n *Machine) Key() string {
	return n.UUID()
}

func (n *Machine) HasProfile(name string) bool {
	for _, e := range n.Profiles {
		if e == name {
			return true
		}
	}
	return false
}

func (n *Machine) getProfile(key string) *Profile {
	if o, found := n.p.fetchOne(n.p.NewProfile(), key); found {
		p := AsProfile(o)
		return p
	}
	return nil
}

func (n *Machine) GetParams() map[string]interface{} {
	m := n.Profile.Params
	if m == nil {
		m = map[string]interface{}{}
	}
	return m
}

func (n *Machine) SetParams(values map[string]interface{}) error {
	n.Profile.Params = values
	e := &Error{Code: 409, Type: ValidationError, o: n}
	_, e2 := n.p.save(n)
	e.Merge(e2)
	return e.OrNil()
}

func (n *Machine) GetParam(key string, searchProfiles bool) (interface{}, bool) {
	mm := n.GetParams()
	if v, found := mm[key]; found {
		return v, true
	}
	if searchProfiles {
		for _, e := range n.Profiles {
			if p := n.getProfile(e); p != nil {
				if v, ok := p.GetParam(key, false); ok {
					return v, true
				}
			}
		}
	}
	return nil, false
}

func (n *Machine) New() store.KeySaver {
	res := &Machine{Name: n.Name, Uuid: n.Uuid, p: n.p}
	return store.KeySaver(res)
}

func (n *Machine) setDT(p *DataTracker) {
	n.p = p
}

func (n *Machine) OnCreate() error {
	e := &Error{Code: 409, Type: ValidationError, o: n}
	// We do not allow duplicate machine names
	machines := AsMachines(n.p.unlockedFetchAll(n.Prefix()))
	for _, m := range machines {
		if m.Name == n.Name {
			e.Errorf("Machine %s is already named %s", m.UUID(), n.Name)
			return e
		}
	}
	return nil
}

func (n *Machine) BeforeSave() error {
	e := &Error{Code: 422, Type: ValidationError, o: n}
	if n.Uuid == nil {
		e.Errorf("Machine %#v was not assigned a uuid!", n)
	}
	if n.Name == "" {
		e.Errorf("Machine %s must have a name", n.Uuid)
	}
	if n.BootEnv == "" {
		n.BootEnv = n.p.defaultBootEnv
	}
	validateMaybeZeroIP4(e, n.Address)
	b, found := n.p.fetchOne(n.p.NewBootEnv(), n.BootEnv)
	if !found {
		e.Errorf("Machine %s has BootEnv %s, which is not present in the DataTracker", n.UUID(), n.BootEnv)
	} else {
		env := AsBootEnv(b)
		if !env.Available {
			e.Errorf("Machine %s wants BootEnv %s, which is not available", n.UUID(), n.BootEnv)
		} else if env.OnlyUnknown {
			e.Errorf("BootEnv %s does not allow Machine assignments, it has the OnlyUnknown flag.", env.Name)
		} else {
			n.toRender = env.Render(n, e)
		}
	}
	if err := index.CheckUnique(n, n.p.objs[n.Prefix()].d); err != nil {
		e.Merge(err)
	}
	return e.OrNil()
}

func (n *Machine) OnChange(oldThing store.KeySaver) error {
	e := &Error{Code: 422, Type: ValidationError, o: n}
	old := AsMachine(oldThing)
	be, found := n.p.fetchOne(n.p.NewBootEnv(), old.BootEnv)
	if found {
		n.toRemove = AsBootEnv(be).Render(n, e)
	}
	return e.OrNil()
}

func (n *Machine) AfterSave() {
	e := &Error{}
	if n.toRemove != nil {
		n.toRemove.deregister(n.p.FS)
		n.toRemove = nil
	}
	if n.toRender != nil {
		n.toRender.register(n.p.FS)
		n.toRender = nil
	}
	if e.containsError {
		n.Errors = e.Messages
	}
}

func (n *Machine) BeforeDelete() error {
	e := &Error{Code: 422, Type: ValidationError, o: n}
	b, found := n.p.fetchOne(n.p.NewBootEnv(), n.BootEnv)
	if !found {
		e.Errorf("Unable to find boot environment %s", n.BootEnv)
		return e
	}
	n.toRemove = AsBootEnv(b).Render(n, e)
	return e.OrNil()
}

func (n *Machine) AfterDelete() {
	e := &Error{}
	if n.toRemove != nil {
		n.toRemove.deregister(n.p.FS)
		n.toRemove = nil
	}
	if e.containsError {
		n.Errors = e.Messages
	}
}

func (b *Machine) List() []*Machine {
	return AsMachines(b.p.FetchAll(b))
}

func (p *DataTracker) NewMachine() *Machine {
	return &Machine{p: p}
}

func AsMachine(o store.KeySaver) *Machine {
	return o.(*Machine)
}

func AsMachines(o []store.KeySaver) []*Machine {
	res := make([]*Machine, len(o))
	for i := range o {
		res[i] = AsMachine(o[i])
	}
	return res
}

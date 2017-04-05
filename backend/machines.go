package backend

import (
	"net"
	"path"
	"strings"

	"github.com/digitalrebar/digitalrebar/go/common/store"
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
	// Any additional parameters that may be needed to expand templates
	// for BootEnv, as documented by that boot environment's
	// RequiredParams and OptionalParams.
	Params map[string]interface{}
	// Errors keeps hold of any errors that happen while writing out
	// rendered templates for the current BootEnv.  This field should be
	// checked any time the boot environment is changed to verify that
	// the boot environment change is valid.
	Errors []string
	p      *DataTracker

	// used during AfterSave() and AfterRemove() to handle boot environment changes.
	toRemove *RenderData
	toRender *RenderData
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
		} else {
			n.toRender = n.p.NewRenderData(n, env)
		}
	}
	return e.OrNil()
}

func (n *Machine) OnChange(oldThing store.KeySaver) error {
	e := &Error{Code: 422, Type: ValidationError, o: n}
	old := AsMachine(oldThing)
	be, found := n.p.fetchOne(n.p.NewBootEnv(), old.BootEnv)
	if found {
		n.toRemove = n.p.NewRenderData(n, AsBootEnv(be))
		n.toRemove.render(e)
	}
	return e.OrNil()
}

func (n *Machine) AfterSave() {
	e := &Error{}
	if n.toRemove != nil {
		n.toRemove.remove(e)
		n.toRemove = nil
	}
	if n.toRender != nil {
		n.toRender.render(e)
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
	n.toRemove = n.p.NewRenderData(n, AsBootEnv(b))
	n.toRemove.render(e)
	return e.OrNil()
}

func (n *Machine) AfterDelete() {
	e := &Error{}
	if n.toRemove != nil {
		n.toRemove.remove(e)
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

package backend

import (
	"fmt"
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
	// The FQDN of the machine.
	// required: true
	// swagger:strfmt hostname
	Name string
	// A description of this machine
	Description string
	// The UUID of the machine.
	// This is auto-created at Create time, and cannot change afterwards.
	// required: true
	// swagger:strfmt uuid
	Uuid uuid.UUID
	// The IPv4 address of the machine.  Specifically, the one
	// that should be used for PXE purposes
	// required: true
	// swagger:strfmt ipv4
	Address net.IP
	// The boot environment that the machine should boot into.
	BootEnv        string
	Params         map[string]interface{} // Any additional parameters that may be needed for template expansion.
	p              *DataTracker
	currentBootEnv *BootEnv
	oldBootEnv     *BootEnv
}

func (n *Machine) Backend() store.SimpleStore {
	return n.p.getBackend(n)
}

// HexAddress returns Address in raw hexadecimal format, suitable for
// pxelinux and elilo usage.
func (n *Machine) HexAddress() string {
	addr := n.Address.To4()
	hexIP := []byte(addr)
	return fmt.Sprintf("%02X%02X%02X%02X", hexIP[0], hexIP[1], hexIP[2], hexIP[3])
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

func (n *Machine) Url() string {
	return n.p.FileURL + "/" + n.Path()
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

func (n *Machine) BeforeSave() error {
	e := &Error{Code: 422, Type: ValidationError, o: n}
	if n.Uuid == nil {
		e.Errorf("Machine %#v was not assigned a uuid!", n)
	}
	if n.Name == "" {
		e.Errorf("Machine %s must have a name", n.Uuid)
	}
	if n.BootEnv == "" {
		n.BootEnv = n.p.DefaultBootEnv
	}
	validateMaybeZeroIP4(e, n.Address)
	b, found := n.p.FetchOne(n.p.NewBootEnv(), n.BootEnv)
	if !found {
		e.Errorf("Machine %s has BootEnv %s, which is not present in the DataTracker", n.Uuid, n.BootEnv)
	} else {
		n.currentBootEnv = AsBootEnv(b)
	}
	return e.OrNil()
}

func (n *Machine) OnChange(oldThing store.KeySaver) error {
	old := AsMachine(oldThing)
	if !uuid.Equal(old.Uuid, n.Uuid) {
		return fmt.Errorf("machine: Cannot change machine UUID %s", old.Uuid)
	} else if old.Name != n.Name {
		return fmt.Errorf("machine: Cannot change name of machine %s", old.Name)
	}
	be, found := n.p.FetchOne(n.p.NewBootEnv(), old.BootEnv)
	if found {
		n.oldBootEnv = AsBootEnv(be)
	}
	return nil
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

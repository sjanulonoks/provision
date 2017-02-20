package backend

import (
	"net"
	"time"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

// Lease models a DHCP Lease
// swagger:model
type Lease struct {
	// Addr is the IP address that the lease handed out.
	//
	// required: true
	// swagger:strfmt ipv4
	Addr net.IP
	// Mac is the hardware address of the device the lease is bound to.
	//
	// required: true
	// swagger:strfmt mac
	Mac string
	// Valid tracks whether the lease is valid
	//
	// required: true
	Valid bool
	// ExpireTime is the time at which the lease expires and is no longer valid
	//
	// required: true
	// swagger:strfmt date-time
	ExpireTime time.Time
	p          *DataTracker
}

func (l *Lease) Prefix() string {
	return "leases"
}

func (l *Lease) Key() string {
	return l.Mac
}

func (l *Lease) Backend() store.SimpleStore {
	return l.p.getBackend(l)
}

func (l *Lease) New() store.KeySaver {
	return &Lease{p: l.p}
}

func (l *Lease) List() []*Lease {
	return AsLeases(l.p.FetchAll(l))
}

func (p *DataTracker) NewLease() *Lease {
	return &Lease{p: p}
}

func AsLease(o store.KeySaver) *Lease {
	return o.(*Lease)
}

func AsLeases(o []store.KeySaver) []*Lease {
	res := make([]*Lease, len(o))
	for i := range o {
		res[i] = AsLease(o[i])
	}
	return res
}

func (l *Lease) BeforeSave() error {
	res := &Error{Code: 422, Type: ValidationError, o: l}
	validateIP4(res, l.Addr)
	validateMac(res, l.Mac)
	return res.OrNil()
}

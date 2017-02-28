package backend

import (
	"net"
	"time"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

var hexDigit = []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'}

func hexaddr(addr net.IP) string {
	b := addr.To4()
	s := make([]byte, len(b)*2)
	for i, tn := range b {
		s[i*2], s[i*2+1] = hexDigit[tn>>4], hexDigit[tn&0xf]
	}
	return string(s)
}

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
	// Strategy is the leasing strategy that will be used determine what to use from
	// the DHCP packet to handle lease management.
	//
	// required: true
	Strategy string

	p *DataTracker
}

func (l *Lease) Prefix() string {
	return "leases"
}

func (l *Lease) subnet() *Subnet {
	subnets := AsSubnets(l.p.fetchAll(l.p.NewSubnet()))
	for i := range subnets {
		if subnets[i].InSubnetRange(l.Addr) {
			return subnets[i]
		}
	}
	return nil
}

func (l *Lease) Key() string {
	return hexaddr(l.Addr)
}

func (l *Lease) Backend() store.SimpleStore {
	return l.p.getBackend(l)
}

func (l *Lease) New() store.KeySaver {
	return &Lease{p: l.p}
}

func (l *Lease) List() []*Lease {
	return AsLeases(l.p.fetchAll(l))
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

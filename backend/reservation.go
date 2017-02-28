package backend

import (
	"net"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

// Reservation tracks persistent DHCP IP address reservations.
// swagger:model
type Reservation struct {
	// Addr is the IP address permanently assigned to the Mac
	//
	// required: true
	// swagger:strfmt ipv4
	Addr net.IP
	// Mac is the interface address to which Addr is permanently bound
	//
	// required: true
	// swagger:strfmt mac
	Mac string
	// NextServer is the address the server should contact next.
	//
	// required: false
	// swagger:strfmt ipv4
	NextServer net.IP
	// Options is the list of DHCP options that apply to this Reservation
	Options []DhcpOption
	// Strategy is the leasing strategy that will be used determine what to use from
	// the DHCP packet to handle lease management.
	//
	// required: true
	Strategy string
	p        *DataTracker
}

func (r *Reservation) Prefix() string {
	return "reservations"
}

func (r *Reservation) Key() string {
	return hexaddr(r.Addr)
}

func (r *Reservation) Backend() store.SimpleStore {
	return r.p.getBackend(r)
}

func (r *Reservation) New() store.KeySaver {
	return &Reservation{p: r.p}
}

func (p *DataTracker) NewReservation() *Reservation {
	return &Reservation{p: p}
}

func (r *Reservation) List() []*Reservation {
	return AsReservations(r.p.FetchAll(r))
}

func AsReservation(o store.KeySaver) *Reservation {
	return o.(*Reservation)
}

func AsReservations(o []store.KeySaver) []*Reservation {
	res := make([]*Reservation, len(o))
	for i := range o {
		res[i] = AsReservation(o[i])
	}
	return res
}

func (r *Reservation) BeforeSave() error {
	e := &Error{Code: 422, Type: ValidationError, o: r}
	validateIP4(e, r.Addr)
	validateMac(e, r.Mac)
	validateMaybeZeroIP4(e, r.NextServer)
	if len(r.NextServer) == 0 || r.NextServer.IsUnspecified() {
		r.NextServer = nil
	}
	return e.OrNil()
}

func (r *Reservation) subnet() *Subnet {
	subnets := AsSubnets(r.p.fetchAll(r.p.NewSubnet()))
	for i := range subnets {
		if subnets[i].InSubnetRange(r.Addr) {
			return subnets[i]
		}
	}
	return nil
}

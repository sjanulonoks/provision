package backend

import (
	"math/big"
	"net"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend/index"
)

// Reservation tracks persistent DHCP IP address reservations.
//
// swagger:model
type Reservation struct {
	// Addr is the IP address permanently assigned to the strategy/token combination.
	//
	// required: true
	// swagger:strfmt ipv4
	Addr net.IP
	// Token is the unique identifier that the strategy for this Reservation should use.
	//
	// required: true
	Token string
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

func (l *Reservation) Indexes() map[string]index.Maker {
	fix := AsReservation
	return map[string]index.Maker{
		"Addr": index.Make(
			func(i, j store.KeySaver) bool {
				n, o := big.Int{}, big.Int{}
				n.SetBytes(fix(i).Addr.To16())
				o.SetBytes(fix(j).Addr.To16())
				return n.Cmp(&o) == -1
			},
			func(ref store.KeySaver) (gte, gt index.Test) {
				addr := &big.Int{}
				addr.SetBytes(fix(ref).Addr.To16())
				return func(s store.KeySaver) bool {
						o := big.Int{}
						o.SetBytes(fix(s).Addr.To16())
						return o.Cmp(addr) != -1
					},
					func(s store.KeySaver) bool {
						o := big.Int{}
						o.SetBytes(fix(s).Addr.To16())
						return o.Cmp(addr) == 1
					}
			}),
		"Token": index.Make(
			func(i, j store.KeySaver) bool { return fix(i).Token < fix(j).Token },
			func(ref store.KeySaver) (gte, gt index.Test) {
				token := fix(ref).Token
				return func(s store.KeySaver) bool {
						return fix(s).Token >= token
					},
					func(s store.KeySaver) bool {
						return fix(s).Token > token
					}
			}),
		"Strategy": index.Make(
			func(i, j store.KeySaver) bool { return fix(i).Strategy < fix(j).Strategy },
			func(ref store.KeySaver) (gte, gt index.Test) {
				strategy := fix(ref).Strategy
				return func(s store.KeySaver) bool {
						return fix(s).Strategy >= strategy
					},
					func(s store.KeySaver) bool {
						return fix(s).Strategy > strategy
					}
			}),
		"NextServer": index.Make(
			func(i, j store.KeySaver) bool {
				n, o := big.Int{}, big.Int{}
				n.SetBytes(fix(i).NextServer.To16())
				o.SetBytes(fix(j).NextServer.To16())
				return n.Cmp(&o) == -1
			},
			func(ref store.KeySaver) (gte, gt index.Test) {
				addr := &big.Int{}
				addr.SetBytes(fix(ref).NextServer.To16())
				return func(s store.KeySaver) bool {
						o := big.Int{}
						o.SetBytes(fix(s).NextServer.To16())
						return o.Cmp(addr) != -1
					},
					func(s store.KeySaver) bool {
						o := big.Int{}
						o.SetBytes(fix(s).NextServer.To16())
						return o.Cmp(addr) == 1
					}
			}),
	}
}

func (r *Reservation) Prefix() string {
	return "reservations"
}

func (r *Reservation) Key() string {
	return Hexaddr(r.Addr)
}

func (r *Reservation) Backend() store.SimpleStore {
	return r.p.getBackend(r)
}

func (r *Reservation) New() store.KeySaver {
	return &Reservation{p: r.p}
}

func (r *Reservation) setDT(p *DataTracker) {
	r.p = p
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

func (r *Reservation) OnChange(oldThing store.KeySaver) error {
	old := AsReservation(oldThing)
	e := &Error{Code: 422, Type: ValidationError, o: r}
	if r.Token != old.Token {
		e.Errorf("Token cannot change")
	}
	if r.Strategy != old.Strategy {
		e.Errorf("Strategy cannot change")
	}
	return e.OrNil()
}

func (r *Reservation) OnCreate() error {
	e := &Error{Code: 422, Type: ValidationError, o: r}
	// Make sure we aren't creating a reservation for a network or
	// a broadcast address in a subnet we know about
	subnets := AsSubnets(r.p.fetchAll(r.p.NewSubnet()))
	for i := range subnets {
		if !subnets[i].subnet().Contains(r.Addr) {
			continue
		}
		if !subnets[i].InSubnetRange(r.Addr) {
			e.Errorf("Address %s is a network or broadcast address for subnet %s", r.Addr.String(), subnets[i].Name)
		}
		break
	}
	return e.OrNil()
}

func (r *Reservation) BeforeSave() error {
	e := &Error{Code: 422, Type: ValidationError, o: r}
	validateIP4(e, r.Addr)
	validateMaybeZeroIP4(e, r.NextServer)
	if len(r.NextServer) == 0 || r.NextServer.IsUnspecified() {
		r.NextServer = nil
	}
	if r.Token == "" {
		e.Errorf("Reservation Token cannot be empty!")
	}
	if r.Strategy == "" {
		e.Errorf("Reservation Strategy cannot be empty!")
	}
	reservations := AsReservations(r.p.unlockedFetchAll("reservations"))
	for i := range reservations {
		if reservations[i].Addr.Equal(r.Addr) {
			continue
		}
		if reservations[i].Token == r.Token &&
			reservations[i].Strategy == r.Strategy {
			e.Errorf("Reservation %s alreay has Strategy %s: Token %s", reservations[i].Key(), r.Strategy, r.Token)
			break
		}
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

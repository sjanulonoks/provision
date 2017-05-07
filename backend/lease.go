package backend

import (
	"errors"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend/index"
)

var hexDigit = []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'}

func Hexaddr(addr net.IP) string {
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
	// Token is the unique token for this lease based on the
	// Strategy this lease used.
	//
	// required: true
	Token string
	// ExpireTime is the time at which the lease expires and is no
	// longer valid The DHCP renewal time will be half this, and the
	// DHCP rebind time will be three quarters of this.
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

func (l *Lease) Indexes() map[string]index.Maker {
	fix := AsLease
	return map[string]index.Maker{
		"Key": index.MakeKey(),
		"Addr": index.Make(
			false,
			"IP Address",
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
			},
			func(s string) (store.KeySaver, error) {
				ip := net.ParseIP(s)
				if ip == nil {
					return nil, errors.New("Addr must be an IP address")
				}
				return &Lease{Addr: ip}, nil
			}),
		"Token": index.Make(
			false,
			"string",
			func(i, j store.KeySaver) bool { return fix(i).Token < fix(j).Token },
			func(ref store.KeySaver) (gte, gt index.Test) {
				token := fix(ref).Token
				return func(s store.KeySaver) bool {
						return fix(s).Token >= token
					},
					func(s store.KeySaver) bool {
						return fix(s).Token > token
					}
			},
			func(s string) (store.KeySaver, error) {
				return &Lease{Token: s}, nil
			}),
		"Strategy": index.Make(
			false,
			"string",
			func(i, j store.KeySaver) bool { return fix(i).Strategy < fix(j).Strategy },
			func(ref store.KeySaver) (gte, gt index.Test) {
				strategy := fix(ref).Strategy
				return func(s store.KeySaver) bool {
						return fix(s).Strategy >= strategy
					},
					func(s store.KeySaver) bool {
						return fix(s).Strategy > strategy
					}
			},
			func(s string) (store.KeySaver, error) {
				return &Lease{Strategy: s}, nil
			}),
		"ExpireTime": index.Make(
			false,
			"Date/Time string",
			func(i, j store.KeySaver) bool { return fix(i).ExpireTime.Before(fix(j).ExpireTime) },
			func(ref store.KeySaver) (gte, gt index.Test) {
				expireTime := fix(ref).ExpireTime
				return func(s store.KeySaver) bool {
						ttime := fix(s).ExpireTime
						return ttime.Equal(expireTime) || ttime.After(expireTime)
					},
					func(s store.KeySaver) bool {
						return fix(s).ExpireTime.After(expireTime)
					}
			},
			func(s string) (store.KeySaver, error) {
				t := &time.Time{}
				if err := t.UnmarshalText([]byte(s)); err != nil {
					return nil, fmt.Errorf("ExpireTime is not valid: %v", err)
				}
				return &Lease{ExpireTime: *t}, nil
			}),
	}
}

func (l *Lease) Prefix() string {
	return "leases"
}

func (l *Lease) Subnet() *Subnet {
	subnets := AsSubnets(l.p.fetchAll(l.p.NewSubnet()))
	for i := range subnets {
		if subnets[i].subnet().Contains(l.Addr) {
			return subnets[i]
		}
	}
	return nil
}

func (l *Lease) Reservation() *Reservation {
	r, ok := l.p.fetchOne(l.p.NewReservation(), Hexaddr(l.Addr))
	if !ok {
		return nil
	}
	return AsReservation(r)
}

func (l *Lease) Key() string {
	return Hexaddr(l.Addr)
}

func (l *Lease) Backend() store.SimpleStore {
	return l.p.getBackend(l)
}

func (l *Lease) New() store.KeySaver {
	return &Lease{p: l.p}
}

func (l *Lease) setDT(p *DataTracker) {
	l.p = p
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

func (l *Lease) OnCreate() error {
	e := &Error{Code: 422, Type: ValidationError, o: l}
	validateIP4(e, l.Addr)
	if l.Token == "" {
		e.Errorf("Lease Token cannot be empty!")
	}
	if l.Strategy == "" {
		e.Errorf("Lease Strategy cannot be empty!")
	}
	// We can only create leases that have a Reservation or that are in
	// the ActiveRange of a subnet.
	if r := l.Reservation(); r != nil {
		return nil
	}
	if e.containsError {
		return e
	}
	leases := AsLeases(l.p.unlockedFetchAll("leases"))
	for i := range leases {
		if leases[i].Addr.Equal(l.Addr) {
			continue
		}
		if leases[i].Token == l.Token &&
			leases[i].Strategy == l.Strategy {
			e.Errorf("Lease %s alreay has Strategy %s: Token %s", leases[i].Key(), l.Strategy, l.Token)
			break
		}
	}
	if e.containsError {
		return e
	}
	if s := l.Subnet(); s == nil {
		e.Errorf("Cannot create Lease without a reservation or a subnet")
	} else if !s.InSubnetRange(l.Addr) {
		e.Errorf("Address %s is a network or broadcast address for subnet %s", l.Addr.String(), s.Name)
	}
	return e.OrNil()
}

func (l *Lease) OnChange(oldThing store.KeySaver) error {
	old := AsLease(oldThing)
	e := &Error{Code: 422, Type: ValidationError, o: l}
	if l.Token != old.Token {
		e.Errorf("Token cannot change")
	}
	if l.Strategy != old.Strategy {
		e.Errorf("Strategy cannot change")
	}
	return e.OrNil()
}

func (l *Lease) Expired() bool {
	return l.ExpireTime.Before(time.Now())
}

func (l *Lease) BeforeSave() error {
	return index.CheckUnique(l, l.p.objs[l.Prefix()].d)
}

func (l *Lease) Expire() {
	l.ExpireTime = time.Now()
}

func (l *Lease) Invalidate() {
	l.ExpireTime = time.Now().Add(2 * time.Second)
	l.Token = ""
	l.Strategy = ""
}

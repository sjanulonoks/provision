package backend

import (
	"errors"
	"fmt"
	"math/big"
	"net"
	"time"

	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

// Lease models a DHCP Lease
// swagger:model
type Lease struct {
	*models.Lease
	validate
	p *DataTracker
}

func (obj *Lease) SaveClean() store.KeySaver {
	mod := *obj.Lease
	mod.ClearValidation()
	return toBackend(obj.p, nil, &mod)
}

func (l *Lease) Indexes() map[string]index.Maker {
	fix := AsLease
	return map[string]index.Maker{
		"Key": index.MakeKey(),
		"Addr": index.Make(
			false,
			"IP Address",
			func(i, j models.Model) bool {
				n, o := big.Int{}, big.Int{}
				n.SetBytes(fix(i).Addr.To16())
				o.SetBytes(fix(j).Addr.To16())
				return n.Cmp(&o) == -1
			},
			func(ref models.Model) (gte, gt index.Test) {
				addr := &big.Int{}
				addr.SetBytes(fix(ref).Addr.To16())
				return func(s models.Model) bool {
						o := big.Int{}
						o.SetBytes(fix(s).Addr.To16())
						return o.Cmp(addr) != -1
					},
					func(s models.Model) bool {
						o := big.Int{}
						o.SetBytes(fix(s).Addr.To16())
						return o.Cmp(addr) == 1
					}
			},
			func(s string) (models.Model, error) {
				ip := net.ParseIP(s)
				if ip == nil {
					return nil, errors.New("Addr must be an IP address")
				}
				lease := fix(l.New())
				lease.Addr = ip
				return lease, nil
			}),
		"Token": index.Make(
			false,
			"string",
			func(i, j models.Model) bool { return fix(i).Token < fix(j).Token },
			func(ref models.Model) (gte, gt index.Test) {
				token := fix(ref).Token
				return func(s models.Model) bool {
						return fix(s).Token >= token
					},
					func(s models.Model) bool {
						return fix(s).Token > token
					}
			},
			func(s string) (models.Model, error) {
				lease := fix(l.New())
				lease.Token = s
				return lease, nil
			}),
		"Strategy": index.Make(
			false,
			"string",
			func(i, j models.Model) bool { return fix(i).Strategy < fix(j).Strategy },
			func(ref models.Model) (gte, gt index.Test) {
				strategy := fix(ref).Strategy
				return func(s models.Model) bool {
						return fix(s).Strategy >= strategy
					},
					func(s models.Model) bool {
						return fix(s).Strategy > strategy
					}
			},
			func(s string) (models.Model, error) {
				lease := fix(l.New())
				lease.Strategy = s
				return lease, nil
			}),
		"ExpireTime": index.Make(
			false,
			"Date/Time string",
			func(i, j models.Model) bool { return fix(i).ExpireTime.Before(fix(j).ExpireTime) },
			func(ref models.Model) (gte, gt index.Test) {
				expireTime := fix(ref).ExpireTime
				return func(s models.Model) bool {
						ttime := fix(s).ExpireTime
						return ttime.Equal(expireTime) || ttime.After(expireTime)
					},
					func(s models.Model) bool {
						return fix(s).ExpireTime.After(expireTime)
					}
			},
			func(s string) (models.Model, error) {
				t := &time.Time{}
				if err := t.UnmarshalText([]byte(s)); err != nil {
					return nil, fmt.Errorf("ExpireTime is not valid: %v", err)
				}
				lease := fix(l.New())
				lease.ExpireTime = *t
				return lease, nil
			}),
	}
}

func (l *Lease) Subnet(d Stores) *Subnet {
	subnets := d("subnets")
	for _, i := range subnets.Items() {
		subnet := AsSubnet(i)
		if subnet.subnet().Contains(l.Addr) {
			return subnet
		}
	}
	return nil
}

func (l *Lease) Reservation(d Stores) *Reservation {
	r := d("reservations").Find(models.Hexaddr(l.Addr))
	if r == nil {
		return nil
	}
	return AsReservation(r)
}

func (l *Lease) Backend() store.Store {
	return l.p.getBackend(l)
}

func (l *Lease) New() store.KeySaver {
	return &Lease{Lease: &models.Lease{}}
}

func (l *Lease) setDT(p *DataTracker) {
	l.p = p
}

func AsLease(o models.Model) *Lease {
	return o.(*Lease)
}

func AsLeases(o []models.Model) []*Lease {
	res := make([]*Lease, len(o))
	for i := range o {
		res[i] = AsLease(o[i])
	}
	return res
}

func (l *Lease) OnCreate() error {
	if r := l.Reservation(l.stores); r != nil {
		return nil
	}
	if s := l.Subnet(l.stores); s == nil {
		l.Errorf("Cannot create Lease without a reservation or a subnet")
	} else if !s.InSubnetRange(l.Addr) {
		l.Errorf("Address %s is a network or broadcast address for subnet %s", l.Addr.String(), s.Name)
	}
	return l.MakeError(422, ValidationError, l)
}

func (l *Lease) OnChange(oldThing store.KeySaver) error {
	old := AsLease(oldThing)
	if l.Token != old.Token {
		l.Errorf("Token cannot change")
	}
	if l.Strategy != old.Strategy {
		l.Errorf("Strategy cannot change")
	}
	return l.MakeError(422, ValidationError, l)
}

func (l *Lease) Expired() bool {
	return l.ExpireTime.Before(time.Now())
}

func (l *Lease) Validate() {
	l.AddError(index.CheckUnique(l, l.stores("leases").Items()))
	validateIP4(l, l.Addr)
	if l.Token == "" {
		l.Errorf("Lease Token cannot be empty!")
	}
	if l.Strategy == "" {
		l.Errorf("Lease Strategy cannot be empty!")
	}
	leases := AsLeases(l.stores("leases").Items())
	for i := range leases {
		if leases[i].Addr.Equal(l.Addr) {
			continue
		}
		if leases[i].Token == l.Token &&
			leases[i].Strategy == l.Strategy {
			l.Errorf("Lease %s alreay has Strategy %s: Token %s", leases[i].Key(), l.Strategy, l.Token)
			break
		}
	}
	l.SetValid()
	l.SetAvailable()
}

func (l *Lease) BeforeSave() error {
	l.Validate()
	if !l.Useable() {
		return l.MakeError(422, ValidationError, l)
	}
	return nil
}

func (l *Lease) OnLoad() error {
	return l.BeforeSave()
}

func (l *Lease) Expire() {
	l.ExpireTime = time.Now()
}

func (l *Lease) Invalidate() {
	l.ExpireTime = time.Now().Add(2 * time.Second)
	l.Token = ""
	l.Strategy = ""
}

var leaseLockMap = map[string][]string{
	"get":    []string{"leases"},
	"create": []string{"leases", "subnets", "reservations"},
	"update": []string{"leases"},
	"patch":  []string{"leases"},
	"delete": []string{"leases"},
}

func (l *Lease) Locks(action string) []string {
	return leaseLockMap[action]
}

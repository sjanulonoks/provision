package backend

import (
	"fmt"
	"math/big"
	"net"

	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

// Reservation tracks persistent DHCP IP address reservations.
//
// swagger:model
type Reservation struct {
	*models.Reservation
	validate
}

func (obj *Reservation) SetReadOnly(b bool) {
	obj.ReadOnly = b
}

func (obj *Reservation) SaveClean() store.KeySaver {
	mod := *obj.Reservation
	mod.ClearValidation()
	return toBackend(&mod, obj.rt)
}

func (l *Reservation) Indexes() map[string]index.Maker {
	fix := AsReservation
	res := index.MakeBaseIndexes(l)
	res["Addr"] = index.Make(
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
			addr := net.ParseIP(s)
			if addr == nil {
				return nil, fmt.Errorf("Invalid Address: %s", s)
			}
			res := fix(l.New())
			res.Addr = addr
			return res, nil
		})
	res["Token"] = index.Make(
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
			res := fix(l.New())
			res.Token = s
			return res, nil
		})
	res["Strategy"] = index.Make(
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
			res := fix(l.New())
			res.Strategy = s
			return res, nil
		})
	res["NextServer"] = index.Make(
		false,
		"IP Address",
		func(i, j models.Model) bool {
			n, o := big.Int{}, big.Int{}
			n.SetBytes(fix(i).NextServer.To16())
			o.SetBytes(fix(j).NextServer.To16())
			return n.Cmp(&o) == -1
		},
		func(ref models.Model) (gte, gt index.Test) {
			addr := &big.Int{}
			addr.SetBytes(fix(ref).NextServer.To16())
			return func(s models.Model) bool {
					o := big.Int{}
					o.SetBytes(fix(s).NextServer.To16())
					return o.Cmp(addr) != -1
				},
				func(s models.Model) bool {
					o := big.Int{}
					o.SetBytes(fix(s).NextServer.To16())
					return o.Cmp(addr) == 1
				}
		},
		func(s string) (models.Model, error) {
			addr := net.ParseIP(s)
			if addr == nil {
				return nil, fmt.Errorf("Invalid Address: %s", s)
			}
			res := fix(l.New())
			res.NextServer = addr
			return res, nil
		})
	return res
}

func (r *Reservation) New() store.KeySaver {
	res := &Reservation{Reservation: &models.Reservation{}}
	if r.Reservation != nil && r.ChangeForced() {
		res.ForceChange()
	}
	res.rt = r.rt
	return res
}

func AsReservation(o models.Model) *Reservation {
	return o.(*Reservation)
}

func AsReservations(o []models.Model) []*Reservation {
	res := make([]*Reservation, len(o))
	for i := range o {
		res[i] = AsReservation(o[i])
	}
	return res
}

func (r *Reservation) OnChange(oldThing store.KeySaver) error {
	old := AsReservation(oldThing)
	if r.Token != old.Token {
		r.Errorf("Token cannot change")
	}
	if r.Strategy != old.Strategy {
		r.Errorf("Strategy cannot change")
	}
	return r.MakeError(422, ValidationError, r)
}

func (r *Reservation) OnCreate() error {
	subnets := AsSubnets(r.rt.stores("subnets").Items())
	for i := range subnets {
		if !subnets[i].subnet().Contains(r.Addr) {
			continue
		}
		if !subnets[i].InSubnetRange(r.Addr) {
			r.Errorf("Address %s is a network or broadcast address for subnet %s", r.Addr.String(), subnets[i].Name)
		}
		break
	}
	return r.MakeError(422, ValidationError, r)
}

func (r *Reservation) Validate() {
	validateIP4(r, r.Addr)
	validateMaybeZeroIP4(r, r.NextServer)
	if len(r.NextServer) == 0 || r.NextServer.IsUnspecified() {
		r.NextServer = nil
	}
	if r.Token == "" {
		r.Errorf("Reservation Token cannot be empty!")
	}
	if r.Strategy == "" {
		r.Errorf("Reservation Strategy cannot be empty!")
	}
	reservations := AsReservations(r.rt.stores("reservations").Items())
	for i := range reservations {
		if reservations[i].Addr.Equal(r.Addr) {
			continue
		}
		if reservations[i].Token == r.Token &&
			reservations[i].Strategy == r.Strategy {
			r.Errorf("Reservation %s alreay has Strategy %s: Token %s", reservations[i].Key(), r.Strategy, r.Token)
			break
		}
	}
	r.AddError(index.CheckUnique(r, r.rt.stores("reservations").Items()))
	r.SetValid()
	r.SetAvailable()
}

func (r *Reservation) BeforeSave() error {
	r.Validate()
	if !r.Useable() {
		return r.MakeError(422, ValidationError, r)
	}
	return nil
}

func (r *Reservation) OnLoad() error {
	defer func() { r.rt = nil }()
	return r.BeforeSave()
}

var reservationLockMap = map[string][]string{
	"get":    []string{"reservations"},
	"create": []string{"reservations", "subnets"},
	"update": []string{"reservations"},
	"patch":  []string{"reservations"},
	"delete": []string{"reservations"},
}

func (r *Reservation) Locks(action string) []string {
	return reservationLockMap[action]
}

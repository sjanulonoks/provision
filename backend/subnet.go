package backend

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/big"
	"net"
	"sort"
	"time"

	"github.com/digitalrebar/store"
	"github.com/digitalrebar/provision/backend/index"
	dhcp "github.com/krolaw/dhcp4"
)

type picker func(*Subnet, map[string]store.KeySaver, string, net.IP) (*Lease, bool)

func pickNone(s *Subnet, usedAddrs map[string]store.KeySaver, token string, hint net.IP) (*Lease, bool) {
	// There are no free addresses, and don't fall through to using the most expired one.
	return nil, false
}

func pickMostExpired(s *Subnet, usedAddrs map[string]store.KeySaver, token string, hint net.IP) (*Lease, bool) {
	currLeases := []*Lease{}
	for _, obj := range usedAddrs {
		lease, ok := obj.(*Lease)
		if ok {
			currLeases = append(currLeases, lease)
		}
	}
	sort.Slice(currLeases,
		func(i, j int) bool {
			return currLeases[i].ExpireTime.Before(currLeases[j].ExpireTime)
		})
	for _, lease := range currLeases {
		if !lease.Expired() {
			// If we got to a non-expired lease, we are done
			break
		}
		// Because if how usedAddrs is built, we are guaranteed that an expired
		// lease here is not associated with a reservation.
		lease.Token = token
		lease.Strategy = s.Strategy
		return lease, false
	}
	return nil, true
}

func pickHint(s *Subnet, usedAddrs map[string]store.KeySaver, token string, hint net.IP) (*Lease, bool) {
	if hint == nil || !s.InActiveRange(hint) {
		return nil, true
	}
	hex := Hexaddr(hint)
	res, found := usedAddrs[hex]
	if !found {
		lease := &Lease{
			Addr:     hint,
			Token:    token,
			Strategy: s.Strategy,
		}
		return lease, false
	}
	if lease, ok := res.(*Lease); ok {
		if lease.Token == token && lease.Strategy == s.Strategy {
			// hey, we already have a lease.  How nice.
			return lease, false
		}
		if lease.Expired() {
			// We don't own this lease, but it is
			// expired, so we can steal it.
			lease.Token = token
			lease.Strategy = s.Strategy
			return lease, false
		}
	}
	return nil, false
}

func pickNextFree(s *Subnet, usedAddrs map[string]store.KeySaver, token string, hint net.IP) (*Lease, bool) {
	if s.nextLeasableIP == nil {
		s.nextLeasableIP = net.IP(make([]byte, 4))
		copy(s.nextLeasableIP, s.ActiveStart.To4())
	}
	one := big.NewInt(1)
	end := &big.Int{}
	curr := &big.Int{}
	end.SetBytes(s.ActiveEnd.To4())
	curr.SetBytes(s.nextLeasableIP.To4())
	// First, check from nextLeasableIp to ActiveEnd
	for curr.Cmp(end) < 1 {
		addr := net.IP(curr.Bytes()).To4()
		hex := Hexaddr(addr)
		curr.Add(curr, one)
		if _, ok := usedAddrs[hex]; !ok {
			s.nextLeasableIP = addr
			return &Lease{
				Addr:     addr,
				Token:    token,
				Strategy: s.Strategy,
			}, false
		}
	}
	// Next, check from ActiveStart to nextLeasableIP
	end.SetBytes(s.nextLeasableIP.To4())
	curr.SetBytes(s.ActiveStart.To4())
	for curr.Cmp(end) < 1 {
		addr := net.IP(curr.Bytes()).To4()
		hex := Hexaddr(addr)
		curr.Add(curr, one)
		if _, ok := usedAddrs[hex]; !ok {
			s.nextLeasableIP = addr
			return &Lease{
				Addr:     addr,
				Token:    token,
				Strategy: s.Strategy,
			}, false
		}
	}
	// No free address, but we can use the most expired one.
	return nil, true
}

var (
	pickStrategies = map[string]picker{}
)

func init() {
	pickStrategies["none"] = pickNone
	pickStrategies["hint"] = pickHint
	pickStrategies["nextFree"] = pickNextFree
	pickStrategies["mostExpired"] = pickMostExpired
}

// Subnet represents a DHCP Subnet
//
// swagger:model
type Subnet struct {
	validate
	// Name is the name of the subnet.
	// Subnet names must be unique
	//
	// required: true
	Name string
	// Enabled indicates if the subnet should hand out leases or continue operating
	// leases if already running.
	//
	// required: true
	Enabled bool
	// Subnet is the network address in CIDR form that all leases
	// acquired in its range will use for options, lease times, and NextServer settings
	// by default
	//
	// required: true
	// pattern: ^([0-9]+\.){3}[0-9]+/[0-9]+$
	Subnet string
	// NextServer is the address of the next server
	//
	// required: true
	// swagger:strfmt ipv4
	NextServer net.IP
	// ActiveStart is the first non-reserved IP address we will hand
	// non-reserved leases from.
	//
	// required: true
	// swagger:strfmt ipv4
	ActiveStart net.IP
	// ActiveEnd is the last non-reserved IP address we will hand
	// non-reserved leases from.
	//
	// required: true
	// swagger:strfmt ipv4
	ActiveEnd net.IP
	// ActiveLeaseTime is the default lease duration in seconds
	// we will hand out to leases that do not have a reservation.
	//
	// required: true
	ActiveLeaseTime int32
	// ReservedLeasTime is the default lease time we will hand out
	// to leases created from a reservation in our subnet.
	//
	// required: true
	ReservedLeaseTime int32
	// OnlyReservations indicates that we will only allow leases for which
	// there is a preexisting reservation.
	//
	// required: true
	OnlyReservations bool
	Options          []DhcpOption
	// Strategy is the leasing strategy that will be used determine what to use from
	// the DHCP packet to handle lease management.
	//
	// required: true
	Strategy string
	// Pickers is list of methods that will allocate IP addresses.
	// Each string must refer to a valid address picking strategy.  The current ones are:
	//
	// "none", which will refuse to hand out an address and refuse
	// to try any remaining strategies.
	//
	// "hint", which will try to reuse the address that the DHCP
	// packet is requesting, if it has one.  If the request does
	// not have a requested address, "hint" will fall through to
	// the next strategy. Otherwise, it will refuse to try ant
	// reamining strategies whether or not it can satisfy the
	// request.  This should force the client to fall back to
	// DHCPDISCOVER with no requsted IP address. "hint" will reuse
	// expired leases and unexpired leases that match on the
	// requested address, strategy, and token.
	//
	// "nextFree", which will try to create a Lease with the next
	// free address in the subnet active range.  It will fall
	// through to the next strategy if it cannot find a free IP.
	// "nextFree" only considers addresses that do not have a
	// lease, whether or not the lease is expired.
	//
	// "mostExpired" will try to recycle the most expired lease in the subnet's active range.
	//
	// All of the address allocation strategies do not consider
	// any addresses that are reserved, as lease creation will be
	// handled by the reservation instead.
	//
	// We will consider adding more address allocation strategies in the future.
	//
	// required: true
	Pickers []string

	p              *DataTracker
	nextLeasableIP net.IP
	sn             *net.IPNet
}

func (s *Subnet) Indexes() map[string]index.Maker {
	fix := AsSubnet
	return map[string]index.Maker{
		"Key": index.MakeKey(),
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
				return &Subnet{Name: s}, nil
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
				return &Subnet{Strategy: s}, nil
			}),
		"NextServer": index.Make(
			false,
			"IP Address",
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
			},
			func(s string) (store.KeySaver, error) {
				addr := net.ParseIP(s)
				if addr == nil {
					return nil, fmt.Errorf("Invalid Address: %s", s)
				}
				return &Subnet{NextServer: addr}, nil
			}),
		"Subnet": index.Make(
			true,
			"CIDR Address",
			func(i, j store.KeySaver) bool {
				a, _, errA := net.ParseCIDR(fix(i).Subnet)
				b, _, errB := net.ParseCIDR(fix(j).Subnet)
				if errA != nil || errB != nil {
					fix(i).p.Logger.Panicf("Illegal Subnets '%s', '%s'", fix(i).Subnet, fix(j).Subnet)
				}
				n, o := big.Int{}, big.Int{}
				n.SetBytes(a.To16())
				o.SetBytes(b.To16())
				return n.Cmp(&o) == -1
			},
			func(ref store.KeySaver) (gte, gt index.Test) {
				cidr, _, err := net.ParseCIDR(fix(ref).Subnet)
				if err != nil {
					fix(ref).p.Logger.Panicf("Illegal subnet %s: %v", fix(ref).Subnet, err)
				}
				addr := &big.Int{}
				addr.SetBytes(cidr.To16())
				return func(s store.KeySaver) bool {
						cidr, _, err := net.ParseCIDR(fix(s).Subnet)
						if err != nil {
							fix(s).p.Logger.Panicf("Illegal subnet %s: %v", fix(s).Subnet, err)
						}
						o := big.Int{}
						o.SetBytes(cidr.To16())
						return o.Cmp(addr) != -1
					},
					func(s store.KeySaver) bool {
						cidr, _, err := net.ParseCIDR(fix(s).Subnet)
						if err != nil {
							fix(s).p.Logger.Panicf("Illegal subnet %s: %v", fix(s).Subnet, err)
						}
						o := big.Int{}
						o.SetBytes(cidr.To16())
						return o.Cmp(addr) == 1
					}
			},
			func(s string) (store.KeySaver, error) {
				if _, _, err := net.ParseCIDR(s); err != nil {
					return nil, fmt.Errorf("Invalid subnet CIDR: %s", s)
				}
				return &Subnet{Subnet: s}, nil
			}),
		"Enabled": index.Make(
			false,
			"boolean",
			func(i, j store.KeySaver) bool {
				return (!fix(i).Enabled) && fix(j).Enabled
			},
			func(ref store.KeySaver) (gte, gt index.Test) {
				avail := fix(ref).Enabled
				return func(s store.KeySaver) bool {
						v := fix(s).Enabled
						return v || (v == avail)
					},
					func(s store.KeySaver) bool {
						return fix(s).Enabled && !avail
					}
			},
			func(s string) (store.KeySaver, error) {
				res := &Subnet{}
				switch s {
				case "true":
					res.Enabled = true
				case "false":
					res.Enabled = false
				default:
					return nil, errors.New("Enabled must be true or false")
				}
				return res, nil
			}),
	}
}

func (s *Subnet) subnet() *net.IPNet {
	if s.sn != nil {
		return s.sn
	}
	_, res, err := net.ParseCIDR(s.Subnet)
	if err != nil {
		panic(err.Error())
	}
	s.sn = res
	return res
}

func (s *Subnet) Prefix() string {
	return "subnets"
}

func (s *Subnet) Key() string {
	return s.Name
}

func (s *Subnet) AuthKey() string {
	return s.Key()
}

func (s *Subnet) Backend() store.Store {
	return s.p.getBackend(s)
}

func (s *Subnet) setDT(p *DataTracker) {
	s.p = p
}

func (s *Subnet) New() store.KeySaver {
	return &Subnet{p: s.p}
}

func (p *DataTracker) NewSubnet() *Subnet {
	return &Subnet{p: p}
}

func (s *Subnet) sBounds() (func(string) bool, func(string) bool) {
	sub := s.subnet()
	first := big.NewInt(0)
	mask := big.NewInt(0)
	last := big.NewInt(0)
	first.SetBytes([]byte(sub.IP.Mask(sub.Mask)))
	notBits := make([]byte, len(sub.Mask))
	for i, b := range sub.Mask {
		notBits[i] = ^b
	}
	mask.SetBytes(notBits)
	last.Or(first, mask)
	firstBytes := first.Bytes()
	lastBytes := last.Bytes()
	// first "address" in this range is the network address, which cannot be handed out.
	lower := func(key string) bool {
		return key > Hexaddr(net.IP(firstBytes))
	}
	// last "address" in this range is the broadcast address, which also cannot be handed out.
	upper := func(key string) bool {
		return key >= Hexaddr(net.IP(lastBytes))
	}
	return lower, upper
}

func (s *Subnet) aBounds() (func(string) bool, func(string) bool) {
	return func(key string) bool {
			return key >= Hexaddr(s.ActiveStart)
		},
		func(key string) bool {
			return key > Hexaddr(s.ActiveEnd)
		}
}

func (s *Subnet) idxABounds() (index.Test, index.Test) {
	return func(o store.KeySaver) bool {
			return o.Key() >= Hexaddr(s.ActiveStart)
		},
		func(o store.KeySaver) bool {
			return o.Key() > Hexaddr(s.ActiveStart)
		}
}

func (s *Subnet) InSubnetRange(ip net.IP) bool {
	lower, upper := s.sBounds()
	hex := Hexaddr(ip)
	return lower(hex) && !upper(hex)
}

func (s *Subnet) InActiveRange(ip net.IP) bool {
	lower, upper := s.aBounds()
	hex := Hexaddr(ip)
	return lower(hex) && !upper(hex)
}

func (s *Subnet) LeaseTimeFor(ip net.IP) time.Duration {
	if s.InActiveRange(ip) {
		return time.Duration(s.ActiveLeaseTime) * time.Second
	} else if s.InSubnetRange(ip) {
		return time.Duration(s.ReservedLeaseTime) * time.Second
	} else {
		return 0
	}
}

func AsSubnet(o store.KeySaver) *Subnet {
	return o.(*Subnet)
}

func AsSubnets(o []store.KeySaver) []*Subnet {
	res := make([]*Subnet, len(o))
	for i := range o {
		res[i] = AsSubnet(o[i])
	}
	return res
}

func (s *Subnet) BeforeSave() error {
	e := &Error{Code: 422, Type: ValidationError, o: s}
	_, subnet, err := net.ParseCIDR(s.Subnet)
	if err != nil {
		e.Errorf("Invalid subnet %s: %v", s.Subnet, err)
		return e
	} else {
		validateIP4(e, subnet.IP)
	}
	if s.Strategy == "" {
		e.Errorf("Strategy must have a value")
	}

	// Make sure that options have the netmask and broadcast options enabled
	needMask := true
	needBCast := true
	for _, opt := range s.Options {
		if opt.Code == dhcp.OptionBroadcastAddress {
			needBCast = false
		}
		if opt.Code == dhcp.OptionSubnetMask {
			needMask = false
		}
	}
	if needMask || needBCast {
		mask := net.IP([]byte(net.IP(subnet.Mask).To4()))
		if needMask {
			s.Options = append(s.Options, DhcpOption{dhcp.OptionSubnetMask, mask.String()})
		}
		if needBCast {
			bcastBits := binary.BigEndian.Uint32(subnet.IP) | ^binary.BigEndian.Uint32(mask)
			buf := make([]byte, 4)
			binary.BigEndian.PutUint32(buf, bcastBits)
			s.Options = append(s.Options, DhcpOption{dhcp.OptionBroadcastAddress, net.IP(buf).String()})
		}
	}

	if !s.OnlyReservations {
		validateIP4(e, s.ActiveStart)
		validateIP4(e, s.ActiveEnd)
		if !subnet.Contains(s.ActiveStart) {
			e.Errorf("ActiveStart %s not in subnet range %s", s.ActiveStart, subnet)
		}
		if !subnet.Contains(s.ActiveEnd) {
			e.Errorf("ActiveEnd %s not in subnet range %s", s.ActiveEnd, subnet)
		}
		startBytes := big.NewInt(0)
		endBytes := big.NewInt(0)
		startBytes.SetBytes(s.ActiveStart)
		endBytes.SetBytes(s.ActiveEnd)
		if startBytes.Cmp(endBytes) != -1 {
			e.Errorf("ActiveStart must be less than ActiveEnd")
		}
		if s.ActiveLeaseTime < 60 {
			e.Errorf("ActiveLeaseTime must be greater than or equal to 60 seconds, not %d", s.ActiveLeaseTime)
		}
	}
	if s.Pickers == nil || len(s.Pickers) == 0 {
		if s.OnlyReservations {
			s.Pickers = []string{"none"}
		} else {
			s.Pickers = []string{"hint", "nextFree", "mostExpired"}
		}
	}
	for _, p := range s.Pickers {
		_, ok := pickStrategies[p]
		if !ok {
			e.Errorf("Picker %s is not a valid lease picking strategy", p)
		}
	}
	if s.ReservedLeaseTime < 7200 {
		e.Errorf("ReservedLeaseTime must be greater than or equal to 7200 seconds, not %d", s.ReservedLeaseTime)
	}
	if e.containsError {
		return e
	}
	subnets := AsSubnets(s.stores("subnets").Items())
	for i := range subnets {
		if subnets[i].Name == s.Name {
			continue
		}
		if subnets[i].subnet().Contains(s.subnet().IP) {
			e.Errorf("Overlaps subnet %s", subnets[i].Name)
		}
	}
	e.Merge(index.CheckUnique(s, s.stores("subnets").Items()))
	return e.OrNil()
}

func (s *Subnet) next(used map[string]store.KeySaver, token string, hint net.IP) (*Lease, bool) {
	for _, p := range s.Pickers {
		l, f := pickStrategies[p](s, used, token, hint)
		if !f {
			return l, f
		}
	}
	return nil, false
}

var subnetLockMap = map[string][]string{
	"get":    []string{"subnets"},
	"create": []string{"subnets"},
	"update": []string{"subnets"},
	"patch":  []string{"subnets"},
	"delete": []string{"subnets"},
}

func (s *Subnet) Locks(action string) []string {
	return subnetLockMap[action]
}

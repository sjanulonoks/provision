package backend

import (
	"math/big"
	"net"
	"time"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

func pickNone(s *Subnet, usedAddrs map[string]bool) (net.IP, bool) {
	// There are no free addresses, and don't fall through to using the most expired one.
	return nil, false
}

func pickNext(s *Subnet, usedAddrs map[string]bool) (net.IP, bool) {
	if s.nextLeasableIP == nil {
		s.nextLeasableIP = net.IP(make([]byte, 4))
		copy(s.nextLeasableIP, s.ActiveStart.To4())
	}
	one := big.NewInt(1)
	end := &big.Int{}
	curr := &big.Int{}
	end.SetBytes(s.ActiveEnd.To4())
	curr.SetBytes(s.nextLeasableIP)
	// First, check from nextLeasableIp to ActiveEnd
	for curr.Cmp(end) != 1 {
		addr := net.IP(curr.Bytes()).To4()
		hex := Hexaddr(addr)
		curr.Add(curr, one)
		if _, ok := usedAddrs[hex]; !ok {
			s.nextLeasableIP = net.IP(curr.Bytes()).To4()
			return addr, false
		}
	}
	// Next, check from ActiveStart to nextLeasableIP
	end.SetBytes(s.nextLeasableIP)
	curr.SetBytes(s.ActiveStart.To4())
	for curr.Cmp(end) != 1 {
		addr := net.IP(curr.Bytes()).To4()
		hex := Hexaddr(addr)
		curr.Add(curr, one)
		if _, ok := usedAddrs[hex]; !ok {
			s.nextLeasableIP = net.IP(curr.Bytes()).To4()
			return addr, false
		}
	}
	// No free address, but we can use the most expired one.
	return nil, true
}

// Subnet represents a DHCP Subnet
//
// swagger:model
type Subnet struct {
	// Name is the name of the subnet.
	// Subnet names must be unique
	//
	// required: true
	Name string
	// Subnet is the network address in CIDR form that all leases
	// acquired in its range will use for options, lease times, and NextServer settings
	// by default
	//
	// required: true
	// pattern: ^([0-9]+\.){3}[0-9]+/[0-9]+$
	Subnet string
	// NextServer is the address of the next server
	//
	// swagger:strfmt ipv4
	// required: true
	NextServer net.IP
	// ActiveStart is the first non-reserved IP address we will hand
	// non-reserved leases from.
	//
	// swagger:strfmt ipv4
	// required: true
	ActiveStart net.IP
	// ActiveEnd is the last non-reserved IP address we will hand
	// non-reserved leases from.
	//
	// swagger:strfmt ipv4
	// required: true
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
	// Picker is the method that will allocate IP addresses.
	// Picker must be one of "none" or "next".  We may add more IP
	// address allocation strategies in the future.
	//
	// required: true
	Picker         string
	p              *DataTracker
	nextLeasableIP net.IP
	sn             *net.IPNet
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

func (s *Subnet) Backend() store.SimpleStore {
	return s.p.getBackend(s)
}

func (s *Subnet) New() store.KeySaver {
	return &Subnet{p: s.p}
}

func (p *DataTracker) NewSubnet() *Subnet {
	return &Subnet{p: p}
}

func (s *Subnet) List() []*Subnet {
	return AsSubnets(s.p.FetchAll(s))
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
	if s.ReservedLeaseTime < 7200 {
		e.Errorf("ReservedLeaseTime must be greater than or equal to 7200 seconds, not %d", s.ReservedLeaseTime)
	}
	if e.containsError {
		return e
	}
	subnets := AsSubnets(s.p.unlockedFetchAll("subnets"))
	for i := range subnets {
		if subnets[i].Name == s.Name {
			continue
		}
		if subnets[i].subnet().Contains(s.subnet().IP) {
			e.Errorf("Overlaps subnet %s", subnets[i].Name)
		}
	}
	return e.OrNil()
}

func (s *Subnet) leases() []*Lease {
	lower, upper := s.sBounds()
	return AsLeases(s.p.fetchSome("leases", lower, upper))
}

func (s *Subnet) activeLeases() []*Lease {
	lower, upper := s.aBounds()
	return AsLeases(s.p.fetchSome("leases", lower, upper))
}

func (s *Subnet) reservations() []*Reservation {
	lower, upper := s.sBounds()
	return AsReservations(s.p.fetchSome("reservations", lower, upper))
}

func (s *Subnet) next(used map[string]bool) (net.IP, bool) {
	switch s.Picker {
	case "nextFree", "":
		return pickNext(s, used)
	case "none":
		return pickNone(s, used)
	}
	return nil, false
}

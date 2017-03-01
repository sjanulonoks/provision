package backend

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

func TestSubnetCrud(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	createTests := []crudTest{
		{"Create empty Subnet", dt.create, &Subnet{p: dt}, false},
		{"Create valid Subnet", dt.create, &Subnet{p: dt, Name: "test", Subnet: "192.168.124.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, true},
		{"Create duplicate Subnet", dt.create, &Subnet{p: dt, Name: "test", Subnet: "192.168.124.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(bad Subnet)", dt.create, &Subnet{p: dt, Name: "test2", Subnet: "127.0.0.0", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(overlapping Subnet)", dt.create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.124.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(swapped Active range endpoints)", dt.create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.254"), ActiveEnd: net.ParseIP("192.168.125.80"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(ActiveStart out of range)", dt.create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.125.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(ActiveEnd out of range)", dt.create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.80"), ActiveEnd: net.ParseIP("192.168.126.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(ActiveLeaseTime too short)", dt.create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.80"), ActiveEnd: net.ParseIP("192.168.125.254"), ActiveLeaseTime: 59, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(ReservedLeaseTime too short)", dt.create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.80"), ActiveEnd: net.ParseIP("192.168.125.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7199, Strategy: "mac"}, false},
		{"Create invalid Subnet(no Strategy)", dt.create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.80"), ActiveEnd: net.ParseIP("192.168.125.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: ""}, false},
	}
	for _, test := range createTests {
		test.Test(t)
	}
}

func TestSubnetFinders(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	expTime := time.Now().Add(10 * time.Hour)
	objs := []crudTest{
		{"Create subnet for testing finders", dt.create, &Subnet{p: dt, Name: "test", Subnet: "192.168.124.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "noop"}, true},
		{"Create lease 1 (below subnet range)", dt.create, &Lease{p: dt, Addr: net.ParseIP("192.168.123.254"), Token: "lease1", Strategy: "noop", ExpireTime: expTime}, true},
		{"Create lease 2 (at start of subnet range)", dt.create, &Lease{p: dt, Addr: net.ParseIP("192.168.124.1"), Token: "lease2", Strategy: "noop", ExpireTime: expTime}, true},
		{"Create lease 3 (at start active range)", dt.create, &Lease{p: dt, Addr: net.ParseIP("192.168.124.80"), Token: "lease3", Strategy: "noop", ExpireTime: expTime}, true},
		{"Create lease 4 (in active range)", dt.create, &Lease{p: dt, Addr: net.ParseIP("192.168.124.128"), Token: "lease4", Strategy: "noop", ExpireTime: expTime}, true},
		{"Create lease 5 (at end of range)", dt.create, &Lease{p: dt, Addr: net.ParseIP("192.168.124.254"), Token: "lease5", Strategy: "noop", ExpireTime: expTime}, true},
		{"Create lease 6 (above subnet range)", dt.create, &Lease{p: dt, Addr: net.ParseIP("192.168.125.1"), Token: "lease6", Strategy: "noop", ExpireTime: expTime}, true},
		{"Create reservation 1 (below subnet range)", dt.create, &Reservation{p: dt, Addr: net.ParseIP("192.168.123.254"), Token: "lease1", Strategy: "noop"}, true},
		{"Create reservation 2 (at start of subnet range)", dt.create, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.1"), Token: "lease2", Strategy: "noop"}, true},
		{"Create reservation 3 (at end of subnet range)", dt.create, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.254"), Token: "lease5", Strategy: "noop"}, true},
		{"Create reservation 4 (after end of subnet range)", dt.create, &Reservation{p: dt, Addr: net.ParseIP("192.168.125.1"), Token: "lease6", Strategy: "noop"}, true},
		{"Create reservation 5 (towards middle of subnet)", dt.create, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.17"), Token: "lease0", Strategy: "noop"}, true},
	}
	for _, test := range objs {
		test.Test(t)
	}
	insboundsAddrs := []net.IP{
		net.ParseIP("192.168.124.1"),
		net.ParseIP("192.168.124.80"),
		net.ParseIP("192.168.124.128"),
		net.ParseIP("192.168.124.254"),
	}
	inaboundsAddrs := insboundsAddrs[1:]
	reservedAddrs := []net.IP{
		net.ParseIP("192.168.124.1"),
		net.ParseIP("192.168.124.17"),
		net.ParseIP("192.168.124.254"),
	}
	sub, ok := dt.FetchOne(dt.NewSubnet(), "test")
	if !ok {
		t.Errorf("Unable to find subnet %s", "test")
		return
	}
	s := AsSubnet(sub)
	allLeases := s.leases()
	activeLeases := s.activeLeases()
	allReservations := s.reservations()

	for i, addr := range insboundsAddrs {
		msg := fmt.Sprintf("Lease %d: Expected addr %s, got %s", i, addr.String(), allLeases[i].Addr.String())
		if addr.Equal(allLeases[i].Addr) {
			t.Log(msg)
		} else {
			t.Error(msg)
		}

	}
	for i, addr := range inaboundsAddrs {
		msg := fmt.Sprintf("Active Lease %d: Expected addr %s, got %s", i, addr.String(), activeLeases[i].Addr.String())
		if addr.Equal(activeLeases[i].Addr) {
			t.Log(msg)
		} else {
			t.Error(msg)
		}
	}
	for i, addr := range reservedAddrs {
		msg := fmt.Sprintf("Reservation %d: Expected addr %s, got %s", i, addr.String(), allReservations[i].Addr.String())
		if addr.Equal(allReservations[i].Addr) {
			t.Log(msg)
		} else {
			t.Error(msg)
		}
	}

}

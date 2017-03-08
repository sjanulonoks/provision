package backend

import (
	"net"
	"testing"
	"time"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

type ltf struct {
	strat, token string
	req          net.IP
	found, err   bool
}

func (l *ltf) find(t *testing.T, dt *DataTracker) {
	res, err := FindLease(dt, l.strat, l.token, l.req)
	if l.found {
		if res == nil {
			t.Errorf("Expected a lease for %s:%s, failed to get one", l.strat, l.token)
		} else if res.Strategy != l.strat || res.Token != l.token {
			t.Errorf("Expected lease to have %s:%s, has %s:%s", l.strat, l.token, res.Strategy, res.Token)
		} else if l.req != nil {
			if !res.Addr.Equal(l.req) {
				t.Errorf("Expected lease %s:%s to have address %s, it has %s", l.strat, l.token, l.req, res.Addr)
			}
		} else {
			t.Logf("Got lease %s:%s (%s)", res.Strategy, res.Token, res.Addr)
		}
	} else {
		if res != nil {
			t.Errorf("Did not expect to get lease, got %s:%s (%s)", res.Strategy, res.Token, res.Addr)
		} else {
			t.Logf("As expected, did not get lease for %s:%s", l.strat, l.token)
		}
	}
	if l.err {
		if err != nil {
			t.Logf("Got expected error %#v", err)
		} else {
			t.Errorf("Did not get an error when we expected one!")
		}
	} else {
		if err == nil {
			t.Logf("No error expected or found")
		} else {
			t.Errorf("Got unexpected error %#v", err)
		}
	}
}

func TestDHCPRenew(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	startObjs := []crudTest{
		{"Initial Subnet", dt.create, &Subnet{p: dt, Name: "sn", Subnet: "192.168.124.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, true},
		{"Initial Standalone Reservation", dt.create, &Reservation{p: dt, Addr: net.ParseIP("192.168.123.10"), Token: "res1", Strategy: "mac"}, true},
		{"Valid Subnet Lease", dt.create, &Lease{p: dt, Addr: net.ParseIP("192.168.124.80"), Strategy: "mac", Token: "subn1", ExpireTime: time.Now().Add(60 * time.Second)}, true},
		{"Valid Reservation Lease", dt.create, &Lease{p: dt, Addr: net.ParseIP("192.168.123.10"), Strategy: "mac", Token: "res1", ExpireTime: time.Now().Add(2 * time.Hour)}, true},
		{"Conflicting Reservation Lease", dt.create, &Lease{p: dt, Addr: net.ParseIP("192.168.124.81"), Strategy: "mac", Token: "subn2", ExpireTime: time.Now().Add(2 * time.Hour)}, true},
		{"Initial Conflicting Reservation", dt.create, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.81"), Token: "res2", Strategy: "mac"}, true},
	}
	for _, obj := range startObjs {
		obj.Test(t)
	}
	ltfs := []ltf{
		{"mac", "subn1", net.ParseIP("192.168.124.80"), true, false},
		{"mac", "res1", net.ParseIP("192.168.123.10"), true, false},
		{"mac", "subn1", nil, true, false},
		{"mac", "res1", nil, true, false},
		{"mac", "res1", net.ParseIP("192.168.124.90"), false, true},
		{"mac", "res5", nil, false, false},
		{"mac", "subn8", net.ParseIP("192.168.124.80"), false, true},
		{"mac", "subn2", net.ParseIP("192.168.124.81"), false, true},
	}
	for _, l := range ltfs {
		l.find(t, dt)
	}
	if ok, err := dt.remove(&Reservation{p: dt, Addr: net.ParseIP("192.168.123.10")}); !ok {
		t.Errorf("Failed to remove reservation for 192.168.123.10: %v", err)
	}
	if l, err := FindLease(dt, "mac", "res1", nil); err == nil {
		t.Errorf("Should have removed lease for %s:%s, as its backing reservation is gone!", l.Strategy, l.Token)
	} else {
		t.Logf("Removed lease that no longer has a Subnet or Reservation covering it: %v", err)
	}
}

type ltc struct {
	strat, token string
	req, via     net.IP
	created      bool
	expected     net.IP
}

func (l *ltc) test(t *testing.T, dt *DataTracker) {
	res := FindOrCreateLease(dt, l.strat, l.token, l.req, l.via)
	if l.created {
		if res == nil {
			t.Errorf("Expected to create a lease with %s:%s, but did not!", l.strat, l.token)
		} else if l.expected != nil && !res.Addr.Equal(l.expected) {
			t.Errorf("Lease %s:%s got %s, expected %s", l.strat, l.token, res.Addr, l.expected)
		} else {
			t.Logf("Created lease %s:%s: %s", res.Strategy, res.Token, res.Addr)
		}
	} else {
		if res != nil {
			t.Errorf("Did not expect to create lease %s:%s: %s", l.strat, l.token, res.Addr)
		} else {
			t.Log("No lease created, as expected")
		}
	}
}

func TestDHCPCreateReservationOnly(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	startObjs := []crudTest{
		{"Res1", dt.create, &Reservation{p: dt, Addr: net.ParseIP("192.168.123.10"), Token: "res1", Strategy: "mac"}, true},
		{"Res2", dt.create, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.10"), Token: "res2", Strategy: "mac"}, true},
	}
	for _, obj := range startObjs {
		obj.Test(t)
	}
	createTests := []ltc{
		{"mac", "res1", nil, nil, true, net.ParseIP("192.168.123.10")},
		{"mac", "resn", net.ParseIP("192.168.123.10"), nil, false, nil},
		{"mac", "res1", net.ParseIP("192.168.123.10"), nil, true, net.ParseIP("192.168.123.10")},
		{"mac", "res1", net.ParseIP("192.168.123.11"), nil, false, nil},
		{"mac", "res1", nil, nil, true, net.ParseIP("192.168.123.10")},
		{"mac", "resn", nil, nil, false, nil},
		{"mac", "res2", nil, nil, true, net.ParseIP("192.168.124.10")},
	}
	for _, obj := range createTests {
		obj.test(t, dt)
	}
	// Expire one lease
	lease := AsLease(dt.load("leases", Hexaddr(net.ParseIP("192.168.123.10"))))
	lease.ExpireTime = time.Now().Add(-2 * time.Second)
	lease.Token = "res3"
	// Make another refer to a different Token
	lease = AsLease(dt.load("leases", Hexaddr(net.ParseIP("192.168.124.10"))))
	lease.Token = "resn"
	renewTests := []ltc{
		{"mac", "res1", nil, nil, true, net.ParseIP("192.168.123.10")},
		{"mac", "res2", nil, nil, false, nil},
	}
	for _, obj := range renewTests {
		obj.test(t, dt)
	}
}

func TestDHCPCreateSubnet(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	// A subnet with 3 active addresses
	startObjs := []crudTest{
		{"Create valid Subnet", dt.create, &Subnet{p: dt, Name: "test", Subnet: "192.168.124.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.82"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, true},
	}
	for _, obj := range startObjs {
		obj.Test(t)
	}
	subnet := AsSubnet(dt.load("subnets", "test"))
	subnet.Picker = "none"
	// Even though there are no leases and no reservations, we should fail to create a lease.
	noneTests := []ltc{
		{"mac", "sub1", nil, nil, false, nil},                           // no via
		{"mac2", "sub1", nil, net.ParseIP("192.168.124.1"), false, nil}, // wrong strategy
		{"mac", "sub1", nil, net.ParseIP("192.168.124.1"), false, nil},
		{"mac", "sub1", net.ParseIP("192.168.124.80"), net.ParseIP("192.168.124.1"), false, nil},
	}
	for _, obj := range noneTests {
		obj.test(t, dt)
	}

	subnet.Picker = "next"
	subnet.nextLeasableIP = net.ParseIP("192.168.124.81")
	dt.save(subnet)
	nextTests := []ltc{
		{"mac", "sub1", net.ParseIP("192.168.124.81"), net.ParseIP("192.168.124.1"), true, net.ParseIP("192.168.124.81")},
		{"mac", "sub2", nil, net.ParseIP("192.168.124.1"), true, net.ParseIP("192.168.124.82")},
		{"mac", "sub3", nil, net.ParseIP("192.168.124.1"), true, net.ParseIP("192.168.124.80")},
	}
	for _, obj := range nextTests {
		obj.test(t, dt)
	}
}

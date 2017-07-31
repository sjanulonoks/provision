package backend

import (
	"net"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

func TestSubnetCrud(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	d, unlocker := dt.LockEnts("subnets", "leases", "reservations")
	defer unlocker()
	createTests := []crudTest{
		{"Create empty Subnet", dt.Create, &Subnet{p: dt}, false, nil},
		{"Create valid Subnet", dt.Create, &Subnet{p: dt, Name: "test", Subnet: "192.168.124.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, true, nil},
		{"Create duplicate Subnet", dt.Create, &Subnet{p: dt, Name: "test", Subnet: "192.168.124.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false, nil},
		{"Create invalid Subnet(bad Subnet)", dt.Create, &Subnet{p: dt, Name: "test2", Subnet: "127.0.0.0", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false, nil},
		{"Create invalid Subnet(overlapping Subnet)", dt.Create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.124.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false, nil},
		{"Create invalid Subnet(swapped Active range endpoints)", dt.Create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.254"), ActiveEnd: net.ParseIP("192.168.125.80"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false, nil},
		{"Create invalid Subnet(ActiveStart out of range)", dt.Create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.125.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false, nil},
		{"Create invalid Subnet(ActiveEnd out of range)", dt.Create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.80"), ActiveEnd: net.ParseIP("192.168.126.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false, nil},
		{"Create invalid Subnet(ActiveLeaseTime too short)", dt.Create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.80"), ActiveEnd: net.ParseIP("192.168.125.254"), ActiveLeaseTime: 59, ReservedLeaseTime: 7200, Strategy: "mac"}, false, nil},
		{"Create invalid Subnet(ReservedLeaseTime too short)", dt.Create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.80"), ActiveEnd: net.ParseIP("192.168.125.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7199, Strategy: "mac"}, false, nil},
		{"Create invalid Subnet(no Strategy)", dt.Create, &Subnet{p: dt, Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.80"), ActiveEnd: net.ParseIP("192.168.125.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: ""}, false, nil},
	}
	for _, test := range createTests {
		test.Test(t, d)
	}
	// List test.
	bes := d("subnets").Items()
	if bes != nil {
		if len(bes) != 1 {
			t.Errorf("List function should have returned: 1, but got %d\n", len(bes))
		}
	} else {
		t.Errorf("List function returned nil!!")
	}
}

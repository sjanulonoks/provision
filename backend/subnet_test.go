package backend

import (
	"net"
	"testing"

	"github.com/digitalrebar/provision/models"
)

func TestSubnetCrud(t *testing.T) {
	dt := mkDT(nil)
	d, unlocker := dt.LockEnts("subnets", "leases", "reservations")
	defer unlocker()
	createTests := []crudTest{
		{"Create empty Subnet", dt.Create, &models.Subnet{}, false},
		{"Create valid Subnet", dt.Create, &models.Subnet{Name: "test", Subnet: "192.168.124.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, true},
		{"Create duplicate Subnet", dt.Create, &models.Subnet{Name: "test", Subnet: "192.168.124.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(bad Subnet)", dt.Create, &models.Subnet{Name: "test2", Subnet: "127.0.0.0", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(overlapping Subnet)", dt.Create, &models.Subnet{Name: "test2", Subnet: "192.168.124.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.124.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(swapped Active range endpoints)", dt.Create, &models.Subnet{Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.254"), ActiveEnd: net.ParseIP("192.168.125.80"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(ActiveStart out of range)", dt.Create, &models.Subnet{Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.124.80"), ActiveEnd: net.ParseIP("192.168.125.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(ActiveEnd out of range)", dt.Create, &models.Subnet{Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.80"), ActiveEnd: net.ParseIP("192.168.126.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(ActiveLeaseTime too short)", dt.Create, &models.Subnet{Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.80"), ActiveEnd: net.ParseIP("192.168.125.254"), ActiveLeaseTime: 59, ReservedLeaseTime: 7200, Strategy: "mac"}, false},
		{"Create invalid Subnet(ReservedLeaseTime too short)", dt.Create, &models.Subnet{Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.80"), ActiveEnd: net.ParseIP("192.168.125.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7199, Strategy: "mac"}, false},
		{"Create invalid Subnet(no Strategy)", dt.Create, &models.Subnet{Name: "test2", Subnet: "192.168.125.0/24", ActiveStart: net.ParseIP("192.168.125.80"), ActiveEnd: net.ParseIP("192.168.125.254"), ActiveLeaseTime: 60, ReservedLeaseTime: 7200, Strategy: ""}, false},
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

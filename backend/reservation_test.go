package backend

import (
	"net"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

func TestReservationCrud(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	d, unlocker := dt.LockEnts("reservations", "subnets")
	defer unlocker()
	tests := []crudTest{
		{"Test Invalid Reservation Create", dt.Create, &Reservation{p: dt}, false},
		{"Test Incorrect IP Address Create", dt.Create, &Reservation{p: dt, Addr: net.ParseIP("127.0.0.1"), Token: "token", Strategy: "token"}, false},
		{"Test EmptyToken Create", dt.Create, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.10"), Token: "", Strategy: "token"}, false},
		{"Test EmptyStrategy Create", dt.Create, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.10"), Token: "token", Strategy: ""}, false},
		{"Test Valid Create", dt.Create, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.10"), Token: "token", Strategy: "token"}, true},
		{"Test Duplicate IP Create", dt.Create, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.10"), Token: "token", Strategy: "token"}, false},
		{"Test Duplicate Token Create", dt.Create, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.11"), Token: "token", Strategy: "token"}, false},
		{"Test Token Update", dt.Update, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.10"), Token: "token2", Strategy: "token"}, false},
		{"Test Strategy Update", dt.Update, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.10"), Token: "token", Strategy: "token2"}, false},
		{"Test Expire Update", dt.Update, &Reservation{p: dt, Addr: net.ParseIP("192.168.124.10"), Token: "token", Strategy: "token"}, true},
	}
	for _, test := range tests {
		test.Test(t, d)
	}
	// List test.
	bes := d("reservations").Items()
	if bes != nil {
		if len(bes) != 1 {
			t.Errorf("List function should have returned: 1, but got %d\n", len(bes))
		}
	} else {
		t.Errorf("List function returned nil!!")
	}
}

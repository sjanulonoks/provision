package backend

import (
	"net"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/pborman/uuid"
)

func TestMachineCrud(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	okUUID := uuid.NewRandom()
	tests := []crudTest{
		{"Create known-good Template", dt.create, &Template{p: dt, ID: "default"}, true},
		{"Create known-good Bootenv", dt.create, &BootEnv{p: dt, Name: "default", Templates: []TemplateInfo{{Name: "ipxe", Path: "{{ .Env.Name }}", ID: "default"}}}, true},
		{"Create known-unavailable Bootenv", dt.create, &BootEnv{p: dt, Name: "unavailable"}, true},
		{"Create empty machine", dt.create, &Machine{p: dt}, false},
		{"Create unnamed machine", dt.create, &Machine{p: dt, Uuid: okUUID}, false},
		{"Create named machine", dt.create, &Machine{p: dt, Uuid: okUUID, Name: "default.fqdn"}, true},
		{"Create new machine with same UUID", dt.create, &Machine{p: dt, Uuid: okUUID, Name: "other.fqdn"}, false},
		{"Create new machine with same name", dt.create, &Machine{p: dt, Uuid: uuid.NewRandom(), Name: "default.fqdn"}, false},
		{"Create new machine with invalid bootenv", dt.create, &Machine{p: dt, Uuid: uuid.NewRandom(), Name: "badenv.fqdn", BootEnv: "blargh"}, false},
		{"Create new machine with bad address", dt.create, &Machine{p: dt, Uuid: uuid.NewRandom(), Name: "badaddr.fqdn", BootEnv: "default", Address: net.ParseIP("127.0.0.1")}, false},
		{"Create another known-good bootenv", dt.create, &BootEnv{p: dt, Name: "new", Templates: []TemplateInfo{{Name: "ipxe", Path: "{{ .Env.Name }}", ID: "default"}}}, true},
		{"Update node with different bootenv", dt.update, &Machine{p: dt, Uuid: okUUID, Name: "default.fqdn", BootEnv: "new"}, true},
		{"Update node with different bootenv", dt.update, &Machine{p: dt, Uuid: okUUID, Name: "default.fqdn", BootEnv: "unavailable"}, false},
		{"Remove machine that does not exist", dt.remove, &Machine{p: dt, Uuid: uuid.NewRandom()}, false},
		{"Remove machine that does exist", dt.remove, &Machine{p: dt, Uuid: okUUID, BootEnv: "new"}, true},
	}
	for _, test := range tests {
		test.Test(t)
	}
}

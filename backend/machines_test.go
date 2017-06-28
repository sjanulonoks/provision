package backend

import (
	"encoding/json"
	"net"
	"testing"

	"github.com/VictorLowther/jsonpatch2"
	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/pborman/uuid"
)

type patchTest struct {
	desc  string
	pass  bool
	loc   int
	patch string
}

func (p *patchTest) test(t *testing.T, target store.KeySaver) {
	t.Logf("Testing %s", p.desc)
	buf, err := json.Marshal(target)
	if err != nil {
		t.Errorf("Unable to marshal %s: %v", target.Key(), err)
		return
	}
	patch, err := jsonpatch2.NewPatch([]byte(p.patch))
	if err != nil {
		t.Errorf("Patch %s is not valid: %v", p.patch, err)
		return
	}
	_, err, loc := patch.Apply(buf)
	if !p.pass && err != nil {
		if loc != p.loc {
			t.Errorf("Expected patch to fail at loc %d, not %d", p.loc, loc)
		} else {
			t.Logf("Failed at expected loc %d", loc)
		}
		t.Logf("Error: %v", err)
	} else if p.pass && err == nil {
		t.Logf("Patch succeeded")
	} else if err == nil {
		t.Errorf("Patch was expected to fail, but succeeded!")
	} else {
		t.Errorf("Patch failed at %d: %v", loc, err)
	}
}

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
		{"Create new machine with invalid bootenv", dt.create, &Machine{p: dt, Uuid: uuid.NewRandom(), Name: "badenv.fqdn", BootEnv: "blargh"}, true},
		{"Create new machine with bad address", dt.create, &Machine{p: dt, Uuid: uuid.NewRandom(), Name: "badaddr.fqdn", BootEnv: "default", Address: net.ParseIP("127.0.0.1")}, false},
		{"Create another known-good bootenv", dt.create, &BootEnv{p: dt, Name: "new", Templates: []TemplateInfo{{Name: "ipxe", Path: "{{ .Env.Name }}", ID: "default"}}}, true},
		{"Update node with different bootenv", dt.update, &Machine{p: dt, Uuid: okUUID, Name: "default.fqdn", BootEnv: "new"}, true},
		{"Update node with different bootenv", dt.update, &Machine{p: dt, Uuid: okUUID, Name: "default.fqdn", BootEnv: "unavailable"}, true},
		{"Remove machine that does not exist", dt.remove, &Machine{p: dt, Uuid: uuid.NewRandom()}, false},
		{"Remove machine that does exist", dt.remove, &Machine{p: dt, Uuid: okUUID, BootEnv: "new"}, true},
		{"Create named machine for patch", dt.create, &Machine{p: dt, Uuid: okUUID, Name: "default.fqdn"}, true},
	}
	for _, test := range tests {
		test.Test(t)
	}
	machine := dt.load("machines", okUUID.String())
	patchTests := []patchTest{
		{"force replace name pass", true, 0, `[{"op":"replace","path":"/Name","value":"default2"}]`},
		{"replace name pass", true, 0, `[
{"op":"test","path":"/Name","value":"default.fqdn"},
{"op":"replace","path":"/Name","value":"default2"}
]`},
		{"replace name fail", false, 0, `[
{"op":"test","path":"/Name","value":"default2"},
{"op":"replace","path":"/Name","value":"default2"}
]`},
	}
	for _, test := range patchTests {
		test.test(t, machine)
	}
	// List test.
	b := dt.NewMachine()
	bes := b.List()
	if bes != nil {
		if len(bes) != 2 {
			t.Errorf("List function should have returned: 1, but got %d\n", len(bes))
		}
	} else {
		t.Errorf("List function returned nil!!")
	}
}

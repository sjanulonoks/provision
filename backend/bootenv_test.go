package backend

import (
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/pborman/uuid"
)

func TestBootEnvCrud(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	tmpl := &Template{p: dt, ID: "ok", Contents: "{{ .Env.Name }}"}
	if ok, err := dt.create(tmpl); !ok {
		t.Errorf("Failed to create test OK template: %v", err)
		return
	}

	tests := []crudTest{
		{"Create Bootenv with nonexistent Name", dt.create, &BootEnv{p: dt}, false},
		{"Create Bootenv with no templates", dt.create, &BootEnv{p: dt, Name: "test 1"}, true},
		{"Create Bootenv with invalid BootParams tmpl", dt.create, &BootEnv{p: dt, Name: "test 2", BootParams: "{{ }"}, false},
		{"Create Bootenv with valid BootParams tmpl", dt.create, &BootEnv{p: dt, Name: "test 2", BootParams: "{{ .Env.Name }}"}, true},
		{"Create Bootenv with invalid TemplateInfo (missing Name)", dt.create, &BootEnv{p: dt, Name: "test 3", Templates: []TemplateInfo{{Path: "{{ .Env.Name }}", ID: "ok"}}}, false},
		{"Create Bootenv with invalid TemplateInfo (missing ID)", dt.create, &BootEnv{p: dt, Name: "test 3", Templates: []TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }}"}}}, false},
		{"Create Bootenv with invalid TemplateInfo (missing Path)", dt.create, &BootEnv{p: dt, Name: "test 3", Templates: []TemplateInfo{{Name: "test 3", ID: "ok"}}}, false},
		{"Create Bootenv with invalid TemplateInfo (invalid ID)", dt.create, &BootEnv{p: dt, Name: "test 3", Templates: []TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }}", ID: "okp"}}}, false},
		{"Create Bootenv with invalid TemplateInfo (invalid Path)", dt.create, &BootEnv{p: dt, Name: "test 3", Templates: []TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }", ID: "ok"}}}, false},
		{"Create Bootenv with valid TemplateInfo (not available}", dt.create, &BootEnv{p: dt, Name: "test 3", Templates: []TemplateInfo{{Name: "unavailable", Path: "{{ .Env.Name }}", ID: "ok"}}}, true},
		{"Create Bootenv with valid TemplateInfo (available)", dt.create, &BootEnv{p: dt, Name: "available", Templates: []TemplateInfo{{Name: "ipxe", Path: "{{ .Env.Name }}", ID: "ok"}}}, true},
	}

	for _, test := range tests {
		test.Test(t)
	}

	// List test.
	b := dt.NewBootEnv()
	bes := b.List()
	if bes != nil {
		if len(bes) != 5 {
			t.Errorf("List function should have returned: 5, but got %d\n", len(bes))
		}
	} else {
		t.Errorf("List function returned nil!!")
	}
	// We need a Machine that refers to one of our BootEnvs to
	// test proper delete restrictions
	machine := &Machine{p: dt, Name: "test 1", BootEnv: "available", Uuid: uuid.NewRandom()}
	if ok, err := dt.create(machine); !ok {
		t.Errorf("Failed to create test machine: %v", err)
		return
	}
	rmTests := []crudTest{
		{"Remove BootEnv that is not in use", dt.remove, &BootEnv{p: dt, Name: "test 1"}, true},
		{"Remove nonexistent BootEnv", dt.remove, &BootEnv{p: dt, Name: "test 1"}, false},
		{"Remove BootEnv that is in use", dt.remove, &BootEnv{p: dt, Name: "available"}, false},
	}
	for _, test := range rmTests {
		test.Test(t)
	}
}

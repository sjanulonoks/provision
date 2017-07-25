package backend

import (
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/pborman/uuid"
)

func TestBootEnvCrud(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	d, unlocker := dt.LockEnts("bootenvs", "templates", "tasks", "machines", "profiles")
	defer unlocker()
	tmpl := &Template{p: dt, ID: "ok", Contents: "{{ .Env.Name }}"}
	if ok, err := dt.Create(d, tmpl, nil); !ok {
		t.Errorf("Failed to create test OK template: %v", err)
		return
	}

	tests := []crudTest{
		{"Create Bootenv with nonexistent Name", dt.Create, &BootEnv{p: dt}, false, nil},
		{"Create Bootenv with no templates", dt.Create, &BootEnv{p: dt, Name: "test 1"}, true, nil},
		{"Create Bootenv with invalid BootParams tmpl", dt.Create, &BootEnv{p: dt, Name: "test 2", BootParams: "{{ }"}, false, nil},
		{"Create Bootenv with valid BootParams tmpl", dt.Create, &BootEnv{p: dt, Name: "test 2", BootParams: "{{ .Env.Name }}"}, true, nil},
		{"Create Bootenv with invalid TemplateInfo (missing Name)", dt.Create, &BootEnv{p: dt, Name: "test 3", Templates: []TemplateInfo{{Path: "{{ .Env.Name }}", ID: "ok"}}}, false, nil},
		{"Create Bootenv with invalid TemplateInfo (missing ID)", dt.Create, &BootEnv{p: dt, Name: "test 3", Templates: []TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }}"}}}, false, nil},
		{"Create Bootenv with invalid TemplateInfo (missing Path)", dt.Create, &BootEnv{p: dt, Name: "test 3", Templates: []TemplateInfo{{Name: "test 3", ID: "ok"}}}, false, nil},
		{"Create Bootenv with invalid TemplateInfo (invalid ID)", dt.Create, &BootEnv{p: dt, Name: "test 3", Templates: []TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }}", ID: "okp"}}}, false, nil},
		{"Create Bootenv with invalid TemplateInfo (invalid Path)", dt.Create, &BootEnv{p: dt, Name: "test 3", Templates: []TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }", ID: "ok"}}}, false, nil},
		{"Create Bootenv with valid TemplateInfo (not available}", dt.Create, &BootEnv{p: dt, Name: "test 3", Templates: []TemplateInfo{{Name: "unavailable", Path: "{{ .Env.Name }}", ID: "ok"}}}, true, nil},
		{"Create Bootenv with valid TemplateInfo (available)", dt.Create, &BootEnv{p: dt, Name: "available", Templates: []TemplateInfo{{Name: "ipxe", Path: "{{ .Env.Name }}", ID: "ok"}}}, true, nil},
	}

	for _, test := range tests {
		test.Test(t, d)
	}

	// List test.
	bes := d("bootenvs").Items()
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
	if ok, err := dt.Create(d, machine, nil); !ok {
		t.Errorf("Failed to create test machine: %v", err)
		return
	}
	rmTests := []crudTest{
		{"Remove BootEnv that is not in use", dt.Remove, &BootEnv{p: dt, Name: "test 1"}, true, nil},
		{"Remove nonexistent BootEnv", dt.Remove, &BootEnv{p: dt, Name: "test 1"}, false, nil},
		{"Remove BootEnv that is in use", dt.Remove, &BootEnv{p: dt, Name: "available"}, false, nil},
	}
	for _, test := range rmTests {
		test.Test(t, d)
	}
}

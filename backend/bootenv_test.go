package backend

import (
	"testing"

	"github.com/digitalrebar/provision/models"
	"github.com/pborman/uuid"
)

func TestBootEnvCrud(t *testing.T) {
	dt := mkDT(nil)
	d, unlocker := dt.LockEnts("bootenvs", "templates", "tasks", "machines", "profiles")
	defer unlocker()
	tmpl := &models.Template{ID: "ok", Contents: "{{ .Env.Name }}"}
	if ok, err := dt.Create(d, tmpl); !ok {
		t.Errorf("Failed to create test OK template: %#v: %#v", tmpl, err)
		return
	}

	tests := []crudTest{
		{"Create Bootenv with nonexistent Name", dt.Create, &models.BootEnv{}, false},
		{"Create Bootenv with no templates", dt.Create, &models.BootEnv{Name: "test 1"}, true},
		{"Create Bootenv with invalid BootParams tmpl", dt.Create, &models.BootEnv{Name: "test 2", BootParams: "{{ }"}, false},
		{"Create Bootenv with valid BootParams tmpl", dt.Create, &models.BootEnv{Name: "test 2", BootParams: "{{ .Env.Name }}"}, true},
		{"Create Bootenv with invalid models.TemplateInfo (missing Name)", dt.Create, &models.BootEnv{Name: "test 3", Templates: []models.TemplateInfo{{Path: "{{ .Env.Name }}", ID: "ok"}}}, false},
		{"Create Bootenv with invalid models.TemplateInfo (missing ID)", dt.Create, &models.BootEnv{Name: "test 3", Templates: []models.TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }}"}}}, false},
		{"Create Bootenv with invalid models.TemplateInfo (missing Path)", dt.Create, &models.BootEnv{Name: "test 3", Templates: []models.TemplateInfo{{Name: "test 3", ID: "ok"}}}, false},
		{"Create Bootenv with invalid models.TemplateInfo (invalid ID)", dt.Create, &models.BootEnv{Name: "test 3", Templates: []models.TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }}", ID: "okp"}}}, false},
		{"Create Bootenv with invalid models.TemplateInfo (invalid Path)", dt.Create, &models.BootEnv{Name: "test 3", Templates: []models.TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }", ID: "ok"}}}, false},
		{"Create Bootenv with valid models.TemplateInfo (not available}", dt.Create, &models.BootEnv{Name: "test 3", Templates: []models.TemplateInfo{{Name: "unavailable", Path: "{{ .Env.Name }}", ID: "ok"}}}, true},
		{"Create Bootenv with valid models.TemplateInfo (available)", dt.Create, &models.BootEnv{Name: "available", Templates: []models.TemplateInfo{{Name: "ipxe", Path: "{{ .Env.Name }}", ID: "ok"}}}, true},
	}

	for _, test := range tests {
		test.Test(t, d)
	}

	// List test.
	bes := d("bootenvs").Items()
	if bes != nil {
		if len(bes) != 6 {
			t.Errorf("List function should have returned: 6, but got %d\n", len(bes))
		}
	} else {
		t.Errorf("List function returned nil!!")
	}
	// We need a Machine that refers to one of our BootEnvs to
	// test proper delete restrictions
	machine := &models.Machine{Name: "test 1", BootEnv: "available", Uuid: uuid.NewRandom()}
	if ok, err := dt.Create(d, machine); !ok {
		t.Errorf("Failed to create test machine: %v", err)
		return
	}
	rmTests := []crudTest{
		{"Remove BootEnv that is not in use", dt.Remove, &models.BootEnv{Name: "test 1"}, true},
		{"Remove nonexistent BootEnv", dt.Remove, &models.BootEnv{Name: "test 1"}, false},
		{"Remove BootEnv that is in use", dt.Remove, &models.BootEnv{Name: "available"}, false},
	}
	for _, test := range rmTests {
		test.Test(t, d)
	}
}

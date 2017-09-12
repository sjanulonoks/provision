package backend

import (
	"testing"

	"github.com/digitalrebar/provision/models"
	"github.com/pborman/uuid"
)

func TestStageCrud(t *testing.T) {
	dt := mkDT(nil)
	d, unlocker := dt.LockEnts("stages", "bootenvs", "templates", "tasks", "machines", "profiles")
	defer unlocker()
	tmpl := &models.Template{ID: "ok", Contents: "{{ .Env.Name }}"}
	if ok, err := dt.Create(d, tmpl); !ok {
		t.Errorf("Failed to create test OK template: %#v: %#v", tmpl, err)
		return
	}

	tests := []crudTest{
		{"Create Stage with nonexistent Name", dt.Create, &models.Stage{}, false},
		{"Create Stage with no BootEnv", dt.Create, &models.Stage{Name: "nobootenv"}, true},
		{"Create Stage with nonexistent BootEnv", dt.Create, &models.Stage{Name: "missingbootenv", BootEnv: "missingbootenv"}, false},
		{"Create Stage with missing Task", dt.Create, &models.Stage{Name: "missingtask", BootEnv: "local", Tasks: []string{"jj"}}, false},
		{"Create Stage with missing profile", dt.Create, &models.Stage{Name: "missingprofile", BootEnv: "local", Profiles: []string{"jj"}}, false},
		{"Create Stage with invalid models.TemplateInfo (missing Name)", dt.Create, &models.Stage{Name: "test 3", BootEnv: "local", Templates: []models.TemplateInfo{{Path: "{{ .Env.Name }}", ID: "ok"}}}, false},
		{"Create Stage with invalid models.TemplateInfo (missing ID)", dt.Create, &models.Stage{Name: "test 3", BootEnv: "local", Templates: []models.TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }}"}}}, false},
		{"Create Stage with invalid models.TemplateInfo (missing Path)", dt.Create, &models.Stage{Name: "test 3", BootEnv: "local", Templates: []models.TemplateInfo{{Name: "test 3", ID: "ok"}}}, false},
		{"Create Stage with invalid models.TemplateInfo (invalid ID)", dt.Create, &models.Stage{Name: "test 3", BootEnv: "local", Templates: []models.TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }}", ID: "okp"}}}, false},
		{"Create Stage with invalid models.TemplateInfo (invalid Path)", dt.Create, &models.Stage{Name: "test 3", BootEnv: "local", Templates: []models.TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }", ID: "ok"}}}, false},
		{"Create Stage with valid models.TemplateInfo (not available}", dt.Create, &models.Stage{Name: "test 1", BootEnv: "local", Templates: []models.TemplateInfo{{Name: "unavailable", Path: "{{ .Env.Name }}", ID: "ok"}}}, true},
		{"Create Stage with valid models.TemplateInfo (available)", dt.Create, &models.Stage{Name: "available", BootEnv: "local", Templates: []models.TemplateInfo{{Name: "ipxe", Path: "{{ .Env.Name }}", ID: "ok"}}}, true},
	}

	for _, test := range tests {
		test.Test(t, d)
	}

	// List test.
	bes := d("stages").Items()
	if bes != nil {
		if len(bes) != 3 {
			t.Errorf("List function should have returned: 3, but got %d\n", len(bes))
		}
	} else {
		t.Errorf("List function returned nil!!")
	}

	// We need a Machine that refers to one of our Stage to
	// test proper delete restrictions
	machine := &models.Machine{Name: "test 1", Stage: "available", Uuid: uuid.NewRandom()}
	if ok, err := dt.Create(d, machine); !ok {
		t.Errorf("Failed to create test machine: %v", err)
		return
	}
	rmTests := []crudTest{
		{"Remove Stage that is not in use", dt.Remove, &models.Stage{Name: "test 1"}, true},
		{"Remove nonexistent Stage", dt.Remove, &models.Stage{Name: "test 1"}, false},
		{"Remove Stage that is in use", dt.Remove, &models.Stage{Name: "available"}, false},
	}
	for _, test := range rmTests {
		test.Test(t, d)
	}
}

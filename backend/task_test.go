package backend

import (
	"testing"

	"github.com/digitalrebar/provision/models"
)

func TestTaskCrud(t *testing.T) {
	dt := mkDT(nil)
	d, unlocker := dt.LockEnts("templates", "tasks", "bootenvs")
	defer unlocker()
	tmpl := &models.Template{ID: "ok", Contents: "{{ .Env.Name }}"}
	if ok, err := dt.Create(d, tmpl); !ok {
		t.Errorf("Failed to create test OK template: %v", err)
		return
	}
	tests := []crudTest{
		{"Create Task with nonexistent Name", dt.Create, &models.Task{}, false},
		{"Create Task with no templates", dt.Create, &models.Task{Name: "test 1"}, true},
		{"Create Task with bad name /", dt.Create, &models.Task{Name: "test/1"}, false},
		{"Create Task with bad name \\", dt.Create, &models.Task{Name: "test\\1"}, false},
		{"Create Task with invalid models.TemplateInfo (missing Name)", dt.Create, &models.Task{Name: "test 3", Templates: []models.TemplateInfo{{Path: "{{ .Env.Name }}", ID: "ok"}}}, false},
		{"Create Task with invalid models.TemplateInfo (missing ID)", dt.Create, &models.Task{Name: "test 3", Templates: []models.TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }}"}}}, false},
		{"Create Task with invalid models.TemplateInfo (invalid ID)", dt.Create, &models.Task{Name: "test 3", Templates: []models.TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }}", ID: "okp"}}}, false},
		{"Create Task with invalid models.TemplateInfo (invalid Path)", dt.Create, &models.Task{Name: "test 3", Templates: []models.TemplateInfo{{Name: "test 3", Path: "{{ .Env.Name }", ID: "ok"}}}, false},
		{"Create Task with valid models.TemplateInfo (not available}", dt.Create, &models.Task{Name: "test 3", Templates: []models.TemplateInfo{{Name: "unavailable", Path: "{{ .Env.Name }}", ID: "ok"}}}, true},
		{"Create Task with valid models.TemplateInfo (available)", dt.Create, &models.Task{Name: "available", Templates: []models.TemplateInfo{{Name: "ipxe", Path: "{{ .Env.Name }}", ID: "ok"}}}, true},
	}

	for _, test := range tests {
		test.Test(t, d)
	}

	// List test.
	bes := d("tasks").Items()
	if bes != nil {
		if len(bes) != 3 {
			t.Errorf("List function should have returned: 5, but got %d\n", len(bes))
		}
	} else {
		t.Errorf("List function returned nil!!")
	}
	/*
		// We need a Machine that refers to one of our Tasks to
		// test proper delete restrictions
		machine := &models.Machine{ Name: "test 1", Task: "available", Uuid: uuid.NewRandom()}
		if ok, err := dt.Create(d, machine); !ok {
			t.Errorf("Failed to create test machine: %v", err)
			return
		}
		rmTests := []crudTest{
			{"Remove Task that is not in use", dt.Remove, &models.Task{ Name: "test 1"}, true},
			{"Remove nonexistent Task", dt.Remove, &models.Task{ Name: "test 1"}, false},
			{"Remove Task that is in use", dt.Remove, &models.Task{ Name: "available"}, false},
		}
		for _, test := range rmTests {
			test.Test(t)
		}
	*/
}

package backend

import (
	"testing"

	"github.com/digitalrebar/provision/models"
)

func TestTemplateCrud(t *testing.T) {
	dt := mkDT(nil)
	d, unlocker := dt.LockEnts("templates", "bootenvs", "tasks", "machines")
	defer unlocker()
	tests := []crudTest{
		{"Create Template with No ID", dt.Create, &models.Template{}, false},
		{"Create Valid Empty Template", dt.Create, &models.Template{ID: "test1"}, true},
		{"Create Valid Nonempty Template", dt.Create, &models.Template{ID: "test2", Contents: "{{ .Foo }}"}, true},
		{"Create Duplicate Template", dt.Create, &models.Template{ID: "test1"}, false},
		{"Create Invalid Template", dt.Create, &models.Template{ID: "test4", Contents: "{{ .Bar }"}, false},
		{"Create Template that refers to another template", dt.Create, &models.Template{ID: "test3", Contents: `{{template "test2"}}`}, true},
		{"Update Valid Contents", dt.Update, &models.Template{ID: "test1", Contents: "{{ .Bar }}"}, true},
		{"Update Invalid Contents", dt.Update, &models.Template{ID: "test1", Contents: "{{}"}, false},
		{"Update ID", dt.Update, &models.Template{ID: "test5"}, false},
		{"Update with blank ID", dt.Update, &models.Template{}, false},
	}
	for _, test := range tests {
		test.Test(t, d)
	}
	b := &models.BootEnv{Name: "scratch"}
	b.Templates = []models.TemplateInfo{{Name: "ipxe", Path: "default.ipxe", ID: "test1"}}
	saved, err := dt.Create(d, b)
	if !saved {
		t.Errorf("Error saving scratch bootenv: %v", err)
	} else {
		t.Logf("Created scratch bootenv")
	}

	tests = []crudTest{
		{"Remove Unused Template", dt.Remove, &models.Template{ID: "test2"}, true},
		{"Remove Used Template", dt.Remove, &models.Template{ID: "test1"}, false},
	}
	for _, test := range tests {
		test.Test(t, d)
	}

	// List test.
	bes := d("templates").Items()
	if bes != nil {
		if len(bes) != 2 {
			t.Errorf("List function should have returned: 2, but got %d\n", len(bes))
		}
	} else {
		t.Errorf("List function returned nil!!")
	}
}

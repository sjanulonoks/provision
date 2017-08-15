package backend

import "testing"

func TestTemplateCrud(t *testing.T) {
	dt := mkDT(nil)
	d, unlocker := dt.LockEnts("templates", "bootenvs", "tasks", "machines")
	defer unlocker()
	tests := []crudTest{
		{"Create Template with No ID", dt.Create, &Template{p: dt}, false, nil},
		{"Create Valid Empty Template", dt.Create, &Template{p: dt, ID: "test1"}, true, nil},
		{"Create Valid Nonempty Template", dt.Create, &Template{p: dt, ID: "test2", Contents: "{{ .Foo }}"}, true, nil},
		{"Create Duplicate Template", dt.Create, &Template{p: dt, ID: "test1"}, false, nil},
		{"Create Invalid Template", dt.Create, &Template{p: dt, ID: "test4", Contents: "{{ .Bar }"}, false, nil},
		{"Create Template that refers to another template", dt.Create, &Template{p: dt, ID: "test3", Contents: `{{template "test2"}}`}, true, nil},
		{"Update Valid Contents", dt.Update, &Template{p: dt, ID: "test1", Contents: "{{ .Bar }}"}, true, nil},
		{"Update Invalid Contents", dt.Update, &Template{p: dt, ID: "test1", Contents: "{{}"}, false, nil},
		{"Update ID", dt.Update, &Template{p: dt, ID: "test5"}, false, nil},
		{"Update with blank ID", dt.Update, &Template{p: dt}, false, nil},
	}
	for _, test := range tests {
		test.Test(t, d)
	}
	b := dt.NewBootEnv()
	b.Name = "scratch"
	b.Templates = []TemplateInfo{{Name: "ipxe", Path: "default.ipxe", ID: "test1"}}
	saved, err := dt.Create(d, b, nil)
	if !saved {
		t.Errorf("Error saving scratch bootenv: %v", err)
	} else {
		t.Logf("Created scratch bootenv")
	}

	tests = []crudTest{
		{"Remove Unused Template", dt.Remove, &Template{p: dt, ID: "test2"}, true, nil},
		{"Remove Used Template", dt.Remove, &Template{p: dt, ID: "test1"}, false, nil},
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

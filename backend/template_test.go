package backend

import (
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

func TestTemplateCrud(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	tests := []crudTest{
		{"Create Template with No ID", dt.Create, &Template{p: dt}, false},
		{"Create Valid Empty Template", dt.Create, &Template{p: dt, ID: "test1"}, true},
		{"Create Valid Nonempty Template", dt.Create, &Template{p: dt, ID: "test2", Contents: "{{ .Foo }}"}, true},
		{"Create Duplicate Template", dt.Create, &Template{p: dt, ID: "test1"}, false},
		{"Create Invalid Template", dt.Create, &Template{p: dt, ID: "test4", Contents: "{{ .Bar }"}, false},
		{"Create Template that refers to another template", dt.Create, &Template{p: dt, ID: "test3", Contents: `{{template "test2"}}`}, true},
		{"Update Valid Contents", dt.update, &Template{p: dt, ID: "test1", Contents: "{{ .Bar }}"}, true},
		{"Update Invalid Contents", dt.update, &Template{p: dt, ID: "test1", Contents: "{{}"}, false},
		{"Update ID", dt.update, &Template{p: dt, ID: "test5"}, false},
		{"Update with blank ID", dt.update, &Template{p: dt}, false},
	}
	for _, test := range tests {
		test.Test(t)
	}
	b := dt.NewBootEnv()
	b.Name = "scratch"
	b.Templates = []TemplateInfo{{Name: "ipxe", Path: "default.ipxe", ID: "test1"}}
	saved, err := dt.Create(b)
	if !saved {
		t.Errorf("Error saving scratch bootenv: %v", err)
	} else {
		t.Logf("Created scratch bootenv")
	}

	tests = []crudTest{
		{"Remove Unused Template", dt.Remove, &Template{p: dt, ID: "test2"}, true},
		{"Remove Used Template", dt.Remove, &Template{p: dt, ID: "test1"}, false},
	}
	for _, test := range tests {
		test.Test(t)
	}

	// List test.
	tmp := dt.NewTemplate()
	bes := tmp.List()
	if bes != nil {
		if len(bes) != 2 {
			t.Errorf("List function should have returned: 2, but got %d\n", len(bes))
		}
	} else {
		t.Errorf("List function returned nil!!")
	}
}

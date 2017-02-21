package backend

import (
	"fmt"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

type tT struct {
	name string
	op   func(store.KeySaver) (bool, error)
	t    *Template
	pass bool
}

func opTemplate(t *testing.T, dt *DataTracker, tmplT tT) {
	tmplT.t.p = dt
	passed, err := tmplT.op(tmplT.t)
	msg := fmt.Sprintf("%s: wanted to pass: %v, passed: %v", tmplT.name, tmplT.pass, passed)
	if passed == tmplT.pass {
		t.Log(msg)
		t.Logf("   err: %v", err)
	} else {
		t.Error(msg)
		t.Errorf("   err: %v", err)
	}
}

func TestTemplateOps(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	tests := []tT{
		{"Create Template with No ID", dt.create, &Template{}, false},
		{"Create Valid Empty Template", dt.create, &Template{ID: "test1"}, true},
		{"Create Valid Nonempty Template", dt.create, &Template{ID: "test2", Contents: "{{ .Foo }}"}, true},
		{"Create Duplicate Template", dt.create, &Template{ID: "test1"}, false},
		{"Create Invalid Template", dt.create, &Template{ID: "test4", Contents: "{{ .Bar }"}, false},

		{"Update Valid Contents", dt.update, &Template{ID: "test1", Contents: "{{ .Bar }}"}, true},
		{"Update Invalid Contents", dt.update, &Template{ID: "test1", Contents: "{{}"}, false},
		{"Update ID", dt.update, &Template{ID: "test5"}, false},
		{"Update with blank ID", dt.update, &Template{}, false},
	}
	for _, tmplT := range tests {
		opTemplate(t, dt, tmplT)
	}
	b := dt.NewBootEnv()
	b.Name = "scratch"
	b.Templates = []TemplateInfo{{Name: "ipxe", Path: "default.ipxe", ID: "test1"}}
	saved, err := dt.create(b)
	if !saved {
		t.Errorf("Error saving scratch bootenv: %v", err)
	} else {
		t.Logf("Created scratch bootenv")
	}

	tests = []tT{
		{"Remove Unused Template", dt.remove, &Template{ID: "test2"}, true},
		{"Remove Used Template", dt.remove, &Template{ID: "test1"}, false},
	}
	for _, tmplT := range tests {
		opTemplate(t, dt, tmplT)
	}

}

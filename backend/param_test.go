package backend

import (
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

func TestParamsCrud(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	d, unlocker := dt.LockEnts("profiles", "params")
	defer unlocker()
	tests := []crudTest{
		{"Create empty profile", dt.Create, &Param{p: dt}, false, nil},
		{"Create new profile with name", dt.Create, &Param{p: dt, Name: "Test Param"}, false, nil},
		{"Create new profile with name and schema", dt.Create, &Param{p: dt,
			Name:   "Test Param",
			Schema: map[string]interface{}{},
		}, true, nil},
		{"Create new profile with name and schema", dt.Create, &Param{p: dt,
			Name: "Test Param 2",
			Schema: map[string]interface{}{
				"type": "boolean",
			},
		}, true, nil},
		{"Create Duplicate Param", dt.Create, &Param{p: dt, Name: "Test Param"}, false, nil},
		{"Delete Param", dt.Remove, &Param{p: dt, Name: "Test Param"}, true, nil},
		{"Delete Nonexistent Param", dt.Remove, &Param{p: dt, Name: "Test Param"}, false, nil},
	}
	for _, test := range tests {
		test.Test(t, d)
	}
	// List test.
	bes := d("profiles").Items()
	if bes != nil {
		if len(bes) != 1 {
			t.Errorf("List function should have returned: 1, but got %d\n", len(bes))
		}
	} else {
		t.Errorf("List function returned nil!!")
	}
}

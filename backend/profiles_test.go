package backend

import (
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

func TestProfilesCrud(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	d, unlocker := dt.LockEnts("profiles", "params", "machines")
	defer unlocker()
	tests := []crudTest{
		{"Create empty profile", dt.Create, &Profile{p: dt}, false, nil},
		{"Create new profile with name", dt.Create, &Profile{p: dt, Name: "Test Profile"}, true, nil},
		{"Create Duplicate Profile", dt.Create, &Profile{p: dt, Name: "Test Profile"}, false, nil},
		{"Delete Profile", dt.Remove, &Profile{p: dt, Name: "Test Profile"}, true, nil},
		{"Delete Nonexistent Profile", dt.Remove, &Profile{p: dt, Name: "Test Profile"}, false, nil},
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

func TestProfilesValidation(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	d, unlocker := dt.LockEnts("profiles", "params")
	defer unlocker()
	tests := []crudTest{
		{
			"Create new Parameter",
			dt.Create,
			&Param{
				p:    dt,
				Name: "Bool",
				Schema: map[string]interface{}{
					"type": "boolean",
				},
			},
			true,
			nil,
		},
		{
			"Create Passing Profile",
			dt.Create,
			&Profile{
				p:    dt,
				Name: "Bool Profile Pass",
				Params: map[string]interface{}{
					"Bool": true,
				},
			},
			true,
			nil,
		},
		{
			"Create Failing Profile",
			dt.Create,
			&Profile{
				p:    dt,
				Name: "Bool Profile Fail",
				Params: map[string]interface{}{
					"Bool": "true",
				},
			},
			false,
			nil,
		},
	}
	for _, test := range tests {
		test.Test(t, d)
	}
}

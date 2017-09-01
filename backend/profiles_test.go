package backend

import (
	"testing"

	"github.com/digitalrebar/provision/models"
)

func TestProfilesCrud(t *testing.T) {
	dt := mkDT(nil)
	d, unlocker := dt.LockEnts("stages", "profiles", "params", "machines")
	defer unlocker()
	tests := []crudTest{
		{"Create empty profile", dt.Create, &models.Profile{}, false},
		{"Create new profile with name", dt.Create, &models.Profile{Name: "Test Profile"}, true},
		{"Create Duplicate Profile", dt.Create, &models.Profile{Name: "Test Profile"}, false},
		{"Delete Profile", dt.Remove, &models.Profile{Name: "Test Profile"}, true},
		{"Delete Nonexistent Profile", dt.Remove, &models.Profile{Name: "Test Profile"}, false},
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
	dt := mkDT(nil)
	d, unlocker := dt.LockEnts("profiles", "params")
	defer unlocker()
	tests := []crudTest{
		{
			"Create new Parameter",
			dt.Create,
			&models.Param{
				Name: "Bool",
				Schema: map[string]interface{}{
					"type": "boolean",
				},
			},
			true,
		},
		{
			"Create Passing Profile",
			dt.Create,
			&models.Profile{
				Name: "Bool Profile Pass",
				Params: map[string]interface{}{
					"Bool": true,
				},
			},
			true,
		},
		{
			"Create Failing Profile",
			dt.Create,
			&models.Profile{
				Name: "Bool Profile Fail",
				Params: map[string]interface{}{
					"Bool": "true",
				},
			},
			true,
		},
	}
	for _, test := range tests {
		test.Test(t, d)
	}
}

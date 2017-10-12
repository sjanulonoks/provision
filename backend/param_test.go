package backend

import (
	"testing"

	"github.com/digitalrebar/provision/models"
)

func TestParamsCrud(t *testing.T) {
	dt := mkDT(nil)
	d, unlocker := dt.LockEnts("profiles", "params")
	defer unlocker()
	tests := []crudTest{
		{"Create empty profile", dt.Create, &models.Param{}, false},
		{"Create new profile with name", dt.Create, &models.Param{Name: "Test Param 0"}, true},
		{"Create new profile with name and schema", dt.Create, &models.Param{
			Name:   "Test Param",
			Schema: map[string]interface{}{},
		}, true},
		{"Create new profile with name and schema", dt.Create, &models.Param{
			Name: "Test Param 2",
			Schema: map[string]interface{}{
				"type": "boolean",
			},
		}, true},
		{"Create Duplicate Param", dt.Create, &models.Param{Name: "Test Param"}, false},
		{"Delete Param", dt.Remove, &models.Param{Name: "Test Param"}, true},
		{"Delete Nonexistent Param", dt.Remove, &models.Param{Name: "Test Param"}, false},
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

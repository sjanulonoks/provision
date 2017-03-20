package backend

import (
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

func TestParamCrud(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	tests := []crudTest{
		{"Create empty Parameter", dt.create, &Param{}, false},
		{"Create test parameter with no value", dt.create, &Param{Name: "test"}, true},
		{"Create duplicate parameter", dt.create, &Param{Name: "test"}, false},
		{"Update test with a value", dt.update, &Param{Name: "test", Value: false}, true},
		{"Try to update a non-existent value", dt.update, &Param{Name: "test2", Value: true}, false},
		{"Delete a parameter", dt.remove, &Param{Name: "test"}, true},
		{"Delete a nonexistent parameter", dt.remove, &Param{Name: "test"}, false},
	}
	for _, test := range tests {
		test.Test(t)
	}
}

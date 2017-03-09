package backend

import (
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

func TestGetInterfaces(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)

	ifs, err := dt.GetInterfaces()
	if err != nil {
		t.Errorf("GetInterfaces should not err: %v\n", err)
	}
	if len(ifs) == 0 {
		t.Errorf("GetInterfaces should return something: %v\n", ifs)
	}
}

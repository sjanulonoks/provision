package backend

import (
	"testing"
)

func TestRandString(t *testing.T) {
	r := randString(16)
	if len(r) != 16 {
		t.Errorf("Random string should be 16 bytes long: %s\n", r)
	}
}

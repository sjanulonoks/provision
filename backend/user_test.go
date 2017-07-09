package backend

import (
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

func TestUserCrud(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	tests := []crudTest{
		{"Create empty user", dt.Create, &User{p: dt}, false},
		{"Create new user with name", dt.Create, &User{p: dt, Name: "Test User"}, true},
		{"Create Duplicate User", dt.Create, &User{p: dt, Name: "Test User"}, false},
		{"Delete User", dt.Remove, &User{p: dt, Name: "Test User"}, true},
		{"Delete Nonexistent User", dt.Remove, &User{p: dt, Name: "Test User"}, false},
	}
	for _, test := range tests {
		test.Test(t)
	}
	// List test.
	b := dt.NewUser()
	bes := b.List()
	if bes != nil {
		if len(bes) != 1 {
			t.Errorf("List function should have returned: 1, but got %d\n", len(bes))
		}
	} else {
		t.Errorf("List function returned nil!!")
	}
}

func TestUserPassword(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	u := dt.NewUser()
	u.Name = "test user"
	saved, err := dt.Create(u)
	if !saved {
		t.Errorf("Unable to create test user: %v", err)
	} else {
		t.Logf("Created test user")
	}
	// should fail because we have no password
	if u.CheckPassword("password") {
		t.Errorf("Checking password should have failed!")
	} else {
		t.Logf("Checking password failed, as expected.")
	}
	if err := u.ChangePassword("password"); err != nil {
		t.Errorf("Changing password failed: %v", err)
	} else {
		t.Logf("Changing password passed.")
	}
	// reload the user, then check the password again.
	newU := dt.NewUser()
	buf, found := dt.FetchOne(newU, "test user")
	newU = AsUser(buf)
	if !found || newU.Name != "test user" {
		t.Errorf("Unable to fetch user from datatracker")
	} else {
		t.Logf("Fetched new user from datatracker cache")
	}
	if !newU.CheckPassword("password") {
		t.Errorf("Checking password should have succeeded.")
	} else {
		t.Logf("CHecking password passed, as expected.")
	}
	// Make sure sanitizing the user works as expected
	newU.Sanitize()
	if len(newU.PasswordHash) != 0 {
		t.Errorf("Sanitize did not strip out the password hash")
	} else {
		t.Logf("Sanitize stripped out the password hash")
	}
}

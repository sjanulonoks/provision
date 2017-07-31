package backend

import (
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

func TestUserCrud(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	d, unlocker := dt.LockEnts("users")
	defer unlocker()
	tests := []crudTest{
		{"Create empty user", dt.Create, &User{p: dt}, false, nil},
		{"Create new user with name", dt.Create, &User{p: dt, Name: "Test User"}, true, nil},
		{"Create Duplicate User", dt.Create, &User{p: dt, Name: "Test User"}, false, nil},
		{"Delete User", dt.Remove, &User{p: dt, Name: "Test User"}, true, nil},
		{"Delete Nonexistent User", dt.Remove, &User{p: dt, Name: "Test User"}, false, nil},
	}
	for _, test := range tests {
		test.Test(t, d)
	}
	// List test.
	bes := d("users").Items()
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
	d, unlocker := dt.LockEnts("users")
	defer unlocker()
	u := dt.NewUser()
	u.Name = "test user"
	saved, err := dt.Create(d, u, nil)
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
	if err := u.ChangePassword(d, "password"); err != nil {
		t.Errorf("Changing password failed: %v", err)
	} else {
		t.Logf("Changing password passed.")
	}
	// reload the user, then check the password again.
	buf := d("users").Find("test user")
	if buf == nil {
		t.Errorf("Unable to fetch user from datatracker")
	} else {
		t.Logf("Fetched new user from datatracker cache")
	}
	newU := AsUser(buf)
	if !newU.CheckPassword("password") {
		t.Errorf("Checking password should have succeeded.")
	} else {
		t.Logf("CHecking password passed, as expected.")
	}
	// Make sure sanitizing the user works as expected
	newU = AsUser(newU.Sanitize())
	if len(newU.PasswordHash) != 0 {
		t.Errorf("Sanitize did not strip out the password hash")
	} else {
		t.Logf("Sanitize stripped out the password hash")
	}
}

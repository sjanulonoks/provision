package backend

import (
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

func TestPreferences(t *testing.T) {
	bs := store.NewSimpleMemoryStore()
	dt := mkDT(bs)
	if be, err := dt.Pref("defaultBootEnv"); err != nil {
		t.Errorf("Expected to get a defaultBootEnv preference, got nothing")
	} else {
		t.Logf("Got defaultBootEnv %s", be)
	}
	if be, err := dt.Pref("unknownBootEnv"); err == nil {
		t.Errorf("Did not expect to get unknownBootEnv %s", be)
	} else {
		t.Logf("Got expected error looking for unknownBootEnv: %v", err)
	}
	prefs := map[string]string{}
	prefs["unknownBootEnv"] = "ignore"
	if err := dt.SetPrefs(prefs); err != nil {
		t.Errorf("Unexpected error setting prefs: %v", err)
	} else {
		t.Logf("Set unknownBootEnv to ignore")
	}
	prefs["defaultBootEnv"] = "default"
	if err := dt.SetPrefs(prefs); err != nil {
		t.Logf("Expected error setting prefs: %v", err)
	} else {
		t.Errorf("Should have failed setting defaultBootEnv to default")
	}
	prefs["defaultBootEnv"] = "ignore"
	if err := dt.SetPrefs(prefs); err != nil {
		t.Errorf("Unexpected error setting prefs: %v", err)
	} else {
		t.Logf("Set defaultBootEnv to ignore")
	}
	prefs["foo"] = "bar"
	if err := dt.SetPrefs(prefs); err != nil {
		t.Logf("Expected error setting prefs: %v", err)
	} else {
		t.Errorf("Should have failed setting foo to bar")
	}
	dt.Prefs()

}

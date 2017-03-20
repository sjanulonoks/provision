package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

var (
	backingStore store.SimpleStore
	tmpDir       string
)

type crudTest struct {
	name string
	op   func(store.KeySaver) (bool, error)
	t    store.KeySaver
	pass bool
}

func (test *crudTest) Test(t *testing.T) {
	passed, err := test.op(test.t)
	msg := fmt.Sprintf("%s: wanted to pass: %v, passed: %v", test.name, test.pass, passed)
	if passed == test.pass {
		t.Log(msg)
		t.Logf("   err: %v", err)
	} else {
		t.Error(msg)
		t.Errorf("   err: %v", err)
	}
}

func loadExample(dt *DataTracker, kind, p string) (bool, error) {
	buf, err := os.Open(p)
	if err != nil {
		return false, err
	}
	defer buf.Close()
	var res store.KeySaver
	switch kind {
	case "users":
		res = dt.NewUser()
	case "machines":
		res = dt.NewMachine()
	case "templates":
		res = dt.NewTemplate()
	case "bootenvs":
		res = dt.NewBootEnv()
	case "leases":
		res = dt.NewLease()
	case "reservations":
		res = dt.NewReservation()
	case "subnets":
		res = dt.NewSubnet()
	}

	dec := json.NewDecoder(buf)
	if err := dec.Decode(&res); err != nil {
		return false, err
	}
	return dt.create(res)
}

func mkDT(bs store.SimpleStore) *DataTracker {
	dt := NewDataTracker(bs, true, true, tmpDir, "CURL", "FURL", "AURL", "127.0.0.1", log.New(os.Stdout, "dt", 0), map[string]string{"defaultBootEnv": "default"})
	return dt
}

func TestBackingStorePersistence(t *testing.T) {
	bs, err := store.NewFileBackend(tmpDir)
	if err != nil {
		t.Errorf("Could not create boltdb: %v", err)
		return
	}
	dt := mkDT(bs)
	explDirs := []string{"users",
		"templates",
		"bootenvs",
		"machines",
		"subnets",
		"reservations",
		"leases",
	}

	for _, d := range explDirs {
		p := path.Join("test-data", d, "default.json")
		created, err := loadExample(dt, d, p)
		if !created {
			t.Errorf("Error loading test data: %v", err)
			return
		}
	}
	t.Logf("Example data loaded into the data tracker")
	t.Logf("Creating new DataTracker using the same backing store")
	dt = nil
	dt = mkDT(bs)
	// There should be one of everything in the cache now.
	for _, ot := range explDirs {
		var items []store.KeySaver
		var cnt int
		switch ot {
		case "users":
			items, cnt = dt.fetchAll(dt.NewUser()), 1
		case "templates":
			items, cnt = dt.fetchAll(dt.NewTemplate()), 1
		case "bootenvs":
			items, cnt = dt.fetchAll(dt.NewBootEnv()), 2
		case "machines":
			items, cnt = dt.fetchAll(dt.NewMachine()), 1
		case "leases":
			items, cnt = dt.fetchAll(dt.NewLease()), 1
		case "reservations":
			items, cnt = dt.fetchAll(dt.NewReservation()), 1
		case "subnets":
			items, cnt = dt.fetchAll(dt.NewSubnet()), 1
		}
		if len(items) != cnt {
			t.Errorf("Expected to find %d %s, instead found %d", cnt, ot, len(items))
		} else {
			t.Logf("Found %d %s, as expected", cnt, ot)
		}
	}
}

func TestMain(m *testing.M) {
	var err error
	tmpDir, err = ioutil.TempDir("", "datatracker-")
	if err != nil {
		log.Printf("Creating temp dir for file root failed: %v", err)
		os.Exit(1)
	}
	ret := m.Run()
	// os.RemoveAll(tmpDir)
	os.Exit(ret)
}

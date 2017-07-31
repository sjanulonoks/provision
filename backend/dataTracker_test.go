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
	op   func(Stores, store.KeySaver) (bool, error)
	t    store.KeySaver
	pass bool
}

func (test *crudTest) Test(t *testing.T, d Stores) {
	passed, err := test.op(d, test.t)
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
	d, unlocker := dt.LockEnts(kind)
	defer unlocker()
	var res store.KeySaver
	switch kind {
	case "users":
		res = dt.NewUser()
	case "machines":
		res = dt.NewMachine()
	case "profiles":
		res = dt.NewProfile()
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
	return dt.Create(d, res)
}

func mkDT(bs store.SimpleStore) *DataTracker {
	logger := log.New(os.Stdout, "dt", 0)
	dt := NewDataTracker(bs,
		tmpDir,
		"127.0.0.1",
		8091,
		8092,
		logger,
		map[string]string{"defaultBootEnv": "default", "unknownBootEnv": "ignore"},
		NewPublishers(logger))
	return dt
}

func TestBackingStorePersistence(t *testing.T) {
	// Comment out for now
	return
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
		"profiles",
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
		items := dt.objs[ot].Items()
		var cnt int
		switch ot {
		case "users":
			cnt = 2
		case "templates":
			cnt = 1
		case "bootenvs":
			cnt = 2
		case "machines":
			cnt = 1
		case "profiles":
			cnt = 2
		case "leases":
			cnt = 1
		case "reservations":
			cnt = 1
		case "subnets":
			cnt = 1
		}
		if len(items) != cnt {
			t.Errorf("Expected to find %d %s, instead found %d", cnt, ot, len(items))
		} else {
			t.Logf("Found %d %s, as expected", cnt, ot)
		}
	}

	s, e := dt.NewToken("fred", 30, "all", "a", "m")
	if e != nil {
		t.Errorf("Failed to sign token: %v\n", e)
	}
	drpClaim, e := dt.GetToken(s)
	if e != nil {
		t.Errorf("Failed to get token: %v\n", e)
	} else {
		if drpClaim.Id != "fred" {
			t.Errorf("Claim ID doesn't match: %v %v\n", "fred", drpClaim.Id)
		}
		if !drpClaim.Match("all", "a", "m") {
			t.Errorf("Claim doesn't match: %v %#v\n", []string{"all", "a", "m"}, drpClaim)
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
	err = os.RemoveAll(tmpDir)
	if err != nil {
		log.Printf("Creating temp dir for file root failed: %v", err)
		os.Exit(1)
	}
	os.Exit(ret)
}

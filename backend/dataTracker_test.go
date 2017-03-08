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
	dt := NewDataTracker(bs, true, true, tmpDir, "CURL", "default", "default", "FURL", "127.0.0.1", log.New(os.Stdout, "dt", 0))
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
		switch ot {
		case "users":
			items = dt.fetchAll(dt.NewUser())
		case "templates":
			items = dt.fetchAll(dt.NewTemplate())
		case "bootenvs":
			items = dt.fetchAll(dt.NewBootEnv())
		case "machines":
			items = dt.fetchAll(dt.NewMachine())
		case "leases":
			items = dt.fetchAll(dt.NewLease())
		case "reservations":
			items = dt.fetchAll(dt.NewReservation())
		case "subnets":
			items = dt.fetchAll(dt.NewSubnet())
		}
		if len(items) != 1 {
			t.Errorf("Expected to find 1 %s, instead found %d", ot, len(items))
		} else {
			t.Logf("Found 1 %s, as expected", ot)
		}
	}
}

func TestExtractAssets(t *testing.T) {
	bs, err := store.NewFileBackend(tmpDir)
	if err != nil {
		t.Errorf("Could not create boltdb: %v", err)
		return
	}
	dt := mkDT(bs)
	if err := dt.ExtractAssets(); err != nil {
		t.Errorf("Could not extract assets: %v", err)
		return
	}

	files := []string{
		"explode_iso.sh",
		"install-sledgehammer.sh",
		"stage1-data/busybox",
		"stage1-data/stage1_init",
		"stage1-data/udhcpc_config",
		"machines/start-up.sh",
		"files/jq",
		"bootia32.efi",
		"bootia64.efi",
		"esxi.0",
		"ipxe.efi",
		"ipxe.pxe",
		"ldlinux.c32",
		"libutil.c32",
		"lpxelinux.0",
		"pxechn.c32",
		"wimboot-2.5.2.tar.bz2",
	}

	for _, f := range files {
		if _, err := os.Stat(path.Join(tmpDir, f)); os.IsNotExist(err) {
			t.Errorf("File %s does NOT exist, but should.", f)
		}
	}

}

// Load should only be used by tests, hence it living in a _test file.
func (p *DataTracker) load(prefix, key string) store.KeySaver {
	objs, idx, found := p.lockedGet(prefix, key)
	defer objs.Unlock()
	if found {
		return objs.d[idx]
	}
	return nil
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

package backend

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"testing"
	"time"

	"github.com/digitalrebar/digitalrebar/go/common/store"
)

var (
	dt *DataTracker
)

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
	return dt.Create(res)
}

func loadExamples(dt *DataTracker) error {
	explDirs := []string{"users",
		"templates",
		"bootenvs",
		"machines",
		"leases",
		"reservations",
		"subnets",
	}

	for _, d := range explDirs {
		p := path.Join("test-data", d, "default.json")
		created, err := loadExample(dt, d, p)
		if !created {
			return err
		}
	}
	return nil
}

func TestMain(m *testing.M) {
	tmpDir, err := ioutil.TempDir("", "datatracker-")
	if err != nil {
		log.Printf("Creating temp dir for file root failed: %v", err)
		os.Exit(1)
	}
	time := time.Now()
	buf, _ := json.Marshal(&time)
	fmt.Println(string(buf))
	defer os.RemoveAll(tmpDir)
	mem := store.NewSimpleMemoryStore()
	dt = NewDataTracker(mem, true, true)
	dt.Logger = log.New(os.Stdout, "dt", 0)
	dt.FileRoot = tmpDir
	dt.DefaultBootEnv = "local"
	dt.UnknownBootEnv = "local"
	if err := dt.ExtractAssets(); err != nil {
		log.Printf("Unable to extract assets: %v", err)
		os.Exit(1)
	}
	if err := loadExamples(dt); err != nil {
		log.Printf("Unable to load test data to the data tracker: %v", err)
		os.Exit(1)
	}

	ret := m.Run()
	os.RemoveAll(tmpDir)
	os.Exit(ret)
}

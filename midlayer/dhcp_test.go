package midlayer

import (
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend"
	dhcp "github.com/krolaw/dhcp4"
)

var tmpDir string
var dataTracker *backend.DataTracker

func TestDhcpHelpers(t *testing.T) {

	xids := "test"
	xids_res := "xid 0x74657374"
	hws := "01:23:45:67:89:ab"
	hw, _ := net.ParseMAC(hws)
	req := dhcp.RequestPacket(dhcp.Discover, hw, net.ParseIP("1.1.1.1"), []byte(xids), false, nil)

	s := xid(req)
	if s != xids_res {
		t.Errorf("xid processing, expected: %s got: %s\n", xids_res, s)
	}

	s = MacStrategy(req, nil) // Options currently ignored
	if s != hws {
		t.Errorf("mac strategy processing, expected: %s got: %s\n", hws, s)
	}
}

func TestDhcpHandler(t *testing.T) {
	handler := &DhcpHandler{
		ifs:    []string{},
		port:   20000,
		bk:     dataTracker,
		strats: []*Strategy{&Strategy{Name: "MAC", GenToken: MacStrategy}},
	}
	handler.Printf("Fred rules")
}

func TestMain(m *testing.M) {
	var err error
	tmpDir, err = ioutil.TempDir("", "midlayer-")
	if err != nil {
		log.Printf("Creating temp dir for file root failed: %v", err)
		os.Exit(1)
	}
	bs, err := store.NewFileBackend(tmpDir)
	if err != nil {
		log.Printf("Could not create boltdb: %v", err)
		os.Exit(1)
	}
	dataTracker = backend.NewDataTracker(bs,
		tmpDir,
		"127.0.0.1",
		8091,
		8092,
		log.New(os.Stdout, "dt", 0),
		map[string]string{"defaultBootEnv": "default", "unknownBootEnv": "ignore"})

	ret := m.Run()
	err = os.RemoveAll(tmpDir)
	if err != nil {
		log.Printf("Creating temp dir for file root failed: %v", err)
		os.Exit(1)
	}
	os.Exit(ret)
}

package midlayer

import (
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"

	"golang.org/x/net/ipv4"

	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/pinger"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

var tmpDir string
var dataTracker *backend.DataTracker
var dhcpHandler *DhcpHandler

type dhcpTestCase struct {
	Name           string
	InUse          bool
	SrcAddr        *net.UDPAddr
	ControlMessage *ipv4.ControlMessage
	Req, Resp      string
}

func rt(t *testing.T, tc dhcpTestCase) {
	t.Helper()
	t.Logf("Processing test case %s", tc.Name)
	reqPkt, err := models.MarshalDHCP(tc.Req)
	if err != nil {
		t.Errorf("%s: Error marshalling Req: %v", tc.Name, tc.Req)
		return
	}
	req := &DhcpRequest{
		Logger: logger.New(nil).Log("dhcp"),
		idxMap: map[int][]*net.IPNet{
			1: []*net.IPNet{&net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.IPv4Mask(255, 0, 0, 0)}},
			2: []*net.IPNet{
				&net.IPNet{IP: net.IPv4(192, 168, 124, 1), Mask: net.IPv4Mask(255, 255, 255, 0)},
				&net.IPNet{IP: net.IPv4(172, 17, 0, 8), Mask: net.IPv4Mask(255, 255, 0, 0)},
			},
			3: []*net.IPNet{&net.IPNet{IP: net.IPv4(10, 0, 0, 10), Mask: net.IPv4Mask(255, 0, 0, 0)}},
		},
		nameMap: map[int]string{1: "lo", 2: "eno1", 3: "eno2"},
		srcAddr: tc.SrcAddr,
		cm:      tc.ControlMessage,
		pkt:     reqPkt,
		pinger:  pinger.Fake(!tc.InUse),
		handler: dhcpHandler,
	}
	resp := models.UnmarshalDHCP(req.Process())
	if resp != tc.Resp {
		t.Errorf("%s: Unexpected DHCP response", tc.Name)
		t.Errorf("Got:\n%s\n", resp)
		t.Errorf("Expected:\n%s\n", tc.Resp)
		return
	} else {
		t.Logf("%s handled as expected", tc.Name)
	}
}

func makeHandler(dt *backend.DataTracker, proxy bool) *DhcpHandler {
	res := &DhcpHandler{
		Logger:    logger.New(nil).Log("dhcp"),
		ifs:       []string{},
		port:      20000,
		bk:        dt,
		strats:    []*Strategy{&Strategy{Name: "MAC", GenToken: MacStrategy}},
		pinger:    pinger.Fake(true),
		proxyOnly: proxy,
	}
	return res
}

func TestMain(m *testing.M) {
	var err error
	tmpDir, err = ioutil.TempDir("", "midlayer-")
	if err != nil {
		log.Printf("Creating temp dir for file root failed: %v", err)
		os.Exit(1)
	}
	s, _ := store.Open("stack:///")
	bs := &store.Directory{Path: tmpDir}
	if err := bs.Open(nil); err != nil {
		log.Printf("Could not create directory: %v", err)
		os.Exit(1)
	}
	s.(*store.StackedStore).Push(bs, false, true)
	s.(*store.StackedStore).Push(backend.BasicContent(), false, false)
	locallogger := log.New(os.Stdout, "dt", 0)
	l := logger.New(locallogger).Log("dhcp")
	dataTracker = backend.NewDataTracker(s,
		tmpDir,
		tmpDir,
		"127.0.0.1",
		false,
		8091,
		8092,
		l,
		map[string]string{"defaultBootEnv": "default", "unknownBootEnv": "ignore"},
		backend.NewPublishers(locallogger))
	dhcpHandler = makeHandler(dataTracker, false)
	ret := m.Run()
	err = os.RemoveAll(tmpDir)
	if err != nil {
		log.Printf("Creating temp dir for file root failed: %v", err)
		os.Exit(1)
	}
	os.Exit(ret)
}

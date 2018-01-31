package midlayer

import (
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"

	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/pinger"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	"github.com/digitalrebar/store"
)

var tmpDir string
var dataTracker *backend.DataTracker
var dhcpHandler *DhcpHandler

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
	rt := dataTracker.Request(l, "subnets")
	rt.Do(func(d backend.Stores) {
		subs := []*models.Subnet{
			&models.Subnet{
				Name:              "sub2",
				Enabled:           true,
				Subnet:            "172.17.0.8/24",
				NextServer:        net.IPv4(172, 17, 0, 8),
				ActiveStart:       net.IPv4(172, 17, 0, 10),
				ActiveEnd:         net.IPv4(172, 17, 0, 15),
				ReservedLeaseTime: 7200,
				ActiveLeaseTime:   60,
				Strategy:          "MAC",
				Options: []models.DhcpOption{
					{Code: 1, Value: "255.255.0.0"},
					{Code: 3, Value: "172.17.0.1"},
					{Code: 6, Value: "172.17.0.1"},
					{Code: 15, Value: "sub2.com"},
					{Code: 28, Value: "172.17.0.255"},
					{Code: 67, Value: `{{if (eq (index . 77) "iPXE") }}default.ipxe{{else if (eq (index . 93) "0")}}lpxelinux.0{{else}}ipxe.efi{{end}}`},
				},
			},
			&models.Subnet{
				Name:              "sub1",
				Enabled:           true,
				Subnet:            "192.168.124.1/24",
				NextServer:        net.IPv4(192, 168, 124, 1),
				ActiveStart:       net.IPv4(192, 168, 124, 10),
				ActiveEnd:         net.IPv4(192, 168, 124, 15),
				ReservedLeaseTime: 7200,
				ActiveLeaseTime:   60,
				Strategy:          "MAC",
				Options: []models.DhcpOption{
					{Code: 1, Value: "255.255.0.0"},
					{Code: 3, Value: "192.168.124.1"},
					{Code: 6, Value: "192.168.124.1"},
					{Code: 15, Value: "sub1.com"},
					{Code: 28, Value: "192.168.124.255"},
					{Code: 67, Value: `{{if (eq (index . 77) "iPXE") }}default.ipxe{{else if (eq (index . 93) "0")}}lpxelinux.0{{else}}ipxe.efi{{end}}`},
				},
			},
			&models.Subnet{
				Name:              "sub3",
				Enabled:           true,
				Proxy:             true,
				Subnet:            "10.0.0.0/8",
				NextServer:        net.IPv4(10, 0, 0, 10),
				ActiveStart:       net.IPv4(10, 0, 0, 10),
				ActiveEnd:         net.IPv4(10, 0, 0, 15),
				ReservedLeaseTime: 7200,
				ActiveLeaseTime:   60,
				Strategy:          "MAC",
				Options: []models.DhcpOption{
					{Code: 1, Value: "255.0.0.0"},
					{Code: 3, Value: "10.0.0.1"},
					{Code: 6, Value: "10.0.0.1"},
					{Code: 15, Value: "sub1.com"},
					{Code: 28, Value: "10.255.255.255"},
					{Code: 67, Value: `{{if (eq (index . 77) "iPXE") }}default.ipxe{{else if (eq (index . 93) "0")}}lpxelinux.0{{else}}ipxe.efi{{end}}`},
				},
			},
		}
		for _, sub := range subs {
			_, err := rt.Create(sub)
			if err != nil {
				log.Fatalf("Error creating subnet %s: %v", sub.Name, err)
			}
		}
	})
	ret := m.Run()
	err = os.RemoveAll(tmpDir)
	if err != nil {
		log.Printf("Creating temp dir for file root failed: %v", err)
		os.Exit(1)
	}
	os.Exit(ret)
}

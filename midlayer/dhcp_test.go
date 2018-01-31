package midlayer

import (
	"net"
	"testing"

	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/pinger"
)

/*
DHCP test layout:

test-data/0000-test-name/:


*/

func rt(t *testing.T, name, in, expect string) {
	t.Helper()
	t.Logf("Processing test case %s", name)
	req := &DhcpRequest{
		Logger: logger.New(nil).Log("dhcp").SetLevel(logger.Info),
		idxMap: map[int][]*net.IPNet{
			1: []*net.IPNet{&net.IPNet{IP: net.IPv4(127, 0, 0, 1), Mask: net.IPv4Mask(255, 0, 0, 0)}},
			2: []*net.IPNet{&net.IPNet{IP: net.IPv4(192, 168, 124, 1), Mask: net.IPv4Mask(255, 255, 255, 0)}},
			3: []*net.IPNet{&net.IPNet{IP: net.IPv4(10, 0, 0, 10), Mask: net.IPv4Mask(255, 0, 0, 0)}},
		},
		nameMap: map[int]string{1: "lo", 2: "eno1", 3: "eno2"},
		pinger:  pinger.Fake(false),
		handler: dhcpHandler,
	}
	if err := req.UnmarshalText([]byte(in)); err != nil {
		t.Errorf("Error parsing request: %v", err)
		return
	}
	resp := req.PrintOutgoing(req.Process())
	if resp != expect {
		t.Errorf("%s: Unexpected DHCP response", name)
		t.Errorf("Got:\n%s\n", resp)
		t.Errorf("Expected:\n%s\n", expect)
		return
	} else {
		t.Logf("%s handled as expected", name)
	}
}

func TestParseMessage(t *testing.T) {
	rt(t, "test discover", `proto:dhcp4 iface:eno1 ifaddr:0.0.0.0:68
op:0x01 htype:0x01 hlen:0x06 hops:0x00 xid:0xed2b0d78 secs:0x0000 flags:0x0000
ci:0.0.0.0 yi:0.0.0.0 si:0.0.0.0 gi:0.0.0.0 ch:52:54:be:1e:00:00
option:code:053 val:"dis"
option:code:057 val:"1472"
option:code:093 val:"0"
option:code:094 val:"1,2,1"
option:code:060 val:"PXEClient:Arch:00000:UNDI:002001"
option:code:077 val:"iPXE"
option:code:055 val:"1,3,6,7,12,15,17,26,43,60,66,67,119,128,129,130,131,132,133,134,135,175,203"
option:code:175 val:"177,5,1,128,134,16,14,235,3,1,0,0,23,1,1,34,1,1,19,1,1,17,1,1,39,1,1,25,1,1,16,1,2,33,1,1,21,1,1,24,1,1,18,1,1"
option:code:061 val:"1,82,84,190,30,0,0"
option:code:097 val:"0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0"
`,
		`proto:dhcp4 iface:eno1 ifaddr:255.255.255.255:68
op:0x02 htype:0x01 hlen:0x06 hops:0x00 xid:0xed2b0d78 secs:0x0000 flags:0x0000
ci:0.0.0.0 yi:192.168.124.10 si:192.168.124.1 gi:0.0.0.0 ch:52:54:be:1e:00:00
sname:"192.168.124.1"
file:"default.ipxe"
option:code:053 val:"ofr"
option:code:054 val:"192.168.124.1"
option:code:051 val:"60"
option:code:001 val:"255.255.255.0"
option:code:003 val:"192.168.124.1"
option:code:006 val:"192.168.124.1"
option:code:015 val:"sub1.com"
option:code:058 val:"30"
option:code:059 val:"45"
`)
	rt(t, "test offer", `proto:dhcp4 iface:eno1 ifaddr:0.0.0.0:68
op:0x01 htype:0x01 hlen:0x06 hops:0x00 xid:0xed2b0d78 secs:0x0012 flags:0x0000
ci:0.0.0.0 yi:0.0.0.0 si:0.0.0.0 gi:0.0.0.0 ch:52:54:be:1e:00:00
option:code:053 val:"req"
option:code:057 val:"1472"
option:code:093 val:"0"
option:code:094 val:"1,2,1"
option:code:060 val:"PXEClient:Arch:00000:UNDI:002001"
option:code:077 val:"iPXE"
option:code:055 val:"1,3,6,7,12,15,17,26,43,60,66,67,119,128,129,130,131,132,133,134,135,175,203"
option:code:175 val:"177,5,1,128,134,16,14,235,3,1,0,0,23,1,1,34,1,1,19,1,1,17,1,1,39,1,1,25,1,1,16,1,2,33,1,1,21,1,1,24,1,1,18,1,1"
option:code:061 val:"1,82,84,190,30,0,0"
option:code:097 val:"0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0"
option:code:054 val:"192.168.124.1"
option:code:050 val:"192.168.124.10"
`, `proto:dhcp4 iface:eno1 ifaddr:255.255.255.255:68
op:0x02 htype:0x01 hlen:0x06 hops:0x00 xid:0xed2b0d78 secs:0x0000 flags:0x0000
ci:0.0.0.0 yi:192.168.124.10 si:192.168.124.1 gi:0.0.0.0 ch:52:54:be:1e:00:00
sname:"192.168.124.1"
file:"default.ipxe"
option:code:053 val:"ack"
option:code:054 val:"192.168.124.1"
option:code:051 val:"60"
option:code:001 val:"255.255.255.0"
option:code:003 val:"192.168.124.1"
option:code:006 val:"192.168.124.1"
option:code:015 val:"sub1.com"
option:code:058 val:"30"
option:code:059 val:"45"
`)
	rt(t, "test release", `proto:dhcp4 iface:eno1 ifaddr:192.168.124.10:68
op:0x01 htype:0x01 hlen:0x06 hops:0x00 xid:0x458a1533 secs:0x0004 flags:0x0000
ci:192.168.124.10 yi:0.0.0.0 si:0.0.0.0 gi:0.0.0.0 ch:52:54:be:1e:00:00
option:code:053 val:"rel"
option:code:061 val:"1,82,84,190,30,0,0"
option:code:054 val:"192.168.124.11"
`, ``)
}

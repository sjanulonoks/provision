package midlayer

import (
	"net"
	"testing"

	dhcp "github.com/krolaw/dhcp4"
)

func TestDHCPComponents(t *testing.T) {

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

package midlayer

import (
	"log"
	"net"
	"strings"

	dhcp "github.com/krolaw/dhcp4"
	"github.com/rackn/rocket-skates/backend"
)

type ipinfo struct {
	intf net.Interface
	myIp string
}

func RunDhcpHandler(dhcpInfo *backend.DataTracker, ifs []ipinfo) {
	handlers := make(map[int]dhcp.Handler, 0)
	for _, ii := range ifs {
		log.Println("Starting on interface: ", ii.intf.Name, " with server ip: ", ii.myIp)

		serverIP, _, _ := net.ParseCIDR(ii.myIp)
		serverIP = serverIP.To4()
		handler := &DhcpHandler{
			intf:   ii.intf,
			ip:     serverIP,
			bk:     dhcpInfo,
			strats: []*Strategy{&Strategy{Name: "MAC", GenToken: MacStrategy}},
		}
		handlers[ii.intf.Index] = handler
	}
	log.Fatal(ListenAndServeIf(handlers))
}

func StartDhcpHandlers(dhcpInfo *backend.DataTracker, serverIp string) error {
	intfs, err := net.Interfaces()
	if err != nil {
		return err
	}
	ifs := make([]ipinfo, 0, 0)
	for _, intf := range intfs {
		if (intf.Flags & net.FlagLoopback) == net.FlagLoopback {
			continue
		}
		if (intf.Flags & net.FlagUp) != net.FlagUp {
			continue
		}
		if strings.HasPrefix(intf.Name, "veth") {
			continue
		}
		var sip string
		var firstIp string

		addrs, err := intf.Addrs()
		if err != nil {
			return err
		}

		for _, addr := range addrs {
			thisIP, _, _ := net.ParseCIDR(addr.String())
			// Only care about addresses that are not link-local.
			if !thisIP.IsGlobalUnicast() {
				continue
			}
			// Only deal with IPv4 for now.
			if thisIP.To4() == nil {
				continue
			}

			if firstIp == "" {
				firstIp = addr.String()
			}
			if serverIp != "" && serverIp == addr.String() {
				sip = addr.String()
				break
			}
		}

		if sip == "" {
			if firstIp == "" {
				continue
			}
			sip = firstIp
		}

		ii := ipinfo{
			intf: intf,
			myIp: sip,
		}
		ifs = append(ifs, ii)
	}
	go RunDhcpHandler(dhcpInfo, ifs)
	return nil
}

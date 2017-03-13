package midlayer

import (
	"log"
	"net"

	dhcp "github.com/krolaw/dhcp4"
	"github.com/rackn/rocket-skates/backend"
)

func RunDhcpHandler(dhcpInfo *backend.DataTracker, ifs []*backend.Interface) {
	handlers := make(map[int]dhcp.Handler, 0)
	log.Println("DOCUMENTATION http://rocket-skates.readthedocs.io/en/latest/doc/faq-troubleshooting.html")
	for _, ii := range ifs {
		log.Println("Starting on interface: ", ii.Name, " with server ip: ", ii.ActiveAddress)

		serverIP, _, _ := net.ParseCIDR(ii.ActiveAddress)
		serverIP = serverIP.To4()
		handler := &DhcpHandler{
			ip:     serverIP,
			bk:     dhcpInfo,
			strats: []*Strategy{&Strategy{Name: "MAC", GenToken: MacStrategy}},
		}
		handlers[ii.Index] = handler
	}
	log.Fatal(ListenAndServeIf(handlers))
}

func StartDhcpHandlers(dhcpInfo *backend.DataTracker) error {
	ifs, err := dhcpInfo.GetInterfaces()
	if err != nil {
		return err
	}
	go RunDhcpHandler(dhcpInfo, ifs)
	return nil
}

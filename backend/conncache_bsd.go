// +build darwin dragonfly freebsd netbsd openbsd

package backend

import (
	"net"

	"golang.org/x/net/route"
)

func defaultIPByRoute() (*net.Interface, net.IP, error) {
	rib, _ := route.FetchRIB(0, route.RIBTypeRoute, 0)
	messages, err := route.ParseRIB(route.RIBTypeRoute, rib)
	if err != nil {
		return nil, nil, err
	}
	for _, message := range messages {
		msg, ok := message.(*route.RouteMessage)
		if !ok || len(msg.Addrs) < 2 {
			continue
		}
		addresses := msg.Addrs

		dest, ok1 := addresses[0].(*route.Inet4Addr)
		gw, ok2 := addresses[1].(*route.Inet4Addr)
		if !(ok1 && ok2) || dest == nil || gw == nil || dest.IP != [4]byte{0, 0, 0, 0} {
			continue
		}
		gwAddr := net.IP(make([]byte, 4))
		copy(gwAddr, gw.IP[:])
		iface, err := net.InterfaceByIndex(msg.Index)
		if err != nil {
			return nil, nil, err
		}
		return iface, gwAddr, nil
	}
	return nil, nil, nil
}

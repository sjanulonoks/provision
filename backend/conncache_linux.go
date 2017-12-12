// +build linux

package backend

import (
	"net"

	"github.com/vishvananda/netlink"
)

func defaultIPByRoute() (*net.Interface, net.IP, error) {
	routes, err := netlink.RouteList(nil, 0)
	if err != nil {
		return nil, nil, err
	}
	for _, route := range routes {
		if route.Gw == nil {
			continue
		}
		iface, err := net.InterfaceByIndex(route.LinkIndex)
		if err != nil {
			return nil, nil, err
		}
		return iface, route.Gw, nil
	}
	return nil, nil, nil
}

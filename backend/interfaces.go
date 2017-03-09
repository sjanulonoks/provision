package backend

import (
	"net"
	"strings"
)

// swagger:model
type Interface struct {
	// Name of the interface
	//
	// required: true
	Name string
	// Index of the interface
	//
	Index int
	// A List of Addresses on the interface (CIDR)
	//
	// required: true
	Addresses []string
	// The interface to use for this interface when
	// advertising or claiming access (CIDR)
	//
	ActiveAddress string
}

func (dt *DataTracker) GetInterfaces() ([]*Interface, error) {
	intfs, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	ifs := make([]*Interface, 0, 0)
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
			return nil, err
		}

		addrList := make([]string, 0, 0)
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
			if dt.OurAddress != "" && dt.OurAddress == addr.String() {
				sip = addr.String()
			}
			addrList = append(addrList, addr.String())
		}

		if sip == "" {
			if firstIp == "" {
				continue
			}
			sip = firstIp
		}

		ii := &Interface{
			Name:          intf.Name,
			Index:         intf.Index,
			Addresses:     addrList,
			ActiveAddress: sip,
		}
		ifs = append(ifs, ii)
	}

	return ifs, nil
}

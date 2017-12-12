// +build !linux,!darwin,!dragonfly,!freebsd,!netbsd,!openbsd

package backend

import "net"

func defaultIPByRoute() (*net.Interface, net.IP, error) {
	return nil, nil, nil
}

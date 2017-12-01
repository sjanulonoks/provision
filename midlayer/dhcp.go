package midlayer

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/ipv4"

	"github.com/digitalrebar/pinger"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/models"
	dhcp "github.com/krolaw/dhcp4"
)

func xid(p dhcp.Packet) string {
	return fmt.Sprintf("xid 0x%x", binary.BigEndian.Uint32(p.XId()))
}

type StrategyFunc func(p dhcp.Packet, options dhcp.Options) string

type Strategy struct {
	Name     string
	GenToken StrategyFunc
}

func MacStrategy(p dhcp.Packet, options dhcp.Options) string {
	return p.CHAddr().String()
}

type DhcpHandler struct {
	waitGroup  *sync.WaitGroup
	closing    bool
	proxyOnly  bool
	ifs        []string
	port       int
	conn       *ipv4.PacketConn
	bk         *backend.DataTracker
	pinger     *pinger.Pinger
	strats     []*Strategy
	publishers *backend.Publishers
}

func (h *DhcpHandler) buildOptions(p dhcp.Packet,
	l *backend.Lease,
	s *backend.Subnet,
	r *backend.Reservation,
	cm *ipv4.ControlMessage) (dhcp.Options, time.Duration, net.IP) {
	var leaseTime uint32 = 7200
	if s != nil {
		leaseTime = uint32(s.LeaseTimeFor(l.Addr) / time.Second)
	}

	opts := make(dhcp.Options)
	options := p.ParseOptions()
	srcOpts := map[int]string{}
	for c, v := range options {
		srcOpts[int(c)] = convertByteToOptionValue(c, v)
		h.Debugf("Received option: %v: %v", c, srcOpts[int(c)])
	}

	// ProxyOnly replies don't include lease info.
	// Subnets marked proxy only, don't include lease info
	dur := time.Duration(leaseTime) * time.Second
	if !h.proxyOnly && (s == nil || !s.Proxy) {
		rt := make([]byte, 4)
		binary.BigEndian.PutUint32(rt, leaseTime/2)
		rbt := make([]byte, 4)
		binary.BigEndian.PutUint32(rbt, leaseTime*3/4)
		opts[dhcp.OptionRenewalTimeValue] = rt
		opts[dhcp.OptionRebindingTimeValue] = rbt
	} else {
		dur = 0

		// Build PXEClient reply parts to get the client booting.
		if srcOpts[int(dhcp.OptionClientArchitecture)] == "0" {
			// Option encoded byte array: option 6: len: 1 value: 8, 255 (end of options)
			opts[dhcp.OptionVendorSpecificInformation] = []byte{6, 1, 8, 255}
		}
		opts[dhcp.OptionVendorClassIdentifier] = []byte("PXEClient")

		// Send back the GUID if we got a guid
		if options[97] != nil {
			opts[97] = options[97]
		}
	}

	nextServer := h.respondFrom(l.Addr, cm)
	if s != nil {
		for _, opt := range s.Options {
			if opt.Value == "" {
				h.Printf("Ignoring DHCP option %d with zero-length value", opt.Code)
				continue
			}
			c, v, err := opt.RenderToDHCP(srcOpts)
			if err != nil {
				h.Printf("Failed to render option %v: %v, %v", opt.Code, opt.Value, err)
				continue
			}
			opts[dhcp.OptionCode(c)] = v
		}
		if s.NextServer.IsGlobalUnicast() {
			nextServer = s.NextServer
		}
	}
	if r != nil {
		for _, opt := range r.Options {
			if opt.Value == "" {
				h.Printf("Ignoring DHCP option %d with zero-length value", opt.Code)
				continue
			}
			c, v, err := opt.RenderToDHCP(srcOpts)
			if err != nil {
				h.Printf("Failed to render option %v: %v, %v", opt.Code, opt.Value, err)
				continue
			}
			opts[dhcp.OptionCode(c)] = v
		}
		if r.NextServer.IsGlobalUnicast() {
			nextServer = r.NextServer
		}
	}
	return opts, dur, nextServer
}

func (h *DhcpHandler) Strategy(name string) StrategyFunc {
	for i := range h.strats {
		if h.strats[i].Name == name {
			return h.strats[i].GenToken
		}
	}
	return nil
}

func (h *DhcpHandler) Printf(f string, args ...interface{}) {
	h.bk.Printf(f, args...)
}
func (h *DhcpHandler) Infof(f string, args ...interface{}) {
	h.bk.Infof("debugDhcp", f, args...)
}
func (h *DhcpHandler) Debugf(f string, args ...interface{}) {
	h.bk.Debugf("debugDhcp", f, args...)
}

func (h *DhcpHandler) nak(p dhcp.Packet, addr net.IP) dhcp.Packet {
	return dhcp.ReplyPacket(p, dhcp.NAK, addr, nil, 0, nil)
}

const (
	reqInit = iota
	reqSelecting
	reqInitReboot
	reqRenewing
)

func reqAddr(p dhcp.Packet, msgType dhcp.MessageType, options dhcp.Options) (addr net.IP, state int) {
	reqBytes, haveReq := options[dhcp.OptionRequestedIPAddress]
	if haveReq {
		addr = net.IP(reqBytes)
	} else {
		addr = p.CIAddr()
	}
	_, haveSI := options[dhcp.OptionServerIdentifier]
	state = reqInit
	switch msgType {
	case dhcp.Request:
		if haveSI {
			state = reqSelecting
		} else if haveReq {
			state = reqInitReboot
		} else {
			state = reqRenewing
		}
	}
	return
}

func (h *DhcpHandler) intf(cm *ipv4.ControlMessage) *net.Interface {
	if cm == nil {
		return nil
	}
	iface, err := net.InterfaceByIndex(cm.IfIndex)
	if err != nil {
		h.Printf("Error looking up interface index %d: %v", cm.IfIndex, err)
	}
	return iface
}

func (h *DhcpHandler) listenAddrs(cm *ipv4.ControlMessage) []*net.IPNet {
	res := []*net.IPNet{}
	iface := h.intf(cm)
	if iface == nil {
		return res
	}
	addrs, err := iface.Addrs()
	if err != nil {
		h.Printf("Error getting addrs for interface %s: %v", iface.Name, err)
		return res
	}
	for _, addr := range addrs {
		ip, cidr, err := net.ParseCIDR(addr.String())
		if err == nil {
			cidr.IP = ip
			res = append(res, cidr)
		}
	}
	return res
}

func (h *DhcpHandler) isOneOfMyAddrs(srcAddr net.IP) bool {
	ifaces, err := net.Interfaces()
	if err != nil {
		return true
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ip, _, _ := net.ParseCIDR(addr.String())
			if ip.Equal(srcAddr) {
				return true
			}
		}
	}
	return false
}

func (h *DhcpHandler) listenIPs(cm *ipv4.ControlMessage) []net.IP {
	addrs := h.listenAddrs(cm)
	res := make([]net.IP, len(addrs))
	for i := range addrs {
		res[i] = addrs[i].IP.To4()
	}
	return res
}

func (h *DhcpHandler) respondFrom(testAddr net.IP, cm *ipv4.ControlMessage) net.IP {
	addrs := h.listenAddrs(cm)
	for _, addr := range addrs {
		if addr.Contains(testAddr) {
			return addr.IP.To4()
		}
	}
	// Well, this sucks.  Return the first address we listen on for this interface.
	if len(addrs) > 0 {
		return addrs[0].IP.To4()
	}
	// Well, this really sucks.  Return our global listen-on address
	return net.ParseIP(h.bk.OurAddress).To4()
}

func (h *DhcpHandler) listenOn(testAddr net.IP, cm *ipv4.ControlMessage) bool {
	for _, addr := range h.listenAddrs(cm) {
		if addr.Contains(testAddr) {
			return true
		}
	}
	return false
}

func (h *DhcpHandler) handleOnePacket(pktBytes []byte, cm *ipv4.ControlMessage, srcAddr net.Addr) {
	req := dhcp.Packet(pktBytes)
	if req.HLen() > 16 {
		return
	}
	options := req.ParseOptions()
	t := options[dhcp.OptionDHCPMessageType]
	if len(t) != 1 {
		return
	}
	reqType := dhcp.MessageType(t[0])
	if reqType < dhcp.Discover || reqType > dhcp.Inform {
		return
	}
	if len(h.ifs) > 0 {
		canProcess := false
		tgtIf := h.intf(cm)
		for _, ifName := range h.ifs {
			if strings.TrimSpace(ifName) == tgtIf.Name {
				canProcess = true
				break
			}
		}
		if !canProcess {
			h.Infof("DHCP: Completly ignoring packet from %s", tgtIf.Name)
			return
		}
	}
	res := h.ServeDHCP(req, reqType, options, cm)
	if res == nil {
		return
	}
	// If IP not available, broadcast
	ipStr, portStr, err := net.SplitHostPort(srcAddr.String())
	if err != nil {
		return
	}
	port, _ := strconv.Atoi(portStr)
	if req.GIAddr().Equal(net.IPv4zero) {
		if net.ParseIP(ipStr).Equal(net.IPv4zero) || req.Broadcast() {
			srcAddr = &net.UDPAddr{IP: net.IPv4bcast, Port: port}
		}
	} else {
		srcAddr = &net.UDPAddr{IP: req.GIAddr(), Port: port}
	}
	cm.Src = nil
	h.conn.WriteTo(res, cm, srcAddr)
}

func (h *DhcpHandler) Serve() error {
	defer h.waitGroup.Done()
	defer h.conn.Close()
	buf := make([]byte, 16384) // account for non-Ethernet devices maybe being used.
	for {
		h.conn.SetReadDeadline(time.Now().Add(time.Second))
		cnt, cm, srcAddr, err := h.conn.ReadFrom(buf)
		if err, ok := err.(net.Error); ok && err.Timeout() {
			continue
		}
		if err != nil {
			return err
		}
		if cnt < 240 {
			continue
		}
		pktBytes := make([]byte, cnt)
		copy(pktBytes, buf)
		go h.handleOnePacket(pktBytes, cm, srcAddr)
	}
}

func (h *DhcpHandler) ServeDHCP(p dhcp.Packet,
	msgType dhcp.MessageType,
	options dhcp.Options,
	cm *ipv4.ControlMessage) (res dhcp.Packet) {
	h.Infof("Received DHCP packet: type %s %s ciaddr %s yiaddr %s giaddr %s server %s chaddr %s",
		msgType.String(),
		xid(p),
		p.CIAddr(),
		p.YIAddr(),
		p.GIAddr(),
		p.SIAddr(),
		p.CHAddr().String())
	// need code to figure out which interface or relay it came from
	req, reqState := reqAddr(p, msgType, options)
	var err error
	switch msgType {
	case dhcp.Offer:
		serverBytes, ok := options[dhcp.OptionServerIdentifier]
		server := net.IP(serverBytes)
		if ok && !h.isOneOfMyAddrs(server) {
			h.Printf("WARNING: %s: Competing DHCP server on network: %s", xid(p), server)
		}
		if !h.isOneOfMyAddrs(cm.Src) {
			h.Printf("WARNING: %s: Competing DHCP server on network: %s", xid(p), cm.Src)
		}

		return nil
	case dhcp.Decline:
		if h.proxyOnly {
			return nil
		}
		d, unlocker := h.bk.LockEnts("leases", "reservations", "subnets")
		defer unlocker()
		leaseThing := d("leases").Find(models.Hexaddr(req))
		if leaseThing == nil {
			h.Infof("%s: Asked to decline a lease we didn't issue by %s, ignoring", xid(p), req)
			return nil
		}
		lease := backend.AsLease(leaseThing)
		stratfn := h.Strategy(lease.Strategy)
		if stratfn != nil && stratfn(p, options) == lease.Token {
			h.Infof("%s: Lease for %s declined, invalidating.", xid(p), lease.Addr)
			lease.Invalidate()
			h.bk.Save(d, lease)
		} else {
			h.Infof("%s: Received spoofed decline for %s, ignoring", xid(p), lease.Addr)
		}
		return nil
	case dhcp.Release:
		if h.proxyOnly {
			return nil
		}
		d, unlocker := h.bk.LockEnts("leases", "reservations", "subnets")
		defer unlocker()
		leaseThing := d("leases").Find(models.Hexaddr(req))
		if leaseThing == nil {
			h.Infof("%s: Asked to release a lease we didn't issue by %s, ignoring", xid(p), req)
			return nil
		}
		lease := backend.AsLease(leaseThing)
		stratfn := h.Strategy(lease.Strategy)
		if stratfn != nil && stratfn(p, options) == lease.Token {
			h.Infof("%s: Lease for %s released, expiring.", xid(p), lease.Addr)
			lease.Expire()
			h.bk.Save(d, lease)
		} else {
			h.Infof("%s: Received spoofed release for %s, ignoring", xid(p), lease.Addr)
		}
		return nil
	case dhcp.Request:
		if h.proxyOnly {
			return nil
		}
		serverBytes, ok := options[dhcp.OptionServerIdentifier]
		server := net.IP(serverBytes)
		if ok && !h.listenOn(server, cm) {
			h.Printf("WARNING: %s: Ignoring request for DHCP server %s", xid(p), net.IP(server))
			return nil
		}
		if !req.IsGlobalUnicast() {
			h.Infof("%s: NAK'ing invalid requested IP %s", xid(p), req)
			return h.nak(p, h.respondFrom(req, cm))
		}
		var lease *backend.Lease
		var reservation *backend.Reservation
		var subnet *backend.Subnet
		for _, s := range h.strats {
			lease, subnet, reservation, err = backend.FindLease(h.bk, s.Name, s.GenToken(p, options), req)
			if lease == nil &&
				subnet == nil &&
				reservation == nil &&
				err == nil {
				continue
			}
			if err != nil {
				if lease != nil {
					h.Infof("%s: %s already leased to %s:%s: %s",
						xid(p),
						req,
						lease.Strategy,
						lease.Token,
						err)
				} else {
					h.Printf("WARNING: %s: Another DHCP server may be on the network: %s", xid(p), net.IP(server))
					h.Infof("%s: %s is no longer able to be leased: %s",
						xid(p),
						req,
						err)
				}
				return h.nak(p, h.respondFrom(req, cm))
			}
			if lease != nil {
				break
			}
		}
		if lease == nil {
			if subnet != nil && subnet.Proxy {
				h.Infof("%s: Proxy Subnet should not respond to %s.", xid(p), req)
				return nil
			}
			if reqState == reqInitReboot {
				h.Infof("%s: No lease for %s in database, client in INIT-REBOOT.  Ignoring request.", xid(p), req)
				return nil
			}
			if subnet != nil || reservation != nil {
				h.Infof("%s: No lease for %s in database, NAK'ing", xid(p), req)
				return h.nak(p, h.respondFrom(req, cm))
			}

			h.Infof("%s: No lease in database, and no subnet or reservation covers %s. Ignoring request", xid(p), req)
			return nil
		}
		opts, duration, nextServer := h.buildOptions(p, lease, subnet, reservation, cm)
		reply := dhcp.ReplyPacket(p, dhcp.ACK,
			h.respondFrom(lease.Addr, cm),
			lease.Addr,
			duration,
			opts.SelectOrderOrAll(opts[dhcp.OptionParameterRequestList]))
		if nextServer.IsGlobalUnicast() {
			reply.SetSIAddr(nextServer)
		}
		h.Infof("%s: Request handing out: %s to %s via %s", xid(p), reply.YIAddr(), reply.CHAddr(), h.respondFrom(lease.Addr, cm))
		return reply
	case dhcp.Discover:
		for _, s := range h.strats {
			strat := s.Name
			token := s.GenToken(p, options)
			via := []net.IP{p.GIAddr()}
			if via[0] == nil || via[0].IsUnspecified() {
				via = h.listenIPs(cm)
			}
			var (
				lease       *backend.Lease
				subnet      *backend.Subnet
				reservation *backend.Reservation
			)
			for {
				var fresh bool
				lease, subnet, reservation, fresh = backend.FindOrCreateLease(h.bk, strat, token, req, via)
				if lease == nil {
					break
				}
				if lease.State == "PROBE" {
					if !fresh {
						// Someone other goroutine is already working this lease.
						h.Debugf("%s: Ignoring DISCOVER from %s, its request is being processed by another goroutine", xid(p), token)
						return nil
					}
					h.Debugf("%s: Testing to see if %s is in use", xid(p), lease.Addr)
					addrUsed, valid := <-h.pinger.InUse(lease.Addr, 3*time.Second)
					if !valid {
						func() {
							h.Debugf("%s: System shutting down, deleting lease for %s", xid(p), lease.Addr)
							d, unlocker := h.bk.LockEnts("leases")
							defer unlocker()
							h.bk.Remove(d, lease)
						}()
						return nil
					}
					if addrUsed {
						func() {
							h.Debugf("%s: IP address %s in use by something else, marking it as unusable for an hour.", xid(p), lease.Addr)
							d, unlocker := h.bk.LockEnts("leases")
							defer unlocker()
							lease.Invalidate()
							h.bk.Save(d, lease)
						}()
						continue
					}
					h.Debugf("%s: IP address %s appears to be free", xid(p), lease.Addr)
				} else {
					h.Debugf("%s: Resusing lease for %s", xid(p), lease.Addr)
				}
				opts, duration, nextServer := h.buildOptions(p, lease, subnet, reservation, cm)
				repType := dhcp.Offer
				addr := lease.Addr
				// Depending upon proxy state, we'll do different things.
				// If we are the proxy only (PXE/BINL port), we will Ack the discover with Boot options.
				//    send the packet back to the IP that it has if present.
				// If we are the proxy subnet, we will send an offer with boot options
				//    send the packet back to the IP that it has in the discover.
				// Otherwise, do the normal thing.
				if h.proxyOnly {
					repType = dhcp.ACK
					addr = p.YIAddr()
				}
				if (subnet != nil && subnet.Proxy) || h.proxyOnly {
					// Return the address if we are a proxy
					addr = p.YIAddr()
				}
				reply := dhcp.ReplyPacket(p, repType,
					h.respondFrom(lease.Addr, cm),
					addr,
					duration,
					opts.SelectOrderOrAll(opts[dhcp.OptionParameterRequestList]))
				if (subnet != nil && subnet.Proxy) || h.proxyOnly {
					// If we are a true proxy (NOT BINL/PXE), then broadcast
					if !h.proxyOnly {
						reply.SetBroadcast(true)
					}
					// Say who we are.
					if nextServer.IsGlobalUnicast() {
						reply.SetSIAddr(nextServer)
					}
				}
				h.Infof("%s: Discovery handing out: %s to %s via %s", xid(p), reply.YIAddr(), reply.CHAddr(), h.respondFrom(lease.Addr, cm))
				return reply
			}
		}
	}
	return nil
}

func (h *DhcpHandler) Shutdown(ctx context.Context) error {
	h.Printf("Shutting down DHCP handler")
	h.closing = true
	h.conn.Close()
	if h.pinger != nil {
		h.pinger.Close()
	}
	h.waitGroup.Wait()
	h.Printf("DHCP handler shut down")
	return nil
}

type Service interface {
	Shutdown(context.Context) error
}

func StartDhcpHandler(dhcpInfo *backend.DataTracker,
	dhcpIfs string,
	dhcpPort int,
	pubs *backend.Publishers,
	proxyOnly bool) (Service, error) {

	ifs := []string{}
	if dhcpIfs != "" {
		ifs = strings.Split(dhcpIfs, ",")
	}
	handler := &DhcpHandler{
		waitGroup:  &sync.WaitGroup{},
		ifs:        ifs,
		bk:         dhcpInfo,
		port:       dhcpPort,
		strats:     []*Strategy{&Strategy{Name: "MAC", GenToken: MacStrategy}},
		publishers: pubs,
		proxyOnly:  proxyOnly,
	}

	// If we aren't the PXE/BINL proxy, run a pinger
	if !proxyOnly {
		pinger, err := pinger.New()
		if err != nil {
			return nil, err
		}
		handler.pinger = pinger

		func() {
			d, unlocker := dhcpInfo.LockEnts("leases")
			defer unlocker()
			for _, leaseThing := range d("leases").Items() {
				lease := backend.AsLease(leaseThing)
				if lease.State != "PROBE" {
					continue
				}
				dhcpInfo.Remove(d, lease)
			}
		}()
	}

	l, err := net.ListenPacket("udp4", fmt.Sprintf(":%d", handler.port))
	if err != nil {
		return nil, err
	}
	handler.conn = ipv4.NewPacketConn(l)
	if err := handler.conn.SetControlMessage(ipv4.FlagInterface, true); err != nil {
		l.Close()
		return nil, err
	}
	handler.waitGroup.Add(1)
	go func() {
		err := handler.Serve()
		if !handler.closing {
			dhcpInfo.Logger.Fatalf("DHCP(%s) handler died: %v", proxyOnly, err)
		}
	}()
	return handler, nil
}

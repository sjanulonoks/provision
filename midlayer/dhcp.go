package midlayer

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/ipv4"

	"github.com/digitalrebar/logger"
	"github.com/digitalrebar/pinger"
	"github.com/digitalrebar/provision/backend"
	"github.com/digitalrebar/provision/backend/index"
	"github.com/digitalrebar/provision/models"
	dhcp "github.com/krolaw/dhcp4"
)

type StrategyFunc func(p dhcp.Packet, options dhcp.Options) string

type Strategy struct {
	Name     string
	GenToken StrategyFunc
}

func MacStrategy(p dhcp.Packet, options dhcp.Options) string {
	return p.CHAddr().String()
}

// DhcpRequest records all the information needed to handle a single
// in-flight DHCP request.  One of these is created for every incoming
// DHCP packet.
type DhcpRequest struct {
	logger.Logger
	idxMap    map[int][]*net.IPNet
	nameMap   map[int]string
	srcAddr   net.Addr
	defaultIP net.IP
	cm        *ipv4.ControlMessage
	pkt       dhcp.Packet
	pktOpts   dhcp.Options
	pinger    pinger.Pinger
	handler   *DhcpHandler
	lPort     int
}

func (dhr *DhcpRequest) xid() string {
	return fmt.Sprintf("xid 0x%x", binary.BigEndian.Uint32(dhr.pkt.XId()))
}

func (dhr *DhcpRequest) ifname() string {
	return dhr.nameMap[dhr.cm.IfIndex]
}
func (dhr *DhcpRequest) fill() *DhcpRequest {
	dhr.idxMap = map[int][]*net.IPNet{}
	dhr.nameMap = map[int]string{}
	ifs, err := net.Interfaces()
	if err != nil {
		dhr.Errorf("Cannot fetch local interface map: %v", err)
		return nil
	}
	for _, iface := range ifs {
		addrs, err := iface.Addrs()
		if err != nil {
			dhr.Errorf("Failed to fetch addresses for %s: %v", iface.Name, err)
			continue
		}
		toAdd := []*net.IPNet{}
		for idx := range addrs {
			addr, ok := addrs[idx].(*net.IPNet)
			if ok {
				toAdd = append(toAdd, addr)
			}
		}
		dhr.idxMap[iface.Index] = toAdd
		dhr.nameMap[iface.Index] = iface.Name
	}
	return dhr
}

func (dhr *DhcpRequest) proxyOnly() bool {
	return dhr.handler.proxyOnly
}

func (dhr *DhcpRequest) Request(locks ...string) *backend.RequestTracker {
	return dhr.handler.bk.Request(dhr.Logger, locks...)
}

func (dhr *DhcpRequest) listenAddrs() []*net.IPNet {
	addrs, ok := dhr.idxMap[dhr.cm.IfIndex]
	if !ok {
		return []*net.IPNet{}
	}
	return addrs
}

func (dhr *DhcpRequest) listenIPs() []net.IP {
	addrs := dhr.listenAddrs()
	res := make([]net.IP, len(addrs))
	for i := range addrs {
		res[i] = addrs[i].IP
	}
	return res
}

func (dhr *DhcpRequest) isOneOfMyAddrs(srcAddr net.IP) bool {
	for _, addrs := range dhr.idxMap {
		for _, addr := range addrs {
			if addr.IP.Equal(srcAddr) {
				return true
			}
		}
	}
	return false
}

func (dhr *DhcpRequest) respondFrom(testAddr net.IP) net.IP {
	addrs := dhr.listenAddrs()
	for _, addr := range addrs {
		if addr.Contains(testAddr) {
			return addr.IP.To4()
		}
	}
	// Well, this sucks.  Return the first address we listen on for this interface.
	if len(addrs) > 0 {
		dhr.Warnf("No matching subnet, will respond to %s from %s", testAddr, addrs[0].IP)
		return addrs[0].IP.To4()
	}
	// Well, this really sucks.  Return our global listen-on address
	if dhr.defaultIP != nil {
		dhr.Errorf("No address on interface index %d, using our static IP %s", dhr.cm.IfIndex, dhr.defaultIP)
		return dhr.defaultIP
	}
	addr := backend.DefaultIP(dhr.Logger)
	dhr.Errorf("No address on interface index %d, using IP with default route %v", dhr.cm.IfIndex, addr)
	return addr
}

func (dhr *DhcpRequest) listenOn(testAddr net.IP) bool {
	for _, addr := range dhr.listenAddrs() {
		if addr.Contains(testAddr) {
			return true
		}
	}
	return false
}

func (dhr *DhcpRequest) shouldOfferPXE(lease *backend.Lease) bool {
	shouldOffer := true
	rt := dhr.Request("machines", "bootenvs")
	var machine *backend.Machine
	var bootEnv *backend.BootEnv
	rt.Do(func(d backend.Stores) {
		machine = rt.MachineForMac(dhr.pkt.CHAddr().String())
		if machine == nil {
			m2 := rt.FindByIndex("machines", machine.Indexes()["Address"], lease.Addr.String())
			if m2 != nil {
				machine = backend.AsMachine(m2)
			}
		}
		if machine == nil {
			// No machine known for this MAC address or IP address.
			return
		}
		if bk := rt.Find("bootenvs", machine.BootEnv); bk != nil {
			bootEnv = backend.AsBootEnv(bk)
		} else {
			// This should never happen, but if it does then Something Bad happened.
			rt.Errorf("%s: Machine %s refers to missing BootEnv %s",
				dhr.xid(),
				machine.UUID(),
				machine.BootEnv)
			return
		}
		if !bootEnv.NetBoot() {
			// If we are not going to offer a bootfile, then there is nothing to do.
			shouldOffer = false
			return
		}
		machineSave := !machine.Address.Equal(lease.Addr)
		others, err := index.All(
			index.Sort(machine.Indexes()["Address"]),
			index.Eq(lease.Addr.String()))(rt.Index("machines"))
		if err == nil && others.Count() > 0 {
			for _, other := range others.Items() {
				if other.Key() == machine.UUID() {
					continue
				}
				oMachine := backend.AsMachine(other)
				rt.Warnf("Machine %s also has address %s, which we are handing out to %s", oMachine.UUID(), lease.Addr, machine.UUID())
				rt.Warnf("Setting machine %s address to all zeros", oMachine.UUID())
				oMachine.Address = net.IPv4(0, 0, 0, 0)
				rt.Save(oMachine)
			}
		}
		if machineSave {
			rt.Infof("%s: Updating machine %s address from %s to %s", machine.UUID(), machine.Address, lease.Addr)
			machine.Address = lease.Addr
			rt.Save(machine)
		}
	})
	return shouldOffer
}

func (dhr *DhcpRequest) buildReply(
	mt dhcp.MessageType,
	serverID,
	yAddr net.IP,
	leaseDuration time.Duration,
	options dhcp.Options) dhcp.Packet {
	order := dhr.pktOpts[dhcp.OptionParameterRequestList]
	toAdd := []dhcp.Option{}
	var fileName, sName []byte
	// The DHCP spec implies that we should use the bootp sname and file
	// fields for options 66 and 67 unless the packet size grows large
	// enough that we should use them for storing DHCP options
	// instead. (RFC2132 sections 9.4 and 9.5), respectively. For now,
	// our DHCP packets are small enough that making that happen is not
	// a concern, so if we have 66 or 67 then fill in file and sname and
	// do not include those options directly.
	//
	// THis also appears to be required to make UEFI boot mode work properly on
	// the Dell T320.
	for _, opt := range options.SelectOrderOrAll(order) {
		c, v := opt.Code, opt.Value
		switch c {
		case dhcp.OptionBootFileName:
			fileName = v
		case dhcp.OptionTFTPServerName:
			sName = v
		default:
			toAdd = append(toAdd, opt)
		}
	}
	if leaseDuration > 0 {
		toAdd = append(toAdd,
			dhcp.Option{
				Code:  dhcp.OptionRenewalTimeValue,
				Value: dhcp.OptionsLeaseTime(leaseDuration / 2),
			},
			dhcp.Option{
				Code:  dhcp.OptionRebindingTimeValue,
				Value: dhcp.OptionsLeaseTime(leaseDuration * 3 / 4),
			},
		)
	}
	res := dhcp.ReplyPacket(dhr.pkt, mt, serverID, yAddr, leaseDuration, toAdd)
	if fileName != nil {
		res.SetFile(fileName)
	}
	if sName != nil {
		res.SetSName(sName)
	}
	return res
}

func (dhr *DhcpRequest) buildOptions(
	l *backend.Lease,
	s *backend.Subnet,
	r *backend.Reservation,
	serverID net.IP) (dhcp.Options, time.Duration, net.IP, bool) {
	var leaseTime uint32 = 7200
	if s != nil {
		leaseTime = uint32(s.LeaseTimeFor(l.Addr) / time.Second)
	}

	opts := make(dhcp.Options)
	options := dhr.pkt.ParseOptions()
	shouldOfferPXE := false
	if vals, ok := options[dhcp.OptionParameterRequestList]; ok {
		shouldOfferPXE = bytes.IndexByte(vals, byte(dhcp.OptionBootFileName)) != -1
	}
	srcOpts := map[int]string{}
	for c, v := range options {
		opt := &models.DhcpOption{Code: byte(c)}
		opt.FillFromPacketOpt(v)
		srcOpts[int(c)] = opt.Value
	}
	nextServer := serverID
	// ProxyOnly replies don't include lease info.
	// Subnets marked proxy only, don't include lease info
	dur := time.Duration(leaseTime) * time.Second
	if s != nil {
		for _, opt := range s.Options {
			if opt.Value == "" {
				dhr.Debugf("Ignoring DHCP option %d with zero-length value", opt.Code)
				continue
			}
			c, v, err := opt.RenderToDHCP(srcOpts)
			if err != nil {
				dhr.Errorf("Failed to render option %v: %v, %v", opt.Code, opt.Value, err)
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
				if dhcp.OptionCode(opt.Code) != dhcp.OptionBootFileName {
					dhr.Debugf("Ignoring DHCP option %d with zero-length value", opt.Code)
					continue
				} else {
					delete(opts, dhcp.OptionBootFileName)
				}
			}
			c, v, err := opt.RenderToDHCP(srcOpts)
			if err != nil {
				dhr.Errorf("Failed to render option %v: %v, %v", opt.Code, opt.Value, err)
				continue
			}
			opts[dhcp.OptionCode(c)] = v
		}
		if r.NextServer.IsGlobalUnicast() {
			nextServer = r.NextServer
		}
	}
	if !shouldOfferPXE {
		delete(opts, dhcp.OptionTFTPServerName)
		delete(opts, dhcp.OptionBootFileName)
	} else if !dhr.shouldOfferPXE(l) {
		dhr.Debugf("Directed to not offer PXE options")
		delete(opts, dhcp.OptionTFTPServerName)
		delete(opts, dhcp.OptionBootFileName)
		shouldOfferPXE = false
	} else if _, ok := opts[dhcp.OptionBootFileName]; ok {
		if _, ok := opts[dhcp.OptionTFTPServerName]; !ok {
			opts[dhcp.OptionTFTPServerName] = []byte(nextServer.String())
		}
	}
	// If we got an incoming request with pxeclient options and the subnet responsible for this request
	// says we should answer like we are a proxy request, add the proxy options.
	if shouldOfferPXE && dhr.handler.proxyOnly || (s != nil && s.Proxy) {
		if pxeClient, ok := srcOpts[int(dhcp.OptionVendorClassIdentifier)]; ok && strings.HasPrefix(pxeClient, "PXEClient:") {
			// A single menu entry named PXE, and a 0 second timeout to automatically boot to it.
			// This is what dnsmasq does when there is only one option it will return.
			opts[dhcp.OptionVendorSpecificInformation] = []byte{0x06, 0x01, 0x08, 0x0a, 0x04, 0x00, 0x50, 0x58, 0x45, 0xff}
			// The PXE server must identify itself as "PXEClient"
			opts[dhcp.OptionVendorClassIdentifier] = []byte("PXEClient")
			// Send back the GUID if we got a guid
			if options[97] != nil {
				opts[97] = options[97]
			}
		}
		// proxy replies have no duration
		dur = 0
	}
	return opts, dur, nextServer, shouldOfferPXE
}

func (dhr *DhcpRequest) Strategy(name string) StrategyFunc {
	for idx := range dhr.handler.strats {
		if dhr.handler.strats[idx].Name == name {
			return dhr.handler.strats[idx].GenToken
		}
	}
	return nil
}

func (dhr *DhcpRequest) nak(addr net.IP) dhcp.Packet {
	return dhcp.ReplyPacket(dhr.pkt, dhcp.NAK, addr, nil, 0, nil)
}

const (
	reqInit = iota
	reqSelecting
	reqInitReboot
	reqRenewing
)

func (dhr *DhcpRequest) reqAddr(msgType dhcp.MessageType) (addr net.IP, state int) {
	reqBytes, haveReq := dhr.pktOpts[dhcp.OptionRequestedIPAddress]
	if haveReq {
		addr = net.IP(reqBytes)
	} else {
		addr = dhr.pkt.CIAddr()
	}
	_, haveSI := dhr.pktOpts[dhcp.OptionServerIdentifier]
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

func (dhr *DhcpRequest) ServeDHCP(msgType dhcp.MessageType) dhcp.Packet {
	// need code to figure out which interface or relay it came from
	req, reqState := dhr.reqAddr(msgType)
	var err error
	switch msgType {
	case dhcp.Offer:
		serverBytes, ok := dhr.pktOpts[dhcp.OptionServerIdentifier]
		server := net.IP(serverBytes)
		if ok && !dhr.isOneOfMyAddrs(server) {
			dhr.Warnf("WARNING: %s: Competing DHCP server on network: %s", dhr.xid(), server)
		}
		if !dhr.isOneOfMyAddrs(dhr.cm.Src) {
			dhr.Warnf("WARNING: %s: Competing DHCP server on network: %s", dhr.xid(), dhr.cm.Src)
		}
	case dhcp.Decline:
		if dhr.proxyOnly() {
			return nil
		}
		rt := dhr.Request("leases")
		rt.Do(func(d backend.Stores) {
			leaseThing := rt.Find("leases", models.Hexaddr(req))
			if leaseThing == nil {
				rt.Infof("%s: Asked to decline a lease we didn't issue by %s, ignoring", dhr.xid(), req)
				return
			}
			lease := backend.AsLease(leaseThing)
			stratfn := dhr.Strategy(lease.Strategy)
			if stratfn != nil && stratfn(dhr.pkt, dhr.pktOpts) == lease.Token {
				dhr.Infof("%s: Lease for %s declined, invalidating.", dhr.xid(), lease.Addr)
				lease.Invalidate()
				rt.Save(lease)
			} else {
				dhr.Infof("%s: Received spoofed decline for %s, ignoring", dhr.xid(), lease.Addr)
			}
		})
	case dhcp.Release:
		if dhr.proxyOnly() {
			return nil
		}
		rt := dhr.Request("leases")
		rt.Do(func(d backend.Stores) {
			leaseThing := rt.Find("leases", models.Hexaddr(req))
			if leaseThing == nil {
				rt.Infof("%s: Asked to release a lease we didn't issue by %s, ignoring", dhr.xid(), req)
				return
			}
			lease := backend.AsLease(leaseThing)
			stratfn := dhr.Strategy(lease.Strategy)
			if stratfn != nil && stratfn(dhr.pkt, dhr.pktOpts) == lease.Token {
				rt.Infof("%s: Lease for %s released, expiring.", dhr.xid(), lease.Addr)
				lease.Expire()
				rt.Save(lease)
			} else {
				rt.Infof("%s: Received spoofed release for %s, ignoring", dhr.xid(), lease.Addr)
			}
		})
	case dhcp.Request:
		serverBytes, ok := dhr.pktOpts[dhcp.OptionServerIdentifier]
		server := net.IP(serverBytes)
		if ok && !dhr.listenOn(server) {
			dhr.Warnf("%s: Ignoring request for DHCP server %s", dhr.xid(), net.IP(server))
			return nil
		}
		if !req.IsGlobalUnicast() {
			dhr.Infof("%s: NAK'ing invalid requested IP %s", dhr.xid(), req)
			return dhr.nak(dhr.respondFrom(req))
		}
		if dhr.proxyOnly() {
			return nil
		}
		var lease *backend.Lease
		var reservation *backend.Reservation
		var subnet *backend.Subnet
		rt := dhr.Request("leases", "reservations", "subnets")
		for _, s := range dhr.handler.strats {
			lease, subnet, reservation, err = backend.FindLease(rt, s.Name, s.GenToken(dhr.pkt, dhr.pktOpts), req)
			if lease == nil &&
				subnet == nil &&
				reservation == nil &&
				err == nil {
				continue
			}
			if err != nil {
				if lease != nil {
					dhr.Infof("%s: %s already leased to %s:%s: %s",
						dhr.xid(),
						req,
						lease.Strategy,
						lease.Token,
						err)
				} else {
					dhr.Warnf("%s: Another DHCP server may be on the network: %s", dhr.xid(), net.IP(server))
					dhr.Infof("%s: %s is no longer able to be leased: %s",
						dhr.xid(),
						req,
						err)
				}
				return dhr.nak(dhr.respondFrom(req))
			}
			if lease != nil {
				break
			}
		}
		if lease == nil {
			if subnet != nil && subnet.Proxy {
				dhr.Infof("%s: Proxy Subnet should not respond to %s.", dhr.xid(), req)
				return nil
			}
			if reqState == reqInitReboot {
				dhr.Infof("%s: No lease for %s in database, client in INIT-REBOOT.  Ignoring request.", dhr.xid(), req)
				return nil
			}
			if subnet != nil || reservation != nil {
				dhr.Infof("%s: No lease for %s in database, NAK'ing", dhr.xid(), req)
				return dhr.nak(dhr.respondFrom(req))
			}

			dhr.Infof("%s: No lease in database, and no subnet or reservation covers %s. Ignoring request", dhr.xid(), req)
			return nil
		}
		serverID := dhr.respondFrom(lease.Addr)
		opts, duration, nextServer, _ := dhr.buildOptions(lease, subnet, reservation, serverID)
		reply := dhr.buildReply(dhcp.ACK, serverID, lease.Addr, duration, opts)
		if nextServer.IsGlobalUnicast() {
			reply.SetSIAddr(nextServer)
		}
		rt.Infof("%s: Request handing out: %s to %s via %s",
			dhr.xid(),
			reply.YIAddr(),
			reply.CHAddr(),
			serverID)
		return reply
	case dhcp.Discover:
		for _, s := range dhr.handler.strats {
			strat := s.Name
			token := s.GenToken(dhr.pkt, dhr.pktOpts)
			via := []net.IP{dhr.pkt.GIAddr()}
			if via[0] == nil || via[0].IsUnspecified() {
				via = dhr.listenIPs()
			}
			var (
				lease       *backend.Lease
				subnet      *backend.Subnet
				reservation *backend.Reservation
			)
			rt := dhr.Request("leases", "reservations", "subnets")
			for {
				var fresh bool
				lease, subnet, reservation, fresh = backend.FindOrCreateLease(rt, strat, token, req, via)
				if lease == nil {
					break
				}
				if lease.State == "PROBE" {
					if !fresh {
						// Someone other goroutine is already working this lease.
						rt.Debugf("%s: Ignoring DISCOVER from %s, its request is being processed by another goroutine", dhr.xid(), token)
						return nil
					}
					rt.Debugf("%s: Testing to see if %s is in use", dhr.xid(), lease.Addr)
					addrUsed, valid := <-dhr.pinger.InUse(lease.Addr.String(), 3*time.Second)
					if !valid {
						rt.Do(func(d backend.Stores) {
							rt.Debugf("%s: System shutting down, deleting lease for %s", dhr.xid(), lease.Addr)
							rt.Remove(lease)
						})
						return nil
					}
					if addrUsed {
						rt.Do(func(d backend.Stores) {
							rt.Debugf("%s: IP address %s in use by something else, marking it as unusable for an hour.", dhr.xid(), lease.Addr)
							lease.Invalidate()
							rt.Save(lease)
						})
						continue
					}
					rt.Do(func(d backend.Stores) {
						rt.Debugf("%s: IP address %s appears to be free", dhr.xid(), lease.Addr)
						lease.State = "OFFER"
						rt.Save(lease)
					})
				} else {
					rt.Debugf("%s: Reusing lease for %s", dhr.xid(), lease.Addr)
				}
				break
			}
			if lease == nil {
				return nil
			}
			serverID := dhr.respondFrom(lease.Addr)
			opts, duration, nextServer, shouldPXE := dhr.buildOptions(lease, subnet, reservation, serverID)
			repType := dhcp.Offer
			addr := lease.Addr
			// If we are responding as a Proxy DHCP server but we were directed to omit PXE options,
			// then we should not respond to this request.
			if !shouldPXE && (dhr.proxyOnly() || (subnet != nil && subnet.Proxy)) {
				return nil
			}
			// Depending upon proxy state, we'll do different things.
			// If we are the proxy only (PXE/BINL port), we will Ack the discover with Boot options.
			//    send the packet back to the IP that it has if present.
			// If we are the proxy subnet, we will send an offer with boot options
			//    send the packet back to the IP that it has in the discover.
			// Otherwise, do the normal thing.
			if dhr.proxyOnly() || (subnet != nil && subnet.Proxy) {
				addr = dhr.pkt.YIAddr()
				if dhr.proxyOnly() {
					repType = dhcp.ACK
				}
			}
			reply := dhr.buildReply(repType, serverID, addr, duration, opts)
			if (subnet != nil && subnet.Proxy) || dhr.proxyOnly() {
				// If we are a true proxy (NOT BINL/PXE), then broadcast
				if !dhr.proxyOnly() {
					reply.SetBroadcast(true)
				}
			}
			// Say who we are.
			if nextServer.IsGlobalUnicast() {
				reply.SetSIAddr(nextServer)
			}
			dhr.Infof("%s: Discovery handing out: %s to %s via %s",
				dhr.xid(),
				reply.YIAddr(),
				reply.CHAddr(),
				serverID)
			return reply
		}
	}
	return nil
}

func (dhr *DhcpRequest) Process() dhcp.Packet {
	if dhr.IsDebug() {
		dhr.Debugf("Handling packet:\n%s", dhr.PrintIncoming())
	}
	if dhr.pkt.HLen() > 16 {
		dhr.Errorf("Invalid hlen")
		return nil
	}
	dhr.pktOpts = dhr.pkt.ParseOptions()
	var reqType dhcp.MessageType
	if t, ok := dhr.pktOpts[dhcp.OptionDHCPMessageType]; !ok || len(t) != 1 {
		dhr.Errorf("Missing DHCP message type")
		return nil
	} else if reqType = dhcp.MessageType(t[0]); reqType < dhcp.Discover || reqType > dhcp.Inform {
		dhr.Errorf("Invalid DHCP message type")
		return nil
	}
	tgtName := dhr.ifname()
	if tgtName == "" {
		dhr.Infof("Inferface at index %d vanished", dhr.cm.IfIndex)
		return nil
	}
	if len(dhr.handler.ifs) > 0 {
		canProcess := false
		for _, ifName := range dhr.handler.ifs {
			if strings.TrimSpace(ifName) == tgtName {
				canProcess = true
				break
			}
		}
		if !canProcess {
			dhr.Infof("%s Ignoring packet from interface %s", dhr.xid(), tgtName)
			return nil
		}
	}
	res := dhr.ServeDHCP(reqType)
	if res == nil {
		return nil
	}
	// If IP not available, broadcast
	ipStr, portStr, err := net.SplitHostPort(dhr.srcAddr.String())
	if err != nil {
		return nil
	}
	port, _ := strconv.Atoi(portStr)
	if dhr.pkt.GIAddr().Equal(net.IPv4zero) {
		if net.ParseIP(ipStr).Equal(net.IPv4zero) || dhr.pkt.Broadcast() {
			dhr.srcAddr = &net.UDPAddr{IP: net.IPv4bcast, Port: port}
		}
	} else {
		dhr.srcAddr = &net.UDPAddr{IP: dhr.pkt.GIAddr(), Port: port}
	}
	dhr.cm.Src = nil
	if dhr.IsDebug() {
		dhr.Debugf("Sending packet:\n%s", dhr.PrintOutgoing(res))
	}
	return res
}

func (dhr *DhcpRequest) Run() {
	res := dhr.Process()
	if res == nil {
		return
	}
	dhr.handler.conn.WriteTo(res, dhr.cm, dhr.srcAddr)
}

// DhcpHandler is responsible for listening to incoming DHCP packets,
// building a DhcpRequest for each one, then kicking that reqest off
// to handle the packet.
type DhcpHandler struct {
	logger.Logger
	waitGroup  *sync.WaitGroup
	closing    bool
	proxyOnly  bool
	ifs        []string
	port       int
	conn       *ipv4.PacketConn
	bk         *backend.DataTracker
	pinger     pinger.Pinger
	strats     []*Strategy
	publishers *backend.Publishers
}

func (h *DhcpHandler) NewRequest(buf []byte, cm *ipv4.ControlMessage, srcAddr net.Addr) *DhcpRequest {
	res := &DhcpRequest{}
	res.Logger = h.Logger.Fork()
	res.srcAddr = srcAddr
	res.defaultIP = net.ParseIP(h.bk.OurAddress)
	res.cm = cm
	res.pkt = dhcp.Packet(buf)
	res.handler = h
	res.pinger = h.pinger
	res.lPort = h.port
	res.fill()
	return res
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
		go h.NewRequest(pktBytes, cm, srcAddr).Run()
	}
}

func (h *DhcpHandler) Shutdown(ctx context.Context) error {
	h.Infof("Shutting down DHCP handler")
	h.closing = true
	h.conn.Close()
	if h.pinger != nil {
		h.pinger.Close()
	}
	h.waitGroup.Wait()
	h.Infof("DHCP handler shut down")
	return nil
}

type Service interface {
	Shutdown(context.Context) error
}

func StartDhcpHandler(dhcpInfo *backend.DataTracker,
	log logger.Logger,
	dhcpIfs string,
	dhcpPort int,
	pubs *backend.Publishers,
	proxyOnly bool,
	fakePinger bool) (Service, error) {

	ifs := []string{}
	if dhcpIfs != "" {
		ifs = strings.Split(dhcpIfs, ",")
	}
	handler := &DhcpHandler{
		Logger:     log,
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
		if handler.pinger == nil {
			if fakePinger {
				handler.pinger = pinger.Fake(true)
			} else {
				pinger, err := pinger.ICMP()
				if err != nil {
					return nil, err
				}
				handler.pinger = pinger
			}
		}
		rt := handler.bk.Request(log, "leases")
		rt.Do(func(d backend.Stores) {
			for _, leaseThing := range d("leases").Items() {
				lease := backend.AsLease(leaseThing)
				if lease.State != "PROBE" {
					continue
				}
				rt.Remove(lease)
			}
		})
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
			handler.Fatalf("DHCP(%v) handler died: %v", proxyOnly, err)
		}
	}()
	return handler, nil
}

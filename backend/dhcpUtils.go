package backend

import (
	"fmt"
	"math/big"
	"net"
	"time"
)

// LeaseNAK is the error that shall be returned when we cannot give a
// system the IP address it requested.  If FindLease or
// FindOrCreateLease return this as their error, then the DHCP
// midlayer must NAK the request.
type LeaseNAK error

func _findLease(leases, reservations *dtobjs, strat, token string, req net.IP) (lease *Lease, err error) {
	if req != nil && req.IsGlobalUnicast() {
		hexreq := hexaddr(req.To4())
		idx, found := leases.find(hexreq)
		if !found {
			err = LeaseNAK(fmt.Errorf("No lease for %s exists", hexreq))
			return
		}
		// Found a lease that exists for the requested address.
		lease = AsLease(leases.d[idx])
		if !lease.Expired() && (lease.Token != token || lease.Strategy != strat) {
			// And it belongs to someone else.  So sad, gotta NAK
			err = LeaseNAK(fmt.Errorf("Lease for %s owned by %s:%s",
				hexreq, lease.Strategy, lease.Token))
			lease = nil
			return
		}
	} else {
		for i := range leases.d {
			lease = AsLease(leases.d[i])
			if lease.Token == token && lease.Strategy == strat {
				break
			}
			lease = nil
		}
		if lease == nil {
			// We did not find a lease for this system to renew.
			return
		}
	}
	// This is the lease we want, but if there is a conflicting reservation we
	// may force the client to give it up.
	if ridx, rfound := reservations.find(lease.Key()); rfound {
		reservation := AsReservation(reservations.d[ridx])
		if reservation.Strategy != lease.Strategy ||
			reservation.Token != lease.Token {
			lease.Invalidate()
			err = LeaseNAK(fmt.Errorf("Reservation %s (%s:%s conflicts with %s:%s",
				reservation.Addr,
				reservation.Strategy,
				reservation.Token,
				lease.Strategy,
				lease.Token))
			return
		}
	}
	lease.Strategy = strat
	lease.Token = token
	lease.ExpireTime = time.Now().Add(2 * time.Second)
	lease.p.Logger.Printf("Found our lease for strat: %s token %s, will use it", strat, token)
	return
}

func findLease(dt *DataTracker, strat, token string, req net.IP) (lease *Lease, err error) {
	leases := dt.lockFor("leases")
	reservations := dt.lockFor("reservations")
	defer leases.Unlock()
	defer reservations.Unlock()
	lease, err = _findLease(leases, reservations, strat, token, req)
	return
}

// FindLease finds an appropriate matching Lease.
// If a non-nil error is returned, the DHCP system must NAK the response.
// If lease and error are nil, the DHCP system must not respond to the request.
// Otherwise, the lease will be returned with its ExpireTime updated and the Lease saved.
//
// This function should be called in response to a DHCPREQUEST.
func FindLease(dt *DataTracker, strat, token string, req net.IP) (lease *Lease, err error) {
	lease, err = findLease(dt, strat, token, req)
	if lease != nil && err == nil {
		subnet := lease.Subnet()
		reservation := lease.Reservation()
		if subnet != nil {
			lease.ExpireTime = time.Now().Add(subnet.LeaseTimeFor(lease.Addr))
		} else if reservation != nil {
			lease.ExpireTime = time.Now().Add(2 * time.Hour)
		} else {
			dt.remove(lease)
			err = LeaseNAK(fmt.Errorf("Lease %s has no reservation or subnet, it is dead to us.", lease.Addr))
			return
		}
		dt.save(lease)
	}
	return lease, err
}

func findViaReservation(leases, reservations *dtobjs, strat, token string) (lease *Lease) {
	var reservation *Reservation
	for idx := range reservations.d {
		reservation = AsReservation(reservations.d[idx])
		if reservation.Token == token && reservation.Strategy == strat {
			break
		}
		reservation = nil
	}
	if reservation == nil {
		return
	}
	// We found a reservation for this strategy/token
	// combination, see if we can create a lease using it.
	if lidx, found := leases.find(reservation.Key()); found {
		// We found a lease for this IP.
		lease = AsLease(leases.d[lidx])
		if lease.Token == reservation.Token &&
			lease.Strategy == reservation.Strategy {
			// This is our lease.  Renew it.
			lease.p.Logger.Printf("Reservation for %s has a lease, using it.", lease.Addr.String())
			return
		}
		if lease.Expired() {
			// The lease has expired.  Take it over
			lease.p.Logger.Printf("Reservation for %s is taking over an expired lease", lease.Addr.String())
			lease.Token = token
			lease.Strategy = strat
			return
		}
		// The lease has not expired, and it is not ours.
		// We have no choice but to fall through to subnet code until
		// the current lease has expired.
		reservation.p.Logger.Printf("Reservation %s (%s:%s): A lease exists for that address, has been handed out to %s:%s", reservation.Addr, reservation.Strategy, reservation.Token, lease.Strategy, lease.Token)
		lease = nil
		return
	}
	// We did not find a lease for this IP, and findLease has already guaranteed that
	// either there is no lease for this token or that the old lease has been NAK'ed.
	// We are free to create a new lease for this Reservation.
	lease = &Lease{
		Addr:     reservation.Addr,
		Strategy: reservation.Strategy,
		Token:    reservation.Token,
	}
	leases.add(lease)
	return
}

func findViaSubnet(leases, subnets, reservations *dtobjs, strat, token string, via net.IP) (lease *Lease) {
	if via == nil || !via.IsGlobalUnicast() {
		// Without a via address, we have no way to look up the appropriate subnet
		// to try.  Since that is the case, return nothing.  The DHCP midlayer
		// should take that as a cue to not respond at all.
		return
	}
	var subnet *Subnet
	for idx := range subnets.d {
		subnet = AsSubnet(subnets.d[idx])
		if subnet.subnet().Contains(via) {
			break
		}
		subnet = nil
	}
	if subnet == nil {
		// There is no subnet that can handle this via.
		return
	}
	if subnet.nextLeasableIP == nil {
		subnet.nextLeasableIP = net.IP(make([]byte, 4))
		copy(subnet.nextLeasableIP, subnet.ActiveStart.To4())
	}
	currLeases := AsLeases(leases.subset(subnet.aBounds()))
	currReservations := AsReservations(reservations.subset(subnet.aBounds()))
	reservedAddrs := map[string]struct{}{}
	for i := range currReservations {
		reservedAddrs[currReservations[i].Key()] = struct{}{}
	}
	leasedAddrs := map[string]int{}
	for i := range currLeases {
		leasedAddrs[currLeases[i].Key()] = i
	}
	one := big.NewInt(1)
	end := &big.Int{}
	curr := &big.Int{}
	end.SetBytes(subnet.ActiveEnd.To4())
	curr.SetBytes(subnet.nextLeasableIP)
	// First, check from nextLeasableIp to ActiveEnd
	for curr.Cmp(end) != 1 {
		addr := net.IP(curr.Bytes()).To4()
		hex := hexaddr(addr)
		if _, ok := reservedAddrs[hex]; ok {
			curr.Add(curr, one)
			continue
		}
		i, ok := leasedAddrs[hex]
		if !ok {
			// No lease exists for this address, and it is available.
			curr.Add(curr, one)
			subnet.nextLeasableIP = net.IP(curr.Bytes()).To4()
			lease = &Lease{
				Addr:     addr,
				Token:    token,
				Strategy: strat,
			}
			leases.add(lease)
			return
		}
		lease = currLeases[i]
		if lease.Expired() {
			// There is a lease for this address, and it is expired.
			// Take it over.
			curr.Add(curr, one)
			subnet.nextLeasableIP = net.IP(curr.Bytes()).To4()
			lease.Addr = addr
			lease.Token = token
			lease.Strategy = strat
			return lease
		}
		// No candidate found, continue to the next one.
		curr.Add(curr, one)
	}
	// Next, check from ActiveStart to nextLeasableIP
	end.SetBytes(subnet.nextLeasableIP)
	curr.SetBytes(subnet.ActiveStart.To4())
	for curr.Cmp(end) != 1 {
		addr := net.IP(curr.Bytes()).To4()
		hex := hexaddr(addr)
		if _, ok := reservedAddrs[hex]; ok {
			curr.Add(curr, one)
			continue
		}
		i, ok := leasedAddrs[hex]
		if !ok {
			// No lease exists for this address, and it is available.
			curr.Add(curr, one)
			subnet.nextLeasableIP = net.IP(curr.Bytes()).To4()
			lease = &Lease{
				Addr:     addr,
				Token:    token,
				Strategy: strat,
			}
			leases.add(lease)
			return lease
		}
		lease = currLeases[i]
		if lease.Expired() {
			// There is a lease for this address, and it is expired.
			// Take it over.
			curr.Add(curr, one)
			subnet.nextLeasableIP = net.IP(curr.Bytes()).To4()
			lease.Addr = addr
			lease.Token = token
			lease.Strategy = strat
			return lease
		}
		// No candidate found, continue to the next one.
		curr.Add(curr, one)
	}
	// Sorry, the subnet is full. No lease for you.
	return nil
}

func findOrCreateLease(dt *DataTracker, strat, token string, req, via net.IP) *Lease {
	leases := dt.lockFor("leases")
	reservations := dt.lockFor("reservations")
	subnets := dt.lockFor("subnets")
	defer leases.Unlock()
	defer reservations.Unlock()
	defer subnets.Unlock()
	lease, err := _findLease(leases, reservations, strat, token, req)
	if lease == nil || err != nil {
		lease = findViaReservation(leases, reservations, strat, token)
	}
	if lease == nil {
		lease = findViaSubnet(leases, subnets, reservations, strat, token, via)
	}
	if lease != nil {
		lease.ExpireTime = time.Now().Add(2 * time.Second)
	}
	return lease
}

// FindOrCreateLease will return a lease for the passed information, creating it if it can.
// If a non-nil Lease is returned, it has been saved and the DHCP system can offer it.
// If the returned lease is nil, then the DHCP system should not respond.
//
// This function should be called for DHCPDISCOVER.
func FindOrCreateLease(dt *DataTracker, strat, token string, req, via net.IP) *Lease {
	lease := findOrCreateLease(dt, strat, token, req, via)
	if lease != nil {
		lease.ExpireTime = time.Now().Add(time.Minute)
		dt.save(lease)
	}
	return lease
}

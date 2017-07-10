package backend

import (
	"fmt"
	"net"
	"time"

	"github.com/digitalrebar/digitalrebar/go/common/store"
	"github.com/digitalrebar/provision/backend/index"
)

// LeaseNAK is the error that shall be returned when we cannot give a
// system the IP address it requested.  If FindLease or
// FindOrCreateLease return this as their error, then the DHCP
// midlayer must NAK the request.
type LeaseNAK error

func findLease(d Stores, dt *DataTracker, strat, token string, req net.IP) (lease *Lease, err error) {
	reservations, leases := d("reservations"), d("leases")
	hexreq := Hexaddr(req.To4())
	found := leases.Find(hexreq)
	if found == nil {
		err = LeaseNAK(fmt.Errorf("No lease for %s exists", hexreq))
		return
	}
	// Found a lease that exists for the requested address.
	lease = AsLease(found)
	if !lease.Expired() && (lease.Token != token || lease.Strategy != strat) {
		// And it belongs to someone else.  So sad, gotta NAK
		err = LeaseNAK(fmt.Errorf("Lease for %s owned by %s:%s",
			hexreq, lease.Strategy, lease.Token))
		lease = nil
		return
	}
	// This is the lease we want, but if there is a conflicting reservation we
	// may force the client to give it up.
	if rfound := reservations.Find(lease.Key()); rfound != nil {
		reservation := AsReservation(rfound)
		if reservation.Strategy != lease.Strategy ||
			reservation.Token != lease.Token {
			lease.Invalidate()
			dt.Save(d, lease)
			err = LeaseNAK(fmt.Errorf("Reservation %s (%s:%s conflicts with %s:%s",
				reservation.Addr,
				reservation.Strategy,
				reservation.Token,
				lease.Strategy,
				lease.Token))
			lease = nil
			return
		}
	}
	lease.Strategy = strat
	lease.Token = token
	lease.ExpireTime = time.Now().Add(2 * time.Second)
	lease.p.Infof("debugDhcp", "Found our lease for strat: %s token %s, will use it", strat, token)
	return
}

// FindLease finds an appropriate matching Lease.
// If a non-nil error is returned, the DHCP system must NAK the response.
// If lease and error are nil, the DHCP system must not respond to the request.
// Otherwise, the lease will be returned with its ExpireTime updated and the Lease saved.
//
// This function should be called in response to a DHCPREQUEST.
func FindLease(dt *DataTracker,
	strat, token string,
	req net.IP) (lease *Lease, subnet *Subnet, reservation *Reservation, err error) {
	d, unlocker := dt.LockEnts("leases", "reservations", "subnets")
	defer unlocker()
	lease, err = findLease(d, dt, strat, token, req)
	if lease != nil && err == nil {
		subnet = lease.Subnet(d)
		reservation = lease.Reservation(d)
		if subnet != nil {
			lease.ExpireTime = time.Now().Add(subnet.LeaseTimeFor(lease.Addr))
		} else if reservation != nil {
			lease.ExpireTime = time.Now().Add(2 * time.Hour)
		} else {
			dt.Remove(d, lease)
			err = LeaseNAK(fmt.Errorf("Lease %s has no reservation or subnet, it is dead to us.", lease.Addr))
			lease = nil
			return
		}
		dt.Save(d, lease)
	}
	return
}

func findViaReservation(leases, reservations *Store, strat, token string, req net.IP) (lease *Lease, reservation *Reservation) {
	if req != nil && req.IsGlobalUnicast() {
		hex := Hexaddr(req)
		ok := reservations.Find(hex)
		if ok != nil {
			reservation = AsReservation(ok)
			if reservation.Token != token || reservation.Strategy != strat {
				reservation = nil
			}
		}
	} else {
		for _, i := range reservations.Items() {
			reservation = AsReservation(i)
			if reservation.Token == token && reservation.Strategy == strat {
				break
			}
			reservation = nil
		}
	}
	if reservation == nil {
		return
	}
	// We found a reservation for this strategy/token
	// combination, see if we can create a lease using it.
	if found := leases.Find(reservation.Key()); found != nil {
		// We found a lease for this IP.
		lease = AsLease(found)
		if lease.Token == reservation.Token &&
			lease.Strategy == reservation.Strategy {
			// This is our lease.  Renew it.
			lease.p.Infof("debugDhcp", "Reservation for %s has a lease, using it.", lease.Addr.String())
			return
		}
		if lease.Expired() {
			// The lease has expired.  Take it over
			lease.p.Infof("debugDhcp", "Reservation for %s is taking over an expired lease", lease.Addr.String())
			lease.Token = token
			lease.Strategy = strat
			return
		}
		// The lease has not expired, and it is not ours.
		// We have no choice but to fall through to subnet code until
		// the current lease has expired.
		reservation.p.Infof("debugDhcp", "Reservation %s (%s:%s): A lease exists for that address, has been handed out to %s:%s", reservation.Addr, reservation.Strategy, reservation.Token, lease.Strategy, lease.Token)
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
	leases.Add(lease)
	return
}

func findViaSubnet(leases, subnets, reservations *Store, strat, token string, req net.IP, vias []net.IP) (lease *Lease, subnet *Subnet) {
	for _, idx := range subnets.Items() {
		candidate := AsSubnet(idx)
		for _, via := range vias {
			if via == nil || !via.IsGlobalUnicast() {
				continue
			}
			if candidate.subnet().Contains(via) && candidate.Strategy == strat {
				subnet = candidate
				break
			}
		}
	}
	if subnet == nil {
		// There is no subnet that can handle the vias we want
		return
	}
	currLeases, _ := index.Between(
		Hexaddr(subnet.ActiveStart),
		Hexaddr(subnet.ActiveEnd))(&leases.Index)
	currReservations, _ := index.Between(
		Hexaddr(subnet.ActiveStart),
		Hexaddr(subnet.ActiveEnd))(&reservations.Index)
	usedAddrs := map[string]store.KeySaver{}
	for _, i := range currLeases.Items() {
		currLease := AsLease(i)
		// While we are iterating over leases, see if we run across a candidate.
		if (req == nil || req.IsUnspecified() || currLease.Addr.Equal(req)) &&
			currLease.Strategy == strat && currLease.Token == token {
			lease = currLease
		}
		// Leases get a false in the map.
		usedAddrs[currLease.Key()] = currLease
	}
	for _, i := range currReservations.Items() {
		// While we are iterating over reservations, see if any candidate we found is still kosher.
		currRes := AsReservation(i)
		if lease != nil &&
			currRes.Strategy == strat &&
			currRes.Token == token {
			// If we have a matching reservation and we found a similar candidate,
			// then the candidate cannot possibly be a lease we should use,
			// because it would have been refreshed by the lease code.
			lease = nil
		}
		// Reservations get true
		usedAddrs[currRes.Key()] = currRes
	}
	if lease != nil {
		subnet.p.Infof("debugDhcp", "Subnet %s: handing out existing lease for %s to %s:%s", subnet.Name, lease.Addr, strat, token)
		return
	}
	subnet.p.Infof("debugDhcp", "Subnet %s: %s:%s is in my range, attempting lease creation.", subnet.Name, strat, token)
	lease, _ = subnet.next(usedAddrs, token, req)
	if lease != nil {
		if leases.Find(lease.Key()) == nil {
			leases.Add(lease)
		}
		return
	}
	subnet.p.Infof("debugDhcp", "Subnet %s: No lease for %s:%s, it gets no IP from us", subnet.Name, strat, token)
	return nil, nil
}

// FindOrCreateLease will return a lease for the passed information, creating it if it can.
// If a non-nil Lease is returned, it has been saved and the DHCP system can offer it.
// If the returned lease is nil, then the DHCP system should not respond.
//
// This function should be called for DHCPDISCOVER.
func FindOrCreateLease(dt *DataTracker,
	strat, token string,
	req net.IP,
	via []net.IP) (lease *Lease, subnet *Subnet, reservation *Reservation) {
	d, unlocker := dt.LockEnts("subnets", "reservations", "leases")
	defer unlocker()
	leases, reservations, subnets := d("leases"), d("reservations"), d("subnets")
	lease, reservation = findViaReservation(leases, reservations, strat, token, req)
	if lease == nil {
		lease, subnet = findViaSubnet(leases, subnets, reservations, strat, token, req, via)
	}
	if lease != nil {
		// Clean up any other leases that have this strategy and token lying around.
		toRemove := []store.KeySaver{}
		for _, dup := range leases.Items() {
			candidate := AsLease(dup)
			if candidate.Strategy == strat &&
				candidate.Token == token &&
				!candidate.Addr.Equal(lease.Addr) {
				toRemove = append(toRemove, candidate)
			}
		}
		leases.Remove(toRemove...)
		lease.p = dt
		lease.ExpireTime = time.Now().Add(time.Minute)
		dt.Save(d, lease)
	}
	return
}

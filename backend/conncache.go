package backend

import (
	"fmt"
	"math/big"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/digitalrebar/logger"
)

type cacheLine struct {
	local, remote net.IP
	timeout       time.Time
}

func (c cacheLine) String() string {
	return fmt.Sprintf("l: %s, r: %s: t:%s", c.local, c.remote, c.timeout)
}

func a2i(n net.IP) *big.Int {
	res := &big.Int{}
	res.SetBytes([]byte(n.To16()))
	return res
}

var addrCache = []cacheLine{}
var addrCacheMux = &sync.RWMutex{}

// How long we will keep cache entries around for.
var connCacheTimeout = 10 * time.Minute

// AddToCache adds a new remote -> local IP address mapping to the
// connection cache.  If the remote address is already in the cache,
// its corresponding local address is updates and the timeout is bumped.
// Mappings that have not been accessed with LocalFor
// or updated with AddToCache will be evicted if not used for more than 10 minutes.
func AddToCache(l logger.Logger, la, ra net.IP) {
	if la == nil || ra == nil {
		l.Errorf("addrCache: nil addr passed: local: %v, remote: %v", la, ra)
		return
	}
	local, remote := net.IP(make([]byte, 16)), net.IP(make([]byte, 16))
	copy(local, la.To16())
	copy(remote, ra.To16())

	addrCacheMux.Lock()
	defer addrCacheMux.Unlock()
	key := a2i(remote)
	idx := sort.Search(len(addrCache), func(i int) bool {
		return a2i(addrCache[i].remote).Cmp(key) != 1
	})
	if idx == len(addrCache) {
		l.Infof("addrCache: Adding local %s ->  remote %s", local, remote)
		addrCache = append(addrCache, cacheLine{local, remote, time.Now().Add(connCacheTimeout)})
		return
	}
	if addrCache[idx].remote.Equal(remote) {
		addrCache[idx].local = local
		addrCache[idx].timeout = time.Now().Add(connCacheTimeout)
		l.Debugf("addrCache: Renewing local %s -> remote %s", local, remote)
		return
	}
	addrCache = append(addrCache, cacheLine{})
	copy(addrCache[idx+1:], addrCache[idx:])
	l.Infof("addrCache: Adding local %s ->  remote %s", local, remote)
	addrCache[idx] = cacheLine{local, remote, time.Now().Add(connCacheTimeout)}
}

// LocalFor returns the local IP address that has responded to TFTP or
// HTTP requests for the given remote IP.  It also bumps the timeout.
func LocalFor(l logger.Logger, ra net.IP) net.IP {
	if ra == nil || ra.IsUnspecified() {
		return nil
	}
	addrCacheMux.RLock()
	defer addrCacheMux.RUnlock()
	remote := net.IP(make([]byte, 16))
	copy(remote, ra.To16())
	key := a2i(remote)
	idx := sort.Search(len(addrCache), func(i int) bool {
		return a2i(addrCache[i].remote).Cmp(key) != 1
	})
	if idx < len(addrCache) && addrCache[idx].remote.Equal(remote) {
		addrCache[idx].timeout = time.Now().Add(connCacheTimeout)
		l.Debugf("addrCache: Remote %s talks through local %s", remote, addrCache[idx].local)
		return addrCache[idx].local
	}
	return nil
}

// DefaultIP figures out the IP address of the interface that has the
// default route.  It is used as a fallback IP address when we don't
// have --static-ip set and we cannot find a local -> remote mapping
// in the cache.
func DefaultIP(l logger.Logger) net.IP {
	iface, gw, err := defaultIPByRoute()
	if err != nil {
		l.Errorf("addrCache: Error getting default route: %v", err)
		return nil
	}
	if iface == nil || gw == nil {
		l.Warnf("addrCache: No default route on system.")
		return nil
	}
	addrs, err := iface.Addrs()
	if err != nil {
		l.Errorf("addrCache: Error getting addresses on %s: %v", iface.Name, err)
		return nil
	}
	for _, addr := range addrs {
		thisIP, thisNet, err := net.ParseCIDR(addr.String())
		if err == nil && thisNet.Contains(gw) {
			return thisIP
		}
	}
	return nil
}

func init() {
	go func() {
		// Garbage collection loop for the address cache.
		for {
			addrCacheMux.Lock()
			toRemove := []int{}
			deadline := time.Now()
			for idx := range addrCache {
				if addrCache[idx].timeout.Before(deadline) {
					toRemove = append(toRemove, idx)
				}
			}
			if len(toRemove) > 0 {
				lastAddr := len(addrCache)
				lastIdx := len(toRemove) - 1
				for i, idx := range toRemove {
					if idx == lastAddr-1 {
						continue
					}
					var final int
					if i != lastIdx {
						final = toRemove[i+1]
					} else {
						final = lastAddr
					}
					copy(addrCache[idx-i:final], addrCache[idx+1:final])
				}
				addrCache = addrCache[:lastAddr-len(toRemove)]
			}
			addrCacheMux.Unlock()
			time.Sleep(13 * time.Second)
		}
	}()
}

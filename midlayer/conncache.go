package midlayer

import (
	"math/big"
	"net"
	"sort"
	"sync"
	"time"
)

type cacheLine struct {
	local, remote net.IP
	unused        bool
}

func a2i(n net.IP) *big.Int {
	res := &big.Int{}
	res.SetBytes([]byte(n.To16()))
	return res
}

var addrCache = []cacheLine{}
var addrCacheMux = &sync.RWMutex{}

func addToCache(local, remote net.IP) {
	addrCacheMux.Lock()
	defer addrCacheMux.Unlock()
	key := a2i(remote)
	idx := sort.Search(len(addrCache), func(i int) bool {
		return a2i(addrCache[i].remote).Cmp(key) != 1
	})
	if idx == len(addrCache) {
		addrCache = append(addrCache, cacheLine{local, remote, false})
		return
	}
	if addrCache[idx].remote.Equal(remote) {
		addrCache[idx].local = local
		addrCache[idx].unused = false
		return
	}
	addrCache = append(addrCache, cacheLine{})
	copy(addrCache[idx+1:], addrCache[idx:])
	addrCache[idx] = cacheLine{local, remote, false}
}

// LocalFor returns the local IP address that has responded
// to TFTP or HTTP requests for the given remote IP.
func LocalFor(remote net.IP) net.IP {
	addrCacheMux.RLock()
	defer addrCacheMux.RUnlock()
	key := a2i(remote)
	idx := sort.Search(len(addrCache), func(i int) bool {
		return a2i(addrCache[i].remote).Cmp(key) != 1
	})
	if idx < len(addrCache) && addrCache[idx].remote.Equal(remote) {
		addrCache[idx].unused = false
		return addrCache[idx].local
	}
	return nil
}

// garbage-collect old addresses after 2 minutes of not being looked
// up in LocalFor
func init() {
	go func() {
		for {
			time.Sleep(time.Minute)
			addrCacheMux.Lock()
			toRemove := []int{}
			for idx := range addrCache {
				if !addrCache[idx].unused {
					toRemove = append(toRemove, idx)
				} else {
					addrCache[idx].unused = true
				}
			}
			if len(toRemove) == 0 {
				addrCacheMux.Unlock()
				continue
			}
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
			addrCacheMux.Unlock()
		}
	}()
}

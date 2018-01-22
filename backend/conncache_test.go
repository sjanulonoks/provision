package backend

import (
	"net"
	"testing"
	"time"

	"github.com/digitalrebar/logger"
)

func TestConnCache(t *testing.T) {
	l := logger.New(nil).Log("")
	// Reset timeout to three seconds
	addrCacheMux.Lock()
	connCacheTimeout = time.Second * 5
	addrCacheMux.Unlock()
	defaultIP := DefaultIP(l)
	t.Logf("Last-ditch fallback IP address: %s", defaultIP)

	t.Log("Testing ConnCache - takes a minute")

	ip1 := net.ParseIP("1.1.1.1")
	ip2 := net.ParseIP("2.2.2.2")
	ip3 := net.ParseIP("3.3.3.3")

	// Add no values
	AddToCache(l, nil, nil)
	AddToCache(l, ip1, nil)
	AddToCache(l, nil, ip2)

	addrCacheMux.Lock()
	if len(addrCache) != 0 {
		t.Errorf("Addr cache should be zero after adding incomplete values\n")
	}
	addrCacheMux.Unlock()

	AddToCache(l, ip3, ip1)
	AddToCache(l, ip2, ip1)
	AddToCache(l, ip2, ip3)

	time.Sleep(15 * time.Second)
	addrCacheMux.RLock()
	if len(addrCache) != 0 {
		t.Errorf("Cache not clean after 30 seconds (%s)", time.Now())
		for _, line := range addrCache {
			t.Errorf("CacheLine: %s", line)
		}
	} else {
		t.Logf("Cache clean.")
	}
	addrCacheMux.RUnlock()
}

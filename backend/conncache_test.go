package backend

import (
	"net"
	"testing"
	"time"
)

func TestConnCache(t *testing.T) {

	// Reset timeout to three seconds
	connCacheTimeout = time.Second * 5

	t.Log("Testing ConnCache - takes a minute")

	ip1 := net.ParseIP("1.1.1.1")
	ip2 := net.ParseIP("2.2.2.2")
	ip3 := net.ParseIP("3.3.3.3")

	// Add no values
	AddToCache(nil, nil)
	AddToCache(ip1, nil)
	AddToCache(nil, ip2)
	if len(addrCache) != 0 {
		t.Errorf("Addr cache should be zero after adding incomplete values\n")
	}

	AddToCache(ip3, ip1)
	AddToCache(ip2, ip1)
	AddToCache(ip2, ip3)

	// Wait for cache to make unused - wait for minute timer to expire
	for len(addrCache) > 0 && !addrCache[0].unused {
		time.Sleep(time.Second)
	}

	if len(addrCache) == 0 {
		t.Errorf("addrCache should not be empty!!\n")
	}

	v := LocalFor(ip1)
	if !ip2.Equal(v) {
		t.Errorf("Should have found %v, but got %v\n", ip2, v)
	}
	if addrCache[1].unused {
		t.Errorf("Cache should have been reset\n")
	}
	v = LocalFor(ip2)
	if v != nil {
		t.Errorf("Should not have found %v, and got %v\n", ip2, v)
	}

	if !addrCache[0].unused {
		t.Errorf("Second entry should be still unused\n")
	}
	AddToCache(ip2, ip3)
	if addrCache[0].unused {
		t.Errorf("Second entry should now be not unused\n")
	}

	time.Sleep(time.Second * 15)

	if len(addrCache) != 0 {
		t.Errorf("Addr cache should be zero after draining\n")
	}
}

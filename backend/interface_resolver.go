// Since most of these are not exposed as default
// routines and we want DNS config information.
//
// Borrowing from golang net and modified for our needs.
//

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// + build darwin dragonfly freebsd linux netbsd openbsd solaris

// Read system DNS config from /etc/resolv.conf

package backend

import (
	"bufio"
	"net"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

var (
	defaultNS   = []string{}
	getHostname = os.Hostname // variable for testing
)

type dnsConfig struct {
	servers    []string      // server addresses (in host:port form) to use
	search     []string      // rooted suffixes to append to local name
	domain     string        // rooted domain
	ndots      int           // number of dots in name to trigger absolute lookup
	timeout    time.Duration // wait before giving up on a query, including retries
	attempts   int           // lost packets before giving up on server
	rotate     bool          // round robin among servers
	unknownOpt bool          // anything unknown was encountered
	lookup     []string      // OpenBSD top-level database "lookup" order
	err        error         // any error that occurs during open of resolv.conf
	soffset    uint32        // used by serverOffset
}

// See resolv.conf(5) on a Linux machine.
func dnsReadConfig(filename string) *dnsConfig {
	conf := &dnsConfig{
		ndots:    1,
		timeout:  5 * time.Second,
		attempts: 2,
	}
	file, err := os.Open(filename)
	if err != nil {
		conf.servers = defaultNS
		conf.search = dnsDefaultSearch()
		conf.err = err
		return conf
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) > 0 && (line[0] == ';' || line[0] == '#') {
			// comment.
			continue
		}
		f := strings.Fields(line)
		if len(f) < 1 {
			continue
		}
		switch f[0] {
		case "nameserver": // add one name server
			if len(f) > 1 && len(conf.servers) < 3 { // small, but the standard limit
				// One more check: make sure server name is
				// just an IP address. Otherwise we need DNS
				// to look it up.
				ip := net.ParseIP(f[1])
				if ip != nil {
					if ip.To4() != nil {
						conf.servers = append(conf.servers, f[1])
					} else {
						conf.servers = append(conf.servers, f[1])
					}
				}
			}

		case "domain": // set domain
			if len(f) > 1 {
				conf.domain = ensureRooted(f[1])
			}

		case "search": // set search path to given servers
			conf.search = make([]string, len(f)-1)
			for i := 0; i < len(conf.search); i++ {
				conf.search[i] = ensureRooted(f[i+1])
			}

		case "options": // magic options
			for _, s := range f[1:] {
				switch {
				case hasPrefix(s, "ndots:"):
					n, _ := strconv.Atoi(s[6:])
					if n < 0 {
						n = 0
					} else if n > 15 {
						n = 15
					}
					conf.ndots = n
				case hasPrefix(s, "timeout:"):
					n, _ := strconv.Atoi(s[8:])
					if n < 1 {
						n = 1
					}
					conf.timeout = time.Duration(n) * time.Second
				case hasPrefix(s, "attempts:"):
					n, _ := strconv.Atoi(s[9:])
					if n < 1 {
						n = 1
					}
					conf.attempts = n
				case s == "rotate":
					conf.rotate = true
				default:
					conf.unknownOpt = true
				}
			}

		case "lookup":
			// OpenBSD option:
			// http://www.openbsd.org/cgi-bin/man.cgi/OpenBSD-current/man5/resolv.conf.5
			// "the legal space-separated values are: bind, file, yp"
			conf.lookup = f[1:]

		default:
			conf.unknownOpt = true
		}
	}
	if conf.domain != "" && len(conf.search) == 0 {
		conf.search = append(conf.search, conf.domain)
	}
	if len(conf.servers) == 0 {
		conf.servers = defaultNS
	}
	if len(conf.search) == 0 {
		conf.search = dnsDefaultSearch()
	}
	if conf.domain == "" && len(conf.search) > 0 {
		conf.domain = conf.search[0]
	}
	return conf
}

// serverOffset returns an offset that can be used to determine
// indices of servers in c.servers when making queries.
// When the rotate option is enabled, this offset increases.
// Otherwise it is always 0.
func (c *dnsConfig) serverOffset() uint32 {
	if c.rotate {
		return atomic.AddUint32(&c.soffset, 1) - 1 // return 0 to start
	}
	return 0
}

func dnsDefaultSearch() []string {
	hn, err := getHostname()
	if err != nil {
		// best effort
		return nil
	}
	if i := strings.IndexRune(hn, '.'); i >= 0 && i < len(hn)-1 {
		return []string{ensureRooted(hn[i+1:])}
	}
	return nil
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

func ensureRooted(s string) string {
	if len(s) > 0 && s[len(s)-1] == '.' {
		return s
	}
	return s + "."
}

// Package notifications provides signed, SSRF-constrained delivery attempts.
// It does not persist jobs, acknowledge outbox rows, or promise exactly-once I/O.
package notifications

import (
	"errors"
	"net/netip"
	"net/url"
	"regexp"
	"strings"
)

var hostnamePattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?)+$`)
var deniedPrefixes = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("10.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"), netip.MustParsePrefix("127.0.0.0/8"),
	netip.MustParsePrefix("169.254.0.0/16"), netip.MustParsePrefix("172.16.0.0/12"), netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("192.31.196.0/24"), netip.MustParsePrefix("192.52.193.0/24"), netip.MustParsePrefix("192.88.99.0/24"), netip.MustParsePrefix("192.168.0.0/16"), netip.MustParsePrefix("192.175.48.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"), netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("224.0.0.0/4"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("2001::/23"), netip.MustParsePrefix("2001:db8::/32"), netip.MustParsePrefix("2002::/16"), netip.MustParsePrefix("2620:4f:8000::/48"), netip.MustParsePrefix("3fff::/20"),
}

// PublicIP rejects every special-use allocation, even globally reachable special
// services. IPv6 is conservatively restricted to allocated global unicast /3.
func PublicIP(ip netip.Addr) bool {
	if !ip.IsValid() || ip.Zone() != "" || ip.Is4In6() || !ip.IsGlobalUnicast() || ip.IsPrivate() {
		return false
	}
	if ip.Is6() && !netip.MustParsePrefix("2000::/3").Contains(ip) {
		return false
	}
	for _, p := range deniedPrefixes {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}
func ValidateURL(raw string) (*url.URL, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.User != nil || u.Fragment != "" || u.RawFragment != "" || u.RawQuery != "" || u.ForceQuery || u.Opaque != "" {
		return nil, errors.New("destination policy")
	}
	host := u.Hostname()
	if host != strings.ToLower(host) || len(host) > 253 || !hostnamePattern.MatchString(host) || (u.Port() != "" && u.Port() != "443") || strings.Contains(u.Host, "%") || (u.Host != host && u.Host != host+":443") {
		return nil, errors.New("destination policy")
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return nil, errors.New("destination policy")
	}
	u.Host = host
	return u, nil
}

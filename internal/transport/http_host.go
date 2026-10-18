package transport

import (
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"strings"

	"agent-board/internal/core"
)

// ValidateExternalHost accepts one canonical lower-case DNS name with an
// explicit port. It is an authority, never a URL, wildcard, or forwarded header.
// Empty keeps the service local-only and retains the SDK's localhost guard.
func ValidateExternalHost(authority string) error {
	if authority == "" {
		return nil
	}
	host, port, ok := canonicalAuthority(authority)
	if !ok || port == "" || !strings.Contains(host, ".") || !canonicalDNS(host) {
		return core.Fail("VALIDATION", "external host requires a canonical DNS name and explicit port")
	}
	if _, err := netip.ParseAddr(host); err == nil {
		return core.Fail("VALIDATION", "external host must be a DNS name")
	}
	return nil
}

func canonicalAuthority(authority string) (host, port string, ok bool) {
	if authority == "" || strings.ContainsAny(authority, " /\\?#@%\t\r\n,") {
		return "", "", false
	}
	if strings.HasPrefix(authority, "[") && strings.HasSuffix(authority, "]") {
		host = authority[1 : len(authority)-1]
		ip, err := netip.ParseAddr(host)
		return host, "", err == nil && ip.Is6() && !ip.Is4In6() && ip.String() == host
	}
	if strings.Contains(authority, ":") {
		var err error
		host, port, err = net.SplitHostPort(authority)
		if err != nil {
			return "", "", false
		}
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 || strconv.Itoa(number) != port {
			return "", "", false
		}
		if net.JoinHostPort(host, port) != authority {
			return "", "", false
		}
	} else {
		host = authority
	}
	if ip, err := netip.ParseAddr(host); err == nil {
		return host, port, ip.String() == host && !ip.Is4In6()
	}
	return host, port, canonicalDNS(host)
}

func canonicalDNS(host string) bool {
	if len(host) < 1 || len(host) > 253 || host != strings.ToLower(host) {
		return false
	}
	labels := strings.Split(host, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return false
		}
		for _, c := range label {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	// Numeric final labels admit alternate IP spellings such as 127.1.
	for _, c := range labels[len(labels)-1] {
		if c >= 'a' && c <= 'z' {
			return true
		}
	}
	return false
}

func allowedHost(r *http.Request, external string) bool {
	// net/http rejects duplicate Host on the wire. Also reject manually built
	// requests that try to carry a second authority through Header.
	for name := range r.Header {
		if strings.EqualFold(name, "Host") {
			return false
		}
	}
	if r.URL != nil && r.URL.Host != "" && r.URL.Host != r.Host {
		return false
	}
	host, _, ok := canonicalAuthority(r.Host)
	if !ok {
		return false
	}
	if r.Host == external && external != "" || host == "localhost" {
		return true
	}
	ip, err := netip.ParseAddr(host)
	return err == nil && ip.IsLoopback()
}

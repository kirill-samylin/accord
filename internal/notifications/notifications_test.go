package notifications

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type resolverFunc func(context.Context, string, string) ([]netip.Addr, error)

func (f resolverFunc) LookupNetIP(c context.Context, n, h string) ([]netip.Addr, error) {
	return f(c, n, h)
}
func body(t *testing.T) []byte {
	t.Helper()
	b, err := Encode(Payload{DeliveryID: "fixture-1", EventCursor: 9, Event: "epic_done", TS: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), Epic: &Epic{Key: "E-1", Title: "Synthetic"}})
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func message(t *testing.T) Message {
	return Message{URL: "https://example.com/fixture", Body: body(t), Secret: []byte(strings.Repeat("s", 32))}
}
func TestPublicAddressAndURLPolicy(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "10.1.2.3", "100.100.100.100", "169.254.169.254", "172.31.1.1", "192.168.0.1", "192.0.0.9", "192.0.2.1", "198.18.1.1", "198.51.100.1", "203.0.113.1", "224.0.0.1", "255.255.255.255", "::1", "::ffff:8.8.8.8", "64:ff9b::808:808", "2001:db8::1", "2001:20::1", "2002::1", "3fff::1", "fc00::1", "fe80::1"} {
		if PublicIP(netip.MustParseAddr(s)) {
			t.Fatal("allowed special address", s)
		}
	}
	for _, s := range []string{"8.8.8.8", "2606:4700:4700::1111"} {
		if !PublicIP(netip.MustParseAddr(s)) {
			t.Fatal("public rejected", s)
		}
	}
	for _, s := range []string{"http://example.com", "https://user:pass@example.com", "https://example.com?token=x", "https://example.com#frag", "https://example.com:8443", "https://EXAMPLE.com", "https://127.0.0.1", "https://[::1]", "https://example.com.", "https://例子.com", "https://example.com?"} {
		if _, err := ValidateURL(s); err == nil {
			t.Fatal("allowed URL", s)
		}
	}
}
func TestSignedExactRequestAndRetryPolicy(t *testing.T) {
	m := message(t)
	req, err := NewRequest(context.Background(), m)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(req.Body)
	mac := hmac.New(sha256.New, m.Secret)
	mac.Write(raw)
	if req.Header.Get("X-Agent-Board-Signature") != "sha256="+hex.EncodeToString(mac.Sum(nil)) || req.Header.Get("X-Agent-Board-Delivery") != "fixture-1" {
		t.Fatal("bad signature")
	}
	for n := 1; n <= 5; n++ {
		delay, ok := RetryDelay(n, 0, 0)
		if !ok || delay != time.Minute*time.Duration(1<<(n-1)) {
			t.Fatal("retry", n)
		}
	}
	if _, ok := RetryDelay(6, 0, 0); ok {
		t.Fatal("seventh attempt allowed")
	}
	delay, _ := RetryDelay(1, .2, 2*time.Hour)
	if delay != time.Hour {
		t.Fatal("Retry-After cap")
	}
	m.Body = append(m.Body, ' ')
	second, err := NewRequest(context.Background(), m)
	if err != nil || second.Header.Get("X-Agent-Board-Signature") == req.Header.Get("X-Agent-Board-Signature") {
		t.Fatal("raw body not signed")
	}
}
func fixtureSender(t *testing.T, handler http.Handler) (*Sender, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	server := httptest.NewTLSServer(handler)
	t.Cleanup(server.Close)
	roots := x509.NewCertPool()
	roots.AddCert(server.Certificate())
	resolves, dials := new(atomic.Int32), new(atomic.Int32)
	resolver := resolverFunc(func(ctx context.Context, network, host string) ([]netip.Addr, error) {
		resolves.Add(1)
		if host != "example.com" {
			t.Error("hostname changed", host)
		}
		return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
	})
	dial := func(ctx context.Context, network, address string) (net.Conn, error) {
		dials.Add(1)
		if address != "8.8.8.8:443" {
			t.Error("not pinned", address)
		}
		return (&net.Dialer{}).DialContext(ctx, "tcp", server.Listener.Addr().String())
	}
	s := NewSender(resolver, dial)
	s.roots = roots
	return s, resolves, dials
}
func TestTLSDeliveryNoProxyNoRedirectAndReresolve(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:1")
	var calls atomic.Int32
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Host != "example.com" || r.TLS.ServerName != "example.com" || r.Header.Get("Authorization") != "" {
			t.Error("wrong identity")
		}
		if calls.Load() == 1 {
			w.Header().Set("Location", "https://example.com/redirect")
			w.WriteHeader(302)
			return
		}
		w.WriteHeader(204)
	})
	s, resolves, dials := fixtureSender(t, handler)
	first := s.Send(context.Background(), message(t))
	if first.Disposition != Dead || first.HTTPStatus != 302 {
		t.Fatal(first)
	}
	second := s.Send(context.Background(), message(t))
	if second.Disposition != Delivered {
		t.Fatal(second)
	}
	if resolves.Load() != 2 || dials.Load() != 2 || calls.Load() != 2 {
		t.Fatal("cache or redirect", resolves.Load(), dials.Load(), calls.Load())
	}
}
func TestRejectMixedDNSBeforeDialAndRebinding(t *testing.T) {
	var calls int
	resolver := resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
		calls++
		return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("10.0.0.1")}, nil
	})
	s := NewSender(resolver, func(context.Context, string, string) (net.Conn, error) {
		t.Error("dialed private/mixed response")
		return nil, io.EOF
	})
	result := s.Send(context.Background(), message(t))
	if result.Disposition != Dead || result.Code != "SSRF_POLICY" || calls != 1 {
		t.Fatal(result, calls)
	}
	s, resolves, _ := fixtureSender(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	if s.Send(context.Background(), message(t)).Disposition != Delivered {
		t.Fatal("first attempt")
	}
	s.resolver = resolver
	if s.Send(context.Background(), message(t)).Disposition != Dead || resolves.Load() != 1 {
		t.Fatal("rebind reused connection")
	}
}
func TestRetryStatusAndCertificateValidation(t *testing.T) {
	for _, status := range []int{408, 429, 500, 503, 400, 401} {
		s, _, _ := fixtureSender(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "7200")
			w.WriteHeader(status)
		}))
		result := s.Send(context.Background(), message(t))
		expected := Retry
		if status == 400 || status == 401 {
			expected = Dead
		}
		if result.Disposition != expected {
			t.Fatal(status, result)
		}
		if expected == Retry && result.RetryAfter != time.Hour {
			t.Fatal("retry cap")
		}
	}
	s, _, _ := fixtureSender(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("untrusted TLS reached handler") }))
	s.roots = nil
	if s.Send(context.Background(), message(t)).Disposition != Retry {
		t.Fatal("untrusted TLS accepted")
	}
}

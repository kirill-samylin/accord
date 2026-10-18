package notifications

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
	"strconv"
	"time"
)

type Resolver interface {
	LookupNetIP(context.Context, string, string) ([]netip.Addr, error)
}
type DialFunc func(context.Context, string, string) (net.Conn, error)
type Sender struct {
	resolver Resolver
	dial     DialFunc
	roots    *x509.CertPool
}

// NewSender accepts transport seams for synthetic tests; they do not disable IP
// policy. Production callers use nil arguments for the system resolver/dialer.
func NewSender(resolver Resolver, dial DialFunc) *Sender {
	if resolver == nil {
		resolver = net.DefaultResolver
	}
	if dial == nil {
		dial = (&net.Dialer{Timeout: 2 * time.Second}).DialContext
	}
	return &Sender{resolver: resolver, dial: dial}
}

type Disposition string

const (
	Delivered Disposition = "delivered"
	Retry     Disposition = "retry"
	Dead      Disposition = "dead"
)

type Result struct {
	Disposition Disposition
	Code        string
	HTTPStatus  int
	RetryAfter  time.Duration
}

var errPolicy = errors.New("destination policy")

func (s *Sender) Send(ctx context.Context, m Message) Result {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := NewRequest(ctx, m)
	if err != nil {
		return Result{Disposition: Dead, Code: "INVALID_MESSAGE"}
	}
	host := req.URL.Hostname()
	transport := &http.Transport{
		Proxy: nil, DisableKeepAlives: true, DisableCompression: true, ForceAttemptHTTP2: false,
		MaxResponseHeaderBytes: 16 << 10, ResponseHeaderTimeout: 2 * time.Second, TLSHandshakeTimeout: 2 * time.Second,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host, RootCAs: s.roots},
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			ips, err := s.resolver.LookupNetIP(ctx, "ip", host)
			if err != nil {
				return nil, err
			}
			if len(ips) == 0 || len(ips) > 64 {
				return nil, errPolicy
			}
			for _, ip := range ips {
				if !PublicIP(ip) {
					return nil, errPolicy
				}
			}
			// No fallback hostname lookup is allowed after validation. A selected address
			// is pinned even if DNS changes between the lookup and TCP connection.
			return s.dial(ctx, "tcp", net.JoinHostPort(ips[0].String(), "443"))
		},
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport, Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(req)
	if err != nil {
		if errors.Is(err, errPolicy) {
			return Result{Disposition: Dead, Code: "SSRF_POLICY"}
		}
		return Result{Disposition: Retry, Code: "NETWORK"}
	}
	defer response.Body.Close()
	// Never persist/log response bodies. Closing after a bounded discard avoids
	// unbounded allocations and keep-alive reuse even for hostile recipients.
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4<<10))
	result := Result{HTTPStatus: response.StatusCode}
	switch {
	case response.StatusCode >= 200 && response.StatusCode < 300:
		result.Disposition = Delivered
		result.Code = "HTTP_2XX"
	case response.StatusCode == 408 || response.StatusCode == 429 || response.StatusCode >= 500:
		result.Disposition = Retry
		result.Code = "HTTP_RETRY"
		result.RetryAfter = parseRetryAfter(response.Header.Get("Retry-After"), time.Now())
	default:
		result.Disposition = Dead
		result.Code = "HTTP_TERMINAL"
	}
	return result
}
func parseRetryAfter(value string, now time.Time) time.Duration {
	var delay time.Duration
	if n, err := strconv.ParseInt(value, 10, 64); err == nil {
		if n > 3600 {
			return time.Hour
		}
		if n > 0 {
			delay = time.Duration(n) * time.Second
		}
	} else if at, err := http.ParseTime(value); err == nil {
		delay = at.Sub(now)
	}
	if delay < 0 {
		return 0
	}
	if delay > time.Hour {
		return time.Hour
	}
	return delay
}

// RetryDelay returns false after the sixth total attempt. Jitter is sampled by
// the caller and MUST be persisted with next_at, not resampled on restart.
func RetryDelay(attempt int, jitter float64, retryAfter time.Duration) (time.Duration, bool) {
	if attempt < 1 || attempt >= 6 || jitter < 0 || jitter > 0.2 || math.IsNaN(jitter) || math.IsInf(jitter, 0) {
		return 0, false
	}
	delay := time.Duration(float64(time.Minute*time.Duration(1<<(attempt-1))) * (1 + jitter))
	if retryAfter > time.Hour {
		retryAfter = time.Hour
	}
	if retryAfter > delay {
		delay = retryAfter
	}
	return delay, true
}

// Package client owns the native clients' endpoint, credential and redirect policy.
package client

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"agent-board/internal/transport"
)

// MaxResponseBytes bounds both native JSON and MCP response bodies.
const MaxResponseBytes = 1 << 20

// Origin accepts only a credential-free HTTPS origin or literal-loopback HTTP.
func Origin(raw string) (*url.URL, error) {
	base, err := url.Parse(raw)
	if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.ForceQuery || base.Fragment != "" || base.Opaque != "" || base.Path != "" && base.Path != "/" {
		return nil, errors.New("server must be an HTTP(S) origin without credentials")
	}
	if base.Scheme != "https" {
		if base.Scheme != "http" || transport.ValidateListen(base.Host) != nil {
			return nil, errors.New("HTTP requires a literal loopback address; remote servers require HTTPS")
		}
	}
	base.Path = ""
	return base, nil
}

// ReadToken loads a bounded private regular file or AGENT_BOARD_TOKEN. Never log it.
func ReadToken(path string) (string, error) {
	token := os.Getenv("AGENT_BOARD_TOKEN")
	if path != "" {
		info, err := os.Lstat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0177 != 0 {
			return "", errors.New("token file must be a private regular file (mode 0600)")
		}
		file, err := os.Open(path)
		if err != nil {
			return "", errors.New("cannot read private token file")
		}
		data, err := io.ReadAll(io.LimitReader(file, 4097))
		closeErr := file.Close()
		if err != nil || closeErr != nil || len(data) > 4096 {
			return "", errors.New("cannot read private token file")
		}
		token = strings.TrimSpace(string(data))
	}
	if token == "" || len(token) > 4096 || strings.ContainsAny(token, "\r\n \t") {
		return "", errors.New("set AGENT_BOARD_TOKEN or provide --token-file")
	}
	return token, nil
}

// HTTP creates a client that never follows redirects or sends credentials off-origin.
// Context deadlines are also applied by callers for lease-sensitive operations.
func HTTP(origin *url.URL, token string, timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout:       timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirects are disabled") },
		Transport:     &bearerTransport{origin: *origin, token: token, next: http.DefaultTransport},
	}
}

type bearerTransport struct {
	origin url.URL
	token  string
	next   http.RoundTripper
}

func (t *bearerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme != t.origin.Scheme || req.URL.Host != t.origin.Host || req.URL.User != nil {
		return nil, errors.New("credential origin mismatch")
	}
	copied := req.Clone(req.Context())
	copied.Header.Set("Authorization", "Bearer "+t.token)
	response, err := t.next.RoundTrip(copied)
	if err != nil {
		return nil, err
	}
	if response.ContentLength > MaxResponseBytes {
		response.Body.Close()
		return nil, errors.New("response exceeds limit")
	}
	response.Body = &boundedBody{ReadCloser: response.Body, remaining: MaxResponseBytes}
	return response, nil
}

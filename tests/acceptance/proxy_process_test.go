package acceptance

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const externalAuthority = "accord.synthetic.example:8443"

// The child is the ordinary CLI produced by TestMain; -race covers this harness.
// A real local TLS proxy forwards its configured authority over an actual socket.
func TestProxyProcessConfiguredHost(t *testing.T) {
	b := startProxyBoard(t, "", "--external-host", externalAuthority)
	c := newClient(t, b, "proxy-worker")
	secret, err := os.ReadFile(c.token)
	if err != nil {
		t.Fatal(err)
	}
	target, err := url.Parse(b.url)
	if err != nil {
		t.Fatal(err)
	}
	proxy := httptest.NewTLSServer(&httputil.ReverseProxy{Rewrite: func(r *httputil.ProxyRequest) {
		r.SetURL(target)
		r.Out.Host = externalAuthority
		// Spoofed metadata cannot replace the configured Host decision.
		r.Out.Header.Set("X-Forwarded-Host", "attacker.example:443")
		r.Out.Header.Set("X-Forwarded-Proto", "http")
	}})
	t.Cleanup(proxy.Close)
	httpClient := proxy.Client() // Trust only this synthetic TLS server's certificate.
	httpClient.Timeout = 5 * time.Second
	httpClient.Transport = proxyBearer{httpClient.Transport, strings.TrimSpace(string(secret))}
	session := connectProxySDK(t, proxy.URL, httpClient)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tools, err := session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 11 {
		t.Fatal("TLS proxy MCP tool discovery failed")
	}
	invoke := func(name string, input payload) *mcp.CallToolResult {
		t.Helper()
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: input})
		if err != nil {
			t.Fatalf("TLS proxy MCP %s failed (response suppressed)", name)
		}
		return result
	}
	view := mcpValue[core.SessionView](t, invoke("describe", payload{"request_id": "proxy-open", "role": "coordinator", "run_id": "proxy", "machine": "proxy-worker"}))
	if view.SessionID == "" || view.Role != "coordinator" {
		t.Fatal("proxy describe lost session identity")
	}
	epic := mcpValue[core.Task](t, invoke("create", payload{"session_id": view.SessionID, "request_id": "proxy-create", "type": "epic", "title": "Synthetic proxy acceptance", "description": "Actual CLI behind local TLS proxy"}))
	got := mcpValue[core.Task](t, invoke("get", payload{"session_id": view.SessionID, "task_key": epic.Key}))
	if epic.Key == "" || got.Key != epic.Key || got.Title != epic.Title {
		t.Fatal("proxy MCP create/get did not preserve stored task")
	}
	// The same configured child still accepts the ordinary loopback SDK client.
	connectProxySDK(t, b.url, &http.Client{Timeout: 5 * time.Second, Transport: proxyBearer{http.DefaultTransport, strings.TrimSpace(string(secret))}})
}

type proxyBearer struct {
	base  http.RoundTripper
	token string
}

func (p proxyBearer) RoundTrip(r *http.Request) (*http.Response, error) {
	clone := r.Clone(r.Context())
	clone.Header = r.Header.Clone()
	clone.Header.Set("Authorization", "Bearer "+p.token)
	return p.base.RoundTrip(clone)
}

func connectProxySDK(t *testing.T, origin string, client *http.Client) *mcp.ClientSession {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	consumer := mcp.NewClient(&mcp.Implementation{Name: "synthetic-proxy-process", Version: "1"}, nil)
	session, err := consumer.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: origin + "/mcp", HTTPClient: client, DisableStandaloneSSE: true, MaxRetries: -1}, nil)
	if err != nil {
		t.Fatal("actual socket SDK initialize failed (response suppressed)")
	}
	if session.InitializeResult().ProtocolVersion == "" {
		t.Fatal("MCP initialize omitted protocol version")
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func TestProxyProcessHostSecurity(t *testing.T) {
	b := startProxyBoard(t, "", "--external-host", externalAuthority)
	c := newClient(t, b, "security-worker")
	secret, err := os.ReadFile(c.token)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name, path, host string
		auth, origin     bool
		want             int
	}{
		{"foreign-mcp", "/mcp", "attacker.example:8443", true, false, 403},
		{"wrong-port", "/mcp", "accord.synthetic.example:443", true, false, 403},
		{"missing-port", "/mcp", "accord.synthetic.example", true, false, 403},
		{"foreign-health", "/healthz", "attacker.example:8443", false, false, 403},
		{"foreign-api", "/api/v1/describe", "attacker.example:8443", true, false, 403},
		{"foreign-metrics", "/metrics", "attacker.example:8443", true, false, 403},
		{"valid-without-bearer", "/mcp", externalAuthority, false, false, 401},
		{"valid-origin", "/mcp", externalAuthority, true, true, 403},
		{"metrics-without-bearer", "/metrics", externalAuthority, false, false, 401},
		{"valid-health", "/healthz", externalAuthority, false, false, 200},
		{"valid-metrics", "/metrics", externalAuthority, true, false, 200},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, b.url+tc.path, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Host = tc.host
			// An attacker supplies the allowed host through both common spellings.
			req.Header.Set("X-Forwarded-Host", externalAuthority)
			req.Header.Set("Forwarded", "host="+externalAuthority+";proto=https")
			req.Header.Set("X-Forwarded-Proto", "https")
			if tc.auth {
				req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(string(secret)))
			}
			if tc.origin {
				req.Header["Origin"] = []string{""} // Present but empty is still denied.
			}
			response, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
			if err != nil {
				t.Fatal("actual socket security request failed")
			}
			defer response.Body.Close()
			if response.StatusCode != tc.want {
				t.Fatalf("status=%d want=%d (body suppressed)", response.StatusCode, tc.want)
			}
		})
	}
}

func TestProxyProcessCLIConfiguration(t *testing.T) {
	for _, tc := range []struct{ name, env, flag string }{
		{"flag-url", "", "https://accord.synthetic.example:8443"},
		{"flag-wildcard", "", "*.synthetic.example:8443"},
		{"flag-portless", "", "accord.synthetic.example"},
		{"env-invalid", "accord.synthetic.example:08443", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			db := filepath.Join(dir, "must-not-exist.db")
			args := []string{"serve", "--db", db, "--listen", "127.0.0.1:0"}
			if tc.flag != "" {
				args = append(args, "--external-host", tc.flag)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, binary, args...)
			cmd.Env = proxyEnvironment(tc.env)
			out, err := cmd.Output()
			if err == nil || ctx.Err() != nil || len(out) != 0 {
				t.Fatal("invalid external host did not fail before server startup")
			}
			if _, err := os.Stat(db); !os.IsNotExist(err) {
				t.Fatal("invalid external host created database")
			}
		})
	}
	for _, tc := range []struct {
		name, env string
		flags     []string
	}{
		{"env-fallback", externalAuthority, nil},
		{"explicit-overrides-env", "invalid", []string{"--external-host", externalAuthority}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			b := startProxyBoard(t, tc.env, tc.flags...)
			req, err := http.NewRequest(http.MethodGet, b.url+"/healthz", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Host = externalAuthority
			response, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
			if err != nil {
				t.Fatal("configured external authority unreachable")
			}
			response.Body.Close()
			if response.StatusCode != http.StatusOK {
				t.Fatalf("external authority status=%d", response.StatusCode)
			}
		})
	}
}

// A local-only server and a separate CLI stdio worker retain their old path.
func TestProxyProcessLocalWorkerCompatibility(t *testing.T) {
	b := startProxyBoard(t, "")
	c := newClient(t, b, "local-worker")
	worker := startMCP(t, c, b.url)
	view := mcpValue[core.SessionView](t, worker.call("describe", payload{"request_id": "local-open", "role": "developer", "run_id": "local", "machine": "local-worker"}))
	if view.SessionID == "" || view.Role != "developer" {
		t.Fatal("local stdio worker did not open the requested board session")
	}
	page := mcpValue[core.EventsPage](t, worker.call("events", payload{"since_cursor": 0, "limit": 1}))
	if page.HighWater < 1 {
		t.Fatal("local stdio worker could not read server events")
	}
	worker.stop()
	worker.assertProtocol(c.token)
	req, err := http.NewRequest(http.MethodGet, b.url+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = externalAuthority
	response, err := (&http.Client{Timeout: 3 * time.Second}).Do(req)
	if err != nil {
		t.Fatal("local-only host probe failed")
	}
	response.Body.Close()
	if response.StatusCode != http.StatusForbidden {
		t.Fatal("local-only server accepted external authority without configuration")
	}
}

func proxyEnvironment(external string) []string {
	var env []string
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "ACCORD_EXTERNAL_HOST=") && !strings.HasPrefix(entry, "AGENT_BOARD_TOKEN=") {
			env = append(env, entry)
		}
	}
	return append(env, "ACCORD_EXTERNAL_HOST="+external)
}

// Reuses board lifecycle/cleanup without changing the existing acceptance helper.
func startProxyBoard(t *testing.T, external string, extra ...string) *board {
	t.Helper()
	b := newBoard(t, "1m")
	args := append([]string{"serve", "--db", b.db, "--listen", "127.0.0.1:0"}, extra...)
	b.cmd = exec.Command(binary, args...)
	b.cmd.Env = proxyEnvironment(external)
	out, err := b.cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	b.cmd.Stderr = io.Discard
	if err = b.cmd.Start(); err != nil {
		t.Fatal("configured server child failed to start")
	}
	b.exited = make(chan error, 1)
	go func() { b.exited <- b.cmd.Wait() }()
	address := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(out)
		for scanner.Scan() {
			if value, ok := strings.CutPrefix(scanner.Text(), "agent-board listening on "); ok {
				select {
				case address <- value:
				default:
				}
			}
		}
	}()
	select {
	case b.url = <-address:
	case <-time.After(5 * time.Second):
		t.Fatal("configured server omitted listening address")
	}
	return b
}

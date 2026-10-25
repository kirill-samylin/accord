package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"agent-board/internal/core"
	"agent-board/internal/store"
	"agent-board/internal/transport"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type synchronizedBuffer struct {
	mu   sync.Mutex
	data bytes.Buffer
}

func (b *synchronizedBuffer) Write(data []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.data.Write(data)
}
func (b *synchronizedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.data.String()
}

type fixture struct {
	t      *testing.T
	store  *store.Store
	token  core.TokenView
	path   string
	http   *httptest.Server
	fault  atomic.Int32
	calls  atomic.Int64
	stderr synchronizedBuffer
	stdout synchronizedBuffer
	client *mcp.ClientSession
	done   chan error
}

func setup(t *testing.T, interval time.Duration) *fixture {
	return setupConfigured(t, interval, core.Config{SessionTTL: time.Second, LeaseTTL: time.Second})
}
func setupConfigured(t *testing.T, interval time.Duration, cfg core.Config) *fixture {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	s, err := store.Open(filepath.Join(dir, "board.db"), cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	token, err := s.CreateToken(context.Background(), core.TokenInput{Owner: "synthetic", Machine: "test-machine", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "test.token")
	if err = os.WriteFile(path, []byte(token.Secret), 0600); err != nil {
		t.Fatal(err)
	}
	f := &fixture{t: t, store: s, token: token, path: path, done: make(chan error, 1)}
	authoritative := transport.NewHandler(s, s.Ready)
	f.http = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			return
		}
		r.Body = io.NopCloser(bytes.NewReader(data))
		var envelope struct {
			Method string `json:"method"`
			Params struct {
				Name string `json:"name"`
			} `json:"params"`
		}
		if json.Unmarshal(data, &envelope) == nil && envelope.Method == "tools/call" {
			f.calls.Add(1)
		}
		switch f.fault.Load() {
		case 1:
			http.Error(w, "synthetic network fault", 503)
			return
		case 2:
			select {
			case <-r.Context().Done():
			case <-time.After(500 * time.Millisecond):
			}
			return
		}
		authoritative.ServeHTTP(w, r)
	}))
	ctx, cancel := context.WithCancel(context.Background())
	clientReader, bridgeWriter := io.Pipe()
	bridgeReader, clientWriter := io.Pipe()
	clientTransport := &mcp.IOTransport{Reader: clientReader, Writer: clientWriter}
	bridgeTransport := StdioTransport(bridgeReader, capturingWriter{Writer: io.MultiWriter(&f.stdout, bridgeWriter), Closer: bridgeWriter}, 64<<10)
	go func() {
		f.done <- Serve(ctx, Options{Server: f.http.URL, TokenFile: path, HeartbeatInterval: interval, Stderr: &f.stderr}, bridgeTransport)
	}()
	consumer := mcp.NewClient(&mcp.Implementation{Name: "bridge-integration", Version: "1"}, nil)
	connecting, stop := context.WithTimeout(ctx, 10*time.Second)
	defer stop()
	f.client, err = consumer.Connect(connecting, clientTransport, nil)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	t.Cleanup(func() { cancel(); f.client.Close(); f.http.Close(); s.Close() })
	return f
}
func (f *fixture) invoke(name string, input any) *mcp.CallToolResult {
	f.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := f.client.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: input})
	if err != nil {
		f.t.Fatalf("%s MCP: %v", name, err)
	}
	return result
}
func value[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()
	if result.IsError {
		t.Fatalf("unexpected tool error: %v", result.Content)
	}
	raw, err := resultJSON(result)
	if err != nil {
		t.Fatal(err)
	}
	var out T
	if err = json.Unmarshal(raw, &out); err != nil {
		t.Fatal(err)
	}
	return out
}
func (f *fixture) describe() core.SessionView {
	f.t.Helper()
	return value[core.SessionView](f.t, f.invoke("describe", core.OpenSession{RequestID: "bridge-open", Role: "developer", RunID: "synthetic-run", Machine: "test-machine"}))
}
func (f *fixture) admin() core.Actor {
	f.t.Helper()
	ctx := context.Background()
	p, err := f.store.Authenticate(ctx, f.token.Secret)
	if err != nil {
		f.t.Fatal(err)
	}
	v, err := f.store.Describe(ctx, p, core.OpenSession{RequestID: fmtID(), Role: "coordinator", RunID: "observer", Machine: "test-machine"})
	if err != nil {
		f.t.Fatal(err)
	}
	return core.Actor{Principal: p, SessionID: v.SessionID}
}

var sequence atomic.Int64

func fmtID() string {
	return time.Now().Format("150405.000000000") + "-" + strconv.FormatInt(sequence.Add(1), 10)
}
func (f *fixture) task() core.Task {
	f.t.Helper()
	a := f.admin()
	epic, err := f.store.Create(context.Background(), a, core.CreateInput{RequestID: fmtID(), Type: "epic", Title: "synthetic epic"})
	if err != nil {
		f.t.Fatal(err)
	}
	task, err := f.store.Create(context.Background(), a, core.CreateInput{RequestID: fmtID(), Type: "task", EpicKey: epic.Key, Title: "synthetic task", Description: "isolated bridge test", Role: "developer"})
	if err != nil {
		f.t.Fatal(err)
	}
	return task
}
func (f *fixture) get(key string) core.Task {
	f.t.Helper()
	task, err := f.store.Get(context.Background(), f.admin(), core.GetInput{TaskKey: key})
	if err != nil {
		f.t.Fatal(err)
	}
	return task
}
func (f *fixture) waitQuarantine() {
	f.t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if f.stderr.String() != "" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	f.t.Fatal("bridge did not report ownership loss")
}

var _ io.Writer = (*synchronizedBuffer)(nil)

// Preserve the IO pipe close contract while capturing stdout protocol frames.
type capturingWriter struct {
	io.Writer
	io.Closer
}

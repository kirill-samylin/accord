package acceptance

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type lockedOutput struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (b *lockedOutput) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b.Write(p)
}
func (b *lockedOutput) data() []byte {
	b.mu.Lock()
	defer b.mu.Unlock()
	return bytes.Clone(b.b.Bytes())
}

type capturedReader struct {
	io.Reader
	io.Closer
}

type mcpProcess struct {
	t       *testing.T
	cmd     *exec.Cmd
	session *mcp.ClientSession
	done    chan error
	stdout  lockedOutput
	stderr  lockedOutput
	stopped bool
}

// A real CLI child exchanges MCP over OS pipes. Only this harness's PIDs are reaped.
func startMCP(t *testing.T, c *client, origin string) *mcpProcess {
	t.Helper()
	p := &mcpProcess{t: t, done: make(chan error, 1)}
	p.cmd = exec.Command(binary, "mcp", "--server", origin, "--token-file", c.token)
	for _, entry := range os.Environ() {
		if !strings.HasPrefix(entry, "AGENT_BOARD_TOKEN=") {
			p.cmd.Env = append(p.cmd.Env, entry)
		}
	}
	p.cmd.Stderr = &p.stderr
	out, err := p.cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	in, err := p.cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = p.cmd.Start(); err != nil {
		t.Fatal("stdio bridge process failed to start")
	}
	t.Cleanup(p.stop)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	consumer := mcp.NewClient(&mcp.Implementation{Name: "synthetic-process-acceptance", Version: "1"}, nil)
	p.session, err = consumer.Connect(ctx, &mcp.IOTransport{Reader: capturedReader{io.TeeReader(out, &p.stdout), out}, Writer: in}, nil)
	if err != nil {
		_ = in.Close()
		t.Fatal("stdio MCP handshake failed (output suppressed)")
	}
	return p
}

func (p *mcpProcess) stop() {
	p.t.Helper()
	if p.stopped {
		return
	}
	p.stopped = true
	if p.session != nil {
		_ = p.session.Close() // EOF: the bridge must stop its pulse and exit.
	}
	go func() { p.done <- p.cmd.Wait() }()
	select {
	case err := <-p.done:
		if err != nil {
			p.t.Error("stdio bridge did not exit cleanly after EOF")
		}
	case <-time.After(7 * time.Second):
		_ = p.cmd.Process.Kill()
		<-p.done
		p.t.Error("owned stdio bridge exceeded shutdown deadline")
	}
}

func (p *mcpProcess) call(name string, input payload) *mcp.CallToolResult {
	p.t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result, err := p.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: input})
	if err != nil {
		p.t.Fatalf("stdio %s protocol call failed (output suppressed)", name)
	}
	return result
}

func mcpJSON(t *testing.T, result *mcp.CallToolResult) []byte {
	t.Helper()
	if result.StructuredContent != nil {
		data, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal("cannot decode structured MCP result")
		}
		return data
	}
	if len(result.Content) == 1 {
		if content, ok := result.Content[0].(*mcp.TextContent); ok && json.Valid([]byte(content.Text)) {
			return []byte(content.Text)
		}
	}
	t.Fatal("MCP result has no JSON payload (output suppressed)")
	return nil
}

func mcpValue[T any](t *testing.T, result *mcp.CallToolResult) T {
	t.Helper()
	if result.IsError {
		t.Fatal("stdio tool returned an unexpected error (output suppressed)")
	}
	return decode[T](t, mcpJSON(t, result))
}

func (p *mcpProcess) uncertain(result *mcp.CallToolResult) {
	p.t.Helper()
	response := decode[struct {
		Code string `json:"code"`
	}](p.t, mcpJSON(p.t, result))
	if !result.IsError || response.Code != "LEASE_UNCERTAIN" {
		p.t.Fatal("stdio bridge did not report permanent lease quarantine")
	}
}

func (p *mcpProcess) assertProtocol(token string) {
	p.t.Helper()
	secret, err := os.ReadFile(token)
	if err != nil {
		p.t.Fatal(err)
	}
	if bytes.Contains(p.stdout.data(), bytes.TrimSpace(secret)) || bytes.Contains(p.stderr.data(), bytes.TrimSpace(secret)) {
		p.t.Fatal("stdio bridge disclosed synthetic bearer")
	}
	scanner := bufio.NewScanner(bytes.NewReader(p.stdout.data()))
	frames := 0
	for scanner.Scan() {
		var frame struct {
			Version string `json:"jsonrpc"`
		}
		if json.Unmarshal(scanner.Bytes(), &frame) != nil || frame.Version != "2.0" {
			p.t.Fatal("bridge stdout contains non-protocol output")
		}
		frames++
	}
	if scanner.Err() != nil || frames < 3 {
		p.t.Fatal("stdio protocol capture incomplete")
	}
}

package acceptance

import (
	"bufio"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

var binary string

// TestMain builds the actual CLI once. It never logs child request bodies or tokens.
func TestMain(m *testing.M) {
	_, source, _, ok := runtime.Caller(0)
	if !ok {
		os.Exit(1)
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(source), "../.."))
	dir, err := os.MkdirTemp("", "agent-board-acceptance-")
	if err != nil {
		os.Exit(1)
	}
	binary = filepath.Join(dir, "agent-board")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	cmd := exec.CommandContext(ctx, filepath.Join(runtime.GOROOT(), "bin", "go"), "build", "-o", binary, "./cmd/agent-board")
	cmd.Dir = root
	err = cmd.Run()
	cancel()
	if err != nil {
		fmt.Fprintln(os.Stderr, "acceptance binary build failed")
		os.RemoveAll(dir)
		os.Exit(1)
	}
	data, err := os.ReadFile(binary)
	if err != nil {
		os.RemoveAll(dir)
		os.Exit(1)
	}
	fmt.Printf("acceptance binary sha256=%x runtime=%s os=%s arch=%s\n", sha256.Sum256(data), runtime.Version(), runtime.GOOS, runtime.GOARCH)
	if evidence := os.Getenv("ACCEPTANCE_EVIDENCE_DIR"); evidence != "" {
		if err = os.MkdirAll(evidence, 0700); err == nil {
			err = os.WriteFile(filepath.Join(evidence, "agent-board"), data, 0700)
		}
		if err != nil {
			fmt.Fprintln(os.Stderr, "cannot preserve acceptance binary")
			os.RemoveAll(dir)
			os.Exit(1)
		}
	}
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

type board struct {
	t      *testing.T
	db     string
	url    string
	ttl    string
	cmd    *exec.Cmd
	exited chan error
}

func newBoard(t *testing.T, ttl string) *board {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	b := &board{t: t, db: filepath.Join(dir, "board.db"), ttl: ttl}
	t.Cleanup(b.stop)
	return b
}

func (b *board) start() {
	b.t.Helper()
	b.cmd = exec.Command(binary, "serve", "--db", b.db, "--listen", "127.0.0.1:0", "--task-ttl", b.ttl, "--session-ttl", "1m")
	stdout, err := b.cmd.StdoutPipe()
	if err != nil {
		b.t.Fatal(err)
	}
	// No child output is forwarded to avoid accidental credential disclosure.
	b.cmd.Stderr = io.Discard
	if err = b.cmd.Start(); err != nil {
		b.t.Fatal("server start failed")
	}
	address := make(chan string, 1)
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if value, ok := strings.CutPrefix(scanner.Text(), "agent-board listening on "); ok {
				select {
				case address <- value:
				default:
				}
			}
		}
	}()
	b.exited = make(chan error, 1)
	go func() { b.exited <- b.cmd.Wait() }()
	select {
	case b.url = <-address:
	case <-time.After(10 * time.Second):
		b.t.Fatal("no server address within startup deadline")
	}
	for _, route := range []string{"/healthz", "/readyz"} {
		response, err := (&http.Client{Timeout: 3 * time.Second}).Get(b.url + route)
		if err != nil {
			b.t.Fatalf("server readiness %s failed", route)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			b.t.Fatalf("server readiness %s status=%d", route, response.StatusCode)
		}
	}
}

func (b *board) stop() {
	b.t.Helper()
	if b.cmd == nil {
		return
	}
	_ = b.cmd.Process.Signal(syscall.SIGTERM)
	select {
	case err := <-b.exited:
		if err != nil {
			b.t.Error("server did not stop gracefully")
		}
	case <-time.After(7 * time.Second):
		_ = b.cmd.Process.Kill()
		<-b.exited
		b.t.Error("server shutdown deadline exceeded; owned process killed")
	}
	b.cmd = nil
}

// waitExpired uses the server's HTTP Date, never an injected client clock.
func (b *board) waitExpired(expires time.Time) {
	b.t.Helper()
	deadline := time.NewTimer(6 * time.Second)
	defer deadline.Stop()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
	client := &http.Client{Timeout: time.Second}
	for {
		select {
		case <-deadline.C:
			b.t.Fatal("server clock did not pass lease expiry")
		case <-ticker.C:
			response, err := client.Get(b.url + "/healthz")
			if err != nil {
				b.t.Fatal("health request while awaiting expiry failed")
			}
			response.Body.Close()
			serverTime, err := http.ParseTime(response.Header.Get("Date"))
			if err != nil {
				b.t.Fatal("missing server Date")
			}
			if !serverTime.Before(expires) {
				return
			}
		}
	}
}

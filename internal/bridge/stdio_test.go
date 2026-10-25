package bridge

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type frame struct {
	ID    json.RawMessage `json:"id"`
	Error *struct {
		Code    int64  `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Result json.RawMessage `json:"result"`
}

type stdioHarness struct {
	send   func(string)
	expect func(string) frame
	done   chan error
}

func startStdio(t *testing.T) stdioHarness {
	t.Helper()
	serverReads, clientWrites := io.Pipe()
	clientReads, serverWrites := io.Pipe()
	t.Cleanup(func() { clientWrites.Close() })
	server := mcp.NewServer(&mcp.Implementation{Name: "stdio-test", Version: "1"}, nil)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx, StdioTransport(serverReads, serverWrites, 1024)) }()
	frames := make(chan frame, 16)
	go func() {
		scanner := bufio.NewScanner(clientReads)
		for scanner.Scan() {
			var f frame
			if json.Unmarshal(scanner.Bytes(), &f) == nil {
				frames <- f
			}
		}
		close(frames)
	}()
	h := stdioHarness{done: done}
	h.send = func(line string) {
		if _, err := io.WriteString(clientWrites, line+"\n"); err != nil {
			t.Fatal(err)
		}
	}
	h.expect = func(id string) frame {
		t.Helper()
		select {
		case f, ok := <-frames:
			if !ok {
				t.Fatalf("bridge stopped before replying to %s: %v", id, <-done)
			}
			if string(f.ID) != id {
				t.Fatalf("reply id %s, want %s", f.ID, id)
			}
			return f
		case <-ctx.Done():
			t.Fatalf("no reply for %s", id)
		}
		return frame{}
	}
	h.send(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`)
	if f := h.expect("1"); f.Error != nil {
		t.Fatalf("initialize: %+v", f.Error)
	}
	h.send(`{"jsonrpc":"2.0","method":"notifications/initialized"}`)
	return h
}

func (h stdioHarness) stillServing(t *testing.T) {
	t.Helper()
	h.send(`{"jsonrpc":"2.0","id":3,"method":"ping"}`)
	if f := h.expect("3"); f.Error != nil || f.Result == nil {
		t.Fatalf("bridge stopped serving after bad lines: %+v", f.Error)
	}
	select {
	case err := <-h.done:
		t.Fatalf("server exited: %v", err)
	default:
	}
}

// An oversized or malformed line must be answered and survived: exiting would
// silently stop heartbeats for a lease the native client still believes held.
func TestStdioOversizedAndMalformedLinesKeepServing(t *testing.T) {
	h := startStdio(t)
	huge := strings.Repeat("x", 200_000)
	h.send(`{"jsonrpc":"2.0","id":"big-1","method":"tools/call","params":{"name":"events","arguments":{"text":"` + huge + `"}}}`)
	if f := h.expect(`"big-1"`); f.Error == nil || f.Error.Code != -32600 || !strings.Contains(f.Error.Message, "1024") {
		t.Fatalf("oversized line reply: %+v", f.Error)
	}
	// The streaming scanner also recovers an id that follows the huge member.
	h.send(`{"jsonrpc":"2.0","method":"ping","params":{"pad":"` + huge + `"},"id":9}`)
	if f := h.expect("9"); f.Error == nil {
		t.Fatal("trailing id oversized line accepted")
	}
	h.send(`{"jsonrpc":"2.0","id":2,"method":"ping"} trailing`)
	if f := h.expect("null"); f.Error == nil || f.Error.Code != -32700 {
		t.Fatalf("malformed line reply: %+v", f.Error)
	}
	h.send("")
	h.stillServing(t)
}

// Valid JSON that is not one JSON-RPC message (an object without the 2.0
// envelope, a scalar, a batch) also ended the SDK connection before.
func TestStdioValidJSONNonMessagesKeepServing(t *testing.T) {
	h := startStdio(t)
	for line, id := range map[string]string{
		`{"foo":1}`: "null",
		`42`:        "null",
		`"ping"`:    "null",
		`[{"jsonrpc":"2.0","id":5,"method":"ping"}]`: "null",
		`[]`:                       "null",
		`{"id":7,"method":"ping"}`: "7",
		`{"jsonrpc":"1.0","id":"q","method":"ping"}`:  `"q"`,
		`{"jsonrpc":"2.0","id":true,"method":"ping"}`: "null",
	} {
		h.send(line)
		if f := h.expect(id); f.Error == nil || f.Error.Code != -32600 {
			t.Fatalf("%s: reply %+v, want -32600", line, f.Error)
		}
	}
	h.stillServing(t)
}

func TestIDScannerRecoversOnlyTopLevelScalarIDs(t *testing.T) {
	for input, want := range map[string]string{
		`{"jsonrpc":"2.0","id":17,"x":1}`:                 "17",
		`{"params":{"id":5},"id":"a\"b"}`:                 `"a\"b"`,
		`{"id":{"nested":1}}`:                             "",
		`[{"id":1}]`:                                      "",
		`{"identity":3,"id" : -2.5e3 }`:                   "-2.5e3",
		`{"method":"x","params":{"s":"\"id\":4"},"id":8}`: "8",
	} {
		var s idScanner
		for i := 0; i < len(input); i += 3 { // feed in small chunks
			s.feed([]byte(input[i:min(i+3, len(input))]))
		}
		if got := string(s.id()); got != want {
			t.Fatalf("%s: id %q, want %q", input, got, want)
		}
	}
}

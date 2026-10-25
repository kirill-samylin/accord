package bridge

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// StdioTransport frames newline-delimited JSON-RPC for a native client.
//
// The SDK's IOTransport ends the whole connection on one oversized or
// undecodable frame, including valid JSON that is not a JSON-RPC message. Exiting would silently stop heartbeats while a lease may
// be held, so this filter answers such a line with a JSON-RPC error (using the
// request id when it can be recovered from the top-level object) and keeps
// serving. Lines are bounded to maxLine bytes; the rest of an oversized line
// is read and discarded in constant memory.
func StdioTransport(in io.ReadCloser, out io.WriteCloser, maxLine int) mcp.Transport {
	writer := &lockedWriter{w: out}
	return &mcp.IOTransport{
		Reader: &lineFilter{src: in, r: bufio.NewReaderSize(in, 32<<10), out: writer, max: maxLine},
		Writer: writer,
		// The filter has already bounded every forwarded line.
		MaxLineLength: -1,
	}
}

// lockedWriter keeps error replies from interleaving with SDK frames; the SDK
// writes each frame with a single Write call.
type lockedWriter struct {
	mu sync.Mutex
	w  io.WriteCloser
}

func (w *lockedWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.w.Write(p)
}
func (w *lockedWriter) Close() error { return w.w.Close() }

type lineFilter struct {
	src     io.Closer
	r       *bufio.Reader
	out     io.Writer
	max     int
	pending []byte
}

func (f *lineFilter) Close() error { return f.src.Close() }

func (f *lineFilter) Read(p []byte) (int, error) {
	for len(f.pending) == 0 {
		line, err := f.next()
		if err != nil {
			return 0, err
		}
		f.pending = line
	}
	n := copy(p, f.pending)
	f.pending = f.pending[n:]
	return n, nil
}

// next returns the next well-formed line including its newline, answering
// oversized or malformed lines itself. Blank lines are skipped.
func (f *lineFilter) next() ([]byte, error) {
	for {
		var line []byte
		oversized := false
		var scan idScanner
		var readErr error
		for {
			chunk, err := f.r.ReadSlice('\n')
			if !oversized && len(line)+len(chunk) > f.max+2 { // +2 allows a CRLF terminator
				oversized = true
				scan.feed(line)
				line = nil
			}
			if oversized {
				scan.feed(chunk)
			} else {
				line = append(line, chunk...)
			}
			if err == bufio.ErrBufferFull {
				continue
			}
			readErr = err
			break
		}
		if oversized {
			f.reply(scan.id(), jsonrpc.CodeInvalidRequest, fmt.Sprintf("JSON-RPC message exceeds %d bytes and was discarded", f.max))
		} else if body := bytes.TrimRight(line, "\r\n"); len(bytes.TrimSpace(body)) > 0 {
			if len(body) > f.max {
				f.reply(nil, jsonrpc.CodeInvalidRequest, fmt.Sprintf("JSON-RPC message exceeds %d bytes and was discarded", f.max))
			} else if !json.Valid(body) {
				f.reply(nil, jsonrpc.CodeParseError, "line is not a single valid JSON value")
			} else if trimmed := bytes.TrimSpace(body); trimmed[0] != '{' {
				// Batches, scalars and strings are not single JSON-RPC
				// messages; the SDK would end the connection on them.
				f.reply(nil, jsonrpc.CodeInvalidRequest, "line is not a single JSON-RPC message object (batches are not supported)")
			} else if _, err := jsonrpc.DecodeMessage(trimmed); err != nil {
				var scan idScanner
				scan.feed(trimmed)
				f.reply(scan.id(), jsonrpc.CodeInvalidRequest, "line is not a valid JSON-RPC 2.0 message")
			} else {
				return append(body, '\n'), nil
			}
		}
		if readErr != nil {
			return nil, readErr
		}
	}
}

func (f *lineFilter) reply(id []byte, code int64, message string) {
	if id == nil {
		id = []byte("null")
	}
	text, _ := json.Marshal(message)
	_, _ = fmt.Fprintf(f.out, "{\"jsonrpc\":\"2.0\",\"id\":%s,\"error\":{\"code\":%d,\"message\":%s}}\n", id, code, text)
}

// idScanner recovers a top-level "id" member from a stream in O(1) memory.
// It only tracks string state and nesting depth; anything unusual yields no
// id, and the reply then uses null as JSON-RPC prescribes.
type idScanner struct {
	depth                     int
	inString, escape          bool
	expectKey, keying, colon  bool
	capturing, overflow, done bool
	key, value, found         []byte
}

func (s *idScanner) feed(data []byte) {
	for _, c := range data {
		if !s.done {
			s.step(c)
		}
	}
}

func (s *idScanner) step(c byte) {
	if s.inString {
		if s.capturing {
			s.capture(c)
		}
		switch {
		case s.escape:
			s.escape = false
		case c == '\\':
			s.escape = true
		case c == '"':
			s.inString = false
			if s.keying {
				s.keying, s.colon = false, true
			}
			return
		}
		if s.keying && len(s.key) <= 4 {
			s.key = append(s.key, c)
		}
		return
	}
	switch c {
	case ' ', '\t', '\r', '\n':
	case '"':
		s.inString = true
		if s.depth == 1 && s.expectKey {
			s.expectKey, s.keying, s.key = false, true, s.key[:0]
		} else if s.capturing {
			s.capture(c)
		}
	case ':':
		if s.depth == 1 && s.colon {
			s.colon = false
			s.capturing = string(s.key) == "id"
			s.value, s.overflow = s.value[:0], false
		}
	case '{', '[':
		s.depth++
		s.capturing = false // object/array ids are invalid
		s.expectKey = s.depth == 1 && c == '{'
	case '}', ']', ',':
		if s.depth == 1 && s.capturing {
			s.finish()
		}
		if c == ',' {
			s.expectKey = s.depth == 1
		} else {
			s.depth--
		}
	default:
		if s.capturing {
			s.capture(c)
		}
	}
}

func (s *idScanner) capture(c byte) {
	if len(s.value) >= 256 {
		s.overflow = true
		return
	}
	s.value = append(s.value, c)
}

func (s *idScanner) finish() {
	s.capturing = false
	if s.overflow || len(s.value) == 0 {
		return
	}
	var v any
	if json.Unmarshal(s.value, &v) != nil {
		return
	}
	switch v.(type) {
	case string, float64:
		s.found, s.done = append([]byte(nil), s.value...), true
	}
}

func (s *idScanner) id() []byte { return s.found }

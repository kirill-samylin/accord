package bridge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// finiteNumbers rejects numbers outside the float64 range (e.g. 1e309). The
// SDK cannot frame such a request, so forwarding it would look like a lost
// reply; rejecting it locally keeps it a plain client validation error.
func finiteNumbers(raw []byte) bool {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	for {
		token, err := d.Token()
		if err == io.EOF {
			return true
		}
		if err != nil {
			return false
		}
		if n, ok := token.(json.Number); ok {
			if _, err := strconv.ParseFloat(string(n), 64); err != nil {
				return false
			}
		}
	}
}

// rejectedBeforeExecution recognises a definitive JSON-RPC error reply whose
// standard code means the request never reached a tool handler. Any other
// error (no reply, transport failure, handler/internal error that may follow
// a commit) stays ambiguous and keeps quarantining.
func rejectedBeforeExecution(err error) (string, bool) {
	var wire *jsonrpc.Error
	if !errors.As(err, &wire) {
		return "", false
	}
	switch wire.Code {
	case jsonrpc.CodeParseError, jsonrpc.CodeInvalidRequest, jsonrpc.CodeMethodNotFound, jsonrpc.CodeInvalidParams:
		return "Accord rejected the request before execution: " + clip(wire.Message, 512), true
	}
	return "", false
}

// ownLoss decides whether an authoritative LEASE_LOST concerns this bridge.
// It does for session-level calls and for the tracked lease (task key and
// generation). For any other task the bridge re-verifies its own ownership
// with a fresh heartbeat instead of trusting or guessing from the error text.
// A quarantined or locally expired bridge never heartbeats to find out.
func (b *bridge) ownLoss(ctx context.Context, name string, fields map[string]json.RawMessage) bool {
	if b.quarantined || !b.live(time.Now()) {
		return true
	}
	switch name {
	case "describe", "heartbeat", "next":
		return true
	}
	key := stringField(fields, "task_key")
	if name == "link" {
		key = stringField(fields, "to") // link fences the consumer task
	}
	if key == "" {
		return true
	}
	if b.task != nil && key == b.taskKey {
		var generation int64
		if json.Unmarshal(fields["generation"], &generation) != nil || generation == b.task.Generation {
			return true
		}
	}
	return b.heartbeat(ctx) != nil
}

// resourceError passes an authoritative resource error through with a usable
// JSON-RPC code. The service reports domain errors as "CODE: message" with
// code 0; the domain code is mapped and also kept in data.code.
func resourceError(err error) error {
	var wire *jsonrpc.Error
	if !errors.As(err, &wire) {
		return errors.New("cannot read project resource; verify session and connectivity")
	}
	out := &jsonrpc.Error{Code: wire.Code, Message: clip(wire.Message, 1024), Data: wire.Data}
	if out.Code != 0 {
		return out
	}
	domain, message, found := strings.Cut(out.Message, ": ")
	if !found || domain == "" || strings.ToUpper(domain) != domain || strings.ContainsAny(domain, " \t") {
		domain, message = "INTERNAL", out.Message
	}
	switch {
	case domain == "NOT_FOUND":
		out.Code = mcp.CodeResourceNotFound
	case domain == "VALIDATION" || strings.HasPrefix(domain, "INVALID_"):
		out.Code = jsonrpc.CodeInvalidParams
	default:
		out.Code = -32000 // implementation-defined server error; see data.code
	}
	if len(out.Data) == 0 {
		out.Data, _ = json.Marshal(map[string]string{"code": domain, "message": message})
	}
	return out
}

func clip(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	cut := limit
	for cut > 0 && !utf8.RuneStart(value[cut]) {
		cut--
	}
	return value[:cut] + "…"
}

package bridge

import (
	"agent-board/internal/transport"
	"context"
	"encoding/json"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (b *bridge) call(ctx context.Context, name string, raw json.RawMessage) (*mcp.CallToolResult, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	reading := b.readOnly[name]
	if b.quarantined && name != "get" && name != "events" {
		return b.uncertain(), nil
	}
	if err := transport.ValidateJSONFields(raw); err != nil {
		return failure("VALIDATION", "Invalid or duplicate JSON fields"), nil
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil {
		return failure("VALIDATION", "JSON object required"), nil
	}
	requestKey := name + "/" + stringField(fields, "request_id")
	replay := b.successful[requestKey]
	if name == "describe" {
		if b.session.SessionID != "" && !replay {
			return failure("SESSION_ALREADY_OPEN", "One bridge has one session. Restart to change role or open a fresh session."), nil
		}
	} else {
		if b.session.SessionID == "" {
			return failure("NO_SESSION", "Call describe first."), nil
		}
		if supplied, ok := fields["session_id"]; ok {
			var id string
			if json.Unmarshal(supplied, &id) != nil || id != b.session.SessionID {
				return failure("SESSION_MISMATCH", "This bridge only serves its own board session."), nil
			}
		}
		fields["session_id"], _ = json.Marshal(b.session.SessionID)
	}
	if b.session.SessionID != "" && !b.quarantined {
		if !b.live(time.Now()) {
			b.quarantine()
		} else if time.Since(b.lastHeartbeat) >= b.interval {
			// A queue of foreground calls must not starve the independent timer.
			// This is a separate fresh heartbeat, never a renewal inferred from replay.
			if err := b.heartbeat(ctx); err != nil {
				b.quarantine()
			}
		}
		if b.quarantined && name != "get" && name != "events" {
			return b.uncertain(), nil
		}
	}
	// Exact successful retries are forwarded for server auth/idempotency checks,
	// but their historical responses never update current ownership or TTL.
	if !reading && !replay && len(b.successful) >= 32768 {
		return b.uncertain(), nil
	}
	if name == "heartbeat" && !replay && !b.matchesHeartbeat(fields) {
		return failure("VALIDATION", "Heartbeat must acknowledge the bridge's current generations."), nil
	}
	if !replay && b.task != nil && (name == "release" || name == "update" || name == "comment" || name == "transition") {
		key := stringField(fields, "task_key")
		// Epics may be changed without claiming a task; actual leased operations
		// remain subject to authoritative server validation.
		if key == b.taskKey {
			var generation int64
			if json.Unmarshal(fields["generation"], &generation) != nil || generation != b.task.Generation {
				return failure("VALIDATION", "Use the current task generation."), nil
			}
		}
	}
	duration := 10 * time.Second
	if b.rpcTimeout > 0 {
		duration = b.rpcTimeout
	}
	calling, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	started := time.Now()
	result, err := b.upstream.CallTool(calling, &mcp.CallToolParams{Name: name, Arguments: fields})
	if err != nil {
		return b.uncertain(), nil
	}
	data, err := resultJSON(result)
	if err != nil {
		return b.uncertain(), nil
	}
	if result.IsError {
		switch errorCode(data) {
		case "LEASE_LOST", "UNAUTHORIZED", "NO_SESSION", "INTERNAL", "":
			return b.uncertain(), nil
		}
		return result, nil
	}
	if replay {
		return result, nil
	}
	if !reading {
		if err = b.accept(name, fields, data, started); err != nil {
			return b.uncertain(), nil
		}
		b.successful[requestKey] = true
		if name == "describe" || name == "next" {
			if err = b.heartbeat(ctx); err != nil {
				return b.uncertain(), nil
			}
			select {
			case b.wake <- struct{}{}:
			default:
			}
		}
	}
	return result, nil
}

package bridge

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"time"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// pulse runs independently of the model. All ownership changes and renewals
// share the mutex, preventing a completed/released generation being renewed.
func (b *bridge) pulse(ctx context.Context) {
	for {
		b.mu.Lock()
		interval := b.interval
		active := b.session.SessionID != "" && !b.quarantined
		b.mu.Unlock()
		if !active {
			select {
			case <-ctx.Done():
				return
			case <-b.wake:
				continue
			}
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-b.wake:
			timer.Stop()
			continue
		case <-timer.C:
		}
		b.mu.Lock()
		if ctx.Err() != nil {
			b.mu.Unlock()
			return
		}
		if !b.quarantined && b.session.SessionID != "" {
			if !b.live(time.Now()) || b.heartbeat(ctx) != nil {
				b.quarantine()
			}
		}
		b.mu.Unlock()
	}
}

// heartbeat always uses a fresh idempotency key. It never acquires ownership.
func (b *bridge) heartbeat(ctx context.Context) error {
	input := core.HeartbeatInput{RequestID: "bridge-" + rand.Text(), RoleGeneration: b.session.RoleGeneration}
	if b.task != nil {
		input.TaskKey = b.taskKey
		input.Generation = b.task.Generation
	}
	raw, _ := json.Marshal(input)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(raw, &fields)
	fields["session_id"], _ = json.Marshal(b.session.SessionID)
	heartbeat, cancel := context.WithTimeout(ctx, b.rpcTimeout)
	defer cancel()
	started := time.Now()
	result, err := b.upstream.CallTool(heartbeat, &mcp.CallToolParams{Name: "heartbeat", Arguments: fields})
	if err != nil {
		return errors.New("heartbeat failed")
	}
	data, err := resultJSON(result)
	if err != nil || result.IsError {
		return errors.New("heartbeat rejected")
	}
	if err = b.accept("heartbeat", fields, data, started); err != nil {
		return err
	}
	b.lastHeartbeat = started
	return nil
}

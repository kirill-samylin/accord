package bridge

import (
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"time"

	"agent-board/internal/core"
)

func (b *bridge) timing(v core.SessionView, started time.Time) error {
	for _, seconds := range []float64{v.SessionTTLSeconds, v.TaskTTLSeconds, v.HeartbeatEverySeconds} {
		if math.IsNaN(seconds) || math.IsInf(seconds, 0) || seconds <= 0 || seconds > 86400 {
			return errors.New("invalid server TTL")
		}
	}
	roleTTL := v.RoleTTLSeconds
	if roleTTL == 0 {
		roleTTL = v.SessionTTLSeconds
	}
	if math.IsNaN(roleTTL) || math.IsInf(roleTTL, 0) || roleTTL <= 0 || roleTTL > 86400 {
		return errors.New("invalid server role TTL")
	}
	interval := time.Duration(min(v.SessionTTLSeconds, v.TaskTTLSeconds, roleTTL) / 3 * float64(time.Second))
	advertised := time.Duration(v.HeartbeatEverySeconds * float64(time.Second))
	interval = min(interval, advertised)
	if b.override > 0 {
		interval = min(interval, b.override)
	}
	if interval <= 0 {
		return errors.New("invalid interval")
	}
	if b.session.SessionID != "" && (v.SessionID != b.session.SessionID || v.Role != b.session.Role || v.RoleGeneration != b.session.RoleGeneration) {
		return errors.New("session identity changed")
	}
	if v.SessionID == "" || !regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`).MatchString(v.Role) || v.ExpiresAt.IsZero() {
		return errors.New("invalid session")
	}
	b.session = v
	b.interval = interval
	b.rpcTimeout = time.Duration(min(v.SessionTTLSeconds, v.TaskTTLSeconds, roleTTL) / 3 * float64(time.Second))
	// A replay or a clock difference cannot extend the local safety horizon.
	b.sessionDeadline = minTime(started.Add(time.Duration(v.SessionTTLSeconds*float64(time.Second))), v.ExpiresAt)
	if v.RoleExpiresAt != nil {
		b.sessionDeadline = minTime(b.sessionDeadline, *v.RoleExpiresAt)
	}
	b.sessionDeadline = minTime(b.sessionDeadline, started.Add(time.Duration(roleTTL*float64(time.Second))))
	if !b.sessionDeadline.After(time.Now()) {
		return errors.New("session already expired")
	}
	return nil
}

func (b *bridge) accept(name string, fields map[string]json.RawMessage, data json.RawMessage, started time.Time) error {
	switch name {
	case "describe", "heartbeat":
		var view core.SessionView
		if err := json.Unmarshal(data, &view); err != nil {
			return err
		}
		if name == "describe" && view.Role != stringField(fields, "role") {
			return errors.New("describe role differs from requested identity")
		}
		if err := b.timing(view, started); err != nil {
			return err
		}
		if name == "heartbeat" && b.task != nil {
			b.taskDeadline = started.Add(time.Duration(view.TaskTTLSeconds * float64(time.Second)))
		}
		return nil
	case "next":
		var claim core.Claim
		if err := json.Unmarshal(data, &claim); err != nil {
			return err
		}
		if claim.Task == nil {
			if claim.Lease != nil || b.task != nil {
				return errors.New("inconsistent empty claim")
			}
		} else {
			lease := claim.Lease
			if lease == nil || claim.Task.Lease == nil || lease.SessionID != b.session.SessionID || lease.Generation <= 0 || lease.Generation != claim.Task.Generation || claim.Task.Lease.Generation != lease.Generation || claim.Task.Lease.SessionID != lease.SessionID || claim.Task.Key == "" {
				return errors.New("inconsistent claim")
			}
			copied := *lease
			b.task = &copied
			b.taskKey = claim.Task.Key
			b.taskDeadline = minTime(started.Add(time.Duration(b.session.TaskTTLSeconds*float64(time.Second))), lease.ExpiresAt)
			if !b.taskDeadline.After(time.Now()) {
				return errors.New("claim already expired")
			}
		}
	case "release", "transition", "update", "comment", "link":
		var task core.Task
		if err := json.Unmarshal(data, &task); err != nil || task.Key == "" {
			return errors.New("invalid task result")
		}
		if b.task != nil && task.Key == b.taskKey {
			if task.Lease == nil {
				b.task = nil
				b.taskKey = ""
				b.taskDeadline = time.Time{}
			} else {
				if task.Lease.SessionID != b.session.SessionID || task.Lease.Generation != b.task.Generation {
					return errors.New("lease changed unexpectedly")
				}
				b.taskDeadline = minTime(started.Add(time.Duration(b.session.TaskTTLSeconds*float64(time.Second))), task.Lease.ExpiresAt)
				if !b.taskDeadline.After(time.Now()) {
					return errors.New("lease already expired")
				}
			}
		}
	}
	ttl := b.session.SessionTTLSeconds
	if b.session.RoleTTLSeconds > 0 {
		ttl = min(ttl, b.session.RoleTTLSeconds)
	}
	b.sessionDeadline = started.Add(time.Duration(ttl * float64(time.Second)))
	return nil
}

func (b *bridge) live(now time.Time) bool {
	return b.sessionDeadline.After(now) && (b.task == nil || b.taskDeadline.After(now))
}
func minTime(a, z time.Time) time.Time {
	if a.Before(z) {
		return a
	}
	return z
}

func (b *bridge) matchesHeartbeat(fields map[string]json.RawMessage) bool {
	var in core.HeartbeatInput
	// session_id is intentionally outside the domain DTO.
	raw, _ := json.Marshal(fields)
	if json.Unmarshal(raw, &in) != nil {
		return false
	}
	if in.RoleGeneration != b.session.RoleGeneration {
		return false
	}
	if b.task == nil {
		return in.TaskKey == "" && in.Generation == 0
	}
	return in.TaskKey == b.taskKey && in.Generation == b.task.Generation
}

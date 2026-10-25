package store

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"agent-board/internal/core"
)

func identifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
func mutate[T any](s *Store, ctx context.Context, a core.Actor, request, method string, input any, fn func(*board, *core.Session, time.Time) (T, error)) (T, error) {
	var zero T
	// Limits count characters (runes), matching the message and the UI.
	if strings.TrimSpace(request) == "" || utf8.RuneCountInString(request) > 128 || !plainText(request, false) {
		return zero, core.Fail("invalid_input", "request_id must contain 1..128 characters without control characters")
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return zero, err
	}
	digest := sha256.Sum256(append([]byte(method+"\n"), raw...))
	hash := hex.EncodeToString(digest[:])
	return transact(s, ctx, func(b *board, now time.Time) (T, error) {
		if err := authenticatePrincipal(ctx, b, a.Principal, now); err != nil {
			return zero, err
		}
		if err := executionGuard(ctx, b, a, method, input, now); err != nil {
			return zero, err
		}
		if err := automationTaskMutationGuard(ctx, b, a, method, input, now); err != nil {
			return zero, err
		}
		if method == "describe" {
			if in, ok := input.(core.OpenSession); ok && managedSessionRequested(in) {
				if _, err := managedSessionProfile(ctx, b, a.Principal, in, now); err != nil {
					return zero, err
				}
			}
		}
		key := a.Principal.ID + "/" + a.SessionID + "/" + request
		var previous replay
		var savedResponse string
		replayErr := b.tx.QueryRowContext(ctx, `SELECT hash,response FROM idempotency WHERE key=?`, key).Scan(&previous.Hash, &savedResponse)
		if replayErr == nil {
			previous.Response = json.RawMessage(savedResponse)
			if previous.Hash != hash {
				return zero, core.Fail("idempotency_conflict", "request_id already used with a different payload")
			}
			var result T
			if err := json.Unmarshal(previous.Response, &result); err != nil {
				return zero, err
			}
			return result, nil
		}
		if replayErr != nil && replayErr != sql.ErrNoRows {
			return zero, replayErr
		}
		var sess *core.Session
		if method != "describe" {
			var err error
			sess, err = sessionFor(b, a, now)
			if err != nil {
				return zero, err
			}
		}
		// After the replay lookup: an exact retry of an already stored request
		// keeps returning its original result even if hygiene rules tightened.
		if err := inputText(input); err != nil {
			return zero, err
		}
		// Domain mutations operate on an isolated copy: a rejected operation cannot
		// persist partial task changes, events, or lease renewals.
		snapshot, err := json.Marshal(b)
		if err != nil {
			return zero, err
		}
		var candidate board
		if err = json.Unmarshal(snapshot, &candidate); err != nil {
			return zero, err
		}
		candidate.tx = b.tx
		candidate.EventBase = b.EventBase
		if sess != nil {
			sess = candidate.Sessions[sess.SessionID]
		}
		result, err := fn(&candidate, sess, now)
		if err != nil {
			return zero, err
		}
		reconcileEpics(&candidate, now, a.SessionID)
		if sess != nil {
			sess.ExpiresAt = now.Add(s.cfg.SessionTTL)
		}
		response, err := json.Marshal(result)
		if err != nil {
			return zero, err
		}
		if _, err = candidate.tx.ExecContext(ctx, `INSERT INTO idempotency(key,hash,response) VALUES(?,?,?)`, key, hash, string(response)); err != nil {
			return zero, err
		}
		candidate.EventBase = b.EventBase
		*b = candidate
		return result, nil
	})
}
func taskFor(b *board, key string) (*core.Task, error) {
	t := b.Tasks[key]
	if t == nil {
		return nil, core.Fail("not_found", "task does not exist")
	}
	return t, nil
}
func owned(b *board, a core.Actor, key string, generation int64) (*core.Task, error) {
	t, err := taskFor(b, key)
	if err != nil {
		return nil, err
	}
	if t.Lease == nil || t.Lease.SessionID != a.SessionID || t.Lease.Generation != generation {
		// LEASE_LOST tells a holder that ITS ownership ended, and clients stop
		// work on it. A caller that never held this task while another session
		// does gets LEASE_HELD instead, so a stray call cannot look like loss.
		if t.Lease != nil && t.Lease.SessionID != a.SessionID && !attemptedBy(t, a.SessionID) {
			return nil, core.Fail("lease_held", "another session holds this task; only its holder may change it")
		}
		return nil, core.Fail("lease_lost", "active lease and current generation required")
	}
	return t, nil
}
func attemptedBy(t *core.Task, session string) bool {
	for _, attempt := range t.Attempts {
		if attempt.SessionID == session {
			return true
		}
	}
	return false
}
func coordinator(sess *core.Session) error {
	if sess.Role != "coordinator" {
		return core.Fail("forbidden", "coordinator role required")
	}
	return nil
}
func bounded(value string, max int) bool {
	return len(strings.TrimSpace(value)) > 0 && len(value) <= max
}

// textLimit explains which rule a free-text field broke instead of one
// combined message; what names the field in Russian ("требование").
func textLimit(value string, max int, what string) error {
	switch {
	case strings.TrimSpace(value) == "":
		return core.Fail("validation", "Поле «"+what+"» пустое — опишите его текстом")
	case len(value) > max:
		return core.Fail("validation", fmt.Sprintf("Поле «%s» длиннее %d КиБ — сократите текст", what, max/1024))
	case !utf8.ValidString(value):
		return core.Fail("validation", "Поле «"+what+"» содержит недопустимые символы")
	}
	return nil
}
func (s *Store) renew(t *core.Task, now time.Time) {
	if t.Lease != nil {
		t.Lease.ExpiresAt = now.Add(s.cfg.LeaseTTL)
	}
}
func viewTask(b *board, t *core.Task) core.Task {
	v := *t
	v.Blocked = blocked(b, t)
	projectPeople(b, &v)
	return v
}
func nextKey(b *board) string { b.Sequence++; return fmt.Sprintf("T-%d", b.Sequence) }

func validTitle(value string) bool {
	return strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= 200 && plainText(value, false)
}

// plainText is the shared core.PlainText rule (no control characters except
// \n, \t and CRLF in multi-line bodies).
func plainText(value string, multiline bool) bool { return core.PlainText(value, multiline) }

// inputText is the shared text boundary for externally supplied mutations;
// internal constructors (automation) validate their own generated text.
func inputText(input any) error {
	switch in := input.(type) {
	case core.OpenSession:
		if !plainText(in.RunID, false) {
			return core.Fail("invalid_input", "run_id must not contain control characters or invalid UTF-8")
		}
	case core.CreateInput:
		if !plainText(in.Description, true) {
			return core.Fail("invalid_input", "description must be valid UTF-8 without control characters other than newline and tab")
		}
		if err := validOwner(in.Type, in.Owner); err != nil {
			return err
		}
		return validAreas(in.Areas)
	case core.UpdateInput:
		if in.Description != nil && !plainText(*in.Description, true) {
			return core.Fail("invalid_input", "description must be valid UTF-8 without control characters other than newline and tab")
		}
	case core.CommentInput:
		if !plainText(in.Text, true) {
			return core.Fail("invalid_input", "comment must be valid UTF-8 without control characters other than newline and tab")
		}
	}
	return nil
}

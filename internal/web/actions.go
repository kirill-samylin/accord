package web

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type webAction struct {
	ID, Operation, Payload, SessionID, Result string
	Fresh                                     bool
}

func (s *Server) issueAction(ctx context.Context, b *browserSession, operation string) (string, error) {
	id, err := randomID()
	if err != nil {
		return "", err
	}
	// Bound unused form capabilities while retaining all bound native replay IDs.
	_, err = s.db.ExecContext(ctx, `DELETE FROM web_actions WHERE id IN (SELECT id FROM web_actions WHERE session_hash=? AND payload='' ORDER BY created DESC LIMIT -1 OFFSET 2047)`, b.Hash)
	if err != nil {
		return "", err
	}
	epoch := int64(0)
	if s.options.ActionEpoch != nil {
		epoch, err = s.options.ActionEpoch(ctx)
		if err != nil {
			return "", err
		}
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(ctx, `INSERT INTO web_actions(id,session_hash,operation,created) VALUES(?,?,?,?)`, id, b.Hash, operation, time.Now().UnixNano())
	if err == nil {
		_, err = tx.ExecContext(ctx, `INSERT INTO web_action_epochs VALUES(?,?)`, id, epoch)
	}
	if err == nil {
		err = tx.Commit()
	}
	return id, err
}
func (s *Server) checkAction(ctx context.Context, b *browserSession, id, operation string) error {
	if len(id) != 64 {
		return core.Fail("idempotency_conflict", "Обновите форму: действие неизвестно")
	}
	var owner, op string
	err := s.db.QueryRowContext(ctx, `SELECT session_hash,operation FROM web_actions WHERE id=?`, id).Scan(&owner, &op)
	if errors.Is(err, sql.ErrNoRows) {
		return core.Fail("idempotency_conflict", "Обновите форму: действие неизвестно")
	}
	if err != nil {
		return err
	}
	if owner != b.Hash || op != operation {
		return core.Fail("idempotency_conflict", "Действие принадлежит другой форме")
	}
	if s.options.ActionEpoch != nil {
		current, e := s.options.ActionEpoch(ctx)
		if e != nil {
			return e
		}
		var issued int64
		e = s.db.QueryRowContext(ctx, `SELECT epoch FROM web_action_epochs WHERE id=?`, id).Scan(&issued)
		if e != nil && !(errors.Is(e, sql.ErrNoRows) && current == 0) {
			return core.Fail("unauthorized", "reopen project")
		}
		if issued != current {
			return core.Fail("unauthorized", "reopen project")
		}
		var original string
		if e = s.db.QueryRowContext(ctx, `SELECT session_id FROM web_actions WHERE id=?`, id).Scan(&original); e != nil {
			return e
		}
		if original != "" && s.options.SessionAllowed != nil {
			if e = s.options.SessionAllowed(ctx, original); e != nil {
				return e
			}
		}
	}
	return nil
}
func (s *Server) bindAction(ctx context.Context, b *browserSession, id, operation string, payload []byte) (*webAction, error) {
	if err := s.checkAction(ctx, b, id, operation); err != nil {
		return nil, err
	}
	a := &webAction{ID: id, Operation: operation}
	err := s.db.QueryRowContext(ctx, `SELECT payload,session_id,result FROM web_actions WHERE id=?`, id).Scan(&a.Payload, &a.SessionID, &a.Result)
	if err != nil {
		return nil, err
	}
	if a.Payload != "" {
		if a.Payload != string(payload) {
			return nil, core.Fail("idempotency_conflict", "Повтор содержит другие поля")
		}
		return a, nil
	}
	a.Fresh = true
	a.Payload = string(payload)
	a.SessionID = b.Session.SessionID
	_, err = s.db.ExecContext(ctx, `UPDATE web_actions SET payload=?,session_id=? WHERE id=? AND payload=''`, a.Payload, a.SessionID, id)
	return a, err
}
func (s *Server) completeAction(ctx context.Context, id, result string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE web_actions SET result=? WHERE id=?`, result, id)
	return err
}

type actionResult struct {
	Redirect     string            `json:"redirect,omitempty"`
	Session      *core.SessionView `json:"session,omitempty"`
	NextActionID string            `json:"next_action_id,omitempty"`
	Error        *core.Error       `json:"error,omitempty"`
}

func (s *Server) runAction(ctx context.Context, b *browserSession, a *webAction) (actionResult, error) {
	var out actionResult
	if err := s.checkAction(ctx, b, a.ID, a.Operation); err != nil {
		return out, err
	}
	// Cached results require the same current write permission as a new action.
	if err := s.guarded(ctx, true, a.SessionID, func() error { return nil }); err != nil {
		return out, err
	}
	if a.Result != "" {
		if strings.HasPrefix(a.Operation, "managed:") {
			if err := s.checkManagedPermission(ctx, core.Actor{Principal: b.Principal, SessionID: a.SessionID}, a); err != nil {
				return out, err
			}
		}
		err := json.Unmarshal([]byte(a.Result), &out)
		if err != nil {
			return out, err
		}
		if out.Error != nil {
			return out, out.Error
		}
		return out, nil
	}
	started := time.Now()
	actor := core.Actor{Principal: b.Principal, SessionID: a.SessionID}
	var err error
	switch a.Operation {
	case "managed:start", "managed:command", "managed:disable":
		out.Redirect, err = s.runManagedAction(ctx, actor, a)
	case "coord:assign", "coord:link", "coord:transition", "coord:checks", "coord:duplicate":
		out.Redirect, err = s.runCoordinationAction(ctx, actor, a)
	case "profile:create", "profile:update", "profile:archive":
		out.Redirect, err = s.runProfileAction(ctx, actor, a)
	case "update":
		var in core.UpdateInput
		if err = json.Unmarshal([]byte(a.Payload), &in); err != nil {
			return out, err
		}
		_, err = s.board.Update(ctx, actor, in)
		out.Redirect = "/web/tasks/" + in.TaskKey
	case "create":
		var in core.CreateInput
		if err = json.Unmarshal([]byte(a.Payload), &in); err != nil {
			return out, err
		}
		var task core.Task
		task, err = s.board.Create(ctx, actor, in)
		out.Redirect = "/web/tasks/" + task.Key
	case "claim":
		var in core.NextInput
		if err = json.Unmarshal([]byte(a.Payload), &in); err != nil {
			return out, err
		}
		var claim core.Claim
		claim, err = s.board.Next(ctx, actor, in)
		if err == nil && claim.Task == nil {
			err = core.Fail("conflict", "Вопрос уже занят или недоступен")
		}
		out.Redirect = "/web/tasks/" + in.TaskKey
		if err == nil && claim.Task != nil && a.SessionID == b.Session.SessionID {
			b.Session.MyTasks = []core.Task{*claim.Task}
		}
	case "answer":
		var in core.TransitionInput
		if err = json.Unmarshal([]byte(a.Payload), &in); err != nil {
			return out, err
		}
		_, err = s.board.Transition(ctx, actor, in)
		if err == nil && a.SessionID == b.Session.SessionID {
			b.Session.MyTasks = nil
		}
		out.Redirect = "/web/tasks/" + in.TaskKey
	case "release":
		var in core.LeaseInput
		if err = json.Unmarshal([]byte(a.Payload), &in); err != nil {
			return out, err
		}
		_, err = s.board.Release(ctx, actor, in)
		if err == nil && a.SessionID == b.Session.SessionID {
			b.Session.MyTasks = nil
		}
		out.Redirect = "/web/tasks/" + in.TaskKey
	case "heartbeat":
		var in core.HeartbeatInput
		if err = json.Unmarshal([]byte(a.Payload), &in); err != nil {
			return out, err
		}
		var session core.SessionView
		session, err = s.board.Heartbeat(ctx, actor, in)
		if err == nil {
			if session.ExpiresAt.After(time.Now()) {
				task, e := s.board.Get(ctx, actor, core.GetInput{TaskKey: in.TaskKey})
				if e != nil {
					return out, e
				}
				if task.Lease == nil || task.Lease.SessionID != actor.SessionID || task.Generation != in.Generation {
					return out, core.Fail("lease_lost", "Владение изменилось")
				}
				session.MyTasks = []core.Task{task}
			}
			out.Session = &session
			out.NextActionID, err = s.issueAction(ctx, b, "heartbeat")
		}
	default:
		return out, core.Fail("validation", "unknown native operation")
	}
	if err != nil {
		var domain *core.Error
		if errors.As(err, &domain) {
			out.Error = domain
			raw, e := json.Marshal(out)
			if e != nil {
				return out, e
			}
			if e = s.completeAction(ctx, a.ID, string(raw)); e != nil {
				return out, e
			}
		}
		return out, err
	}
	if a.SessionID == b.Session.SessionID {
		if out.Session != nil {
			b.Session = *out.Session
		} else if a.Fresh && b.Session.ExpiresAt.After(started) {
			b.Session.ExpiresAt = started.Add(time.Duration(b.Session.SessionTTLSeconds * float64(time.Second)))
		}
		if err = s.saveSession(ctx, b); err != nil {
			return out, err
		}
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return out, err
	}
	err = s.completeAction(ctx, a.ID, string(raw))
	return out, err
}

// Resolve uncertain native calls before opening another native session. Stored
// payload and the original session are replayed even after that session expires.
func (s *Server) recoverActions(ctx context.Context, b *browserSession) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id,operation,payload,session_id,result FROM web_actions WHERE session_hash=? AND payload<>'' AND result='' AND operation IN ('create','claim','answer','release','heartbeat','update','profile:create','profile:update','profile:archive','coord:assign','coord:link','coord:transition','coord:checks','coord:duplicate','managed:start','managed:command','managed:disable') ORDER BY created`, b.Hash)
	if err != nil {
		return err
	}
	var pending []*webAction
	for rows.Next() {
		a := new(webAction)
		if err = rows.Scan(&a.ID, &a.Operation, &a.Payload, &a.SessionID, &a.Result); err != nil {
			rows.Close()
			return err
		}
		pending = append(pending, a)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, a := range pending {
		if _, err = s.runAction(ctx, b, a); err != nil {
			var domain *core.Error
			if !errors.As(err, &domain) {
				return err
			}
		}
	}
	return nil
}

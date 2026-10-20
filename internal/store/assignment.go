package store

import (
	"agent-board/internal/core"
	"context"
	"time"
	"unicode/utf8"
)

func taskAssignment(b *board, role, kind, id string, version int64) (*core.TaskAssignment, error) {
	if id == "" && version == 0 {
		return nil, nil
	}
	if id == "" || version < 1 || kind != "task" || role == "human" {
		return nil, core.Fail("validation", "Назначение требует задачу агента и полную пару профиля/версии")
	}
	p, err := profileFor(b, id)
	if err != nil {
		return nil, err
	}
	if p.State != "active" || p.Version != version {
		return nil, core.Fail("conflict", "Профиль изменён или архивирован")
	}
	if p.Role != role {
		return nil, core.Fail("validation", "Роль профиля должна совпадать с задачей")
	}
	return &core.TaskAssignment{ProfileID: p.ID, ProfileVersion: p.Version, ProfileName: p.Name, Role: p.Role}, nil
}
func assignmentAvailable(b *board, t *core.Task) bool {
	if t.Assignment == nil {
		return true
	}
	p := b.Profiles[t.Assignment.ProfileID]
	return p != nil && p.State == "active" && p.Version == t.Assignment.ProfileVersion && p.Role == t.Role && t.Assignment.Role == t.Role
}
func assignmentMatches(b *board, t *core.Task, sess *core.Session) bool {
	if !assignmentAvailable(b, t) {
		return false
	}
	a := t.Assignment
	return a == nil || sess.Profile != nil && sess.Profile.ID == a.ProfileID && sess.Profile.Version == a.ProfileVersion && sess.Profile.Role == a.Role
}
func administrative(sess *core.Session, t *core.Task, a core.Actor) bool {
	return sess.Role == "coordinator" || sess.Role == "human" || t.Requester == a.Principal.ID
}
func (s *Store) Assign(ctx context.Context, a core.Actor, in core.AssignInput) (core.Task, error) {
	return mutate(s, ctx, a, in.RequestID, "assign", in, func(b *board, sess *core.Session, now time.Time) (core.Task, error) {
		t, err := taskFor(b, in.TaskKey)
		if err != nil {
			return core.Task{}, err
		}
		if !administrative(sess, t, a) {
			return core.Task{}, core.Fail("forbidden", "Назначение доступно координатору, человеку или заказчику")
		}
		if t.Type != "task" || t.State != "ready" && t.State != "created" || t.Lease != nil {
			return core.Task{}, core.Fail("conflict", "Назначать можно только незахваченную готовую задачу")
		}
		if in.Version < 1 || in.Version != t.Version {
			return core.Task{}, core.Fail("conflict", "Задача изменилась; обновите форму")
		}
		if !bounded(in.Reason, 4096) || !utf8.ValidString(in.Reason) {
			return core.Task{}, core.Fail("validation", "Укажите причину до4КиБ")
		}
		var assignment *core.TaskAssignment
		switch in.Mode {
		case "profile":
			if in.ProfileID == "" || in.ProfileVersion < 1 {
				return core.Task{}, core.Fail("validation", "Нужен профиль и версия")
			}
			assignment, err = taskAssignment(b, t.Role, t.Type, in.ProfileID, in.ProfileVersion)
		case "role_queue":
			if in.ProfileID != "" || in.ProfileVersion != 0 {
				return core.Task{}, core.Fail("validation", "Очередь роли не принимает профиль")
			}
		default:
			return core.Task{}, core.Fail("validation", "Выберите profile или role_queue")
		}
		if err != nil {
			return core.Task{}, err
		}
		previous := t.Assignment
		t.Assignment = assignment
		t.Version++
		emit(b, now, "task_assigned", t.Key, a.SessionID)
		b.Events[len(b.Events)-1].Reason = in.Reason
		b.Events[len(b.Events)-1].Data = &core.EventData{AssignmentChange: &core.AssignmentChange{Previous: previous, Next: assignment, Reason: in.Reason}}
		return viewTask(b, t), nil
	})
}

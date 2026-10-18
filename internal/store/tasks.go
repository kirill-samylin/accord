package store

import (
	"agent-board/internal/core"
	"context"
	"fmt"
	"sort"
	"time"
)

func (s *Store) Create(ctx context.Context, a core.Actor, in core.CreateInput) (core.Task, error) {
	return mutate(s, ctx, a, in.RequestID, "create", in, func(b *board, sess *core.Session, now time.Time) (core.Task, error) {
		var zero core.Task
		if !validTitle(in.Title) || len(in.Description) > 16384 || in.Priority < 0 || in.Priority > 100 {
			return zero, core.Fail("invalid_input", "invalid title, description or priority")
		}
		if in.Type != "task" && in.Type != "epic" {
			return zero, core.Fail("invalid_input", "type must be task or epic")
		}
		t := &core.Task{CreatedAt: &now, Category: in.Category, Complexity: in.Complexity, ExternalID: in.ExternalID, Key: nextKey(b), Type: in.Type, Title: in.Title, Description: in.Description, Role: in.Role, State: "ready", EpicKey: in.EpicKey, Priority: in.Priority, Version: 1, Requester: a.Principal.ID, ReadyAt: now, Areas: in.Areas}
		if t.Priority == 0 {
			t.Priority = 3
		}
		if in.Type == "epic" {
			if err := requireCreate(b, sess, "epic"); err != nil {
				return zero, err
			}
			t.Key = fmt.Sprintf("E-%d", b.Sequence)
			t.Role = ""
			t.EpicKey = ""
			t.State = "open"
			t.Owner = in.Owner

		} else {
			if err := requireCreate(b, sess, "task"); err != nil {
				return zero, err
			}
			epic, err := readTaskFor(b, in.EpicKey)
			if err != nil {
				return zero, err
			}
			if epic.Type != "epic" || epic.State == "cancelled" {
				return zero, core.Fail("invalid_input", "active epic required")
			}
			if !validRole(b, in.Role) || (in.Ready == nil || *in.Ready) && !bounded(in.Description, 16384) {
				return zero, core.Fail("invalid_input", "task role and description required")
			}
			if in.Priority == 0 {
				t.Priority = epic.Priority
			}
			if in.Areas == nil {
				t.Areas = append([]string(nil), epic.Areas...)
			}
			if in.Ready != nil && !*in.Ready {
				t.State = "created"
			}
			if verifier(b, in.Role) {
				target, err := readTaskFor(b, in.TargetKey)
				if err != nil {
					return zero, err
				}
				if !reviewTargetAllowed(b, in.Role, target.Role) || !validCommit(in.TargetCommit) {
					return zero, core.Fail("invalid_input", "review requires developer target and commit")
				}
				generation := int64(0)
				if target.State == "done" {
					generation = target.Generation
				}
				t.Review = &core.Review{TargetKey: in.TargetKey, TargetCommit: in.TargetCommit, TargetGeneration: generation, AuthorSession: target.AuthorSession}
				reopenEpic(b, target.EpicKey, now, a.SessionID)
			}
			if epic.State == "done" {
				epic.State = "open"
				epic.Version++
				emit(b, now, "epic_reopened", epic.Key, a.SessionID)
			}
		}
		if err := prepareAreas(b, t); err != nil {
			return zero, err
		}
		if err := checkDuplicate(b, t, sess, in.Force, in.Reason); err != nil {
			return zero, err
		}
		if err := validateMetadata(b, t); err != nil {
			return zero, err
		}
		b.putTask(t)
		emitWarnings(b, t, now, a.SessionID)
		emit(b, now, "task_created", t.Key, a.SessionID)
		return viewTask(b, t), nil
	})
}
func (s *Store) Get(ctx context.Context, a core.Actor, in core.GetInput) (core.Task, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.Task, error) {
		if err := authenticatePrincipal(ctx, b, a.Principal, now); err != nil {
			return core.Task{}, err
		}
		if err := readActor(ctx, b, a, now); err != nil {
			return core.Task{}, err
		}
		sess, _ := sessionFor(b, a, now)
		if err := toolAllowed(b, sess.Role, "get"); err != nil {
			return core.Task{}, err
		}
		t, err := readTaskFor(b, in.TaskKey)
		if err != nil {
			return core.Task{}, err
		}
		return viewTask(b, t), nil
	})
}
func (s *Store) Next(ctx context.Context, a core.Actor, in core.NextInput) (core.Claim, error) {
	return mutate(s, ctx, a, in.RequestID, "next", in, func(b *board, sess *core.Session, now time.Time) (core.Claim, error) {
		for _, t := range b.ownedTasks(a.SessionID) {
			if t.Lease != nil && t.Lease.SessionID == a.SessionID {
				return core.Claim{}, core.Fail("lease_held", "session already owns a task")
			}
		}
		candidates := make([]*core.Task, 0)
		for _, t := range b.roleTasks(sess.Role) {
			epic := b.Tasks[t.EpicKey]
			if epic == nil || epic.Type != "epic" || epic.State != "open" {
				continue
			}
			if t.Type == "task" && t.Role == sess.Role && t.State == "ready" && t.Lease == nil && !blocked(b, t) && (in.TaskKey == "" || in.TaskKey == t.Key) && (in.EpicKey == "" || in.EpicKey == t.EpicKey) {
				if eligibleClaim(b, t, sess, now) == nil {
					candidates = append(candidates, t)
				}
			}
		}
		sort.Slice(candidates, func(i, j int) bool {
			a, z := candidates[i], candidates[j]
			if b.Config.Workflow.Rules.EpicOwner.Mode == "warn" {
				af, zf := foreignEpic(b, a, sess), foreignEpic(b, z, sess)
				if af != zf {
					return !af
				}
			}
			if a.Priority != z.Priority {
				return a.Priority < z.Priority
			}
			if !a.ReadyAt.Equal(z.ReadyAt) {
				return a.ReadyAt.Before(z.ReadyAt)
			}
			if b.Tasks[a.EpicKey].Priority != b.Tasks[z.EpicKey].Priority {
				return b.Tasks[a.EpicKey].Priority < b.Tasks[z.EpicKey].Priority
			}
			return numericKey(a.Key) < numericKey(z.Key)
		})
		if len(candidates) == 0 {
			if in.TaskKey != "" {
				if t := b.Tasks[in.TaskKey]; t != nil && t.Role == sess.Role && t.State == "ready" {
					if err := eligibleClaim(b, t, sess, now); err != nil {
						return core.Claim{}, err
					}
				}
			}
			return core.Claim{ConfigVersion: b.Config.ConfigVersion, Queue: sessionQueueSummary(b, sess, now)}, nil
		}
		t := b.editTask(candidates[0].Key)
		t.Warnings = nil
		if err := claimRules(b, t, sess, now); err != nil {
			return core.Claim{}, err
		}
		if t.Review != nil && t.Review.TargetGeneration == 0 {
			t.Review.TargetGeneration = b.Tasks[t.Review.TargetKey].Generation
		}
		if t.CreatedAt != nil && t.FirstClaimAt == nil {
			t.FirstClaimAt = &now
		}
		t.Generation++
		t.Version++
		t.State = "in_progress"
		t.Lease = &core.Lease{SessionID: a.SessionID, Generation: t.Generation, ExpiresAt: now.Add(b.Config.LeaseTTL)}
		t.Attempts = append(t.Attempts, core.Attempt{SessionID: a.SessionID, PrincipalID: a.Principal.ID, RunID: sess.RunID, Machine: sess.Machine, Generation: t.Generation, StartedAt: now})
		emit(b, now, "claimed", t.Key, a.SessionID)
		emitWarnings(b, t, now, a.SessionID)
		v := viewTask(b, t)
		return core.Claim{ConfigVersion: b.Config.ConfigVersion, Task: &v, Lease: t.Lease, EpicNotes: scopeNotes(b, "epic", t.EpicKey), Queue: sessionQueueSummary(b, sess, now)}, nil
	})
}
func (s *Store) Update(ctx context.Context, a core.Actor, in core.UpdateInput) (core.Task, error) {
	return mutate(s, ctx, a, in.RequestID, "update", in, func(b *board, sess *core.Session, now time.Time) (core.Task, error) {
		t, err := taskFor(b, in.TaskKey)
		if err != nil {
			return core.Task{}, err
		}
		if t.Lease != nil || in.Generation != 0 {
			if _, err = owned(b, a, t.Key, in.Generation); err != nil {
				return core.Task{}, err
			}
		} else if sess.Role != "coordinator" && t.Requester != a.Principal.ID {
			return core.Task{}, core.Fail("forbidden", "only requester or coordinator may update unclaimed work")
		}
		t.Warnings = nil
		t.Similar = nil
		if in.Priority != nil {
			if *in.Priority < 1 || *in.Priority > 100 {
				return core.Task{}, core.Fail("VALIDATION", "priority must be 1..100")
			}
			t.Priority = *in.Priority
		}
		if in.Areas != nil {
			t.Areas = append([]string{}, (*in.Areas)...)
			if err := prepareAreas(b, t); err != nil {
				return core.Task{}, err
			}
			if t.Type == "task" {
				if t.Lease != nil {
					if err := areaRoleAllowed(b, t.Areas, sess.Role); err != nil {
						return core.Task{}, err
					}
				}
				if err := checkAreaOverlap(b, t, now); err != nil {
					return core.Task{}, err
				}
			}
		}
		if in.Title != nil {
			if !validTitle(*in.Title) {
				return core.Task{}, core.Fail("invalid_input", "invalid title")
			}
			t.Title = *in.Title
		}
		if in.Description != nil {
			if len(*in.Description) > 16384 {
				return core.Task{}, core.Fail("invalid_input", "description too long")
			}
			t.Description = *in.Description
		}
		if in.Category != nil {
			t.Category = *in.Category
		}
		if in.Complexity != nil {
			t.Complexity = *in.Complexity
		}
		if in.ExternalID != nil {
			t.ExternalID = *in.ExternalID
		}
		if err := checkDuplicate(b, t, sess, in.Force, in.Reason); err != nil {
			return core.Task{}, err
		}
		if err := validateMetadata(b, t); err != nil {
			return core.Task{}, err
		}
		t.Version++
		renew(b, t, now)
		emit(b, now, "task_updated", t.Key, a.SessionID)
		emitWarnings(b, t, now, a.SessionID)
		return viewTask(b, t), nil
	})
}

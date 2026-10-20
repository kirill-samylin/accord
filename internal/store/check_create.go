package store

import (
	"agent-board/internal/core"
	"context"
	"fmt"
	"time"
)

// CheckCreate captures the target and installs the QA prerequisite atomically.
// It deliberately shares the generic constructor and authoritative gate evaluator.
func (s *Store) CheckCreate(ctx context.Context, a core.Actor, in core.CheckCreateInput) (core.Task, error) {
	return mutate(s, ctx, a, in.RequestID, "check_create", in, func(b *board, sess *core.Session, now time.Time) (core.Task, error) {
		if sess.Role != "human" && sess.Role != "coordinator" {
			return core.Task{}, core.Fail("forbidden", "Создание проверки доступно координатору или человеку")
		}
		target, err := taskFor(b, in.TargetKey)
		if err != nil {
			return core.Task{}, err
		}
		if target.Type != "task" || target.Role != "developer" || target.State != "done" || !validCommit(target.Evidence.Commit) {
			return core.Task{}, core.Fail("conflict", "Нужен завершённый результат разработчика с полным SHA")
		}
		if in.TargetVersion < 1 || in.TargetVersion != target.Version {
			return core.Task{}, core.Fail("conflict", "Проверяемая задача изменилась")
		}
		title := "Ревью " + target.Key
		var reviewer *core.Task
		switch in.Role {
		case "reviewer":
		case "tester":
			title = "QA " + target.Key
			reviewer = latestVerification(b, target.Key, "reviewer")
			if reviewer == nil || !successful(b, reviewer, "approved") {
				return core.Task{}, core.Fail("rule_violation", "Для QA требуется актуальное положительное ревью")
			}
		default:
			return core.Task{}, core.Fail("validation", "Выберите reviewer или tester")
		}
		task, err := s.createInBoard(b, sess, a, core.CreateInput{RequestID: in.RequestID, Type: "task", EpicKey: target.EpicKey, Title: title, Description: fmt.Sprintf("Проверить результат %s: %s\nCommit: %s\nGeneration: %d", target.Key, target.Title, target.Evidence.Commit, target.Generation), Role: in.Role, Priority: target.Priority, TargetKey: target.Key, TargetCommit: target.Evidence.Commit, ProfileID: in.ProfileID, ProfileVersion: in.ProfileVersion}, now)
		if err != nil {
			return core.Task{}, err
		}
		if reviewer != nil {
			created := b.Tasks[task.Key]
			created.Links = append(created.Links, core.Link{From: reviewer.Key, To: created.Key, Type: "blocks", Condition: "approved"})
			emit(b, now, "linked", created.Key, a.SessionID)
			task = viewTask(b, created)
		}
		emit(b, now, "check_created", task.Key, a.SessionID)
		b.Events[len(b.Events)-1].Data = &core.EventData{CheckCreated: &core.CheckCreated{TargetKey: target.Key, TargetCommit: target.Evidence.Commit, TargetGeneration: target.Generation, CheckKey: task.Key}}
		return task, nil
	})
}

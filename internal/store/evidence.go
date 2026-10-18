package store

import (
	"agent-board/internal/core"
	"encoding/hex"
	"strings"
	"time"
)

func validCommit(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}
func finishAttempt(t *core.Task, now time.Time, reason, branch string) {
	if len(t.Attempts) > 0 {
		v := &t.Attempts[len(t.Attempts)-1]
		if v.EndedAt == nil {
			v.EndedAt = &now
			v.Reason = reason
			if branch != "" {
				v.Branch = branch
			}
			v.Commit = t.Evidence.Commit
		}
	}
}
func completion(b *board, t *core.Task, sess *core.Session, in core.TransitionInput) error {
	if in.Evidence == nil {
		return core.Fail("evidence_missing", "completion evidence required")
	}
	e := *in.Evidence
	if len(e.RunSummary) > 1024 || len(e.Summary) > 4096 || len(e.Findings) > 16384 || len(e.Answer) > 16384 || len(e.Artifact) > 32 {
		return core.Fail("invalid_input", "evidence exceeds limits")
	}
	for _, artifact := range e.Artifact {
		if !bounded(artifact, 2048) {
			return core.Fail("invalid_input", "invalid artifact reference")
		}
	}
	switch t.Role {
	case "developer":
		if !validCommit(e.Commit) || !bounded(e.RunSummary, 1024) {
			return core.Fail("evidence_missing", "developer requires commit and run_summary")
		}
	case "analyst", "architect":
		if len(e.Artifact) == 0 || !bounded(e.Summary, 4096) {
			return core.Fail("evidence_missing", "artifact and summary required")
		}
	case "human":
		if !bounded(e.Answer, 16384) {
			return core.Fail("evidence_missing", "answer required")
		}
	case "coordinator":
		if !bounded(e.Summary, 4096) {
			return core.Fail("evidence_missing", "summary required")
		}
	case "reviewer", "tester":
		if t.Review == nil {
			return core.Fail("evidence_missing", "verification target required")
		}
		target := b.Tasks[t.Review.TargetKey]
		if target == nil || target.State != "done" || e.VerifiedCommit != t.Review.TargetCommit || target.Evidence.Commit != t.Review.TargetCommit || target.Generation != t.Review.TargetGeneration {
			return core.Fail("rule_violation", "verification must reference current completed developer commit")
		}
		if target.AuthorSession == sess.SessionID {
			return core.Fail("rule_violation", "review must use a separate server session")
		}
		for _, attempt := range target.Attempts {
			if attempt.SessionID == sess.SessionID && (attempt.Commit == e.VerifiedCommit || attempt.Commit == "") {
				return core.Fail("rule_violation", "author attempt cannot review its own result")
			}
		}
		if t.Role == "reviewer" {
			if in.Outcome != "approve" && in.Outcome != "changes_requested" && in.Outcome != "blocked" {
				return core.Fail("evidence_missing", "review outcome required")
			}
			if strings.TrimSpace(e.Findings) == "" {
				return core.Fail("evidence_missing", "findings required")
			}
		}
		if t.Role == "tester" {
			if in.Outcome != "passed" && in.Outcome != "failed" && in.Outcome != "not_run" && in.Outcome != "not_applicable" {
				return core.Fail("evidence_missing", "test outcome required")
			}
			if !bounded(e.RunSummary, 1024) {
				return core.Fail("evidence_missing", "run_summary required")
			}
		}
		t.Review.Outcome = in.Outcome
		t.Review.AuthorSession = target.AuthorSession
	}
	t.Evidence = e
	return nil
}

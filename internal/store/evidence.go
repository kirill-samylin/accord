package store

import (
	"encoding/hex"
	"encoding/json"
	"time"

	"agent-board/internal/core"
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
			v.Commit = t.Evidence.Text("commit")
		}
	}
}
func completion(b *board, t *core.Task, sess *core.Session, in core.TransitionInput) error {
	if in.Evidence == nil {
		return core.Fail("EVIDENCE_MISSING", "completion evidence required")
	}
	e := core.Evidence{}
	for key, value := range *in.Evidence {
		e[key] = append(json.RawMessage(nil), value...)
	}
	if verifier(b, t.Role) {
		field := outcomeField(b, t.Role)
		outcome, err := e.String(field)
		if err != nil {
			return core.Fail("VALIDATION", err.Error())
		}
		if in.Outcome != "" {
			if outcome != "" && outcome != in.Outcome {
				return core.Fail("VALIDATION", "outcome and evidence verdict/result disagree")
			}
			outcome = in.Outcome
			e[field], _ = json.Marshal(outcome)
		}
	}
	raw, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if b.Config.Snapshot == nil {
		return core.Fail("CONFIG_INTERNAL", "active snapshot required")
	}
	validation := b.Config.Snapshot.ValidateEvidence(t.Role, raw)
	if !validation.Valid {
		p := validation.Errors[0]
		code := p.Code
		if code == "EVIDENCE_REQUIRED" {
			code = "EVIDENCE_MISSING"
		}
		return core.Fail(code, p.Path+": "+p.Message)
	}
	if err := humanTaskAllowed(b, t, sess.Owner); err != nil {
		return err
	}
	if verifier(b, t.Role) {
		if t.Review == nil {
			return core.Fail("EVIDENCE_MISSING", "verification target required")
		}
		target := b.Tasks[t.Review.TargetKey]
		verified := e.Text("verified_commit")
		if target == nil || target.State != "done" || verified != t.Review.TargetCommit || target.Evidence.Text("commit") != t.Review.TargetCommit || target.Generation != t.Review.TargetGeneration {
			return core.Fail("RULE_VIOLATION", "verification must reference current completed target commit")
		}
		if target.AuthorSession == sess.SessionID {
			return core.Fail("RULE_VIOLATION", "review must use a separate server session")
		}
		for _, attempt := range target.Attempts {
			if attempt.SessionID == sess.SessionID && (attempt.Commit == verified || attempt.Commit == "") {
				return core.Fail("RULE_VIOLATION", "author attempt cannot review its own result")
			}
		}
		t.Review.OutcomeField = outcomeField(b, t.Role)
		t.Review.Outcome = e.Text(t.Review.OutcomeField)
		t.Review.AuthorSession = target.AuthorSession
	}
	t.Evidence = e
	t.EvidenceComplete = &validation.Complete
	for _, warning := range validation.Warnings {
		ruleWarning(b, t, "evidence_required", warning.Message, sess.SessionID)
	}
	return nil
}

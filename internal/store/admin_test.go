package store

import (
	"agent-board/internal/core"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestTokenAdministrationAuditsAndSafeViews(t *testing.T) {
	f := newFixture(t)
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "bob", Machine: "pc-b", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	listed, err := f.s.ListTokens(testContext)
	if err != nil || len(listed) != 2 {
		t.Fatalf("list tokens: %v; %d", err, len(listed))
	}
	raw, err := json.Marshal(listed)
	if err != nil || strings.Contains(string(raw), token.Secret) || strings.Contains(string(raw), tokenHash(token.Secret)) || strings.Contains(string(raw), `"hash"`) {
		t.Fatal("token listing exposed credential material")
	}
	if err = f.s.RevokeToken(testContext, token.ID); err != nil {
		t.Fatal(err)
	}
	if err = f.s.RevokeToken(testContext, token.ID); err != nil {
		t.Fatal("repeat revoke must be idempotent", err)
	}
	var creates, revokes int
	if err = f.s.db.QueryRow(`SELECT COUNT(*) FROM events WHERE json_extract(data,'$.type')='token_created' AND json_extract(data,'$.token_id')=?`, token.ID).Scan(&creates); err != nil {
		t.Fatal(err)
	}
	if err = f.s.db.QueryRow(`SELECT COUNT(*) FROM events WHERE json_extract(data,'$.type')='token_revoked' AND json_extract(data,'$.token_id')=?`, token.ID).Scan(&revokes); err != nil || creates != 1 || revokes != 1 {
		t.Fatalf("audit creates=%d revokes=%d err=%v", creates, revokes, err)
	}
	sessions, err := f.s.ListSessions(testContext)
	if err != nil || len(sessions) != 1 || sessions[0].Owner != "alice" || sessions[0].Role != "coordinator" {
		t.Fatalf("session view: %+v %v", sessions, err)
	}
	if _, err = f.s.RevokeAllTokens(testContext, ""); err == nil {
		t.Fatal("empty revocation reason accepted")
	}
	count, err := f.s.RevokeAllTokens(testContext, "rotation")
	if err != nil || count != 1 {
		t.Fatalf("revoke all %d %v", count, err)
	}
	_, err = f.s.Create(testContext, f.coord, core.CreateInput{RequestID: "epic", Type: "epic", Title: "Pilot"})
	requireCode(t, err, "UNAUTHORIZED") // Authentication precedes replay.
}

func TestTokenAuditFailureRollsBackCredentialChange(t *testing.T) {
	f := newFixture(t)
	if _, err := f.s.db.Exec(`CREATE TRIGGER reject_admin_audit BEFORE INSERT ON events BEGIN SELECT RAISE(ABORT, 'audit unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "bob", Machine: "pc-b", TTL: time.Hour}); err == nil {
		t.Fatal("create succeeded without audit")
	}
	if err := f.s.RevokeToken(testContext, f.p.ID); err == nil {
		t.Fatal("revoke succeeded without audit")
	}
	if _, err := f.s.RevokeAllTokens(testContext, "rotation"); err == nil {
		t.Fatal("revoke-all succeeded without audit")
	}
	listed, err := f.s.ListTokens(testContext)
	if err != nil || len(listed) != 1 || listed[0].Revoked {
		t.Fatalf("credential mutation escaped rollback: %+v %v", listed, err)
	}
}

func TestRecoveryInvalidatesRestoredIdentityAndPreservesHistory(t *testing.T) {
	f := newFixture(t)
	worker := f.actor(t, "developer")
	task := f.task(t, "developer")
	claim := f.claim(t, worker, task.Key)
	architect := f.actor(t, "architect")
	var beforeReplay int
	if err := f.s.db.QueryRow(`SELECT COUNT(*) FROM idempotency`).Scan(&beforeReplay); err != nil {
		t.Fatal(err)
	}
	// Recovery labels an overdue restored attempt as restore, not ordinary expiry.
	f.clock.advance(2 * time.Hour)
	result, err := f.s.RecoverState(testContext, "restore")
	if err != nil || result.RevokedTokens != 1 || result.Leases != 1 || result.Sessions != 3 {
		t.Fatalf("recovery: %+v %v", result, err)
	}
	var saved string
	if err = f.s.db.QueryRow(`SELECT data FROM tasks WHERE key=?`, task.Key).Scan(&saved); err != nil {
		t.Fatal(err)
	}
	var restored core.Task
	if err = json.Unmarshal([]byte(saved), &restored); err != nil {
		t.Fatal(err)
	}
	if restored.Lease != nil || restored.State != "ready" || restored.Generation != claim.Lease.Generation || len(restored.Attempts) != 1 || restored.Attempts[0].Reason != "restore" || restored.Attempts[0].EndedAt == nil {
		t.Fatalf("incorrect recovered task: %+v", restored)
	}
	var afterReplay, generation int
	if err = f.s.db.QueryRow(`SELECT COUNT(*) FROM idempotency`).Scan(&afterReplay); err != nil || afterReplay != beforeReplay {
		t.Fatalf("replay history changed: %d->%d %v", beforeReplay, afterReplay, err)
	}
	if err = f.s.db.QueryRow(`SELECT generation FROM role_generations WHERE role='architect'`).Scan(&generation); err != nil || generation != 1 {
		t.Fatalf("role generation changed: %d %v", generation, err)
	}
	_, err = f.s.Next(testContext, worker, core.NextInput{RequestID: "stale", TaskKey: task.Key})
	requireCode(t, err, "UNAUTHORIZED")
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: f.p.Owner, Machine: f.p.Machine, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	f.p, err = f.s.Authenticate(testContext, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.s.Heartbeat(testContext, core.Actor{Principal: f.p, SessionID: architect.SessionID}, core.HeartbeatInput{RequestID: "stale-session", RoleGeneration: 1})
	requireCode(t, err, "UNAUTHORIZED")
	fresh := f.actor(t, "developer")
	newClaim := f.claim(t, fresh, task.Key)
	if newClaim.Lease.Generation <= claim.Lease.Generation {
		t.Fatal("recovery decreased fencing generation")
	}
	newArchitect := f.actor(t, "architect")
	var newGeneration int
	if err = f.s.db.QueryRow(`SELECT generation FROM role_generations WHERE role='architect'`).Scan(&newGeneration); err != nil || newGeneration != 2 || newArchitect.SessionID == architect.SessionID {
		t.Fatalf("architect fencing did not advance after recovery: %d %v", newGeneration, err)
	}
}

func TestRecoveryAuditFailureRollsBackAllState(t *testing.T) {
	f := newFixture(t)
	worker := f.actor(t, "developer")
	task := f.task(t, "developer")
	f.claim(t, worker, task.Key)
	if _, err := f.s.db.Exec(`CREATE TRIGGER reject_recovery_audit BEFORE INSERT ON events BEGIN SELECT RAISE(ABORT, 'audit unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.RecoverState(testContext, "restore"); err == nil {
		t.Fatal("recovery succeeded without audit")
	}
	listed, err := f.s.ListTokens(testContext)
	if err != nil || listed[0].Revoked {
		t.Fatal("recovery revoked token despite rollback", err)
	}
	var raw string
	if err = f.s.db.QueryRow(`SELECT data FROM tasks WHERE key=?`, task.Key).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	var saved core.Task
	if err = json.Unmarshal([]byte(raw), &saved); err != nil || saved.Lease == nil || saved.Attempts[0].EndedAt != nil {
		t.Fatal("recovery changed lease/attempt despite rollback", err)
	}
	var sessionRaw string
	if err = f.s.db.QueryRow(`SELECT data FROM sessions WHERE key=?`, worker.SessionID).Scan(&sessionRaw); err != nil {
		t.Fatal(err)
	}
	var session core.Session
	if err = json.Unmarshal([]byte(sessionRaw), &session); err != nil || !session.ExpiresAt.After(f.clock.read()) {
		t.Fatal("recovery changed session despite rollback", err)
	}
}

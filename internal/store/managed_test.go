package store

import (
	"agent-board/internal/core"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const managedTestSecret = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
const managedTestSession = "12345678-1234-1234-1234-123456789abc"

type managedFixture struct {
	f *fixture
	a core.Actor
	p core.AgentProfile
	h core.ManagedHost
}

func managedSetup(t *testing.T) *managedFixture {
	t.Helper()
	f := newFixture(t)
	a := f.actor(t, "human")
	p := createProfile(t, f)
	h, err := f.s.ManagedHostEnroll(testContext, a, core.ManagedHostEnrollInput{RequestID: f.id(), Name: "Local", HostSecret: managedTestSecret, Workspaces: []string{"gui"}, Providers: []string{"codex"}, Controllers: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	return &managedFixture{f, a, p, h}
}
func (m *managedFixture) start(t *testing.T) core.ManagedResult {
	t.Helper()
	r, err := m.f.s.ManagedStart(testContext, m.a, core.ManagedStartInput{RequestID: m.f.id(), HostID: m.h.ID, Workspace: "gui", Provider: "codex", ProfileID: m.p.ID, ProfileVersion: m.p.Version, Prompt: "Review only"})
	if err != nil {
		t.Fatal(err)
	}
	return r
}
func (m *managedFixture) poll(t *testing.T) core.ManagedPollResult {
	t.Helper()
	p, err := m.f.s.ManagedPoll(testContext, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "worker-a"})
	if err != nil {
		t.Fatal(err)
	}
	return p
}
func (m *managedFixture) report(t *testing.T, p core.ManagedPollResult, phase string) core.ManagedResult {
	t.Helper()
	in := m.reportInput(p, phase)
	r, err := m.f.s.ManagedReport(testContext, m.a, in)
	if err != nil {
		t.Fatal(phase, err)
	}
	return r
}
func (m *managedFixture) reportInput(p core.ManagedPollResult, phase string) core.ManagedReportInput {
	return core.ManagedReportInput{ManagedPollInput: core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "worker-a", WorkerEpoch: p.WorkerEpoch}, CommandID: p.Command.ID, DispatchID: p.Command.DispatchID, RunID: p.Run.ID, Generation: p.Command.Generation, Phase: phase, ProviderSessionID: managedTestSession}
}

func TestManagedExactDispatchRetryProofAndNoDoubleStart(t *testing.T) {
	m := managedSetup(t)
	first := m.start(t)
	_, err := m.f.s.ManagedStart(testContext, m.a, core.ManagedStartInput{RequestID: m.f.id(), HostID: m.h.ID, Workspace: "gui", Provider: "codex", ProfileID: m.p.ID, ProfileVersion: 1, Prompt: "Other"})
	requireCode(t, err, "CONFLICT")
	p := m.poll(t)
	repeated := m.poll(t)
	if p.Command == nil || p.Run.ID != first.Run.ID || p.Command.DispatchID != repeated.Command.DispatchID || p.Run.Generation != 1 {
		t.Fatal("redispatched as new start")
	}
	receipt := m.reportInput(p, "accepted")
	if _, err = m.f.s.ManagedReport(testContext, m.a, receipt); err != nil {
		t.Fatal(err)
	}
	if _, err = m.f.s.ManagedReport(testContext, m.a, receipt); err != nil {
		t.Fatal("receipt replay", err)
	}
	receipt.HostSecret = strings.Repeat("0", 64)
	_, err = m.f.s.ManagedReport(testContext, m.a, receipt)
	requireCode(t, err, "UNAUTHORIZED")
	started := m.report(t, p, "started")
	if started.Run.ProviderSessionID != managedTestSession || started.Command.Status != "applied" {
		t.Fatal("missing provider receipt")
	}
	if next := m.poll(t); next.Command != nil {
		t.Fatal("applied start redelivered")
	}
	var raw string
	if err = m.f.s.db.QueryRow(`SELECT data FROM managed_hosts WHERE key=?`, m.h.ID).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, managedTestSecret) {
		t.Fatal("secret persisted")
	}
	rows, err := m.f.s.db.Query(`SELECT response FROM idempotency`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&raw)
		if strings.Contains(raw, managedTestSecret) {
			t.Fatal("replay leaked secret")
		}
	}
}
func TestManagedStopPriorityReceiptForcedAndExactResume(t *testing.T) {
	m := managedSetup(t)
	m.start(t)
	p := m.poll(t)
	m.report(t, p, "accepted")
	live := m.report(t, p, "started")
	stop, err := m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: live.Run.ID, Version: live.Run.Version, Kind: "stop"})
	if err != nil {
		t.Fatal(err)
	}
	if stop.Run.State != "stopping" {
		t.Fatal("not stopping")
	}
	q := m.poll(t)
	if q.Command == nil || q.Command.Kind != "stop" {
		t.Fatal("stop blocked behind running command")
	}
	m.report(t, q, "accepted")
	r := m.reportInput(q, "stopped")
	r.Forced = true
	stopped, err := m.f.s.ManagedReport(testContext, m.a, r)
	if err != nil || !stopped.Command.Forced || stopped.Run.State != "stopped" {
		t.Fatal("stop not proven", err)
	}
	resume, err := m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: live.Run.ID, Version: stopped.Run.Version, Kind: "resume"})
	if err != nil || resume.Run.Generation != 2 {
		t.Fatal("resume", err)
	}
	again := m.poll(t)
	if again.Run.ProviderSessionID != managedTestSession || again.Command.Kind != "resume" {
		t.Fatal("native session lost")
	}
	m.report(t, again, "accepted")
	wrong := m.reportInput(again, "started")
	wrong.ProviderSessionID = "aaaaaaaa-1234-1234-1234-123456789abc"
	_, err = m.f.s.ManagedReport(testContext, m.a, wrong)
	requireCode(t, err, "CONFLICT")
	m.report(t, again, "started")
}
func TestManagedClarificationBoundaryStoppedMailboxAndRecoveryFencing(t *testing.T) {
	m := managedSetup(t)
	m.start(t)
	p := m.poll(t)
	m.report(t, p, "accepted")
	live := m.report(t, p, "started")
	clarify, err := m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: live.Run.ID, Version: live.Run.Version, Kind: "clarify", Text: "Check Windows"})
	if err != nil || clarify.Command.Status != "queued" {
		t.Fatal(err)
	}
	if q := m.poll(t); q.Command != nil {
		t.Fatal("clarification interrupted live CLI")
	}
	m.f.clock.advance(time.Millisecond)
	m.report(t, p, "completed")
	q := m.poll(t)
	if q.Command == nil || q.Command.Kind != "clarify" || q.Command.Generation != 2 {
		t.Fatal("queued message not dispatched after turn", q)
	}
	m.report(t, q, "accepted")
	m.report(t, q, "started")
	m.f.clock.advance(31 * time.Second)
	_, err = m.f.s.ManagedPoll(testContext, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "worker-b"})
	requireCode(t, err, "MANAGED_UNCERTAIN")
	_, err = m.f.s.ManagedReport(testContext, m.a, m.reportInput(q, "completed"))
	requireCode(t, err, "CONFLICT")
	unknown := m.reportInput(q, "delivery_unknown")
	unknown.WorkerID = "worker-b"
	r, err := m.f.s.ManagedReport(testContext, m.a, unknown)
	if err != nil || r.Run.State != "needs_recovery" {
		t.Fatal("ambiguous state", err)
	}
	_, err = m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: r.Run.ID, Version: r.Run.Version, Kind: "resume"})
	requireCode(t, err, "CONFLICT")
}
func TestManagedCurrentControllerBeforeCachedMutationAndPendingDispatch(t *testing.T) {
	m := managedSetup(t)
	cred, err := m.f.s.CreateToken(testContext, core.TokenInput{Owner: "second", Machine: "pc-b", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	principal, err := m.f.s.Authenticate(testContext, cred.Secret)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := m.f.s.Describe(testContext, principal, core.OpenSession{RequestID: m.f.id(), Role: "human", RunID: "second", Machine: principal.Machine})
	if err != nil {
		t.Fatal(err)
	}
	second := core.Actor{Principal: principal, SessionID: sess.SessionID}
	enroll := core.ManagedHostEnrollInput{RequestID: m.f.id(), Name: m.h.Name, HostSecret: managedTestSecret, Controllers: []string{principal.ID}, Workspaces: m.h.Workspaces, Providers: m.h.Providers}
	m.h, err = m.f.s.ManagedHostUpdate(testContext, m.a, core.ManagedHostUpdateInput{ManagedHostEnrollInput: enroll, HostID: m.h.ID, Version: m.h.Version, State: "active"})
	if err != nil {
		t.Fatal(err)
	}
	in := core.ManagedStartInput{RequestID: m.f.id(), HostID: m.h.ID, Workspace: "gui", Provider: "codex", ProfileID: m.p.ID, ProfileVersion: 1, Prompt: "Allowed"}
	_, err = m.f.s.ManagedStart(testContext, second, in)
	if err != nil {
		t.Fatal(err)
	}
	enroll.RequestID = m.f.id()
	enroll.Controllers = []string{}
	m.h, err = m.f.s.ManagedHostUpdate(testContext, m.a, core.ManagedHostUpdateInput{ManagedHostEnrollInput: enroll, HostID: m.h.ID, Version: m.h.Version, State: "active"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.f.s.ManagedStart(testContext, second, in)
	requireCode(t, err, "FORBIDDEN_ROLE")
	if q := m.poll(t); q.Command != nil {
		t.Fatal("revoked queued actor dispatched")
	}
}
func TestManagedRestoreDisablesWorkersPreservesHistoryAndNoAutomaticResume(t *testing.T) {
	m := managedSetup(t)
	started := m.start(t)
	m.poll(t)
	if _, err := m.f.s.RecoverState(testContext, "restore"); err != nil {
		t.Fatal(err)
	}
	var host managedHostRecord
	var raw string
	m.f.s.db.QueryRow(`SELECT data FROM managed_hosts WHERE key=?`, m.h.ID).Scan(&raw)
	json.Unmarshal([]byte(raw), &host)
	if host.SecretHash != "" || host.Host.State != "disabled" {
		t.Fatal("restored host can execute")
	}
	var run core.ManagedRun
	m.f.s.db.QueryRow(`SELECT data FROM managed_runs WHERE key=?`, started.Run.ID).Scan(&raw)
	json.Unmarshal([]byte(raw), &run)
	if run.State != "needs_recovery" || run.ProviderSessionID != "" {
		t.Fatal("restore resumed run")
	}
}

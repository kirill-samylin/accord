package store

import (
	"agent-board/internal/core"
	"testing"
	"time"
)

// T022-R2 contract: managed report/poll refusals carry explicit codes so the
// worker never classifies them by message text. All map to HTTP 409 except
// RUN_ARCHIVED/NOT_FOUND (404); LEASE_HELD is the transient, retryable one.
func TestManagedReportAndPollRefusalsCarryExplicitCodes(t *testing.T) {
	m := managedSetup(t)
	m.start(t)
	p := m.poll(t)
	m.report(t, p, "accepted")
	m.report(t, p, "started")

	report := func(mutate func(*core.ManagedReportInput)) error {
		in := m.reportInput(p, "completed")
		mutate(&in)
		_, err := m.f.s.ManagedReport(testContext, m.a, in)
		return err
	}
	requireCode(t, report(func(in *core.ManagedReportInput) { in.DispatchID = "md_not_current" }), "STALE_DISPATCH")
	requireCode(t, report(func(in *core.ManagedReportInput) { in.Generation++ }), "STALE_GENERATION")
	requireCode(t, report(func(in *core.ManagedReportInput) { in.RunID = "mr_never_existed" }), "NOT_FOUND")
	if _, err := m.f.s.db.Exec(`INSERT INTO managed_runs VALUES(?,?)`, automationArchivePrefix+"mr_rotated", `{"id":"mr_rotated","state":"stopped"}`); err != nil {
		t.Fatal(err)
	}
	requireCode(t, report(func(in *core.ManagedReportInput) { in.RunID = "mr_rotated" }), "RUN_ARCHIVED")

	_, err := m.f.s.ManagedPoll(testContext, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "worker-b"})
	requireCode(t, err, "LEASE_HELD")
	requireCode(t, report(func(in *core.ManagedReportInput) { in.Phase, in.WorkerID = "recovered_stopped", "worker-b" }), "LEASE_HELD")

	m.f.clock.advance(31 * time.Second)
	_, err = m.f.s.ManagedPoll(testContext, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "worker-b"})
	requireCode(t, err, "MANAGED_UNCERTAIN")
	_, err = m.f.s.ManagedPoll(testContext, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "worker-a", WorkerEpoch: p.WorkerEpoch})
	requireCode(t, err, "MANAGED_UNCERTAIN")

	terminal := managedSetup(t)
	terminal.start(t)
	q := terminal.poll(t)
	terminal.report(t, q, "accepted")
	terminal.report(t, q, "failed")
	_, err = terminal.f.s.ManagedReport(testContext, terminal.a, terminal.reportInput(q, "failed"))
	requireCode(t, err, "COMMAND_TERMINAL")
}

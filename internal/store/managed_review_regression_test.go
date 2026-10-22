package store

import (
	"agent-board/internal/core"
	"testing"
	"time"
)

func TestManagedReportReplayAfterExpiryAndLaterGenerationDoesNotRenewOrReapply(t *testing.T) {
	m := managedSetup(t)
	m.start(t)
	p := m.poll(t)
	m.report(t, p, "accepted")
	m.report(t, p, "started")
	doneInput := m.reportInput(p, "completed")
	old, err := m.f.s.ManagedReport(testContext, m.a, doneInput)
	if err != nil {
		t.Fatal(err)
	}
	m.f.clock.advance(31 * time.Second)
	replayed, err := m.f.s.ManagedReport(testContext, m.a, doneInput)
	if err != nil || replayed.Run.Version != old.Run.Version {
		t.Fatal("lost terminal receipt not recoverable", err)
	}
	resume, err := m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: old.Run.ID, Version: old.Run.Version, Kind: "resume"})
	if err != nil {
		t.Fatal(err)
	}
	q := m.poll(t)
	m.report(t, q, "accepted")
	m.report(t, q, "started")
	replayed, err = m.f.s.ManagedReport(testContext, m.a, doneInput)
	if err != nil || replayed.Run.Generation != 1 {
		t.Fatal("exact old result lost", err)
	}
	detail, err := m.f.s.ManagedRun(testContext, m.a, core.ManagedRunInput{RunID: old.Run.ID})
	if err != nil || detail.Run.Generation != resume.Run.Generation || detail.Run.State != "running" {
		t.Fatal("old replay rewrote new generation", err)
	}
	recovery := doneInput
	recovery.RequestID = m.f.id()
	recovery.Phase = "recovered_stopped"
	_, err = m.f.s.ManagedReport(testContext, m.a, recovery)
	requireCode(t, err, "CONFLICT")
}
func TestManagedExpiredSameWorkerLeaseCannotRenewActiveProcess(t *testing.T) {
	m := managedSetup(t)
	m.start(t)
	p := m.poll(t)
	m.report(t, p, "accepted")
	m.report(t, p, "started")
	m.f.clock.advance(31 * time.Second)
	_, err := m.f.s.ManagedPoll(testContext, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: "worker-a", WorkerEpoch: p.WorkerEpoch})
	requireCode(t, err, "CONFLICT")
}
func TestManagedFailedRunRequiresRecoveryBeforeResume(t *testing.T) {
	m := managedSetup(t)
	m.start(t)
	p := m.poll(t)
	m.report(t, p, "accepted")
	m.report(t, p, "started")
	failed := m.report(t, p, "failed")
	_, err := m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: failed.Run.ID, Version: failed.Run.Version, Kind: "resume"})
	requireCode(t, err, "CONFLICT")
}
func TestManagedRevocationAfterDispatchDoesNotPretendProcessStopped(t *testing.T) {
	for _, phase := range []string{"dispatched", "accepted", "started", "started-with-stop"} {
		t.Run(phase, func(t *testing.T) {
			m := managedSetup(t)
			cred, err := m.f.s.CreateToken(testContext, core.TokenInput{Owner: "issuer", Machine: "other", TTL: time.Hour})
			if err != nil {
				t.Fatal(err)
			}
			p, err := m.f.s.Authenticate(testContext, cred.Secret)
			if err != nil {
				t.Fatal(err)
			}
			session, err := m.f.s.Describe(testContext, p, core.OpenSession{RequestID: m.f.id(), Role: "human", RunID: "issuer", Machine: p.Machine})
			if err != nil {
				t.Fatal(err)
			}
			issuer := core.Actor{Principal: p, SessionID: session.SessionID}
			grant := core.ManagedHostEnrollInput{RequestID: m.f.id(), Name: m.h.Name, HostSecret: managedTestSecret, Controllers: []string{p.ID}, Workspaces: m.h.Workspaces, Providers: m.h.Providers}
			m.h, err = m.f.s.ManagedHostUpdate(testContext, m.a, core.ManagedHostUpdateInput{ManagedHostEnrollInput: grant, HostID: m.h.ID, Version: m.h.Version, State: "active"})
			if err != nil {
				t.Fatal(err)
			}
			run, err := m.f.s.ManagedStart(testContext, issuer, core.ManagedStartInput{RequestID: m.f.id(), HostID: m.h.ID, Workspace: "gui", Provider: "codex", ProfileID: m.p.ID, ProfileVersion: 1, Prompt: "do"})
			if err != nil {
				t.Fatal(err)
			}
			dispatched := m.poll(t)
			if phase != "dispatched" {
				m.report(t, dispatched, "accepted")
			}
			if phase == "started" || phase == "started-with-stop" {
				m.report(t, dispatched, "started")
			}

			var priorStop core.ManagedPollResult
			if phase == "started-with-stop" {
				detail, e := m.f.s.ManagedRun(testContext, m.a, core.ManagedRunInput{RunID: run.Run.ID})
				if e != nil {
					t.Fatal(e)
				}
				_, e = m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: run.Run.ID, Version: detail.Run.Version, Kind: "stop"})
				if e != nil {
					t.Fatal(e)
				}
				priorStop = m.poll(t)
				m.report(t, priorStop, "accepted")
			}
			grant.RequestID = m.f.id()
			grant.Controllers = []string{}
			m.h, err = m.f.s.ManagedHostUpdate(testContext, m.a, core.ManagedHostUpdateInput{ManagedHostEnrollInput: grant, HostID: m.h.ID, Version: m.h.Version, State: "active"})
			if err != nil {
				t.Fatal(err)
			}
			stop := m.poll(t)
			if phase == "started-with-stop" {
				if stop.Command != nil {
					t.Fatal("duplicate safety stop despite accepted owner stop")
				}
				detail, e := m.f.s.ManagedRun(testContext, m.a, core.ManagedRunInput{RunID: run.Run.ID})
				if e != nil {
					t.Fatal(e)
				}
				count := 0
				for _, c := range detail.Commands {
					if c.Kind == "stop" {
						count++
					}
				}
				if count != 1 {
					t.Fatal("safety stop duplicated audit", count)
				}
				stop = priorStop
			}
			if stop.Run == nil || stop.Run.State == "stopped" || stop.Command == nil || stop.Command.Kind != "stop" {
				t.Fatal("dispatched revoke freed workspace without process proof")
			}
			_, err = m.f.s.ManagedReport(testContext, m.a, m.reportInput(dispatched, "accepted"))
			requireCode(t, err, "FORBIDDEN_ROLE")
			_, err = m.f.s.ManagedStart(testContext, m.a, core.ManagedStartInput{RequestID: m.f.id(), HostID: m.h.ID, Workspace: "gui", Provider: "codex", ProfileID: m.p.ID, ProfileVersion: 1, Prompt: "duplicate"})
			requireCode(t, err, "CONFLICT")
			if phase != "started-with-stop" {
				m.report(t, stop, "accepted")
			}
			stopped := m.report(t, stop, "stopped")
			if stopped.Run.ID != run.Run.ID || stopped.Run.State != "stopped" {
				t.Fatal("safety stop failed")
			}
		})
	}
}

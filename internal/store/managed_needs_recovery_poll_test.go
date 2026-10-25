package store

import (
	"agent-board/internal/core"
	"testing"
	"time"
)

// T022-R2-5: a needs_recovery run is settled only by `worker recover`. Its
// pending stop must not be re-delivered first on every poll, and a lease lapse
// or worker restart must not refuse the poll for other runs whose ownership is
// clear. Dispatched work of another run after a lapse stays refused.
func TestManagedNeedsRecoveryRunDoesNotStarveOrBlockOtherRuns(t *testing.T) {
	for _, restart := range []bool{false, true} {
		name := "lease_lapse_same_worker"
		if restart {
			name = "restarted_worker"
		}
		t.Run(name, func(t *testing.T) {
			m := managedSetup(t)
			grant := core.ManagedHostEnrollInput{RequestID: m.f.id(), Name: m.h.Name, HostSecret: managedTestSecret, Controllers: []string{}, Workspaces: []string{"gui", "api"}, Providers: m.h.Providers}
			var err error
			if m.h, err = m.f.s.ManagedHostUpdate(testContext, m.a, core.ManagedHostUpdateInput{ManagedHostEnrollInput: grant, HostID: m.h.ID, Version: m.h.Version, State: "active"}); err != nil {
				t.Fatal(err)
			}
			pollAs := func(worker string, epoch int64) (core.ManagedPollResult, error) {
				return m.f.s.ManagedPoll(testContext, m.a, core.ManagedPollInput{RequestID: m.f.id(), HostID: m.h.ID, HostSecret: managedTestSecret, WorkerID: worker, WorkerEpoch: epoch})
			}
			a := m.start(t)
			startA := m.poll(t)
			m.report(t, startA, "accepted")
			started := m.report(t, startA, "started")
			if _, err = m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: a.Run.ID, Version: started.Run.Version, Kind: "stop"}); err != nil {
				t.Fatal(err)
			}
			stopA := m.poll(t)
			if stopA.Command == nil || stopA.Command.Kind != "stop" {
				t.Fatal("stop not dispatched")
			}
			unknown := m.reportInput(startA, "delivery_unknown")
			if got, e := m.f.s.ManagedReport(testContext, m.a, unknown); e != nil || got.Run.State != "needs_recovery" {
				t.Fatal("delivery_unknown", e, got.Run.State)
			}
			if idle := m.poll(t); idle.Command != nil {
				t.Fatal("needs_recovery run re-delivered", idle.Command.Kind)
			}
			b, err := m.f.s.ManagedStart(testContext, m.a, core.ManagedStartInput{RequestID: m.f.id(), HostID: m.h.ID, Workspace: "api", Provider: "codex", ProfileID: m.p.ID, ProfileVersion: m.p.Version, Prompt: "Other"})
			if err != nil {
				t.Fatal(err)
			}
			m.f.clock.advance(31 * time.Second)
			worker, epoch := "worker-a", startA.WorkerEpoch
			if restart {
				worker, epoch = "worker-b", 0
			}
			startB, err := pollAs(worker, epoch)
			if err != nil {
				t.Fatal("poll refused for a clearly owned run while another run needs recovery", err)
			}
			if startB.Command == nil || startB.Command.RunID != b.Run.ID || startB.Command.Kind != "start" {
				t.Fatal("other run starved", startB.Command)
			}
			if restart {
				recovered := m.reportInput(startA, "recovered_stopped")
				recovered.WorkerID, recovered.WorkerEpoch = worker, startB.WorkerEpoch
				got, e := m.f.s.ManagedReport(testContext, m.a, recovered)
				if e != nil || got.Run.State != "stopped" {
					t.Fatal("recover after takeover", e, got.Run.State)
				}
				detail, e := m.f.s.ManagedRun(testContext, m.a, core.ManagedRunInput{RunID: a.Run.ID})
				if e != nil {
					t.Fatal(e)
				}
				for _, c := range detail.Commands {
					if c.ID == stopA.Command.ID && c.Status != "cancelled" {
						t.Fatal("recovered run kept its stop pending", c.Status)
					}
				}
				return
			}
			in := m.reportInput(startB, "accepted")
			if _, err = m.f.s.ManagedReport(testContext, m.a, in); err != nil {
				t.Fatal(err)
			}
			in.RequestID, in.Phase, in.ProviderSessionID = m.f.id(), "started", "22345678-1234-1234-1234-123456789abc"
			if _, err = m.f.s.ManagedReport(testContext, m.a, in); err != nil {
				t.Fatal(err)
			}
			m.f.clock.advance(31 * time.Second)
			_, err = pollAs("worker-b", 0)
			requireCode(t, err, "MANAGED_UNCERTAIN")
		})
	}
}

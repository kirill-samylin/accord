package store

import (
	"agent-board/internal/core"
	"strings"
	"testing"
)

func TestManagedProvenExitReleasesOnlyBoundAttemptAndResumesImmediately(t *testing.T) {
	for _, phase := range []string{"completed", "stopped", "failed", "recovered_stopped"} {
		t.Run(phase, func(t *testing.T) {
			m := managedSetup(t)
			settings := m.p.ProfileSettings
			settings.Role = "architect"
			profile, err := m.f.s.ProfileCreate(testContext, m.a, core.ProfileCreateInput{RequestID: m.f.id(), ProfileSettings: settings})
			if err != nil {
				t.Fatal(err)
			}
			m.p = profile
			task := assignedTask(t, m.f, profile)
			m.start(t)
			p := m.poll(t)
			m.report(t, p, "accepted")
			nativeView, err := m.f.s.Describe(testContext, m.f.p, managedDescribeInput(m, *p.Run))
			if err != nil {
				t.Fatal(err)
			}
			native := core.Actor{Principal: m.f.p, SessionID: nativeView.SessionID}
			claim := m.f.claim(t, native, task.Key)
			live := m.report(t, p, "started")
			terminal := p
			if phase == "stopped" {
				_, err = m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: live.Run.ID, Version: live.Run.Version, Kind: "stop"})
				if err != nil {
					t.Fatal(err)
				}
				terminal = m.poll(t)
				m.report(t, terminal, "accepted")
			}
			input := m.reportInput(terminal, phase)
			closed, err := m.f.s.ManagedReport(testContext, m.a, input)
			if err != nil {
				t.Fatal(err)
			}
			ready, err := m.f.s.Get(testContext, m.f.coord, core.GetInput{TaskKey: task.Key})
			if err != nil || ready.State != "ready" || ready.Lease != nil || ready.Generation != claim.Lease.Generation || ready.Attempts[0].EndedAt == nil || !strings.HasPrefix(ready.Attempts[0].Reason, "managed_"+phase+":") {
				t.Fatal("terminal proof did not release exact task attempt", err)
			}
			var expiry string
			if err = m.f.s.db.QueryRow(`SELECT json_extract(data,'$.expires_at') FROM sessions WHERE key=?`, native.SessionID).Scan(&expiry); err != nil || expiry != m.f.clock.read().UTC().Format("2006-01-02T15:04:05Z07:00") {
				t.Fatal("bound role session not expired", expiry, err)
			}
			_, err = m.f.s.Comment(testContext, native, core.CommentInput{LeaseInput: core.LeaseInput{RequestID: m.f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, Text: "stale"})
			requireCode(t, err, "LEASE_LOST")
			if phase == "failed" {
				closed = m.report(t, p, "recovered_stopped")
			}
			resume, err := m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: closed.Run.ID, Version: closed.Run.Version, Kind: "resume"})
			if err != nil {
				t.Fatal(err)
			}
			next := m.poll(t)
			m.report(t, next, "accepted")
			newView, err := m.f.s.Describe(testContext, m.f.p, managedDescribeInput(m, *next.Run))
			if err != nil || newView.RoleGeneration <= nativeView.RoleGeneration || newView.ManagedGeneration != resume.Run.Generation {
				t.Fatal("singleton role did not transfer to fresh managed attempt", err)
			}
			newActor := core.Actor{Principal: m.f.p, SessionID: newView.SessionID}
			newClaim := m.f.claim(t, newActor, task.Key)
			if newClaim.Lease.Generation <= claim.Lease.Generation {
				t.Fatal("new claim reused old task fencing")
			}
			var countBefore int
			m.f.s.db.QueryRow(`SELECT count(*) FROM events`).Scan(&countBefore)
			if _, err = m.f.s.ManagedReport(testContext, m.a, input); err != nil {
				t.Fatal("terminal exact replay lost", err)
			}
			var countAfter int
			m.f.s.db.QueryRow(`SELECT count(*) FROM events`).Scan(&countAfter)
			current, err := m.f.s.Get(testContext, m.f.coord, core.GetInput{TaskKey: task.Key})
			if err != nil || countAfter != countBefore || current.Lease == nil || current.Lease.SessionID != newActor.SessionID || current.Lease.Generation != newClaim.Lease.Generation {
				t.Fatal("historical terminal replay released new attempt", err)
			}
		})
	}
}

func TestManagedUnknownAndInvalidTerminalProofDoNotReleaseBoardOwnership(t *testing.T) {
	m := managedSetup(t)
	task := assignedTask(t, m.f, m.p)
	m.start(t)
	p := m.poll(t)
	m.report(t, p, "accepted")
	view, err := m.f.s.Describe(testContext, m.f.p, managedDescribeInput(m, *p.Run))
	if err != nil {
		t.Fatal(err)
	}
	native := core.Actor{Principal: m.f.p, SessionID: view.SessionID}
	claim := m.f.claim(t, native, task.Key)
	m.report(t, p, "started")
	bad := m.reportInput(p, "completed")
	bad.Generation++
	_, err = m.f.s.ManagedReport(testContext, m.a, bad)
	requireCode(t, err, "STALE_GENERATION")
	m.report(t, p, "delivery_unknown")
	still, err := m.f.s.Get(testContext, m.f.coord, core.GetInput{TaskKey: task.Key})
	if err != nil || still.Lease == nil || still.Lease.SessionID != native.SessionID || still.Lease.Generation != claim.Lease.Generation || still.Attempts[0].EndedAt != nil {
		t.Fatal("unknown exit released ownership", err)
	}
	if _, err = m.f.s.Heartbeat(testContext, native, core.HeartbeatInput{RequestID: m.f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}); err != nil {
		t.Fatal("unknown scope silently expired board session", err)
	}
}

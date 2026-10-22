package store

import (
	"agent-board/internal/core"
	"strings"
	"testing"
)

func TestManagedRecoveredStartClosesAcceptedStopAndResumesWithQueuedClarification(t *testing.T) {
	m := managedSetup(t)
	m.start(t)
	start := m.poll(t)
	m.report(t, start, "accepted")
	live := m.report(t, start, "started")
	clarify, err := m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: live.Run.ID, Version: live.Run.Version, Kind: "clarify", Text: "Keep this clarification"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: live.Run.ID, Version: clarify.Run.Version, Kind: "stop"})
	if err != nil {
		t.Fatal(err)
	}
	stop := m.poll(t)
	if stop.Command == nil || stop.Command.Kind != "stop" {
		t.Fatal("missing stop")
	}
	m.report(t, stop, "accepted")
	recovered := m.report(t, start, "recovered_stopped")
	if recovered.Run.State != "stopped" {
		t.Fatal("recovery did not confirm stopped")
	}
	detail, err := m.f.s.ManagedRun(testContext, m.a, core.ManagedRunInput{RunID: live.Run.ID})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range detail.Commands {
		if c.ID == stop.Command.ID && c.Status != "cancelled" {
			t.Fatal("accepted stop outlived recovered execution")
		}
		if c.ID == clarify.Command.ID && c.Status != "queued" {
			t.Fatal("undispatched clarification lost")
		}
	}
	resume, err := m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: live.Run.ID, Version: recovered.Run.Version, Kind: "resume"})
	if err != nil {
		t.Fatal(err)
	}
	next := m.poll(t)
	if next.Command == nil || next.Command.ID != resume.Command.ID || next.Command.Generation != 2 || !strings.Contains(next.Command.Text, "Keep this clarification") {
		t.Fatal("old stop blocked resumed generation or mailbox lost", next)
	}
	m.report(t, next, "accepted")
	started := m.report(t, next, "started")
	if started.Run.State != "running" || started.Run.ProviderSessionID != managedTestSession {
		t.Fatal("exact native continuation unavailable")
	}
}

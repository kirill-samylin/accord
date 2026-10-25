package store

import (
	"agent-board/internal/core"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func insertTerminalRuns(t *testing.T, f *fixture, host string, n int, workflow string) {
	t.Helper()
	base := f.clock.read().Add(-time.Hour)
	for i := 0; i < n; i++ {
		r := core.ManagedRun{ID: fmt.Sprintf("mr_history_%04d", i), HostID: host, Version: 1, Provider: "codex", Workspace: "gui", State: "stopped", Generation: 1, CreatedAt: base, UpdatedAt: base.Add(time.Duration(i) * time.Second), Output: []string{"old"}}
		if workflow != "" {
			r.Automation = &core.ManagedAutomationBinding{WorkflowID: workflow}
		}
		raw, _ := json.Marshal(r)
		if _, err := f.s.db.Exec(`INSERT INTO managed_runs VALUES(?,?)`, r.ID, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
}

// Lifetime history no longer locks a project: admission counts live runs, and
// at the terminal bound the oldest idle runs move to audited archive rows that
// remain readable and are never deleted.
func TestManagedStartRotatesOldestTerminalHistoryToAuditedArchive(t *testing.T) {
	m := managedSetup(t)
	insertTerminalRuns(t, m.f, m.h.ID, managedTerminalBound, "")
	started := m.start(t)
	page, err := m.f.s.ManagedList(testContext, m.a, core.ManagedListInput{})
	if err != nil || len(page.Runs) != managedTerminalTarget+1 {
		t.Fatal("live history not rotated to target", err, len(page.Runs))
	}
	if _, err = m.f.s.ManagedRun(testContext, m.a, core.ManagedRunInput{RunID: "mr_history_0000"}); err == nil {
		t.Fatal("oldest run still live")
	}
	if _, err = m.f.s.ManagedRun(testContext, m.a, core.ManagedRunInput{RunID: started.Run.ID}); err != nil {
		t.Fatal("new run missing", err)
	}
	seen := []string{}
	cursor := ""
	for {
		v, err := m.f.s.AutomationArchive(testContext, m.a, core.AutomationArchiveInput{Kind: "managed_runs", Cursor: cursor, Limit: 32})
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range v.Runs {
			if len(r.Output) != 0 || r.State != "stopped" {
				t.Fatal("archive projection leaked output or state", r.ID)
			}
			seen = append(seen, r.ID)
		}
		if cursor = v.NextCursor; cursor == "" {
			break
		}
	}
	if len(seen) != managedTerminalBound-managedTerminalTarget || seen[0] != "mr_history_0000" || seen[len(seen)-1] != fmt.Sprintf("mr_history_%04d", len(seen)-1) {
		t.Fatal("archive did not keep exactly the oldest runs", len(seen))
	}
	var rows int
	if err = m.f.s.db.QueryRow(`SELECT count(*) FROM managed_runs`).Scan(&rows); err != nil || rows != managedTerminalBound+1 {
		t.Fatal("history rows were deleted", rows, err)
	}
	events, err := m.f.s.Events(testContext, m.a, core.EventsInput{Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	audited := false
	for _, e := range events.Events {
		audited = audited || e.Type == automationArchiveAuditEv && strings.Contains(e.Reason, "archived_runs=64")
	}
	if !audited {
		t.Fatal("rotation was not audited")
	}
}

// Runs still owned by a live workflow are never archived; admission then fails
// with an explicit capacity reason instead of silently dropping history.
func TestManagedStartRefusesWhenTerminalHistoryIsNotArchivable(t *testing.T) {
	m := managedSetup(t)
	insertTerminalRuns(t, m.f, m.h.ID, managedTerminalBound, "wf_live")
	raw, _ := json.Marshal(core.Workflow{ID: "wf_live", Version: 1, State: "paused", CreatedAt: m.f.clock.read(), UpdatedAt: m.f.clock.read()})
	if _, err := m.f.s.db.Exec(`INSERT INTO workflows VALUES(?,?)`, "wf_live", string(raw)); err != nil {
		t.Fatal(err)
	}
	_, err := m.f.s.ManagedStart(testContext, m.a, core.ManagedStartInput{RequestID: m.f.id(), HostID: m.h.ID, Workspace: "gui", Provider: "codex", ProfileID: m.p.ID, ProfileVersion: m.p.Version, Prompt: "Review only"})
	if err == nil || !strings.Contains(err.Error(), "ротация невозможна") {
		t.Fatal("unarchivable history did not refuse with capacity", err)
	}
}

// Submit counts only non-terminal workflows; terminal workflows above the bound
// are archived with their runs instead of locking intake forever.
func TestAutomationSubmitCountsLiveWorkflowsAndArchivesTerminalHistory(t *testing.T) {
	m := automationSetup(t)
	now := m.f.clock.read()
	for i := 0; i < workflowTerminalBound; i++ {
		w := core.Workflow{ID: fmt.Sprintf("wf_done_%04d", i), Version: 3, State: "completed", CreatedAt: now, UpdatedAt: now.Add(time.Duration(i-1000) * time.Second)}
		raw, _ := json.Marshal(w)
		if _, err := m.f.s.db.Exec(`INSERT INTO workflows VALUES(?,?)`, w.ID, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	m.submit(t)
	list, err := m.f.s.AutomationArchive(m.ctx, m.a, core.AutomationArchiveInput{Kind: "workflows", Limit: 32})
	if err != nil || len(list.Workflows) != workflowTerminalBound-workflowTerminalTarget || list.Workflows[0].ID != "wf_done_0000" || list.NextCursor != "" {
		t.Fatal("terminal workflows not archived oldest-first", err, len(list.Workflows))
	}
	for i := 1; i < workflowLiveLimit; i++ {
		w := core.Workflow{ID: fmt.Sprintf("wf_paused_%04d", i), Version: 1, State: "paused", CreatedAt: now, UpdatedAt: now}
		raw, _ := json.Marshal(w)
		if _, err := m.f.s.db.Exec(`INSERT INTO workflows VALUES(?,?)`, w.ID, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	_, err = m.f.s.WorkflowSubmit(m.ctx, m.a, core.WorkflowSubmitInput{RequestID: m.f.id(), PolicyID: m.p.ID, PolicyVersion: m.p.Version, Text: "One more."})
	if err == nil || !strings.Contains(err.Error(), "незавершённых") {
		t.Fatal("live workflow cap not enforced", err)
	}
}

// T022-R2-7: archiving writes the entity's final value (capabilities of an
// archived workflow are revoked in the archive row), restore revokes archived
// capability rows of older versions too, and nothing is renamed twice.
func TestAutomationArchiveRevokesCapabilitiesAndRestoreCoversArchiveRows(t *testing.T) {
	m := automationSetup(t)
	now := m.f.clock.read()
	for i := 0; i < workflowTerminalBound; i++ {
		w := core.Workflow{ID: fmt.Sprintf("wf_done_%04d", i), Version: 3, State: "completed", CreatedAt: now, UpdatedAt: now.Add(time.Duration(i-1000) * time.Second)}
		raw, _ := json.Marshal(w)
		if _, err := m.f.s.db.Exec(`INSERT INTO workflows VALUES(?,?)`, w.ID, string(raw)); err != nil {
			t.Fatal(err)
		}
		c := executionCapabilityRecord{ExecutionCapability: core.ExecutionCapability{ID: fmt.Sprintf("ec_done_%04d", i), WorkflowID: w.ID, RunID: fmt.Sprintf("mr_done_%04d", i), Generation: 1, State: "active", CreatedAt: now, ExpiresAt: now}, Digest: "history"}
		raw, _ = json.Marshal(c)
		if _, err := m.f.s.db.Exec(`INSERT INTO execution_capabilities VALUES(?,?)`, c.ID, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	legacy, _ := json.Marshal(executionCapabilityRecord{ExecutionCapability: core.ExecutionCapability{ID: "ec_legacy", RunID: "mr_legacy", Generation: 1, State: "active"}, Digest: "legacy"})
	if _, err := m.f.s.db.Exec(`INSERT INTO execution_capabilities VALUES(?,?)`, automationArchivePrefix+"ec_legacy", string(legacy)); err != nil {
		t.Fatal(err)
	}
	m.submit(t)
	count := func(query string) int {
		t.Helper()
		var n int
		if err := m.f.s.db.QueryRow(query).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}
	archived := workflowTerminalBound - workflowTerminalTarget
	if got := count(`SELECT count(*) FROM execution_capabilities WHERE substr(key,1,8)='archive/' AND key<>'archive/ec_legacy' AND json_extract(data,'$.state')='revoked'`); got != archived {
		t.Fatal("archived capabilities not revoked in their archive rows", got)
	}
	if got := count(`SELECT count(*) FROM execution_capabilities WHERE json_extract(data,'$.state')<>'revoked'`); got != workflowTerminalTarget+1 {
		t.Fatal("unexpected live capability inventory", got)
	}
	if _, err := m.f.s.RecoverState(testContext, "restore"); err != nil {
		t.Fatal(err)
	}
	if got := count(`SELECT count(*) FROM execution_capabilities WHERE json_extract(data,'$.state')<>'revoked'`); got != 0 {
		t.Fatal("restore retained execution authority in capability rows", got)
	}
	if got := count(`SELECT count(*) FROM execution_capabilities WHERE substr(key,1,8)='archive/'`); got != archived+1 {
		t.Fatal("restore changed archive inventory", got)
	}

	// A move whose live row is missing fails the save instead of being a no-op.
	tx, err := m.f.s.db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	err = saveBoard(testContext, tx, &board{archive: []archiveMove{{"workflows", "wf_missing", core.Workflow{ID: "wf_missing"}}}})
	if err == nil || !strings.Contains(err.Error(), "want 1") {
		t.Fatal("archive of a missing row did not fail the transaction", err)
	}
}

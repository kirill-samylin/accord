package store

import (
	"agent-board/internal/core"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestAutomationSchema5MigrationPreservesRowsAndSnapshotsBeforeDDL(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	var before string
	if e := f.s.db.QueryRow(`SELECT data FROM tasks WHERE key=?`, task.Key).Scan(&before); e != nil {
		t.Fatal(e)
	}
	dropAutomationFixture(t, f.s)
	if _, e := f.s.db.Exec(`PRAGMA user_version=5`); e != nil {
		t.Fatal(e)
	}
	f.s.Close()
	var e error
	f.s, e = Open(f.path, core.Config{}, f.clock.read)
	if e != nil {
		t.Fatal(e)
	}
	var version int
	var after string
	f.s.db.QueryRow(`PRAGMA user_version`).Scan(&version)
	f.s.db.QueryRow(`SELECT data FROM tasks WHERE key=?`, task.Key).Scan(&after)
	if version != 6 || before != after {
		t.Fatal("legacy rows changed")
	}
	snapshots, _ := filepath.Glob(f.path + ".migrations/schema5-*.db")
	if len(snapshots) != 1 {
		t.Fatal("missing preupgrade snapshot")
	}
	if _, e = os.Stat(snapshots[0] + ".sha256"); e != nil {
		t.Fatal(e)
	}
	backup, e := openReadOnly(testContext, snapshots[0])
	if e != nil {
		t.Fatal(e)
	}
	defer backup.Close()
	backup.QueryRow(`PRAGMA user_version`).Scan(&version)
	if version != 5 {
		t.Fatal("snapshot mutated")
	}
}
func TestAutomationDeclaredSchema6RejectsMissingOrWeakenedConstraint(t *testing.T) {
	for _, change := range []string{`DROP TABLE workflow_steps`, `DROP TABLE automation_metadata`, `DROP INDEX automation_cap_attempt`, `DROP INDEX automation_step_execution; CREATE INDEX automation_step_execution ON workflow_steps(key)`} {
		t.Run(change, func(t *testing.T) {
			f := newFixture(t)
			if _, e := f.s.db.Exec(change); e != nil {
				t.Fatal(e)
			}
			f.s.Close()
			if s, e := Open(f.path, core.Config{}, f.clock.read); e == nil {
				s.Close()
				t.Fatal("damaged schema6 repaired/accepted")
			}
		})
	}
}
func TestAutomationRestoreRetainsHistoryAndReservationsButInvalidatesAuthority(t *testing.T) {
	m := automationSetup(t)
	flow := m.submit(t)
	m.tick(t, flow.Workflow.ID)
	attempt := m.attempt(t, flow.Workflow.ID)
	before := m.page(t, flow.Workflow.ID)
	result, e := m.f.s.RecoverState(m.ctx, "restore")
	if e != nil || result.Leases != 1 {
		t.Fatal("restore", e)
	}
	_, e = transact(m.f.s, testContext, func(b *board, now time.Time) (bool, error) {
		w := b.Automation.Workflows[flow.Workflow.ID]
		if w.State != "needs_recovery" || w.ControlEpoch <= before.Workflow.ControlEpoch || w.Grant == nil || len(b.Automation.Steps) != len(before.Steps) || b.Automation.Reservations[w.ID] == nil {
			t.Fatal("restore losthistory orforgedfreeworkspace")
		}
		if b.Automation.Policies[m.p.ID].State != "revoked" {
			t.Fatal("policy authority survived")
		}
		for _, c := range b.Automation.Capabilities {
			if c.State != "revoked" {
				t.Fatal("cap authority survived")
			}
		}
		if b.ManagedRuns[attempt.poll.Run.ID].State != "needs_recovery" {
			t.Fatal("run exitforged")
		}
		return true, nil
	})
	if e != nil {
		t.Fatal(e)
	}
}

func TestAutomationConstraintJSONPathsAreCaseSensitive(t *testing.T) {
	for _, spec := range []struct{ name, table, fields string }{{"automation_step_execution", "workflow_steps", `json_extract(data,'$.WORKFLOW_ID'),json_extract(data,'$.sequence'),json_extract(data,'$.attempt_no')`}, {"automation_cap_attempt", "execution_capabilities", `json_extract(data,'$.RUN_ID'),json_extract(data,'$.generation')`}, {"automation_step_execution", "workflow_steps", `json_extract(data,'$.workflow_ id'),json_extract(data,'$.sequence'),json_extract(data,'$.attempt_no')`}, {"automation_cap_attempt", "execution_capabilities", `json_extract(data,'$.run_ id'),json_extract(data,'$.generation')`}} {
		t.Run(spec.name, func(t *testing.T) {
			f := newFixture(t)
			if _, e := f.s.db.Exec("DROP INDEX " + spec.name); e != nil {
				t.Fatal(e)
			}
			if _, e := f.s.db.Exec("CREATE UNIQUE INDEX " + spec.name + " ON " + spec.table + "(" + spec.fields + ")"); e != nil {
				t.Fatal(e)
			}
			f.s.Close()
			if s, e := Open(f.path, core.Config{}, f.clock.read); e == nil {
				s.Close()
				t.Fatal("NULL unique JSON path index accepted")
			}
		})
	}
}

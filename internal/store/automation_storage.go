package store

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"time"
)

type executionCapabilityRecord struct {
	core.ExecutionCapability
	Digest string `json:"digest"`
}
type workflowReservation struct {
	WorkflowID string `json:"workflow_id"`
	HostID     string `json:"host_id"`
	Workspace  string `json:"workspace"`
}
type automationBoard struct {
	Policies     map[string]*core.AutomationPolicy       `json:"policies"`
	Revisions    map[string]*core.AutomationPolicy       `json:"revisions"`
	Workflows    map[string]*core.Workflow               `json:"workflows"`
	Steps        map[string]*core.WorkflowStep           `json:"steps"`
	Plans        map[string]*core.WorkflowPlanSnapshot   `json:"plans"`
	Questions    map[string]*core.WorkflowQuestionRecord `json:"questions"`
	Attestations map[string]*core.WorkflowAttestation    `json:"attestations"`
	Capabilities map[string]*executionCapabilityRecord   `json:"capabilities"`
	Reservations map[string]*workflowReservation         `json:"reservations"`
}

func newAutomationBoard() *automationBoard {
	return &automationBoard{Policies: map[string]*core.AutomationPolicy{}, Revisions: map[string]*core.AutomationPolicy{}, Workflows: map[string]*core.Workflow{}, Steps: map[string]*core.WorkflowStep{}, Plans: map[string]*core.WorkflowPlanSnapshot{}, Questions: map[string]*core.WorkflowQuestionRecord{}, Attestations: map[string]*core.WorkflowAttestation{}, Capabilities: map[string]*executionCapabilityRecord{}, Reservations: map[string]*workflowReservation{}}
}

var automationTables = []string{"automation_policies", "automation_policy_revisions", "workflows", "workflow_steps", "workflow_plans", "workflow_questions", "workflow_attestations", "execution_capabilities", "workflow_reservations"}

func automationDDL() []string {
	q := []string{}
	for _, table := range automationTables {
		q = append(q, "CREATE TABLE "+table+" (key TEXT PRIMARY KEY,data TEXT NOT NULL)")
	}
	return append(q, `CREATE TABLE automation_metadata(id INTEGER PRIMARY KEY CHECK(id=1),version INTEGER NOT NULL CHECK(version=1))`, `INSERT INTO automation_metadata VALUES(1,1)`, `CREATE UNIQUE INDEX automation_step_execution ON workflow_steps(json_extract(data,'$.workflow_id'),json_extract(data,'$.sequence'),json_extract(data,'$.attempt_no'))`, `CREATE UNIQUE INDEX automation_cap_attempt ON execution_capabilities(json_extract(data,'$.run_id'),json_extract(data,'$.generation'))`)
}
func loadAutomation(ctx context.Context, tx *sql.Tx, b *board) error {
	b.Automation = newAutomationBoard()
	a := b.Automation
	loaders := []func() error{func() error { return loadEntities(ctx, tx, automationTables[0], a.Policies) }, func() error { return loadEntities(ctx, tx, automationTables[1], a.Revisions) }, func() error { return loadEntities(ctx, tx, automationTables[2], a.Workflows) }, func() error { return loadEntities(ctx, tx, automationTables[3], a.Steps) }, func() error { return loadEntities(ctx, tx, automationTables[4], a.Plans) }, func() error { return loadEntities(ctx, tx, automationTables[5], a.Questions) }, func() error { return loadEntities(ctx, tx, automationTables[6], a.Attestations) }, func() error { return loadEntities(ctx, tx, automationTables[7], a.Capabilities) }, func() error { return loadEntities(ctx, tx, automationTables[8], a.Reservations) }}
	for _, load := range loaders {
		if err := load(); err != nil {
			return err
		}
	}
	return nil
}
func saveAutomation(ctx context.Context, tx *sql.Tx, b *board) error {
	if b.Automation == nil {
		return nil
	}
	a := b.Automation
	// Only active reservations are ephemeral; removal is committed atomically with
	// proven terminal state. The workflow/step/evidence audit is never deleted.
	if _, err := tx.ExecContext(ctx, `DELETE FROM workflow_reservations`); err != nil {
		return err
	}
	savers := []func() error{func() error { return saveEntities(ctx, tx, automationTables[0], a.Policies) }, func() error { return saveEntities(ctx, tx, automationTables[1], a.Revisions) }, func() error { return saveEntities(ctx, tx, automationTables[2], a.Workflows) }, func() error { return saveEntities(ctx, tx, automationTables[3], a.Steps) }, func() error { return saveEntities(ctx, tx, automationTables[4], a.Plans) }, func() error { return saveEntities(ctx, tx, automationTables[5], a.Questions) }, func() error { return saveEntities(ctx, tx, automationTables[6], a.Attestations) }, func() error { return saveEntities(ctx, tx, automationTables[7], a.Capabilities) }, func() error { return saveEntities(ctx, tx, automationTables[8], a.Reservations) }}
	for _, save := range savers {
		if err := save(); err != nil {
			return err
		}
	}
	return nil
}
func resetAutomation(b *board, now time.Time) {
	if b.Automation == nil {
		return
	}
	a := b.Automation
	for _, p := range a.Policies {
		p.State = "revoked"
		p.Version++
		p.UpdatedAt = now
	}
	for _, w := range a.Workflows {
		if w.State != "completed" && w.State != "cancelled" {
			w.State = "needs_recovery"
			workflowReason(w, "restore_authority_invalidated", "")
			w.ControlEpoch++
			w.Version++
			w.UpdatedAt = now
		}
	}
	for _, c := range a.Capabilities {
		c.State = "revoked"
	}
	// Reservations intentionally survive until explicit owned-process recovery.
}

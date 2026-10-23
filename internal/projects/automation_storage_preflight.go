package projects

import (
	"database/sql"
	"errors"
	"strings"
)

// Keep the offline preflight strict before a migration writer opens any board.
// These definitions mirror store automationDDL without importing its private API.
func validateAutomationStorage(db *sql.DB) error {
	for _, table := range []string{"automation_policies", "automation_policy_revisions", "workflows", "workflow_steps", "workflow_plans", "workflow_questions", "workflow_attestations", "execution_capabilities", "workflow_reservations"} {
		if err := requireColumns(db, []string{"SELECT key,data FROM " + table + " LIMIT 0"}); err != nil {
			return err
		}
	}
	var count, version int
	if err := db.QueryRow(`SELECT count(*),COALESCE(MAX(version),0) FROM automation_metadata`).Scan(&count, &version); err != nil || count != 1 || version != 1 {
		return errors.New("published automation metadata incomplete")
	}
	if err := db.QueryRow(`SELECT version FROM automation_metadata WHERE id=1`).Scan(&version); err != nil || version != 1 {
		return errors.New("published automation metadata identity missing")
	}
	for name, expected := range map[string]string{
		"automation_step_execution": `CREATE UNIQUE INDEX automation_step_execution ON workflow_steps(json_extract(data,'$.workflow_id'),json_extract(data,'$.sequence'),json_extract(data,'$.attempt_no'))`,
		"automation_cap_attempt":    `CREATE UNIQUE INDEX automation_cap_attempt ON execution_capabilities(json_extract(data,'$.run_id'),json_extract(data,'$.generation'))`,
	} {
		var actual string
		if err := db.QueryRow(`SELECT sql FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&actual); err != nil || strings.TrimSpace(actual) != expected {
			return errors.New("published automation execution constraint changed")
		}
	}
	return nil
}

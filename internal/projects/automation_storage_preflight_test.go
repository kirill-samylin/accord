package projects

import (
	"context"
	"database/sql"
	"testing"
)

var dropAutomationStorageFixture = []string{
	`DROP TABLE automation_policies`, `DROP TABLE automation_policy_revisions`, `DROP TABLE workflows`, `DROP TABLE workflow_steps`, `DROP TABLE workflow_plans`, `DROP TABLE workflow_questions`, `DROP TABLE workflow_attestations`, `DROP TABLE execution_capabilities`, `DROP TABLE workflow_reservations`, `DROP TABLE automation_metadata`,
}

func TestOfflineAutomationPreflightRejectsDamageBeforeAnyMigration(t *testing.T) {
	for _, damage := range []string{`DROP TABLE workflows`, `DELETE FROM automation_metadata`, `DROP INDEX automation_cap_attempt`, `DROP INDEX automation_step_execution; CREATE INDEX automation_step_execution ON workflow_steps(key)`, `DROP INDEX automation_step_execution; CREATE UNIQUE INDEX automation_step_execution ON workflow_steps(json_extract(data,'$.WORKFLOW_ID'),json_extract(data,'$.sequence'),json_extract(data,'$.attempt_no'))`, `DROP INDEX automation_step_execution; CREATE UNIQUE INDEX automation_step_execution ON workflow_steps(json_extract(data,'$.workflow_ id'),json_extract(data,'$.sequence'),json_extract(data,'$.attempt_no'))`, `PRAGMA user_version=7`} {
		t.Run(damage, func(t *testing.T) {
			f := setup(t)
			p := f.create(t, f.who, "damaged-automation")
			root := f.r.options.DataDir
			path, _, err := f.r.projectPaths(p.ID)
			if err != nil {
				t.Fatal(err)
			}
			f.r.Close()
			f.browser.Close()
			f.store.Close()
			closedSchema2(t, f.db)
			db, err := sql.Open("sqlite", path)
			if err != nil {
				t.Fatal(err)
			}
			_, err = db.Exec(damage)
			db.Close()
			if err != nil {
				t.Fatal(err)
			}
			if _, err = MigrateStorage(context.Background(), root, f.db, f.webDir); err == nil {
				t.Fatal("incomplete automation accepted")
			}
			db, err = sql.Open("sqlite", f.db)
			if err != nil {
				t.Fatal(err)
			}
			defer db.Close()
			var v int
			if err = db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil || v != 2 {
				t.Fatal("partial migration before full inventory validation", v, err)
			}
		})
	}
}

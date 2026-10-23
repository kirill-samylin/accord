package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestAutomationDraftRESTCurrentACLAndProjectIsolation(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	p := f.create(t, f.who, "workflow")
	other := f.create(t, f.who, "other")
	credential, who, err := f.r.CreateCredential(ctx, "requester", "machine", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.r.changeMember(ctx, f.who, p.ID, who.ID, "editor", "grant", 0); err != nil {
		t.Fatal(err)
	}
	session := f.describe(t, p.ID, credential.Secret, "intake", "analyst")
	path := "/api/v1/p/" + p.ID + "/automation/"
	raw := `{"request_id":"requirement-1","text":"Добавить страницу результата"}`
	response := f.native(t, "POST", path+"submit", credential.Secret, session.SessionID, raw)
	if response.Code != 200 {
		t.Fatal(response.Code, response.Body.String())
	}
	var page core.WorkflowPage
	if json.Unmarshal(response.Body.Bytes(), &page) != nil || page.Workflow.State != "awaiting_authorization" || page.Workflow.Grant != nil || len(page.Steps) != 0 {
		t.Fatal("draft started execution", response.Body.String())
	}
	if replay := f.native(t, "POST", path+"submit", credential.Secret, session.SessionID, raw); replay.Code != 200 || replay.Body.String() != response.Body.String() {
		t.Fatal("receipt changed", replay.Code, replay.Body.String())
	}
	foreign := f.describe(t, other.ID, f.token.Secret, "other-intake", "analyst")
	get, _ := json.Marshal(core.WorkflowReadInput{WorkflowID: page.Workflow.ID})
	if out := f.native(t, "POST", "/api/v1/p/"+other.ID+"/automation/get", f.token.Secret, foreign.SessionID, string(get)); out.Code != 404 {
		t.Fatal("cross project read", out.Code, out.Body.String())
	}
	if out := f.native(t, "POST", path+"submit?unexpected=1", credential.Secret, session.SessionID, raw); out.Code != 400 {
		t.Fatal("query accepted", out.Code)
	}
	if out := f.native(t, "POST", path+"submit", credential.Secret, session.SessionID, `{"request_id":"x","text":"a","text":"b"}`); out.Code != 400 {
		t.Fatal("duplicate fields", out.Code)
	}
	if err := f.r.changeMember(ctx, f.who, p.ID, who.ID, "", "revoke", 1); err != nil {
		t.Fatal(err)
	}
	if out := f.native(t, "POST", path+"submit", credential.Secret, session.SessionID, raw); out.Code != 403 && out.Code != 404 {
		t.Fatal("revoked replay accepted", out.Code, out.Body.String())
	}
}

func TestExecutionCapabilityCannotUseProjectReadOrControlOperations(t *testing.T) {
	ctx := store.WithExecutionCapability(context.Background(), core.ExecutionIdentity{Capability: core.ExecutionCapability{Operations: []string{"get", "workflow_get"}}})
	for _, op := range []string{"workflow_submit", "workflow_command", "workflow_list", "team", "profiles", "managed_list", "automation_policy_create", "events"} {
		if err := automationOperationAllowed(ctx, op); err == nil {
			t.Fatal("operation expanded", op)
		}
	}
	if err := automationOperationAllowed(ctx, "workflow_get"); err != nil {
		t.Fatal(err)
	}
	f := setup(t)
	for _, path := range []string{"/api/v1/p/p_00000000000000000000000000000000/automation/get", "/api/v1/managed/list", "/api/v1/team"} {
		out := f.native(t, "POST", path, core.ExecutionTokenPrefix+strings.Repeat("a", 64), "", `{}`)
		if out.Code == 200 {
			t.Fatal("unregistered capability accepted", path)
		}
	}
}

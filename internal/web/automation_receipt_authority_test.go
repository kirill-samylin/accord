package web

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"database/sql"
	"errors"
	"net/url"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

type uncertainWorkflowAnswer struct {
	*store.Store
	fail bool
}

func (b *uncertainWorkflowAnswer) WorkflowAnswer(ctx context.Context, a core.Actor, in core.WorkflowAnswerInput) (core.WorkflowPage, error) {
	out, e := b.Store.WorkflowAnswer(ctx, a, in)
	if e == nil && b.fail {
		b.fail = false
		return core.WorkflowPage{}, errors.New("lost answer reply fixture")
	}
	return out, e
}
func TestAutomationCachedAnswerUsesCurrentQuestionDesignation(t *testing.T) {
	m := automationReceiptSetup(t, 24*time.Hour)
	w := m.submit(t)
	asked := m.question(t, w.Workflow)
	q := asked.Questions[0]
	m.f.s.board = &uncertainWorkflowAnswer{Store: m.f.board, fail: true}
	page := m.page(t, w.Workflow.ID)
	fields := jsonBrowserFields(page, "automation:answer", url.Values{"workflow_id": {w.Workflow.ID}, "version": {strconv.FormatInt(page.Workflow.Workflow.Version, 10)}, "question_id": {q.ID}, "question_version": {strconv.FormatInt(q.Version, 10)}, "answer": {"A"}, "resume": {"false"}})
	failed := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/answer", fields, m.cookie), 503)
	if !failed.RetrySameAction || failed.ActionID != fields.Get("action_id") {
		t.Fatal("lost answer reply did not preserve action")
	}
	first := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/answer", fields, m.cookie), 200)
	cached := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/answer", fields, m.cookie), 200)
	if first.Redirect != cached.Redirect {
		t.Fatal("cached answer lost exact receipt")
	}
	answered := m.page(t, w.Workflow.ID)
	if answered.Workflow.Questions[0].State != "answered" {
		t.Fatal("answer not committed")
	}
	// Deliberately change only the durable designation fixture. Current question
	// authority must be read even when a successful browser receipt already exists.
	db, e := sql.Open("sqlite", filepath.Join(m.f.dir, "board.db"))
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	if _, e = db.Exec(`UPDATE workflow_questions SET data=json_set(data,'$.designated_principals',json('["different-human"]')) WHERE key=?`, q.ID); e != nil {
		t.Fatal(e)
	}
	decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/answer", fields, m.cookie), 403)
}
func TestAutomationCachedResumeRejectsRevokedGrant(t *testing.T) {
	m := automationReceiptSetup(t, 24*time.Hour)
	w := m.submit(t)
	m.command(t, w.Workflow, "pause")
	page := m.page(t, w.Workflow.ID)
	fields := jsonBrowserFields(page, "automation:command", url.Values{"workflow_id": {w.Workflow.ID}, "version": {strconv.FormatInt(page.Workflow.Workflow.Version, 10)}, "kind": {"resume"}})
	decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/command", fields, m.cookie), 200)
	decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/command", fields, m.cookie), 200)
	if _, e := m.f.board.AutomationPolicyRevoke(m.ctx, m.a, core.AutomationPolicyRevokeInput{RequestID: "revoke", PolicyID: m.policy.ID, Version: m.policy.Version, Reason: "revoke grant"}); e != nil {
		t.Fatal(e)
	}
	decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/command", fields, m.cookie), 403)
}
func TestAutomationCachedRefreshRejectsChangedPolicyRevision(t *testing.T) {
	m := automationReceiptSetup(t, 24*time.Hour)
	w := m.submit(t)
	m.command(t, w.Workflow, "pause")
	page := m.page(t, w.Workflow.ID)
	fields := jsonBrowserFields(page, "automation:command", url.Values{"workflow_id": {w.Workflow.ID}, "version": {strconv.FormatInt(page.Workflow.Workflow.Version, 10)}, "kind": {"resume"}, "refresh_policy_id": {m.policy.ID}, "refresh_policy_version": {strconv.FormatInt(m.policy.Version, 10)}})
	decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/command", fields, m.cookie), 200)
	decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/command", fields, m.cookie), 200)
	m.updatePolicy(t)
	decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/command", fields, m.cookie), 409)
}
func TestAutomationCachedPolicyChecksExactWorkspaceExpiry(t *testing.T) {
	m := automationReceiptSetup(t, 30*time.Second)
	decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/policies/create", m.create, m.cookie), 200)
	m.f.now = m.f.now.Add(31 * time.Second)
	decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/policies/create", m.create, m.cookie), 403)
}

func TestAutomationBrowserIntakeStillWorksWhenNativeClientIntakeIsOff(t *testing.T) {
	m := automationReceiptSetup(t, 24*time.Hour)
	if m.policy.ClientIntake {
		t.Fatal("fixture accidentally enabled native intake")
	}
	_, e := m.f.board.WorkflowSubmit(m.ctx, m.a, core.WorkflowSubmitInput{RequestID: "native-human-forged", PolicyID: m.policy.ID, PolicyVersion: m.policy.Version, Text: "Human role is not a browser action"})
	if e == nil {
		t.Fatal("untrusted human role bypassed disabled client intake")
	}
	page := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "GET", "/web/automation", nil, m.cookie), 200).Page
	fields := jsonBrowserFields(page, "automation:submit", url.Values{"policy_id": {m.policy.ID}, "policy_version": {strconv.FormatInt(m.policy.Version, 10)}, "text": {"Trusted browser intake"}})
	forged := cloneAutomationValues(fields)
	forged.Set("intake_approval", "true")
	decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/submit", forged, m.cookie), 400)
	created := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/submit", fields, m.cookie), 200)
	cached := decodeBrowser(t, automationBrowserRequest(m.f, m.ctx, "POST", "/web/automation/submit", fields, m.cookie), 200)
	if created.Redirect != cached.Redirect {
		t.Fatal("browser intake receipt changed")
	}
}

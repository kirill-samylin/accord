package web

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"errors"
	"net/url"
	"testing"
	"time"
)

type uncertainAutomationBoard struct {
	*store.Store
	fail   bool
	before bool
}

func (b *uncertainAutomationBoard) WorkflowSubmit(ctx context.Context, a core.Actor, in core.WorkflowSubmitInput) (core.WorkflowPage, error) {
	if b.fail && b.before {
		b.fail = false
		return core.WorkflowPage{}, errors.New("private pre-commit fixture")
	}
	result, err := b.Store.WorkflowSubmit(ctx, a, in)
	if err == nil && b.fail {
		b.fail = false
		return core.WorkflowPage{}, errors.New("private lost-reply fixture")
	}
	return result, err
}
func TestAutomationBrowserUnknownReplySurvivesSessionContinuation(t *testing.T) {
	f, ctx, cookie, _ := automationBrowserSetup(t)
	f.s.board = &uncertainAutomationBoard{Store: f.board, fail: true}
	page := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	fields := jsonBrowserFields(page, "automation:submit", url.Values{"text": {"Keep exact requirement"}})
	unknown := decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/submit", fields, cookie), 503)
	if !unknown.RetrySameAction || unknown.ActionID != fields.Get("action_id") || unknown.Actions["automation:submit"] != fields.Get("action_id") || unknown.Draft["text"] != "Keep exact requirement" {
		t.Fatal("unknown reply lost original action/draft")
	}
	f.now = f.now.Add(2 * time.Minute)
	if _, err := f.s.db.Exec(`UPDATE browser_sessions SET board_session=json_set(board_session,'$.expires_at',?)`, time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	expired := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	if !expired.CanContinue {
		t.Fatal("session did not expire")
	}
	continuation := jsonBrowserFields(expired, "continue", nil)
	decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/session/continue", continuation, cookie), 200)
	receipt := decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/submit", fields, cookie), 200)
	detail := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", receipt.Redirect, nil, cookie), 200).Page
	if detail.Workflow == nil || detail.Workflow.Workflow.IntakeText != "Keep exact requirement" {
		t.Fatal("lost original workflow receipt")
	}
	list := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	if len(list.Automation.Workflows) != 1 {
		t.Fatal("continuation duplicated committed workflow")
	}
}

func TestAutomationBrowserContinuationCannotExecuteUncommittedExpiredAction(t *testing.T) {
	f, ctx, cookie, _ := automationBrowserSetup(t)
	f.s.board = &uncertainAutomationBoard{Store: f.board, fail: true, before: true}
	page := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	fields := jsonBrowserFields(page, "automation:submit", url.Values{"text": {"Never committed"}})
	decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/submit", fields, cookie), 503)
	f.now = f.now.Add(2 * time.Minute)
	if _, err := f.s.db.Exec(`UPDATE browser_sessions SET board_session=json_set(board_session,'$.expires_at',?)`, time.Now().Add(-time.Minute).UTC().Format(time.RFC3339Nano)); err != nil {
		t.Fatal(err)
	}
	expired := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/session/continue", jsonBrowserFields(expired, "continue", nil), cookie), 200)
	decodeBrowser(t, automationBrowserRequest(f, ctx, "POST", "/web/automation/submit", fields, cookie), 409)
	list := decodeBrowser(t, automationBrowserRequest(f, ctx, "GET", "/web/automation", nil, cookie), 200).Page
	if len(list.Automation.Workflows) != 0 {
		t.Fatal("session continuation silently authorized an uncommitted request")
	}
}

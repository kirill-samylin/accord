package web

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"net/http"
)

func (s *Server) handleAutomation(w http.ResponseWriter, r *http.Request, b *browserSession) {
	if r.Method != "POST" {
		if r.URL.Path == "/web/automation/host-scope" {
			p, err := s.automationHostScopePage(r, b)
			if err != nil {
				s.fail(w, r, b, err)
				return
			}
			s.render(w, 200, p.View, p)
			return
		}
		kind, id, ok := automationRoute(r.URL.Path)
		if !ok {
			http.NotFound(w, r)
			return
		}
		p, err := s.automationPage(r, b, kind, id)
		if err != nil {
			s.fail(w, r, b, err)
			return
		}
		s.render(w, 200, p.View, p)
		return
	}
	op, ok := automationOperations[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	action := r.PostForm.Get("action_id")
	if err := s.checkAction(r.Context(), b, action, op); err != nil {
		s.automationError(w, r, b, op, err)
		return
	}
	in, err := automationForm(r, op, action)
	if err != nil {
		s.automationError(w, r, b, op, err)
		return
	}
	raw, err := json.Marshal(in)
	if err != nil {
		s.fail(w, r, b, err)
		return
	}
	a, err := s.bindAction(r.Context(), b, action, op, raw)
	if err != nil {
		s.automationError(w, r, b, op, err)
		return
	}
	out, err := s.runAction(r.Context(), b, a)
	if err != nil {
		s.automationError(w, r, b, op, err)
		return
	}
	http.Redirect(w, r, out.Redirect, http.StatusSeeOther)
}
func (s *Server) automationError(w http.ResponseWriter, r *http.Request, b *browserSession, op string, err error) {
	kind, id := "list", ""
	if value := r.PostForm.Get("workflow_id"); value != "" {
		kind, id = "workflows", value
	}
	if value := r.PostForm.Get("policy_id"); value != "" && (op == "automation:policy-update" || op == "automation:policy-revoke") {
		kind, id = "policies", value
	}
	p, e := s.automationPage(r, b, kind, id)
	if e != nil {
		s.fail(w, r, b, err)
		return
	}
	p.Error, p.Draft, p.ActionID = publicError(err), draft(r), r.PostForm.Get("action_id")
	if errorStatus(err) >= 500 {
		p.Actions[op] = p.ActionID
	}
	browserErrorCode(w, err)
	s.render(w, errorStatus(err), p.View, p)
}
func (s *Server) runAutomationAction(ctx context.Context, a core.Actor, action *webAction) (string, error) {
	m, ok := s.board.(automationBoard)
	if !ok {
		return "", core.Fail("not_found", "Автоматический цикл недоступен")
	}
	var workflow core.WorkflowPage
	var policy core.AutomationPolicy
	var err error
	switch action.Operation {
	case "automation:policy-create":
		var in core.AutomationPolicyCreateInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			var c context.Context
			c, err = s.automationApproval(ctx, a, action, in.HostID, in.HostConsentVersion)
			if err == nil {
				policy, err = m.AutomationPolicyCreate(c, a, in)
			}
		}
	case "automation:policy-update":
		var in core.AutomationPolicyUpdateInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			var c context.Context
			c, err = s.automationApproval(ctx, a, action, in.HostID, in.HostConsentVersion)
			if err == nil {
				policy, err = m.AutomationPolicyUpdate(c, a, in)
			}
		}
	case "automation:policy-revoke":
		var in core.AutomationPolicyRevokeInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			policy, err = m.AutomationPolicyRevoke(ctx, a, in)
		}
	case "automation:submit":
		var in core.WorkflowSubmitInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			workflow, err = m.WorkflowSubmit(automationIntakeContext(ctx, a, action), a, in)
		}
	case "automation:command":
		var in core.WorkflowCommandInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			if in.RefreshPolicyID != "" {
				var current core.AutomationPolicyPage
				current, err = m.AutomationPolicyGet(ctx, a, core.AutomationPolicyInput{PolicyID: in.RefreshPolicyID, Version: in.RefreshPolicyVersion})
				if err == nil {
					ctx, err = s.automationApproval(ctx, a, action, current.Policy.HostID, current.Policy.HostConsentVersion)
				}
			}
			if err == nil {
				workflow, err = m.WorkflowCommand(ctx, a, in)
			}
		}
	case "automation:answer":
		var in core.WorkflowAnswerInput
		if err = json.Unmarshal([]byte(action.Payload), &in); err == nil {
			workflow, err = m.WorkflowAnswer(ctx, a, in)
		}
	default:
		return "", core.Fail("validation", "Неизвестная операция")
	}
	if err != nil {
		return "", err
	}
	if policy.ID != "" {
		return "/web/automation/policies/" + policy.ID, nil
	}
	return "/web/automation/workflows/" + workflow.Workflow.ID, nil
}

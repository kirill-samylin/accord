package web

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"encoding/json"
	"strings"
)

func (s *Server) automationApproval(ctx context.Context, a core.Actor, action *webAction, host string, consent int64) (context.Context, error) {
	epoch := int64(0)
	if s.options.ActionEpoch != nil {
		var err error
		epoch, err = s.options.ActionEpoch(ctx)
		if err != nil {
			return ctx, err
		}
	}
	return store.WithAutomationPolicyApproval(ctx, store.AutomationPolicyApproval{PrincipalID: a.Principal.ID, SessionID: a.SessionID, HostID: host, ConsentVersion: consent, Epoch: epoch, Source: "web", ActionID: action.ID}), nil
}
func (s *Server) checkAutomationPermission(ctx context.Context, a core.Actor, action *webAction) error {
	m, ok := s.board.(automationBoard)
	if !ok {
		return core.Fail("not_found", "Автоматический цикл недоступен")
	}
	var fields struct {
		HostID               string `json:"host_id"`
		Workspace            string `json:"workspace"`
		CommandKind          string `json:"kind"`
		QuestionID           string `json:"question_id"`
		HostConsentVersion   int64  `json:"host_consent_version"`
		PolicyID             string `json:"policy_id"`
		WorkflowID           string `json:"workflow_id"`
		RefreshPolicyID      string `json:"refresh_policy_id"`
		RefreshPolicyVersion int64  `json:"refresh_policy_version"`
	}
	if err := json.Unmarshal([]byte(action.Payload), &fields); err != nil {
		return err
	}
	kind := strings.TrimPrefix(action.Operation, "automation:")
	switch kind {
	case "policy-create":
		kind = "create"
	case "policy-update":
		kind = "update"
	case "policy-revoke":
		kind = "revoke"
	}
	if kind == "submit" {
		ctx = automationIntakeContext(ctx, a, action)
	}
	in := core.AutomationPermissionInput{
		PolicyID: fields.PolicyID, WorkflowID: fields.WorkflowID, HostID: fields.HostID, Kind: kind,
		Workspace: fields.Workspace, CommandKind: fields.CommandKind, RefreshPolicyID: fields.RefreshPolicyID,
		RefreshPolicyVersion: fields.RefreshPolicyVersion, QuestionID: fields.QuestionID,
	}
	if kind == "create" || kind == "update" {
		var err error
		ctx, err = s.automationApproval(ctx, a, action, fields.HostID, fields.HostConsentVersion)
		if err != nil {
			return err
		}
	}
	if kind == "command" && fields.RefreshPolicyID != "" {
		p, err := m.AutomationPolicyGet(ctx, a, core.AutomationPolicyInput{PolicyID: fields.RefreshPolicyID, Version: fields.RefreshPolicyVersion})
		if err != nil {
			return err
		}
		ctx, err = s.automationApproval(ctx, a, action, p.Policy.HostID, p.Policy.HostConsentVersion)
		if err != nil {
			return err
		}
	}
	return m.AutomationPermission(ctx, a, in)
}

func automationIntakeContext(ctx context.Context, a core.Actor, action *webAction) context.Context {
	return store.WithAutomationIntakeApproval(ctx, store.AutomationIntakeApproval{PrincipalID: a.Principal.ID, SessionID: a.SessionID, ActionID: action.ID})
}

package web

import (
	"agent-board/internal/core"
	"context"
	"net/http"
	"net/url"
	"strings"
)

type automationBoard interface {
	AutomationPermission(context.Context, core.Actor, core.AutomationPermissionInput) error
	AutomationPolicyCreate(context.Context, core.Actor, core.AutomationPolicyCreateInput) (core.AutomationPolicy, error)
	AutomationPolicyUpdate(context.Context, core.Actor, core.AutomationPolicyUpdateInput) (core.AutomationPolicy, error)
	AutomationPolicyRevoke(context.Context, core.Actor, core.AutomationPolicyRevokeInput) (core.AutomationPolicy, error)
	AutomationPolicyGet(context.Context, core.Actor, core.AutomationPolicyInput) (core.AutomationPolicyPage, error)
	WorkflowSubmit(context.Context, core.Actor, core.WorkflowSubmitInput) (core.WorkflowPage, error)
	WorkflowCommand(context.Context, core.Actor, core.WorkflowCommandInput) (core.WorkflowPage, error)
	WorkflowAnswer(context.Context, core.Actor, core.WorkflowAnswerInput) (core.WorkflowPage, error)
	WorkflowList(context.Context, core.Actor, core.WorkflowReadInput) (core.WorkflowListPage, error)
	WorkflowGet(context.Context, core.Actor, core.WorkflowReadInput) (core.WorkflowPage, error)
}

func automationRoute(path string) (kind, id string, ok bool) {
	if path == "/web/automation" {
		return "list", "", true
	}
	parts := strings.Split(strings.TrimPrefix(path, "/web/automation/"), "/")
	if len(parts) == 2 && (parts[0] == "workflows" || parts[0] == "policies") && parts[1] != "" {
		return parts[0], parts[1], true
	}
	return "", "", false
}
func (s *Server) automationPage(r *http.Request, b *browserSession, kind, id string) (Page, error) {
	p, err := s.page(r, b)
	if err != nil {
		return p, err
	}
	p.View, p.Title = "automation", "Выполнить требование"
	if p.CanContinue {
		return p, nil
	}
	input, err := automationReadQuery(r.URL.RawQuery)
	if err != nil {
		return p, err
	}
	m, ok := s.board.(automationBoard)
	if !ok {
		return p, core.Fail("not_found", "Автоматический цикл недоступен")
	}
	listInput := input
	if kind != "list" {
		listInput = core.WorkflowReadInput{Limit: 32}
	}
	list, err := m.WorkflowList(r.Context(), b.actor(), listInput)
	if err != nil {
		return p, err
	}
	p.Automation, p.Profiles = &list, p.AssignmentProfiles
	if kind == "workflows" {
		input.WorkflowID = id
		detail, e := m.WorkflowGet(r.Context(), b.actor(), input)
		if e != nil {
			return p, e
		}
		p.Workflow, p.View = &detail, "workflow"
	}
	if kind == "policies" {
		detail, e := m.AutomationPolicyGet(r.Context(), b.actor(), core.AutomationPolicyInput{PolicyID: id})
		if e != nil {
			return p, e
		}
		p.AutomationPolicy, p.View = &detail, "automation-policy"
	}
	canWrite := true
	if s.options.ProjectInfo != nil {
		project, e := s.options.ProjectInfo(r.Context())
		if e != nil {
			return p, e
		}
		p.Project, canWrite = project, project.CanWrite
	}
	if managed, supported := s.board.(managedBoard); supported {
		data, e := managed.ManagedList(r.Context(), b.actor(), core.ManagedListInput{})
		if e != nil {
			return p, e
		}
		for i := range data.Hosts {
			data.Hosts[i].CanControl = data.Hosts[i].CanControl && canWrite
			p.CanManageAutomation = p.CanManageAutomation || data.Hosts[i].CanControl && data.Hosts[i].State == "active"
		}
		p.Managed = &data
		if p.Workflow != nil {
			for _, step := range p.Workflow.Steps {
				if step.ID == p.Workflow.Workflow.CurrentStepID && step.ManagedRunID != "" {
					run, e := managed.ManagedRun(r.Context(), b.actor(), core.ManagedRunInput{RunID: step.ManagedRunID})
					if e != nil {
						return p, e
					}
					p.ManagedRun = &run
					break
				}
			}
		}
	}
	if p.AutomationPolicy != nil {
		p.CanManageAutomation = false
		if p.Managed != nil {
			for _, host := range p.Managed.Hosts {
				if host.ID == p.AutomationPolicy.Policy.HostID && host.CanControl && host.State == "active" && p.AutomationPolicy.Policy.IssuerPrincipalID == list.ActorPrincipalID {
					p.CanManageAutomation = true
				}
			}
		}
	}
	p.CanManageProfiles = canWrite
	if p.Workflow != nil {
		p.Workflow.CanControl = p.Workflow.CanControl && canWrite
		p.Workflow.CanResume = p.Workflow.CanResume && canWrite
	}
	if canWrite {
		operations := []string{"automation:submit"}
		if p.Workflow != nil && p.Workflow.CanControl {
			operations = append(operations, "automation:command", "automation:answer")
		}
		if p.CanManageAutomation {
			operations = append(operations, "automation:policy-create", "automation:policy-update", "automation:policy-revoke")
		}
		for _, op := range operations {
			p.Actions[op], err = s.issueAction(r.Context(), b, op)
			if err != nil {
				return p, err
			}
		}
		for _, role := range []string{"analyst", "developer", "reviewer", "tester"} {
			p.Actions["automation:team:"+role], err = s.issueAction(r.Context(), b, "profile:create")
			if err != nil {
				return p, err
			}
		}
	}
	return p, nil
}
func automationReadQuery(raw string) (core.WorkflowReadInput, error) {
	in := core.WorkflowReadInput{Limit: 32}
	q, err := url.ParseQuery(raw)
	if err != nil {
		return in, core.Fail("validation", "Неверные параметры URL")
	}
	for key, values := range q {
		if len(values) != 1 || key != "cursor" && key != "state" {
			return in, core.Fail("validation", "Неизвестный или повторный параметр")
		}
	}
	in.Cursor, in.State = q.Get("cursor"), q.Get("state")
	return in, nil
}

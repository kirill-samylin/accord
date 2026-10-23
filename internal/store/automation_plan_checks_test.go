package store

import (
	"agent-board/internal/core"
	"fmt"
	"testing"
)

func TestAutomationPolicyMandatoryChecks17And32HaveValidAnalystPlan(t *testing.T) {
	for _, count := range []int{17, 32} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			m := automationSetup(t)
			checks := []core.AutomationCheckPresetRef{}
			ids := []string{}
			for n := 0; n < count; n++ {
				c := m.check
				c.PresetID = fmt.Sprintf("unit-%02d", n)
				checks = append(checks, c)
				ids = append(ids, c.PresetID)
			}
			consent := *m.h.Automation
			consent.Version++
			scope := consent.Scopes["src"]
			scope.Version++
			scope.Checks = checks
			consent.Scopes = map[string]core.AutomationWorkspaceScope{"src": scope}
			host, err := m.f.s.ManagedHostUpdate(m.ctx, m.a, core.ManagedHostUpdateInput{ManagedHostEnrollInput: core.ManagedHostEnrollInput{RequestID: m.f.id(), Name: m.h.Name, HostSecret: managedTestSecret, Controllers: m.h.Controllers, Workspaces: m.h.Workspaces, Providers: m.h.Providers, Automation: &consent}, HostID: m.h.ID, Version: m.h.Version, State: "active"})
			if err != nil {
				t.Fatal("owner consent", err)
			}
			m.h = host
			m.ctx = WithAutomationPolicyApproval(m.ctx, AutomationPolicyApproval{PrincipalID: m.f.p.ID, SessionID: m.a.SessionID, Epoch: 0, HostID: m.h.ID, ConsentVersion: consent.Version, Source: "local", HostProof: managedTestSecret})
			settings := m.p.AutomationPolicySettings
			settings.HostConsentVersion = consent.Version
			settings.Checks = checks
			m.p, err = m.f.s.AutomationPolicyUpdate(m.ctx, m.a, core.AutomationPolicyUpdateInput{AutomationPolicyCreateInput: core.AutomationPolicyCreateInput{RequestID: m.f.id(), AutomationPolicySettings: settings}, PolicyID: m.p.ID, Version: m.p.Version})
			if err != nil {
				t.Fatal("policy with mandatory checks", err)
			}
			w := m.submit(t)
			m.tick(t, w.Workflow.ID)
			a := m.attempt(t, w.Workflow.ID)
			page := m.page(t, w.Workflow.ID)
			task := core.WorkflowPlanTask{ClientKey: "one", Title: "Implement requirement", Description: "Source and tests", Acceptance: []string{"verified"}, RequiredCheckIDs: ids}
			missing := task
			missing.RequiredCheckIDs = ids[:len(ids)-1]
			_, err = m.f.s.WorkflowPlan(a.ctx, a.actor, core.WorkflowPlanInput{RequestID: m.f.id(), WorkflowID: w.Workflow.ID, Version: page.Workflow.Version, TaskGeneration: a.claim.Lease.Generation, Tasks: []core.WorkflowPlanTask{missing}})
			requireCode(t, err, "VALIDATION")
			planned, err := m.f.s.WorkflowPlan(a.ctx, a.actor, core.WorkflowPlanInput{RequestID: m.f.id(), WorkflowID: w.Workflow.ID, Version: page.Workflow.Version, TaskGeneration: a.claim.Lease.Generation, Tasks: []core.WorkflowPlanTask{task}})
			if err != nil {
				t.Fatal("valid full mandatory plan rejected", err)
			}
			if len(planned.Plans) != 1 || len(planned.Plans[0].Tasks[0].RequiredCheckIDs) != count {
				t.Fatal("mandatory check omitted")
			}
			assertAutomationWire(t, planned)
		})
	}
}

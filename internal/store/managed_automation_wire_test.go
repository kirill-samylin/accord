package store

import (
	"agent-board/internal/core"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestManagedAutomationListSummaryAndExplicitScopeBoundNewFields(t *testing.T) {
	m := automationSetup(t)
	consent := *m.h.Automation
	consent.Scopes = map[string]core.AutomationWorkspaceScope{}
	aliases := []string{}
	for i := 0; i < 32; i++ {
		alias := fmt.Sprintf("ws-%02d", i)
		aliases = append(aliases, alias)
		scope := m.h.Automation.Scopes["src"]
		scope.Checks = nil
		for n := 0; n < 32; n++ {
			check := m.check
			check.PresetID = fmt.Sprint(n) + strings.Repeat("<", 120)
			scope.Checks = append(scope.Checks, check)
		}
		consent.Scopes[alias] = scope
	}
	_, err := transact(m.f.s, m.ctx, func(b *board, _ time.Time) (bool, error) {
		b.ManagedHosts[m.h.ID].Host.Automation = &consent
		b.ManagedHosts[m.h.ID].Host.Workspaces = aliases
		for n := 1; n < 32; n++ {
			id := fmt.Sprintf("host-%02d", n)
			b.ManagedHosts[id] = &managedHostRecord{Host: core.ManagedHost{ID: id, Automation: &consent, Workspaces: aliases}}
		}
		for n := 0; n < 128; n++ {
			id := fmt.Sprintf("run-%03d", n)
			b.ManagedRuns[id] = &core.ManagedRun{ID: id, HostID: m.h.ID, Automation: &core.ManagedAutomationBinding{WorkflowID: "workflow", StepID: "step", Checks: consent.Scopes["ws-00"].Checks}}
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := m.f.s.ManagedList(m.ctx, m.a, core.ManagedListInput{})
	if err != nil {
		t.Fatal(err)
	}
	assertAutomationWire(t, page)
	if len(page.Hosts) != 32 || len(page.Runs) != 128 {
		t.Fatal("list omitted hosts/runs")
	}
	for _, h := range page.Hosts {
		if !h.Automation.ScopesTruncated || len(h.Automation.Scopes) != 0 || h.Automation.Version != consent.Version || !h.Automation.Enabled {
			t.Fatal("summary masquerades as full consent")
		}
	}
	for _, r := range page.Runs {
		if len(r.Automation.Checks) != 0 || r.Automation.StepID != "step" {
			t.Fatal("run binding projection")
		}
	}
	selected, err := m.f.s.ManagedList(m.ctx, m.a, core.ManagedListInput{HostID: m.h.ID, Workspace: "ws-15"})
	if err != nil {
		t.Fatal(err)
	}
	assertAutomationWire(t, selected)
	c := selected.Hosts[0].Automation
	if c.ScopesTruncated || len(c.Scopes) != 1 || !reflect.DeepEqual(c.Scopes["ws-15"], consent.Scopes["ws-15"]) || c.Version != consent.Version {
		t.Fatal("explicit authoritative scope lost")
	}
	_, err = m.f.s.ManagedList(m.ctx, m.a, core.ManagedListInput{Workspace: "ws-00"})
	requireCode(t, err, "VALIDATION")
	_, err = m.f.s.ManagedList(m.ctx, m.a, core.ManagedListInput{HostID: m.h.ID, Workspace: "unknown"})
	requireCode(t, err, "NOT_FOUND")
	_, err = transact(m.f.s, m.ctx, func(b *board, _ time.Time) (bool, error) {
		if len(b.ManagedHosts[m.h.ID].Host.Automation.Scopes) != 32 || b.ManagedHosts[m.h.ID].Host.Automation.ScopesTruncated || len(b.ManagedRuns["run-000"].Automation.Checks) != 32 {
			t.Fatal("projection mutated durable grants/bindings")
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestManagedAutomationListDoesNotChangeNativePollBinding(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	m.tick(t, w.Workflow.ID)
	_, err := transact(m.f.s, m.ctx, func(b *board, _ time.Time) (bool, error) {
		// Exercise a binding with checks, as developer/tester stages carry.
		for _, r := range b.ManagedRuns {
			r.Automation.Checks = []core.AutomationCheckPresetRef{m.check}
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := m.f.s.ManagedList(m.ctx, m.a, core.ManagedListInput{})
	if err != nil || len(page.Runs) != 1 || len(page.Runs[0].Automation.Checks) != 0 {
		t.Fatal("summary binding", err)
	}
	a := m.attempt(t, w.Workflow.ID)
	if !reflect.DeepEqual(a.poll.Run.Automation.Checks, []core.AutomationCheckPresetRef{m.check}) {
		t.Fatal("full native binding was stripped")
	}
	partial := *m.h.Automation
	partial.ScopesTruncated = true
	err = automationConsentValid(&partial, m.h.Workspaces, m.h.Providers, m.f.clock.read())
	requireCode(t, err, "VALIDATION")
}

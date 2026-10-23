package store

import (
	"agent-board/internal/core"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestAutomationListEncodedBudgetIncludesPoliciesAndWorkflowSummaries(t *testing.T) {
	m := automationSetup(t)
	_, err := transact(m.f.s, m.ctx, func(b *board, _ time.Time) (bool, error) {
		for i := 0; i < 12; i++ {
			id := fmt.Sprintf("p-%02d", i)
			p := m.p
			p.ID = id
			p.IntakePrincipals = nil
			p.ControlPrincipals = nil
			for n := 0; n < 128; n++ {
				// IDs are bounded128 bytes; HTML escaping expands each '<'.
				p.IntakePrincipals = append(p.IntakePrincipals, fmt.Sprint(n)+strings.Repeat("<", 120))
				p.ControlPrincipals = append(p.ControlPrincipals, fmt.Sprint(n)+strings.Repeat("&", 120))
			}
			b.Automation.Policies[id] = &p
			w := &core.Workflow{ID: fmt.Sprintf("w-%02d", i), Version: 1, State: "draft", IntakeText: strings.Repeat("界", 4000), Grant: &core.WorkflowGrant{Profiles: map[string]core.ProfileSnapshot{"analyst": {ID: "p", Version: 1, ProfileSettings: core.ProfileSettings{Instructions: strings.Repeat("<", 16384)}}}}}
			b.Automation.Workflows[w.ID] = w
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var policies, workflows []string
	cursor := ""
	pages := 0
	for {
		page, err := m.f.s.WorkflowList(m.ctx, m.a, core.WorkflowReadInput{Cursor: cursor})
		if err != nil {
			t.Fatal(err)
		}
		assertAutomationWire(t, page)
		for _, p := range page.Policies {
			policies = append(policies, p.ID)
		}
		for _, w := range page.Workflows {
			if !w.IntakeTruncated || w.Grant.Profiles["analyst"].Instructions != "" {
				t.Fatal("list projection missing")
			}
			workflows = append(workflows, w.ID)
		}
		pages++
		if pages > 24 {
			t.Fatal("list cursor did not progress")
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if pages < 6 || len(policies) != 13 || len(workflows) != 12 {
		t.Fatal("policies/workflows skipped", pages, policies, workflows)
	}
	if policies[0] != m.p.ID {
		t.Fatal("original policy omitted")
	}
	for n := range workflows {
		if policies[n+1] != fmt.Sprintf("p-%02d", n) || workflows[n] != fmt.Sprintf("w-%02d", n) {
			t.Fatal("list order/duplicate")
		}
	}
	first, err := m.f.s.WorkflowList(m.ctx, m.a, core.WorkflowReadInput{Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	foreign := automationListCursor{Namespace: "another-project", State: "", Digest: "other"}
	_, err = m.f.s.WorkflowList(m.ctx, m.a, core.WorkflowReadInput{Cursor: encodeAutomationCursor(foreign)})
	requireCode(t, err, "VALIDATION")
	_, err = m.f.s.WorkflowList(m.ctx, m.a, core.WorkflowReadInput{Cursor: first.NextCursor, State: "draft"})
	requireCode(t, err, "VALIDATION")
	_, err = transact(m.f.s, m.ctx, func(b *board, _ time.Time) (bool, error) {
		b.Automation.Policies["p-00"].Version++
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.f.s.WorkflowList(m.ctx, m.a, core.WorkflowReadInput{Cursor: first.NextCursor})
	requireCode(t, err, "CONFLICT")
}

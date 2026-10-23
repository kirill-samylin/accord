package store

import (
	"agent-board/internal/core"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAutomationPolicyHistory128RevisionsBoundedWithoutSkippedItems(t *testing.T) {
	policy := core.AutomationPolicy{ID: "policy", Version: 128}
	policy.Name = strings.Repeat("<", 120)
	for n := 0; n < 128; n++ {
		policy.IntakePrincipals = append(policy.IntakePrincipals, fmt.Sprintf("%064x", n))
		policy.ControlPrincipals = append(policy.ControlPrincipals, fmt.Sprintf("%064x", n))
	}
	for n := 0; n < 32; n++ {
		policy.Checks = append(policy.Checks, core.AutomationCheckPresetRef{PresetID: fmt.Sprint(n) + strings.Repeat("<", 120), LocalDigest: strings.Repeat("a", 64), LocalVersion: 1, Kind: "test", Required: true, Isolation: "trusted_owner_process"})
	}
	var revisions []core.AutomationPolicy
	for n := 1; n <= 128; n++ {
		p := policy
		p.Version = int64(n)
		revisions = append(revisions, p)
	}
	before, _ := json.Marshal(revisions)
	cursor := ""
	seen := 0
	pages := 0
	for {
		page, err := automationPolicyPageWindow(policy, revisions, core.AutomationPolicyInput{Cursor: cursor}, "project-one")
		if err != nil {
			t.Fatal(err)
		}
		assertAutomationWire(t, page)
		if !reflect.DeepEqual(page.Policy, policy) {
			t.Fatal("current policy changed on historical page")
		}
		for _, revision := range page.Revisions {
			seen++
			if revision.Version != int64(seen) {
				t.Fatal("historical revision skipped/repeated", seen, revision.Version)
			}
		}
		pages++
		if pages > 128 {
			t.Fatal("cursor did not progress")
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if seen != 128 || pages < 10 {
		t.Fatal("history not fully paged", seen, pages)
	}
	after, _ := json.Marshal(revisions)
	if string(before) != string(after) {
		t.Fatal("immutable revision content changed")
	}
	filtered, err := automationPolicyPageWindow(policy, revisions[43:44], core.AutomationPolicyInput{Version: 44}, "project-one")
	if err != nil || len(filtered.Revisions) != 1 || filtered.Revisions[0].Version != 44 || filtered.Policy.Version != 128 || filtered.NextCursor != "" {
		t.Fatal("exact historical filter lost current policy", err)
	}
}

func TestAutomationPolicyGetCursorFencesAndExactRevisionFilter(t *testing.T) {
	m := automationSetup(t)
	_, err := transact(m.f.s, m.ctx, func(b *board, _ time.Time) (bool, error) {
		p := b.Automation.Policies[m.p.ID]
		for n := 2; n <= 4; n++ {
			p.Version = int64(n)
			revision := *p
			b.Automation.Revisions[fmt.Sprintf("%s/%d", p.ID, n)] = &revision
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := m.f.s.AutomationPolicyGet(m.ctx, m.a, core.AutomationPolicyInput{PolicyID: m.p.ID, Limit: 1})
	if err != nil || first.NextCursor == "" || first.Policy.Version != 4 || first.Revisions[0].Version != 1 {
		t.Fatal("first page", err)
	}
	second, err := m.f.s.AutomationPolicyGet(m.ctx, m.a, core.AutomationPolicyInput{PolicyID: m.p.ID, Cursor: first.NextCursor, Limit: 1})
	if err != nil || second.Revisions[0].Version != 2 || second.Policy.Version != 4 {
		t.Fatal("continuation", err)
	}
	exact, err := m.f.s.AutomationPolicyGet(m.ctx, m.a, core.AutomationPolicyInput{PolicyID: m.p.ID, Version: 3})
	if err != nil || len(exact.Revisions) != 1 || exact.Revisions[0].Version != 3 || exact.Policy.Version != 4 {
		t.Fatal("exact filter", err)
	}
	_, err = m.f.s.AutomationPolicyGet(m.ctx, m.a, core.AutomationPolicyInput{PolicyID: m.p.ID, Version: 3, Cursor: first.NextCursor})
	requireCode(t, err, "VALIDATION")
	_, err = automationPolicyPageWindow(first.Policy, first.Revisions, core.AutomationPolicyInput{Cursor: first.NextCursor}, "another-project")
	requireCode(t, err, "VALIDATION")
	foreign := automationPolicyCursor{PolicyID: "different-policy", Namespace: automationCursorNamespace(&board{Config: m.f.s.cfg}), CurrentVersion: 4}
	_, err = m.f.s.AutomationPolicyGet(m.ctx, m.a, core.AutomationPolicyInput{PolicyID: m.p.ID, Cursor: encodeAutomationCursor(foreign)})
	requireCode(t, err, "VALIDATION")
	_, err = transact(m.f.s, m.ctx, func(b *board, _ time.Time) (bool, error) { b.Automation.Policies[m.p.ID].Version++; return true, nil })
	if err != nil {
		t.Fatal(err)
	}
	_, err = m.f.s.AutomationPolicyGet(m.ctx, m.a, core.AutomationPolicyInput{PolicyID: m.p.ID, Cursor: first.NextCursor})
	requireCode(t, err, "CONFLICT")
}

func TestAutomationPolicyHistoryOversizedItemCannotDisappear(t *testing.T) {
	p := core.AutomationPolicy{ID: "policy", Version: 1}
	revision := p
	revision.Name = strings.Repeat("<", automationWireBudget)
	_, err := automationPolicyPageWindow(p, []core.AutomationPolicy{revision}, core.AutomationPolicyInput{}, "one")
	requireCode(t, err, "RESPONSE_CAPACITY")
}

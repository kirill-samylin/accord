package store

import (
	"agent-board/internal/core"
	"fmt"
	"reflect"
	"testing"
	"time"
)

func TestAutomationPolicySafetyRevokeAtRevisionCapAndRepeatedNewRequestsAreNoop(t *testing.T) {
	m := automationSetup(t)
	_, err := transact(m.f.s, m.ctx, func(b *board, _ time.Time) (bool, error) {
		p := b.Automation.Policies[m.p.ID]
		for n := 1; n <= 128; n++ {
			p.Version = int64(n)
			revision := *p
			b.Automation.Revisions[fmt.Sprintf("%s/%d", p.ID, n)] = &revision
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	in := core.AutomationPolicyRevokeInput{RequestID: m.f.id(), PolicyID: m.p.ID, Version: 128, Reason: "Safety revoke remains possible at update cap"}
	first, err := m.f.s.AutomationPolicyRevoke(m.ctx, m.a, in)
	if err != nil || first.State != "revoked" || first.Version != 129 {
		t.Fatal("first safety revoke at cap failed", first, err)
	}
	for n := 0; n < 5; n++ {
		m.f.clock.advance(time.Second)
		repeated, err := m.f.s.AutomationPolicyRevoke(m.ctx, m.a, core.AutomationPolicyRevokeInput{RequestID: m.f.id(), PolicyID: m.p.ID, Version: first.Version, Reason: "Already revoked; no additional revision"})
		if err != nil || !reflect.DeepEqual(first, repeated) {
			t.Fatal("already revoked policy changed", repeated, err)
		}
	}
	_, err = m.f.s.AutomationPolicyRevoke(m.ctx, m.a, core.AutomationPolicyRevokeInput{RequestID: m.f.id(), PolicyID: m.p.ID, Version: 128, Reason: "Stale version is not a fresh noop"})
	requireCode(t, err, "CONFLICT")
	denied := WithAutomationAuthority(m.ctx, AutomationAuthority{ProjectID: "test", Writable: false})
	_, err = m.f.s.AutomationPolicyRevoke(denied, m.a, in)
	requireCode(t, err, "FORBIDDEN_ROLE")
	replayed, err := m.f.s.AutomationPolicyRevoke(m.ctx, m.a, in)
	if err != nil || !reflect.DeepEqual(first, replayed) {
		t.Fatal("original committed receipt lost", err)
	}
	_, err = transact(m.f.s, m.ctx, func(b *board, _ time.Time) (bool, error) {
		count := 0
		for _, revision := range b.Automation.Revisions {
			if revision.ID == m.p.ID {
				count++
			}
		}
		if count != 129 || !reflect.DeepEqual(*b.Automation.Policies[m.p.ID], first) {
			t.Fatal("revoke history grew", count)
		}
		return true, nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

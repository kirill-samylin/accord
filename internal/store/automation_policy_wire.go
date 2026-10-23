package store

import (
	"agent-board/internal/core"
	"encoding/base64"
	"encoding/json"
	"sort"
)

type automationPolicyCursor struct {
	Namespace      string `json:"namespace"`
	PolicyID       string `json:"policy_id"`
	CurrentVersion int64  `json:"current_version"`
	FilterVersion  int64  `json:"filter_version"`
	Position       int    `json:"position"`
}

// Policy remains authoritative and current on every page. Historical revisions
// are immutable; changing current policy invalidates the page cursor explicitly.
func automationPolicyPageWindow(p core.AutomationPolicy, revisions []core.AutomationPolicy, in core.AutomationPolicyInput, namespace string) (core.AutomationPolicyPage, error) {
	out := core.AutomationPolicyPage{Policy: p, Revisions: []core.AutomationPolicy{}}
	limit := in.Limit
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 || in.Version < 0 {
		return out, core.Fail("validation", "Неверный policy limit/version")
	}
	sort.Slice(revisions, func(i, j int) bool { return revisions[i].Version < revisions[j].Version })
	cursor := automationPolicyCursor{Namespace: namespace, PolicyID: p.ID, CurrentVersion: p.Version, FilterVersion: in.Version}
	if in.Cursor != "" {
		raw, err := base64.RawURLEncoding.DecodeString(in.Cursor)
		var c automationPolicyCursor
		if err != nil || len(raw) > 768 || json.Unmarshal(raw, &c) != nil || c.Namespace != namespace || c.PolicyID != p.ID || c.FilterVersion != in.Version {
			return out, core.Fail("validation", "Cursor другого проекта/policy/filter")
		}
		if c.CurrentVersion != p.Version {
			return out, core.Fail("conflict", "Политика обновилась; откройте первую страницу")
		}
		if c.Position < 0 || c.Position > len(revisions) {
			return out, core.Fail("validation", "Неверная позиция policy cursor")
		}
		cursor = c
	}
	if automationWireSize(out)+automationCursorReserve > automationWireBudget {
		return out, core.Fail("response_capacity", "Текущая политика превышает размер страницы")
	}
	for cursor.Position < len(revisions) && len(out.Revisions) < limit {
		next := out
		next.Revisions = append(next.Revisions, revisions[cursor.Position])
		if automationWireSize(next)+automationCursorReserve > automationWireBudget {
			if len(out.Revisions) == 0 {
				return out, core.Fail("response_capacity", "Одна ревизия с текущей политикой превышает размер страницы")
			}
			break
		}
		out = next
		cursor.Position++
	}
	if cursor.Position < len(revisions) {
		out.NextCursor = encodeAutomationCursor(cursor)
	}
	if automationWireSize(out) > automationWireBudget {
		return core.AutomationPolicyPage{}, core.Fail("response_capacity", "Страница превышает wire budget")
	}
	return out, nil
}

package store

import (
	"agent-board/internal/core"
	"context"
	"encoding/base64"
	"encoding/json"
	"sort"
	"time"
)

type automationListCursor struct {
	Namespace           string `json:"namespace"`
	State               string `json:"state"`
	Digest              string `json:"digest"`
	Policies, Workflows int
}

func (s *Store) WorkflowList(ctx context.Context, a core.Actor, in core.WorkflowReadInput) (core.WorkflowListPage, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.WorkflowListPage, error) {
		out := core.WorkflowListPage{ObservedAt: now, ActorPrincipalID: a.Principal.ID, Policies: []core.AutomationPolicy{}, Workflows: []core.Workflow{}}
		if _, _, err := automationActor(ctx, b, a, now, false); err != nil {
			return out, err
		}
		if _, err := sessionFor(b, a, now); err != nil {
			return out, err
		}
		if _, ok := ExecutionIdentityFromContext(ctx); ok {
			return out, core.Fail("forbidden", "Дочерняя сессия не перечисляет workflow")
		}
		limit := in.Limit
		if limit == 0 {
			limit = 32
		}
		if limit < 1 || limit > 50 || invalidCursorFilter(in.State) {
			return out, core.Fail("validation", "Неверный list filter/limit")
		}
		policies := []core.AutomationPolicy{}
		flows := []core.Workflow{}
		for _, p := range b.Automation.Policies {
			policies = append(policies, *p)
		}
		for _, w := range b.Automation.Workflows {
			if in.State == "" || in.State == w.State {
				flows = append(flows, wireWorkflow(w, true))
			}
		}
		sort.Slice(policies, func(i, j int) bool { return policies[i].ID < policies[j].ID })
		sort.Slice(flows, func(i, j int) bool { return flows[i].ID < flows[j].ID })
		facts := []any{}
		for _, p := range policies {
			facts = append(facts, []any{p.ID, p.Version, p.State})
		}
		for _, w := range flows {
			facts = append(facts, []any{w.ID, w.Version, w.State})
		}
		raw, _ := json.Marshal(facts)
		cursor := automationListCursor{Namespace: automationCursorNamespace(b), State: in.State, Digest: managedHash(string(raw))}
		if in.Cursor != "" {
			data, e := base64.RawURLEncoding.DecodeString(in.Cursor)
			if e != nil || len(data) > 768 {
				return out, core.Fail("validation", "Неверный list cursor")
			}
			var c automationListCursor
			if json.Unmarshal(data, &c) != nil || c.Namespace != cursor.Namespace || c.State != cursor.State {
				return out, core.Fail("validation", "Cursor другого проекта/filter")
			}
			if c.Digest != cursor.Digest {
				return out, core.Fail("conflict", "Список обновился; откройте первую страницу")
			}
			if c.Policies < 0 || c.Policies > len(policies) || c.Workflows < 0 || c.Workflows > len(flows) {
				return out, core.Fail("validation", "Неверная позиция list cursor")
			}
			cursor = c
		}
		counts := [2]int{}
		blocked := false
		used := 0
		for !blocked {
			progress := false
			for i := 0; i < 2; i++ {
				if counts[i] >= limit || i == 0 && cursor.Policies >= len(policies) || i == 1 && cursor.Workflows >= len(flows) {
					continue
				}
				next := out
				if i == 0 {
					next.Policies = append(next.Policies, policies[cursor.Policies])
				} else {
					next.Workflows = append(next.Workflows, flows[cursor.Workflows])
				}
				if automationWireSize(next)+automationCursorReserve > automationWireBudget {
					if used == 0 {
						return out, core.Fail("response_capacity", "Один элемент списка превышает размер страницы")
					}
					blocked = true
					break
				}
				out = next
				if i == 0 {
					cursor.Policies++
				} else {
					cursor.Workflows++
				}
				counts[i]++
				used++
				progress = true
			}
			if !progress {
				break
			}
		}
		if cursor.Policies < len(policies) || cursor.Workflows < len(flows) {
			out.NextCursor = encodeAutomationCursor(cursor)
		}
		if automationWireSize(out) > automationWireBudget {
			return out, core.Fail("response_capacity", "Страница превышает wire budget")
		}
		return out, nil
	})
}

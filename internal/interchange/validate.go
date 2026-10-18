package interchange

import (
	"errors"
	"strings"
)

func validateData(r Record) error {
	invalid := errors.New("invalid data")
	switch r.Kind {
	case "task":
		var v Task
		if strict(r.Data, &v) != nil {
			return invalid
		}
		if v.Title == "" || v.Role == "" || v.Type != "task" {
			return invalid
		}
		switch v.State {
		case "created", "ready", "in_progress", "done", "cancelled":
		default:
			return invalid
		}
	case "epic":
		var v Epic
		if strict(r.Data, &v) != nil || v.Title == "" {
			return invalid
		}
		switch v.State {
		case "open", "done", "cancelled":
		default:
			return invalid
		}
	case "link":
		var v Link
		if strict(r.Data, &v) != nil {
			return invalid
		}
		switch v.Type {
		case "blocks", "contract_of", "spawned_by", "related", "duplicate_of":
		default:
			return invalid
		}
		switch v.Condition {
		case "", "completed", "approved", "passed":
		default:
			return invalid
		}
	case "note":
		var v Note
		if strict(r.Data, &v) != nil || v.Text == "" || v.Kind == "" {
			return invalid
		}
	case "history":
		var v History
		if strict(r.Data, &v) != nil || v.Type == "" || v.At.IsZero() || v.ConfigVersion < 0 || (v.TokensSpent != nil && *v.TokensSpent < 0) {
			return invalid
		}
		if v.OriginSessionLabel != "" && !strings.Contains(v.OriginSessionLabel, ":") {
			return invalid
		}
	default:
		return invalid
	}
	return nil
}

// ValidateReferences permits existing-source references only through an
// authoritative mapping callback. It never interprets external IDs as local keys.
func ValidateReferences(records []Record, exists func(Ref) bool) []Diagnostic {
	known := map[string]bool{}
	graph := map[string][]string{}
	result := []Diagnostic{}
	for _, r := range records {
		known[identity(r.Kind, r.ExternalID)] = true
	}
	add := func(r Record, path, code string) {
		if len(result) < 100 {
			result = append(result, Diagnostic{r.Line, path, code})
		}
	}
	check := func(r Record, p string, ref Ref, allowed ...string) {
		valid := false
		for _, kind := range allowed {
			if ref.Kind == kind {
				valid = true
			}
		}
		if !valid || ref.ExternalID == "" || len(ref.ExternalID) > 256 || (!known[identity(ref.Kind, ref.ExternalID)] && (exists == nil || !exists(ref))) {
			add(r, p, "REFERENCE")
		}
	}
	for _, r := range records {
		switch r.Kind {
		case "task":
			var v Task
			if strict(r.Data, &v) != nil {
				continue
			}
			if v.Epic != nil {
				check(r, "data.epic", *v.Epic, "epic")
			}
			if v.Target != nil {
				check(r, "data.target", *v.Target, "task")
			}
		case "note":
			var v Note
			if strict(r.Data, &v) != nil {
				continue
			}
			if v.Epic != nil {
				check(r, "data.epic", *v.Epic, "epic")
			}
		case "history":
			var v History
			if strict(r.Data, &v) != nil {
				continue
			}
			check(r, "data.origin_ref", v.OriginRef, "epic", "task", "note", "link")
		case "link":
			var v Link
			if strict(r.Data, &v) != nil {
				continue
			}
			check(r, "data.from", v.From, "epic", "task")
			check(r, "data.to", v.To, "epic", "task")
			if v.Type == "blocks" || v.Type == "contract_of" || v.Type == "spawned_by" {
				a, b := identity(v.From.Kind, v.From.ExternalID), identity(v.To.Kind, v.To.ExternalID)
				graph[a] = append(graph[a], b)
			}
		}
	}
	// Iterative DFS keeps adversarial 10k-node chains off the call stack.
	color := map[string]uint8{}
	type frame struct {
		node string
		next int
	}
	for node := range graph {
		if color[node] != 0 {
			continue
		}
		stack := []frame{{node: node}}
		color[node] = 1
		for len(stack) > 0 {
			top := &stack[len(stack)-1]
			edges := graph[top.node]
			if top.next == len(edges) {
				color[top.node] = 2
				stack = stack[:len(stack)-1]
				continue
			}
			to := edges[top.next]
			top.next++
			if color[to] == 1 {
				result = append(result, Diagnostic{0, "links", "CYCLE"})
				return result
			}
			if color[to] == 0 {
				color[to] = 1
				stack = append(stack, frame{node: to})
			}
		}
	}
	return result
}

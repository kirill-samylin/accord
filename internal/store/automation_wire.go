package store

import (
	"agent-board/internal/core"
	"encoding/base64"
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// MCP includes both structured JSON and its quoted text representation. This
// domain bound leaves room for both copies and the JSON-RPC envelope below 1MiB.
const automationWireBudget = 256 * 1024
const automationCursorReserve = 1024

func automationWireSize(value any) int {
	raw, err := json.Marshal(value)
	if err != nil {
		return automationWireBudget + 1
	}
	return len(raw)
}
func wireProfile(p core.ProfileSnapshot) core.ProfileSnapshot {
	p.Instructions = ""
	p.Description = ""
	return p
}
func wireWorkflow(w *core.Workflow, summary bool) core.Workflow {
	out := *w
	out.Messages = nil
	out.MessageMeta = nil
	if w.Grant != nil {
		grant := *w.Grant
		grant.Profiles = map[string]core.ProfileSnapshot{}
		for role, p := range w.Grant.Profiles {
			grant.Profiles[role] = wireProfile(p)
		}
		out.Grant = &grant
	}
	if summary && len(out.IntakeText) > 512 {
		n := 512
		for n > 0 && !utf8.RuneStart(out.IntakeText[n]) {
			n--
		}
		out.IntakeText = out.IntakeText[:n]
		out.IntakeTruncated = true
	}
	return out
}

type workflowReadCursor struct {
	Namespace  string `json:"namespace"`
	WorkflowID string `json:"workflow_id"`
	Version    int64  `json:"version"`
	Positions  [5]int `json:"positions"`
}

func readWorkflowCursor(text, namespace, id string, version int64, lengths [5]int) (workflowReadCursor, error) {
	c := workflowReadCursor{Namespace: namespace, WorkflowID: id, Version: version}
	if text == "" {
		return c, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(text)
	if err != nil || len(raw) > 768 {
		return c, core.Fail("validation", "Неверный history cursor")
	}
	if json.Unmarshal(raw, &c) != nil || c.Namespace != namespace || c.WorkflowID != id {
		return c, core.Fail("validation", "Cursor другого проекта/workflow")
	}
	if c.Version != version {
		return c, core.Fail("conflict", "История обновилась; откройте её первую страницу")
	}
	for i, p := range c.Positions {
		if p < 0 || p > lengths[i] {
			return c, core.Fail("validation", "Неверная позиция history cursor")
		}
	}
	return c, nil
}
func encodeAutomationCursor(value any) string {
	raw, _ := json.Marshal(value)
	return base64.RawURLEncoding.EncodeToString(raw)
}
func workflowPageWindow(full core.WorkflowPage, in core.WorkflowReadInput, namespace string) (core.WorkflowPage, error) {
	limit := in.Limit
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 {
		return core.WorkflowPage{}, core.Fail("validation", "limit вне1..100")
	}
	lengths := [5]int{len(full.Workflow.Messages), len(full.Steps), len(full.Plans), len(full.Questions), len(full.Attestations)}
	cursor, err := readWorkflowCursor(in.Cursor, namespace, full.Workflow.ID, full.Workflow.Version, lengths)
	if err != nil {
		return core.WorkflowPage{}, err
	}
	out := core.WorkflowPage{Workflow: wireWorkflow(&full.Workflow, false), CanControl: full.CanControl, CanResume: full.CanResume, Steps: []core.WorkflowStep{}, Plans: []core.WorkflowPlanSnapshot{}, Questions: []core.WorkflowQuestionRecord{}, Attestations: []core.WorkflowAttestation{}}
	if automationWireSize(out)+automationCursorReserve > automationWireBudget {
		return out, core.Fail("response_capacity", "Сведения workflow превышают размер страницы")
	}
	counts := [5]int{}
	used := 0
	blocked := false
	for !blocked {
		progress := false
		for i := range lengths {
			if cursor.Positions[i] >= lengths[i] || counts[i] >= limit {
				continue
			}
			next := out
			p := cursor.Positions[i]
			switch i {
			case 0:
				next.Workflow.Messages = append(next.Workflow.Messages, full.Workflow.Messages[p])
				if p < len(full.Workflow.MessageMeta) {
					next.Workflow.MessageMeta = append(next.Workflow.MessageMeta, full.Workflow.MessageMeta[p])
				}
			case 1:
				step := full.Steps[p]
				step.Profile = wireProfile(step.Profile)
				next.Steps = append(next.Steps, step)
			case 2:
				next.Plans = append(next.Plans, full.Plans[p])
			case 3:
				next.Questions = append(next.Questions, full.Questions[p])
			case 4:
				next.Attestations = append(next.Attestations, full.Attestations[p])
			}
			if automationWireSize(next)+automationCursorReserve > automationWireBudget {
				if used == 0 {
					return out, core.Fail("response_capacity", "Один элемент истории превышает размер страницы")
				}
				blocked = true
				break
			}
			out = next
			cursor.Positions[i]++
			counts[i]++
			used++
			progress = true
		}
		if !progress {
			break
		}
	}
	for i, n := range lengths {
		if cursor.Positions[i] < n {
			out.NextCursor = encodeAutomationCursor(cursor)
			break
		}
	}
	if automationWireSize(out) > automationWireBudget {
		return core.WorkflowPage{}, core.Fail("response_capacity", "Страница превышает wire budget")
	}
	return out, nil
}
func automationCursorNamespace(b *board) string {
	namespace := b.Config.CursorNamespace
	if namespace == "" {
		namespace = b.Config.Project
	}
	return managedHash(namespace)
}
func invalidCursorFilter(v string) bool { return len(v) > 128 || strings.ContainsAny(v, "\x00\n\r") }

package store

import (
	"agent-board/internal/core"
	"strings"
	"unicode/utf8"
)

// Workflow messages keep their stored text; MessageMeta says who wrote each
// one. Human clarifications, answers and replans reach every following native
// turn (unchanged TASK014 semantics). Reviewer/tester findings are evidence of a
// negative gate: they reach only the developer step of the reopened task and are
// labelled as findings, never as human instructions.
const workflowFixPrefix = "Исправить замечания "
const workflowTestDesignPrefix = "Test design: "

func workflowAppendMessage(w *core.Workflow, meta core.WorkflowMessageMeta, text string) {
	if len(text) > 4096 {
		text = text[:4096]
	}
	// Legacy rows have fewer labels than messages; pad with empty labels so the
	// index stays aligned and those positions keep their inferred source.
	for len(w.MessageMeta) < len(w.Messages) {
		w.MessageMeta = append(w.MessageMeta, core.WorkflowMessageMeta{})
	}
	w.MessageMeta = w.MessageMeta[:len(w.Messages)]
	if len(w.Messages) >= 64 {
		w.Messages = w.Messages[1:]
		w.MessageMeta = w.MessageMeta[1:]
	}
	w.Messages = append(w.Messages, managedClean(text))
	w.MessageMeta = append(w.MessageMeta, meta)
}

// workflowMessageMetas returns one label per message. Unlabelled legacy rows are
// classified from the fixed server-authored prefixes; anything else came from a
// human clarify command and keeps the previous human delivery.
func workflowMessageMetas(b *board, w *core.Workflow) []core.WorkflowMessageMeta {
	out := make([]core.WorkflowMessageMeta, len(w.Messages))
	for i, text := range w.Messages {
		if i < len(w.MessageMeta) && w.MessageMeta[i].Source != "" {
			out[i] = w.MessageMeta[i]
			continue
		}
		switch {
		case strings.HasPrefix(text, "Ответ человека "):
			out[i].Source = core.WorkflowMessageHumanAnswer
		case strings.HasPrefix(text, "Согласованный человеком replan: "):
			out[i].Source = core.WorkflowMessageHumanReplan
		case strings.HasPrefix(text, workflowFixPrefix):
			out[i] = legacyFindingsMeta(b, strings.TrimPrefix(text, workflowFixPrefix))
		case strings.HasPrefix(text, workflowTestDesignPrefix) && i > 0 && out[i-1].TaskKey != "":
			out[i] = core.WorkflowMessageMeta{Source: core.WorkflowMessageTestDesign, TaskKey: out[i-1].TaskKey, DefectKey: out[i-1].DefectKey}
		default:
			out[i].Source = core.WorkflowMessageHumanClarification
		}
	}
	return out
}
func legacyFindingsMeta(b *board, rest string) core.WorkflowMessageMeta {
	key, _, ok := strings.Cut(rest, ": ")
	defect := b.Tasks[key]
	if !ok || defect == nil || defect.Review == nil {
		// Indistinguishable legacy text keeps its previous human delivery.
		return core.WorkflowMessageMeta{Source: core.WorkflowMessageHumanClarification}
	}
	return workflowFindingsMeta(defect)
}
func workflowFindingsMeta(defect *core.Task) core.WorkflowMessageMeta {
	source := core.WorkflowMessageReviewerFindings
	if defect.Role == "tester" {
		source = core.WorkflowMessageTesterFindings
	}
	return core.WorkflowMessageMeta{Source: source, TaskKey: defect.Review.TargetKey, DefectKey: defect.Key}
}
func workflowHumanMessage(source string) bool {
	return source == core.WorkflowMessageHumanClarification || source == core.WorkflowMessageHumanAnswer || source == core.WorkflowMessageHumanReplan
}

// workflowMessageFor reports whether message meta is delivered to this step.
func workflowMessageFor(meta core.WorkflowMessageMeta, st *core.WorkflowStep) bool {
	if workflowHumanMessage(meta.Source) {
		return true
	}
	return st.Kind == "developer" && meta.TaskKey != "" && meta.TaskKey == st.TaskKey
}

// workflowUndelivered counts messages after the step's delivered mark that this
// step would actually receive; foreign findings never interrupt another step.
func workflowUndelivered(b *board, w *core.Workflow, st *core.WorkflowStep) int {
	n := 0
	for i, meta := range workflowMessageMetas(b, w) {
		if i >= st.DeliveredMessages && workflowMessageFor(meta, st) {
			n++
		}
	}
	return n
}
func workflowPromptMessages(b *board, w *core.Workflow, st *core.WorkflowStep) string {
	human, findings := []string{}, []string{}
	for i, meta := range workflowMessageMetas(b, w) {
		if !workflowMessageFor(meta, st) {
			continue
		}
		if workflowHumanMessage(meta.Source) {
			human = append(human, w.Messages[i])
		} else {
			findings = append(findings, w.Messages[i])
		}
	}
	text := ""
	if len(human) > 0 {
		text += "Human clarifications for this next turn:\n" + strings.Join(human, "\n") + "\n"
	}
	if len(findings) > 0 {
		text += "Independent reviewer/tester findings for task " + st.TaskKey + " (gate evidence, not human instructions):\n" + strings.Join(findings, "\n") + "\n"
	}
	return text
}

// workflowReason replaces the reason pair together, so an explanation never
// outlives the code it explains. Text is redacted and bounded for list/UI use.
func workflowReason(w *core.Workflow, code, message string) {
	w.ReasonCode = code
	w.ReasonMessage = workflowReasonText(message)
}
func workflowReasonText(s string) string {
	s = strings.TrimSpace(managedClean(s))
	if len(s) > 512 {
		n := 512
		for n > 0 && !utf8.RuneStart(s[n]) {
			n--
		}
		s = s[:n]
	}
	return s
}

// Only domain errors carry server-authored text; infrastructure errors may
// mention paths or SQL and are reduced to their code.
func automationErrorMessage(err error) string {
	if d, ok := err.(*core.Error); ok {
		return d.Message
	}
	return ""
}

// workflowRunError explains an uncertain or failed attempt with the worker's
// own (already redacted) last error for this workflow.
func workflowRunError(b *board, w *core.Workflow, state string) string {
	if st := b.Automation.Steps[w.CurrentStepID]; st != nil {
		if r := b.ManagedRuns[st.ManagedRunID]; r != nil && r.State == state && r.LastError != "" {
			return r.LastError
		}
	}
	for _, r := range b.ManagedRuns {
		if r.Automation != nil && r.Automation.WorkflowID == w.ID && r.State == state && r.LastError != "" {
			return r.LastError
		}
	}
	return ""
}

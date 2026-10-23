package store

import (
	"agent-board/internal/core"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// This exercises the production MCP result shape, not a provider simulation.
func assertAutomationWire(t *testing.T, value any) {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil || len(raw) > automationWireBudget {
		t.Fatalf("domain encoded size=%d: %v", len(raw), err)
	}
	result := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(raw)}}, StructuredContent: value}
	envelope, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": "test", "result": result})
	if err != nil || len(envelope) >= 1<<20 {
		t.Fatalf("MCP encoded size=%d: %v", len(envelope), err)
	}
}

func wireHistoryFixture() core.WorkflowPage {
	p := core.ProfileSnapshot{ID: "profile", Version: 7, ProfileSettings: core.ProfileSettings{Role: "reviewer", Name: "Preserved identity", Instructions: strings.Repeat("<", 16384), Description: strings.Repeat("&", 4000)}}
	full := core.WorkflowPage{Workflow: core.Workflow{ID: "workflow", Version: 9, IntakeText: strings.Repeat("<", 16384), Grant: &core.WorkflowGrant{Profiles: map[string]core.ProfileSnapshot{"reviewer": p}}}}
	for i := 0; i < 24; i++ {
		id := fmt.Sprintf("item-%02d", i)
		full.Workflow.Messages = append(full.Workflow.Messages, id+strings.Repeat("<", 4000))
		full.Steps = append(full.Steps, core.WorkflowStep{ID: id, Profile: p})
		full.Plans = append(full.Plans, core.WorkflowPlanSnapshot{Revision: int64(i), Tasks: []core.WorkflowPlanTask{{ClientKey: id, Description: strings.Repeat("<", 6000)}}})
		full.Questions = append(full.Questions, core.WorkflowQuestionRecord{ID: id, Question: strings.Repeat("&", 4000)})
		full.Attestations = append(full.Attestations, core.WorkflowAttestation{ID: id, Observations: core.WorkflowObservations{ArtifactRefs: []string{strings.Repeat("<", 4000)}}})
	}
	return full
}

func TestAutomationHistoryEncodedBudgetAllCollectionsWithoutSkippedItems(t *testing.T) {
	full := wireHistoryFixture()
	before, _ := json.Marshal(full)
	var seen [5][]string
	cursor := ""
	pages := 0
	for {
		page, err := workflowPageWindow(full, core.WorkflowReadInput{Cursor: cursor, Limit: 50}, "project-a")
		if err != nil {
			t.Fatal(err)
		}
		assertAutomationWire(t, page)
		if page.Workflow.IntakeText != full.Workflow.IntakeText || page.Workflow.Grant.Profiles["reviewer"].Instructions != "" {
			t.Fatal("detail intake or profile projection changed")
		}
		for _, v := range page.Workflow.Messages {
			seen[0] = append(seen[0], v[:7])
		}
		for _, v := range page.Steps {
			if v.Profile.ID != "profile" || v.Profile.Version != 7 || v.Profile.Instructions != "" || v.Profile.Description != "" {
				t.Fatal("step profile identity/projection")
			}
			seen[1] = append(seen[1], v.ID)
		}
		for _, v := range page.Plans {
			seen[2] = append(seen[2], v.Tasks[0].ClientKey)
		}
		for _, v := range page.Questions {
			seen[3] = append(seen[3], v.ID)
		}
		for _, v := range page.Attestations {
			seen[4] = append(seen[4], v.ID)
		}
		pages++
		if pages > 100 {
			t.Fatal("cursor did not progress")
		}
		if page.NextCursor == "" {
			break
		}
		cursor = page.NextCursor
	}
	if pages < 3 {
		t.Fatal("escaped payload did not exercise budget boundary")
	}
	for i, items := range seen {
		if len(items) != 24 {
			t.Fatalf("collection %d count=%d", i, len(items))
		}
		for n, id := range items {
			if id != fmt.Sprintf("item-%02d", n) {
				t.Fatalf("collection %d skipped/repeated %d: %q", i, n, id)
			}
		}
	}
	after, _ := json.Marshal(full)
	if string(before) != string(after) {
		t.Fatal("wire projection mutated immutable history")
	}
}

func TestAutomationHistoryCursorNamespaceVersionAndOversizedItem(t *testing.T) {
	full := wireHistoryFixture()
	page, err := workflowPageWindow(full, core.WorkflowReadInput{Limit: 1}, "one")
	if err != nil || page.NextCursor == "" {
		t.Fatal(err)
	}
	_, err = workflowPageWindow(full, core.WorkflowReadInput{Cursor: page.NextCursor}, "two")
	requireCode(t, err, "VALIDATION")
	full.Workflow.Version++
	_, err = workflowPageWindow(full, core.WorkflowReadInput{Cursor: page.NextCursor}, "one")
	requireCode(t, err, "CONFLICT")
	full = core.WorkflowPage{Workflow: core.Workflow{ID: "oversized", Version: 1, Messages: []string{strings.Repeat("<", automationWireBudget)}}}
	_, err = workflowPageWindow(full, core.WorkflowReadInput{}, "one")
	requireCode(t, err, "RESPONSE_CAPACITY")
}

func TestAutomationMutationWireKeepsImmutablePromptAndUTF8ListPreview(t *testing.T) {
	m := automationSetup(t)
	intake := strings.Repeat("界", 4000)
	page, err := m.f.s.WorkflowSubmit(m.ctx, m.a, core.WorkflowSubmitInput{RequestID: m.f.id(), PolicyID: m.p.ID, PolicyVersion: m.p.Version, Text: intake})
	if err != nil {
		t.Fatal(err)
	}
	assertAutomationWire(t, page)
	if page.Workflow.IntakeText != intake {
		t.Fatal("mutation lost original intake")
	}
	var original core.ProfileSnapshot
	_, err = transact(m.f.s, m.ctx, func(b *board, _ time.Time) (bool, error) {
		original = b.Automation.Workflows[page.Workflow.ID].Grant.Profiles["analyst"]
		return true, nil
	})
	if err != nil || original.Instructions == "" {
		t.Fatal("stored native instructions missing", err)
	}
	if page.Workflow.Grant.Profiles["analyst"].Instructions != "" {
		t.Fatal("mutation leaked full profile")
	}
	list, err := m.f.s.WorkflowList(m.ctx, m.a, core.WorkflowReadInput{})
	if err != nil {
		t.Fatal(err)
	}
	assertAutomationWire(t, list)
	w := list.Workflows[0]
	if !w.IntakeTruncated || !utf8.ValidString(w.IntakeText) || len(w.IntakeText) > 512 || !strings.HasPrefix(intake, w.IntakeText) {
		t.Fatal("invalid UTF8 preview")
	}
	if !reflect.DeepEqual(wireProfile(original), page.Workflow.Grant.Profiles["analyst"]) {
		t.Fatal("profile identity lost")
	}
}

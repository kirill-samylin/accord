package worker

import (
	"agent-board/internal/core"
	"encoding/json"
	"strings"
	"testing"
)

func TestCompletedAutomationHistoryFitsJournalWithoutPruningUnacknowledgedFacts(t *testing.T) {
	profile := &core.ProfileSnapshot{ID: "profile", Version: 7, ProfileSettings: core.ProfileSettings{Role: "developer", Provider: "openai", Instructions: strings.Repeat("reviewed instructions ", 700), Description: strings.Repeat("description", 100)}}
	original := core.ManagedRun{ID: "run", Generation: 2, Prompt: strings.Repeat("large bounded requirement context ", 8000), Profile: profile, Automation: &core.ManagedAutomationBinding{WorkflowID: "wf", StepID: "step", ControlEpoch: 3, TaskGeneration: 4}}
	a := &automationAttempt{Run: original, NativeExited: true, ExitDurable: true, ManagedAck: true, ExitAck: false, LockReleased: false, Pending: []*core.WorkflowAttestInput{{Kind: "run_exit"}}}
	if compactAutomation(a) || a.Run.Prompt != original.Prompt || a.Run.Profile.Instructions != profile.Instructions {
		t.Fatal("unacknowledged recovery facts were compacted")
	}
	a.Pending = nil
	a.ExitAck = true
	a.LockReleased = true
	if !compactAutomation(a) {
		t.Fatal("fully acknowledged duplicate context was retained")
	}
	if len(a.RunSnapshotSHA256) != 64 || len(a.ProfileSnapshotSHA256) != 64 || a.Run.ID != original.ID || a.Run.Generation != 2 || a.Run.Profile.ID != "profile" || a.Run.Profile.Version != 7 || a.Run.Automation.ControlEpoch != 3 || a.Run.Automation.TaskGeneration != 4 {
		t.Fatal("immutable authority identity or proof hash lost")
	}
	if profile.Instructions == "" {
		t.Fatal("compaction mutated a shared saved profile")
	}
	if a.Run.Prompt != "" || a.Run.Profile.Instructions != "" {
		t.Fatal("heavy duplicate text retained")
	}
	history := map[string]*automationAttempt{}
	for i := 0; i < 128; i++ {
		history[string(rune(i+1))] = a
	}
	raw, err := json.Marshal(history)
	if err != nil || len(raw) >= 8<<20 {
		t.Fatal("completed bounded managed history exceeded the private journal", len(raw), err)
	}
}

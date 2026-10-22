package worker

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestManagedBridgeFilesRemainBoundToImmutableExecutionAttempt(t *testing.T) {
	for _, provider := range []string{"codex", "claude"} {
		t.Run(provider, func(t *testing.T) {
			e := testEngine(t)
			run := RunState{ID: "run_binding", Provider: provider, Workspace: "test", Generation: 1}
			args, err := e.bridgeArgs(run)
			if err != nil {
				t.Fatal(err)
			}
			first := filepath.Join(e.config.StateDir, "runs", run.ID, "attempt-1", "bridge.json")
			before, err := os.ReadFile(first)
			if err != nil {
				t.Fatal(err)
			}
			config, _, err := ReadBridgeConfig(first)
			if err != nil || config.ManagedRunID != run.ID || config.ManagedGeneration != 1 {
				t.Fatal("first binding missing", err)
			}
			same, err := e.bridgeArgs(run)
			if err != nil || !reflect.DeepEqual(same, args) {
				t.Fatal("exact bridge preparation retry changed args", err)
			}
			run.Generation = 2
			if _, err = e.bridgeArgs(run); err != nil {
				t.Fatal(err)
			}
			after, err := os.ReadFile(first)
			if err != nil || string(before) != string(after) {
				t.Fatal("old bridge was rewritten to new attempt", err)
			}
			next := filepath.Join(e.config.StateDir, "runs", run.ID, "attempt-2", "bridge.json")
			config, _, err = ReadBridgeConfig(next)
			if err != nil || config.ManagedGeneration != 2 {
				t.Fatal("continuation binding missing", err)
			}
			if strings.Contains(string(after), e.journal.HostSecret) {
				t.Fatal("host proof leaked to provider child")
			}
			e.config.Project = "p_00000000000000000000000000000002"
			run.Generation = 1
			if _, err = e.bridgeArgs(run); err == nil {
				t.Fatal("attempt bridge authority silently rewritten")
			}
		})
	}
}

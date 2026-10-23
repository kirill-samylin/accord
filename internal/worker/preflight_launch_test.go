package worker

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestBridgePrivatePreparationBoundaries(t *testing.T) {
	for _, step := range []string{"binding", "run_directory", "attempt_directory", "bridge_file", "claude_file"} {
		t.Run(step, func(t *testing.T) {
			e := testEngine(t)
			run := RunState{ID: "run_test", Provider: "codex", Workspace: "test", Generation: 1}
			runDir := filepath.Join(e.config.StateDir, "runs", run.ID)
			attempt := filepath.Join(runDir, "attempt-1")
			write := func(path string) {
				if err := os.WriteFile(path, []byte("synthetic-private-binding"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			switch step {
			case "binding":
				run.ID = "../unsafe"
			case "run_directory":
				write(filepath.Join(e.config.StateDir, "runs"))
			case "attempt_directory":
				if err := os.MkdirAll(runDir, 0700); err != nil {
					t.Fatal(err)
				}
				write(attempt)
			default:
				if err := os.MkdirAll(attempt, 0700); err != nil {
					t.Fatal(err)
				}
				if step == "claude_file" {
					run.Provider = "claude"
					write(filepath.Join(attempt, "mcp.json"))
				} else {
					write(filepath.Join(attempt, "bridge.json"))
				}
			}
			_, err := e.bridgeArgs(run)
			if err == nil || preparationDiagnostic(err) != "prepare_"+step || err.Error() != preparationError {
				t.Fatal("private bridge failure not classified safely")
			}
		})
	}
}

func TestPreparationLaunchRefusalPersistsOnlyPrivateDiagnostic(t *testing.T) {
	for _, step := range []string{"discovery_wait", "verify_binding", "deadline", "pins", "authorization"} {
		t.Run(step, func(t *testing.T) {
			e := testEngine(t)
			e.start = func(launchSpec) (process, error) {
				t.Error("native launch after failed preparation")
				return nil, errors.New("unexpected")
			}
			expected := "prepare_" + step
			if step == "discovery_wait" {
				e.discover = func(string, string) ([]mcpConnector, error) {
					return nil, preparationFailureAt("prepare_discovery_wait", ownershipSystemFailure("unix_reap_kill", syscall.EPERM))
				}
				expected += "__unix_reap_kill_errno_1"
			} else {
				e.verifyMCP = func(string, string, []string, string, []string) error {
					e.mu.Lock()
					defer e.mu.Unlock()
					switch step {
					case "verify_binding":
						return checkCodexMCP(nil, "/synthetic/self", nil)
					case "deadline":
						e.deadline = time.Now().Add(-time.Second)
					case "pins":
						e.journal.PathPins["workspace:test"] = pathPin{}
					case "authorization":
						e.journal.OwnerPrincipalID = "revoked"
					}
					return nil
				}
			}
			command, run := testDispatch("start")
			if err := e.Accept(command, run); err != nil {
				t.Fatal(err)
			}
			if err := e.Ack(*e.Pending()); err != nil {
				t.Fatal(err)
			}
			local := e.journal.Runs[run.ID]
			if local.State != "failed" || local.LocalDiagnostic != expected {
				t.Fatal("failed preparation lost private diagnostic")
			}
			data, err := os.ReadFile(filepath.Join(e.config.StateDir, "state.json"))
			if err != nil || !strings.Contains(string(data), expected) {
				t.Fatal("preparation diagnostic not persisted")
			}
			receipt := e.Pending()
			data, _ = json.Marshal(receipt)
			if receipt.Phase != "failed" || receipt.Error != preparationError || strings.Contains(string(data), expected) || strings.Contains(string(data), "local_diagnostic") {
				t.Fatal("private preparation details leaked into public report")
			}
		})
	}
}

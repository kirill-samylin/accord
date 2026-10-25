package worker

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

// Exercises the actual launch/preparation/config/capability/bridge chain with
// only an injected no-child starter. It never starts an OS process or model.
func TestOptedInLaunchUsesProvenProfileAndNeverLegacyFallback(t *testing.T) {
	for _, changed := range []bool{false, true} {
		name := "stable"
		if changed {
			name = "changed-second-proof"
		}
		t.Run(name, func(t *testing.T) {
			e, p, oldActive := gitPreparationFixture(t)
			oldActive.automation.cancel()
			delete(e.active, p.run.ID)
			r := e.journal.Runs[p.run.ID]
			r.Automation = nil
			r.SessionID = ""
			r.State = "accepted"
			a := e.config.Automation.Workspaces["test"]
			identity, err := directoryIdentity(e.config.Workspaces["test"].Path)
			if err != nil {
				t.Fatal(err)
			}
			preset := a.Checks["unit"]
			preset.TrustedOwnerConsent = presetConsentDigest(e.config.Workspaces["test"].Path, identity, "unit", a, preset)
			a.Checks["unit"] = preset
			e.config.Automation.Workspaces["test"] = a
			p.run.Automation.Checks[0].LocalDigest = preset.TrustedOwnerConsent
			e.automationTransport = func(_ context.Context, _ string, input, output any) error {
				in := input.(core.ExecutionCapabilityRegisterInput)
				*output.(*core.ExecutionCapability) = core.ExecutionCapability{ID: in.CapabilityID, RunID: in.RunID, Generation: in.Generation, WorkflowID: in.WorkflowID, StepID: in.StepID, WorkerEpoch: in.WorkerEpoch, ControlEpoch: in.ControlEpoch, Role: p.run.Profile.Role, ProfileID: p.run.Profile.ID, ProfileVersion: p.run.Profile.Version, State: "active", ExpiresAt: time.Now().Add(time.Second)}
				return nil
			}
			inspections := 0
			e.inspectPermissions = func(_ context.Context, s launchSpec) (permissionProof, error) {
				inspections++
				digest := strings.Repeat("a", 64)
				if changed && inspections == 2 {
					digest = strings.Repeat("b", 64)
				}
				return permissionProof{Digest: digest, ExecutableSHA256: s.inspection.ExecutableSHA256, GitIdentity: s.inspection.GitIdentity}, nil
			}
			starts := 0
			e.start = func(s launchSpec) (process, error) {
				starts++
				args := strings.Join(s.args, " ")
				if strings.Contains(args, "sandbox_mode") || !strings.Contains(args, "default_permissions=") || !strings.Contains(args, "approval_policy=\"never\"") {
					t.Error("native did not use proven narrow profile")
				}
				return nil, errors.New("synthetic no-child start failure")
			}
			e.mu.Lock()
			err = e.launchAutomationLocked(p)
			e.mu.Unlock()
			if err != nil || inspections != 2 {
				t.Fatal("did not complete both mocked proofs", err, inspections)
			}
			if changed {
				if starts != 0 || r.LocalDiagnostic != "native_git_permission_unproven" {
					t.Fatal("changed proof spawned or lost safe diagnostic")
				}
			} else if starts != 1 || r.Automation.PermissionProof == nil {
				t.Fatal("proven profile missing from actual launch")
			} else if budget := e.journal.AutomationBudgets[p.run.Automation.WorkflowID]; budget == nil || budget.Invocations != 0 || r.LocalDiagnostic != "native_start_failed" {
				t.Fatal("a native start failure consumed a local invocation or lost its cause", r.LocalDiagnostic)
			}
		})
	}
}

package store

import (
	"agent-board/internal/core"
	"reflect"
	"strings"
	"testing"
)

func TestAutomationAbortedExitZeroCheckDoesNotQualifyAnyRequiredPreset(t *testing.T) {
	for _, kind := range []string{"test", "lint", "build"} {
		t.Run(kind, func(t *testing.T) {
			clean, exit := true, 0
			w := &core.Workflow{ID: "w", PlanRevision: 1, Grant: &core.WorkflowGrant{Checks: []core.AutomationCheckPresetRef{{PresetID: "unit", LocalVersion: 1, LocalDigest: strings.Repeat("a", 64), Kind: "test", Required: true}, {PresetID: "extra", LocalVersion: 1, LocalDigest: strings.Repeat("b", 64), Kind: kind, Required: true}}}}
			st := &core.WorkflowStep{ID: "step", ManagedRunID: "run", ManagedGeneration: 1, TargetSHA: commitA}
			task := &core.Task{Key: "AB-1", Generation: 1}
			b := &board{Automation: newAutomationBoard()}
			b.Automation.Plans["w/1"] = &core.WorkflowPlanSnapshot{Tasks: []core.WorkflowPlanTask{{ClientKey: "one", RequiredCheckIDs: []string{"unit", "extra"}}}, TaskKeys: map[string]string{"one": task.Key}}
			for _, c := range w.Grant.Checks {
				b.Automation.Attestations[c.PresetID] = &core.WorkflowAttestation{ID: c.PresetID, StepID: st.ID, RunID: st.ManagedRunID, Generation: 1, TaskGeneration: 1, Kind: "check", Status: "completed", PresetID: c.PresetID, PresetVersion: c.LocalVersion, PresetDigest: c.LocalDigest, Observations: core.WorkflowObservations{ActualHead: commitA, TargetSHA: commitA, SourceClean: &clean, ManifestSHA256: strings.Repeat("c", 64), CheckExitCode: &exit, ChecksPassed: 7, ReportKind: "junit", ReportSHA256: strings.Repeat("d", 64)}}
			}
			if !workflowTestsGate(b, w, st, task) {
				t.Fatal("positive actual report fixture does not qualify")
			}
			observed := b.Automation.Attestations["extra"]
			observed.Observations.CheckAborted = true
			before := observed.Observations
			if workflowTestsGate(b, w, st, task) {
				t.Fatal("aborted check exit0/JUnit falsely qualified")
			}
			if !reflect.DeepEqual(before, observed.Observations) || *observed.Observations.CheckExitCode != 0 || observed.Observations.ChecksPassed != 7 {
				t.Fatal("gate fabricated exit/counts/source facts")
			}
		})
	}
}

func TestAutomationCheckAbortedActualExitZeroPersistsButBlocksPipeline(t *testing.T) {
	m := automationSetup(t)
	w := m.submit(t)
	id := w.Workflow.ID
	m.tick(t, id)
	a := m.attempt(t, id)
	p := m.plan(t, id, a)
	m.finish(t, a, core.Evidence{Summary: "plan", Artifact: []string{"workflow-plan:" + p.Plans[0].Hash}}, "")
	m.exit(t, a, commitA, "")
	m.tick(t, id)
	d := m.attempt(t, id)
	m.finish(t, d, core.Evidence{Commit: commitB, RunSummary: "commit"}, "")
	m.exit(t, d, commitB, "")
	m.tick(t, id)
	r := m.attempt(t, id)
	m.finish(t, r, core.Evidence{VerifiedCommit: commitB, Findings: "source reviewed"}, "approve")
	m.exit(t, r, commitB, commitB)
	m.tick(t, id)
	q := m.attempt(t, id)
	m.attest(t, q, "check_reserved", "check-one", core.WorkflowObservations{})
	m.attest(t, q, "check_started", "check-one", core.WorkflowObservations{})
	clean, exit := true, 0
	obs := core.WorkflowObservations{CheckAborted: true, ActualHead: commitB, TargetSHA: commitB, SourceClean: &clean, ManifestSHA256: strings.Repeat("b", 64), CheckExitCode: &exit, ChecksPassed: 7, ReportKind: "junit", ReportSHA256: strings.Repeat("c", 64)}
	finished := m.attest(t, q, "check_finished", "check-one", obs)
	if !reflect.DeepEqual(finished.Attestation.Observations, obs) {
		t.Fatal("attestation changed actual exit/report facts")
	}
	m.finish(t, q, core.Evidence{VerifiedCommit: commitB, RunSummary: "agent claimed pass after interrupted check"}, "passed")
	m.exit(t, q, commitB, commitB)
	w = m.tick(t, id)
	if w.Workflow.State != "blocked" || w.Workflow.ReasonCode != "test_evidence_missing" && w.Workflow.ReasonCode != "TEST_EVIDENCE_MISSING" {
		t.Fatal("aborted check falsely completed workflow", w.Workflow)
	}
	for _, record := range w.Attestations {
		if record.CheckRequestID == "check-one" && !reflect.DeepEqual(record.Observations, obs) {
			t.Fatal("durable observations no longer actual")
		}
	}
}

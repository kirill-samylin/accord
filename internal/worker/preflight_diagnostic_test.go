package worker

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

type inspectionFake struct {
	output  io.ReadCloser
	waitErr error
	stopErr error
	waits   int
	stops   int
}

func (p *inspectionFake) Output() io.ReadCloser { return p.output }
func (p *inspectionFake) Wait() error           { p.waits++; return p.waitErr }
func (p *inspectionFake) Stop() error           { p.stops++; return p.stopErr }
func (p *inspectionFake) Forced() bool          { return false }

type inspectionReadFailure struct{ err error }

func (p inspectionReadFailure) Read([]byte) (int, error) { return 0, p.err }

type timedInspectionFake struct {
	reader *io.PipeReader
	writer *io.PipeWriter
	done   chan struct{}
	once   sync.Once
}

func (p *timedInspectionFake) Output() io.ReadCloser { return p.reader }
func (p *timedInspectionFake) Forced() bool          { return false }
func (p *timedInspectionFake) Wait() error           { <-p.done; return errNativeExit }
func (p *timedInspectionFake) Stop() error {
	p.once.Do(func() { _ = p.writer.Close(); close(p.done) })
	return nil
}

func TestNativeInspectionKeepsItsFiveSecondTimeout(t *testing.T) {
	reader, writer := io.Pipe()
	child := &timedInspectionFake{reader: reader, writer: writer, done: make(chan struct{})}
	defer child.Stop()
	started := time.Now()
	_, err := readNativeFor("verify", "/synthetic/codex", "/synthetic/workspace", nil, func(launchSpec) (process, error) { return child, nil })
	if err == nil || preparationDiagnostic(err) != "prepare_verify_timeout" || time.Since(started) < 5*time.Second {
		t.Fatal("inspection timeout changed or was not classified")
	}
}

func TestNativeInspectionKeepsExactSizeBoundary(t *testing.T) {
	child := &inspectionFake{output: io.NopCloser(strings.NewReader(strings.Repeat("x", 1<<20)))}
	data, err := readNativeFor("discovery", "/synthetic/codex", "/synthetic/workspace", nil, func(launchSpec) (process, error) { return child, nil })
	if err != nil || len(data) != 1<<20 || child.waits != 1 || child.stops != 0 {
		t.Fatal("valid inspection size boundary changed")
	}
}

func TestNativeInspectionStagesRetainOwnershipWithoutExposingCause(t *testing.T) {
	secretCause := errors.New("synthetic-secret /private/host/token https://user:pass@example.invalid")
	ownership := ownershipSystemFailure("unix_reap_kill", syscall.EPERM)
	for _, context := range []string{"discovery", "verify"} {
		for _, tc := range []struct {
			name, step string
			output     io.Reader
			startErr   error
			waitErr    error
			stopErr    error
			stop       bool
			owner      bool
		}{
			{name: "start", step: "start", startErr: secretCause},
			{name: "read", step: "read", output: inspectionReadFailure{secretCause}, stop: true},
			{name: "limit", step: "limit", output: strings.NewReader(strings.Repeat("x", (1<<20)+1)), stop: true},
			{name: "wait", step: "wait", output: strings.NewReader("[]"), waitErr: secretCause},
			{name: "ownership wait", step: "wait", output: strings.NewReader("[]"), waitErr: ownership, owner: true},
			{name: "read and uncertain cleanup", step: "read", output: inspectionReadFailure{secretCause}, waitErr: ownership, stop: true, owner: true},
			{name: "limit and uncertain stop", step: "limit", output: strings.NewReader(strings.Repeat("x", (1<<20)+1)), stopErr: ownership, stop: true, owner: true},
		} {
			t.Run(context+"/"+tc.name, func(t *testing.T) {
				child := &inspectionFake{waitErr: tc.waitErr, stopErr: tc.stopErr}
				if tc.output != nil {
					child.output = io.NopCloser(tc.output)
				}
				_, err := readNativeFor(context, "/synthetic/codex", "/synthetic/workspace", nil, func(launchSpec) (process, error) { return child, tc.startErr })
				if err == nil || err.Error() != preparationError {
					t.Fatal("inspection error exposes cause or lost its generic message")
				}
				expected := "prepare_" + context + "_" + tc.step
				if tc.owner {
					expected += "__unix_reap_kill_errno_1"
				}
				if preparationDiagnostic(err) != expected || errors.Is(err, errOwnership) != tc.owner {
					t.Fatal("lost private stage or ownership uncertainty")
				}
				if tc.startErr == nil && child.waits != 1 || tc.stop != (child.stops == 1) {
					t.Fatal("inspection changed wait/cleanup behavior")
				}
			})
		}
	}
}

func TestPreparationDiagnosticEncodingIsClosedAndBounded(t *testing.T) {
	cause := errors.New("synthetic-secret stdout path token provider data")
	for _, unsafe := range []string{"synthetic-secret", "prepare_discovery_wait__anything", strings.Repeat("x", 1000)} {
		err := preparationFailureAt(unsafe, cause)
		if preparationDiagnostic(err) != "prepare_stage_invalid" || err.Error() != preparationError || !errors.Is(err, cause) {
			t.Fatal("untrusted stage or cause exposed")
		}
	}
	forged := &preparationFailure{stage: "prepare_discovery_wait", cause: &ownershipFailure{code: "synthetic-secret"}}
	if got := preparationDiagnostic(forged); got != "prepare_discovery_wait__ownership_diagnostic_invalid" || len(got) > 128 {
		t.Fatal("unvalidated underlying ownership diagnostic copied")
	}
	if preparationDiagnostic(cause) != "prepare_unspecified" {
		t.Fatal("unclassified error contents used as diagnostic")
	}
	state := RunState{LocalDiagnostic: preparationDiagnostic(preparationFailureAt("prepare_verify_wait", errOwnership))}
	data, _ := json.Marshal(state)
	if !strings.Contains(string(data), "prepare_verify_wait__ownership_unspecified") {
		t.Fatal("private state lost diagnostic")
	}
	receipt := Receipt{Error: preparationError}
	data, _ = json.Marshal(receipt)
	if strings.Contains(string(data), "prepare_verify") || strings.Contains(string(data), "local_diagnostic") {
		t.Fatal("diagnostic leaked into report")
	}
}

func TestInspectionJSONMetadataTransportAndBindingAreDistinct(t *testing.T) {
	for _, context := range []string{"discovery", "verify"} {
		for _, tc := range []struct{ data, step string }{
			{`{"private":"synthetic-secret"}`, "json"},
			{`[{"name":"private-name","transport":{"type":"stdio"}}]`, "metadata"},
			{`[{"name":"private-name","enabled":true,"transport":{"type":"unknown"}}]`, "transport"},
		} {
			_, err := parseMCPFor(context, []byte(tc.data))
			if err == nil || preparationDiagnostic(err) != "prepare_"+context+"_"+tc.step || err.Error() != preparationError {
				t.Fatal("inspection parser stage missing or local data exposed")
			}
		}
	}
	err := checkCodexMCP([]mcpConnector{{Name: "unexpected", Enabled: true}}, "/synthetic/self", nil)
	if err == nil || preparationDiagnostic(err) != "prepare_verify_binding" || err.Error() != preparationError {
		t.Fatal("post-override binding mismatch not classified")
	}
}

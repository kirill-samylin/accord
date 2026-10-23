package automationipc

import (
	"bytes"
	"context"
	"encoding/binary"
	"strings"
	"testing"
)

func TestPrivateFramesRejectOversizeDuplicateAndUnknownFields(t *testing.T) {
	for _, raw := range []string{`{"v":1,"v":1}`, `{"arbitrary_command":"danger"}`, `{} {}`} {
		var wire bytes.Buffer
		var n [4]byte
		binary.BigEndian.PutUint32(n[:], uint32(len(raw)))
		wire.Write(n[:])
		wire.WriteString(raw)
		if readFrame(&wire, &Request{}) == nil {
			t.Fatal("unsafe frame accepted")
		}
	}
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], MaxFrame+1)
	if readFrame(bytes.NewReader(size[:]), &Request{}) == nil {
		t.Fatal("oversized frame accepted")
	}
}
func TestNonceAndEveryAttemptCoordinateAreFixed(t *testing.T) {
	c := Config{Nonce: strings.Repeat("a", 64), Binding: Binding{WorkflowID: "wf", StepID: "step", RunID: "run", ControlEpoch: 3, Generation: 2}}
	r := Request{Version: 1, Nonce: c.Nonce, RequestID: "id", Operation: "start", WorkflowID: "wf", StepID: "step", RunID: "run", ControlEpoch: 3, Generation: 2, SessionID: "session", TaskKey: "task", TaskGeneration: 5, PresetID: "unit"}
	if !valid(c, r) {
		t.Fatal("valid binding rejected")
	}
	for _, mutate := range []func(*Request){func(r *Request) { r.Nonce = strings.Repeat("b", 64) }, func(r *Request) { r.ControlEpoch++ }, func(r *Request) { r.Generation++ }, func(r *Request) { r.WorkflowID = "other" }, func(r *Request) { r.StepID = "other" }, func(r *Request) { r.RunID = "other" }, func(r *Request) { r.Operation = "execute" }} {
		copy := r
		mutate(&copy)
		if valid(c, copy) {
			t.Fatal("mutable local authority accepted")
		}
	}
	if _, err := Call(context.Background(), c, Request{Operation: "execute"}); err == nil {
		t.Fatal("unsupported RPC accepted")
	}
}

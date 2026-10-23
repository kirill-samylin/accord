//go:build darwin || linux

package automationipc

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLongAttemptUsesShortPrivateOwnedRuntimeAndRemovesSocket(t *testing.T) {
	short, e := os.MkdirTemp("/tmp", "accord-ipc-test-")
	if e != nil {
		t.Fatal(e)
	}
	short, e = filepath.EvalSymlinks(short)
	if e != nil {
		t.Fatal(e)
	}
	defer os.RemoveAll(short)
	t.Setenv("HOME", short)
	long := filepath.Join(t.TempDir(), strings.Repeat("attempt-", 20))
	if e = os.MkdirAll(long, 0700); e != nil {
		t.Fatal(e)
	}
	binding := Binding{WorkflowID: "wf", StepID: "step", RunID: "run", ControlEpoch: 1, Generation: 1}
	calls := 0
	s, e := Listen(long, binding, func(context.Context, Request) (Response, error) { calls++; return Response{Status: "unknown"}, nil })
	if e != nil {
		t.Fatal(e)
	}
	config := s.Config()
	if !strings.HasPrefix(config.Endpoint, filepath.Join(short, ".accord-ipc")) {
		t.Fatal("long path did not use the private short owner runtime")
	}
	info, e := os.Lstat(config.Endpoint)
	if e != nil || info.Mode().Perm() != 0600 {
		t.Fatal("socket is not private", e)
	}
	result, e := Call(context.Background(), config, Request{Operation: "status", RequestID: "same", PresetID: "unit", SessionID: "s", TaskKey: "task", TaskGeneration: 2})
	if e != nil || result.Status != "unknown" || calls != 1 {
		t.Fatal(result, e)
	}
	if e = s.Close(); e != nil {
		t.Fatal(e)
	}
	if _, e = os.Stat(config.Endpoint); !os.IsNotExist(e) {
		t.Fatal("attempt socket survived close")
	}
}

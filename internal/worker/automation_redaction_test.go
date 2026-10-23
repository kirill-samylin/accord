package worker

import (
	"agent-board/internal/automationipc"
	"strings"
	"testing"
)

func TestManagedOutputRedactsExecutionBearerAndPrivateIPCNonce(t *testing.T) {
	token := "ac_exec1_" + strings.Repeat("a", 64)
	if strings.Contains(cleanOutput("diagnostic="+token+"suffix"), token) {
		t.Fatal("raw scoped execution bearer escaped")
	}
	colored := token[:30] + "\x1b[31m" + token[30:] + "\x1b[0m"
	if strings.Contains(cleanOutput(colored), token) {
		t.Fatal("ANSI normalization exposed an execution credential")
	}
	e := testEngine(t)
	nonce := strings.Repeat("b", 64)
	e.journal.Runs["synthetic"] = &RunState{ID: "synthetic", State: "failed", Automation: &automationAttempt{IPC: &automationipc.Config{Nonce: nonce}}}
	cleaned := cleanOutput("endpoint nonce "+nonce+"\ncredential "+token, e.outputSecrets()...)
	if strings.Contains(cleaned, nonce) || strings.Contains(cleaned, token) {
		t.Fatal("per-attempt bridge secret escaped managed output")
	}
}

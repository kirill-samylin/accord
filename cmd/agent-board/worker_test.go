package main

import (
	"io"
	"strings"
	"testing"
)

func TestWorkerRecoverRequiresExplicitRun(t *testing.T) {
	for _, args := range [][]string{{"recover", "--config", "/nonexistent/worker.json"}, {"run", "--config", "/nonexistent/worker.json", "--run", "run_test"}} {
		err := workerCommand(args, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "--run RUN_ID") {
			t.Fatal("recover run selection is not explicit", args, err)
		}
	}
}

func TestWorkerUnverifiedIsOnlyForRecover(t *testing.T) {
	err := workerCommand([]string{"run", "--config", "/nonexistent/worker.json", "--unverified"}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "--unverified") {
		t.Fatal("--unverified was accepted outside worker recover", err)
	}
}

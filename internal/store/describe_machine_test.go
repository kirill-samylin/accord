package store

import (
	"testing"

	"agent-board/internal/core"
)

func TestDescribeDefaultsToTokenMachineAndRejectsAnotherOne(t *testing.T) {
	f := newFixture(t)
	v, err := f.s.Describe(testContext, f.p, core.OpenSession{RequestID: f.id(), Role: "developer", RunID: "omitted-machine"})
	if err != nil || v.SessionID == "" {
		t.Fatal("describe without machine must use the token machine", err)
	}
	if _, err = f.s.Describe(testContext, f.p, core.OpenSession{RequestID: f.id(), Role: "developer", RunID: "wrong-machine", Machine: "Someones-MacBook.local"}); err == nil {
		t.Fatal("a machine that differs from the token was accepted")
	}
}

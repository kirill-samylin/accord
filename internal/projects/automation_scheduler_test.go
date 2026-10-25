package projects

import (
	"context"
	"os"
	"testing"
	"time"
)

// One broken project must not stop the scheduler (and with it the server):
// its failure is backed off and the other projects are still scanned.
func TestAutomationSchedulerIsolatesBrokenProjectAndBacksItOff(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	broken := f.create(t, f.who, "broken")
	healthy := f.create(t, f.who, "healthy")
	db, _, err := f.r.projectPaths(broken.ID)
	if err != nil {
		t.Fatal(err)
	}
	f.r.mu.Lock()
	if v := f.r.runtimes[broken.ID]; v != nil {
		v.store.Close()
		delete(f.r.runtimes, broken.ID)
	}
	f.r.mu.Unlock()
	if err = os.Remove(db); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = f.r.automationCycle(ctx, 0, map[string]int{}); err != nil {
			t.Fatal("per-project failure stopped the scheduler", err)
		}
	}
	if !f.r.backoff.skip(broken.ID, time.Now()) {
		t.Fatal("broken project was not backed off")
	}
	if f.r.backoff.skip(healthy.ID, time.Now()) {
		t.Fatal("healthy project was penalised")
	}
	f.r.backoff.mu.Lock()
	failures := f.r.backoff.projects[broken.ID].failures
	f.r.backoff.mu.Unlock()
	if failures != 1 {
		t.Fatal("backed-off project was retried before its delay", failures)
	}
	if _, err = f.r.runtime(healthy.ID); err != nil {
		t.Fatal("healthy project runtime", err)
	}
}

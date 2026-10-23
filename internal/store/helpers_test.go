package store

import (
	"agent-board/internal/core"
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var testContext = context.Background()

const commitA = "1111111111111111111111111111111111111111"
const commitB = "2222222222222222222222222222222222222222"

type testClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *testClock) read() time.Time         { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *testClock) advance(d time.Duration) { c.mu.Lock(); defer c.mu.Unlock(); c.now = c.now.Add(d) }

type fixture struct {
	s     *Store
	clock *testClock
	p     core.Principal
	coord core.Actor
	epic  core.Task
	path  string
	seq   int
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	clock := &testClock{now: time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)}
	path := filepath.Join(t.TempDir(), "board.db")
	s, err := Open(path, core.Config{SessionTTL: time.Hour, LeaseTTL: time.Minute, Project: "test"}, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	token, err := s.CreateToken(testContext, core.TokenInput{Owner: "alice", Machine: "pc-a", TTL: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Authenticate(testContext, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{s: s, clock: clock, p: p, path: path}
	f.coord = f.actor(t, "coordinator")
	f.epic, err = s.Create(testContext, f.coord, core.CreateInput{RequestID: "epic", Type: "epic", Title: "Pilot"})
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func (f *fixture) id() string { f.seq++; return time.Unix(int64(f.seq), 0).Format(time.RFC3339) }
func (f *fixture) actor(t *testing.T, role string) core.Actor {
	t.Helper()
	v, err := f.s.Describe(testContext, f.p, core.OpenSession{RequestID: f.id(), Role: role, RunID: f.id(), Machine: f.p.Machine})
	if err != nil {
		t.Fatal(err)
	}
	return core.Actor{Principal: f.p, SessionID: v.SessionID}
}
func (f *fixture) task(t *testing.T, role string) core.Task {
	t.Helper()
	v, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: role, Description: "implement", Role: role})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func (f *fixture) claim(t *testing.T, a core.Actor, key string) core.Claim {
	t.Helper()
	v, err := f.s.Next(testContext, a, core.NextInput{RequestID: f.id(), TaskKey: key})
	if err != nil {
		t.Fatal(err)
	}
	if v.Task == nil {
		t.Fatal("expected claim")
	}
	return v
}
func (f *fixture) finish(t *testing.T, a core.Actor, claim core.Claim, e core.Evidence, outcome string) core.Task {
	t.Helper()
	v, err := f.s.Transition(testContext, a, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: claim.Task.Key, Generation: claim.Lease.Generation}, State: "done", Outcome: outcome, Evidence: &e})
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func requireCode(t *testing.T, err error, code string) {
	t.Helper()
	var domain *core.Error
	if !errors.As(err, &domain) || domain.Code != code {
		t.Fatalf("want %s; got %v", code, err)
	}
}

func dropAutomationFixture(t *testing.T, s *Store) {
	t.Helper()
	for _, table := range append(append([]string{}, automationTables...), "automation_metadata") {
		if _, err := s.db.Exec("DROP TABLE " + table); err != nil {
			t.Fatal(err)
		}
	}
}

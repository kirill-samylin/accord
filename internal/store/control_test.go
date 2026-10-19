package store

import (
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestControlOverviewSearchAndFullSessionProjection(t *testing.T) {
	f := newFixture(t)
	var target core.Task
	for i := 0; i < 105; i++ {
		target = f.task(t, "analyst")
	}
	a := f.actor(t, "analyst")
	claimed := f.claim(t, a, target.Key)

	// The fixture writes recorded content directly to isolate reader behavior from ownership mutations.
	err := f.s.adminTransaction(testContext, func(b *board, _ time.Time) error {
		b.Tasks[target.Key].Title = "СМЕШАННЫЙ <script>Название</script>"
		b.Tasks[target.Key].Description = "Уникальное ОПИСАНИЕ"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := f.s.ControlOverview(testContext, f.coord)
	if err != nil {
		t.Fatal(err)
	}
	if before.TasksTotal != 105 || before.TasksReady != 104 || before.TasksInProgress != 1 || before.LeasedTasks != 1 || before.OpenEpics != 1 || before.ActiveAgentSessions != 2 {
		t.Fatalf("overview %+v", before)
	}
	for _, query := range []string{"смешанный", "описание", strings.ToLower(target.Key), "<script>"} {
		page, err := f.s.ControlTasks(testContext, f.coord, core.ControlTasksInput{Query: query})
		if err != nil || len(page.Tasks) != 1 || page.Tasks[0].Key != target.Key {
			t.Fatalf("search %q: %+v %v", query, page, err)
		}
	}
	agents, err := f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{SessionID: a.SessionID, IncludeClosed: true})
	if err != nil || len(agents.Sessions) != 1 || agents.Sessions[0].Task == nil || agents.Sessions[0].Task.Key != target.Key || agents.Sessions[0].LastEventAt == nil {
		t.Fatalf("full lease join %+v %v", agents, err)
	}
	for i := 0; i < 3; i++ {
		if _, err := f.s.ControlOverview(testContext, f.coord); err != nil {
			t.Fatal(err)
		}
		if _, err := f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{}); err != nil {
			t.Fatal(err)
		}
		if _, err := f.s.RecentEvents(testContext, f.coord, core.RecentEventsInput{}); err != nil {
			t.Fatal(err)
		}
	}
	current, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: target.Key})
	if err != nil {
		t.Fatal(err)
	}
	if !current.Lease.ExpiresAt.Equal(claimed.Lease.ExpiresAt) || current.Generation != claimed.Task.Generation {
		t.Fatal("read renewed lease or generation")
	}
	after, err := f.s.ControlOverview(testContext, f.coord)
	if err != nil {
		t.Fatal(err)
	}
	if after.LatestEventCursor != before.LatestEventCursor {
		t.Fatal("read emitted events")
	}
	first, err := f.s.ControlTasks(testContext, f.coord, core.ControlTasksInput{ListInput: core.ListInput{Limit: 2}, Query: "analyst"})
	if err != nil || first.NextCursor == "" {
		t.Fatalf("search pass %v", err)
	}
	_, err = f.s.ControlTasks(testContext, f.coord, core.ControlTasksInput{ListInput: core.ListInput{Limit: 2, Cursor: first.NextCursor}, Query: "описание"})
	requireCode(t, err, "VALIDATION")
	_, err = f.s.ControlTasks(testContext, f.coord, core.ControlTasksInput{Query: strings.Repeat("я", 201)})
	requireCode(t, err, "VALIDATION")
}

func TestControlSessionExactPagingKindAndCredentialValidity(t *testing.T) {
	f := newFixture(t)
	for i := 0; i < 101; i++ {
		f.actor(t, "analyst")
	}
	human := f.actor(t, "human")
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "other", Machine: "other", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.s.Authenticate(testContext, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	view, err := f.s.Describe(testContext, p, core.OpenSession{RequestID: "other", RunID: "other", Role: "analyst", Machine: p.Machine})
	if err != nil {
		t.Fatal(err)
	}
	other := core.Actor{Principal: p, SessionID: view.SessionID}
	task := f.task(t, "analyst")
	f.claim(t, other, task.Key)
	exact, err := f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{SessionID: other.SessionID, Limit: 1})
	if err != nil || len(exact.Sessions) != 1 || exact.Sessions[0].Task == nil {
		t.Fatalf("exact beyond first100: %+v %v", exact, err)
	}
	first, err := f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{Kind: "agent", Limit: 100})
	if err != nil || len(first.Sessions) != 100 || !first.HasMore {
		t.Fatalf("first %+v %v", first, err)
	}
	second, err := f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{Kind: "agent", Limit: 100, Cursor: first.NextCursor})
	if err != nil || len(second.Sessions) != 3 {
		t.Fatalf("second %+v %v", second, err)
	}
	_, err = f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{Kind: "human", Cursor: first.NextCursor})
	requireCode(t, err, "VALIDATION")
	humans, err := f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{Kind: "human"})
	if err != nil || len(humans.Sessions) != 1 || humans.Sessions[0].SessionID != human.SessionID {
		t.Fatalf("human kind %+v %v", humans, err)
	}
	if err = f.s.RevokeToken(testContext, token.ID); err != nil {
		t.Fatal(err)
	}
	invalid, err := f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{SessionID: other.SessionID, IncludeClosed: true})
	if err != nil || len(invalid.Sessions) != 1 || invalid.Sessions[0].Status != "credential_invalid" || invalid.Sessions[0].Task != nil {
		t.Fatalf("revoked %+v %v", invalid, err)
	}
	overview, err := f.s.ControlOverview(testContext, f.coord)
	if err != nil || overview.ActiveAgentSessions != 102 || overview.ActiveHumanSessions != 1 || overview.LeasedTasks != 0 || overview.TasksInProgress != 1 {
		t.Fatalf("revoked counts %+v %v", overview, err)
	}
	native, err := f.s.Sessions(testContext, f.coord, core.ListInput{Owner: "other", State: "active"})
	if err != nil || len(native.Sessions) != 1 {
		t.Fatal("native TTL semantics changed", err)
	}
	_, err = f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{Kind: "human", Role: "analyst"})
	requireCode(t, err, "VALIDATION")
	_, err = f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{SessionID: "bad"})
	requireCode(t, err, "VALIDATION")
}

func TestControlTokenExpiryAndGenerationTruth(t *testing.T) {
	f := newFixture(t)
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "short", Machine: "short", TTL: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.s.Authenticate(testContext, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	v, err := f.s.Describe(testContext, p, core.OpenSession{RequestID: "short", RunID: "short", Role: "analyst", Machine: p.Machine})
	if err != nil {
		t.Fatal(err)
	}
	f.clock.advance(2 * time.Second)
	page, err := f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{SessionID: v.SessionID, IncludeClosed: true})
	if err != nil || page.Sessions[0].Status != "credential_invalid" {
		t.Fatalf("expired token %+v %v", page, err)
	}
	a := f.actor(t, "analyst")
	task := f.task(t, "analyst")
	f.claim(t, a, task.Key)
	err = f.s.adminTransaction(testContext, func(b *board, _ time.Time) error { b.Tasks[task.Key].Generation++; return nil })
	if err != nil {
		t.Fatal(err)
	}
	page, err = f.s.ControlSessions(testContext, f.coord, core.ControlSessionsInput{SessionID: a.SessionID})
	if err != nil || page.Sessions[0].Task != nil {
		t.Fatal("stale generation shown", err)
	}
	overview, err := f.s.ControlOverview(testContext, f.coord)
	if err != nil || overview.LeasedTasks != 0 {
		t.Fatal("stale generation counted", err)
	}
}

func TestControlRecentEventsFiltersAndNewInsertion(t *testing.T) {
	f := newFixture(t)
	first := f.task(t, "analyst")
	for i := 0; i < 105; i++ {
		f.task(t, "analyst")
	}
	filtered, err := f.s.RecentEvents(testContext, f.coord, core.RecentEventsInput{TaskKey: first.Key, Type: "task_created", SessionID: f.coord.SessionID, Limit: 1})
	if err != nil || len(filtered.Events) != 1 || filtered.Events[0].TaskKey != first.Key {
		t.Fatalf("filter before limit %+v %v", filtered, err)
	}
	page, err := f.s.RecentEvents(testContext, f.coord, core.RecentEventsInput{Limit: 10})
	if err != nil || len(page.Events) != 10 || !page.HasMore {
		t.Fatalf("page %+v %v", page, err)
	}
	newTask := f.task(t, "analyst")
	next, err := f.s.RecentEvents(testContext, f.coord, core.RecentEventsInput{BeforeCursor: page.NextCursor, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int64]bool{}
	for _, e := range append(page.Events, next.Events...) {
		if seen[e.Cursor] || e.TaskKey == newTask.Key {
			t.Fatal("shifted pagination")
		}
		seen[e.Cursor] = true
	}
	for i, e := range next.Events {
		if e.Cursor >= page.NextCursor || i > 0 && e.Cursor >= next.Events[i-1].Cursor {
			t.Fatal("not strict DESC")
		}
	}
	for _, in := range []core.RecentEventsInput{{BeforeCursor: -1}, {Limit: 101}, {Type: "imaginary"}, {SessionID: "bad"}, {TaskKey: "T-0"}} {
		_, err := f.s.RecentEvents(testContext, f.coord, in)
		requireCode(t, err, "VALIDATION")
	}
	// Expiry caused by this read is included in the DESC result.
	a := f.actor(t, "analyst")
	task := f.task(t, "analyst")
	f.claim(t, a, task.Key)
	f.clock.advance(time.Minute)
	expired, err := f.s.RecentEvents(testContext, f.coord, core.RecentEventsInput{Type: "lease_expired", TaskKey: task.Key})
	if err != nil || len(expired.Events) != 1 {
		t.Fatalf("materialized expiry %+v %v", expired, err)
	}
}

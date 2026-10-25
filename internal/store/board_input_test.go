package store

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

func (f *fixture) createTask(in core.CreateInput) (core.Task, error) {
	in.RequestID = f.id()
	if in.Type == "" {
		in.Type = "task"
	}
	if in.Type == "task" {
		in.EpicKey, in.Role = f.epic.Key, "developer"
		if in.Description == "" {
			in.Description = "implement"
		}
	}
	if in.Title == "" {
		in.Title = "bounded input"
	}
	return f.s.Create(testContext, f.coord, in)
}

func TestCreateAreasAreBounded(t *testing.T) {
	f := newFixture(t)
	many := make([]string, 33)
	for i := range many {
		many[i] = "area/" + strings.Repeat("a", i+1)
	}
	for name, areas := range map[string][]string{
		"too many": many, "empty": {""}, "blank": {"  "}, "long": {strings.Repeat("я", 129)},
		"NUL": {"web\x00ui"}, "escape": {"\x1b[31mred"}, "traversal": {"../.."}, "dot segment": {"web/./ui"},
		"duplicate": {"web", "web"}, "invalid UTF-8": {"web\xff"},
	} {
		_, err := f.createTask(core.CreateInput{Areas: areas})
		if err == nil {
			t.Fatalf("%s areas accepted", name)
		}
		requireCode(t, err, "VALIDATION")
	}
	task, err := f.createTask(core.CreateInput{Areas: append(many[:31], strings.Repeat("я", 128))})
	if err != nil || len(task.Areas) != 32 {
		t.Fatalf("32 bounded areas rejected: %v", err)
	}
}

func TestOwnerOnlyForEpicsAndValidated(t *testing.T) {
	f := newFixture(t)
	_, err := f.createTask(core.CreateInput{Owner: "bob"})
	requireCode(t, err, "VALIDATION")
	if !strings.Contains(err.Error(), "only to epics") {
		t.Fatalf("unclear task owner message: %v", err)
	}
	for _, owner := range []string{strings.Repeat("o", 129), "bob\x1b[2J", "bob\x00", " ", "bad\xff"} {
		_, err = f.createTask(core.CreateInput{Type: "epic", Owner: owner})
		requireCode(t, err, "VALIDATION")
	}
	epic, err := f.createTask(core.CreateInput{Type: "epic", Owner: strings.Repeat("я", 128)})
	if err != nil || epic.Owner != strings.Repeat("я", 128) {
		t.Fatalf("valid epic owner: %v", err)
	}
}

func TestControlCharactersAndInvalidUTF8Rejected(t *testing.T) {
	f := newFixture(t)
	for _, title := range []string{"a\x00b", "a\nb", "\x1b[31mred", "bad\xff", "c1\u009b"} {
		_, err := f.createTask(core.CreateInput{Title: title})
		requireCode(t, err, "VALIDATION")
	}
	for _, description := range []string{"a\x00b", "\x1b]0;title\x07", "bad\xff", "lone\rreturn"} {
		_, err := f.createTask(core.CreateInput{Description: description})
		requireCode(t, err, "VALIDATION")
	}
	task, err := f.createTask(core.CreateInput{Description: "line 1\nline\t2\r\nline 3"})
	if err != nil {
		t.Fatalf("multi-line description rejected: %v", err)
	}
	bad := "x\x00"
	_, err = f.s.Update(testContext, f.coord, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key}, Description: &bad})
	requireCode(t, err, "VALIDATION")
	_, err = f.s.Update(testContext, f.coord, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key}, Title: &bad})
	requireCode(t, err, "VALIDATION")
	dev := f.actor(t, "developer")
	claim := f.claim(t, dev, task.Key)
	_, err = f.s.Comment(testContext, dev, core.CommentInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, Text: "done\x1b[2J"})
	requireCode(t, err, "VALIDATION")
	if _, err = f.s.Comment(testContext, dev, core.CommentInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, Text: "done\n\tok"}); err != nil {
		t.Fatal(err)
	}
	for _, request := range []string{"req\x00", "req\n", "bad\xff"} {
		_, err = f.s.Create(testContext, f.coord, core.CreateInput{RequestID: request, Type: "epic", Title: "x"})
		requireCode(t, err, "VALIDATION")
	}
	_, err = f.s.Describe(testContext, f.p, core.OpenSession{RequestID: f.id(), Role: "analyst", RunID: "run\x1b[0m", Machine: f.p.Machine})
	requireCode(t, err, "VALIDATION")
}

func TestRequestIDLimitCountsCharacters(t *testing.T) {
	f := newFixture(t)
	if _, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: strings.Repeat("я", 128), Type: "epic", Title: "runes"}); err != nil {
		t.Fatalf("128 characters (256 bytes) rejected: %v", err)
	}
	_, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: strings.Repeat("я", 129), Type: "epic", Title: "runes"})
	requireCode(t, err, "VALIDATION")
}

func TestSelfLinkRejectedForEveryType(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	for _, kind := range []string{"related", "blocks", "contract_of", "spawned_by", "duplicate_of"} {
		_, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: task.Key, To: task.Key, Type: kind})
		requireCode(t, err, "VALIDATION")
	}
	_, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: f.epic.Key, To: f.epic.Key, Type: "related"})
	requireCode(t, err, "VALIDATION")
	if current := f.get(t, task.Key); len(current.Links) != 0 || current.State != "ready" {
		t.Fatal("self link changed the task")
	}
}

func TestEventsNextCursorNeverExceedsHighWater(t *testing.T) {
	f := newFixture(t)
	page, err := f.s.Events(testContext, f.coord, core.EventsInput{SinceCursor: 1 << 40, Limit: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Events) != 0 || page.HighWater == 0 || page.NextCursor != page.HighWater || page.HasMore {
		t.Fatalf("cursor beyond high_water: next=%d high=%d", page.NextCursor, page.HighWater)
	}
	again, err := f.s.Events(testContext, f.coord, core.EventsInput{SinceCursor: page.HighWater})
	if err != nil || again.NextCursor != page.HighWater {
		t.Fatalf("cursor at high_water: %+v %v", again, err)
	}
}

func TestEmptyOrUnchangedUpdateIsNoOp(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	dev := f.actor(t, "developer")
	claim := f.claim(t, dev, task.Key)
	before := f.get(t, task.Key)
	page, err := f.s.Events(testContext, f.coord, core.EventsInput{})
	if err != nil {
		t.Fatal(err)
	}
	f.clock.advance(10 * time.Second)
	same := before.Title
	for _, in := range []core.UpdateInput{{}, {Title: &same, Description: &before.Description}} {
		in.LeaseInput = core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}
		got, err := f.s.Update(testContext, dev, in)
		if err != nil || got.Version != before.Version {
			t.Fatalf("no-op update changed version: %v %d->%d", err, before.Version, got.Version)
		}
	}
	after, err := f.s.Events(testContext, f.coord, core.EventsInput{})
	if err != nil || after.HighWater != page.HighWater {
		t.Fatal("no-op update emitted an event")
	}
	if current := f.get(t, task.Key); !current.Lease.ExpiresAt.Equal(before.Lease.ExpiresAt) || current.Version != before.Version {
		t.Fatal("no-op update renewed the lease or changed the task")
	}
	title := "Changed"
	got, err := f.s.Update(testContext, dev, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, Title: &title})
	if err != nil || got.Version != before.Version+1 {
		t.Fatalf("real update: %v", err)
	}
}

func TestForeignLeaseIsHeldNotLost(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	holder := f.actor(t, "developer")
	stranger := f.actor(t, "developer")
	claim := f.claim(t, holder, task.Key)
	lease := core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}
	_, err := f.s.Comment(testContext, stranger, core.CommentInput{LeaseInput: lease, Text: "not mine"})
	requireCode(t, err, "LEASE_HELD")
	title := "not mine"
	lease.RequestID = f.id()
	_, err = f.s.Update(testContext, stranger, core.UpdateInput{LeaseInput: lease, Title: &title})
	requireCode(t, err, "LEASE_HELD")
	// The real holder with a stale generation still learns it lost the lease.
	lease.RequestID, lease.Generation = f.id(), claim.Lease.Generation+1
	_, err = f.s.Comment(testContext, holder, core.CommentInput{LeaseInput: lease, Text: "stale"})
	requireCode(t, err, "LEASE_LOST")
}

func TestCommitSHAIsNormalisedToLowercase(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	dev := f.actor(t, "developer")
	upper := strings.ToUpper("abcdef" + commitA[6:])
	done := f.finish(t, dev, f.claim(t, dev, task.Key), core.Evidence{Commit: upper, RunSummary: "go test: PASS"}, "")
	if done.Evidence.Commit != strings.ToLower(upper) || done.Attempts[0].Commit != strings.ToLower(upper) {
		t.Fatalf("developer commit not normalised: %s", done.Evidence.Commit)
	}
	review, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "review", Description: "review", Role: "reviewer", TargetKey: task.Key, TargetCommit: upper})
	if err != nil || review.Review.TargetCommit != strings.ToLower(upper) {
		t.Fatalf("target commit not normalised: %v", err)
	}
	reviewer := f.actor(t, "reviewer")
	mixed := "AbCdEf" + commitA[6:]
	approved := f.finish(t, reviewer, f.claim(t, reviewer, review.Key), core.Evidence{VerifiedCommit: mixed, Findings: "ok"}, "approve")
	if approved.Evidence.VerifiedCommit != strings.ToLower(upper) {
		t.Fatalf("verified commit not normalised: %s", approved.Evidence.VerifiedCommit)
	}
}

// T022-R2-10c: a final total is bounded by what it adds to the reported
// deltas, so an attempt whose deltas already exceed 10^12 can still finish.
func TestFinalTokensBoundOnlyTheNewlyReportedAmount(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	dev := f.actor(t, "developer")
	claim := f.claim(t, dev, task.Key)
	lease := core.LeaseInput{TaskKey: task.Key, Generation: claim.Lease.Generation}
	for i := 0; i < 2; i++ {
		delta := maxAttemptTokens
		lease.RequestID = f.id()
		if _, err := f.s.Comment(testContext, dev, core.CommentInput{LeaseInput: lease, Text: "progress", TokensSpentDelta: &delta}); err != nil {
			t.Fatal(err)
		}
	}
	evidence := core.Evidence{Commit: commitA, RunSummary: "go test: PASS"}
	total := 3*maxAttemptTokens + 1
	lease.RequestID = f.id()
	_, err := f.s.Transition(testContext, dev, core.TransitionInput{LeaseInput: lease, State: "done", Evidence: &evidence, TokensSpent: &total})
	requireCode(t, err, "VALIDATION")
	total = 2 * maxAttemptTokens
	lease.RequestID = f.id()
	if _, err = f.s.Transition(testContext, dev, core.TransitionInput{LeaseInput: lease, State: "done", Evidence: &evidence, TokensSpent: &total}); err != nil {
		t.Fatal("final total equal to legacy-size deltas rejected", err)
	}
}

func TestStatsSaturateLegacyHugeTokenRows(t *testing.T) {
	f := newFixture(t)
	huge := int64(math.MaxInt64)
	for i, cursor := range []int64{1_000_001, 1_000_002} {
		raw, err := json.Marshal(core.Event{Cursor: cursor, Type: "released", TaskKey: "T-legacy", At: f.clock.read(), Data: &core.EventData{AttemptEnded: true, TokensSpent: &huge, Role: "developer"}})
		if err != nil {
			t.Fatal(err)
		}
		if _, err = f.s.db.Exec(`INSERT INTO events(cursor,data) VALUES(?,?)`, cursor, string(raw)); err != nil {
			t.Fatal(i, err)
		}
	}
	now := f.clock.read()
	stats, err := f.s.Stats(testContext, f.coord, core.StatsInput{From: now.Add(-time.Hour), To: now.Add(time.Hour), GroupBy: "role"})
	if err != nil {
		t.Fatalf("legacy huge counters broke statistics: %v", err)
	}
	if !stats.IncompleteData || stats.Totals.TokensSpent != math.MaxInt64 || stats.Totals.Attempts != 2 {
		t.Fatalf("stats did not saturate: %+v", stats.Totals)
	}
}

func TestFinalTokensAreBounded(t *testing.T) {
	f := newFixture(t)
	task := f.task(t, "developer")
	dev := f.actor(t, "developer")
	claim := f.claim(t, dev, task.Key)
	total := maxAttemptTokens + 1
	evidence := core.Evidence{Commit: commitA, RunSummary: "go test: PASS"}
	_, err := f.s.Transition(testContext, dev, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, State: "done", Evidence: &evidence, TokensSpent: &total})
	requireCode(t, err, "VALIDATION")
	total = maxAttemptTokens
	if _, err = f.s.Transition(testContext, dev, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: task.Key, Generation: claim.Lease.Generation}, State: "done", Evidence: &evidence, TokensSpent: &total}); err != nil {
		t.Fatal(err)
	}
}

// T022-R2-10a: analyst plan areas keep their original 1..1024 byte bound (so
// previously valid plans still validate) plus the printable/no-repeat rules.
func TestPlanAreasKeepLegacyByteBound(t *testing.T) {
	for _, ok := range [][]string{{strings.Repeat("a", 1024)}, {strings.Repeat("я", 300)}, {"../shared", "./src"}} {
		if err := validPlanAreas(ok); err != nil {
			t.Fatalf("previously valid plan areas rejected: %v", err)
		}
	}
	for _, bad := range [][]string{{strings.Repeat("a", 1025)}, {" "}, {"a\x1b[31m"}, {"\xff"}, {"x", "x"}, make([]string, 33)} {
		requireCode(t, validPlanAreas(bad), "VALIDATION")
	}
}

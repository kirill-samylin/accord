package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

// Poison only the caller-owned public DTO graph, including map values, raw JSON
// byte slices and pointer targets. No cache implementation detail is inspected.
func performanceExtraPoison(value any) {
	var visit func(reflect.Value)
	visit = func(v reflect.Value) {
		if !v.IsValid() {
			return
		}
		if v.CanSet() && v.Type() == reflect.TypeFor[time.Time]() {
			v.Set(reflect.ValueOf(time.Unix(1, 0).UTC()))
			return
		}
		switch v.Kind() {
		case reflect.Pointer:
			if !v.IsNil() {
				visit(v.Elem())
			}
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if v.Field(i).CanSet() {
					visit(v.Field(i))
				}
			}
		case reflect.Slice:
			for i := 0; i < v.Len(); i++ {
				visit(v.Index(i))
			}
		case reflect.Map:
			for _, key := range v.MapKeys() {
				copy := reflect.New(v.Type().Elem()).Elem()
				copy.Set(v.MapIndex(key))
				visit(copy)
				v.SetMapIndex(key, copy)
			}
		case reflect.String:
			if v.CanSet() {
				v.SetString("caller-corruption")
			}
		case reflect.Int, reflect.Int64:
			if v.CanSet() {
				v.SetInt(v.Int() + 1)
			}
		case reflect.Uint8:
			if v.CanSet() {
				v.SetUint('!')
			}
		case reflect.Bool:
			if v.CanSet() {
				v.SetBool(!v.Bool())
			}
		}
	}
	visit(reflect.ValueOf(value))
}

func performanceExtraJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func performanceExtraTaskDetached(t *testing.T, f *fixture, returned core.Task) {
	t.Helper()
	key, want := returned.Key, performanceExtraJSON(t, returned)
	durable := performanceExtraDump(t, f)
	performanceExtraPoison(&returned)
	if got := performanceExtraDump(t, f); got != durable {
		t.Fatal("mutating returned task changed durable state")
	}
	got, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: key})
	if err != nil || performanceExtraJSON(t, got) != want {
		t.Fatalf("returned graph aliases next Get: %v\nwant %s\ngot %s", err, want, performanceExtraJSON(t, got))
	}
}

func TestPerformanceExtraReturnedTaskClaimAndReplayGraphs(t *testing.T) {
	f := newFixture(t)
	resetConfig(t, f, func(c *config.Config) {
		r := c.Roles["developer"]
		r.EvidenceOnDone = append(r.EvidenceOnDone, "pr")
		c.Roles["developer"] = r
		c.Rules.EvidenceRequired.Mode = "warn"
		c.Areas = []config.Area{{Name: "src", Aliases: []string{}}}
	})
	dev := f.actor(t, "developer")
	created, err := f.s.Create(testContext, f.coord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "detached graph", Description: "implement", Role: "developer", Areas: []string{"src/core"}})
	if err != nil {
		t.Fatal(err)
	}
	performanceExtraTaskDetached(t, f, created)
	consumer := f.task(t, "developer")
	if _, err = f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: created.Key, To: consumer.Key, Type: "blocks", Condition: "completed"}); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.Note(testContext, dev, core.NoteInput{RequestID: f.id(), Action: "create", ScopeType: "epic", ScopeID: f.epic.Key, SourceTaskKey: created.Key, Text: "retained epic note"}); err != nil {
		t.Fatal(err)
	}
	in := core.NextInput{RequestID: f.id(), TaskKey: created.Key}
	claim, err := f.s.Next(testContext, dev, in)
	if err != nil || claim.Task == nil || claim.Lease == nil || len(claim.EpicNotes) != 1 {
		t.Fatal("missing alias fixture shape", claim, err)
	}
	generation, wantClaim := claim.Lease.Generation, performanceExtraJSON(t, claim)
	performanceExtraTaskDetached(t, f, *claim.Task)
	performanceExtraPoison(&claim)
	replay, err := f.s.Next(testContext, dev, in)
	if err != nil || performanceExtraJSON(t, replay) != wantClaim {
		t.Fatal("caller corruption changed durable claim replay", err)
	}
	const exact int64 = 9007199254740993
	commented, err := f.s.Comment(testContext, dev, core.CommentInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: created.Key, Generation: generation}, Text: "nested comment", TokensSpentDelta: new(exact)})
	if err != nil {
		t.Fatal(err)
	}
	performanceExtraTaskDetached(t, f, commented)
	finish := core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: created.Key, Generation: generation}, State: "done", Evidence: evidencePtr(core.EvidenceOf(map[string]any{"commit": commitA, "run_summary": "checked", "tokens_spent": json.Number("9007199254740993")}))}
	done, err := f.s.Transition(testContext, dev, finish)
	if err != nil {
		t.Fatal(err)
	}
	if len(done.Warnings) != 1 || done.EvidenceComplete == nil || done.Attempts[0].EndedAt == nil || done.Attempts[0].TokensSpent == nil || *done.Attempts[0].TokensSpent != exact {
		t.Fatal("missing precise nested completion fixture", done)
	}
	wantDone := performanceExtraJSON(t, done)
	performanceExtraTaskDetached(t, f, done)
	replayedDone, err := f.s.Transition(testContext, dev, finish)
	if err != nil || performanceExtraJSON(t, replayedDone) != wantDone {
		t.Fatal("completion raw bytes/int64 changed on replay", err)
	}
	page, err := f.s.ListTasks(testContext, f.coord, core.ListInput{IncludeClosed: true})
	if err != nil {
		t.Fatal(err)
	}
	wantPage := performanceExtraJSON(t, page)
	performanceExtraPoison(&page)
	fresh, err := f.s.ListTasks(testContext, f.coord, core.ListInput{IncludeClosed: true})
	if err != nil || performanceExtraJSON(t, fresh) != wantPage {
		t.Fatal("list graph aliases retained snapshot", err)
	}
	events, err := f.s.Events(testContext, f.coord, core.EventsInput{Limit: 200})
	if err != nil {
		t.Fatal(err)
	}
	wantEvents := performanceExtraJSON(t, events)
	performanceExtraPoison(&events)
	events, err = f.s.Events(testContext, f.coord, core.EventsInput{Limit: 200})
	if err != nil || performanceExtraJSON(t, events) != wantEvents {
		t.Fatal("event time/token pointer graph aliases snapshot", err)
	}
}

func TestPerformanceExtraReturnedSessionNotesEventsAndConfigGraphs(t *testing.T) {
	f := newFixture(t)
	architect := f.actor(t, "architect")
	source := f.task(t, "architect")
	note, err := f.s.Note(testContext, architect, core.NoteInput{RequestID: f.id(), Action: "create", ScopeType: "role", ScopeID: "architect", SourceTaskKey: source.Key, Text: "role note"})
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := f.s.Note(testContext, architect, core.NoteInput{RequestID: f.id(), Action: "delete", Key: note.Key, Reason: "replace"})
	if err != nil || deleted.DeletedAt == nil {
		t.Fatal(deleted, err)
	}
	wantNotes, err := f.s.ListNotes(testContext, f.coord, core.NotesInput{IncludeDeleted: true})
	if err != nil {
		t.Fatal(err)
	}
	notesJSON := performanceExtraJSON(t, wantNotes)
	performanceExtraPoison(&deleted)
	performanceExtraPoison(&wantNotes)
	freshNotes, err := f.s.ListNotes(testContext, f.coord, core.NotesInput{IncludeDeleted: true})
	if err != nil || performanceExtraJSON(t, freshNotes) != notesJSON {
		t.Fatal("deleted note pointer aliases snapshot", err)
	}
	claim := f.claim(t, architect, source.Key)
	hb := core.HeartbeatInput{RequestID: f.id(), RoleGeneration: 1, TaskKey: source.Key, Generation: claim.Lease.Generation}
	view, err := f.s.Heartbeat(testContext, architect, hb)
	if err != nil || view.RoleExpiresAt == nil {
		t.Fatal(view, err)
	}
	wantView := performanceExtraJSON(t, view)
	performanceExtraPoison(&view)
	view, err = f.s.Heartbeat(testContext, architect, core.HeartbeatInput{RequestID: f.id(), RoleGeneration: 1, TaskKey: source.Key, Generation: claim.Lease.Generation})
	if err != nil || performanceExtraJSON(t, view) != wantView {
		t.Fatal("heartbeat role deadline aliases snapshot", err)
	}
	sessions, err := f.s.Sessions(testContext, f.coord, core.ListInput{})
	if err != nil {
		t.Fatal(err)
	}
	wantSessions := performanceExtraJSON(t, sessions)
	performanceExtraPoison(&sessions)
	sessions, err = f.s.Sessions(testContext, f.coord, core.ListInput{})
	if err != nil || performanceExtraJSON(t, sessions) != wantSessions {
		t.Fatal("session catalogue aliases snapshot", err)
	}
	if _, err = f.s.Note(testContext, f.coord, core.NoteInput{RequestID: f.id(), Action: "create", ScopeType: "role", ScopeID: "coordinator", SourceTaskKey: source.Key, Text: "session summary note"}); err != nil {
		t.Fatal(err)
	}
	open := core.OpenSession{RequestID: f.id(), Role: "coordinator", RunID: "caller-view", Machine: f.p.Machine}
	view, err = f.s.Describe(testContext, f.p, open)
	if err != nil || len(view.Notes) == 0 || len(view.Rules) == 0 {
		t.Fatal("missing returned rules/notes map fixture", view, err)
	}
	wantView = performanceExtraJSON(t, view)
	durable := performanceExtraDump(t, f)
	performanceExtraPoison(&view)
	view, err = f.s.Describe(testContext, f.p, open)
	if err != nil || performanceExtraJSON(t, view) != wantView || performanceExtraDump(t, f) != durable {
		t.Fatal("describe rules/notes aliases retained or replay state", err)
	}
	configView, err := f.s.ReadConfig(testContext, f.p)
	if err != nil {
		t.Fatal(err)
	}
	wantConfig := performanceExtraJSON(t, configView)
	performanceExtraPoison(&configView)
	history, err := f.s.ReadConfigHistory(testContext, f.p, 0, 200)
	if err != nil {
		t.Fatal(err)
	}
	wantHistory := performanceExtraJSON(t, history)
	performanceExtraPoison(&history)
	history, err = f.s.ReadConfigHistory(testContext, f.p, 0, 200)
	if err != nil || performanceExtraJSON(t, history) != wantHistory {
		t.Fatal("config history RawMessage aliases persisted revisions", err)
	}
	configView, err = f.s.ReadConfig(testContext, f.p)
	if err != nil || performanceExtraJSON(t, configView) != wantConfig {
		t.Fatal("config RawMessage aliases active rules", err)
	}
}

func TestPerformanceExtraReturnedReviewSimilarityAndWarningAreas(t *testing.T) {
	f := newFixture(t)
	resetConfig(t, f, func(c *config.Config) {
		c.Rules.AreaOverlap.Mode = "warn"
		c.Rules.DuplicateCheck.Mode = "warn"
		c.Areas = []config.Area{{Name: "src", Aliases: []string{}}}
	})
	dev, otherDev := f.actor(t, "developer"), f.actor(t, "developer")
	in := core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "same area title", Description: "implement", Role: "developer", Areas: []string{"src"}}
	first, err := f.s.Create(testContext, f.coord, in)
	if err != nil {
		t.Fatal(err)
	}
	in.RequestID = f.id()
	second, err := f.s.Create(testContext, f.coord, in)
	if err != nil || len(second.Similar) == 0 {
		t.Fatal("missing similarity graph fixture", second, err)
	}
	performanceExtraTaskDetached(t, f, second)
	firstClaim := f.claim(t, dev, first.Key)
	secondClaim := f.claim(t, otherDev, second.Key)
	if len(secondClaim.Task.Warnings) == 0 || len(secondClaim.Task.Warnings[0].Areas) == 0 {
		t.Fatal("missing overlap warning area slice", secondClaim)
	}
	performanceExtraTaskDetached(t, f, *secondClaim.Task)
	f.finish(t, dev, firstClaim, core.EvidenceOf(map[string]any{"commit": commitA, "run_summary": "checked"}), "")
	review := f.verification(t, "reviewer", first.Key, commitA)
	if review.Review == nil || review.Review.AuthorSession != dev.SessionID {
		t.Fatal("missing bound review graph", review)
	}
	performanceExtraTaskDetached(t, f, review)
}

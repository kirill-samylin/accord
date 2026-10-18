package store

import (
	"agent-board/internal/core"
	"encoding/json"
	"testing"
	"time"
)

func performanceFixReopenedEpic(t *testing.T, f *fixture) core.Task {
	t.Helper()
	f.closeEpic(t, f.epic.Key)
	ep, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: f.epic.Key, Reason: "new work"}, State: "open"})
	if err != nil {
		t.Fatal(err)
	}
	return ep
}

func TestPerformanceFixLocalHandoffReturnsDetachedGraph(t *testing.T) {
	for _, richHistory := range []bool{false, true} {
		t.Run(map[bool]string{false: "api_history", true: "nested_persisted_history"}[richHistory], func(t *testing.T) {
			f := newFixture(t)
			ep := performanceFixReopenedEpic(t, f)
			if richHistory {
				// Store accepts historical Task JSON without rewriting its optional
				// fields. Include the full nested graph to protect this admin path
				// from a field-specific copy that happens to fix FirstDoneAt only.
				now, exact, complete := f.clock.read(), int64(9007199254740993), true
				ep.FirstClaimAt = &now
				ep.Attempts = []core.Attempt{{StartedAt: now, EndedAt: &now, TokensSpent: &exact, Reason: "historical"}}
				ep.Areas = []string{"src/core"}
				ep.Links = []core.Link{{From: ep.Key, To: ep.Key, Type: "related"}}
				ep.Comments = []core.Comment{{Text: "historical", At: now}}
				ep.Warnings = []core.RuleWarning{{Rule: "area_overlap", Message: "historical", Areas: []string{"src/core"}}}
				ep.Similar = []core.SimilarTask{{Key: ep.Key, Score: 0.9}}
				ep.Evidence = core.EvidenceOf(map[string]any{"summary": "historical", "nested": map[string]any{"numbers": []any{json.Number("9007199254740993")}}})
				ep.EvidenceComplete = &complete
				ep.Review = &core.Review{TargetKey: ep.Key, TargetCommit: commitA}
				raw := performanceExtraJSON(t, ep)
				if _, err := f.s.db.Exec(`UPDATE tasks SET data=? WHERE key=?`, raw, ep.Key); err != nil {
					t.Fatal(err)
				}
				ep = f.get(t, ep.Key) // External SQL must invalidate the retained cache.
			}
			out, err := f.s.HandoffEpic(testContext, ep.Key, "alice", ep.Version, "handoff", "local-unit-admin")
			if err != nil || out.CreatedAt == nil || out.FirstDoneAt == nil || out.Version != ep.Version+1 {
				t.Fatal("missing handoff fixture", out, err)
			}
			// Check sequential isolation first, so a broken baseline fails with a
			// deterministic alias assertion before concurrent map access begins.
			performanceExtraTaskDetached(t, f, out)
			out, err = f.s.HandoffEpic(testContext, ep.Key, "alice", out.Version, "handoff again", "local-unit-admin")
			if err != nil {
				t.Fatal(err)
			}
			want, key := performanceExtraJSON(t, out), out.Key
			durable := performanceExtraDump(t, f)
			// Poison caller memory while another goroutine reads the committed
			// graph. The race run must also catch shared nested pointer targets.
			done := make(chan struct{})
			go func() {
				defer close(done)
				for range 10 {
					performanceExtraPoison(&out)
				}
			}()
			for range 10 {
				if got := performanceExtraJSON(t, f.get(t, key)); got != want {
					<-done
					t.Fatalf("local handoff result aliases retained graph: want=%s got=%s", want, got)
				}
			}
			<-done
			if performanceExtraDump(t, f) != durable {
				t.Fatal("caller poisoning changed SQL")
			}
			other := performanceExtraOther(t, f)
			got, err := other.Get(testContext, f.coord, core.GetInput{TaskKey: key})
			if err != nil || performanceExtraJSON(t, got) != want {
				t.Fatal("fresh SQL reader differs from retained handoff graph", err)
			}
		})
	}
}

func TestPerformanceFixLocalHandoffFailureIsAtomic(t *testing.T) {
	f := newFixture(t)
	ep := performanceFixReopenedEpic(t, f)
	if _, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "bob", Machine: "synthetic", TTL: time.Hour}); err != nil {
		t.Fatal(err)
	}
	if _, err := f.s.db.Exec(`CREATE TRIGGER performance_fix_handoff_audit BEFORE INSERT ON events WHEN json_extract(NEW.data,'$.type')='epic_handoff' BEGIN SELECT RAISE(ABORT,'handoff audit unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	durable := performanceExtraDump(t, f)
	out, err := f.s.HandoffEpic(testContext, ep.Key, "bob", ep.Version, "handoff", "local-unit-admin")
	if err == nil || out.Key != "" {
		t.Fatal("failed handoff exposed successful result", out, err)
	}
	if performanceExtraDump(t, f) != durable || performanceExtraJSON(t, f.get(t, ep.Key)) != performanceExtraJSON(t, ep) {
		t.Fatal("failed handoff published owner/version/audit/projection")
	}
	if _, err = f.s.db.Exec(`DROP TRIGGER performance_fix_handoff_audit`); err != nil {
		t.Fatal(err)
	}
	out, err = f.s.HandoffEpic(testContext, ep.Key, "bob", ep.Version, "handoff", "local-unit-admin")
	if err != nil || out.Owner != "bob" || out.Version != ep.Version+1 {
		t.Fatal("retry after audit rollback failed", out, err)
	}
	durable = performanceExtraDump(t, f)
	_, err = f.s.HandoffEpic(testContext, ep.Key, "alice", ep.Version, "stale", "local-unit-admin")
	requireCode(t, err, "CONFLICT")
	if performanceExtraDump(t, f) != durable {
		t.Fatal("stale handoff changed durable state")
	}
}

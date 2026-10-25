package acceptance

import (
	"bytes"
	"encoding/json"
	"math"
	"testing"
	"time"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Prefer JSON text to the SDK any decoder, which cannot preserve >2^53 itself.
func exactMCP[T any](t *testing.T, r *mcp.CallToolResult) T {
	t.Helper()
	if r.IsError || len(r.Content) != 1 {
		t.Fatal("unexpected tool error (response suppressed)")
	}
	text, ok := r.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("missing exact JSON text")
	}
	return decode[T](t, []byte(text.Text))
}

func TestCatalogExactTokensStdioChildWire(t *testing.T) {
	b := newBoard(t, "30s")
	coord := newClient(t, b, "metrics-coordinator")
	worker := newClient(t, b, "metrics-participant")
	b.start()
	coord.open(t, "coordinator", "metrics-coordinator")
	task := coord.task(t, coord.epic(t).Key, "developer")
	p := startMCP(t, worker, b.url)
	exactMCP[core.SessionView](t, p.call("describe", payload{"request_id": "metrics-open", "role": "developer", "run_id": "metrics", "machine": "metrics-participant"}))
	claim := exactMCP[core.Claim](t, p.call("next", payload{"request_id": "metrics-claim", "task_key": task.Key}))
	// Counters are bounded to 10^12 per attempt; use the largest valid values.
	// Exact >2^53 decoding is covered by internal/bridge TestResultJSONKeepsExactInt64.
	const delta int64 = 999_999_999_999
	const total int64 = 1_000_000_000_000
	comment := payload{"request_id": "metrics-comment", "task_key": task.Key, "generation": claim.Lease.Generation, "text": "Exact synthetic counter", "tokens_spent_delta": delta}
	for i := 0; i < 2; i++ {
		got := exactMCP[core.Task](t, p.call("comment", comment))
		if len(got.Attempts) != 1 || got.Attempts[0].TokensSpent == nil || *got.Attempts[0].TokensSpent != delta || len(got.Comments) != 1 {
			t.Fatal("exact delta/replay was rounded or duplicated")
		}
	}
	window := payload{"from": time.Now().Add(-time.Hour), "to": time.Now().Add(time.Hour), "group_by": "owner"}
	active := decode[core.Stats](t, coord.ok(t, "stats", window))
	if active.Totals.Attempts != 0 || active.Totals.TokensSpent != 0 {
		t.Fatal("active delta leaked into closed-attempt statistics")
	}
	doneInput := payload{"request_id": "metrics-done", "task_key": task.Key, "generation": claim.Lease.Generation, "state": "done", "tokens_spent": total, "evidence": payload{"commit": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "run_summary": "Synthetic process pass"}}
	for i := 0; i < 2; i++ {
		got := exactMCP[core.Task](t, p.call("transition", doneInput))
		if got.State != "done" || *got.Attempts[0].TokensSpent != total || got.Attempts[0].TokensDelta != delta {
			t.Fatal("final int64 total changed across replay")
		}
	}
	events := exactMCP[core.EventsPage](t, p.call("events", payload{"task_key": task.Key, "since_cursor": 0, "limit": 200}))
	deltas, finals := 0, 0
	var closed time.Time
	for _, e := range events.Events {
		if e.Data == nil {
			continue
		}
		if e.Data.TokensSpentDelta != nil {
			deltas++
			if *e.Data.TokensSpentDelta != delta {
				t.Fatal("delta event lost precision")
			}
		}
		if e.Data.TokensSpentTotal != nil {
			finals++
			if *e.Data.TokensSpentTotal != total || e.Data.CorrectionDelta == nil || *e.Data.CorrectionDelta != total-delta {
				t.Fatal("final correction event lost precision")
			}
		}
		if e.Type == "task_done" {
			closed = e.At
		}
	}
	if deltas != 1 || finals != 1 || closed.IsZero() {
		t.Fatal("replay duplicated or omitted token audit")
	}
	exactMCP[core.Task](t, p.call("get", payload{"task_key": task.Key}))
	p.stop()
	p.assertProtocol(worker.token)
	// Check actual child stdout, including structuredContent, independently of SDK.
	checked := 0
	for _, line := range bytes.Split(p.stdout.data(), []byte{'\n'}) {
		var frame struct {
			Result struct {
				Structured json.RawMessage `json:"structuredContent"`
			} `json:"result"`
		}
		if json.Unmarshal(line, &frame) != nil || len(frame.Result.Structured) == 0 {
			continue
		}
		var got core.Task
		if json.Unmarshal(frame.Result.Structured, &got) != nil || got.Key != task.Key || len(got.Attempts) == 0 {
			continue
		}
		want := delta
		if got.State == "done" {
			want = total
		}
		if got.Attempts[0].TokensSpent == nil || *got.Attempts[0].TokensSpent != want {
			t.Fatal("child structuredContent rounded exact int64")
		}
		checked++
	}
	if checked != 5 {
		t.Fatalf("expected comment/replay/done/replay/get wire frames, got %d", checked)
	}
	b.stop()
	b.start()
	before := decode[core.Stats](t, coord.ok(t, "stats", payload{"from": closed.Add(-time.Hour), "to": closed, "group_by": "owner"}))
	after := decode[core.Stats](t, coord.ok(t, "stats", payload{"from": closed, "to": closed.Add(time.Second), "group_by": "owner"}))
	if before.Totals.Attempts != 0 || after.Totals.Attempts != 1 || after.Totals.TokensSpent != total || after.Totals.MissingTokens != 0 || after.IncompleteData {
		t.Fatal("closed-attempt [from,to) stats did not survive restart")
	}
	found := false
	for _, g := range after.Groups {
		if g.Key == "metrics-participant" {
			found = g.Metrics.Attempts == 1 && g.Metrics.TokensSpent == total && g.Metrics.FirstCompletions == 1
		}
		if g.Key == "metrics-coordinator" && (g.Metrics.Attempts != 0 || g.Metrics.TokensSpent != 0) {
			t.Fatal("attempt attributed to coordinator instead of participant")
		}
	}
	if !found {
		t.Fatal("participant owner group missing")
	}
	t.Log("C-02 exact bounded delta and final total on child wire; replay/audit and participant [from,to) stats persisted")
}

func TestCatalogCancelledBlockedDurationProcess(t *testing.T) {
	b := newBoard(t, "30s")
	c := newClient(t, b, "blocked-coordinator")
	b.start()
	c.open(t, "coordinator", "blocked-coordinator")
	epic := c.epic(t)
	source, target := c.task(t, epic.Key, "analyst"), c.task(t, epic.Key, "developer")
	c.ok(t, "link", payload{"from": source.Key, "to": target.Key, "type": "blocks"})
	c.ok(t, "transition", payload{"task_key": target.Key, "state": "cancelled", "reason": "Synthetic scope cancellation"})
	events := decode[core.EventsPage](t, c.ok(t, "events", payload{"task_key": target.Key, "limit": 200}))
	var begin, end time.Time
	for _, e := range events.Events {
		if e.Type == "blocked_changed" && e.Data != nil && e.Data.Blocked {
			begin = e.At
		}
		if e.Data != nil && e.Data.NewState == "cancelled" {
			end = e.At
		}
	}
	if begin.IsZero() || !end.After(begin) {
		t.Fatal("observed blocked/cancel events missing or unordered")
	}
	b.stop()
	b.start()
	for _, to := range []time.Time{end, end.Add(time.Hour)} {
		stat := decode[core.Stats](t, c.ok(t, "stats", payload{"from": begin, "to": to, "group_by": "role"}))
		if math.Abs(stat.Totals.BlockedDurationSeconds-end.Sub(begin).Seconds()) > 1e-9 {
			t.Fatal("cancelled blocked span kept accruing after cancellation/restart")
		}
	}
	clip := begin.Add(end.Sub(begin) / 2)
	stat := decode[core.Stats](t, c.ok(t, "stats", payload{"from": clip, "to": end.Add(time.Second), "group_by": "role"}))
	if math.Abs(stat.Totals.BlockedDurationSeconds-end.Sub(clip).Seconds()) > 1e-9 {
		t.Fatal("blocked span was not clipped at from")
	}
	got := decode[core.Task](t, c.ok(t, "get", payload{"task_key": target.Key}))
	if !got.Blocked || got.State != "cancelled" {
		t.Fatal("terminal metrics changed dependency gate predicate")
	}
	t.Log("C-03 real blocked/cancel events close and clip span; restart does not accrue cancelled duration")
}

package bridge

import (
	"encoding/json"
	"math"
	"strconv"
	"strings"
	"testing"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Token counters are bounded to 10^12 per attempt (below 2^53), so the
// end-to-end path uses the largest valid values; exact int64 decoding beyond
// 2^53 is covered by TestResultJSONKeepsExactInt64 below.
func TestExactInt64ThroughHTTPWorkerAndDownstreamWire(t *testing.T) {
	for _, tokens := range []int64{999_999_999_999, 1_000_000_000_000} {
		t.Run(strconv.FormatInt(tokens, 10), func(t *testing.T) {
			f := setup(t, 0)
			task := f.task()
			f.describe()
			claim := value[core.Claim](t, f.invoke("next", core.NextInput{RequestID: "precision-next", TaskKey: task.Key}))
			in := core.CommentInput{LeaseInput: core.LeaseInput{RequestID: "precision-comment", TaskKey: task.Key, Generation: claim.Lease.Generation}, Text: "exact token count", TokensSpentDelta: &tokens}
			first := exactTask(t, f.invoke("comment", in), tokens)
			replayed := exactTask(t, f.invoke("comment", in), tokens)
			if string(first) != string(replayed) {
				t.Fatal("idempotent result changed across worker replay")
			}
			exactTask(t, f.invoke("get", core.GetInput{TaskKey: task.Key}), tokens)
			events := value[core.EventsPage](t, f.invoke("events", core.EventsInput{TaskKey: task.Key}))
			deltas := 0
			for _, event := range events.Events {
				if event.Data != nil && event.Data.TokensSpentDelta != nil {
					if *event.Data.TokensSpentDelta != tokens {
						t.Fatal("event DTO rounded exact counter")
					}
					deltas++
				}
			}
			if deltas != 1 {
				t.Fatal("event counter missing or duplicated by replay")
			}
			exactTask(t, f.invoke("release", core.LeaseInput{RequestID: "precision-release", TaskKey: task.Key, Generation: claim.Lease.Generation, Reason: "handoff"}), tokens)
			// next must accept exact historical attempt counters, too.
			again := value[core.Claim](t, f.invoke("next", core.NextInput{RequestID: "precision-next-again", TaskKey: task.Key}))
			if again.Lease.Generation <= claim.Lease.Generation || *again.Task.Attempts[0].TokensSpent != tokens {
				t.Fatal("history changed or next failed to advance generation")
			}
			persisted := f.get(task.Key)
			if *persisted.Attempts[0].TokensSpent != tokens || len(persisted.Comments) != 1 || f.stderr.String() != "" {
				t.Fatal("exact successful mutation quarantined or replayed twice")
			}
			// The SDK client's any decoder itself uses float64. Inspect actual MCP
			// wire independently so the proxy cannot hide rounded structuredContent
			// behind an exact text block or this test's typed DTO decoder.
			found := 0
			for _, line := range strings.Split(f.stdout.String(), "\n") {
				var frame struct {
					Result struct {
						Structured json.RawMessage `json:"structuredContent"`
					} `json:"result"`
				}
				if json.Unmarshal([]byte(line), &frame) != nil || len(frame.Result.Structured) == 0 {
					continue
				}
				var returned core.Task
				if json.Unmarshal(frame.Result.Structured, &returned) != nil || returned.Key != task.Key || len(returned.Attempts) == 0 || returned.Attempts[0].TokensSpent == nil {
					continue
				}
				if *returned.Attempts[0].TokensSpent != tokens {
					t.Fatal("downstream structuredContent rounded int64")
				}
				found++
			}
			if found < 4 {
				t.Fatalf("expected exact comment/replay/get/release wire results, got %d", found)
			}
		})
	}
}

func TestResultJSONKeepsExactInt64(t *testing.T) {
	for _, n := range []int64{9007199254740993, math.MaxInt64} {
		text := `{"attempts":[{"tokens_spent":` + strconv.FormatInt(n, 10) + `}],"key":"T-1"}`
		result := &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
		if _, err := resultJSON(result); err != nil {
			t.Fatal(err)
		}
		wire, err := json.Marshal(result.StructuredContent)
		if err != nil || string(wire) != text {
			t.Fatalf("structuredContent rounded int64: %s %v", wire, err)
		}
	}
}

func TestOversizedTokenCounterIsValidationWithoutQuarantine(t *testing.T) {
	f := setup(t, 0)
	task := f.task()
	f.describe()
	claim := value[core.Claim](t, f.invoke("next", core.NextInput{RequestID: "bound-next", TaskKey: task.Key}))
	tokens := int64(math.MaxInt64)
	result := f.invoke("comment", core.CommentInput{LeaseInput: core.LeaseInput{RequestID: "bound-comment", TaskKey: task.Key, Generation: claim.Lease.Generation}, Text: "too many", TokensSpentDelta: &tokens})
	raw, _ := resultJSON(result)
	if !result.IsError || errorCode(raw) != "VALIDATION" || f.stderr.String() != "" {
		t.Fatalf("oversized counter: %s / %q", raw, f.stderr.String())
	}
	value[core.Task](t, f.invoke("comment", core.CommentInput{LeaseInput: core.LeaseInput{RequestID: "bound-ok", TaskKey: task.Key, Generation: claim.Lease.Generation}, Text: "still owned"}))
}

func exactTask(t *testing.T, result *mcp.CallToolResult, tokens int64) json.RawMessage {
	t.Helper()
	if result.IsError || len(result.Content) != 1 {
		t.Fatalf("successful exact counter returned tool error: %+v", result)
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatal("missing exact JSON text")
	}
	raw := json.RawMessage(text.Text)
	var task core.Task
	if err := json.Unmarshal(raw, &task); err != nil || len(task.Attempts) == 0 || task.Attempts[0].TokensSpent == nil || *task.Attempts[0].TokensSpent != tokens {
		t.Fatalf("client counter is not exact: %s", raw)
	}
	return raw
}

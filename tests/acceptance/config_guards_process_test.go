package acceptance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

func TestConfigLiveAreaQueueAndCancellationProcess(t *testing.T) {
	b := configBoard(t, func(c *config.Config) {
		c.Rules.AreaOverlap.Mode = "block"
		c.Areas = []config.Area{{Name: "public", Aliases: []string{}}, {Name: "busy", Aliases: []string{}}, {Name: "restricted", Aliases: []string{"secret"}, SingleEditorRole: "analyst"}}
		r := c.Roles["coordinator"]
		r.CanCancel = false
		c.Roles["coordinator"] = r
	})
	coord := newClient(t, b, "guard-coord")
	worker := newClient(t, b, "guard-worker")
	other := newClient(t, b, "guard-other")
	foreign := newClient(t, b, "guard-foreign")
	b.start()
	coord.open(t, "coordinator", "guard-coord")
	worker.open(t, "developer", "guard-worker")
	other.open(t, "developer", "guard-other")
	foreign.open(t, "coordinator", "guard-foreign")
	epic := coord.epic(t)
	create := func(title, area string, priority int) core.Task {
		return decode[core.Task](t, coord.ok(t, "create", payload{"type": "task", "epic_key": epic.Key, "role": "developer", "title": title, "description": "synthetic", "priority": priority, "areas": []string{area}}))
	}
	active := create("holding", "busy", 1)
	lease := worker.claim(t, active.Key)
	db := inspectDatabase(t, b.db)
	before := sqlString(t, db, "SELECT data FROM tasks WHERE key='"+active.Key+"'")
	events := sqlString(t, db, "SELECT count(*) FROM events")
	for _, area := range []string{"secret", "restricted/child"} {
		worker.fails(t, "update", "FORBIDDEN_ROLE", payload{"task_key": active.Key, "generation": lease.Lease.Generation, "areas": []string{area}})
	}
	if sqlString(t, db, "SELECT data FROM tasks WHERE key='"+active.Key+"'") != before || sqlString(t, db, "SELECT count(*) FROM events") != events {
		t.Fatal("area refusal changed task/lease/version/audit")
	}
	blocked := create("skip busy", "busy/child", 2)
	free := create("take free", "public", 3)
	skipped := sqlString(t, db, "SELECT data FROM tasks WHERE key='"+blocked.Key+"'")
	other.fails(t, "next", "AREA_OVERLAP", payload{"task_key": blocked.Key})
	got := decode[core.Claim](t, other.ok(t, "next", payload{}))
	if got.Task == nil || got.Task.Key != free.Key || sqlString(t, db, "SELECT data FROM tasks WHERE key='"+blocked.Key+"'") != skipped {
		t.Fatal("automatic queue did not atomically skip busy candidate")
	}
	eBefore := sqlString(t, db, "SELECT data FROM tasks WHERE key='"+epic.Key+"'")
	events = sqlString(t, db, "SELECT count(*) FROM events")
	foreign.fails(t, "transition", "FORBIDDEN_ROLE", payload{"task_key": epic.Key, "state": "cancelled", "reason": "foreign cannot cancel"})
	if sqlString(t, db, "SELECT data FROM tasks WHERE key='"+epic.Key+"'") != eBefore || sqlString(t, db, "SELECT count(*) FROM events") != events {
		t.Fatal("foreign epic cancellation refusal not atomic")
	}
	t.Log("D056-06 live alias/prefix area denied atomically; auto queue skips area-blocked candidate; can_cancel=false foreign epic denied without writes")
}

func TestConfigIncompleteEvidenceCannotCloseEpicProcess(t *testing.T) {
	for _, mode := range []string{"warn", "off"} {
		t.Run(mode, func(t *testing.T) {
			b := configBoard(t, func(c *config.Config) {
				r := c.Roles["developer"]
				r.EvidenceOnDone = append(r.EvidenceOnDone, "pr")
				c.Roles["developer"] = r
				c.Rules.EvidenceRequired.Mode = mode
			})
			coord := newClient(t, b, "incomplete-coord")
			dev := newClient(t, b, "incomplete-dev")
			b.start()
			coord.open(t, "coordinator", "incomplete-coord")
			dev.open(t, "developer", "incomplete-dev")
			epic := coord.epic(t)
			task := coord.task(t, epic.Key, "developer")
			claim := dev.claim(t, task.Key)
			done := decode[core.Task](t, dev.ok(t, "transition", payload{"task_key": task.Key, "generation": claim.Lease.Generation, "state": "done", "evidence": payload{"commit": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "run_summary": "synthetic"}}))
			if done.EvidenceComplete == nil || *done.EvidenceComplete {
				t.Fatal("incomplete child reported complete")
			}
			before := decode[core.Task](t, coord.ok(t, "get", payload{"task_key": epic.Key}))
			db := inspectDatabase(t, b.db)
			events := sqlString(t, db, "SELECT count(*) FROM events")
			coord.fails(t, "transition", "RULE_VIOLATION", payload{"task_key": epic.Key, "state": "done"})
			after := decode[core.Task](t, coord.ok(t, "get", payload{"task_key": epic.Key}))
			if !reflect.DeepEqual(before, after) || events != sqlString(t, db, "SELECT count(*) FROM events") {
				t.Fatal("incomplete gate rejection changed epic/audit")
			}
			t.Logf("D056-07 %s mode stores incomplete done child but cannot close epic", mode)
		})
	}
}

func TestConfigOmittedEmptyAreasDTORESTAndMCPProcess(t *testing.T) {
	b := configBoard(t, func(c *config.Config) { c.Areas = []config.Area{{Name: "inherited", Aliases: []string{}}} })
	coord := newClient(t, b, "areas-coord")
	b.start()
	coord.open(t, "coordinator", "areas-coord")
	epic := decode[core.Task](t, coord.ok(t, "create", payload{"type": "epic", "title": "Areas", "description": "inheritance", "areas": []string{"inherited"}}))
	// Real typed DTO marshal retains both absence and an explicit empty array.
	dto := core.CreateInput{RequestID: "typed-empty", Type: "task", EpicKey: epic.Key, Role: "developer", Title: "empty", Description: "typed", Areas: []string{}}
	raw, err := json.Marshal(dto)
	if err != nil || !bytes.Contains(raw, []byte(`"areas":[]`)) {
		t.Fatal("typed DTO lost explicit empty")
	}
	status, body := configHTTP(t, coord, "POST", "/api/v1/tasks", raw, true)
	if status != 200 || len(decode[core.Task](t, body).Areas) != 0 {
		t.Fatal("REST explicit empty inherited")
	}
	dto.Areas = nil
	conflict, _ := json.Marshal(dto)
	status, body = configHTTP(t, coord, "POST", "/api/v1/tasks", conflict, true)
	if status != 409 || domainCode(t, body) != "IDEMPOTENCY_CONFLICT" {
		t.Fatal("REST absence/empty shared request id did not conflict")
	}
	dto.RequestID = "typed-omitted"
	raw, _ = json.Marshal(dto)
	status, body = configHTTP(t, coord, "POST", "/api/v1/tasks", raw, true)
	if status != 200 || !reflect.DeepEqual(decode[core.Task](t, body).Areas, []string{"inherited"}) {
		t.Fatal("REST omission did not inherit")
	}
	p := startMCP(t, coord, b.url)
	exactMCP[core.SessionView](t, p.call("describe", payload{"request_id": "areas-mcp-open", "role": "coordinator", "run_id": "areas-mcp", "machine": "areas-coord"}))
	for i, empty := range []bool{false, true} {
		input := payload{"request_id": fmt.Sprintf("mcp-areas-%d", i), "type": "task", "epic_key": epic.Key, "role": "developer", "title": "MCP area", "description": "presence"}
		if empty {
			input["areas"] = []string{}
		}
		first := p.call("create", input)
		task := exactMCP[core.Task](t, first)
		if (len(task.Areas) == 0) != empty {
			t.Fatal("MCP areas presence changed")
		}
		replay := p.call("create", input)
		if replay.IsError || !bytes.Equal(mcpJSON(t, first), mcpJSON(t, replay)) {
			t.Fatal("MCP exact presence replay changed")
		}
		if empty {
			delete(input, "areas")
		} else {
			input["areas"] = []string{}
		}
		denied := p.call("create", input)
		code := domainCode(t, mcpJSON(t, denied))
		if !denied.IsError || code != "IDEMPOTENCY_CONFLICT" {
			t.Fatalf("MCP absence/empty expected IDEMPOTENCY_CONFLICT, got isError=%v code=%q payload=%s", denied.IsError, code, mcpJSON(t, denied))
		}
	}
	p.stop()
	p.assertProtocol(coord.token)
	t.Log("D056-08 typed DTO→REST and SDK→stdio MCP preserve omitted vs []; exact replay stable, opposing payload same request_id conflicts")
}

package acceptance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

func TestConfigScopesGrantsAndHumanProcess(t *testing.T) {
	b := configBoard(t, func(c *config.Config) {
		c.Auth.Local.Users = []string{"acceptance", "scope-human"}
		c.MCP.ToolsByRole["analyst"] = []string{"describe", "get", "next", "transition", "heartbeat"}
		r := c.Roles["human"]
		r.Categories = map[string]config.Category{"technical": {AnswerBy: json.RawMessage(`"any"`), ComplexityMax: "normal"}}
		c.Roles["human"] = r
	})
	agent := newClient(t, b, "default-agent")
	human := mintRoles(t, b, "scope-human", "human")
	coord := newClient(t, b, "scope-coord")
	r := command(nil, "token", "list", "--db", b.db)
	if r.err != nil {
		t.Fatal("token list failed")
	}
	tokens := decode[[]core.TokenInfo](t, r.data)
	for _, token := range tokens {
		if token.ID == agent.tokenID && (slices.Contains(token.Roles, "human") || token.ScopesVersion != 1) {
			t.Fatal("new token default escalated to human")
		}
	}
	bad := command(nil, "token", "create", "--db", b.db, "--owner", "outsider", "--machine", "outsider", "--roles", "human", "--output", filepath.Join(filepath.Dir(b.db), "denied.token"))
	localFails(t, bad, "FORBIDDEN_ROLE")
	b.start()
	agent.fails(t, "describe", "FORBIDDEN_ROLE", payload{"role": "human", "run_id": "denied", "machine": "default-agent"})
	agent.open(t, "analyst", "default-agent")
	agent.fails(t, "create", "FORBIDDEN_ROLE", payload{"type": "epic", "title": "No grant", "description": "deny"})
	// Listed catalog remains twelve tools even when this role has fewer grants.
	p := startMCP(t, agent, b.url)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tools, err := p.session.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 12 {
		t.Fatal("role changed stable twelve-tool catalog")
	}
	mcpValue[core.SessionView](t, p.call("describe", payload{"request_id": "scope-stdio", "role": "analyst", "run_id": "scope-stdio", "machine": "default-agent"}))
	p.stop()
	p.assertProtocol(agent.token)
	scope := func(version int, roles string) result {
		return command(nil, "token", "scopes", "set", "--db", b.db, "--id", human.tokenID, "--roles", roles, "--expected-version", fmt.Sprint(version), "--actor", "qa056")
	}
	update := scope(1, "human,developer")
	if update.err != nil || decode[core.TokenInfo](t, update.data).ScopesVersion != 2 {
		t.Fatal("scope CAS failed")
	}
	localFails(t, scope(1, "human"), "CONFIG_CONFLICT")
	human.open(t, "human", "scope-human")
	localFails(t, scope(2, "developer"), "CONFIG_IN_USE")
	coord.open(t, "coordinator", "scope-coord")
	epic := coord.epic(t)
	denied := decode[core.Task](t, coord.ok(t, "create", payload{"type": "task", "epic_key": epic.Key, "role": "human", "title": "Human high", "description": "restricted", "category": "technical", "complexity": "high"}))
	human.fails(t, "next", "FORBIDDEN_ROLE", payload{"task_key": denied.Key})
	allowed := decode[core.Task](t, coord.ok(t, "create", payload{"type": "task", "epic_key": epic.Key, "role": "human", "title": "Human normal", "description": "allowed", "category": "technical", "complexity": "normal"}))
	claim := human.claim(t, allowed.Key)
	human.ok(t, "transition", payload{"task_key": allowed.Key, "generation": claim.Lease.Generation, "state": "done", "evidence": payload{"answer": "Synthetic operator answer"}})
	t.Log("D056-04 default agent scopes, explicit human allowlist, scope CAS/in-use, role tool denial with stable 12 tools, human complexity claim and answer")
}

func TestConfigCustomRawEvidenceStdioProcess(t *testing.T) {
	b := configBoard(t, func(c *config.Config) {
		r := c.Roles["developer"]
		r.EvidenceOnDone = []string{"score"}
		r.EvidenceSchema = json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"score":{"type":"integer","minimum":0},"label":{"type":"string"}}}`)
		c.Roles["inspector"] = r
		c.MCP.ToolsByRole = map[string][]string{}
	})
	coord := newClient(t, b, "raw-coord")
	worker := newClient(t, b, "raw-worker")
	b.start()
	coord.open(t, "coordinator", "raw-coord")
	task := coord.task(t, coord.epic(t).Key, "inspector")
	p := startMCP(t, worker, b.url)
	exactMCP[core.SessionView](t, p.call("describe", payload{"request_id": "raw-open", "role": "inspector", "run_id": "raw", "machine": "raw-worker"}))
	claim := exactMCP[core.Claim](t, p.call("next", payload{"request_id": "raw-next", "task_key": task.Key}))
	for i, e := range []payload{{"score": "9007199254740993"}, {"score": nil}, {"score": int64(9007199254740993), "unknown": true}} {
		out := p.call("transition", payload{"request_id": fmt.Sprintf("raw-bad-%d", i), "task_key": task.Key, "generation": claim.Lease.Generation, "state": "done", "evidence": e})
		code := domainCode(t, mcpJSON(t, out))
		if !out.IsError || code != "VALIDATION" {
			t.Fatalf("bad custom evidence expected VALIDATION, got %s", code)
		}
	}
	done := exactMCP[core.Task](t, p.call("transition", payload{"request_id": "raw-done", "task_key": task.Key, "generation": claim.Lease.Generation, "state": "done", "evidence": payload{"score": json.RawMessage(`9007199254740993`), "label": "precise"}}))
	if string(done.Evidence["score"]) != "9007199254740993" {
		t.Fatal("custom raw integer rounded on SDK stdio return")
	}
	p.stop()
	p.assertProtocol(worker.token)
	db := inspectDatabase(t, b.db)
	var raw string
	if err := db.QueryRow("SELECT json_extract(data,'$.evidence.score') FROM tasks WHERE key=?", task.Key).Scan(&raw); err != nil || raw != "9007199254740993" {
		t.Fatal("store lost exact integer")
	}
	if !bytes.Contains(p.stdout.data(), []byte(`"score":9007199254740993`)) {
		t.Fatal("actual bridge stdout lacks exact custom integer")
	}
	// A DTO top-level unknown property must still fail across actual HTTP.
	input := coord.prepare(t, payload{"type": "epic", "title": "Unknown", "description": "strict", "unknown": true})
	bad := coord.raw("create", input)
	if bad.err == nil || domainCode(t, bad.data) != "VALIDATION" {
		t.Fatal("unknown DTO field accepted")
	}
	secret, _ := os.ReadFile(worker.token)
	if bytes.Contains(p.stdout.data(), bytes.TrimSpace(secret)) {
		t.Fatal("raw wire leaked token")
	}
	t.Log("D056-05 custom role raw >2^53 via SDK→stdio→HTTP→SQLite; wrong type/null/unknown evidence and DTO field rejected with VALIDATION")
}

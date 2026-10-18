package config

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func template() []byte {
	return bytes.ReplaceAll(DavinciTemplate(), []byte("OWNER"), []byte("operator-a"))
}
func configMap(t *testing.T) map[string]any {
	t.Helper()
	s, r := Parse(template(), Options{})
	if !r.Valid {
		t.Fatalf("template invalid: %+v", r)
	}
	var m map[string]any
	if err := json.Unmarshal(s.CanonicalJSON(), &m); err != nil {
		t.Fatal(err)
	}
	return m
}
func parseMap(t *testing.T, m map[string]any, o Options) (*Snapshot, Result) {
	t.Helper()
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	return Parse(raw, o)
}
func field(m map[string]any, k string) map[string]any { return m[k].(map[string]any) }
func TestDefaultsCanonicalAndIsolation(t *testing.T) {
	s, r := Parse(template(), Options{})
	if !r.Valid {
		t.Fatalf("%+v", r)
	}
	c := s.Decode()
	if c.Leases.SessionTTL != "15m0s" || len(c.MCP.ToolsByRole["developer"]) != 12 {
		t.Fatal("defaults absent")
	}
	if len(c.Roles["human"].CanCreate) != 0 || len(c.Roles["developer"].CanCreate) != 1 {
		t.Fatal("kind defaults")
	}
	again, r := Parse(s.CanonicalJSON(), Options{})
	if !r.Valid || again.SHA256() != s.SHA256() {
		t.Fatalf("canonical roundtrip: %+v", r)
	}
	c.Roles["developer"] = Role{}
	c.Auth.Local.Users[0] = "mutated"
	raw := s.CanonicalJSON()
	raw[0] = 'x'
	if s.Decode().Auth.Local.Users[0] != "operator-a" {
		t.Fatal("mutable snapshot")
	}
	task, role, session, hb := s.TTLs()
	if task != role || role != session || hb != task/3 {
		t.Fatal("TTL hints")
	}
	m := configMap(t)
	field(field(m, "roles"), "developer")["can_create"] = []any{}
	empty, result := parseMap(t, m, Options{})
	if !result.Valid || len(empty.Decode().Roles["developer"].CanCreate) != 0 {
		t.Fatal("explicit [] replaced")
	}
}
func TestRejectHostileSyntax(t *testing.T) {
	for name, raw := range map[string][]byte{
		"duplicate": append(template(), []byte("version: 1\n")...), "tag": bytes.Replace(template(), []byte("version: 1"), []byte("version: !!int 1"), 1), "anchor": bytes.Replace(template(), []byte("version: 1"), []byte("version: &v 1"), 1), "alias": []byte("x: *v"), "merge": append(template(), []byte("<<: {}\n")...), "multiple": append(template(), []byte("---\nversion: 1\n")...), "nonstring": append(template(), []byte("1: value\n")...), "utf8": {0xff}, "oversize": bytes.Repeat([]byte("x"), MaxInputBytes+1), "timestamp": bytes.Replace(template(), []byte("name: accord"), []byte("name: 2026-10-18"), 1), "null": []byte("null"), "jsonduplicate": []byte(`{"version":1,"version":1}`),
	} {
		t.Run(name, func(t *testing.T) {
			s, r := Parse(raw, Options{})
			if s != nil || r.Valid || len(r.Errors) == 0 {
				t.Fatal("accepted hostile syntax")
			}
		})
	}
}
func TestSemanticValidation(t *testing.T) {
	cases := map[string]func(map[string]any){
		"unknown": func(m map[string]any) { m["unknown"] = true }, "leaseoverflow": func(m map[string]any) { field(m, "leases")["task_ttl"] = "999999999999999h" }, "shortttl": func(m map[string]any) { field(m, "leases")["task_ttl"] = "1ms" }, "silent": func(m map[string]any) { field(m, "leases")["max_silent"] = "24h" }, "sweep": func(m map[string]any) { field(m, "leases")["sweep_interval"] = "24h" }, "escalation": func(m map[string]any) { field(m, "questions")["escalate_after"] = "-1s" }, "owner": func(m map[string]any) { field(field(m, "auth"), "local")["users"] = []any{"OWNER"} }, "reopen": func(m map[string]any) { field(m, "reopen")["tester"] = []any{"missing"} }, "role": func(m map[string]any) { field(field(m, "roles"), "tester")["independent_from"] = []any{"missing"} }, "member": func(m map[string]any) { field(field(m, "roles"), "human")["members"] = []any{"unknown"} }, "budget": func(m map[string]any) { field(m, "mcp")["tool_budget_tokens"] = 1 }, "tool": func(m map[string]any) { field(field(m, "mcp"), "tools_by_role")["developer"] = []any{"bad"} }, "missingtools": func(m map[string]any) { delete(field(field(m, "mcp"), "tools_by_role"), "developer") }, "singleton": func(m map[string]any) { field(field(m, "rules"), "singleton_role")["mode"] = "warn" }, "cycle": func(m map[string]any) { field(field(m, "rules"), "cycle_check")["mode"] = "off" }, "onetask": func(m map[string]any) { field(field(m, "rules"), "one_task_per_agent")["mode"] = "off" }, "prefix": func(m map[string]any) { field(field(m, "project"), "ids")["task_prefix"] = "X" }, "state": func(m map[string]any) { field(m, "states")["contract_delivered_at"] = "review" }, "github": func(m map[string]any) { field(m, "auth")["mode"] = "github" },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			m := configMap(t)
			change(m)
			s, r := parseMap(t, m, Options{})
			if s != nil || r.Valid || len(r.Errors) == 0 {
				t.Fatal("invalid config accepted")
			}
		})
	}
}
func TestDistributableAssetsMatch(t *testing.T) {
	for path, want := range map[string][]byte{"../../schema/agent-board-v1.json": Schema(), "../../templates/davinci.yaml": DavinciTemplate()} {
		got, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("asset drift %s: %v", path, err)
		}
	}
}
func TestNoSourceValuesInErrors(t *testing.T) {
	raw := strings.Replace(string(template()), "version: 1", "version: PRIVATE-VALUE", 1)
	_, r := Parse([]byte(raw), Options{})
	b, _ := json.Marshal(r)
	if bytes.Contains(b, []byte("PRIVATE-VALUE")) {
		t.Fatal("value exposed")
	}
}
func TestToolBudgetIsBytesAndSeparateSchemas(t *testing.T) {
	s, r := Parse(template(), Options{})
	if !r.Valid {
		t.Fatal(r)
	}
	budgets := s.ToolBudgets(nil, map[string]int{"describe": 1000})
	for _, b := range budgets {
		if b.DescriptionBytes <= 0 || b.DescriptionBytes >= 3000 || b.SchemaBytes != 1000 || b.Unit != "conservative_utf8_bytes" {
			t.Fatal(b)
		}
	}
}

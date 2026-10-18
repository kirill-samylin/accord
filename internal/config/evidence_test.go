package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestEvidenceFloorsModesAndPrecision(t *testing.T) {
	m := configMap(t)
	field(field(m, "rules"), "evidence_required")["mode"] = "warn"
	s, r := parseMap(t, m, Options{})
	if !r.Valid {
		t.Fatal(r)
	}
	valid := `{"commit":"` + strings.Repeat("a", 40) + `","run_summary":"unit passed","tokens_spent":9007199254740993}`
	result := s.ValidateEvidence("developer", json.RawMessage(valid))
	if !result.Valid || result.Complete || len(result.Warnings) != 1 {
		t.Fatalf("extra pr should warn: %+v", result)
	}
	result = s.ValidateEvidence("developer", json.RawMessage(`{"pr":"https://example.test/p/1"}`))
	if result.Valid {
		t.Fatal("warn disabled hard floor")
	}
	for name, raw := range map[string]string{"duplicate": `{"commit":"x","commit":"y"}`, "unknown": strings.Replace(valid, `"tokens_spent":9007199254740993`, `"unknown":1`, 1), "fraction": strings.Replace(valid, "9007199254740993", "1.5", 1), "negative": strings.Replace(valid, "9007199254740993", "-1", 1), "sha": strings.Replace(valid, strings.Repeat("a", 40), "garbage", 1), "uri": strings.Replace(valid, `"tokens_spent":9007199254740993`, `"pr":"http://example.test/p/1"`, 1)} {
		t.Run(name, func(t *testing.T) {
			if s.ValidateEvidence("developer", json.RawMessage(raw)).Valid {
				t.Fatal("invalid evidence accepted")
			}
		})
	}
	field(field(m, "rules"), "evidence_required")["mode"] = "off"
	s, r = parseMap(t, m, Options{})
	if !r.Valid {
		t.Fatal(r)
	}
	result = s.ValidateEvidence("developer", json.RawMessage(valid))
	if !result.Valid || result.Complete || len(result.Warnings) != 0 {
		t.Fatal(result)
	}
}
func TestCustomSchemaExactNumbersRefsFormats(t *testing.T) {
	m := configMap(t)
	field(m, "roles")["custom"] = map[string]any{"kind": "agent", "evidence_on_done": []any{"cost", "when", "target"}, "evidence_schema": json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"cost":{"$ref":"#/$defs/cost"},"when":{"type":"string","format":"date-time"},"target":{"type":"string","format":"uri"}},"$defs":{"cost":{"type":"integer","minimum":9007199254740993,"maximum":9007199254740993}}}`)}
	field(m, "mcp")["tools_by_role"] = map[string]any{}
	s, r := parseMap(t, m, Options{})
	if !r.Valid {
		t.Fatal(r)
	}
	good := `{"cost":9007199254740993,"when":"2026-10-18T12:00:00Z","target":"https://example.test/item"}`
	if got := s.ValidateEvidence("custom", json.RawMessage(good)); !got.Valid || !got.Complete {
		t.Fatal(got)
	}
	for _, raw := range []string{strings.Replace(good, "9007199254740993", "9007199254740992", 1), strings.Replace(good, "9007199254740993", "9007199254740994", 1), strings.Replace(good, "2026-10-18T12:00:00Z", "2026-99-99", 1), strings.Replace(good, "https://example.test/item", "relative-path", 1)} {
		if s.ValidateEvidence("custom", json.RawMessage(raw)).Valid {
			t.Fatal("custom constraints bypassed")
		}
	}
	if !strings.Contains(string(s.CanonicalJSON()), "9007199254740993") {
		t.Fatal("integer precision lost")
	}
}
func TestRejectedEvidenceSchemas(t *testing.T) {
	schemas := map[string]string{
		"remote":     `{"type":"object","additionalProperties":false,"properties":{"x":{"$ref":"https://example.test/schema"}}}`,
		"cycle":      `{"type":"object","additionalProperties":false,"properties":{"x":{"$ref":"#/$defs/x"}},"$defs":{"x":{"$ref":"#/$defs/y"},"y":{"$ref":"#/$defs/x"}}}`,
		"unresolved": `{"type":"object","additionalProperties":false,"properties":{"x":{"$ref":"#/$defs/missing"}}}`,
		"unknown":    `{"type":"object","additionalProperties":false,"properties":{},"script":"run"}`,
		"id":         `{"type":"object","additionalProperties":false,"properties":{},"$id":"file:/etc/passwd"}`,
		"additional": `{"type":"object","properties":{}}`, "pattern": `{"type":"object","additionalProperties":false,"properties":{"x":{"type":"string","pattern":"["}}}`,
		"format":   `{"type":"object","additionalProperties":false,"properties":{"x":{"type":"string","format":"email"}}}`,
		"required": `{"type":"object","additionalProperties":false,"properties":{},"required":["missing"]}`,
		"builtin":  `{"type":"object","additionalProperties":false,"properties":{"commit":{"type":"number"}}}`,
	}
	for name, schema := range schemas {
		t.Run(name, func(t *testing.T) {
			m := configMap(t)
			field(field(m, "roles"), "developer")["evidence_schema"] = json.RawMessage(schema)
			s, r := parseMap(t, m, Options{})
			if s != nil || r.Valid {
				t.Fatal("unsafe schema accepted")
			}
		})
	}
}
func TestAreasAndCapabilities(t *testing.T) {
	m := configMap(t)
	m["areas"] = []any{map[string]any{"name": "src/ui/", "aliases": []any{"ui"}, "single_editor_role": "developer"}, map[string]any{"name": "src/ui/shared"}}
	s, r := parseMap(t, m, Options{})
	if !r.Valid {
		t.Fatal(r)
	}
	if got, err := s.ResolveArea("ui"); err != nil || got != "src/ui" {
		t.Fatal(got, err)
	}
	if got, err := s.ResolveArea("src/ui/child"); err != nil || got != "src/ui/child" {
		t.Fatal(got, err)
	}
	if AreasIntersect("src/ui", "src/uitest") {
		t.Fatal("text prefix collision")
	}
	for _, bad := range []string{"/abs", "../parent", "a/./b", "a//b", "a\\b", "C:/a", "a\x00b", "a//"} {
		if _, err := CanonicalArea(bad); err == nil {
			t.Fatalf("accepted area %q", bad)
		}
	}
	m["areas"] = []any{map[string]any{"name": "src", "aliases": []any{"alias"}}, map[string]any{"name": "alias"}}
	if _, r := parseMap(t, m, Options{}); r.Valid {
		t.Fatal("alias collision accepted")
	}
	m = configMap(t)
	m["github"] = map[string]any{"repository_allowlist": []any{"github.com/owner/repo"}, "webhook_secret_env": "WEBHOOK_SECRET", "task_key_pattern": "T-[0-9]+", "require_approval_for_merge": true, "events": []any{"pull_request"}}
	if _, r := parseMap(t, m, Options{}); r.Valid {
		t.Fatal("ignored integration")
	}
	if _, r := parseMap(t, m, Options{GitHubEvents: true}); !r.Valid {
		t.Fatal(r)
	}
}
func TestSchemaPointerAndLimits(t *testing.T) {
	m := configMap(t)
	field(m, "roles")["custom"] = map[string]any{"kind": "agent", "evidence_schema": json.RawMessage(`{"type":"object","additionalProperties":false,"properties":{"value":{"$ref":"#/$defs/a~1b"}},"$defs":{"a/b":{"type":"integer","const":9007199254740993}}}`)}
	field(m, "mcp")["tools_by_role"] = map[string]any{}
	s, r := parseMap(t, m, Options{})
	if !r.Valid {
		t.Fatal(r)
	}
	if got := s.ValidateEvidence("custom", json.RawMessage(`{"value":9007199254740993}`)); !got.Valid {
		t.Fatal(got)
	}
	if got := s.ValidateEvidence("custom", json.RawMessage(`{"value":1e999999999}`)); got.Valid {
		t.Fatal("unbounded exponent accepted")
	}
	role := field(field(m, "roles"), "custom")
	schema := map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{}}
	props := field(schema, "properties")
	for i := 0; i < 64; i++ {
		props[strings.Repeat("x", i+1)] = map[string]any{"type": "string"}
	}
	role["evidence_schema"] = schema
	if _, r := parseMap(t, m, Options{}); r.Valid {
		t.Fatal("merged property cap exceeded")
	}
	m = configMap(t)
	field(field(m, "roles"), "human")["evidence_schema"] = map[string]any{"type": "object", "additionalProperties": false, "properties": map[string]any{}, "required": []any{"answer"}}
	if _, r := parseMap(t, m, Options{}); !r.Valid {
		t.Fatal("builtin required should resolve after injection", r)
	}
}
func TestBuiltinByteBounds(t *testing.T) {
	s, r := Parse(template(), Options{})
	if !r.Valid {
		t.Fatal(r)
	}
	raw, _ := json.Marshal(map[string]any{"summary": strings.Repeat("я", 3000)})
	if s.ValidateEvidence("coordinator", raw).Valid {
		t.Fatal("unicode bypassed byte limit")
	}
	raw, _ = json.Marshal(map[string]any{"artifact": []string{" "}, "summary": "done"})
	if s.ValidateEvidence("analyst", raw).Valid {
		t.Fatal("blank reference accepted")
	}
}
func FuzzParse(f *testing.F) {
	f.Add(template())
	f.Add([]byte("version: &x 1\n"))
	f.Add([]byte(`{"roles":{}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		if len(raw) > MaxInputBytes+1 {
			t.Skip()
		}
		snapshot, result := Parse(raw, Options{})
		if result.Valid {
			if snapshot == nil || snapshot.SHA256() == "" {
				t.Fatal("invalid success")
			}
			again, result := Parse(snapshot.CanonicalJSON(), Options{})
			if !result.Valid || again.SHA256() != snapshot.SHA256() {
				t.Fatal("noncanonical valid snapshot")
			}
		} else if snapshot != nil || len(result.Errors) == 0 {
			t.Fatal("invalid failure contract")
		}
	})
}

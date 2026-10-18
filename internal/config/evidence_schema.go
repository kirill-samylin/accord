package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
)

var schemaKeywords = map[string]bool{"type": true, "properties": true, "required": true, "additionalProperties": true, "items": true, "enum": true, "const": true, "minimum": true, "maximum": true, "minLength": true, "maxLength": true, "minItems": true, "maxItems": true, "pattern": true, "format": true, "$defs": true, "$ref": true}
var builtinSchemas = map[string]string{
	"commit":          `{"type":"string","pattern":"^(?:[a-fA-F0-9]{40}|[a-fA-F0-9]{64})$"}`,
	"verified_commit": `{"type":"string","pattern":"^(?:[a-fA-F0-9]{40}|[a-fA-F0-9]{64})$"}`,
	"pr":              `{"type":"string","format":"uri","pattern":"^https://[^/]+(?:/|$)","maxLength":2048}`,
	"artifact":        `{"type":"array","minItems":1,"maxItems":32,"items":{"type":"string","minLength":1,"maxLength":2048}}`,
	"refs":            `{"type":"array","minItems":1,"maxItems":32,"items":{"type":"string","minLength":1,"maxLength":2048}}`,
	"screens":         `{"type":"array","minItems":1,"maxItems":32,"items":{"type":"string","minLength":1,"maxLength":2048}}`,
	"summary":         `{"type":"string","minLength":1,"maxLength":4096}`, "run_summary": `{"type":"string","minLength":1,"maxLength":1024}`,
	"findings": `{"type":"string","maxLength":16384}`, "answer": `{"type":"string","minLength":1,"maxLength":16384}`,
	"run_id": `{"type":"string","minLength":1,"maxLength":128}`, "tokens_spent": `{"type":"integer","minimum":0}`,
	"verdict": `{"type":"string","enum":["approve","request_changes","changes_requested","blocked"]}`, "result": `{"type":"string","enum":["passed","failed","not_run","not_applicable"]}`,
}

func decodeJSON(raw []byte, out any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	return d.Decode(out)
}
func schemaObject(raw []byte) (map[string]any, error) {
	var m map[string]any
	err := decodeJSON(raw, &m)
	if err == nil && m == nil {
		err = fmt.Errorf("schema must be an object")
	}
	return m, err
}
func prepareEvidence(name string, role *Role) error {
	root, err := schemaObject(role.EvidenceSchema)
	if err != nil {
		return fmt.Errorf("invalid evidence schema")
	}
	nodes := 0
	if err = checkSchema(root, root, 0, &nodes, map[string]bool{}); err != nil {
		return err
	}
	if root["type"] != "object" {
		return fmt.Errorf("evidence schema root must have type object")
	}
	props, _ := root["properties"].(map[string]any)
	if props == nil {
		props = map[string]any{}
		root["properties"] = props
	}
	for key, text := range builtinSchemas {
		var builtin map[string]any
		_ = decodeJSON([]byte(text), &builtin)
		if existing, ok := props[key]; ok {
			a, _ := json.Marshal(existing)
			b, _ := json.Marshal(builtin)
			if !bytes.Equal(a, b) {
				return fmt.Errorf("built-in evidence property cannot be redefined")
			}
		}
		props[key] = builtin
	}
	for _, key := range append(append(append([]string{}, role.EvidenceOnDone...), role.RecommendedOnDone...), builtinFloors[name]...) {
		if _, ok := props[key]; !ok {
			return fmt.Errorf("configured evidence field requires an explicit schema")
		}
	}
	if len(props) > 64 {
		return fmt.Errorf("merged evidence properties exceed 64")
	}
	required := append([]string{}, role.EvidenceOnDone...)
	if list, ok := root["required"].([]any); ok {
		for _, v := range list {
			if k, ok := v.(string); ok {
				required = append(required, k)
			}
		}
	}
	if len(role.IndependentFrom) > 0 {
		_, commit := props["verified_commit"]
		verdict := slices.Contains(required, "verdict")
		result := slices.Contains(required, "result")
		if !commit || (!slices.Contains(required, "verified_commit") && !slices.Contains(builtinFloors[name], "verified_commit")) || verdict == result {
			return fmt.Errorf("verifier requires verified_commit and exactly verdict or result")
		}
	}
	nodes = 0
	if err = checkSchema(root, root, 0, &nodes, map[string]bool{}); err != nil {
		return err
	}
	role.EvidenceSchema, err = json.Marshal(root)
	return err
}
func checkSchema(s, root map[string]any, depth int, nodes *int, visiting map[string]bool) error {
	*nodes++
	if depth > 8 || *nodes > 128 {
		return fmt.Errorf("evidence schema depth/node limit exceeded")
	}
	for key := range s {
		if !schemaKeywords[key] {
			return fmt.Errorf("unknown evidence schema keyword")
		}
	}
	if ref, exists := s["$ref"]; exists {
		text, ok := ref.(string)
		if !ok || !strings.HasPrefix(text, "#/$defs/") {
			return fmt.Errorf("only local direct $defs references are supported")
		}
		if visiting[text] {
			return fmt.Errorf("cyclic evidence schema reference")
		}
		target, ok := resolveRef(root, text)
		if !ok {
			return fmt.Errorf("unresolved evidence schema reference")
		}
		copyVisit := map[string]bool{}
		for k, v := range visiting {
			copyVisit[k] = v
		}
		copyVisit[text] = true
		if err := checkSchema(target, root, depth+1, nodes, copyVisit); err != nil {
			return err
		}
	}
	if value, exists := s["type"]; exists {
		t, ok := value.(string)
		if !ok || !slices.Contains([]string{"object", "array", "string", "integer", "number", "boolean", "null"}, t) {
			return fmt.Errorf("unsupported evidence schema type")
		}
	}
	_, hasProperties := s["properties"]
	if s["type"] == "object" || hasProperties {
		if s["additionalProperties"] != false {
			return fmt.Errorf("evidence object requires additionalProperties false")
		}
	}
	if v, ok := s["additionalProperties"]; ok && v != false {
		return fmt.Errorf("additionalProperties must be false")
	}
	for _, key := range []string{"properties", "$defs"} {
		if value, exists := s[key]; exists {
			m, ok := value.(map[string]any)
			if !ok || len(m) > 64 {
				return fmt.Errorf("schema property map limit/type invalid")
			}
			for name, v := range m {
				if strings.TrimSpace(name) == "" {
					return fmt.Errorf("schema property name is blank")
				}
				child, ok := v.(map[string]any)
				if !ok {
					return fmt.Errorf("schema child must be an object")
				}
				if err := checkSchema(child, root, depth+1, nodes, visiting); err != nil {
					return err
				}
			}
		}
	}
	if v, exists := s["items"]; exists {
		child, ok := v.(map[string]any)
		if !ok {
			return fmt.Errorf("items must be one schema object")
		}
		if err := checkSchema(child, root, depth+1, nodes, visiting); err != nil {
			return err
		}
	}
	if value, ok := s["pattern"]; ok {
		p, ok := value.(string)
		if !ok {
			return fmt.Errorf("pattern must be a string")
		}
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("pattern is not a Go regular expression")
		}
	}
	if value, ok := s["format"]; ok {
		f, ok := value.(string)
		if !ok || !slices.Contains([]string{"uri", "date-time"}, f) {
			return fmt.Errorf("unsupported asserted format")
		}
	}
	if value, ok := s["required"]; ok {
		list, ok := value.([]any)
		if !ok || len(list) > 64 {
			return fmt.Errorf("required must be a bounded array")
		}
		seen := map[string]bool{}
		props, _ := s["properties"].(map[string]any)
		for _, v := range list {
			k, ok := v.(string)
			if !ok || seen[k] {
				return fmt.Errorf("required names must be unique strings")
			}
			seen[k] = true
			if _, ok := props[k]; !ok && !(depth == 0 && builtinSchemas[k] != "") {
				return fmt.Errorf("required property has no schema")
			}
		}
	}
	for _, key := range []string{"minLength", "maxLength", "minItems", "maxItems"} {
		if v, ok := s[key]; ok {
			n, ok := v.(json.Number)
			if !ok {
				return fmt.Errorf("schema limit must be an integer")
			}
			i, err := n.Int64()
			if err != nil || i < 0 || i > MaxEvidenceBytes {
				return fmt.Errorf("schema limit is out of bounds")
			}
		}
	}
	for _, key := range []string{"minimum", "maximum"} {
		if _, ok := s[key]; ok {
			if _, valid := number(s[key]); !valid {
				return fmt.Errorf("numeric bound must be a finite number")
			}
		}
	}
	if value, ok := s["enum"]; ok {
		list, ok := value.([]any)
		if !ok || len(list) == 0 || len(list) > 128 {
			return fmt.Errorf("enum must be a nonempty bounded array")
		}
	}
	return nil
}

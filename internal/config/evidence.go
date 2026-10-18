package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"unicode/utf8"
)

type EvidenceResult struct {
	Valid    bool      `json:"valid"`
	Complete bool      `json:"complete"`
	Errors   []Problem `json:"errors"`
	Warnings []Problem `json:"warnings"`
}

// ValidateEvidence asserts shape/types and built-in floors. Lease provenance,
// target commit/generation and cross-task gates remain authoritative store checks.
func (s *Snapshot) ValidateEvidence(role string, raw json.RawMessage) EvidenceResult {
	out := EvidenceResult{Complete: false, Errors: []Problem{}, Warnings: []Problem{}}
	c := s.Decode()
	r, ok := c.Roles[role]
	if !ok {
		out.Errors = append(out.Errors, Problem{"role", "CONFIG_INVALID", "unknown role"})
		return out
	}
	value, err := strictEvidence(raw)
	if err != nil {
		out.Errors = append(out.Errors, Problem{"evidence", "VALIDATION", err.Error()})
		return out
	}
	root, err := schemaObject(r.EvidenceSchema)
	if err != nil {
		out.Errors = append(out.Errors, Problem{"evidence", "CONFIG_INTERNAL", "snapshot schema is invalid"})
		return out
	}
	out.Complete = true
	required := append([]string{}, r.EvidenceOnDone...)
	if list, ok := root["required"].([]any); ok {
		for _, v := range list {
			if text, ok := v.(string); ok {
				required = append(required, text)
			}
		}
	}
	delete(root, "required")
	if err = validateBuiltinValues(value); err != nil {
		out.Errors = append(out.Errors, Problem{"evidence", "VALIDATION", err.Error()})
	}
	if err = validateValue(value, root, root, 0); err != nil {
		out.Errors = append(out.Errors, Problem{"evidence", "VALIDATION", err.Error()})
	}
	for _, key := range builtinFloors[role] {
		if _, exists := value[key]; !exists {
			out.Errors = append(out.Errors, Problem{"evidence." + key, "EVIDENCE_REQUIRED", "built-in evidence is required"})
		}
	}
	for _, key := range required {
		if _, exists := value[key]; exists || slices.Contains(builtinFloors[role], key) {
			continue
		}
		out.Complete = false
		p := Problem{"evidence." + key, "EVIDENCE_REQUIRED", "configured evidence is missing"}
		switch c.Rules.EvidenceRequired.Mode {
		case "block":
			out.Errors = append(out.Errors, p)
		case "warn":
			out.Warnings = append(out.Warnings, p)
		}
	}
	out.Valid = len(out.Errors) == 0
	if !out.Valid {
		out.Complete = false
	}
	return out
}
func strictEvidence(raw []byte) (map[string]any, error) {
	if len(raw) > MaxEvidenceBytes || !utf8.Valid(raw) || !json.Valid(raw) {
		return nil, fmt.Errorf("evidence must be one UTF-8 JSON object at most 32 KiB")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	nodes := 0
	v, err := readJSON(d, 0, &nodes)
	if err != nil {
		return nil, err
	}
	if _, err = d.Token(); err != io.EOF {
		return nil, fmt.Errorf("trailing JSON value")
	}
	m, ok := v.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("evidence must be an object")
	}
	return m, nil
}
func readJSON(d *json.Decoder, depth int, nodes *int) (any, error) {
	*nodes++
	if depth > 16 || *nodes > 4096 {
		return nil, fmt.Errorf("evidence nesting/node limit exceeded")
	}
	token, err := d.Token()
	if err != nil {
		return nil, fmt.Errorf("invalid JSON")
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return token, nil
	}
	switch delimiter {
	case '{':
		m := map[string]any{}
		for d.More() {
			token, err := d.Token()
			if err != nil {
				return nil, err
			}
			key, ok := token.(string)
			if !ok {
				return nil, fmt.Errorf("invalid JSON object key")
			}
			if _, exists := m[key]; exists {
				return nil, fmt.Errorf("duplicate evidence property")
			}
			v, err := readJSON(d, depth+1, nodes)
			if err != nil {
				return nil, err
			}
			m[key] = v
		}
		_, err = d.Token()
		return m, err
	case '[':
		a := []any{}
		for d.More() {
			v, err := readJSON(d, depth+1, nodes)
			if err != nil {
				return nil, err
			}
			a = append(a, v)
		}
		_, err = d.Token()
		return a, err
	}
	return nil, fmt.Errorf("invalid JSON delimiter")
}

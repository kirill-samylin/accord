package config

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"strings"
	"unicode/utf8"

	"github.com/google/jsonschema-go/jsonschema"
	"go.yaml.in/yaml/v3"
)

//go:embed agent-board-v1.json
var shapeJSON []byte

//go:embed davinci.yaml
var davinciYAML []byte

// Schema returns a copy of the distributable configuration JSON Schema.
func Schema() []byte { return append([]byte(nil), shapeJSON...) }

// DavinciTemplate contains OWNER placeholders; callers must replace them explicitly.
func DavinciTemplate() []byte { return append([]byte(nil), davinciYAML...) }

func fail(path, code, message string) (*Snapshot, Result) {
	return nil, Result{Errors: []Problem{{path, code, message}}, Warnings: []Problem{}}
}

// Parse validates one complete YAML/JSON document without loading references or
// expanding variables. Errors contain paths and diagnostics, never source values.
func Parse(data []byte, opts Options) (*Snapshot, Result) {
	if len(data) > MaxInputBytes || !utf8.Valid(data) {
		return fail("$", "CONFIG_INVALID", "input must be UTF-8 and at most 256 KiB")
	}
	d := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := d.Decode(&doc); err != nil || len(doc.Content) != 1 {
		return fail("$", "CONFIG_INVALID", "one valid YAML document is required")
	}
	var extra yaml.Node
	if err := d.Decode(&extra); err != io.EOF {
		return fail("$", "CONFIG_INVALID", "multiple documents are forbidden")
	}
	count := 0
	raw, err := nodeValue(doc.Content[0], 0, &count)
	if err != nil {
		return fail("$", "CONFIG_INVALID", err.Error())
	}
	root, ok := raw.(map[string]any)
	if !ok {
		return fail("$", "CONFIG_INVALID", "configuration must be an object")
	}
	if project, ok := root["project"].(map[string]any); ok {
		if ids, ok := project["ids"].(map[string]any); ok {
			for k, want := range map[string]string{"epic_prefix": "E", "task_prefix": "T", "note_prefix": "N"} {
				if v, exists := ids[k]; exists && v != want {
					return fail("project.ids."+k, "CONFIG_UNSUPPORTED", "prefix changes are unsupported")
				}
			}
		}
	}
	if states, ok := root["states"].(map[string]any); ok {
		if v, exists := states["contract_delivered_at"]; exists && v != "done" {
			return fail("states.contract_delivered_at", "CONFIG_UNSUPPORTED", "contracts are delivered only at done")
		}
	}
	var schema jsonschema.Schema
	if err := json.Unmarshal(shapeJSON, &schema); err != nil {
		return fail("$", "CONFIG_INTERNAL", "embedded schema is invalid")
	}
	resolved, err := schema.Resolve(nil)
	if err != nil {
		return fail("$", "CONFIG_INTERNAL", "embedded schema cannot resolve")
	}
	if err = resolved.Validate(shapeValue(root)); err != nil {
		return fail("$", "CONFIG_INVALID", "configuration contains invalid, missing or unknown fields")
	}
	encoded, err := json.Marshal(root)
	if err != nil {
		return fail("$", "CONFIG_INVALID", "configuration numbers cannot be encoded")
	}
	var c Config
	if err = json.Unmarshal(encoded, &c); err != nil {
		return fail("$", "CONFIG_INVALID", "configuration types are invalid")
	}
	materialize(&c)
	result := Result{Errors: []Problem{}, Warnings: []Problem{}}
	validateConfig(&c, opts, &result)
	if len(result.Errors) > 0 {
		return nil, result
	}
	canonical, err := json.Marshal(c)
	if err != nil {
		return fail("$", "CONFIG_INTERNAL", "canonical encoding failed")
	}
	hash := sha256.Sum256(canonical)
	s := &Snapshot{canonical: canonical, hash: hex.EncodeToString(hash[:])}
	result.Valid = true
	result.CanonicalSHA256 = s.hash
	return s, result
}

func nodeValue(n *yaml.Node, depth int, count *int) (any, error) {
	*count++
	if depth > 24 || *count > 16384 {
		return nil, fmt.Errorf("document nesting or node limit exceeded")
	}
	if n.Anchor != "" || n.Kind == yaml.AliasNode || n.Style&yaml.TaggedStyle != 0 {
		return nil, fmt.Errorf("tags, anchors and aliases are forbidden")
	}
	switch n.Kind {
	case yaml.MappingNode:
		m := map[string]any{}
		for i := 0; i < len(n.Content); i += 2 {
			k := n.Content[i]
			if k.Kind != yaml.ScalarNode || k.Tag != "!!str" || k.Value == "<<" || k.Anchor != "" || k.Style&yaml.TaggedStyle != 0 {
				return nil, fmt.Errorf("mapping keys must be plain strings; merge keys are forbidden")
			}
			if _, exists := m[k.Value]; exists {
				return nil, fmt.Errorf("duplicate mapping key")
			}
			v, err := nodeValue(n.Content[i+1], depth+1, count)
			if err != nil {
				return nil, err
			}
			m[k.Value] = v
		}
		return m, nil
	case yaml.SequenceNode:
		out := make([]any, 0, len(n.Content))
		for _, child := range n.Content {
			v, err := nodeValue(child, depth+1, count)
			if err != nil {
				return nil, err
			}
			out = append(out, v)
		}
		return out, nil
	case yaml.ScalarNode:
		switch n.Tag {
		case "!!str":
			return n.Value, nil
		case "!!int":
			text := strings.ReplaceAll(n.Value, "_", "")
			value, ok := new(big.Int).SetString(text, 0)
			if !ok {
				value, ok = new(big.Int).SetString(text, 10)
			}
			if !ok {
				return nil, fmt.Errorf("invalid integer")
			}
			return json.Number(value.String()), nil
		case "!!float":
			text := strings.ReplaceAll(n.Value, "_", "")
			text = strings.TrimPrefix(text, "+")
			if strings.HasPrefix(text, ".") {
				text = "0" + text
			}
			if strings.HasPrefix(text, "-.") {
				text = "-0" + text[1:]
			}
			if strings.HasSuffix(text, ".") {
				text += "0"
			}
			if !json.Valid([]byte(text)) {
				return nil, fmt.Errorf("invalid or non-finite number")
			}
			return json.Number(text), nil
		case "!!bool", "!!null":
			var v any
			if err := n.Decode(&v); err != nil {
				return nil, fmt.Errorf("invalid scalar")
			}
			return v, nil
		default:
			return nil, fmt.Errorf("unsupported scalar type")
		}
	default:
		return nil, fmt.Errorf("unsupported YAML node")
	}
}

// The schema library treats json.Number as string for type checks. Only this
// disposable shape view uses native numbers; canonical/source values stay exact.
func shapeValue(v any) any {
	switch value := v.(type) {
	case json.Number:
		if i, err := value.Int64(); err == nil {
			return i
		}
		f, err := value.Float64()
		if err != nil {
			return value
		}
		return f
	case map[string]any:
		out := map[string]any{}
		for k, child := range value {
			if k == "evidence_schema" {
				out[k] = child
			} else {
				out[k] = shapeValue(child)
			}
		}
		return out
	case []any:
		out := make([]any, len(value))
		for i, child := range value {
			out[i] = shapeValue(child)
		}
		return out
	default:
		return value
	}
}

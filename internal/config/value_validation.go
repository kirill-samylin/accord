package config

import (
	"fmt"
	"math/big"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

func validateValue(v any, s, root map[string]any, depth int) error {
	if depth > 16 {
		return fmt.Errorf("evidence schema expansion limit exceeded")
	}
	if ref, ok := s["$ref"].(string); ok {
		target, ok := resolveRef(root, ref)
		if !ok {
			return fmt.Errorf("invalid compiled reference")
		}
		if err := validateValue(v, target, root, depth+1); err != nil {
			return err
		}
	}
	t, _ := s["type"].(string)
	valid := true
	switch t {
	case "object":
		_, valid = v.(map[string]any)
	case "array":
		_, valid = v.([]any)
	case "string":
		_, valid = v.(string)
	case "boolean":
		_, valid = v.(bool)
	case "null":
		valid = v == nil
	case "number":
		_, valid = number(v)
	case "integer":
		n, ok := number(v)
		valid = ok && n.IsInt()
	}
	if !valid {
		return fmt.Errorf("evidence value has the wrong type")
	}
	if allowed, ok := s["enum"].([]any); ok {
		found := false
		for _, candidate := range allowed {
			if equalJSON(v, candidate) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("evidence value is outside its enum")
		}
	}
	if constant, ok := s["const"]; ok && !equalJSON(v, constant) {
		return fmt.Errorf("evidence value does not match const")
	}
	if n, ok := number(v); ok {
		for _, key := range []string{"minimum", "maximum"} {
			if bound, ok := number(s[key]); ok {
				cmp := n.Cmp(bound)
				if (key == "minimum" && cmp < 0) || (key == "maximum" && cmp > 0) {
					return fmt.Errorf("evidence number exceeds bounds")
				}
			}
		}
	}
	if text, ok := v.(string); ok {
		size := utf8.RuneCountInString(text)
		if violatesLimit(size, s, "minLength", "maxLength") {
			return fmt.Errorf("evidence string length exceeds bounds")
		}
		if len(text) > MaxEvidenceBytes {
			return fmt.Errorf("evidence string exceeds byte limit")
		}
		if pattern, ok := s["pattern"].(string); ok {
			re, err := regexp.Compile(pattern)
			if err != nil || !re.MatchString(text) {
				return fmt.Errorf("evidence string does not match pattern")
			}
		}
		switch s["format"] {
		case "uri":
			u, err := url.Parse(text)
			if err != nil || !u.IsAbs() || strings.ContainsAny(text, " \t\n\r") {
				return fmt.Errorf("evidence URI is invalid")
			}
		case "date-time":
			if _, err := time.Parse(time.RFC3339Nano, text); err != nil {
				return fmt.Errorf("evidence date-time is invalid")
			}
		}
	}
	if items, ok := v.([]any); ok {
		if violatesLimit(len(items), s, "minItems", "maxItems") {
			return fmt.Errorf("evidence array length exceeds bounds")
		}
		if child, ok := s["items"].(map[string]any); ok {
			for _, item := range items {
				if err := validateValue(item, child, root, depth+1); err != nil {
					return err
				}
			}
		}
	}
	if object, ok := v.(map[string]any); ok {
		properties, _ := s["properties"].(map[string]any)
		for key, value := range object {
			if child, ok := properties[key].(map[string]any); ok {
				if err := validateValue(value, child, root, depth+1); err != nil {
					return err
				}
			} else if s["additionalProperties"] == false {
				return fmt.Errorf("unknown evidence property")
			}
		}
		if list, ok := s["required"].([]any); ok {
			for _, key := range list {
				text, _ := key.(string)
				if _, ok := object[text]; !ok {
					return fmt.Errorf("required nested evidence property is missing")
				}
			}
		}
	}
	return nil
}
func violatesLimit(value int, s map[string]any, minKey, maxKey string) bool {
	if n, ok := number(s[minKey]); ok && new(big.Rat).SetInt64(int64(value)).Cmp(n) < 0 {
		return true
	}
	if n, ok := number(s[maxKey]); ok && new(big.Rat).SetInt64(int64(value)).Cmp(n) > 0 {
		return true
	}
	return false
}
func equalJSON(a, b any) bool {
	if an, ok := number(a); ok {
		bn, valid := number(b)
		return valid && an.Cmp(bn) == 0
	}
	switch value := a.(type) {
	case map[string]any:
		other, ok := b.(map[string]any)
		if !ok || len(value) != len(other) {
			return false
		}
		for key, v := range value {
			ov, exists := other[key]
			if !exists || !equalJSON(v, ov) {
				return false
			}
		}
		return true
	case []any:
		other, ok := b.([]any)
		if !ok || len(value) != len(other) {
			return false
		}
		for i, v := range value {
			if !equalJSON(v, other[i]) {
				return false
			}
		}
		return true
	case string:
		other, ok := b.(string)
		return ok && value == other
	case bool:
		other, ok := b.(bool)
		return ok && value == other
	case nil:
		return b == nil
	default:
		return false
	}
}

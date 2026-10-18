package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"unicode/utf8"
)

// Evidence preserves exact raw JSON numbers and custom configured fields. Typed
// accessors report a wrong type rather than silently treating it as missing.
type Evidence map[string]json.RawMessage

func (e Evidence) String(key string) (string, error) {
	raw, ok := e[key]
	if !ok {
		return "", nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return "", fmt.Errorf("evidence.%s must be a string", key)
	}
	return value, nil
}
func (e Evidence) Strings(key string) ([]string, error) {
	raw, ok := e[key]
	if !ok {
		return nil, nil
	}
	var value []string
	if err := json.Unmarshal(raw, &value); err != nil || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return nil, fmt.Errorf("evidence.%s must be a string array", key)
	}
	return value, nil
}
func (e Evidence) Integer(key string) (int64, bool, error) {
	raw, ok := e[key]
	if !ok {
		return 0, false, nil
	}
	value, ok := new(big.Rat).SetString(string(raw))
	if !ok || !value.IsInt() || !value.Num().IsInt64() {
		return 0, true, fmt.Errorf("evidence.%s must be an int64", key)
	}
	return value.Num().Int64(), true, nil
}
func (e Evidence) Text(key string) string { value, _ := e.String(key); return value }

// EvidenceOf is a convenience constructor for typed internal fixtures. Runtime
// input is decoded directly into RawMessages and is validated before completion.
func EvidenceOf(values map[string]any) Evidence {
	out := Evidence{}
	for key, value := range values {
		raw, err := json.Marshal(value)
		if err != nil {
			panic(err)
		}
		out[key] = raw
	}
	return out
}

// UnmarshalJSON rejects duplicate top-level fields before a map can discard them.
// Nested JSON is retained verbatim for the bounded configured schema validator.
func (e *Evidence) UnmarshalJSON(raw []byte) error {
	if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		*e = nil
		return nil
	}
	if len(raw) > 32<<10 || !utf8.Valid(raw) {
		return fmt.Errorf("evidence exceeds UTF8 byte limit")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	token, err := d.Token()
	if err != nil || token != json.Delim('{') {
		return fmt.Errorf("evidence must be an object")
	}
	out := Evidence{}
	for d.More() {
		token, err = d.Token()
		if err != nil {
			return err
		}
		key, ok := token.(string)
		if !ok {
			return fmt.Errorf("invalid evidence key")
		}
		if _, exists := out[key]; exists {
			return fmt.Errorf("duplicate evidence key")
		}
		var value json.RawMessage
		if err = d.Decode(&value); err != nil {
			return err
		}
		out[key] = value
	}
	if _, err = d.Token(); err != nil {
		return err
	}
	if _, err = d.Token(); err != io.EOF {
		return fmt.Errorf("trailing evidence")
	}
	*e = out
	return nil
}

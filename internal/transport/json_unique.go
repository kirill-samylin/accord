package transport

import (
	"agent-board/internal/core"
	"bytes"
	"encoding/json"
	"io"
)

// encoding/json accepts repeated object keys by default. Reject them before
// canonicalization so the caller's exact meaning cannot be silently changed.
func uniqueJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := uniqueValue(d); err != nil {
		return core.Fail("validation", "invalid or duplicate JSON fields")
	}
	if _, err := d.Token(); err != io.EOF {
		return core.Fail("validation", "exactly one JSON value required")
	}
	return nil
}
func uniqueValue(d *json.Decoder) error {
	token, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return err
			}
			name, ok := key.(string)
			if !ok || seen[name] {
				return core.Fail("validation", "duplicate JSON key")
			}
			seen[name] = true
			if err = uniqueValue(d); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err = uniqueValue(d); err != nil {
				return err
			}
		}
	default:
		return core.Fail("validation", "invalid JSON delimiter")
	}
	_, err = d.Token()
	return err
}

// ValidateJSONFields lets the CLI reject ambiguous input before converting it
// to a field map. Native HTTP and MCP use the same duplicate-key validation.
func ValidateJSONFields(data []byte) error { return uniqueJSON(data) }

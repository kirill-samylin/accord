package web

import (
	"agent-board/internal/core"
	"bytes"
	"encoding/json"
	"io"
)

// encoding/json accepts repeated object keys by default. Reject them before
// canonicalization so the caller's exact meaning cannot be silently changed.
func automationUniqueJSON(data []byte) error {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if err := automationUniqueValue(d, 0); err != nil {
		return core.Fail("validation", "invalid or duplicate JSON fields")
	}
	if _, err := d.Token(); err != io.EOF {
		return core.Fail("validation", "exactly one JSON value required")
	}
	return nil
}
func automationUniqueValue(d *json.Decoder, depth int) error {
	if depth > 32 {
		return core.Fail("validation", "JSON nesting too deep")
	}
	token, err := d.Token()
	if err != nil {
		return err
	}
	if token == nil {
		return core.Fail("validation", "null settings fields forbidden")
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
			if err = automationUniqueValue(d, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err = automationUniqueValue(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return core.Fail("validation", "invalid JSON delimiter")
	}
	_, err = d.Token()
	return err
}

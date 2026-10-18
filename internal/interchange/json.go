// Package interchange validates untrusted, non-authoritative portability data.
package interchange

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"strings"
	"time"
	"unicode/utf8"
)

// CheckJSON rejects duplicate keys, malformed UTF-8 and excessive nesting before
// ordinary decoding. Numbers remain json.Number; no float64 round trip occurs.
func CheckJSON(raw []byte, maxDepth int) error {
	if !utf8.Valid(raw) {
		return errors.New("invalid UTF-8")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := walkJSON(d, 0, maxDepth); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return errors.New("trailing JSON")
	}
	return nil
}
func walkJSON(d *json.Decoder, depth, max int) error {
	t, err := d.Token()
	if err != nil {
		return err
	}
	delim, ok := t.(json.Delim)
	if !ok {
		return nil
	}
	if depth >= max {
		return errors.New("nesting limit")
	}
	switch delim {
	case '{':
		keys := map[string]bool{}
		for d.More() {
			k, err := d.Token()
			if err != nil {
				return err
			}
			s, ok := k.(string)
			if !ok || keys[s] {
				return errors.New("duplicate object key")
			}
			keys[s] = true
			if err := walkJSON(d, depth+1, max); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := walkJSON(d, depth+1, max); err != nil {
				return err
			}
		}
	default:
		return errors.New("unexpected closing delimiter")
	}
	_, err = d.Token()
	return err
}

// DecodeStrict rejects duplicate/non-exact/unknown fields and missing required
// properties. Size limits remain the caller's responsibility.
func DecodeStrict(raw []byte, dst any) error {
	if err := CheckJSON(raw, 32); err != nil {
		return err
	}
	typ := reflect.TypeOf(dst)
	if typ == nil || typ.Kind() != reflect.Pointer || reflect.ValueOf(dst).IsNil() {
		return errors.New("non-nil pointer required")
	}
	return strict(raw, dst)
}
func strict(raw []byte, dst any) error {
	if err := exactFields(raw, reflect.TypeOf(dst).Elem()); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	d.UseNumber()
	return d.Decode(dst)
}

// exactFields closes encoding/json's case-insensitive field-name matching and
// distinguishes missing required scalars from their legitimate zero values.
func exactFields(raw []byte, typ reflect.Type) error {
	if typ.Kind() == reflect.Pointer {
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return nil
		}
		return exactFields(raw, typ.Elem())
	}
	if typ == reflect.TypeFor[json.RawMessage]() || typ == reflect.TypeFor[time.Time]() {
		return nil
	}
	switch typ.Kind() {
	case reflect.Struct:
		if !bytes.HasPrefix(bytes.TrimSpace(raw), []byte("{")) {
			return errors.New("object required")
		}
		var values map[string]json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return err
		}
		fields := map[string]reflect.Type{}
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			tag := strings.Split(field.Tag.Get("json"), ",")
			if tag[0] == "-" {
				continue
			}
			name := tag[0]
			if name == "" {
				name = field.Name
			}
			fields[name] = field.Type
			if len(tag) == 1 {
				if _, ok := values[name]; !ok {
					return errors.New("missing required field")
				}
			}
		}
		for name, value := range values {
			field, ok := fields[name]
			if !ok {
				return errors.New("unknown field")
			}
			if err := exactFields(value, field); err != nil {
				return err
			}
		}
	case reflect.Slice:
		var values []json.RawMessage
		if err := json.Unmarshal(raw, &values); err != nil {
			return err
		}
		for _, value := range values {
			if err := exactFields(value, typ.Elem()); err != nil {
				return err
			}
		}
	default:
		if bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
			return errors.New("null scalar")
		}
	}
	return nil
}

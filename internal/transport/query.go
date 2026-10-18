package transport

import (
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"agent-board/internal/core"
)

// readQuery rejects unknown, duplicate and malformed parameters before any
// storage access. Typed inputs retain missing-vs-zero validation in the store.
func readQuery(kind, raw string) ([]byte, error) {
	query, err := url.ParseQuery(raw)
	if err != nil {
		return nil, core.Fail("VALIDATION", "malformed query parameters")
	}
	fields := make(map[string]any, len(query))
	for key, values := range query {
		if len(values) != 1 || values[0] == "" {
			return nil, core.Fail("VALIDATION", "query parameters must occur once with a value")
		}
		value := values[0]
		if kind == "stats" {
			switch key {
			case "from", "to":
				parsed, e := time.Parse(time.RFC3339Nano, value)
				if e != nil {
					return nil, core.Fail("VALIDATION", "stats dates must be RFC3339")
				}
				fields[key] = parsed.UTC()
			case "group_by":
				fields[key] = value
			default:
				return nil, core.Fail("VALIDATION", "unknown stats query parameter")
			}
			continue
		}
		switch key {
		case "limit":
			n, e := strconv.Atoi(value)
			if e != nil || n < 1 || n > 200 {
				return nil, core.Fail("VALIDATION", "limit must be between 1 and 200")
			}
			fields[key] = n
		case "cursor":
			fields[key] = value
		case "include_closed":
			if kind == "notes" {
				return nil, core.Fail("VALIDATION", "unknown notes query parameter")
			}
			v, e := queryBool(value)
			if e != nil {
				return nil, e
			}
			fields[key] = v
		case "scope_type", "scope_id", "include_deleted":
			if kind != "notes" {
				return nil, core.Fail("VALIDATION", "unknown list query parameter")
			}
			if key == "include_deleted" {
				v, e := queryBool(value)
				if e != nil {
					return nil, e
				}
				fields[key] = v
			} else {
				fields[key] = value
			}
		case "state", "role", "epic_key", "owner":
			if kind == "notes" {
				return nil, core.Fail("VALIDATION", "unknown notes query parameter")
			}
			fields[key] = value
		default:
			return nil, core.Fail("VALIDATION", "unknown list query parameter")
		}
	}
	return json.Marshal(fields)
}

func queryBool(raw string) (bool, error) {
	if raw != "true" && raw != "false" {
		return false, core.Fail("VALIDATION", "boolean query parameters require true or false")
	}
	return raw == "true", nil
}

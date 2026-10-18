package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/url"

	"agent-board/internal/core"
)

func readClientRoute(operation string, fields map[string]json.RawMessage) (string, string, []byte, error) {
	switch operation {
	case "tasks", "epics", "sessions":
		return typedReadRoute[core.ListInput](operation, fields)
	case "notes":
		return typedReadRoute[core.NotesInput](operation, fields)
	case "stats":
		return typedReadRoute[core.StatsInput](operation, fields)
	default:
		return "", "", nil, errors.New("unknown read operation")
	}
}

// Re-encoding a typed DTO both rejects unknown fields and preserves exact
// integer strings. It never decodes pagination or counters through float64.
func typedReadRoute[I any](operation string, fields map[string]json.RawMessage) (string, string, []byte, error) {
	data, err := json.Marshal(fields)
	if err != nil {
		return "", "", nil, errors.New("invalid read fields")
	}
	var input I
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&input); err != nil {
		return "", "", nil, errors.New("invalid " + operation + " fields")
	}
	// Only supplied fields become query parameters; omitted dates and limits
	// remain omitted and are validated by the authoritative service.
	query := url.Values{}
	for key, raw := range fields {
		if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
			return "", "", nil, errors.New("read fields must not be null")
		}
		var text string
		if raw[0] == '"' {
			if err := json.Unmarshal(raw, &text); err != nil {
				return "", "", nil, errors.New("invalid read string")
			}
		} else {
			text = string(raw)
		}
		query.Set(key, text)
	}
	return "GET", "/api/v1/" + operation + "?" + query.Encode(), nil, nil
}

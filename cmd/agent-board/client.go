package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"agent-board/internal/client"
	"agent-board/internal/core"
	"agent-board/internal/transport"
)

const maxResponseBytes = client.MaxResponseBytes

func call(args []string, in io.Reader, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: agent-board call <operation> [--server URL] [--token-file FILE] < request.json")
	}
	operation := args[0]
	f := flags("call")
	server := f.String("server", "http://127.0.0.1:8080", "server origin")
	project := f.String("project", "", "opaque project ID; omitted pins the legacy default")
	tokenFile := f.String("token-file", "", "private token file; otherwise AGENT_BOARD_TOKEN")
	if err := parse(f, args[1:]); err != nil {
		return err
	}
	if err := client.Project(*project); err != nil {
		return err
	}
	base, err := client.Origin(*server)
	if err != nil {
		return err
	}
	token, err := readToken(*tokenFile)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(in, transport.MaxBodyBytes+1))
	if err != nil || int64(len(data)) > transport.MaxBodyBytes {
		return errors.New("request JSON exceeds limit or cannot be read")
	}
	if err := transport.ValidateJSONFields(data); err != nil {
		return errors.New("stdin contains invalid or duplicate JSON fields")
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil || fields == nil {
		return errors.New("stdin must contain one JSON object")
	}
	session, err := takeString(fields, "session_id")
	if err != nil {
		return err
	}
	method, path, body, err := clientRoute(operation, fields)
	if err != nil {
		return err
	}
	if *project != "" {
		path = "/api/v1/p/" + *project + strings.TrimPrefix(path, "/api/v1")
	}
	target := *base
	target.Path = strings.TrimSuffix(base.Path, "/") + path
	if method == http.MethodGet {
		parts := strings.SplitN(path, "?", 2)
		target.Path = strings.TrimSuffix(base.Path, "/") + parts[0]
		if len(parts) == 2 {
			target.RawQuery = parts[1]
		}
	}
	req, err := http.NewRequest(method, target.String(), bytes.NewReader(body))
	if err != nil {
		return errors.New("cannot construct request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-Session", session)
	if method != http.MethodGet {
		req.Header.Set("Content-Type", "application/json")
	}
	httpClient := client.HTTP(base, token, 30*time.Second)
	response, err := httpClient.Do(req)
	if err != nil {
		return errors.New("request failed; ownership may be uncertain: stop local edits, reconnect and verify your lease")
	}
	defer response.Body.Close()
	result, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(result) > maxResponseBytes {
		return errors.New("response exceeds limit or cannot be read; ownership may be uncertain: stop local edits, reconnect and verify your lease")
	}
	if _, err = out.Write(result); err != nil {
		return err
	}
	if response.StatusCode >= 300 {
		return fmt.Errorf("server returned HTTP %d", response.StatusCode)
	}
	return nil
}

func readToken(path string) (string, error) { return client.ReadToken(path) }

func takeString(fields map[string]json.RawMessage, name string) (string, error) {
	var value string
	if data, ok := fields[name]; ok {
		if err := json.Unmarshal(data, &value); err != nil {
			return "", errors.New(name + " must be a string")
		}
		delete(fields, name)
	}
	return value, nil
}

func clientRoute(operation string, fields map[string]json.RawMessage) (string, string, []byte, error) {
	method, path := "POST", ""
	switch operation {
	case "describe":
		path = "/api/v1/session"
	case "create":
		path = "/api/v1/tasks"
	case "next":
		path = "/api/v1/tasks/next"
	case "link":
		path = "/api/v1/links"
	case "heartbeat":
		path = "/api/v1/heartbeat"
	case "note":
		path = "/api/v1/notes"
	case "profile_create", "profile_update", "profile_archive":
		path = "/api/v1/" + operation
	case "profiles", "profile":
		return profileClientRoute(operation, fields)
	case "tasks", "epics", "sessions", "notes", "stats":
		return readClientRoute(operation, fields)
	case "events":
		data, _ := json.Marshal(fields)
		var input core.EventsInput
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			return "", "", nil, errors.New("invalid events fields")
		}
		query := url.Values{}
		query.Set("since_cursor", fmt.Sprint(input.SinceCursor))
		if input.Limit != 0 {
			query.Set("limit", fmt.Sprint(input.Limit))
		}
		if len(input.Types) > 0 {
			query.Set("types", strings.Join(input.Types, ","))
		}
		if input.TaskKey != "" {
			query.Set("task_key", input.TaskKey)
		}
		return "GET", "/api/v1/events?" + query.Encode(), nil, nil
	case "get", "update", "transition", "release", "comment", "coordination", "assign", "check_create":
		field := "task_key"
		if operation == "check_create" {
			field = "target_key"
		}
		key, err := takeString(fields, field)
		if err != nil || key == "" || strings.ContainsAny(key, "/?#%") {
			return "", "", nil, errors.New("canonical task_key is required")
		}
		path = "/api/v1/tasks/" + key
		switch operation {
		case "get", "coordination":
			if len(fields) != 0 {
				return "", "", nil, errors.New("unknown get fields")
			}
			if operation == "coordination" {
				path += "/coordination"
			}
			return "GET", path, nil, nil
		case "update":
			method = "PATCH"
		case "check_create":
			path += "/checks"
		case "assign", "transition", "release":
			path += "/" + operation
		case "comment":
			path += "/comments"
		}
	default:
		return "", "", nil, errors.New("unknown operation")
	}
	body, err := json.Marshal(fields)
	return method, path, body, err
}

func profileClientRoute(operation string, fields map[string]json.RawMessage) (string, string, []byte, error) {
	query := url.Values{}
	for key, raw := range fields {
		if operation == "profiles" && key == "state" || operation == "profile" && key == "profile_id" {
			var value string
			if json.Unmarshal(raw, &value) != nil {
				return "", "", nil, errors.New("invalid profile field")
			}
			query.Set(key, value)
		} else if operation == "profile" && key == "version" {
			var value int64
			if json.Unmarshal(raw, &value) != nil || value < 1 {
				return "", "", nil, errors.New("invalid profile version")
			}
			query.Set(key, fmt.Sprint(value))
		} else {
			return "", "", nil, errors.New("unknown profile field")
		}
	}
	return "GET", "/api/v1/" + operation + "?" + query.Encode(), nil, nil
}

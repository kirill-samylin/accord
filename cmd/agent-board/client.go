package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"agent-board/internal/core"
	"agent-board/internal/transport"
)

const maxResponseBytes = 1 << 20

func call(args []string, in io.Reader, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: agent-board call <operation> [--server URL] [--token-file FILE] < request.json")
	}
	operation := args[0]
	f := flags("call")
	server := f.String("server", "http://127.0.0.1:8080", "server origin")
	tokenFile := f.String("token-file", "", "private token file; otherwise AGENT_BOARD_TOKEN")
	if err := parse(f, args[1:]); err != nil {
		return err
	}
	base, err := url.Parse(*server)
	if err != nil || base.Host == "" || base.User != nil || base.RawQuery != "" || base.Fragment != "" || base.Path != "" && base.Path != "/" {
		return errors.New("server must be an HTTP(S) origin without credentials")
	}
	if base.Scheme != "https" {
		if base.Scheme != "http" || transport.ValidateListen(base.Host) != nil {
			return errors.New("HTTP requires a literal loopback address; remote servers require HTTPS")
		}
	}
	token, err := readToken(*tokenFile)
	if err != nil {
		return err
	}
	data, err := io.ReadAll(io.LimitReader(in, transport.MaxBodyBytes+1))
	if err != nil || int64(len(data)) > transport.MaxBodyBytes {
		return errors.New("request JSON exceeds limit or cannot be read")
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
	client := &http.Client{Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return errors.New("redirects are disabled") }}
	response, err := client.Do(req)
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

func readToken(path string) (string, error) {
	token := os.Getenv("AGENT_BOARD_TOKEN")
	if path != "" {
		info, err := os.Stat(path)
		if err != nil || !info.Mode().IsRegular() || info.Mode().Perm()&0077 != 0 {
			return "", errors.New("token file must be a private regular file (mode 0600)")
		}
		data, err := os.ReadFile(path)
		if err != nil || len(data) > 4096 {
			return "", errors.New("cannot read private token file")
		}
		token = strings.TrimSpace(string(data))
	}
	if token == "" || strings.ContainsAny(token, "\r\n \t") {
		return "", errors.New("set AGENT_BOARD_TOKEN or provide --token-file")
	}
	return token, nil
}

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
	case "get", "update", "transition", "release", "comment":
		key, err := takeString(fields, "task_key")
		if err != nil || key == "" || strings.ContainsAny(key, "/?#%") {
			return "", "", nil, errors.New("canonical task_key is required")
		}
		path = "/api/v1/tasks/" + key
		switch operation {
		case "get":
			if len(fields) != 0 {
				return "", "", nil, errors.New("unknown get fields")
			}
			return "GET", path, nil, nil
		case "update":
			method = "PATCH"
		case "transition", "release":
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

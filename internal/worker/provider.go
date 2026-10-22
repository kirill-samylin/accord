package worker

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
)

func providerArgs(provider, session string) ([]string, error) {
	if session != "" && !sessionID.MatchString(session) {
		return nil, errors.New("invalid exact provider session ID")
	}
	switch provider {
	case "codex":
		if session != "" {
			return []string{"exec", "resume", "--json", session, "-"}, nil
		}
		return []string{"exec", "--json", "-"}, nil
	case "claude":
		args := []string{"--print", "--verbose", "--output-format", "stream-json"}
		if session != "" {
			args = append(args, "--resume", session)
		}
		return args, nil
	}
	return nil, errors.New("unsupported native provider")
}

// Native authentication and permission settings stay under local provider control.
// Accord credentials never reach the native process or its descendants.
func providerEnv() []string {
	var env []string
	for _, value := range os.Environ() {
		key, _, _ := strings.Cut(value, "=")
		upper := strings.ToUpper(key)
		if strings.HasPrefix(upper, "AGENT_BOARD_") || strings.HasPrefix(upper, "ACCORD_") {
			continue
		}
		env = append(env, value)
	}
	return env
}

type providerEvent struct {
	session  string
	complete bool
	failed   bool
	started  bool
	output   string
}

func parseProviderEvent(provider string, line []byte) providerEvent {
	var event struct {
		Type      string `json:"type"`
		Subtype   string `json:"subtype"`
		ThreadID  string `json:"thread_id"`
		SessionID string `json:"session_id"`
		IsError   bool   `json:"is_error"`
		Result    string `json:"result"`
		Item      struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"item"`
	}
	if json.Unmarshal(line, &event) != nil {
		return providerEvent{}
	}
	out := providerEvent{}
	if provider == "codex" {
		if event.Type == "thread.started" && sessionID.MatchString(event.ThreadID) {
			out.session = event.ThreadID
		}
		out.complete = event.Type == "turn.completed"
		out.started = event.Type == "turn.started"
		out.failed = event.Type == "turn.failed" || event.Type == "error"
		if event.Type == "item.completed" && event.Item.Type == "agent_message" {
			out.output = event.Item.Text
		}
	} else if provider == "claude" {
		if sessionID.MatchString(event.SessionID) {
			out.session = event.SessionID
		}
		out.complete = event.Type == "result" && !event.IsError && event.Subtype == "success"
		out.failed = event.Type == "result" && (event.IsError || event.Subtype != "success")
		out.started = event.Type == "assistant"
		if out.complete {
			out.output = event.Result
		}
	}
	return out
}

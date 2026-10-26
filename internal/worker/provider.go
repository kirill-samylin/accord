package worker

import (
	"encoding/json"
	"errors"
	"os"
	"strings"

	"agent-board/internal/core"
)

// nativeProviders are the only locally runnable native CLIs. Gennadiy is the
// corporate OpenCode fork: headless `run --format json`, exact `--session`.
var nativeProviders = core.NativeProviders

func profileProviderFor(provider string) string { return core.ProfileProviderFor(provider) }
func knownProvider(provider string) bool        { return core.KnownNativeProvider(provider) }

// validSessionID accepts only the exact native conversation ID format of the
// provider: UUID for Codex/Claude, `ses_` identifiers for Gennadiy/OpenCode.
func validSessionID(provider, session string) bool {
	if provider == "gennadiy" {
		return gennadiySessionID.MatchString(session)
	}
	return sessionID.MatchString(session)
}

func providerArgs(provider, session string) ([]string, error) {
	if session != "" && !validSessionID(provider, session) {
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
	case "gennadiy":
		// The prompt arrives on stdin; `build` is the native coding agent whose
		// permissions the injected local preset pins. No --continue/--fork/last.
		args := []string{"run", "--format", "json", "--agent", gennadiyAgent}
		if session != "" {
			args = append(args, "--session", session)
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
		if strings.HasPrefix(upper, "AGENT_BOARD_") || strings.HasPrefix(upper, "ACCORD_") || strings.HasPrefix(upper, "OPENCODE_CONFIG") {
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
		Type       string `json:"type"`
		Subtype    string `json:"subtype"`
		ThreadID   string `json:"thread_id"`
		SessionID  string `json:"session_id"`
		SessionRef string `json:"sessionID"`
		IsError    bool   `json:"is_error"`
		Result     string `json:"result"`
		Item       struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"item"`
		Part struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"part"`
	}
	if json.Unmarshal(line, &event) != nil {
		return providerEvent{}
	}
	out := providerEvent{}
	switch provider {
	case "codex":
		if event.Type == "thread.started" && sessionID.MatchString(event.ThreadID) {
			out.session = event.ThreadID
		}
		out.complete = event.Type == "turn.completed"
		out.started = event.Type == "turn.started"
		out.failed = event.Type == "turn.failed" || event.Type == "error"
		if event.Type == "item.completed" && event.Item.Type == "agent_message" {
			out.output = event.Item.Text
		}
	case "claude":
		if sessionID.MatchString(event.SessionID) {
			out.session = event.SessionID
		}
		out.complete = event.Type == "result" && !event.IsError && event.Subtype == "success"
		out.failed = event.Type == "result" && (event.IsError || event.Subtype != "success")
		out.started = event.Type == "assistant"
		if out.complete {
			out.output = event.Result
		}
	case "gennadiy":
		// `run --format json` stamps every event with the session; a step is
		// one model turn. There is no explicit success record: completion is
		// the last observed step_finish without a later error and exit 0.
		if gennadiySessionID.MatchString(event.SessionRef) {
			out.session = event.SessionRef
		}
		out.started = event.Type == "step_start"
		out.complete = event.Type == "step_finish"
		out.failed = event.Type == "error"
		if event.Type == "text" && event.Part.Type == "text" {
			out.output = event.Part.Text
		}
	}
	return out
}

package worker

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

const syntheticSession = "00000000-0000-4000-8000-000000000001"

func TestProviderExactSessionAndPermissions(t *testing.T) {
	for _, test := range []struct {
		provider, session string
		want              []string
	}{
		{"codex", "", []string{"exec", "--json", "-"}},
		{"codex", syntheticSession, []string{"exec", "resume", "--json", syntheticSession, "-"}},
		{"claude", "", []string{"--print", "--verbose", "--output-format", "stream-json"}},
		{"claude", syntheticSession, []string{"--print", "--verbose", "--output-format", "stream-json", "--resume", syntheticSession}},
	} {
		args, err := providerArgs(test.provider, test.session)
		if err != nil || !reflect.DeepEqual(args, test.want) {
			t.Fatalf("provider arguments mismatch: %v", err)
		}
	}
	for _, bad := range []string{"--last", "latest", "../session", "id\n--dangerously-bypass-approvals-and-sandbox"} {
		if _, err := providerArgs("codex", bad); err == nil {
			t.Fatal("accepted a non-exact session")
		}
	}
	if _, err := providerArgs("shell", ""); err == nil {
		t.Fatal("accepted unknown provider")
	}
}

func TestNativeEventsRequireObservedTurnAndExplicitSuccess(t *testing.T) {
	for _, test := range []struct {
		provider, line                     string
		session, started, complete, failed bool
	}{
		{"codex", `{"type":"thread.started","thread_id":"` + syntheticSession + `"}`, true, false, false, false},
		{"codex", `{"type":"turn.started"}`, false, true, false, false},
		{"codex", `{"type":"turn.completed"}`, false, false, true, false},
		{"codex", `{"type":"turn.failed"}`, false, false, false, true},
		{"claude", `{"type":"system","subtype":"init","session_id":"` + syntheticSession + `"}`, true, false, false, false},
		{"claude", `{"type":"assistant","session_id":"` + syntheticSession + `"}`, true, true, false, false},
		{"claude", `{"type":"result","subtype":"error_during_execution","is_error":true}`, false, false, false, true},
		{"claude", `{"type":"result","subtype":"success","is_error":false,"result":"done"}`, false, false, true, false},
		{"claude", `{"type":"result","subtype":"success","is_error":true}`, false, false, false, true},
		{"codex", `not json`, false, false, false, false},
	} {
		event := parseProviderEvent(test.provider, []byte(test.line))
		if (event.session != "") != test.session || event.started != test.started || event.complete != test.complete || event.failed != test.failed {
			t.Fatalf("unexpected event: %+v", event)
		}
	}
}

func TestCredentialEnvironmentIsNotInherited(t *testing.T) {
	t.Setenv("AGENT_BOARD_TOKEN", "synthetic-board-secret")
	t.Setenv("ACCORD_HOST_SECRET", "synthetic-host-proof")
	t.Setenv("ANTHROPIC_API_KEY", "synthetic-provider-auth")
	for _, value := range providerEnv() {
		if strings.HasPrefix(value, "AGENT_BOARD_") || strings.HasPrefix(value, "ACCORD_") {
			t.Fatal("Accord credential inherited")
		}
	}
	if !strings.Contains(strings.Join(providerEnv(), "\n"), "ANTHROPIC_API_KEY=synthetic-provider-auth") {
		t.Fatal("native provider auth was removed")
	}
}

func TestMonotonicLeaseIgnoresServerClockSkew(t *testing.T) {
	start := time.Now()
	for _, skew := range []time.Duration{-time.Hour, 0, time.Hour} {
		// A server expiry is display metadata; only the fresh request's local
		// monotonic start determines when this worker must stop.
		_ = start.Add(skew + 30*time.Second)
		if got := pollDeadline(start).Sub(start); got != 25*time.Second {
			t.Fatalf("deadline drift: %v", got)
		}
	}
}

func TestMCPDiscoveryParsesNamesWithoutReplayingSecrets(t *testing.T) {
	data := []byte(`[{"name":"accord_gui","transport":{"env":{"TOKEN":"synthetic"}}},{"name":"other.tool"}]`)
	names, err := parseMCPNames(data)
	if err != nil || !reflect.DeepEqual(names, []string{"accord_gui", "other.tool"}) {
		t.Fatal("invalid name discovery")
	}
	if _, err = parseMCPNames([]byte(`[{"name":""}]`)); err == nil {
		t.Fatal("accepted empty connector")
	}
	if _, err = parseMCPNames(json.RawMessage(`null`)); err == nil {
		t.Fatal("accepted missing connector inventory")
	}
}

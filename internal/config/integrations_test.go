package config

import (
	"agent-board/internal/notifications"
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func githubConfig(repositories []string, events []string) map[string]any {
	g := map[string]any{"webhook_secret_env": "FIXTURE_GITHUB_SECRET", "task_key_pattern": "T-[0-9]+", "require_approval_for_merge": true, "events": events}
	if repositories != nil {
		g["repository_allowlist"] = repositories
	}
	return g
}

func TestGitHubExplicitTrustAndCapabilities(t *testing.T) {
	for _, events := range [][]string{{}, {"pull_request"}} {
		for _, repos := range [][]string{nil, {}, {"github.com/Owner/Repo"}} {
			for _, enabled := range []bool{false, true} {
				m := configMap(t)
				// Existing D metadata must never authorize an event source.
				field(m, "project")["repository"] = "github.com/owner/repo"
				m["github"] = githubConfig(repos, events)
				s, r := parseMap(t, m, Options{GitHubEvents: enabled})
				want := len(events) == 0 || (enabled && len(repos) > 0)
				if r.Valid != want || (s != nil) != want {
					t.Fatalf("events=%v repos=%v enabled=%v: %+v", events, repos, enabled, r)
				}
				if want && s.Decode().Project.Repository != "github.com/owner/repo" {
					t.Fatal("D repository metadata was lost")
				}
				if len(events) > 0 && len(repos) == 0 {
					found := false
					for _, err := range r.Errors {
						found = found || (err.Path == "github.repository_allowlist" && err.Code == "CONFIG_INVALID")
					}
					if !found {
						t.Fatal("missing explicit-trust diagnostic", r)
					}
				}
			}
		}
	}
}

func TestGitHubRepositoryBoundsAndNormalization(t *testing.T) {
	for _, value := range []string{"owner/repo", "https://github.com/owner/repo", "github.com/owner/repo.git", "github.com/owner/repo.GIT", "github.com/owner/repo?q=1", "github.com/owner/repo#x", "github.com:443/owner/repo", "github.com/owner/рr", "github.com/owner/repo/", "github.com/owner/repo\x00", "github.com/aa/" + strings.Repeat("r", 243)} {
		m := configMap(t)
		m["github"] = githubConfig([]string{value}, []string{})
		if _, r := parseMap(t, m, Options{GitHubEvents: true}); r.Valid {
			t.Fatalf("invalid repository accepted: %q", value)
		}
	}
	for _, repos := range [][]string{{"github.com/Owner/Repo", "github.com/owner/repo"}, {"github.com/owner/repo", "github.com/owner/repo"}} {
		m := configMap(t)
		m["github"] = githubConfig(repos, []string{})
		if _, r := parseMap(t, m, Options{}); r.Valid {
			t.Fatal("duplicate accepted", repos)
		}
	}
	m := configMap(t)
	repos := []string{"github.com/AA/" + strings.Repeat("R", 242)}
	for i := 1; i < 17; i++ {
		repos = append(repos, fmt.Sprintf("github.com/owner/repo%d", i))
	}
	m["github"] = githubConfig(repos, []string{})
	if _, r := parseMap(t, m, Options{}); r.Valid {
		t.Fatal("17 repositories accepted")
	}
	m["github"] = githubConfig(repos[:16], []string{"pull_request"})
	s, r := parseMap(t, m, Options{GitHubEvents: true})
	if !r.Valid || len(s.Decode().GitHub.RepositoryAllowlist[0]) != 256 || s.Decode().GitHub.RepositoryAllowlist[0] != strings.ToLower(repos[0]) {
		t.Fatal("16 repository / 256 byte boundary", r)
	}
	copy := s.Decode()
	copy.GitHub.RepositoryAllowlist[0] = "github.com/other/project"
	copy.GitHub.Events[0] = "check_suite"
	again, result := Parse(s.CanonicalJSON(), Options{GitHubEvents: true})
	if !result.Valid || again.SHA256() != s.SHA256() || s.Decode().GitHub.Events[0] != "pull_request" {
		t.Fatal("snapshot aliasing or canonical drift", result)
	}
}

func notificationConfig(url, env string, events ...string) map[string]any {
	return map[string]any{"url": url, "secret_env": env, "events": events}
}

func TestNotificationEndpointPolicy(t *testing.T) {
	for _, url := range []string{"http://example.invalid/", "https://Example.invalid/", "https://example.invalid:444/", "https://127.0.0.1/", "https://[::1]/", "https://localhost/", "https://example.invalid/?private=value", "https://example.invalid/?", "https://user:password@example.invalid/", "https://example.invalid/#fragment", "https://éxample.invalid/"} {
		m := configMap(t)
		m["notifications"] = map[string]any{"webhooks": []any{notificationConfig(url, "FIXTURE_SECRET", "epic_done")}}
		_, r := parseMap(t, m, Options{Notifications: true})
		if r.Valid {
			t.Fatalf("destination accepted: %s", url)
		}
		raw, _ := json.Marshal(r)
		if bytes.Contains(raw, []byte(url)) || bytes.Contains(raw, []byte("password")) || bytes.Contains(raw, []byte("private=value")) {
			t.Fatal("destination exposed in diagnostics")
		}
	}
}

func TestNotificationCanonicalIdentity(t *testing.T) {
	for _, pair := range [][2]string{{"https://example.invalid", "https://example.invalid/"}, {"https://example.invalid:443/", "https://example.invalid"}, {"https://example.invalid:443/notify", "https://example.invalid/notify"}} {
		m := configMap(t)
		m["notifications"] = map[string]any{"webhooks": []any{notificationConfig(pair[0], "FIXTURE_SECRET", "epic_done"), notificationConfig(pair[1], "FIXTURE_SECRET", "escalated")}}
		if _, r := parseMap(t, m, Options{Notifications: true}); r.Valid {
			t.Fatal("equivalent endpoints bypassed duplicate check", pair)
		}
	}
	m := configMap(t)
	m["notifications"] = map[string]any{"webhooks": []any{notificationConfig("https://example.invalid:443", "FIXTURE_ONE", "epic_done"), notificationConfig("https://example.invalid/", "FIXTURE_TWO", "epic_done"), notificationConfig("https://example.invalid/path", "FIXTURE_ONE", "epic_done")}}
	s, r := parseMap(t, m, Options{Notifications: true})
	if !r.Valid || s.Decode().Notifications.Webhooks[0].URL != "https://example.invalid/" {
		t.Fatal(r)
	}
	copy := s.Decode()
	copy.Notifications.Webhooks[0].Events[0] = "escalated"
	copy.Notifications.Webhooks[0].URL = "https://other.invalid/"
	if s.Decode().Notifications.Webhooks[0].Events[0] != "epic_done" || s.Decode().Notifications.Webhooks[0].URL != "https://example.invalid/" {
		t.Fatal("notification snapshot aliasing")
	}
	again, result := Parse(s.CanonicalJSON(), Options{Notifications: true})
	if !result.Valid || again.SHA256() != s.SHA256() {
		t.Fatal("notification canonical round trip", result)
	}
}

func TestNotificationEventsCapabilitiesAndOfflinePrivacy(t *testing.T) {
	events := []string{"task_created_for_human", "escalated", "lease_expired", "epic_done"}
	for _, event := range events {
		if _, err := notifications.Encode(notifications.Payload{DeliveryID: "fixture", EventCursor: 1, Event: event, TS: time.Unix(1, 0)}); err != nil {
			t.Fatal("config/encoder event drift", event, err)
		}
	}
	m := configMap(t)
	m["notifications"] = map[string]any{"webhooks": []any{notificationConfig("https://no-dns-required.invalid", "FIXTURE_SECRET", events...)}}
	if _, r := parseMap(t, m, Options{}); r.Valid || r.Errors[0].Code != "CONFIG_UNSUPPORTED" {
		t.Fatal("disabled capability silently accepted", r)
	}
	s, r := parseMap(t, m, Options{Notifications: true})
	if !r.Valid {
		t.Fatal(r)
	}
	// A short synthetic value proves offline validation neither reads env nor
	// checks runtime secret readiness. No actual credential is used.
	t.Setenv("FIXTURE_SECRET", "fixture-private-value")
	again, result := parseMap(t, m, Options{Notifications: true})
	if !result.Valid || s.SHA256() != again.SHA256() || bytes.Contains(again.CanonicalJSON(), []byte("fixture-private-value")) {
		t.Fatal("environment expanded", result)
	}
	for _, unsupported := range [][]string{{"unknown"}, {"epic_done", "epic_done"}, {"task_done"}} {
		m["notifications"] = map[string]any{"webhooks": []any{notificationConfig("https://example.invalid/", "FIXTURE_SECRET", unsupported...)}}
		if _, r := parseMap(t, m, Options{Notifications: true}); r.Valid {
			t.Fatal("invalid notification events accepted", unsupported)
		}
	}
}

func TestNotificationCountAndOldTemplate(t *testing.T) {
	m := configMap(t)
	webhooks := []any{}
	for i := 0; i < 17; i++ {
		webhooks = append(webhooks, notificationConfig(fmt.Sprintf("https://example.invalid/%d", i), "FIXTURE_SECRET", "epic_done"))
	}
	m["notifications"] = map[string]any{"webhooks": webhooks}
	if _, r := parseMap(t, m, Options{Notifications: true}); r.Valid {
		t.Fatal("17 webhooks accepted")
	}
	field(m, "notifications")["webhooks"] = webhooks[:16]
	if _, r := parseMap(t, m, Options{Notifications: true}); !r.Valid {
		t.Fatal("16 webhooks rejected", r)
	}
	before, r := Parse(template(), Options{})
	after, result := Parse(template(), Options{GitHubEvents: true, Notifications: true})
	if !r.Valid || !result.Valid || before.SHA256() != after.SHA256() || after.Decode().GitHub != nil || after.Decode().Notifications != nil {
		t.Fatal("old template acquired integration behavior", r, result)
	}
	m = configMap(t)
	m["notifications"] = map[string]any{"webhooks": []any{}}
	if _, r := parseMap(t, m, Options{}); !r.Valid {
		t.Fatal("disabled empty notifications rejected", r)
	}
	field(m, "auth")["mode"] = "github"
	if _, r := parseMap(t, m, Options{GitHubEvents: true, Notifications: true}); r.Valid {
		t.Fatal("event/notification capabilities enabled OAuth")
	}
}

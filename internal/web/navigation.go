package web

import (
	"net/http"
	"net/url"
	"strings"

	"agent-board/internal/core"
)

// BrowserPageDestination maps only known page addresses to the single Next UI.
// It does not authenticate, create a session, issue a nonce, or read board data.
// JSON readers and binary/data routes continue through their existing handlers.
func BrowserPageDestination(r *http.Request) (string, bool) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead || WantsBrowserJSON(r) {
		return "", false
	}
	path, scope := r.URL.Path, ""
	if path == "/" {
		return "/app", true
	}
	if strings.HasPrefix(path, "/web/p/") {
		parts := strings.SplitN(strings.TrimPrefix(path, "/web/p/"), "/", 2)
		if !validNavigationProjectID(parts[0]) {
			return "", false
		}
		tail := ""
		if len(parts) == 2 {
			tail = parts[1]
		}
		scope, path = "/p/"+parts[0], "/web/"+tail
	}
	tail := strings.TrimPrefix(path, "/web")
	if path == "/web/login" {
		tail, scope = "/login", ""
	} else if path == "/web/" || path == "/web/overview" {
		tail = ""
	} else if path == "/web/catalog/admin" {
		// The administration page is catalog scope only (docs/admin.md).
		if scope != "" {
			return "", false
		}
		tail = "/admin"
	} else {
		switch path {
		case "/web/tasks", "/web/questions", "/web/agents", "/web/team", "/web/activity", "/web/projects", "/web/settings", "/web/stats", "/web/history", "/web/designs", "/web/profiles", "/web/managed", "/web/automation", "/web/automation/host-scope":
		default:
			valid := false
			for _, prefix := range []string{"/web/tasks/", "/web/agents/", "/web/profiles/", "/web/managed/runs/", "/web/automation/workflows/", "/web/automation/policies/", "/web/catalog/projects/"} {
				if id, ok := strings.CutPrefix(path, prefix); ok && validNavigationSegment(id) {
					valid = true
					switch prefix {
					case "/web/tasks/":
						valid = core.ValidControlTaskKey(id) || core.ValidControlEpicKey(id)
					case "/web/agents/":
						valid = core.ValidControlSessionID(id)
					case "/web/profiles/":
						valid = validNavigationOpaqueID(id, "ap_")
					case "/web/catalog/projects/":
						valid = scope == "" && validNavigationProjectID(id)
						tail = "/projects/" + id
					}
					break
				}
			}
			if !valid {
				return "", false
			}
		}
	}
	// Encode the fixed relative destination and preserve only the original query.
	// Decoded slashes, dot segments and backslashes cannot become destination paths.
	target := &url.URL{Path: "/app" + scope + tail, RawQuery: r.URL.RawQuery}
	return target.String(), true
}

func validNavigationSegment(value string) bool {
	if value == "" || len(value) > 200 {
		return false
	}
	for _, c := range value {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func validNavigationProjectID(value string) bool { return validNavigationOpaqueID(value, "p_") }
func validNavigationOpaqueID(value, prefix string) bool {
	if len(value) != len(prefix)+32 || !strings.HasPrefix(value, prefix) {
		return false
	}
	for _, c := range value[len(prefix):] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

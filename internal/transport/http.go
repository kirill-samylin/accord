package transport

import (
	"context"
	"encoding/json"
	"io"
	"mime"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"agent-board/internal/core"
)

const MaxBodyBytes int64 = 64 << 10

type route struct {
	method, path, name string
	body               bool
}

var routes = []route{
	{"GET", "/api/v1/tasks/{key}/coordination", "coordination", false},
	{"POST", "/api/v1/tasks/{key}/assign", "assign", true},
	{"POST", "/api/v1/tasks/{key}/checks", "check_create", true},
	{"GET", "/api/v1/profiles", "profiles", false},
	{"GET", "/api/v1/profile", "profile", false},
	{"POST", "/api/v1/profile_create", "profile_create", true},
	{"POST", "/api/v1/profile_update", "profile_update", true},
	{"POST", "/api/v1/profile_archive", "profile_archive", true},
	{"POST", "/api/v1/session", "describe", true},
	{"POST", "/api/v1/tasks", "create", true},
	{"GET", "/api/v1/tasks/{key}", "get", false},
	{"POST", "/api/v1/tasks/next", "next", true},
	{"PATCH", "/api/v1/tasks/{key}", "update", true},
	{"POST", "/api/v1/tasks/{key}/transition", "transition", true},
	{"POST", "/api/v1/tasks/{key}/release", "release", true},
	{"POST", "/api/v1/tasks/{key}/comments", "comment", true},
	{"POST", "/api/v1/links", "link", true},
	{"POST", "/api/v1/heartbeat", "heartbeat", true},
	{"GET", "/api/v1/events", "events", false},
	{"GET", "/api/v1/tasks", "tasks", false},
	{"GET", "/api/v1/epics", "epics", false},
	{"GET", "/api/v1/sessions", "sessions", false},
	{"GET", "/api/v1/notes", "notes", false},
	{"POST", "/api/v1/notes", "note", true},
	{"GET", "/api/v1/stats", "stats", false},
}

// NewHandler authenticates every API request before dispatch, including MCP list
// and replay. Optional ready checks storage without opening a board session.
func NewHandler(s Service, ready func(context.Context) error) http.Handler {
	return NewHandlerWithOptions(s, ready, HandlerOptions{})
}

// NewHandlerWithOptions retains the native HTTP/MCP contract while allowing
// operators and isolated tests to lower the bounded admission limits.
func NewHandlerWithOptions(s Service, ready func(context.Context) error, options HandlerOptions) http.Handler {
	if ValidateExternalHost(options.ExternalHost) != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeError(w, core.Fail("INTERNAL", "invalid HTTP host policy"))
		})
	}
	admission := newAdmission(options)
	ops := operations(s)
	all := append(append([]operation(nil), ops...), readOperations(s)...)
	all = append(all, profileOperations(s)...)
	all = append(all, coordinationOperations(s)...)
	byName := make(map[string]operation, len(all))
	for _, op := range all {
		byName[op.name] = op
	}
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcpHandler(ops, s, options.ExternalHost != "", options.Designs))
	mux.HandleFunc("GET /metrics", admission.metrics)
	for _, r := range routes {
		op := byName[r.name]
		mux.HandleFunc(r.method+" "+r.path, func(w http.ResponseWriter, req *http.Request) {
			principal, ok := req.Context().Value(principalKey{}).(core.Principal)
			if !ok {
				writeError(w, core.Fail("UNAUTHORIZED", "authentication is required"))
				return
			}
			body, err := requestBody(w, req, r)
			if err != nil {
				writeError(w, err)
				return
			}
			value, err := op.run(req.Context(), principal, req.Header.Get("X-Session"), body)
			if err != nil {
				writeError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, value)
		})
	}
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		methods := methodsForPath(r.URL.Path)
		if len(methods) > 0 {
			w.Header().Set("Allow", strings.Join(methods, ", "))
			writeJSON(w, http.StatusMethodNotAllowed, errorResponse{core.Error{Code: "METHOD_NOT_ALLOWED", Message: "method is not supported for this route"}})
			return
		}
		writeError(w, core.Fail("NOT_FOUND", "route does not exist"))
	})
	protection := http.NewCrossOriginProtection().Handler(mux)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Never log panic values: arbitrary values can include caller secrets.
		defer func() {
			if recover() != nil {
				writeError(w, core.Fail("INTERNAL", "internal server error"))
			}
		}()
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		if nativePath(r.URL.Path) {
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		}
		if !admission.enter(w) {
			return
		}
		defer admission.leave()
		if !allowedHost(r, options.ExternalHost) {
			writeError(w, core.Fail("FORBIDDEN", "Host is not allowed"))
			return
		}
		if nativePath(r.URL.Path) && hasOrigin(r.Header) {
			writeError(w, core.Fail("FORBIDDEN", "browser Origin is not allowed on native endpoints"))
			return
		}
		if r.URL.Path == "/healthz" || r.URL.Path == "/readyz" {
			if r.Method != "GET" {
				w.Header().Set("Allow", "GET")
				writeJSON(w, http.StatusMethodNotAllowed, errorResponse{core.Error{Code: "METHOD_NOT_ALLOWED", Message: "method is not supported for this route"}})
				return
			}
			if r.URL.Path == "/readyz" && ready != nil {
				if err := ready(r.Context()); err != nil {
					writeJSON(w, http.StatusServiceUnavailable, errorResponse{core.Error{Code: "NOT_READY", Message: "storage unavailable"}})
					return
				}
			}
			writeJSON(w, http.StatusOK, struct {
				Status string `json:"status"`
			}{"ok"})
			return
		}
		if options.ProjectRouter != nil && r.URL.Path != "/metrics" {
			options.ProjectRouter.ServeHTTP(w, r)
			return
		}
		if options.Extra != nil && extraPath(r.URL.Path) {
			options.Extra.ServeHTTP(w, r)
			return
		}
		fields := strings.Fields(r.Header.Get("Authorization"))
		if len(fields) != 2 || !strings.EqualFold(fields[0], "Bearer") {
			unauthorized(w)
			return
		}
		principal, err := s.Authenticate(r.Context(), fields[1])
		if err != nil {
			unauthorized(w)
			return
		}
		protection.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), principalKey{}, principal)))
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	writeError(w, core.Fail("UNAUTHORIZED", "valid bearer token is required"))
}

func requestBody(w http.ResponseWriter, r *http.Request, route route) ([]byte, error) {
	if !route.body {
		if r.ContentLength > 0 || len(r.TransferEncoding) > 0 {
			return nil, core.Fail("VALIDATION", "GET body is not supported")
		}
		if route.name == "get" || route.name == "coordination" {
			if r.URL.RawQuery != "" {
				return nil, core.Fail("VALIDATION", "unexpected query parameters")
			}
			return json.Marshal(core.GetInput{TaskKey: r.PathValue("key")})
		}
		if route.name == "events" {
			return eventsQuery(r)
		}
		return readQuery(route.name, r.URL.RawQuery)
	}
	if (route.name == "assign" || route.name == "check_create") && r.URL.RawQuery != "" {
		return nil, core.Fail("validation", "unexpected query parameters")
	}
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return nil, core.Fail("VALIDATION", "Content-Type application/json is required")
	}
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBodyBytes))
	if err != nil {
		return nil, core.Fail("VALIDATION", "request body exceeds limit or cannot be read")
	}
	if key := r.PathValue("key"); key != "" {
		var fields map[string]json.RawMessage
		if err := decodeObject(data, &fields); err != nil {
			return nil, err
		}
		field := "task_key"
		if route.name == "check_create" {
			field = "target_key"
		}
		if raw, exists := fields[field]; exists {
			var provided string
			if json.Unmarshal(raw, &provided) != nil || provided != key {
				return nil, core.Fail("VALIDATION", "task_key must match route")
			}
		}
		fields[field], _ = json.Marshal(key)
		return json.Marshal(fields)
	}
	return data, nil
}

func eventsQuery(r *http.Request) ([]byte, error) {
	query, parseErr := url.ParseQuery(r.URL.RawQuery)
	if parseErr != nil {
		return nil, core.Fail("VALIDATION", "malformed query parameters")
	}
	for key, values := range query {
		if len(values) != 1 {
			return nil, core.Fail("VALIDATION", "query parameters must not repeat")
		}
		switch key {
		case "since_cursor", "limit", "types", "task_key":
		default:
			return nil, core.Fail("VALIDATION", "unknown query parameter")
		}
	}
	var in core.EventsInput
	var err error
	if raw := query.Get("since_cursor"); raw != "" {
		in.SinceCursor, err = strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return nil, core.Fail("VALIDATION", "invalid cursor")
		}
	}
	if raw := query.Get("limit"); raw != "" {
		in.Limit, err = strconv.Atoi(raw)
		if err != nil {
			return nil, core.Fail("VALIDATION", "invalid limit")
		}
	}
	if raw := query.Get("types"); raw != "" {
		in.Types = strings.Split(raw, ",")
	}
	in.TaskKey = query.Get("task_key")
	return json.Marshal(in)
}

func writeError(w http.ResponseWriter, err error) {
	body, status := publicError(err)
	writeJSON(w, status, body)
}
func writeJSON(w http.ResponseWriter, status int, body any) {
	data, err := json.Marshal(body)
	if err != nil {
		status = http.StatusInternalServerError
		data = []byte(`{"error":{"code":"INTERNAL","message":"internal server error"}}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(append(data, '\n'))
}

// ValidateListen requires a literal loopback address; remote access belongs to
// a separately configured TLS reverse proxy, never an unprotected public bind.
func ValidateListen(address string) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return core.Fail("VALIDATION", "listen requires loopback IP and port")
	}
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return core.Fail("VALIDATION", "listen must use a literal loopback IP")
	}
	return nil
}

func methodsForPath(path string) []string {
	if path == "/metrics" {
		return []string{"GET"}
	}
	var methods []string
	actual := strings.Split(path, "/")
	for _, route := range routes {
		expected := strings.Split(route.path, "/")
		if len(actual) != len(expected) {
			continue
		}
		matches := true
		for i, part := range expected {
			if part == "{key}" && actual[i] != "" {
				continue
			}
			if part != actual[i] {
				matches = false
				break
			}
		}
		if matches {
			methods = append(methods, route.method)
		}
	}
	return methods
}

// Extra never intercepts native task routes, metrics or MCP authentication.
func extraPath(path string) bool {
	return !nativePath(path) || path == "/api/v1/designs" || strings.HasPrefix(path, "/api/v1/designs/")
}

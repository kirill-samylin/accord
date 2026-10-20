package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

func dashboard(t *testing.T, f *webFixture, cookie *http.Cookie) DashboardSnapshot {
	t.Helper()
	w := request(f.s, "GET", "/web/data/dashboard", nil, cookie)
	if w.Code != 200 || w.Header().Get("Content-Type") != "application/json; charset=utf-8" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("dashboard %d %s", w.Code, w.Body)
	}
	var snapshot DashboardSnapshot
	if err := json.Unmarshal(w.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	return snapshot
}

func controlAgent(t *testing.T, f *webFixture, token core.TokenView, id string) core.Actor {
	t.Helper()
	p, err := f.board.Authenticate(context.Background(), token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	session, err := f.board.Describe(context.Background(), p, core.OpenSession{Role: "analyst", Machine: p.Machine, RunID: id, RequestID: id})
	if err != nil {
		t.Fatal(err)
	}
	return core.Actor{Principal: p, SessionID: session.SessionID}
}

func TestDashboardBoundedActualCollectionsAndRevocation(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "operator")
	actor := f.native(t, token)
	epic, err := f.board.Create(context.Background(), actor, core.CreateInput{RequestID: "epic", Type: "epic", Title: "Project"})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		controlAgent(t, f, token, "agent-"+strconv.Itoa(i))
	}
	workerToken := f.token(t, "worker")
	worker := controlAgent(t, f, workerToken, "worker")
	task, err := f.board.Create(context.Background(), actor, core.CreateInput{RequestID: "task", Type: "task", Role: "analyst", EpicKey: epic.Key, Title: "Work", Description: "Context"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.board.Next(context.Background(), worker, core.NextInput{RequestID: "claim"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 22; i++ {
		if _, err = f.board.Create(context.Background(), actor, core.CreateInput{RequestID: "event-" + strconv.Itoa(i), Type: "epic", Title: "Event"}); err != nil {
			t.Fatal(err)
		}
	}
	cookie := f.login(t, token)
	first := dashboard(t, f, cookie)
	if first.Overview.TasksTotal != 1 || first.Overview.ActiveAgentSessions != 9 || first.Overview.LeasedTasks != 1 || len(first.Agents.Sessions) != 6 || first.Agents.NextCursor == "" || len(first.Activity.Events) != 20 || !first.Activity.HasMore {
		t.Fatalf("bounded snapshot %+v", first)
	}
	for _, agent := range first.Agents.Sessions {
		if agent.Role == "human" || agent.Status != "active" {
			t.Fatalf("unexpected dashboard agent %+v", agent)
		}
	}
	for i, event := range first.Activity.Events {
		if event.TokenID != "" || i > 0 && event.Cursor >= first.Activity.Events[i-1].Cursor {
			t.Fatalf("activity projection/order %+v", first.Activity)
		}
	}
	ssr := f.page(t, cookie, "/web/")
	if ssr.Overview.TasksTotal != first.Overview.TasksTotal || len(ssr.Agents) != len(first.Agents.Sessions) || len(ssr.Activity) != len(first.Activity.Events) {
		t.Fatal("SSR and dashboard use different collection bounds")
	}
	if err = f.board.RevokeToken(context.Background(), workerToken.ID); err != nil {
		t.Fatal(err)
	}
	after := dashboard(t, f, cookie)
	if after.Overview.ActiveAgentSessions != 8 || after.Overview.LeasedTasks != 0 || after.Overview.TasksInProgress != 1 {
		t.Fatalf("revoked worker still effective %+v", after.Overview)
	}
	detail := f.page(t, cookie, "/web/agents/"+worker.SessionID)
	if detail.Agent.Status != "credential_invalid" || detail.Agent.Task != nil {
		t.Fatal("revoked credential retained effective lease")
	}
	if got, err := f.board.Get(context.Background(), actor, core.GetInput{TaskKey: task.Key}); err != nil || got.State != "in_progress" {
		t.Fatal("read rewrote revoked task workflow")
	}
}

func TestDashboardReadDoesNotRenewSessionLeaseOrCreateActions(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "operator")
	task := f.question(t, token)
	cookie := f.login(t, token)
	p := f.page(t, cookie, "/web/tasks/"+task.Key)
	if w := f.post(cookie, "/web/tasks/"+task.Key+"/claim", p, "claim", nil); w.Code != 303 {
		t.Fatal(w.Code)
	}
	b, err := f.s.loadBrowser(newBrowserRequest(cookie))
	if err != nil {
		t.Fatal(err)
	}
	before, err := f.board.Get(context.Background(), b.actor(), core.GetInput{TaskKey: task.Key})
	if err != nil {
		t.Fatal(err)
	}
	var actions, afterActions int
	if err = f.s.db.QueryRow(`SELECT count(*) FROM web_actions`).Scan(&actions); err != nil {
		t.Fatal(err)
	}
	first := dashboard(t, f, cookie)
	second := dashboard(t, f, cookie)
	if second.Overview.LatestEventCursor != first.Overview.LatestEventCursor {
		t.Fatal("poll emitted events")
	}
	if err = f.s.db.QueryRow(`SELECT count(*) FROM web_actions`).Scan(&afterActions); err != nil {
		t.Fatal(err)
	}
	after, err := f.board.Get(context.Background(), b.actor(), core.GetInput{TaskKey: task.Key})
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := f.s.loadBrowser(newBrowserRequest(cookie))
	if err != nil {
		t.Fatal(err)
	}
	if actions != afterActions || !before.Lease.ExpiresAt.Equal(after.Lease.ExpiresAt) || before.Generation != after.Generation || !loaded.Session.ExpiresAt.Equal(b.Session.ExpiresAt) || loaded.Session.SessionID != b.Session.SessionID {
		t.Fatal("poll changed actions, lease or session")
	}
	f.now = f.now.Add(2 * time.Minute)
	w := request(f.s, "GET", "/web/data/dashboard", nil, cookie)
	if w.Code != 409 || !strings.Contains(w.Body.String(), `"code":"LEASE_LOST"`) {
		t.Fatalf("board expiry %d %s", w.Code, w.Body)
	}
}

func TestDashboardBrowserBoundaryMethodsAndQueries(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "operator")
	cookie := f.login(t, token)
	for _, query := range []string{"?q=a", "?kind=agent", "?q=a&q=b", "?x=1&x=2", "?=x", "?q=%XX", "?x=1;y=2"} {
		w := request(f.s, "GET", "/web/data/dashboard"+query, nil, cookie)
		if w.Code != 400 || !strings.Contains(w.Body.String(), `"code":"VALIDATION"`) {
			t.Fatalf("query %s: %d %s", query, w.Code, w.Body)
		}
	}
	for _, method := range []string{"POST", "PUT", "PATCH", "DELETE", "OPTIONS"} {
		w := request(f.s, method, "/web/data/dashboard", nil, cookie)
		if w.Code != 405 || w.Header().Get("Allow") != "GET, HEAD" || !strings.HasPrefix(w.Header().Get("Content-Type"), "application/json") {
			t.Fatalf("method %s: %d %s", method, w.Code, w.Body)
		}
	}
	for _, authenticated := range []bool{false, true} {
		var c *http.Cookie
		status := http.StatusUnauthorized
		if authenticated {
			c, status = cookie, http.StatusOK
		}
		w := request(f.s, "HEAD", "/web/data/dashboard", nil, c)
		if w.Code != status || w.Body.Len() != 0 {
			t.Fatalf("HEAD auth=%t: %d %s", authenticated, w.Code, w.Body)
		}
	}
	r := httptest.NewRequest("GET", "https://accord.test/web/data/dashboard", nil)
	r.Header.Set("Authorization", "Bearer "+token.Secret)
	w := httptest.NewRecorder()
	f.s.ServeHTTP(w, r)
	if w.Code != 401 || strings.Contains(w.Body.String(), "<html") || strings.Contains(w.Body.String(), token.Secret) {
		t.Fatal("native credential authorized browser data")
	}
	if err := f.board.RevokeToken(context.Background(), token.ID); err != nil {
		t.Fatal(err)
	}
	w = request(f.s, "GET", "/web/data/dashboard", nil, cookie)
	if w.Code != 401 || !strings.Contains(w.Body.String(), `"code":"UNAUTHORIZED"`) {
		t.Fatalf("revocation %d %s", w.Code, w.Body)
	}
}

package web

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestExactQuestionRaceAnswerCrashReplayAndFencing(t *testing.T) {
	f := fixture(t)
	one, two := f.token(t, "one"), f.token(t, "two")
	task := f.question(t, one)
	c1, c2 := f.login(t, one), f.login(t, two)
	p1, p2 := f.page(t, c1, "/web/tasks/"+task.Key), f.page(t, c2, "/web/tasks/"+task.Key)
	codes := make([]int, 2)
	var wg sync.WaitGroup
	for i, v := range []struct {
		c *http.Cookie
		p Page
	}{{c1, p1}, {c2, p2}} {
		wg.Add(1)
		go func(i int, v struct {
			c *http.Cookie
			p Page
		}) {
			defer wg.Done()
			codes[i] = f.post(v.c, "/web/tasks/"+task.Key+"/claim", v.p, "claim", nil).Code
		}(i, v)
	}
	wg.Wait()
	var winner, loser *http.Cookie
	if codes[0] == 303 && codes[1] == 409 {
		winner, loser = c1, c2
	} else if codes[1] == 303 && codes[0] == 409 {
		winner, loser = c2, c1
	} else {
		t.Fatalf("claim race %#v", codes)
	}
	p := f.page(t, winner, "/web/tasks/"+task.Key)
	if !p.CanEdit || p.Task == nil {
		t.Fatal("winner cannot edit")
	}
	observer := f.page(t, loser, "/web/tasks/"+task.Key)
	if observer.CanEdit {
		t.Fatal("observer can edit")
	}
	fields := url.Values{"generation": {strconv.FormatInt(p.Task.Generation, 10)}, "answer": {"<script>answer</script>"}}
	w := f.post(winner, "/web/tasks/"+task.Key+"/answer", p, "answer", fields)
	if w.Code != 303 {
		t.Fatalf("answer %d %s", w.Code, w.Body)
	}
	id := p.Actions["answer"]
	if _, err := f.s.db.Exec(`UPDATE web_actions SET result='' WHERE id=?`, id); err != nil {
		t.Fatal(err)
	}
	// Simulate process death between board commit and web result commit, then
	// expire the original board session. Exact native replay must still succeed.
	if err := f.s.Close(); err != nil {
		t.Fatal(err)
	}
	var reopenErr error
	f.s, reopenErr = New(f.board, Options{Origin: "https://accord.test", DataDir: f.dir + "/web", Renderer: f.render})
	if reopenErr != nil {
		t.Fatal(reopenErr)
	}
	f.now = f.now.Add(2 * time.Minute)
	w = f.post(winner, "/web/tasks/"+task.Key+"/answer", p, "answer", fields)
	if w.Code != 303 {
		t.Fatalf("expired exact replay %d %s", w.Code, w.Body)
	}
	actor := f.native(t, one)
	events, err := f.board.ListDecisions(context.Background(), actor, core.DecisionsInput{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(events.Events) != 1 || events.Events[0].Data.Answer == nil || *events.Events[0].Data.Answer != "<script>answer</script>" {
		t.Fatalf("duplicate/missing immutable decision %#v", events)
	}
	// A different payload under that saved action cannot replace the answer.
	fields.Set("answer", "changed")
	w = f.post(winner, "/web/tasks/"+task.Key+"/answer", p, "answer", fields)
	if w.Code != 409 {
		t.Fatalf("changed replay %d", w.Code)
	}
}
func TestExplicitHeartbeatReleaseAndStaleDraft(t *testing.T) {
	f := fixture(t)
	one, two := f.token(t, "one"), f.token(t, "two")
	task := f.question(t, one)
	c1, c2 := f.login(t, one), f.login(t, two)
	p := f.page(t, c1, "/web/tasks/"+task.Key)
	if w := f.post(c1, "/web/tasks/"+task.Key+"/claim", p, "claim", nil); w.Code != 303 {
		t.Fatalf("claim %d", w.Code)
	}
	owned := f.page(t, c1, "/web/tasks/"+task.Key)
	generation := strconv.FormatInt(owned.Task.Generation, 10)
	heartbeat := url.Values{"task_key": {task.Key}, "generation": {generation}}
	w := f.post(c1, "/web/heartbeat", owned, "heartbeat", heartbeat)
	if w.Code != 200 {
		t.Fatalf("heartbeat %d %s", w.Code, w.Body)
	}
	var result actionResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.NextActionID == "" || result.Session == nil || len(result.Session.MyTasks) != 1 || result.Session.MyTasks[0].Lease == nil {
		t.Fatal("heartbeat missing next action/session")
	}
	release := url.Values{"generation": {generation}, "reason": {"Need another owner"}}
	if w = f.post(c1, "/web/tasks/"+task.Key+"/release", owned, "release", release); w.Code != 303 {
		t.Fatalf("release %d %s", w.Code, w.Body)
	}
	other := f.page(t, c2, "/web/tasks/"+task.Key)
	if w = f.post(c2, "/web/tasks/"+task.Key+"/claim", other, "claim", nil); w.Code != 303 {
		t.Fatalf("reclaim %d", w.Code)
	}
	stale := url.Values{"generation": {generation}, "answer": {"Saved draft after lost ownership"}}
	w = f.post(c1, "/web/tasks/"+task.Key+"/answer", owned, "answer", stale)
	if w.Code != 409 {
		t.Fatalf("stale answer %d", w.Code)
	}
	draft := f.render.last()
	if draft.CanEdit || draft.Draft["answer"] != stale.Get("answer") {
		t.Fatal("stale draft lost or editor reacquired")
	}
	w = f.post(c1, "/web/heartbeat", owned, "heartbeat", heartbeat)
	if w.Code != 200 {
		t.Fatalf("exact old heartbeat replay %d", w.Code)
	}
	// The next fresh tick, unlike exact replay, must acknowledge actual ownership.
	heartbeat.Set("action_id", result.NextActionID)
	heartbeat.Set("csrf", owned.CSRF)
	w = request(f.s, "POST", "/web/heartbeat", heartbeat, c1, "https://accord.test")
	if w.Code != 409 {
		t.Fatalf("stale fresh heartbeat %d %s", w.Code, w.Body)
	}
}
func TestGetDoesNotDescribeOrRenewAndContinueIsExplicit(t *testing.T) {
	f := fixture(t)
	token := f.token(t, "one")
	task := f.question(t, token)
	c := f.login(t, token)
	p := f.page(t, c, "/web/tasks/"+task.Key)
	if w := f.post(c, "/web/tasks/"+task.Key+"/claim", p, "claim", nil); w.Code != 303 {
		t.Fatal(w.Code)
	}
	owned := f.page(t, c, "/web/tasks/"+task.Key)
	expires := owned.Task.Lease.ExpiresAt
	session := owned.Session
	for i := 0; i < 3; i++ {
		p = f.page(t, c, "/web/tasks/"+task.Key)
		if !p.Task.Lease.ExpiresAt.Equal(expires) {
			t.Fatal("GET renewed lease")
		}
	}
	// Expiring the local native session must not log out the browser or silently
	// create another native session on the next GET.
	b, err := f.s.loadBrowser(newBrowserRequest(c))
	if err != nil {
		t.Fatal(err)
	}
	b.Session.ExpiresAt = time.Now().Add(-time.Second)
	if err = f.s.saveSession(context.Background(), b); err != nil {
		t.Fatal(err)
	}
	p = f.page(t, c, "/web/")
	if !p.CanContinue || p.Session.SessionID != session.SessionID {
		t.Fatal("GET silently continued session")
	}
	w := f.post(c, "/web/session/continue", p, "continue", nil)
	if w.Code != 303 {
		t.Fatalf("continue %d %s", w.Code, w.Body)
	}
	fresh := f.page(t, c, "/web/")
	if fresh.Session.SessionID == session.SessionID {
		t.Fatal("explicit continuation reused expired session")
	}
	// Repeating the same continuation does not open a third native session.
	if w = f.post(c, "/web/session/continue", p, "continue", nil); w.Code != 303 {
		t.Fatal(w.Code)
	}
	again := f.page(t, c, "/web/")
	if again.Session.SessionID != fresh.Session.SessionID {
		t.Fatal("continue replay opened duplicate session")
	}
}

func newBrowserRequest(c *http.Cookie) *http.Request {
	r := httptest.NewRequest("GET", "https://accord.test/web/", nil)
	r.AddCookie(c)
	return r
}

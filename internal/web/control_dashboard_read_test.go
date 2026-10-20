package web

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

type dashboardReadBoard struct {
	Board
	fail  string
	calls []string
	actor core.Actor
	base  time.Time
}

func (b *dashboardReadBoard) read(name string, actor core.Actor) error {
	b.calls = append(b.calls, name)
	b.actor = actor
	if b.fail == name {
		return errors.New("private failure details")
	}
	return nil
}

func (b *dashboardReadBoard) ControlOverview(_ context.Context, actor core.Actor) (core.ControlOverview, error) {
	err := b.read("overview", actor)
	return core.ControlOverview{ObservedAt: b.base, Project: "fixture"}, err
}

func (b *dashboardReadBoard) ControlSessions(_ context.Context, actor core.Actor, in core.ControlSessionsInput) (core.ControlSessionsPage, error) {
	if in != (core.ControlSessionsInput{Kind: "agent", State: "active", Limit: 6}) {
		panic("unexpected dashboard agent bounds")
	}
	err := b.read("agents", actor)
	return core.ControlSessionsPage{ObservedAt: b.base.Add(time.Second), Sessions: []core.ControlSession{}}, err
}

func (b *dashboardReadBoard) RecentEvents(_ context.Context, actor core.Actor, in core.RecentEventsInput) (core.RecentEventsPage, error) {
	if in != (core.RecentEventsInput{Limit: 20}) {
		panic("unexpected dashboard event bounds")
	}
	err := b.read("activity", actor)
	return core.RecentEventsPage{ObservedAt: b.base.Add(2 * time.Second), Events: []core.Event{}}, err
}

func TestDashboardPreservesIndependentObservationTimes(t *testing.T) {
	f := fixture(t)
	cookie := f.login(t, f.token(t, "operator"))
	b, err := f.s.loadBrowser(newBrowserRequest(cookie))
	if err != nil {
		t.Fatal(err)
	}
	probe := &dashboardReadBoard{Board: f.board, base: f.now}
	f.s.board = probe
	snapshot := dashboard(t, f, cookie)
	if !snapshot.Overview.ObservedAt.Equal(probe.base) || !snapshot.Agents.ObservedAt.Equal(probe.base.Add(time.Second)) || !snapshot.Activity.ObservedAt.Equal(probe.base.Add(2*time.Second)) || !reflect.DeepEqual(probe.calls, []string{"overview", "agents", "activity"}) || probe.actor != b.actor() {
		t.Fatalf("dashboard reads %+v %+v", snapshot, probe)
	}
	probe.calls = nil
	page := f.page(t, cookie, "/web/")
	if !reflect.DeepEqual(probe.calls, []string{"overview", "agents", "activity"}) || !page.ObservedAt.Equal(snapshot.Overview.ObservedAt) || !page.AgentsObservedAt.Equal(snapshot.Agents.ObservedAt) || !page.ActivityObservedAt.Equal(snapshot.Activity.ObservedAt) {
		t.Fatal("SSR overview bypassed shared reader")
	}
	probe.calls = nil
	w := request(f.s, "GET", "/web/data/overview", nil, cookie)
	if w.Code != 200 || strings.Contains(w.Body.String(), `"overview"`) || !reflect.DeepEqual(probe.calls, []string{"overview"}) {
		t.Fatal("legacy overview contract changed")
	}
}

func TestDashboardFailsWholeResponseOnAnyCollectionFailure(t *testing.T) {
	for i, stage := range []string{"overview", "agents", "activity"} {
		t.Run(stage, func(t *testing.T) {
			f := fixture(t)
			cookie := f.login(t, f.token(t, "operator"))
			probe := &dashboardReadBoard{Board: f.board, base: f.now, fail: stage}
			f.s.board = probe
			w := request(f.s, "GET", "/web/data/dashboard", nil, cookie)
			if w.Code != http.StatusServiceUnavailable || !strings.Contains(w.Body.String(), `"code":"UNAVAILABLE"`) || strings.Contains(w.Body.String(), `"overview"`) || strings.Contains(w.Body.String(), "private failure") || len(probe.calls) != i+1 {
				t.Fatalf("partial/error data at %s: %d %s calls=%v", stage, w.Code, w.Body, probe.calls)
			}
			b, err := f.s.loadBrowser(newBrowserRequest(cookie))
			if err != nil {
				t.Fatal(err)
			}
			if partial, err := f.s.readDashboard(context.Background(), b.actor()); err == nil || !reflect.DeepEqual(partial, DashboardSnapshot{}) {
				t.Fatal("reader returned partial successful collections")
			}
		})
	}
}

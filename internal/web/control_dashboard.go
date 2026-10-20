package web

import (
	"context"

	"agent-board/internal/core"
)

// DashboardSnapshot contains bounded collections, each observed by its own
// authorized board read. It does not promise one transaction across collections.
type DashboardSnapshot struct {
	Overview core.ControlOverview     `json:"overview"`
	Agents   core.ControlSessionsPage `json:"agents"`
	Activity core.RecentEventsPage    `json:"activity"`
}

func (s *Server) readDashboard(ctx context.Context, actor core.Actor) (DashboardSnapshot, error) {
	var snapshot DashboardSnapshot
	var err error
	snapshot.Overview, err = s.board.ControlOverview(ctx, actor)
	if err != nil {
		return DashboardSnapshot{}, err
	}
	snapshot.Agents, err = s.board.ControlSessions(ctx, actor, core.ControlSessionsInput{Kind: "agent", State: "active", Limit: 6})
	if err != nil {
		return DashboardSnapshot{}, err
	}
	snapshot.Activity, err = s.board.RecentEvents(ctx, actor, core.RecentEventsInput{Limit: 20})
	if err != nil {
		return DashboardSnapshot{}, err
	}
	return snapshot, nil
}

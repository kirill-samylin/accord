package core

import "time"

// ControlOverview contains complete counts from one authoritative transaction.
// Active sessions and leases require a currently valid credential.
type ControlOverview struct {
	ObservedAt          time.Time `json:"observed_at"`
	Project             string    `json:"project"`
	TasksTotal          int       `json:"tasks_total"`
	TasksCreated        int       `json:"tasks_created"`
	TasksReady          int       `json:"tasks_ready"`
	TasksInProgress     int       `json:"tasks_in_progress"`
	TasksDone           int       `json:"tasks_done"`
	TasksCancelled      int       `json:"tasks_cancelled"`
	TasksBlocked        int       `json:"tasks_blocked"`
	OpenEpics           int       `json:"open_epics"`
	ActiveAgentSessions int       `json:"active_agent_sessions"`
	ActiveHumanSessions int       `json:"active_human_sessions"`
	LeasedTasks         int       `json:"leased_tasks"`
	LatestEventCursor   int64     `json:"latest_event_cursor"`
}

// ControlSession describes recorded participation, not an OS process.
// Task is present only for a live lease of the current task generation.
type ControlSession struct {
	Profile        *ProfileSnapshot `json:"profile,omitempty"`
	SessionID      string           `json:"session_id"`
	Role           string           `json:"role"`
	RunID          string           `json:"run_id"`
	Machine        string           `json:"machine"`
	Owner          string           `json:"owner"`
	RoleGeneration int64            `json:"role_generation"`
	ExpiresAt      time.Time        `json:"expires_at"`
	Status         string           `json:"status"`
	Task           *Task            `json:"task,omitempty"`
	LastEventAt    *time.Time       `json:"last_event_at,omitempty"`
}
type ControlSessionsInput struct {
	Cursor        string `json:"cursor,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
	Kind          string `json:"kind,omitempty"`
	Role          string `json:"role,omitempty"`
	Owner         string `json:"owner,omitempty"`
	State         string `json:"state,omitempty"`
	IncludeClosed bool   `json:"include_closed,omitempty"`
}
type ControlSessionsPage struct {
	PageInfo
	ObservedAt time.Time        `json:"observed_at"`
	Sessions   []ControlSession `json:"sessions"`
}
type ControlTasksInput struct {
	ListInput
	Query string `json:"query,omitempty"`
}
type RecentEventsInput struct {
	BeforeCursor int64  `json:"before_cursor"`
	Limit        int    `json:"limit"`
	TaskKey      string `json:"task_key,omitempty"`
	SessionID    string `json:"session_id,omitempty"`
	Type         string `json:"type,omitempty"`
}
type RecentEventsPage struct {
	Events     []Event   `json:"events"`
	NextCursor int64     `json:"next_cursor"`
	HasMore    bool      `json:"has_more"`
	ObservedAt time.Time `json:"observed_at"`
}

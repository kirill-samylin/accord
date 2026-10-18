package core

import "time"

// ListInput keeps a fixed numeric creation boundary across a paginated pass.
// Changing a filter requires starting a new pass without Cursor.
type ListInput struct {
	Cursor        string `json:"cursor,omitempty"`
	Limit         int    `json:"limit,omitempty"`
	State         string `json:"state,omitempty"`
	Role          string `json:"role,omitempty"`
	EpicKey       string `json:"epic_key,omitempty"`
	Owner         string `json:"owner,omitempty"`
	IncludeClosed bool   `json:"include_closed,omitempty"`
}
type PageInfo struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HighWater  int64  `json:"high_water"`
	HasMore    bool   `json:"has_more"`
}
type TasksPage struct {
	PageInfo
	Tasks []Task `json:"tasks"`
}
type SessionsPage struct {
	PageInfo
	Sessions []Session `json:"sessions"`
}
type NotesPage struct {
	PageInfo
	Notes []Note `json:"notes"`
}
type NotesInput struct {
	Cursor         string `json:"cursor,omitempty"`
	Limit          int    `json:"limit,omitempty"`
	ScopeType      string `json:"scope_type,omitempty"`
	ScopeID        string `json:"scope_id,omitempty"`
	IncludeDeleted bool   `json:"include_deleted,omitempty"`
}
type Note struct {
	Key             string     `json:"key"`
	ScopeType       string     `json:"scope_type"`
	ScopeID         string     `json:"scope_id"`
	Text            string     `json:"text"`
	SourceTaskKey   string     `json:"source_task_key"`
	AuthorPrincipal string     `json:"author_principal"`
	AuthorSession   string     `json:"author_session"`
	CreatedAt       time.Time  `json:"created_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
	DeleteReason    string     `json:"delete_reason,omitempty"`
}
type NoteInput struct {
	RequestID     string `json:"request_id"`
	Action        string `json:"action"`
	Key           string `json:"key,omitempty"`
	Reason        string `json:"reason,omitempty"`
	ScopeType     string `json:"scope_type,omitempty"`
	ScopeID       string `json:"scope_id,omitempty"`
	Text          string `json:"text,omitempty"`
	SourceTaskKey string `json:"source_task_key,omitempty"`
}
type NoteLimits struct{ MaxPerRole, MaxBytesPerRole, MaxPerEpic, MaxBytesPerEpic int }
type QueueSummary struct {
	Ready      int `json:"ready"`
	Waiting    int `json:"waiting"`
	InProgress int `json:"in_progress"`
}

// EventData records facts at mutation time. Missing Data in historical events
// means unknown provenance; consumers must never reconstruct it from today's task.
type EventData struct {
	NoteKey          string     `json:"note_key,omitempty"`
	ScopeType        string     `json:"scope_type,omitempty"`
	ScopeID          string     `json:"scope_id,omitempty"`
	TokensSpentDelta *int64     `json:"tokens_spent_delta,omitempty"`
	TokensSpentTotal *int64     `json:"tokens_spent_total,omitempty"`
	CorrectionDelta  *int64     `json:"correction_delta,omitempty"`
	Incomplete       bool       `json:"incomplete"`
	AttemptEnded     bool       `json:"attempt_ended"`
	Role             string     `json:"role,omitempty"`
	Owner            string     `json:"owner,omitempty"`
	EpicKey          string     `json:"epic_key,omitempty"`
	Kind             string     `json:"kind,omitempty"`
	OldState         string     `json:"old_state,omitempty"`
	NewState         string     `json:"new_state,omitempty"`
	FirstDone        bool       `json:"first_done"`
	Rework           bool       `json:"rework"`
	Returns          int64      `json:"returns,omitempty"`
	TokensSpent      *int64     `json:"tokens_spent,omitempty"`
	FirstClaimAt     *time.Time `json:"first_claim_at,omitempty"`
	FirstDoneAt      *time.Time `json:"first_done_at,omitempty"`
	Blocked          bool       `json:"blocked"`
}
type StatsInput struct {
	From    time.Time `json:"from"`
	To      time.Time `json:"to"`
	GroupBy string    `json:"group_by"`
}
type StatsMetrics struct {
	FirstCompletions       int64        `json:"first_completions"`
	ReworkCompletions      int64        `json:"rework_completions"`
	EpicCompletions        int64        `json:"epic_completions"`
	Returns                int64        `json:"returns"`
	TokensSpent            int64        `json:"tokens_spent"`
	Attempts               int64        `json:"attempts"`
	MissingTokens          int64        `json:"missing_tokens"`
	MissingTokensShare     float64      `json:"missing_tokens_share"`
	ExpiredLeases          int64        `json:"expired_leases"`
	RuleWarnings           int64        `json:"rule_warnings"`
	WallDurationSeconds    float64      `json:"wall_duration_seconds"`
	BlockedDurationSeconds float64      `json:"blocked_duration_seconds"`
	CurrentQueue           QueueSummary `json:"current_queue"`
}
type StatsGroup struct {
	Key     string       `json:"key"`
	Metrics StatsMetrics `json:"metrics"`
}
type Stats struct {
	From             time.Time    `json:"from"`
	To               time.Time    `json:"to"`
	GroupBy          string       `json:"group_by"`
	Totals           StatsMetrics `json:"totals"`
	Groups           []StatsGroup `json:"groups"`
	IncompleteData   bool         `json:"incomplete_data"`
	IncompleteEvents int64        `json:"incomplete_events"`
}

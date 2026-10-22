// Package core defines the pilot's transport-independent public contracts.
package core

import (
	"strings"
	"time"
)

type Principal struct {
	ID      string `json:"id"`
	Owner   string `json:"owner"`
	Machine string `json:"machine"`
}
type Actor struct {
	Principal Principal
	SessionID string
}
type Config struct {
	SessionTTL time.Duration
	LeaseTTL   time.Duration
	// DuplicateMode is off|warn|strict; DuplicateThreshold is 0.3..1.
	DuplicateMode      string
	DuplicateThreshold float64
	Project            string
	CursorNamespace    string
	Notes              NoteLimits
}
type Clock func() time.Time

type OpenSession struct {
	ProfileID      string `json:"profile_id,omitempty"`
	ProfileVersion int64  `json:"profile_version,omitempty"`
	RequestID      string `json:"request_id"`
	Role           string `json:"role"`
	RunID          string `json:"run_id"`
	Machine        string `json:"machine"`
}
type SessionView struct {
	Profile               *ProfileSnapshot  `json:"profile,omitempty"`
	SessionID             string            `json:"session_id"`
	Role                  string            `json:"role"`
	RoleGeneration        int64             `json:"role_generation"`
	ExpiresAt             time.Time         `json:"expires_at"`
	Project               string            `json:"project"`
	SessionTTLSeconds     float64           `json:"session_ttl_seconds"`
	TaskTTLSeconds        float64           `json:"task_ttl_seconds"`
	HeartbeatEverySeconds float64           `json:"heartbeat_every_seconds"`
	Notes                 []Note            `json:"notes,omitempty"`
	Rules                 map[string]string `json:"rules,omitempty"`
	MyTasks               []Task            `json:"my_tasks,omitempty"`
	Queue                 QueueSummary      `json:"queue"`
}
type Session struct {
	SessionView
	PrincipalID string `json:"principal_id"`
	Order       int64  `json:"order"`
	Owner       string `json:"owner,omitempty"`
	RunID       string `json:"run_id"`
	Machine     string `json:"machine"`
}
type Lease struct {
	SessionID  string    `json:"session_id"`
	Generation int64     `json:"generation"`
	ExpiresAt  time.Time `json:"expires_at"`
}
type Evidence struct {
	Commit         string   `json:"commit,omitempty"`
	RunSummary     string   `json:"run_summary,omitempty"`
	Artifact       []string `json:"artifact,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	Answer         string   `json:"answer,omitempty"`
	VerifiedCommit string   `json:"verified_commit,omitempty"`
	Findings       string   `json:"findings,omitempty"`
}
type Attempt struct {
	TokensSpent *int64     `json:"tokens_spent,omitempty"`
	TokensDelta int64      `json:"tokens_delta,omitempty"`
	SessionID   string     `json:"session_id"`
	PrincipalID string     `json:"principal_id"`
	RunID       string     `json:"run_id"`
	Machine     string     `json:"machine"`
	Generation  int64      `json:"generation"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	Reason      string     `json:"reason,omitempty"`
	Branch      string     `json:"branch,omitempty"`
	Commit      string     `json:"commit,omitempty"`
}
type Review struct {
	TargetKey        string `json:"target_key"`
	TargetCommit     string `json:"target_commit"`
	TargetGeneration int64  `json:"target_generation"`
	Outcome          string `json:"outcome"`
	AuthorSession    string `json:"author_session"`
}
type Link struct {
	From      string `json:"from"`
	To        string `json:"to"`
	Type      string `json:"type"`
	Condition string `json:"condition,omitempty"`
}
type Comment struct {
	Text      string    `json:"text"`
	SessionID string    `json:"session_id"`
	At        time.Time `json:"at"`
}
type Task struct {
	Assignment    *TaskAssignment `json:"assignment,omitempty"`
	CreatedAt     *time.Time      `json:"created_at,omitempty"`
	FirstClaimAt  *time.Time      `json:"first_claim_at,omitempty"`
	FirstDoneAt   *time.Time      `json:"first_done_at,omitempty"`
	ReturnsCount  int64           `json:"returns_count"`
	Category      string          `json:"category,omitempty"`
	Complexity    string          `json:"complexity,omitempty"`
	ExternalID    string          `json:"external_id,omitempty"`
	Key           string          `json:"key"`
	Type          string          `json:"type"`
	Owner         string          `json:"owner,omitempty"`
	Requester     string          `json:"requester,omitempty"`
	ReadyAt       time.Time       `json:"ready_at"`
	Attempts      []Attempt       `json:"attempts,omitempty"`
	Areas         []string        `json:"areas,omitempty"`
	EpicKey       string          `json:"epic_key,omitempty"`
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	Role          string          `json:"role"`
	State         string          `json:"state"`
	Priority      int             `json:"priority"`
	Version       int64           `json:"version"`
	Generation    int64           `json:"generation"`
	Lease         *Lease          `json:"lease,omitempty"`
	Evidence      Evidence        `json:"evidence"`
	Review        *Review         `json:"review,omitempty"`
	AuthorSession string          `json:"author_session,omitempty"`
	Comments      []Comment       `json:"comments,omitempty"`
	Links         []Link          `json:"links,omitempty"`
	Blocked       bool            `json:"blocked"`
	// CreatedBy is the creator's owner label, persisted from new creations.
	CreatedBy         string             `json:"created_by,omitempty"`
	DuplicateOverride *DuplicateOverride `json:"duplicate_override,omitempty"`
	// Response-only projections; never persisted on the stored task.
	HolderOwner   string    `json:"holder_owner,omitempty"`
	HolderMachine string    `json:"holder_machine,omitempty"`
	DuplicateOf   string    `json:"duplicate_of,omitempty"`
	Warnings      []Warning `json:"warnings,omitempty"`
}
type CreateInput struct {
	ProfileID      string   `json:"profile_id,omitempty"`
	ProfileVersion int64    `json:"profile_version,omitempty"`
	Category       string   `json:"category,omitempty"`
	Complexity     string   `json:"complexity,omitempty"`
	ExternalID     string   `json:"external_id,omitempty"`
	RequestID      string   `json:"request_id"`
	Type           string   `json:"type"`
	Owner          string   `json:"owner,omitempty"`
	Areas          []string `json:"areas,omitempty"`
	EpicKey        string   `json:"epic_key,omitempty"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Role           string   `json:"role"`
	Priority       int      `json:"priority"`
	TargetKey      string   `json:"target_key,omitempty"`
	TargetCommit   string   `json:"target_commit,omitempty"`
	// Force creates despite similar work under strict mode; ForceReason is
	// required with it and recorded on the task.
	Force       bool   `json:"force,omitempty"`
	ForceReason string `json:"force_reason,omitempty"`
}
type NextInput struct {
	RequestID string `json:"request_id"`
	EpicKey   string `json:"epic_key,omitempty"`
	TaskKey   string `json:"task_key,omitempty"`
}
type Claim struct {
	Warnings  []Warning    `json:"warnings,omitempty"`
	EpicNotes []Note       `json:"epic_notes,omitempty"`
	Queue     QueueSummary `json:"queue"`
	Task      *Task        `json:"task"`
	Lease     *Lease       `json:"lease,omitempty"`
}
type GetInput struct {
	TaskKey string `json:"task_key"`
}
type LeaseInput struct {
	RequestID  string `json:"request_id"`
	TaskKey    string `json:"task_key"`
	Generation int64  `json:"generation"`
	Reason     string `json:"reason,omitempty"`
	Branch     string `json:"branch,omitempty"`
}
type UpdateInput struct {
	Category   *string `json:"category,omitempty"`
	Complexity *string `json:"complexity,omitempty"`
	ExternalID *string `json:"external_id,omitempty"`
	LeaseInput
	Title       *string `json:"title,omitempty"`
	Description *string `json:"description,omitempty"`
}
type TransitionInput struct {
	TokensSpent *int64 `json:"tokens_spent,omitempty"`
	LeaseInput
	EvidenceRef string    `json:"evidence_ref,omitempty"`
	State       string    `json:"state"`
	Outcome     string    `json:"outcome,omitempty"`
	Evidence    *Evidence `json:"evidence,omitempty"`
}
type LinkInput struct {
	RequestID  string `json:"request_id"`
	Generation int64  `json:"generation,omitempty"`
	From       string `json:"from"`
	To         string `json:"to"`
	Type       string `json:"type"`
	Condition  string `json:"condition,omitempty"`
	Unlink     bool   `json:"unlink,omitempty"`
}
type HeartbeatInput struct {
	RequestID      string `json:"request_id"`
	TaskKey        string `json:"task_key,omitempty"`
	Generation     int64  `json:"generation,omitempty"`
	RoleGeneration int64  `json:"role_generation,omitempty"`
}
type CommentInput struct {
	TokensSpentDelta *int64 `json:"tokens_spent_delta,omitempty"`
	LeaseInput
	Text string `json:"text"`
}
type Event struct {
	Data      *EventData `json:"data,omitempty"`
	Cursor    int64      `json:"cursor"`
	Type      string     `json:"type"`
	TaskKey   string     `json:"task_key,omitempty"`
	SessionID string     `json:"session_id,omitempty"`
	TokenID   string     `json:"token_id,omitempty"`
	Reason    string     `json:"reason,omitempty"`
	At        time.Time  `json:"at"`
}
type EventsInput struct {
	SinceCursor int64    `json:"since_cursor"`
	Limit       int      `json:"limit"`
	Types       []string `json:"types,omitempty"`
	TaskKey     string   `json:"task_key,omitempty"`
}
type EventsPage struct {
	Events     []Event `json:"events"`
	NextCursor int64   `json:"next_cursor"`
	HasMore    bool    `json:"has_more"`
	HighWater  int64   `json:"high_water"`
}
type TokenInput struct {
	Owner   string
	Machine string
	TTL     time.Duration
}
type TokenView struct {
	ID        string    `json:"id"`
	Secret    string    `json:"secret"`
	ExpiresAt time.Time `json:"expires_at"`
}
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }
func Fail(code, message string) error {
	codes := map[string]string{"invalid_input": "VALIDATION", "forbidden": "FORBIDDEN_ROLE", "session_expired": "LEASE_LOST"}
	if mapped, ok := codes[code]; ok {
		code = mapped
	}
	return &Error{Code: strings.ToUpper(code), Message: message}
}
func ValidRole(role string) bool {
	switch role {
	case "coordinator", "analyst", "architect", "developer", "reviewer", "tester", "human":
		return true
	}
	return false
}
func Singleton(role string) bool { return role == "architect" }

package core

import "time"

// Duplicate detection modes. Off disables checks, warn reports similar work in
// the create response, strict rejects creation until the caller passes
// force with a reason. Detection never edits or cancels existing work.
const (
	DuplicateOff    = "off"
	DuplicateWarn   = "warn"
	DuplicateStrict = "strict"
)

// SimilarTask is a bounded summary of open or recently finished work that
// resembles a proposed or existing item. Owners are display labels taken from
// recorded sessions/credentials, not authorization facts.
type SimilarTask struct {
	Key            string          `json:"key"`
	Type           string          `json:"type"`
	Title          string          `json:"title"`
	State          string          `json:"state"`
	Role           string          `json:"role,omitempty"`
	EpicKey        string          `json:"epic_key,omitempty"`
	Score          float64         `json:"score"`
	Reasons        []string        `json:"reasons"`
	CreatedBy      string          `json:"created_by,omitempty"`
	HolderOwner    string          `json:"holder_owner,omitempty"`
	Assignment     *TaskAssignment `json:"assignment,omitempty"`
	CreatedAt      *time.Time      `json:"created_at,omitempty"`
	SharedAreas    []string        `json:"shared_areas,omitempty"`
	LiveLease      bool            `json:"live_lease"`
	FinishedRecent bool            `json:"finished_recent,omitempty"`
}

// Warning is advisory: the operation succeeded.
type Warning struct {
	Rule    string        `json:"rule"`
	Message string        `json:"message"`
	Similar []SimilarTask `json:"similar,omitempty"`
}

// DuplicateOverride records that a creator saw similar work and proceeded.
type DuplicateOverride struct {
	Reason  string   `json:"reason"`
	Similar []string `json:"similar"`
}

type SimilarInput struct {
	Type        string   `json:"type,omitempty"`
	Role        string   `json:"role,omitempty"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	ExternalID  string   `json:"external_id,omitempty"`
	Areas       []string `json:"areas,omitempty"`
	ExcludeKey  string   `json:"exclude_key,omitempty"`
	Limit       int      `json:"limit,omitempty"`
}
type SimilarPage struct {
	ObservedAt time.Time     `json:"observed_at"`
	Mode       string        `json:"mode"`
	Threshold  float64       `json:"threshold"`
	Similar    []SimilarTask `json:"similar"`
}

// DuplicateInput cancels unleased TaskKey as a duplicate of CanonicalKey.
type DuplicateInput struct {
	RequestID    string `json:"request_id"`
	TaskKey      string `json:"task_key"`
	CanonicalKey string `json:"canonical_key"`
	Version      int64  `json:"version,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// DuplicatePair is a server-detected candidate. Pairs already linked in either
// direction (including "related" — an explicit "not a duplicate") are omitted.
type DuplicatePair struct {
	Score   float64     `json:"score"`
	Reasons []string    `json:"reasons"`
	Older   SimilarTask `json:"older"`
	Newer   SimilarTask `json:"newer"`
}

// TeamMember aggregates recorded participation for one owner label.
type TeamMember struct {
	Owner         string           `json:"owner"`
	Machines      []string         `json:"machines"`
	Agents        []ControlSession `json:"agents"`
	HumanOnline   bool             `json:"human_online"`
	InProgress    int              `json:"in_progress"`
	OpenRequested int              `json:"open_requested"`
	DoneRecent    int              `json:"done_recent"`
	CreatedRecent int              `json:"created_recent"`
	LastEventAt   *time.Time       `json:"last_event_at,omitempty"`
}
type TeamView struct {
	ObservedAt         time.Time       `json:"observed_at"`
	Project            string          `json:"project"`
	RecentWindowHours  int             `json:"recent_window_hours"`
	Members            []TeamMember    `json:"members"`
	Unclaimed          []Task          `json:"unclaimed"`
	UnclaimedTotal     int             `json:"unclaimed_total"`
	Duplicates         []DuplicatePair `json:"duplicates"`
	DuplicatesTotal    int             `json:"duplicates_total"`
	DuplicateMode      string          `json:"duplicate_mode"`
	DuplicateThreshold float64         `json:"duplicate_threshold"`
}

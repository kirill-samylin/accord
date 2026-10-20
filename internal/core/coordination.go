package core

import "time"

// TaskAssignment pins routing to an explicit immutable profile revision.
// It is neither a credential nor ownership of a live lease.
type TaskAssignment struct {
	ProfileID      string `json:"profile_id"`
	ProfileVersion int64  `json:"profile_version"`
	ProfileName    string `json:"profile_name"`
	Role           string `json:"role"`
}
type AssignInput struct {
	RequestID      string `json:"request_id"`
	TaskKey        string `json:"task_key"`
	Version        int64  `json:"version"`
	Mode           string `json:"mode"`
	ProfileID      string `json:"profile_id,omitempty"`
	ProfileVersion int64  `json:"profile_version,omitempty"`
	Reason         string `json:"reason"`
}
type CheckCreateInput struct {
	RequestID      string `json:"request_id"`
	TargetKey      string `json:"target_key"`
	TargetVersion  int64  `json:"target_version"`
	Role           string `json:"role"`
	ProfileID      string `json:"profile_id,omitempty"`
	ProfileVersion int64  `json:"profile_version,omitempty"`
}
type AssignmentChange struct {
	Previous *TaskAssignment `json:"previous"`
	Next     *TaskAssignment `json:"next"`
	Reason   string          `json:"reason"`
}
type CheckCreated struct {
	TargetKey        string `json:"target_key"`
	TargetCommit     string `json:"target_commit"`
	TargetGeneration int64  `json:"target_generation"`
	CheckKey         string `json:"check_key"`
}
type CoordinationInput struct {
	TaskKey string `json:"task_key"`
}
type CoordinationReason struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TaskKey string `json:"task_key,omitempty"`
}
type CoordinationDependency struct {
	Link      Link `json:"link"`
	Satisfied bool `json:"satisfied"`
}
type CoordinationCheck struct {
	TaskKey          string `json:"task_key"`
	Role             string `json:"role"`
	State            string `json:"state"`
	Outcome          string `json:"outcome"`
	TargetCommit     string `json:"target_commit"`
	TargetGeneration int64  `json:"target_generation"`
	Current          bool   `json:"current"`
	Successful       bool   `json:"successful"`
	Latest           bool   `json:"latest"`
}
type TaskCoordination struct {
	TaskKey        string                   `json:"task_key"`
	TaskVersion    int64                    `json:"task_version"`
	ObservedAt     time.Time                `json:"observed_at"`
	Assignment     *TaskAssignment          `json:"assignment,omitempty"`
	RoutingReasons []CoordinationReason     `json:"routing_reasons"`
	Blockers       []CoordinationReason     `json:"blockers"`
	Dependencies   []CoordinationDependency `json:"dependencies"`
	Checks         []CoordinationCheck      `json:"checks"`
	EpicClosable   *bool                    `json:"epic_closable,omitempty"`
	ClosureReasons []CoordinationReason     `json:"closure_reasons"`
	Truncated      bool                     `json:"truncated,omitempty"`
}

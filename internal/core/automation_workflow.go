package core

import "time"

type WorkflowGrant struct {
	PolicyID           string                           `json:"policy_id"`
	PolicyVersion      int64                            `json:"policy_version"`
	IssuerID           string                           `json:"issuer_id"`
	IssuerEpoch        int64                            `json:"issuer_epoch"`
	RequesterID        string                           `json:"requester_id"`
	RequesterEpoch     int64                            `json:"requester_epoch"`
	HostOwnerID        string                           `json:"host_owner_id"`
	HostOwnerEpoch     int64                            `json:"host_owner_epoch"`
	HostConsentVersion int64                            `json:"host_consent_version"`
	RoleBindings       map[string]AutomationRoleBinding `json:"role_bindings"`
	Profiles           map[string]ProfileSnapshot       `json:"profiles"`
	Checks             []AutomationCheckPresetRef       `json:"checks"`
	Limits             AutomationLimits                 `json:"limits"`
	ApprovedAt         time.Time                        `json:"approved_at"`
	ExpiresAt          time.Time                        `json:"expires_at"`
}
type WorkflowCounters struct {
	Invocations int            `json:"invocations"`
	Started     int            `json:"started"`
	FixTotal    int            `json:"fix_total"`
	FixPerTask  map[string]int `json:"fix_per_task"`
	Tasks       int            `json:"tasks"`
}
type Workflow struct {
	// IntakeTruncated is a list-only wire projection, never an edited requirement.
	IntakeTruncated bool             `json:"intake_truncated,omitempty"`
	ID              string           `json:"id"`
	Version         int64            `json:"version"`
	ControlEpoch    int64            `json:"control_epoch"`
	State           string           `json:"state"`
	ReasonCode      string           `json:"reason_code,omitempty"`
	IntakeText      string           `json:"intake_text"`
	IntakeHash      string           `json:"intake_hash"`
	Requester       Principal        `json:"requester"`
	Grant           *WorkflowGrant   `json:"grant,omitempty"`
	HostID          string           `json:"host_id,omitempty"`
	Workspace       string           `json:"workspace,omitempty"`
	EpicKey         string           `json:"epic_key,omitempty"`
	PlanRevision    int64            `json:"plan_revision"`
	CurrentStepID   string           `json:"current_step_id,omitempty"`
	Counters        WorkflowCounters `json:"counters"`
	ReservedRuns    int              `json:"reserved_runs"`
	DeadlineAt      time.Time        `json:"deadline_at"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	Messages        []string         `json:"messages,omitempty"`
}
type WorkflowStep struct {
	ID                string          `json:"id"`
	WorkflowID        string          `json:"workflow_id"`
	Sequence          int64           `json:"sequence"`
	Kind              string          `json:"kind"`
	Status            string          `json:"status"`
	AttemptNo         int64           `json:"attempt_no"`
	TaskKey           string          `json:"task_key"`
	TaskVersion       int64           `json:"task_version"`
	TaskGeneration    int64           `json:"task_generation"`
	Profile           ProfileSnapshot `json:"profile_snapshot"`
	Provider          string          `json:"provider"`
	TargetSHA         string          `json:"target_sha,omitempty"`
	ManagedRunID      string          `json:"managed_run_id"`
	ManagedGeneration int64           `json:"managed_generation"`
	NativeSessionID   string          `json:"native_session_id,omitempty"`
	StartedAt         time.Time       `json:"started_at"`
	DeadlineAt        time.Time       `json:"deadline_at"`
	EndedAt           *time.Time      `json:"ended_at,omitempty"`
	EvidenceRefs      []string        `json:"evidence_refs,omitempty"`
	ContributorRunIDs []string        `json:"contributor_run_ids,omitempty"`
	Continuations     int             `json:"continuations"`
	DeliveredMessages int             `json:"delivered_messages"`
}
type WorkflowSubmitInput struct {
	RequestID     string `json:"request_id"`
	PolicyID      string `json:"policy_id,omitempty"`
	PolicyVersion int64  `json:"policy_version,omitempty"`
	Text          string `json:"text"`
}
type WorkflowCommandInput struct {
	RequestID            string `json:"request_id"`
	WorkflowID           string `json:"workflow_id"`
	Version              int64  `json:"version"`
	Kind                 string `json:"kind"`
	Text                 string `json:"text,omitempty"`
	RefreshPolicyID      string `json:"refresh_policy_id,omitempty"`
	RefreshPolicyVersion int64  `json:"refresh_policy_version,omitempty"`
}
type WorkflowReadInput struct {
	WorkflowID string `json:"workflow_id,omitempty"`
	State      string `json:"state,omitempty"`
	Cursor     string `json:"cursor,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}
type WorkflowListPage struct {
	ObservedAt       time.Time          `json:"observed_at"`
	ActorPrincipalID string             `json:"actor_principal_id"`
	Policies         []AutomationPolicy `json:"policies"`
	Workflows        []Workflow         `json:"workflows"`
	NextCursor       string             `json:"next_cursor,omitempty"`
}
type WorkflowPage struct {
	Workflow     Workflow                 `json:"workflow"`
	Steps        []WorkflowStep           `json:"steps"`
	Plans        []WorkflowPlanSnapshot   `json:"plans"`
	Questions    []WorkflowQuestionRecord `json:"questions"`
	Attestations []WorkflowAttestation    `json:"attestations"`
	CanControl   bool                     `json:"can_control"`
	NextCursor   string                   `json:"next_cursor,omitempty"`
}
type WorkflowAdvanceInput struct {
	WorkflowID      string `json:"workflow_id"`
	ExpectedVersion int64  `json:"expected_version"`
	ControlEpoch    int64  `json:"control_epoch"`
}
type WorkflowAdvanceResult struct {
	Workflow  Workflow  `json:"workflow"`
	NextDueAt time.Time `json:"next_due_at"`
	Changed   bool      `json:"changed"`
}

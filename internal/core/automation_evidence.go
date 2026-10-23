package core

import "time"

type WorkflowPlanTask struct {
	ClientKey        string   `json:"client_key"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Areas            []string `json:"areas"`
	Acceptance       []string `json:"acceptance"`
	Prerequisites    []string `json:"prerequisites"`
	RequiredCheckIDs []string `json:"required_check_ids"`
}
type WorkflowPlanInput struct {
	RequestID      string             `json:"request_id"`
	WorkflowID     string             `json:"workflow_id"`
	Version        int64              `json:"version"`
	TaskGeneration int64              `json:"task_generation"`
	Tasks          []WorkflowPlanTask `json:"tasks"`
}
type WorkflowPlanSnapshot struct {
	WorkflowID         string             `json:"workflow_id"`
	Revision           int64              `json:"revision"`
	Hash               string             `json:"hash"`
	TaskGeneration     int64              `json:"task_generation"`
	Tasks              []WorkflowPlanTask `json:"tasks"`
	TaskKeys           map[string]string  `json:"task_keys"`
	SubmittedSessionID string             `json:"submitted_session_id"`
	Status             string             `json:"status"`
	CreatedAt          time.Time          `json:"created_at"`
}
type WorkflowQuestionInput struct {
	RequestID      string   `json:"request_id"`
	WorkflowID     string   `json:"workflow_id"`
	Version        int64    `json:"version"`
	TaskGeneration int64    `json:"task_generation"`
	Question       string   `json:"question"`
	Options        []string `json:"options,omitempty"`
}
type WorkflowQuestionRecord struct {
	ID                   string    `json:"id"`
	WorkflowID           string    `json:"workflow_id"`
	StepID               string    `json:"step_id"`
	Version              int64     `json:"version"`
	HumanTaskKey         string    `json:"human_task_key"`
	DesignatedPrincipals []string  `json:"designated_principals"`
	State                string    `json:"state"`
	Question             string    `json:"question"`
	Options              []string  `json:"options,omitempty"`
	AnswerRef            string    `json:"answer_ref,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}
type WorkflowAnswerInput struct {
	RequestID       string `json:"request_id"`
	WorkflowID      string `json:"workflow_id"`
	Version         int64  `json:"version"`
	QuestionID      string `json:"question_id"`
	QuestionVersion int64  `json:"question_version"`
	Answer          string `json:"answer"`
	Resume          bool   `json:"resume"`
}
type WorkflowObservations struct {
	ActualHead            string   `json:"actual_head,omitempty"`
	TargetSHA             string   `json:"target_sha,omitempty"`
	SourceClean           *bool    `json:"source_clean,omitempty"`
	ManifestSHA256        string   `json:"manifest_sha256,omitempty"`
	CheckExitCode         *int     `json:"check_exit_code,omitempty"`
	CheckAborted          bool     `json:"check_aborted,omitempty"`
	ChecksPassed          int64    `json:"checks_passed"`
	ChecksFailed          int64    `json:"checks_failed"`
	ChecksSkipped         int64    `json:"checks_skipped"`
	ReportKind            string   `json:"report_kind,omitempty"`
	ReportSHA256          string   `json:"report_sha256,omitempty"`
	ArtifactRefs          []string `json:"artifact_refs,omitempty"`
	OutcomeBytesTruncated bool     `json:"outcome_bytes_truncated"`
	TestDesign            bool     `json:"test_design,omitempty"`
}
type WorkflowAttestInput struct {
	ManagedPollInput
	CommandID       string               `json:"command_id"`
	DispatchID      string               `json:"dispatch_id"`
	RunID           string               `json:"run_id"`
	Generation      int64                `json:"generation"`
	WorkflowID      string               `json:"workflow_id"`
	StepID          string               `json:"step_id"`
	ControlEpoch    int64                `json:"control_epoch"`
	TaskKey         string               `json:"task_key"`
	TaskGeneration  int64                `json:"task_generation"`
	NativeSessionID string               `json:"native_session_id"`
	Kind            string               `json:"kind"`
	CheckRequestID  string               `json:"check_request_id,omitempty"`
	PresetID        string               `json:"preset_id,omitempty"`
	PresetVersion   int64                `json:"preset_version,omitempty"`
	PresetDigest    string               `json:"preset_digest,omitempty"`
	Observations    WorkflowObservations `json:"observations"`
}
type WorkflowAttestation struct {
	Sequence        int64                `json:"sequence"`
	ID              string               `json:"id"`
	WorkflowID      string               `json:"workflow_id"`
	StepID          string               `json:"step_id"`
	ControlEpoch    int64                `json:"control_epoch"`
	RunID           string               `json:"run_id"`
	Generation      int64                `json:"generation"`
	TaskKey         string               `json:"task_key"`
	TaskGeneration  int64                `json:"task_generation"`
	NativeSessionID string               `json:"native_session_id"`
	Kind            string               `json:"kind"`
	CheckRequestID  string               `json:"check_request_id,omitempty"`
	PresetID        string               `json:"preset_id,omitempty"`
	PresetVersion   int64                `json:"preset_version,omitempty"`
	PresetDigest    string               `json:"preset_digest,omitempty"`
	Status          string               `json:"status"`
	Observations    WorkflowObservations `json:"observations"`
	CreatedAt       time.Time            `json:"created_at"`
	UpdatedAt       time.Time            `json:"updated_at"`
}
type WorkflowAttestResult struct {
	// Attestation retains the exact original attempt tuple, including its epoch.
	Attestation WorkflowAttestation `json:"attestation"`
	// ControlEpoch describes current workflow state, not a refreshed grant. A
	// terminal closure after pause/revoke may return a newer epoch than its proof.
	ControlEpoch          int64 `json:"control_epoch"`
	MaxRuntimeSeconds     int64 `json:"max_runtime_seconds"`
	ReservationTTLSeconds int64 `json:"reservation_ttl_seconds"`
}

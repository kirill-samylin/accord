package core

import "time"

const ExecutionTokenPrefix = "ac_exec1_"

type ExecutionCapability struct {
	ID             string    `json:"id"`
	HostID         string    `json:"host_id"`
	WorkflowID     string    `json:"workflow_id"`
	StepID         string    `json:"step_id"`
	ControlEpoch   int64     `json:"control_epoch"`
	RunID          string    `json:"run_id"`
	Generation     int64     `json:"generation"`
	WorkerEpoch    int64     `json:"worker_epoch"`
	Role           string    `json:"role"`
	ProfileID      string    `json:"profile_id"`
	ProfileVersion int64     `json:"profile_version"`
	State          string    `json:"state"`
	Operations     []string  `json:"operations"`
	CreatedAt      time.Time `json:"created_at"`
	ExpiresAt      time.Time `json:"expires_at"`
}
type ExecutionCapabilityRegisterInput struct {
	ManagedPollInput
	CommandID    string `json:"command_id"`
	DispatchID   string `json:"dispatch_id"`
	RunID        string `json:"run_id"`
	Generation   int64  `json:"generation"`
	WorkflowID   string `json:"workflow_id"`
	StepID       string `json:"step_id"`
	ControlEpoch int64  `json:"control_epoch"`
	CapabilityID string `json:"capability_id"`
	Digest       string `json:"digest"`
}

// ExecutionIdentity is server-only authentication output. It carries no secret.
type ExecutionIdentity struct {
	Principal  Principal
	Capability ExecutionCapability
}

// ManagedAutomationBinding is server-generated execution scope, never model input.
type ManagedAutomationBinding struct {
	// TaskGeneration is the one permitted next claim generation; before claim
	// the observed board generation is TaskGeneration-1.
	TaskGeneration int64                      `json:"task_generation"`
	WorkflowID     string                     `json:"workflow_id"`
	StepID         string                     `json:"step_id"`
	ControlEpoch   int64                      `json:"control_epoch"`
	MessageCount   int                        `json:"message_count"`
	PolicyVersion  int64                      `json:"policy_version"`
	DeadlineAt     time.Time                  `json:"deadline_at"`
	TargetSHA      string                     `json:"target_sha,omitempty"`
	Checks         []AutomationCheckPresetRef `json:"checks"`
}

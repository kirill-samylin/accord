package worker

import (
	"context"
	"time"

	"agent-board/internal/automationipc"
	"agent-board/internal/core"
)

type automationAttempt struct {
	Compacted             bool                        `json:"compacted,omitempty"`
	RunSnapshotSHA256     string                      `json:"run_snapshot_sha256,omitempty"`
	ProfileSnapshotSHA256 string                      `json:"profile_snapshot_sha256,omitempty"`
	IPC                   *automationipc.Config       `json:"ipc,omitempty"`
	Run                   core.ManagedRun             `json:"run"`
	CapabilityID          string                      `json:"capability_id"`
	CapabilityDigest      string                      `json:"capability_digest"`
	CapabilityRequestID   string                      `json:"capability_request_id"`
	TokenFile             string                      `json:"token_file"`
	Registered            bool                        `json:"registered"`
	Baseline              gitProof                    `json:"baseline"`
	NativeExited          bool                        `json:"native_exited"`
	ExitObservations      *core.WorkflowObservations  `json:"exit_observations,omitempty"`
	ExitUnreportable      bool                        `json:"exit_unreportable,omitempty"`
	ExitDurable           bool                        `json:"exit_durable"`
	ManagedAck            bool                        `json:"managed_ack"`
	LockReleased          bool                        `json:"lock_released"`
	ExitAck               bool                        `json:"exit_ack"`
	Checks                map[string]*automationCheck `json:"checks"`
	Pending               []*core.WorkflowAttestInput `json:"pending"`
}
type automationCheck struct {
	ID                string                    `json:"id"`
	PayloadHash       string                    `json:"payload_hash"`
	Request           automationipc.Request     `json:"request"`
	Status            string                    `json:"status"`
	ErrorCode         string                    `json:"error_code,omitempty"`
	Result            *automationipc.Result     `json:"result,omitempty"`
	ReservedRequestID string                    `json:"reserved_request_id"`
	StartedRequestID  string                    `json:"started_request_id"`
	FinishedRequestID string                    `json:"finished_request_id"`
	StartedAt         time.Time                 `json:"started_at"`
	Observations      core.WorkflowObservations `json:"observations"`
}
type automationBudget struct {
	FirstStartedAt time.Time `json:"first_started_at"`
	DeadlineAt     time.Time `json:"deadline_at"`
	Invocations    int       `json:"invocations"`
}
type automationRuntime struct {
	ctx        context.Context
	cancel     context.CancelFunc
	ipc        *automationipc.Server
	checks     map[string]process
	checksDone chan struct{}
	checkCount int
	deadline   time.Time
	closing    bool
}

func (e *Engine) automationPost(ctx context.Context, path string, input, output any) error {
	e.mu.Lock()
	post := e.automationTransport
	e.mu.Unlock()
	if post == nil {
		return consentError()
	}
	return post(ctx, path, input, output)
}

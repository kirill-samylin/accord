package core

import "time"

// ManagedHost grants control of an explicitly enrolled local workspace. It is
// neither a native board session nor evidence that a provider process is alive.
type ManagedHost struct {
	Automation         *AutomationHostConsent `json:"automation,omitempty"`
	WorkerCapabilities map[string]int         `json:"worker_capabilities,omitempty"`
	ID                 string                 `json:"id"`
	OwnerPrincipalID   string                 `json:"owner_principal_id"`
	OwnerAccessEpoch   int64                  `json:"owner_access_epoch"`
	Name               string                 `json:"name"`
	Version            int64                  `json:"version"`
	State              string                 `json:"state"`
	Controllers        []string               `json:"controllers"`
	Workspaces         []string               `json:"workspaces"`
	Providers          []string               `json:"providers"`
	LastSeenAt         time.Time              `json:"last_seen_at"`
	WorkerEpoch        int64                  `json:"worker_epoch"`
	LeaseExpiresAt     time.Time              `json:"lease_expires_at"`
	CanControl         bool                   `json:"can_control"`
}
type ManagedRun struct {
	Automation        *ManagedAutomationBinding `json:"automation,omitempty"`
	ID                string                    `json:"id"`
	HostID            string                    `json:"host_id"`
	Version           int64                     `json:"version"`
	Provider          string                    `json:"provider"`
	Workspace         string                    `json:"workspace"`
	Prompt            string                    `json:"prompt"`
	Profile           *ProfileSnapshot          `json:"profile_snapshot,omitempty"`
	TaskKey           string                    `json:"task_key,omitempty"`
	CreatedBy         string                    `json:"created_by"`
	State             string                    `json:"state"`
	Generation        int64                     `json:"generation"`
	ProviderSessionID string                    `json:"provider_session_id,omitempty"`
	CreatedAt         time.Time                 `json:"created_at"`
	UpdatedAt         time.Time                 `json:"updated_at"`
	LastError         string                    `json:"last_error,omitempty"`
	Output            []string                  `json:"output"`
	OutputTruncated   bool                      `json:"output_truncated"`
}
type ManagedCommand struct {
	ID         string     `json:"id"`
	RunID      string     `json:"run_id"`
	Sequence   int64      `json:"sequence"`
	Generation int64      `json:"generation"`
	Kind       string     `json:"kind"`
	Text       string     `json:"text,omitempty"`
	CreatedBy  string     `json:"created_by"`
	CreatedAt  time.Time  `json:"created_at"`
	Status     string     `json:"status"`
	DispatchID string     `json:"dispatch_id,omitempty"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	AppliedAt  *time.Time `json:"applied_at,omitempty"`
	Error      string     `json:"error,omitempty"`
	Forced     bool       `json:"forced,omitempty"`
}
type ManagedListInput struct {
	Workspace string `json:"workspace,omitempty"`
	HostID    string `json:"host_id,omitempty"`
}
type ManagedPage struct {
	ObservedAt       time.Time     `json:"observed_at"`
	ActorPrincipalID string        `json:"actor_principal_id"`
	Hosts            []ManagedHost `json:"hosts"`
	Runs             []ManagedRun  `json:"runs"`
}
type ManagedRunInput struct {
	RunID string `json:"run_id"`
}

// ManagedPermissionInput is internal to cached browser authorization. It cannot
// execute a command and is deliberately absent from native route bindings.
type ManagedPermissionInput struct{ HostID, RunID, Kind string }
type ManagedRunPage struct {
	Run               ManagedRun       `json:"run"`
	Commands          []ManagedCommand `json:"commands"`
	CommandsTruncated bool             `json:"commands_truncated"`
}
type ManagedHostEnrollInput struct {
	Automation  *AutomationHostConsent `json:"automation,omitempty"`
	RequestID   string                 `json:"request_id"`
	Name        string                 `json:"name"`
	HostSecret  string                 `json:"host_secret"`
	Controllers []string               `json:"controllers"`
	Workspaces  []string               `json:"workspaces"`
	Providers   []string               `json:"providers"`
}
type ManagedHostUpdateInput struct {
	ManagedHostEnrollInput
	HostID  string `json:"host_id"`
	Version int64  `json:"version"`
	State   string `json:"state"`
}
type ManagedHostDisableInput struct {
	RequestID string `json:"request_id"`
	HostID    string `json:"host_id"`
	Version   int64  `json:"version"`
}
type ManagedStartInput struct {
	RequestID      string `json:"request_id"`
	HostID         string `json:"host_id"`
	Workspace      string `json:"workspace"`
	Provider       string `json:"provider"`
	ProfileID      string `json:"profile_id"`
	ProfileVersion int64  `json:"profile_version"`
	Prompt         string `json:"prompt"`
	TaskKey        string `json:"task_key,omitempty"`
}
type ManagedCommandInput struct {
	RequestID string `json:"request_id"`
	RunID     string `json:"run_id"`
	Version   int64  `json:"version"`
	Kind      string `json:"kind"`
	Text      string `json:"text,omitempty"`
}
type ManagedResult struct {
	Run     ManagedRun     `json:"run"`
	Command ManagedCommand `json:"command"`
}
type ManagedPollInput struct {
	WorkerCapabilities map[string]int `json:"worker_capabilities,omitempty"`
	RequestID          string         `json:"request_id"`
	HostID             string         `json:"host_id"`
	HostSecret         string         `json:"host_secret"`
	WorkerID           string         `json:"worker_id"`
	WorkerEpoch        int64          `json:"worker_epoch,omitempty"`
}
type ManagedPollResult struct {
	Host           ManagedHost     `json:"host"`
	WorkerEpoch    int64           `json:"worker_epoch"`
	LeaseExpiresAt time.Time       `json:"lease_expires_at"`
	Command        *ManagedCommand `json:"command,omitempty"`
	Run            *ManagedRun     `json:"run,omitempty"`
}
type ManagedReportInput struct {
	ManagedPollInput
	CommandID         string   `json:"command_id"`
	DispatchID        string   `json:"dispatch_id"`
	RunID             string   `json:"run_id"`
	Generation        int64    `json:"generation"`
	Phase             string   `json:"phase"`
	ProviderSessionID string   `json:"provider_session_id,omitempty"`
	Error             string   `json:"error,omitempty"`
	Output            []string `json:"output,omitempty"`
	Forced            bool     `json:"forced,omitempty"`
}

// NativeProviders are the managed native CLIs a worker may be enrolled with:
// Codex, Claude Code and Gennadiy (the corporate OpenCode fork).
var NativeProviders = []string{"codex", "claude", "gennadiy"}

func KnownNativeProvider(provider string) bool {
	for _, known := range NativeProviders {
		if provider == known {
			return true
		}
	}
	return false
}

// ProfileProviderFor maps a native CLI to the agent profile provider it serves;
// an unknown CLI maps to "" and is rejected by callers.
func ProfileProviderFor(provider string) string {
	return map[string]string{"codex": "openai", "claude": "anthropic", "gennadiy": "gennadiy"}[provider]
}

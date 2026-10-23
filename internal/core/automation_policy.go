package core

import "time"

// Automation policies delegate a bounded workflow, never raw host execution.
type AutomationRoleBinding struct {
	Provider       string `json:"provider"`
	ProfileID      string `json:"profile_id"`
	ProfileVersion int64  `json:"profile_version"`
}
type AutomationLimits struct {
	PlanTasks           int   `json:"plan_tasks"`
	FixPerTask          int   `json:"fix_per_task"`
	FixTotal            int   `json:"fix_total"`
	NativeInvocations   int   `json:"native_invocations"`
	ContinuationPerStep int   `json:"continuation_per_step"`
	StepSeconds         int64 `json:"step_seconds"`
	WorkflowSeconds     int64 `json:"workflow_seconds"`
	GrantSeconds        int64 `json:"grant_seconds"`
}
type AutomationCheckPresetRef struct {
	PresetID     string `json:"preset_id"`
	LocalVersion int64  `json:"local_version"`
	LocalDigest  string `json:"local_digest"`
	Kind         string `json:"kind"`
	Required     bool   `json:"required"`
	Isolation    string `json:"isolation"`
}
type AutomationWorkspaceScope struct {
	Version              int64                      `json:"version"`
	ExpiresAt            time.Time                  `json:"expires_at"`
	Roles                map[string][]string        `json:"roles"`
	Checks               []AutomationCheckPresetRef `json:"checks"`
	MaxStepSeconds       int64                      `json:"max_step_seconds"`
	MaxWorkflowSeconds   int64                      `json:"max_workflow_seconds"`
	MaxNativeInvocations int                        `json:"max_native_invocations"`
}

type AutomationHostConsent struct {
	// ScopesTruncated marks a list-only summary; never use it as a grant.
	ScopesTruncated bool                                `json:"scopes_truncated,omitempty"`
	Scopes          map[string]AutomationWorkspaceScope `json:"scopes"`
	Enabled         bool                                `json:"enabled"`
	Version         int64                               `json:"version"`
	ExpiresAt       time.Time                           `json:"expires_at"`
	Workspaces      []string                            `json:"workspaces"`
	Roles           []string                            `json:"roles"`
	Providers       []string                            `json:"providers"`
	Checks          []AutomationCheckPresetRef          `json:"checks"`
	MaxStepSeconds  int64                               `json:"max_step_seconds"`
}
type AutomationPolicySettings struct {
	Name               string                           `json:"name"`
	HostID             string                           `json:"host_id"`
	HostConsentVersion int64                            `json:"host_consent_version"`
	Workspace          string                           `json:"workspace"`
	RoleBindings       map[string]AutomationRoleBinding `json:"role_bindings"`
	IntakePrincipals   []string                         `json:"intake_principals"`
	ControlPrincipals  []string                         `json:"control_principals"`
	ClientIntake       bool                             `json:"client_intake"`
	Checks             []AutomationCheckPresetRef       `json:"checks"`
	GitPolicy          string                           `json:"git_policy"`
	Limits             AutomationLimits                 `json:"limits"`
	ExpiresAt          time.Time                        `json:"expires_at"`
}
type AutomationPolicy struct {
	AutomationPolicySettings
	ID                string    `json:"id"`
	Version           int64     `json:"version"`
	State             string    `json:"state"`
	IssuerPrincipalID string    `json:"issuer_principal_id"`
	IssuerEpoch       int64     `json:"issuer_epoch"`
	HostOwnerEpoch    int64     `json:"host_owner_epoch"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
type AutomationPolicyCreateInput struct {
	RequestID string `json:"request_id"`
	AutomationPolicySettings
}
type AutomationPolicyUpdateInput struct {
	AutomationPolicyCreateInput
	PolicyID string `json:"policy_id"`
	Version  int64  `json:"version"`
}
type AutomationPolicyRevokeInput struct {
	RequestID string `json:"request_id"`
	PolicyID  string `json:"policy_id"`
	Version   int64  `json:"version"`
	Reason    string `json:"reason"`
}
type AutomationPolicyInput struct {
	PolicyID string `json:"policy_id"`
	Version  int64  `json:"version,omitempty"`
	Cursor   string `json:"cursor,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}
type AutomationPolicyPage struct {
	Policy     AutomationPolicy   `json:"policy"`
	Revisions  []AutomationPolicy `json:"revisions"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

// Read-only current authority check for browser receipts.
type AutomationPermissionInput struct {
	Workspace            string `json:"workspace,omitempty"`
	CommandKind          string `json:"command_kind,omitempty"`
	RefreshPolicyID      string `json:"refresh_policy_id,omitempty"`
	RefreshPolicyVersion int64  `json:"refresh_policy_version,omitempty"`
	QuestionID           string `json:"question_id,omitempty"`
	PolicyID             string `json:"policy_id,omitempty"`
	WorkflowID           string `json:"workflow_id,omitempty"`
	HostID               string `json:"host_id,omitempty"`
	Kind                 string `json:"kind"`
}

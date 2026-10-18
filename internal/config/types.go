// Package config parses bounded, strictly validated offline workflow snapshots.
// Snapshots contain no mutable shared state and perform no filesystem/network IO.
package config

import (
	"encoding/json"
	"time"
)

const MaxInputBytes = 256 << 10
const MaxEvidenceBytes = 32 << 10

type Problem struct {
	Path    string `json:"path"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Result struct {
	Valid           bool      `json:"valid"`
	CanonicalSHA256 string    `json:"canonical_sha256,omitempty"`
	Errors          []Problem `json:"errors"`
	Warnings        []Problem `json:"warnings"`
}

// Options declares implemented features; integrations default to disabled.
// ToolDescriptions, when nil, uses the stable built-in twelve-tool catalog.
type Options struct {
	GitHubAuth, GitHubEvents, Notifications bool
	ToolDescriptions                        map[string]string
}
type Config struct {
	Version       int                 `json:"version"`
	Project       Project             `json:"project"`
	Auth          Auth                `json:"auth"`
	Leases        Leases              `json:"leases"`
	Roles         map[string]Role     `json:"roles"`
	States        States              `json:"states"`
	Reopen        map[string][]string `json:"reopen"`
	Rules         Rules               `json:"rules"`
	Areas         []Area              `json:"areas"`
	Questions     Questions           `json:"questions"`
	Notes         Notes               `json:"notes"`
	MCP           MCP                 `json:"mcp"`
	GitHub        *GitHub             `json:"github,omitempty"`
	Notifications *Notifications      `json:"notifications,omitempty"`
}
type Project struct {
	Name       string `json:"name"`
	Repository string `json:"repository,omitempty"`
	IDs        IDs    `json:"ids"`
}
type IDs struct {
	EpicPrefix string `json:"epic_prefix"`
	TaskPrefix string `json:"task_prefix"`
	NotePrefix string `json:"note_prefix"`
}
type Auth struct {
	Mode              string      `json:"mode"`
	Local             Local       `json:"local"`
	AgentTokenTTLDays int         `json:"agent_token_ttl_days"`
	GitHub            *GitHubAuth `json:"github,omitempty"`
}
type Local struct {
	Users []string `json:"users"`
}
type GitHubAuth struct {
	AllowedLogins []string `json:"allowed_logins,omitempty"`
	AllowedOrg    string   `json:"allowed_org,omitempty"`
	AllowedTeam   string   `json:"allowed_team,omitempty"`
}
type Leases struct {
	TaskTTL       string `json:"task_ttl"`
	RoleTTL       string `json:"role_ttl"`
	SessionTTL    string `json:"session_ttl"`
	MaxSilent     string `json:"max_silent"`
	SweepInterval string `json:"sweep_interval"`
}
type Role struct {
	Kind              string              `json:"kind"`
	Singleton         bool                `json:"singleton"`
	CanCreate         []string            `json:"can_create"`
	CanCancel         bool                `json:"can_cancel"`
	CanCurateNotes    bool                `json:"can_curate_notes"`
	EvidenceOnDone    []string            `json:"evidence_on_done"`
	RecommendedOnDone []string            `json:"recommended_on_done"`
	IndependentFrom   []string            `json:"independent_from"`
	Members           []string            `json:"members"`
	Categories        map[string]Category `json:"categories"`
	EvidenceSchema    json.RawMessage     `json:"evidence_schema"`
}
type Category struct {
	AnswerBy      json.RawMessage `json:"answer_by"`
	ComplexityMax string          `json:"complexity_max"`
}
type States struct {
	ContractDeliveredAt string `json:"contract_delivered_at"`
}
type Rule struct {
	Mode string `json:"mode"`
}
type OwnerRule struct {
	Mode        string   `json:"mode"`
	SharedRoles []string `json:"shared_roles"`
}
type DuplicateRule struct {
	Mode      string  `json:"mode"`
	Threshold float64 `json:"threshold"`
}
type Rules struct {
	EvidenceRequired  Rule          `json:"evidence_required"`
	IndependentReview Rule          `json:"independent_review"`
	SingletonRole     Rule          `json:"singleton_role"`
	OneTaskPerAgent   Rule          `json:"one_task_per_agent"`
	EpicOwner         OwnerRule     `json:"epic_owner"`
	AreaOverlap       Rule          `json:"area_overlap"`
	DuplicateCheck    DuplicateRule `json:"duplicate_check"`
	CycleCheck        Rule          `json:"cycle_check"`
}
type Area struct {
	Name             string   `json:"name"`
	Aliases          []string `json:"aliases"`
	SingleEditorRole string   `json:"single_editor_role,omitempty"`
}
type Questions struct {
	EscalateAfter     string `json:"escalate_after"`
	DefaultOptionNote string `json:"default_option_note"`
}
type Notes struct {
	MaxPerRole      int `json:"max_per_role"`
	MaxBytesPerRole int `json:"max_bytes_per_role"`
	MaxPerEpic      int `json:"max_per_epic"`
	MaxBytesPerEpic int `json:"max_bytes_per_epic"`
}
type MCP struct {
	ToolBudgetTokens int                 `json:"tool_budget_tokens"`
	ToolsByRole      map[string][]string `json:"tools_by_role"`
}
type GitHub struct {
	WebhookSecretEnv        string   `json:"webhook_secret_env"`
	TaskKeyPattern          string   `json:"task_key_pattern"`
	RequireApprovalForMerge bool     `json:"require_approval_for_merge"`
	Events                  []string `json:"events"`
}
type Notifications struct {
	Webhooks []Webhook `json:"webhooks"`
}
type Webhook struct {
	URL       string   `json:"url"`
	SecretEnv string   `json:"secret_env"`
	Events    []string `json:"events"`
}

// Snapshot stores canonical bytes privately. Decode returns an independent copy.
type Snapshot struct {
	canonical []byte
	hash      string
}

func (s *Snapshot) CanonicalJSON() []byte { return append([]byte(nil), s.canonical...) }
func (s *Snapshot) SHA256() string        { return s.hash }
func (s *Snapshot) Decode() Config        { var c Config; _ = json.Unmarshal(s.canonical, &c); return c }

// TTLs returns validated lease durations and the conservative heartbeat interval.
func (s *Snapshot) TTLs() (task, role, session, heartbeat time.Duration) {
	c := s.Decode()
	task, _ = time.ParseDuration(c.Leases.TaskTTL)
	role, _ = time.ParseDuration(c.Leases.RoleTTL)
	session, _ = time.ParseDuration(c.Leases.SessionTTL)
	heartbeat = min(task, role, session) / 3
	return
}

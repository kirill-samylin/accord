package interchange

import (
	"encoding/json"
	"time"
)

const MaxBytes = 16 << 20
const MaxLine = 256 << 10
const MaxRecords = 10000

type Manifest struct {
	Format        string `json:"format"`
	Version       int    `json:"version"`
	Source        string `json:"source"`
	HighWater     int64  `json:"high_water,omitempty"`
	ConfigVersion int64  `json:"config_version,omitempty"`
}
type Ref struct {
	Kind       string `json:"kind"`
	ExternalID string `json:"external_id"`
}
type Evidence struct {
	Commit         string   `json:"commit,omitempty"`
	PR             string   `json:"pr,omitempty"`
	RunSummary     string   `json:"run_summary,omitempty"`
	Artifact       []string `json:"artifact,omitempty"`
	Summary        string   `json:"summary,omitempty"`
	Answer         string   `json:"answer,omitempty"`
	VerifiedCommit string   `json:"verified_commit,omitempty"`
	Findings       string   `json:"findings,omitempty"`
	Verdict        string   `json:"verdict,omitempty"`
}

// Task contains source metadata only. It deliberately cannot carry leases,
// generations, author sessions, approval bindings, or completion trust.
type Task struct {
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Role        string    `json:"role"`
	State       string    `json:"state"`
	Priority    int       `json:"priority"`
	Owner       string    `json:"owner,omitempty"`
	Requester   string    `json:"requester,omitempty"`
	Areas       []string  `json:"areas,omitempty"`
	Category    string    `json:"category,omitempty"`
	Complexity  string    `json:"complexity,omitempty"`
	Epic        *Ref      `json:"epic,omitempty"`
	Evidence    *Evidence `json:"evidence,omitempty"`
	Target      *Ref      `json:"target,omitempty"`
}
type Epic struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	State       string   `json:"state"`
	Priority    int      `json:"priority"`
	Owner       string   `json:"owner,omitempty"`
	Areas       []string `json:"areas,omitempty"`
}
type Link struct {
	From      Ref    `json:"from"`
	To        Ref    `json:"to"`
	Type      string `json:"type"`
	Condition string `json:"condition,omitempty"`
}
type Note struct {
	Text string `json:"text"`
	Kind string `json:"kind"`
	Role string `json:"role,omitempty"`
	Epic *Ref   `json:"epic,omitempty"`
}

// History is foreign evidence, never a server attempt, session or event.
// Configuration history may contain only this explicit sanitized summary.
type History struct {
	OriginRef          Ref       `json:"origin_ref"`
	Type               string    `json:"type"`
	At                 time.Time `json:"at"`
	Text               string    `json:"text,omitempty"`
	OriginSessionLabel string    `json:"origin_session_label,omitempty"`
	TokensSpent        *int64    `json:"tokens_spent,omitempty"`
	Evidence           *Evidence `json:"evidence,omitempty"`
	SourceState        string    `json:"source_state,omitempty"`
	Role               string    `json:"role,omitempty"`
	ConfigVersion      int64     `json:"config_version,omitempty"`
}
type Record struct {
	Kind       string          `json:"kind"`
	ExternalID string          `json:"external_id"`
	Data       json.RawMessage `json:"data"`
	Line       int             `json:"-"`
	Hash       string          `json:"-"`
}
type Document struct {
	Manifest   Manifest
	Records    []Record
	FileSHA256 string
}
type Diagnostic struct {
	Line int    `json:"line"`
	Path string `json:"path"`
	Code string `json:"code"`
}
type Invalid struct{ Diagnostics []Diagnostic }

func (e *Invalid) Error() string { return "invalid interchange document" }

// Validator runs after structural checks, without giving portability records
// authority. Store owners must revalidate roles/evidence/current config in-lock.
type Validator interface{ ValidateRecord(Record) []Diagnostic }
type PlanEntry struct {
	Kind       string `json:"kind"`
	ExternalID string `json:"external_id"`
	Action     string `json:"action"`
	Reason     string `json:"reason,omitempty"`
}
type Plan struct {
	FileSHA256    string      `json:"file_sha256"`
	ConfigVersion int64       `json:"config_version"`
	Create        int         `json:"create"`
	Update        int         `json:"update"`
	Unchanged     int         `json:"unchanged"`
	Conflicts     int         `json:"conflicts"`
	Entries       []PlanEntry `json:"entries"`
	Continuation  string      `json:"continuation,omitempty"`
}
type Result struct {
	Plan
	Applied bool `json:"applied"`
}

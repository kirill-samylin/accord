package core

import (
	"encoding/json"
	"time"
)

type ConfigView struct {
	ConfigVersion int64           `json:"config_version"`
	SHA256        string          `json:"sha256"`
	Config        json.RawMessage `json:"config"`
}
type ConfigRevision struct {
	ConfigView
	Actor     string    `json:"actor"`
	AppliedAt time.Time `json:"applied_at"`
}
type ConfigHistory struct {
	Revisions   []ConfigRevision `json:"revisions"`
	NextVersion int64            `json:"next_version"`
	HasMore     bool             `json:"has_more"`
}
type ScopesInput struct {
	ID              string
	Roles           []string
	ExpectedVersion int64
	Actor           string
}

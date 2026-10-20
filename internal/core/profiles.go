package core

import "time"

// ProfileSettings are saved preferences. They grant neither provider execution
// nor authorization; instruction text is data shared with project members.
type ProfileSettings struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Role         string `json:"role"`
	Instructions string `json:"instructions,omitempty"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
}
type AgentProfile struct {
	ProfileSettings
	ID        string    `json:"id"`
	State     string    `json:"state"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	CreatedBy string    `json:"created_by"`
	UpdatedBy string    `json:"updated_by"`
}
type ProfileSnapshot struct {
	ProfileSettings
	ID      string `json:"id"`
	Version int64  `json:"version"`
}
type ProfileRevision struct {
	Version   int64     `json:"version"`
	State     string    `json:"state"`
	UpdatedAt time.Time `json:"updated_at"`
	UpdatedBy string    `json:"updated_by"`
}
type ProfilesInput struct {
	State string `json:"state,omitempty"`
}
type ProfilesPage struct {
	Profiles []AgentProfile `json:"profiles"`
}
type ProfileInput struct {
	ProfileID string `json:"profile_id"`
	Version   int64  `json:"version,omitempty"`
}
type ProfilePage struct {
	Profile   AgentProfile      `json:"profile"`
	Revisions []ProfileRevision `json:"revisions"`
}
type ProfileCreateInput struct {
	RequestID string `json:"request_id"`
	ProfileSettings
}
type ProfileUpdateInput struct {
	RequestID string `json:"request_id"`
	ProfileID string `json:"profile_id"`
	Version   int64  `json:"version"`
	ProfileSettings
}
type ProfileArchiveInput struct {
	RequestID string `json:"request_id"`
	ProfileID string `json:"profile_id"`
	Version   int64  `json:"version"`
	State     string `json:"state"`
}

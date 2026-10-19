package core

// DecisionsInput pages immutable human answers newest first. BeforeCursor is
// exclusive; zero starts at the latest committed decision.
type DecisionsInput struct {
	EpicKey      string `json:"epic_key,omitempty"`
	BeforeCursor int64  `json:"before_cursor,omitempty"`
	Limit        int    `json:"limit,omitempty"`
}
type DecisionsPage struct {
	Events     []Event `json:"events"`
	NextCursor int64   `json:"next_cursor,omitempty"`
	HasMore    bool    `json:"has_more"`
}

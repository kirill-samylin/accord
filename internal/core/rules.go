package core

type RuleWarning struct {
	Rule      string   `json:"rule"`
	Message   string   `json:"message"`
	TaskKey   string   `json:"task_key,omitempty"`
	SessionID string   `json:"session_id,omitempty"`
	Owner     string   `json:"owner,omitempty"`
	Areas     []string `json:"areas,omitempty"`
}
type SimilarTask struct {
	Key   string  `json:"key"`
	Score float64 `json:"score"`
}

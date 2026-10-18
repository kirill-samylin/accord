package github

import (
	"regexp"
	"time"
)

type Policy struct {
	Repositories []string
	TaskPattern  *regexp.Regexp
	// KnownTasks maps normalized upper-case matches to existing canonical keys.
	KnownTasks map[string]string
	Bindings   []Binding
}
type Binding struct {
	Repository string     `json:"repository"`
	PR         int64      `json:"pr"`
	SHA        string     `json:"sha"`
	TaskKey    string     `json:"task_key"`
	SourceTime *time.Time `json:"source_time,omitempty"`
}
type Match struct {
	TaskKey string `json:"task_key,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

// Fact is immutable observed provider data. It cannot approve a task or prove
// completeness. The store persists historical versions and chooses pointers.
type Fact struct {
	Repository string     `json:"repository"`
	Kind       string     `json:"kind"`
	ObjectID   int64      `json:"object_id"`
	SHA        string     `json:"sha"`
	AppID      int64      `json:"app_id,omitempty"`
	PRs        []int64    `json:"prs,omitempty"`
	SourceTime *time.Time `json:"source_time,omitempty"`
	State      string     `json:"state"`
	Conclusion string     `json:"conclusion,omitempty"`
	URL        string     `json:"url,omitempty"`
	User       string     `json:"user,omitempty"`
	MergedAt   *time.Time `json:"merged_at,omitempty"`
	Hash       string     `json:"-"`
}
type Delivery struct {
	Receipt Receipt
	Outcome string
	Match   Match
	Facts   []Fact
	Binding *Binding
	// Completeness always remains unknown; no API backfill occurs here.
	Completeness string
}
type repository struct {
	FullName string `json:"full_name"`
}
type pullRequest struct {
	Number    int64      `json:"number"`
	State     string     `json:"state"`
	Body      string     `json:"body"`
	UpdatedAt *time.Time `json:"updated_at"`
	MergedAt  *time.Time `json:"merged_at"`
	Head      struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
	Base struct {
		Repo repository `json:"repo"`
	} `json:"base"`
}
type payload struct {
	Action     string       `json:"action"`
	Repository repository   `json:"repository"`
	Number     int64        `json:"number"`
	PR         *pullRequest `json:"pull_request"`
	Suite      *struct {
		ID          int64      `json:"id"`
		HeadSHA     string     `json:"head_sha"`
		HeadBranch  string     `json:"head_branch"`
		Status      string     `json:"status"`
		Conclusion  string     `json:"conclusion"`
		UpdatedAt   *time.Time `json:"updated_at"`
		CompletedAt *time.Time `json:"completed_at"`
		App         struct {
			ID int64 `json:"id"`
		} `json:"app"`
		PRs []struct {
			Number int64 `json:"number"`
		} `json:"pull_requests"`
	} `json:"check_suite"`
	Review *struct {
		ID          int64      `json:"id"`
		CommitID    string     `json:"commit_id"`
		State       string     `json:"state"`
		SubmittedAt *time.Time `json:"submitted_at"`
		User        struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"review"`
}

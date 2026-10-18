package github

import (
	"agent-board/internal/interchange"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

var repoPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*/[A-Za-z0-9_.-]+$`)
var shaPattern = regexp.MustCompile(`^(?:[0-9a-fA-F]{40}|[0-9a-fA-F]{64})$`)

func CanonicalRepository(repo string) (string, bool) {
	repo = strings.TrimPrefix(repo, "github.com/")
	if !repoPattern.MatchString(repo) || len(repo) > 245 || strings.HasSuffix(strings.ToLower(repo), ".git") {
		return "", false
	}
	return "github.com/" + strings.ToLower(repo), true
}
func bad(code string) error { return &Error{422, code} }

// Decode accepts unknown provider fields after duplicate-key validation, but only
// emits explicit immutable fact fields. Verify must precede it in ingress code.
func Decode(e Envelope, p Policy) (Delivery, error) {
	d := Delivery{Receipt: Receipt{e.DeliveryID, e.Event, e.BodySHA256}, Completeness: "unknown"}
	if len(e.Raw) > MaxBody || interchange.CheckJSON(e.Raw, 32) != nil || !bytes.HasPrefix(bytes.TrimSpace(e.Raw), []byte("{")) {
		return d, bad("JSON")
	}
	var v payload
	if json.Unmarshal(e.Raw, &v) != nil {
		return d, bad("PAYLOAD")
	}
	if e.Event == "ping" {
		d.Outcome = "ping"
		return d, nil
	}
	if (e.Event == "pull_request" || e.Event == "check_suite" || e.Event == "pull_request_review") && v.Action == "" {
		return d, bad("ACTION")
	}
	supported := (e.Event == "pull_request" && (v.Action == "opened" || v.Action == "synchronize" || v.Action == "closed")) || (e.Event == "check_suite" && v.Action == "completed") || (e.Event == "pull_request_review" && v.Action == "submitted")
	if !supported {
		d.Outcome = "ignored"
		return d, nil
	}
	repo, ok := CanonicalRepository(v.Repository.FullName)
	if !ok {
		return d, bad("REPOSITORY")
	}
	allowed := false
	for _, r := range p.Repositories {
		c, ok := CanonicalRepository(r)
		if ok && c == repo {
			allowed = true
		}
	}
	if !allowed {
		return d, &Error{403, "REPOSITORY_DENIED"}
	}
	var fact Fact
	var match Match
	switch e.Event {
	case "pull_request", "pull_request_review":
		pr := v.PR
		if pr == nil || pr.Number <= 0 || !shaPattern.MatchString(pr.Head.SHA) || pr.UpdatedAt == nil || pr.UpdatedAt.IsZero() {
			return d, bad("PULL_REQUEST")
		}
		base, ok := CanonicalRepository(pr.Base.Repo.FullName)
		if !ok || base != repo {
			return d, bad("BASE_REPOSITORY")
		}
		if pr.State != "open" && pr.State != "closed" {
			return d, bad("PR_STATE")
		}
		if e.Event == "pull_request" && (v.Number <= 0 || v.Number != pr.Number) {
			return d, bad("PR_NUMBER")
		}
		match = MatchTask(p, pr.Head.Ref, pr.Body)
		sha := strings.ToLower(pr.Head.SHA)
		fact = Fact{Repository: repo, Kind: "pull_request", ObjectID: pr.Number, SHA: sha, SourceTime: pr.UpdatedAt, State: pr.State, MergedAt: pr.MergedAt, URL: "https://" + repo + "/pull/" + strconv.FormatInt(pr.Number, 10)}
		if match.TaskKey != "" {
			d.Binding = &Binding{repo, pr.Number, sha, match.TaskKey, pr.UpdatedAt}
		}
		if e.Event == "pull_request_review" {
			review := v.Review
			if review == nil || review.ID <= 0 || !shaPattern.MatchString(review.CommitID) || review.SubmittedAt == nil || review.SubmittedAt.IsZero() || review.User.Login == "" {
				return d, bad("REVIEW")
			}
			switch review.State {
			case "approved", "changes_requested", "commented", "dismissed", "pending":
			default:
				return d, bad("REVIEW_STATE")
			}
			fact = Fact{Repository: repo, Kind: "review", ObjectID: review.ID, SHA: strings.ToLower(review.CommitID), PRs: []int64{pr.Number}, SourceTime: review.SubmittedAt, State: review.State, User: review.User.Login}
			// A review observes the PR but cannot advance its head/binding pointer.
			d.Binding = nil
		}
	case "check_suite":
		suite := v.Suite
		if suite == nil || suite.ID <= 0 || suite.App.ID <= 0 || !shaPattern.MatchString(suite.HeadSHA) || suite.Status != "completed" {
			return d, bad("CHECK_SUITE")
		}
		switch suite.Conclusion {
		case "success", "failure", "neutral", "cancelled", "timed_out", "action_required", "stale", "startup_failure", "skipped":
		default:
			return d, bad("CONCLUSION")
		}
		prs := []int64{}
		wanted := map[int64]bool{}
		keys := map[string]bool{}
		for _, pr := range suite.PRs {
			if pr.Number <= 0 {
				return d, bad("PR_NUMBER")
			}
			if !wanted[pr.Number] {
				prs = append(prs, pr.Number)
				wanted[pr.Number] = true
			}
		}
		for _, b := range p.Bindings {
			if b.Repository == repo && wanted[b.PR] && strings.EqualFold(b.SHA, suite.HeadSHA) {
				keys[strings.ToUpper(b.TaskKey)] = true
			}
		}
		sort.Slice(prs, func(i, j int) bool { return prs[i] < prs[j] })
		branchKeys, bounded := extractKeys(p, suite.HeadBranch)
		for key := range branchKeys {
			keys[key] = true
		}
		match = classifyKeys(keys, p)
		if !bounded {
			match = Match{Reason: "ambiguous_key"}
		}
		at := suite.UpdatedAt
		if at == nil {
			at = suite.CompletedAt
		}
		fact = Fact{Repository: repo, Kind: "check_suite", ObjectID: suite.ID, SHA: strings.ToLower(suite.HeadSHA), AppID: suite.App.ID, PRs: prs, SourceTime: at, State: suite.Status, Conclusion: suite.Conclusion}
	}
	fact.SourceTime = utcTime(fact.SourceTime)
	fact.MergedAt = utcTime(fact.MergedAt)
	if d.Binding != nil {
		d.Binding.SourceTime = utcTime(d.Binding.SourceTime)
	}
	raw, err := json.Marshal(fact)
	if err != nil {
		return d, bad("FACT")
	}
	sum := sha256.Sum256(raw)
	fact.Hash = hex.EncodeToString(sum[:])
	d.Match = match
	d.Facts = []Fact{fact}
	d.Outcome = "matched"
	if match.TaskKey == "" {
		d.Outcome = "unmatched"
	}
	return d, nil
}
func MatchTask(p Policy, inputs ...string) Match {
	keys, bounded := extractKeys(p, inputs...)
	if !bounded {
		return Match{Reason: "ambiguous_key"}
	}
	return classifyKeys(keys, p)
}
func extractKeys(p Policy, inputs ...string) (map[string]bool, bool) {
	keys := map[string]bool{}
	if p.TaskPattern == nil {
		return keys, true
	}
	for _, input := range inputs {
		if len(input) > MaxBody {
			return keys, false
		}
		matches := p.TaskPattern.FindAllString(input, 1025)
		if len(matches) > 1024 {
			return keys, false
		}
		for _, key := range matches {
			keys[strings.ToUpper(key)] = true
		}
	}
	return keys, true
}
func classifyKeys(keys map[string]bool, p Policy) Match {
	if len(keys) == 0 {
		return Match{Reason: "missing_key"}
	}
	if len(keys) > 1 {
		return Match{Reason: "ambiguous_key"}
	}
	for key := range keys {
		if canonical, ok := p.KnownTasks[strings.ToUpper(key)]; ok {
			return Match{TaskKey: canonical}
		}
	}
	return Match{Reason: "unknown_task"}
}

func utcTime(at *time.Time) *time.Time {
	if at == nil {
		return nil
	}
	value := at.UTC()
	return &value
}

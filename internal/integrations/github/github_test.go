package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"
)

var testSecret = []byte(strings.Repeat("s", 32))

const testID = "12345678-1234-1234-1234-123456789abc"

func headers(raw []byte, event string) http.Header {
	mac := hmac.New(sha256.New, testSecret)
	mac.Write(raw)
	return http.Header{"Content-Type": []string{"application/json"}, "X-Hub-Signature-256": []string{"sha256=" + hex.EncodeToString(mac.Sum(nil))}, "X-Github-Delivery": []string{testID}, "X-Github-Event": []string{event}}
}
func fixturePR(branch, body, sha string) string {
	return `{"action":"opened","number":4,"repository":{"full_name":"Owner/Repo"},"pull_request":{"number":4,"state":"open","updated_at":"2026-01-01T00:00:00Z","body":"` + body + `","head":{"sha":"` + sha + `","ref":"` + branch + `","repo":{"full_name":"fork/code"}},"base":{"repo":{"full_name":"Owner/Repo"}}}}`
}
func policy() Policy {
	return Policy{Repositories: []string{"github.com/owner/repo"}, TaskPattern: regexp.MustCompile(`(?i)T-[0-9]+`), KnownTasks: map[string]string{"T-1": "T-1", "T-2": "T-2"}}
}
func decode(t *testing.T, raw, event string, p Policy) (Delivery, error) {
	t.Helper()
	e, err := Verify(headers([]byte(raw), event), []byte(raw), testSecret)
	if err != nil {
		t.Fatal(err)
	}
	return Decode(e, p)
}
func TestVerifyRawBytesAndHeaders(t *testing.T) {
	raw := []byte(`{"body":"Привет"}`)
	h := headers(raw, "ping")
	if _, err := Verify(h, raw, testSecret); err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(h, append(raw, ' '), testSecret); err == nil {
		t.Fatal("byte mutation accepted")
	}
	for _, name := range []string{"X-Hub-Signature-256", "X-Github-Delivery", "X-Github-Event"} {
		dup := h.Clone()
		dup.Add(name, dup.Get(name))
		if _, err := Verify(dup, raw, testSecret); err == nil {
			t.Fatal("duplicate", name)
		}
		missing := h.Clone()
		missing.Del(name)
		if _, err := Verify(missing, raw, testSecret); err == nil {
			t.Fatal("missing", name)
		}
	}
	mixed := h.Clone()
	mixed["x-hub-signature-256"] = []string{h.Get("X-Hub-Signature-256")}
	if _, err := Verify(mixed, raw, testSecret); err == nil {
		t.Fatal("case bypass")
	}
	h.Set("Content-Encoding", "gzip")
	if _, err := Verify(h, raw, testSecret); err == nil {
		t.Fatal("compression")
	}
	if _, err := Verify(headers(make([]byte, MaxBody+1), "ping"), make([]byte, MaxBody+1), testSecret); err == nil {
		t.Fatal("body bounds")
	}
}
func TestPRAllowlistMatchAndFacts(t *testing.T) {
	raw := fixturePR("feature/T-1", "T-1", strings.Repeat("a", 40))
	d, err := decode(t, raw, "pull_request", policy())
	if err != nil {
		t.Fatal(err)
	}
	if d.Match.TaskKey != "T-1" || d.Binding == nil || d.Facts[0].URL != "https://github.com/owner/repo/pull/4" || d.Completeness != "unknown" {
		t.Fatalf("wrong normalization: %+v", d)
	}
	for _, pair := range []struct{ branch, body, reason string }{{"none", "", "missing_key"}, {"T-9", "", "unknown_task"}, {"T-1", "T-2", "ambiguous_key"}} {
		d, err := decode(t, fixturePR(pair.branch, pair.body, strings.Repeat("b", 64)), "pull_request", policy())
		if err != nil || d.Match.Reason != pair.reason || len(d.Facts) != 1 {
			t.Fatalf("match %+v %v", d, err)
		}
	}
	p := policy()
	p.Repositories = nil
	if _, err := decode(t, raw, "pull_request", p); err == nil {
		t.Fatal("allowlist bypass")
	}
	for _, broken := range []string{strings.Replace(raw, `"Owner/Repo"}}}}`, `"attacker/repo"}}}}`, 1), strings.Replace(raw, `"number":4`, `"number":5`, 1), strings.Replace(raw, `"body":"T-1"`, `"body":"T-1","body":"T-2"`, 1)} {
		if _, err := decode(t, broken, "pull_request", policy()); err == nil {
			t.Fatal("broken accepted", broken)
		}
	}
}
func TestReviewCIAndReplayOrdering(t *testing.T) {
	sha := strings.Repeat("c", 40)
	raw := fixturePR("T-1", "", sha)
	review := strings.TrimSuffix(strings.Replace(raw, `"action":"opened"`, `"action":"submitted"`, 1), "}") + `,"review":{"id":9007199254740993,"commit_id":"` + sha + `","state":"approved","submitted_at":"2026-01-01T00:00:00Z","user":{"login":"fixture"}}}`
	d, err := decode(t, review, "pull_request_review", policy())
	if err != nil {
		t.Fatal(err)
	}
	if d.Facts[0].ObjectID != 9007199254740993 || d.Binding != nil || d.Facts[0].Kind != "review" {
		t.Fatal("review gained authority")
	}
	suite := `{"action":"completed","repository":{"full_name":"Owner/Repo"},"check_suite":{"id":8,"head_sha":"` + sha + `","head_branch":"","app":{"id":5},"status":"completed","conclusion":"success","pull_requests":[{"number":4}]}}`
	p := policy()
	p.Bindings = []Binding{{Repository: "github.com/owner/repo", PR: 4, SHA: sha, TaskKey: "T-1"}}
	ci, err := decode(t, suite, "check_suite", p)
	if err != nil || ci.Match.TaskKey != "T-1" || ci.Facts[0].SourceTime != nil {
		t.Fatal(ci, err)
	}
	p.Bindings[0].SHA = strings.Repeat("d", 40)
	ci, err = decode(t, suite, "check_suite", p)
	if err != nil || ci.Match.TaskKey != "" {
		t.Fatal("cross-SHA binding")
	}
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	earlier := at.Add(-time.Second)
	if CompareVersions("a", &at, "b", &earlier) != Older || CompareVersions("a", &at, "b", &at) != OrderConflict || CompareVersions("a", nil, "b", nil) != OrderConflict {
		t.Fatal("arrival ordering")
	}
	r := Receipt{testID, "pull_request", "hash"}
	other := r
	other.DeliveryID = "new"
	if ClassifyReplay(r, other) != Duplicate {
		t.Fatal("new GUID replay")
	}
	other = r
	other.BodySHA256 = "changed"
	if ClassifyReplay(r, other) != ReplayConflict {
		t.Fatal("conflict")
	}
}

func TestIgnoredPingMissingActionAndMatcherBound(t *testing.T) {
	for _, event := range []string{"ping", "unknown_event"} {
		d, err := decode(t, `{}`, event, policy())
		if err != nil || len(d.Facts) != 0 {
			t.Fatal(d, err)
		}
	}
	if _, err := decode(t, `{}`, "pull_request", policy()); err == nil {
		t.Fatal("missing action accepted")
	}
	if _, err := decode(t, `null`, "ping", policy()); err == nil {
		t.Fatal("null payload accepted")
	}
	if MatchTask(policy(), strings.Repeat("T-1 ", 1025)+"T-2").Reason != "ambiguous_key" {
		t.Fatal("matcher truncation")
	}
	sha := strings.Repeat("e", 40)
	suite := `{"action":"completed","repository":{"full_name":"owner/repo"},"check_suite":{"id":1,"head_sha":"` + sha + `","head_branch":"T-9","app":{"id":1},"status":"completed","conclusion":"success","pull_requests":[]}}`
	d, err := decode(t, suite, "check_suite", policy())
	if err != nil || d.Match.Reason != "unknown_task" {
		t.Fatal("unknown branch misclassified", d, err)
	}
}

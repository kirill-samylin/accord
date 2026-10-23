package web

import (
	"agent-board/internal/core"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func automationFormRequest(fields url.Values) *http.Request {
	r := httptest.NewRequest("POST", "https://accord.test/web/automation/command", strings.NewReader(fields.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_ = r.ParseForm()
	return r
}
func TestAutomationFormsRejectAmbiguousAuthorityAndPreserveCAS(t *testing.T) {
	values := url.Values{"workflow_id": {"wf_example"}, "version": {"7"}, "kind": {"resume"}, "refresh_policy_id": {"apolicy_current"}, "refresh_policy_version": {"2"}}
	in, err := automationForm(automationFormRequest(values), "automation:command", "issued-action")
	if err != nil {
		t.Fatal(err)
	}
	command := in.(core.WorkflowCommandInput)
	if command.Version != 7 || command.RefreshPolicyVersion != 2 || command.RequestID != "issued-action" {
		t.Fatal("lost exact authority/version binding")
	}
	for _, key := range []string{"approval", "source", "host_secret", "principal_id", "epoch", "session_id", "request_id"} {
		forged := cloneAutomationValues(values)
		forged.Set(key, "forged")
		if _, err = automationForm(automationFormRequest(forged), "automation:command", "issued-action"); err == nil {
			t.Fatalf("accepted %s", key)
		}
	}
	duplicate := cloneAutomationValues(values)
	duplicate.Add("version", "8")
	if _, err = automationForm(automationFormRequest(duplicate), "automation:command", "issued-action"); err == nil {
		t.Fatal("accepted duplicate version")
	}
	missing := cloneAutomationValues(values)
	missing.Del("version")
	if _, err = automationForm(automationFormRequest(missing), "automation:command", "issued-action"); err == nil {
		t.Fatal("accepted missing CAS")
	}
}
func TestAutomationPolicyJSONRejectsDuplicateUnknownNullAndTrailingValues(t *testing.T) {
	for _, raw := range []string{`{"name":"one","name":"two"}`, `{"role_bindings":{"analyst":{"provider":"codex","provider":"claude"}}}`, `{"name":null}`, `{"approved":true}`, `{} {}`, `[]`, strings.Repeat("[", 40) + strings.Repeat("]", 40)} {
		var settings core.AutomationPolicySettings
		if err := decodeAutomationSettings(raw, &settings); err == nil {
			t.Fatalf("accepted ambiguous settings %q", raw)
		}
	}
	var settings core.AutomationPolicySettings
	if err := decodeAutomationSettings(`{"name":"example","client_intake":false,"git_policy":"local_commits","role_bindings":{},"checks":[]}`, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.ClientIntake || settings.GitPolicy != "local_commits" {
		t.Fatal("settings changed")
	}
}
func TestAutomationDraftAndAnswerFormsRequireExplicitSemantics(t *testing.T) {
	in, err := automationForm(automationFormRequest(url.Values{"text": {"Read requirements"}}), "automation:submit", "draft-action")
	if err != nil {
		t.Fatal(err)
	}
	draft := in.(core.WorkflowSubmitInput)
	if draft.PolicyID != "" || draft.PolicyVersion != 0 || draft.RequestID != "draft-action" {
		t.Fatal("draft acquired a grant")
	}
	values := url.Values{"workflow_id": {"wf_example"}, "version": {"3"}, "question_id": {"wq_example"}, "question_version": {"2"}, "answer": {"Clarification"}}
	if _, err = automationForm(automationFormRequest(values), "automation:answer", "answer-action"); err == nil {
		t.Fatal("implicit resume accepted")
	}
	values.Set("resume", "false")
	in, err = automationForm(automationFormRequest(values), "automation:answer", "answer-action")
	if err != nil {
		t.Fatal(err)
	}
	answer := in.(core.WorkflowAnswerInput)
	if answer.Resume || answer.QuestionVersion != 2 || answer.Version != 3 {
		t.Fatal("answer lost expected versions")
	}
}

func cloneAutomationValues(in url.Values) url.Values {
	out := url.Values{}
	for key, values := range in {
		out[key] = append([]string(nil), values...)
	}
	return out
}

package web

import (
	"agent-board/internal/core"
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var automationOperations = map[string]string{
	"/web/automation/submit": "automation:submit", "/web/automation/command": "automation:command", "/web/automation/answer": "automation:answer",
	"/web/automation/policies/create": "automation:policy-create", "/web/automation/policies/update": "automation:policy-update", "/web/automation/policies/revoke": "automation:policy-revoke",
}

func automationForm(r *http.Request, op, action string) (any, error) {
	allowed := "csrf action_id "
	switch op {
	case "automation:submit":
		allowed += "policy_id policy_version text"
	case "automation:command":
		allowed += "workflow_id version kind text refresh_policy_id refresh_policy_version"
	case "automation:answer":
		allowed += "workflow_id version question_id question_version answer resume"
	case "automation:policy-create":
		allowed += "settings"
	case "automation:policy-update":
		allowed += "settings policy_id version"
	case "automation:policy-revoke":
		allowed += "policy_id version reason"
	default:
		return nil, core.Fail("validation", "Неизвестная операция")
	}
	if r.URL.RawQuery != "" {
		return nil, core.Fail("validation", "Параметры URL запрещены")
	}
	for key, values := range r.PostForm {
		if len(values) != 1 || !strings.Contains(" "+allowed+" ", " "+key+" ") {
			return nil, core.Fail("validation", "Неизвестное или повторное поле формы")
		}
	}
	number := func(key string, optional bool) (int64, error) {
		raw := r.PostForm.Get(key)
		if optional && raw == "" {
			return 0, nil
		}
		value, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || value < 1 {
			return 0, core.Fail("validation", "Нужна положительная ожидаемая версия")
		}
		return value, nil
	}
	version, err := number("version", op == "automation:submit" || op == "automation:policy-create")
	if err != nil {
		return nil, err
	}
	switch op {
	case "automation:submit":
		pv, e := number("policy_version", r.PostForm.Get("policy_id") == "")
		if e != nil {
			return nil, e
		}
		return core.WorkflowSubmitInput{RequestID: action, PolicyID: r.PostForm.Get("policy_id"), PolicyVersion: pv, Text: r.PostForm.Get("text")}, nil
	case "automation:command":
		pv, e := number("refresh_policy_version", r.PostForm.Get("refresh_policy_id") == "")
		if e != nil {
			return nil, e
		}
		return core.WorkflowCommandInput{RequestID: action, WorkflowID: r.PostForm.Get("workflow_id"), Version: version, Kind: r.PostForm.Get("kind"), Text: r.PostForm.Get("text"), RefreshPolicyID: r.PostForm.Get("refresh_policy_id"), RefreshPolicyVersion: pv}, nil
	case "automation:answer":
		qv, e := number("question_version", false)
		if e != nil {
			return nil, e
		}
		resume := r.PostForm.Get("resume")
		if resume != "true" && resume != "false" {
			return nil, core.Fail("validation", "Укажите, продолжать ли работу")
		}
		return core.WorkflowAnswerInput{RequestID: action, WorkflowID: r.PostForm.Get("workflow_id"), Version: version, QuestionID: r.PostForm.Get("question_id"), QuestionVersion: qv, Answer: r.PostForm.Get("answer"), Resume: resume == "true"}, nil
	case "automation:policy-revoke":
		return core.AutomationPolicyRevokeInput{RequestID: action, PolicyID: r.PostForm.Get("policy_id"), Version: version, Reason: r.PostForm.Get("reason")}, nil
	default:
		var settings core.AutomationPolicySettings
		if e := decodeAutomationSettings(r.PostForm.Get("settings"), &settings); e != nil {
			return nil, e
		}
		create := core.AutomationPolicyCreateInput{RequestID: action, AutomationPolicySettings: settings}
		if op == "automation:policy-create" {
			return create, nil
		}
		return core.AutomationPolicyUpdateInput{AutomationPolicyCreateInput: create, PolicyID: r.PostForm.Get("policy_id"), Version: version}, nil
	}
}
func decodeAutomationSettings(raw string, dest *core.AutomationPolicySettings) error {
	if len(raw) == 0 || len(raw) > 49152 || !strings.HasPrefix(strings.TrimSpace(raw), "{") {
		return core.Fail("validation", "Нужны настройки разрешения")
	}
	if err := automationUniqueJSON([]byte(raw)); err != nil {
		return err
	}
	d := json.NewDecoder(bytes.NewBufferString(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(dest); err != nil {
		return core.Fail("validation", "Неверные настройки разрешения")
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return core.Fail("validation", "Лишние данные настроек")
	}
	return nil
}

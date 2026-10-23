package worker

import (
	"net/url"
	"os"
	"regexp"
	"strings"
)

var ansi = regexp.MustCompile("\x1b(?:\\[[0-?]*[ -/]*[@-~]|\\][^\x07]*(?:\x07|\x1b\\\\))")
var urlPattern = regexp.MustCompile(`https?://[^\s<>]+`)
var executionSecret = regexp.MustCompile(`ac_exec1_[a-f0-9]{64}`)

var labeledSecret = regexp.MustCompile(`(?i)(bearer\s+|(?:token|api[_-]?key|secret|password)\s*[:=]\s*)[^\s,;]+`)

func cleanOutput(text string, secrets ...string) string {
	text = ansi.ReplaceAllString(text, "")
	text = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' || r == 127 {
			return -1
		}
		return r
	}, text)
	for _, secret := range secrets {
		if secret != "" {
			text = strings.ReplaceAll(text, secret, "[redacted]")
		}
	}
	text = executionSecret.ReplaceAllString(text, "[redacted execution credential]")
	text = labeledSecret.ReplaceAllString(text, "${1}[redacted]")
	text = urlPattern.ReplaceAllStringFunc(text, func(value string) string {
		parsed, err := url.Parse(value)
		if err != nil {
			return "[redacted URL]"
		}
		parsed.User = nil
		query := parsed.Query()
		for key := range query {
			lower := strings.ToLower(key)
			if strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "key") || strings.Contains(lower, "auth") || lower == "sig" || lower == "signature" || lower == "password" || lower == "code" {
				query.Set(key, "[redacted]")
			}
		}
		parsed.RawQuery = query.Encode()
		return parsed.String()
	})
	if len(text) > 8192 {
		return boundedText(text, 8160) + "\n[output truncated]"
	}
	return text
}

func (e *Engine) outputSecrets() []string {
	secrets := []string{e.journal.HostSecret}
	for _, run := range e.journal.Runs {
		if run.Automation != nil && run.Automation.IPC != nil {
			secrets = append(secrets, run.Automation.IPC.Nonce)
		}
	}
	if token, err := ReadToken(e.config.TokenFile); err == nil {
		secrets = append(secrets, token)
	}
	for _, entry := range os.Environ() {
		key, value, _ := strings.Cut(entry, "=")
		upper := strings.ToUpper(key)
		if strings.Contains(upper, "API_KEY") || strings.Contains(upper, "TOKEN") || strings.Contains(upper, "SECRET") {
			if len(value) >= 8 {
				secrets = append(secrets, value)
			}
		}
	}
	return secrets
}

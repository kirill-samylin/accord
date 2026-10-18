package config

import (
	"fmt"
	"net/url"
	"strings"
)

func validateBuiltinValues(value map[string]any) error {
	for key, limit := range map[string]int{"summary": 4096, "run_summary": 1024, "findings": 16384, "answer": 16384, "run_id": 128, "pr": 2048} {
		if text, ok := value[key].(string); ok && (len(text) > limit || strings.TrimSpace(text) == "") {
			return fmt.Errorf("built-in evidence string exceeds byte limit")
		}
	}
	if text, ok := value["pr"].(string); ok {
		u, err := url.Parse(text)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
			return fmt.Errorf("PR must be an absolute HTTPS URL without credentials")
		}
	}
	for _, key := range []string{"artifact", "screens", "refs"} {
		if refs, ok := value[key].([]any); ok {
			for _, item := range refs {
				text, ok := item.(string)
				if !ok || strings.TrimSpace(text) == "" || len(text) > 2048 {
					return fmt.Errorf("evidence reference must be nonempty and at most 2048 bytes")
				}
			}
		}
	}
	return nil
}

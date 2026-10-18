package config

import (
	"agent-board/internal/integrations/github"
	"agent-board/internal/notifications"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

func validateIntegrations(c *Config, o Options, r *Result) {
	if g := c.GitHub; g != nil {
		if !o.GitHubEvents && len(g.Events) > 0 {
			add(r, "github.events", "CONFIG_UNSUPPORTED", "GitHub events are not implemented")
		}
		if len(g.Events) > 0 && len(g.RepositoryAllowlist) == 0 {
			add(r, "github.repository_allowlist", "CONFIG_INVALID", "enabled events require an explicit repository allowlist")
		}
		seen := map[string]bool{}
		for i, value := range g.RepositoryAllowlist {
			// Provider payloads accept owner/repo; configuration must name the host.
			canonical, valid := github.CanonicalRepository(value)
			if !strings.HasPrefix(value, "github.com/") || len(value) > 256 || !valid || seen[canonical] {
				add(r, "github.repository_allowlist", "CONFIG_INVALID", "repositories must be unique canonical GitHub identities")
				continue
			}
			seen[canonical] = true
			g.RepositoryAllowlist[i] = canonical
		}
		if _, err := regexp.Compile(g.TaskKeyPattern); err != nil {
			add(r, "github.task_key_pattern", "CONFIG_INVALID", "invalid Go regular expression")
		}
	}
	if n := c.Notifications; n != nil {
		if !o.Notifications && len(n.Webhooks) > 0 {
			add(r, "notifications.webhooks", "CONFIG_UNSUPPORTED", "notifications are not implemented")
		}
		seen := map[string]bool{}
		for i, w := range n.Webhooks {
			canonical, id, err := canonicalEndpoint(w)
			if err != nil {
				add(r, "notifications.webhooks.url", "CONFIG_INVALID", "webhook URL violates the public HTTPS destination policy")
				continue
			}
			if seen[id] {
				add(r, "notifications.webhooks", "CONFIG_INVALID", "duplicate notification endpoint identity")
			}
			seen[id] = true
			n.Webhooks[i].URL = canonical
		}
	}
}

// Endpoint identity includes the env NAME, never its value. A root path and
// implicit HTTPS port have one identity, irrespective of event subscriptions.
func canonicalEndpoint(w Webhook) (string, string, error) {
	u, err := notifications.ValidateURL(w.URL)
	if err != nil {
		return "", "", err
	}
	if u.Path == "" {
		u.Path = "/"
	}
	canonical := u.String()
	sum := sha256.Sum256([]byte(canonical + "\x00" + w.SecretEnv))
	return canonical, hex.EncodeToString(sum[:]), nil
}

package config

import (
	"encoding/json"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
)

var roleName = regexp.MustCompile(`^[a-z][a-z0-9-]{0,47}$`)

func add(r *Result, path, code, message string) {
	r.Errors = append(r.Errors, Problem{path, code, message})
}
func validateConfig(c *Config, o Options, r *Result) {
	users := map[string]bool{}
	for _, u := range c.Auth.Local.Users {
		if strings.TrimSpace(u) == "" || u == "OWNER" {
			add(r, "auth.local.users", "CONFIG_INVALID", "local users must be nonempty resolved logins")
		}
		users[u] = true
	}
	if len(users) == 0 {
		add(r, "auth.local.users", "CONFIG_INVALID", "at least one local user is required")
	}
	if strings.TrimSpace(c.Project.Name) == "" {
		add(r, "project.name", "CONFIG_INVALID", "project name cannot be blank")
	}
	if c.Auth.Mode == "github" && !o.GitHubAuth {
		add(r, "auth.mode", "CONFIG_UNSUPPORTED", "GitHub authentication is not implemented by this binary")
	}
	if c.Auth.GitHub != nil && !o.GitHubAuth {
		add(r, "auth.github", "CONFIG_UNSUPPORTED", "GitHub authentication settings are not implemented")
	}
	if c.GitHub != nil {
		if !o.GitHubEvents && len(c.GitHub.Events) > 0 {
			add(r, "github.events", "CONFIG_UNSUPPORTED", "GitHub events are not implemented")
		}
		if _, err := regexp.Compile(c.GitHub.TaskKeyPattern); err != nil {
			add(r, "github.task_key_pattern", "CONFIG_INVALID", "invalid Go regular expression")
		}
	}
	if c.Notifications != nil {
		if !o.Notifications && len(c.Notifications.Webhooks) > 0 {
			add(r, "notifications.webhooks", "CONFIG_UNSUPPORTED", "notifications are not implemented")
		}
		for _, w := range c.Notifications.Webhooks {
			u, err := url.Parse(w.URL)
			if err != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Fragment != "" {
				add(r, "notifications.webhooks.url", "CONFIG_INVALID", "webhook URL must be absolute HTTPS without credentials or fragment")
			}
		}
	}
	validateDurations(c, r)
	for _, key := range []string{"one_task_per_agent", "cycle_check"} {
		mode := c.Rules.OneTaskPerAgent.Mode
		if key == "cycle_check" {
			mode = c.Rules.CycleCheck.Mode
		}
		if mode != "block" {
			add(r, "rules."+key, "CONFIG_UNSUPPORTED", "safety rule must remain block")
		}
	}
	singleton := false
	names := make([]string, 0, len(c.Roles))
	for name := range c.Roles {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		role := c.Roles[name]
		p := "roles." + name
		singleton = singleton || role.Singleton
		if !roleName.MatchString(name) {
			add(r, p, "CONFIG_INVALID", "invalid role name")
		}
		wantHuman := name == "human" || name == "design-owner"
		if _, builtin := builtinFloors[name]; builtin && ((wantHuman && role.Kind != "human") || (!wantHuman && role.Kind != "agent")) {
			add(r, p+".kind", "CONFIG_UNSUPPORTED", "built-in role kind cannot change")
		}
		for _, member := range role.Members {
			if !users[member] {
				add(r, p+".members", "CONFIG_INVALID", "member is not an allowed local user")
			}
		}
		for category, cat := range role.Categories {
			if strings.TrimSpace(category) == "" || len(category) > 128 {
				add(r, p+".categories", "CONFIG_INVALID", "invalid category name")
			}
			if string(cat.AnswerBy) != `"any"` {
				var permitted []string
				if json.Unmarshal(cat.AnswerBy, &permitted) == nil {
					if len(permitted) == 0 {
						add(r, p+".categories."+category, "CONFIG_INVALID", "answer_by list cannot be empty")
					}
					for _, u := range permitted {
						if !users[u] {
							add(r, p+".categories."+category, "CONFIG_INVALID", "answer_by references an unknown local user")
						}
					}
				}
			}
		}
		for _, target := range role.IndependentFrom {
			if _, ok := c.Roles[target]; !ok || target == name {
				add(r, p+".independent_from", "CONFIG_INVALID", "independence target must be another existing role")
			}
		}
		if err := prepareEvidence(name, &role); err != nil {
			add(r, p+".evidence_schema", "CONFIG_INVALID", err.Error())
		}
		c.Roles[name] = role
	}
	if singleton && c.Rules.SingletonRole.Mode != "block" {
		add(r, "rules.singleton_role", "CONFIG_UNSUPPORTED", "configured singleton roles require block")
	}
	for source, targets := range c.Reopen {
		if _, ok := c.Roles[source]; !ok {
			add(r, "reopen", "CONFIG_INVALID", "unknown source role")
		}
		for _, target := range targets {
			if target != "*" {
				if _, ok := c.Roles[target]; !ok {
					add(r, "reopen."+source, "CONFIG_INVALID", "unknown target role")
				}
			}
		}
	}
	for _, name := range c.Rules.EpicOwner.SharedRoles {
		if _, ok := c.Roles[name]; !ok {
			add(r, "rules.epic_owner.shared_roles", "CONFIG_INVALID", "unknown shared role")
		}
	}
	validateTools(c, o, r)
	validateAreas(c, r)
}
func validateDurations(c *Config, r *Result) {
	values := []struct {
		path  string
		value *string
		ttl   bool
	}{{"leases.task_ttl", &c.Leases.TaskTTL, true}, {"leases.role_ttl", &c.Leases.RoleTTL, true}, {"leases.session_ttl", &c.Leases.SessionTTL, true}, {"leases.max_silent", &c.Leases.MaxSilent, false}, {"leases.sweep_interval", &c.Leases.SweepInterval, false}, {"questions.escalate_after", &c.Questions.EscalateAfter, false}}
	durations := map[string]time.Duration{}
	for _, v := range values {
		d, err := time.ParseDuration(*v.value)
		if err != nil || d <= 0 || (v.ttl && (d < time.Second || d > 24*time.Hour)) {
			add(r, v.path, "CONFIG_INVALID", "duration must be positive and lease TTL must be 1s..24h")
			continue
		}
		*v.value = d.String()
		durations[v.path] = d
	}
	cap := min(durations["leases.task_ttl"], durations["leases.role_ttl"], durations["leases.session_ttl"])
	for _, name := range []string{"leases.max_silent", "leases.sweep_interval"} {
		if cap > 0 && durations[name] > cap {
			add(r, name, "CONFIG_INVALID", "duration exceeds the minimum lease TTL")
		}
	}
}
func validateTools(c *Config, o Options, r *Result) {
	descriptions := o.ToolDescriptions
	if descriptions == nil {
		descriptions = defaultDescriptions
	}
	for role, tools := range c.MCP.ToolsByRole {
		if _, ok := c.Roles[role]; !ok {
			add(r, "mcp.tools_by_role."+role, "CONFIG_INVALID", "unknown role")
		}
		total := 0
		for _, name := range tools {
			text, ok := descriptions[name]
			if !ok {
				add(r, "mcp.tools_by_role."+role, "CONFIG_UNSUPPORTED", "unknown or unavailable tool")
			}
			total += len(text)
		}
		if total > c.MCP.ToolBudgetTokens {
			add(r, "mcp.tools_by_role."+role, "CONFIG_INVALID", "conservative UTF-8 description bytes exceed tool budget")
		}
	}
	for role := range c.Roles {
		if _, ok := c.MCP.ToolsByRole[role]; !ok {
			add(r, "mcp.tools_by_role."+role, "CONFIG_INVALID", "explicit tool map must cover every role")
		}
	}
}

package worker

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"
)

// The local owner opts into specific bridge operations. The dashboard cannot
// set this map; unset and empty entries preserve the native permission policy.
var managedMCPTools = []string{"comment", "create", "describe", "events", "get", "heartbeat", "link", "next", "note", "release", "transition", "update"}

func validateNativeMCPConsent(grants map[string][]string, executables map[string]string) error {
	for provider, tools := range grants {
		if (provider != "codex" && provider != "claude") || executables[provider] == "" {
			return errors.New("native_mcp_approved_tools requires a configured codex or claude provider")
		}
		seen := map[string]bool{}
		for _, tool := range tools {
			known := false
			for _, allowed := range managedMCPTools {
				if tool == allowed {
					known = true
					break
				}
			}
			if !known || seen[tool] {
				return errors.New("native_mcp_approved_tools requires unique exact managed bridge tool names")
			}
			seen[tool] = true
		}
	}
	return nil
}

func codexMCPConsent(approved []string) string {
	if len(approved) == 0 {
		return ""
	}
	enabled, _ := json.Marshal(approved)
	selected := map[string]bool{}
	for _, tool := range approved {
		selected[tool] = true
	}
	var policies []string
	for _, tool := range managedMCPTools {
		mode := "prompt"
		if selected[tool] {
			mode = "approve"
		}
		policies = append(policies, strconv.Quote(tool)+"={approval_mode="+strconv.Quote(mode)+"}")
	}
	// Codex merges nested tables: reset known policies and restrict the catalog
	// so a stored approval for any unselected/unknown tool cannot expand this grant.
	return ",enabled_tools=" + string(enabled) + ",default_tools_approval_mode=\"prompt\",tools={" + strings.Join(policies, ",") + "}"
}

func claudeMCPConsent(approved []string) []string {
	if len(approved) == 0 {
		return nil
	}
	selected := map[string]bool{}
	var allow, deny []string
	for _, tool := range approved {
		selected[tool] = true
		allow = append(allow, "mcp__"+managedMCPName+"__"+tool)
	}
	for _, tool := range managedMCPTools {
		if !selected[tool] {
			deny = append(deny, "mcp__"+managedMCPName+"__"+tool)
		}
	}
	args := []string{"--allowedTools", strings.Join(allow, ",")}
	if len(deny) != 0 {
		args = append(args, "--disallowedTools", strings.Join(deny, ","))
	}
	return args
}

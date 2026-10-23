package worker

import (
	"encoding/json"
	"reflect"
	"strings"
)

// No arbitrary configuration strings are returned, persisted or formatted into
// errors. In particular layers may carry credentials unrelated to permissions.
func provePermissionConfig(raw, requirements json.RawMessage, i configInspection) (permissionProof, error) {
	var response struct {
		Config  map[string]any   `json:"config"`
		Layers  []map[string]any `json:"layers"`
		Origins map[string]any   `json:"origins"`
	}
	if json.Unmarshal(raw, &response) != nil || response.Config == nil || response.Layers == nil || response.Origins == nil || len(response.Layers) > 128 {
		return permissionProof{}, permissionError()
	}
	c := response.Config
	if unexpectedPermissionKey(c) {
		return permissionProof{}, permissionError()
	}
	// An effective legacy mode is ambiguous even when no explicit originating
	// layer was returned. Null typed defaults confer no legacy authority.
	for _, key := range []string{"sandbox_mode", "sandbox_workspace_write"} {
		if c[key] != nil {
			return permissionProof{}, permissionError()
		}
	}
	if c["default_permissions"] != i.Profile || c["approval_policy"] != "never" {
		return permissionProof{}, permissionError()
	}
	profiles, ok := c["permissions"].(map[string]any)
	if !ok || !reflect.DeepEqual(normalizeEffectiveProfile(profiles[i.Profile]), permissionProfile(i)) {
		return permissionProof{}, permissionError()
	}
	projection := []any{}
	known := map[string]bool{}
	for _, name := range i.Connectors {
		known[name] = true
	}
	mcp, ok := c["mcp_servers"].(map[string]any)
	if !ok {
		return permissionProof{}, permissionError()
	}
	if len(mcp) != len(known) {
		return permissionProof{}, permissionError()
	}
	for name, value := range mcp {
		entry, ok := value.(map[string]any)
		if !ok || !known[name] || entry["enabled"] != false {
			return permissionProof{}, permissionError()
		}
	}
	for _, layer := range response.Layers {
		disabled, hasDisabled := layer["disabledReason"]
		if hasDisabled && disabled != nil {
			if s, ok := disabled.(string); !ok || s == "" {
				return permissionProof{}, permissionError()
			}
			continue
		}
		name, ok := layer["name"].(map[string]any)
		if !ok {
			return permissionProof{}, permissionError()
		}
		typeName, ok := name["type"].(string)
		if !ok || !knownConfigLayer(typeName) {
			return permissionProof{}, permissionError()
		}
		if v, ok := layer["version"].(string); !ok || v == "" {
			return permissionProof{}, permissionError()
		}
		config, ok := layer["config"].(map[string]any)
		if !ok {
			return permissionProof{}, permissionError()
		}
		if unexpectedPermissionKey(config) {
			return permissionProof{}, permissionError()
		}
		for _, key := range []string{"sandbox_mode", "sandbox_workspace_write"} {
			if value, present := config[key]; present && value != nil {
				return permissionProof{}, permissionError()
			}
		}
		selected := map[string]any{}
		for _, key := range []string{"approval_policy", "approvals_reviewer", "default_permissions", "sandbox_mode", "sandbox_workspace_write", "include_permissions_instructions"} {
			if v, present := config[key]; present && v != nil {
				selected[key] = v
			}
		}
		if value, present := config["permissions"]; present {
			p, ok := value.(map[string]any)
			if !ok {
				return permissionProof{}, permissionError()
			}
			for name := range p {
				if strings.HasPrefix(name, ":") {
					return permissionProof{}, permissionError()
				}
			}
			if v, present := p[i.Profile]; present {
				selected["selected_profile"] = v
			}
		}
		if value, present := config["mcp_servers"]; present {
			servers, ok := value.(map[string]any)
			if !ok {
				return permissionProof{}, permissionError()
			}
			for name := range servers {
				if !known[name] {
					return permissionProof{}, permissionError()
				}
			}
		}
		if len(selected) > 0 {
			projection = append(projection, map[string]any{"source": typeName, "policy": selected})
		}
	}
	// Config is the actual effective tree, not reconstructed from our intended
	// flags. A CLI omitting the additional-properties tree is unsupported.
	if len(projection) == 0 {
		return permissionProof{}, permissionError()
	}
	managed, err := provePermissionRequirements(requirements, i.Profile)
	if err != nil {
		return permissionProof{}, err
	}
	return permissionProof{Digest: permissionDigest([]any{projection, permissionProfile(i), managed, i.Connectors}), ExecutableSHA256: i.ExecutableSHA256, GitIdentity: i.GitIdentity}, nil
}
func knownConfigLayer(name string) bool {
	switch name {
	case "packagedDefaults", "mdm", "system", "enterpriseManaged", "user", "project", "sessionFlags", "legacyManagedConfigTomlFromFile", "legacyManagedConfigTomlFromMdm":
		return true
	}
	return false
}
func provePermissionRequirements(raw json.RawMessage, profile string) (any, error) {
	var result map[string]any
	if json.Unmarshal(raw, &result) != nil {
		return nil, permissionError()
	}
	value, present := result["requirements"]
	if !present {
		return nil, permissionError()
	}
	if value == nil {
		return nil, nil
	}
	r, ok := value.(map[string]any)
	if !ok {
		return nil, permissionError()
	}
	if v, ok := r["allowedPermissionProfiles"]; ok && v != nil {
		m, ok := v.(map[string]any)
		if !ok || m[profile] != true {
			return nil, permissionError()
		}
	}
	if v := r["defaultPermissions"]; v != nil && v != profile {
		return nil, permissionError()
	}
	if v := r["allowedApprovalPolicies"]; v != nil {
		a, ok := v.([]any)
		found := false
		for _, p := range a {
			found = found || p == "never"
		}
		if !ok || !found {
			return nil, permissionError()
		}
	}
	// Unknown managed execution constraints must not silently broaden or replace
	// the selected profile. Other known product settings are outside this proof.
	for key, v := range r {
		if v == nil {
			continue
		}
		if !knownRequirementKey(key) {
			return nil, permissionError()
		}
		if key == "featureRequirements" {
			m, ok := v.(map[string]any)
			if !ok || len(m) > 0 {
				return nil, permissionError()
			}
		}
		lower := strings.ToLower(key)
		if strings.Contains(lower, "permission") || strings.Contains(lower, "sandbox") || strings.Contains(lower, "approval") || strings.Contains(lower, "guardian") || strings.Contains(lower, "autoreview") {
			switch key {
			case "allowedPermissionProfiles", "defaultPermissions", "allowedApprovalPolicies":
			default:
				return nil, permissionError()
			}
		}
	}
	if r["network"] != nil || r["experimentalNetwork"] != nil {
		return nil, permissionError()
	}
	projection := map[string]any{}
	for _, key := range []string{"allowedPermissionProfiles", "defaultPermissions", "allowedApprovalPolicies"} {
		if v := r[key]; v != nil {
			projection[key] = v
		}
	}
	return projection, nil
}

func knownRequirementKey(key string) bool {
	switch key {
	case "allowedPermissionProfiles", "defaultPermissions", "allowedApprovalPolicies", "allowedSandboxModes", "allowedWindowsSandboxImplementations", "autoReview",
		"additionalDeveloperInstructions", "allowAppshots", "allowBrowserAndComputerUse", "allowLoginShell", "allowManagedHooksOnly", "allowRemoteControl", "allowedLoginMethods", "allowedWebSearchModes", "browserUse", "chatgptBaseUrl", "checkForUpdateOnStartup", "cliAuthCredentialsStore", "computerUse", "enforceResidency", "featureRequirements", "feedback", "inAppBrowser", "logDir", "modelCatalogJson", "modelProvider", "modelProviders", "models", "sqliteHome":
		return true
	}
	return false
}

func unexpectedPermissionKey(config map[string]any) bool {
	for key, v := range config {
		if v == nil {
			continue
		}
		if key == "include_permissions_instructions" {
			enabled, ok := v.(bool)
			if !ok || !enabled {
				return true
			}
			continue
		}
		lower := strings.ToLower(key)
		if lower == "experimental_network" || lower == "writable_roots" || lower == "writableroots" || lower == "network_access" {
			return true
		}
		if strings.Contains(lower, "permission") || strings.Contains(lower, "sandbox") || strings.Contains(lower, "approval") {
			switch key {
			case "permissions", "default_permissions", "sandbox_mode", "sandbox_workspace_write", "approval_policy", "approvals_reviewer":
			default:
				return true
			}
		}
	}
	return false
}

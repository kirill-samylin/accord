package worker

import (
	"reflect"
	"strings"
	"testing"
)

func TestNativeMCPConsentIsStrictAndDefaultOff(t *testing.T) {
	providers := map[string]string{"codex": "/synthetic/codex", "claude": "/synthetic/claude"}
	for _, grants := range []map[string][]string{nil, {}, {"codex": nil}, {"claude": {}}, {"codex": {"describe", "next"}, "claude": {"get"}}} {
		if err := validateNativeMCPConsent(grants, providers); err != nil {
			t.Fatal(err)
		}
	}
	for _, grants := range []map[string][]string{{"other": {}}, {"codex": {"Describe"}}, {"codex": {"*"}}, {"claude": {"describe", "describe"}}, {"codex": {"mcp__accord_managed__next"}}} {
		if validateNativeMCPConsent(grants, providers) == nil {
			t.Fatal("accepted unsupported or duplicate grant")
		}
	}
	if validateNativeMCPConsent(map[string][]string{"codex": {}}, nil) == nil {
		t.Fatal("unconfigured provider accepted")
	}
	if codexMCPConsent(nil) != "" || codexMCPConsent([]string{}) != "" || len(claudeMCPConsent(nil)) != 0 || len(claudeMCPConsent([]string{})) != 0 {
		t.Fatal("default-off consent emits permission overrides")
	}
}

func TestCodexMCPConsentNarrowsOnlyFixedManagedConnector(t *testing.T) {
	plain, err := codexMCPOverrides(nil, "/synthetic/self", []string{"worker", "bridge"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	granted, err := codexMCPOverrides(nil, "/synthetic/self", []string{"worker", "bridge"}, []string{"describe", "next"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain[1], "approval_mode") || strings.Contains(plain[1], "enabled_tools") {
		t.Fatal("empty grant changes policy")
	}
	value := granted[1]
	if !strings.Contains(value, `,enabled_tools=["describe","next"],default_tools_approval_mode="prompt",tools={`) {
		t.Fatal("managed catalog is not exactly narrowed")
	}
	for _, tool := range managedMCPTools {
		mode := "prompt"
		if tool == "describe" || tool == "next" {
			mode = "approve"
		}
		if !strings.Contains(value, `"`+tool+`"={approval_mode="`+mode+`"}`) {
			t.Fatal("missing exact tool policy")
		}
	}
	if strings.Count(value, `approval_mode="approve"`) != 2 {
		t.Fatal("grant extends beyond selected tools")
	}
	for _, forbidden := range []string{"approval_policy", "sandbox_mode", "bypass", "dangerously", "ask_for_approval"} {
		if strings.Contains(value, forbidden) {
			t.Fatal("grant changed global/native permissions")
		}
	}
}

func TestClaudeMCPConsentUsesExactNamesAndRemovesUnselectedBridgeTools(t *testing.T) {
	args := claudeMCPConsent([]string{"describe", "next"})
	if len(args) != 4 || args[0] != "--allowedTools" || args[1] != "mcp__accord_managed__describe,mcp__accord_managed__next" || args[2] != "--disallowedTools" {
		t.Fatal("incorrect exact allowed MCP names")
	}
	deny := strings.Split(args[3], ",")
	if len(deny) != len(managedMCPTools)-2 {
		t.Fatal("unselected MCP operations remain available")
	}
	for _, name := range deny {
		if !strings.HasPrefix(name, "mcp__accord_managed__") || strings.Contains(name, "*") || strings.HasSuffix(name, "__describe") || strings.HasSuffix(name, "__next") {
			t.Fatal("deny list changes selected/builtin tools")
		}
	}
}

func TestNativeMCPConsentSnapshotOwnsProviderLists(t *testing.T) {
	original := Config{NativeMCPApprovedTools: map[string][]string{"codex": {"describe", "next"}}}
	snapshot := snapshotConfig(original)
	original.NativeMCPApprovedTools["codex"][0] = "get"
	original.NativeMCPApprovedTools["claude"] = []string{"update"}
	if !reflect.DeepEqual(snapshot.NativeMCPApprovedTools, map[string][]string{"codex": {"describe", "next"}}) {
		t.Fatal("consent snapshot shares mutable local grants")
	}
}

func TestBridgeAppliesOnlyLocallySelectedProviderConsent(t *testing.T) {
	e := testEngine(t)
	e.config.NativeMCPApprovedTools = map[string][]string{"codex": {"describe", "next"}, "claude": {"get"}}
	seen := false
	e.verifyMCP = func(_, _ string, overrides []string, _ string, _ []string) error {
		seen = strings.Contains(overrides[1], `enabled_tools=["describe","next"]`)
		return nil
	}
	if _, err := e.bridgeArgs(RunState{ID: "run_consent_codex", Provider: "codex", Workspace: "test", Generation: 1}); err != nil || !seen {
		t.Fatalf("Codex local consent not passed to inspected invocation: %v", err)
	}
	launch, err := e.bridgeArgs(RunState{ID: "run_consent_claude", Provider: "claude", Workspace: "test", Generation: 1})
	if err != nil {
		t.Fatal(err)
	}
	args := launch.Args
	if len(args) != 7 || args[0] != "--strict-mcp-config" || args[1] != "--mcp-config" || args[3] != "--allowedTools" || args[4] != "mcp__accord_managed__get" {
		t.Fatal("Claude consent altered strict MCP binding or selected other provider's grants")
	}
}

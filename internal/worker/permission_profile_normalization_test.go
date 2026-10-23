package worker

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func independentEffectiveProfile(t *testing.T) (configInspection, map[string]any, map[string]any) {
	t.Helper()
	i, r := permissionFixture(t)
	var effective map[string]any
	if err := json.Unmarshal(permissionRaw(r["config"]), &effective); err != nil {
		t.Fatal(err)
	}
	r["config"] = effective
	return i, r, effective["permissions"].(map[string]any)[i.Profile].(map[string]any)
}

func TestEffectiveProfileTypedNullDefaultsPreserveExactGrantAndRawLayerProof(t *testing.T) {
	i, r, profile := independentEffectiveProfile(t)
	requirements := permissionRaw(map[string]any{"requirements": nil})
	baseline, err := provePermissionConfig(permissionRaw(r), requirements, i)
	if err != nil {
		t.Fatal(err)
	}
	layersBefore := string(permissionRaw(r["layers"]))
	// Reproduce fifteen absent typed defaults without using captured user data.
	for _, entry := range []map[string]any{profile, profile["network"].(map[string]any), profile["filesystem"].(map[string]any)} {
		for _, key := range []string{"nullable_a", "nullable_b", "nullable_c", "nullable_d", "nullable_e"} {
			entry[key] = nil
		}
	}
	proof, err := provePermissionConfig(permissionRaw(r), requirements, i)
	if err != nil || proof != baseline {
		t.Fatal("null defaults changed permissions or proof", err)
	}
	if string(permissionRaw(r["layers"])) != layersBefore {
		t.Fatal("normalization mutated raw layers")
	}
	layer := r["layers"].([]any)[0].(map[string]any)["config"].(map[string]any)["permissions"].(map[string]any)[i.Profile].(map[string]any)
	layer["raw_nullable_default"] = nil
	changed, err := provePermissionConfig(permissionRaw(r), requirements, i)
	if err != nil || changed.Digest == proof.Digest {
		t.Fatal("raw layer projection was normalized or ignored", err)
	}
}

func TestEffectiveProfileNormalizationDoesNotEraseMeaningfulExtras(t *testing.T) {
	for _, extra := range []any{false, float64(0), "", map[string]any{}, []any{}, []any{nil}, map[string]any{"nullable": nil}, map[string]any{"outside": "write"}} {
		i, r, p := independentEffectiveProfile(t)
		p["unknown_extra"] = extra
		if _, err := provePermissionConfig(permissionRaw(r), permissionRaw(map[string]any{"requirements": nil}), i); err == nil {
			t.Fatal("meaningful extra was normalized away")
		}
	}
	cases := map[string]func(configInspection, map[string]any){
		"config-write": func(i configInspection, p map[string]any) {
			p["filesystem"].(map[string]any)[filepath.Join(i.Directory, ".git")].(map[string]any)["config"] = "write"
		},
		"hooks-write": func(i configInspection, p map[string]any) {
			p["filesystem"].(map[string]any)[filepath.Join(i.Directory, ".git")].(map[string]any)["hooks"] = "write"
		},
		"required-config-null": func(i configInspection, p map[string]any) {
			p["filesystem"].(map[string]any)[filepath.Join(i.Directory, ".git")].(map[string]any)["config"] = nil
		},
		"extra-root": func(i configInspection, p map[string]any) {
			p["filesystem"].(map[string]any)["/synthetic/outside"] = "write"
		},
		"workspace-root": func(i configInspection, p map[string]any) {
			p["workspace_roots"] = map[string]any{"/synthetic/outside": true}
		},
		"extends":               func(i configInspection, p map[string]any) { p["extends"] = ":read-only" },
		"required-parent-null":  func(i configInspection, p map[string]any) { p["extends"] = nil },
		"network":               func(i configInspection, p map[string]any) { p["network"].(map[string]any)["enabled"] = true },
		"required-network-null": func(i configInspection, p map[string]any) { p["network"].(map[string]any)["enabled"] = nil },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			i, r, p := independentEffectiveProfile(t)
			mutate(i, p)
			p["nullable_default"] = nil
			if _, err := provePermissionConfig(permissionRaw(r), permissionRaw(map[string]any{"requirements": nil}), i); err == nil {
				t.Fatal("changed grant accepted")
			}
		})
	}
	containers := []any{nil, false, float64(0), "", []any{}, []any{nil, map[string]any{"null": nil}}, map[string]any{}}
	for _, value := range containers {
		if !reflect.DeepEqual(normalizeEffectiveProfile(value), value) {
			t.Fatal("type/container/meaningful value transformed")
		}
	}
}

func TestPermissionInstructionsTrueOnlyAndActiveLayerDigestBinding(t *testing.T) {
	i, r, _ := independentEffectiveProfile(t)
	requirements := permissionRaw(map[string]any{"requirements": nil})
	baseline, err := provePermissionConfig(permissionRaw(r), requirements, i)
	if err != nil {
		t.Fatal(err)
	}
	c := r["config"].(map[string]any)
	c["include_permissions_instructions"] = true
	proof, err := provePermissionConfig(permissionRaw(r), requirements, i)
	if err != nil || proof != baseline {
		t.Fatal("known true effective instruction setting changed grants", err)
	}
	layer := r["layers"].([]any)[0].(map[string]any)["config"].(map[string]any)
	layer["include_permissions_instructions"] = true
	bound, err := provePermissionConfig(permissionRaw(r), requirements, i)
	if err != nil || bound.Digest == proof.Digest {
		t.Fatal("explicit active instruction setting not bound to raw projection", err)
	}
	layer["include_permissions_instructions"] = nil
	c["include_permissions_instructions"] = nil
	nullProof, err := provePermissionConfig(permissionRaw(r), requirements, i)
	if err != nil || nullProof != baseline {
		t.Fatal("null default changed previous absent handling", err)
	}
}

func TestPermissionInstructionsRejectFalseTypesAndUnknownSimilarKeys(t *testing.T) {
	for _, position := range []string{"effective", "layer"} {
		for _, value := range []any{false, "true", float64(1), map[string]any{}, []any{}, map[string]any{"enabled": true}} {
			i, r, _ := independentEffectiveProfile(t)
			target := r["config"].(map[string]any)
			if position == "layer" {
				target = r["layers"].([]any)[0].(map[string]any)["config"].(map[string]any)
			}
			target["include_permissions_instructions"] = value
			if _, err := provePermissionConfig(permissionRaw(r), permissionRaw(map[string]any{"requirements": nil}), i); err == nil {
				t.Fatal("non-true instruction setting accepted", position)
			}
		}
		for _, key := range []string{"include_permissions_instructions_extra", "include_permission_instructions", "include_sandbox_instructions", "include_approval_instructions", "Include_permissions_instructions"} {
			i, r, _ := independentEffectiveProfile(t)
			target := r["config"].(map[string]any)
			if position == "layer" {
				target = r["layers"].([]any)[0].(map[string]any)["config"].(map[string]any)
			}
			target[key] = true
			if _, err := provePermissionConfig(permissionRaw(r), permissionRaw(map[string]any{"requirements": nil}), i); err == nil {
				t.Fatal("unknown include key accepted", position)
			}
		}
	}
}

func TestUnrelatedInstructionSettingsPreservePreviousPermissionProof(t *testing.T) {
	for _, position := range []string{"effective", "layer"} {
		i, r, _ := independentEffectiveProfile(t)
		c := r["config"].(map[string]any)
		layer := r["layers"].([]any)[0].(map[string]any)["config"].(map[string]any)
		c["include_permissions_instructions"] = true
		layer["include_permissions_instructions"] = true
		requirements := permissionRaw(map[string]any{"requirements": nil})
		baseline, err := provePermissionConfig(permissionRaw(r), requirements, i)
		if err != nil {
			t.Fatal(err)
		}
		target := c
		if position == "layer" {
			target = layer
		}
		for _, key := range []string{"include_apps_instructions", "include_collaboration_mode_instructions", "include_environment_context", "include_environment_context_time"} {
			target[key] = true
		}
		for _, value := range []any{true, false, "synthetic unrelated setting", float64(1), map[string]any{}, []any{}} {
			target["include_unknown_flag"] = value
			proof, err := provePermissionConfig(permissionRaw(r), requirements, i)
			if err != nil || proof != baseline {
				t.Fatal("unrelated instruction metadata changed the previous permission boundary", position, err)
			}
		}
	}
}

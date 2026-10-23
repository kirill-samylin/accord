package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"
)

type failedCapabilityWriter struct{}

func (failedCapabilityWriter) Write([]byte) (int, error) {
	return 0, errors.New("fixture writer failure")
}
func TestCapabilitiesRequireFullAutomationFloor(t *testing.T) {
	var out bytes.Buffer
	if err := printCapabilities(&out); err != nil {
		t.Fatal(err)
	}
	var got map[string]json.RawMessage
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	for _, floor := range []string{"project_registry", "project_acl", "agent_profiles", "coordination", "deduplication", "managed_agents", "automation"} {
		if string(got[floor]) != "1" {
			t.Fatalf("missing floor %s", floor)
		}
	}
	if string(got["registry_schema"]) != "[1]" || string(got["board_schema"]) != "[6]" {
		t.Fatal("unsupported schema declaration")
	}
	if err := printCapabilities(failedCapabilityWriter{}); err == nil {
		t.Fatal("writer failure hidden")
	}
}

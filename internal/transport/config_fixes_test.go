package transport

import (
	"agent-board/internal/core"
	"encoding/json"
	"slices"
	"testing"
)

func TestCreateAreasRemainOptionalAndPreserveExplicitEmpty(t *testing.T) {
	f := newFixture(t)
	found := false
	for _, op := range operations(f.store) {
		if op.name != "create" {
			continue
		}
		found = true
		if slices.Contains(op.schema.Required, "areas") {
			t.Fatal("areas inheritance became a required MCP field")
		}
		if _, ok := op.schema.Properties["areas"]; !ok {
			t.Fatal("areas missing from schema")
		}
	}
	if !found {
		t.Fatal("create tool missing")
	}
	for _, input := range []struct {
		raw     string
		present bool
	}{
		{`{"type":"task"}`, false},
		{`{"type":"task","areas":[]}`, true},
	} {
		var dto core.CreateInput
		if err := decode([]byte(input.raw), &dto); err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(dto)
		if err != nil {
			t.Fatal(err)
		}
		var encoded map[string]json.RawMessage
		if err = json.Unmarshal(raw, &encoded); err != nil {
			t.Fatal(err)
		}
		areas, present := encoded["areas"]
		if present != input.present || present && string(areas) != "[]" {
			t.Fatal("SDK/hash serialization lost areas presence", string(raw))
		}
	}
}

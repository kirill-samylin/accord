package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"encoding/json"
	"reflect"
	"testing"
)

func TestCreateAreasPresenceIsPreservedForIdempotency(t *testing.T) {
	f := newFixture(t)
	resetConfig(t, f, func(c *config.Config) { c.Areas = []config.Area{{Name: "src", Aliases: []string{}}} })
	areas := []string{"src"}
	_, err := f.s.Update(testContext, f.coord, core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: f.epic.Key}, Areas: &areas})
	if err != nil {
		t.Fatal(err)
	}
	decode := func(raw string) core.CreateInput {
		t.Helper()
		var input core.CreateInput
		if err := json.Unmarshal([]byte(raw), &input); err != nil {
			t.Fatal(err)
		}
		input.RequestID = "presence"
		input.EpicKey = f.epic.Key
		return input
	}
	omitted := decode(`{"type":"task","title":"inherit","description":"implement","role":"developer"}`)
	explicit := decode(`{"type":"task","title":"inherit","description":"implement","role":"developer","areas":[]}`)
	inherited, err := f.s.Create(testContext, f.coord, omitted)
	if err != nil || !reflect.DeepEqual(inherited.Areas, areas) {
		t.Fatal(inherited, err)
	}
	replay, err := f.s.Create(testContext, f.coord, omitted)
	if err != nil || !reflect.DeepEqual(replay, inherited) {
		t.Fatal("omitted replay differs", replay, err)
	}
	_, err = f.s.Create(testContext, f.coord, explicit)
	requireCode(t, err, "IDEMPOTENCY_CONFLICT")
	explicit.RequestID = f.id()
	empty, err := f.s.Create(testContext, f.coord, explicit)
	if err != nil || len(empty.Areas) != 0 {
		t.Fatal("explicit [] inherited areas", empty, err)
	}
	replay, err = f.s.Create(testContext, f.coord, explicit)
	expectedJSON, marshalErr := json.Marshal(empty)
	actualJSON, replayMarshalErr := json.Marshal(replay)
	if err != nil || marshalErr != nil || replayMarshalErr != nil || string(expectedJSON) != string(actualJSON) {
		t.Fatal("explicit [] replay differs", replay, err)
	}
	omitted.RequestID = explicit.RequestID
	_, err = f.s.Create(testContext, f.coord, omitted)
	requireCode(t, err, "IDEMPOTENCY_CONFLICT")
	// Optional pointers retain presence even when their values are zero/empty.
	draft := false
	omitted.RequestID = "pointer-presence"
	_, err = f.s.Create(testContext, f.coord, omitted)
	if err != nil {
		t.Fatal(err)
	}
	omitted.Ready = &draft
	_, err = f.s.Create(testContext, f.coord, omitted)
	requireCode(t, err, "IDEMPOTENCY_CONFLICT")
	update := core.UpdateInput{LeaseInput: core.LeaseInput{RequestID: "update-presence", TaskKey: empty.Key}}
	_, err = f.s.Update(testContext, f.coord, update)
	if err != nil {
		t.Fatal(err)
	}
	cleared := []string{}
	update.Areas = &cleared
	_, err = f.s.Update(testContext, f.coord, update)
	requireCode(t, err, "IDEMPOTENCY_CONFLICT")
}

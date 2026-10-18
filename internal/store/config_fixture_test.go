package store

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"encoding/json"
	"testing"
)

func (f *fixture) configure(t *testing.T, edit func(*config.Config)) core.ConfigView {
	t.Helper()
	v, err := f.s.Config(testContext)
	if err != nil {
		t.Fatal(err)
	}
	var c config.Config
	if err = json.Unmarshal(v.Config, &c); err != nil {
		t.Fatal(err)
	}
	edit(&c)
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	out, err := f.s.ApplyConfig(testContext, raw, v.ConfigVersion, "unit-admin")
	if err != nil {
		t.Fatal(err)
	}
	return out
}

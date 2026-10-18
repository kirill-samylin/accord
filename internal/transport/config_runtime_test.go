package transport

import (
	"agent-board/internal/config"
	"agent-board/internal/core"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"
)

func TestConfigReadsAreAuthenticatedCanonicalAndReadOnly(t *testing.T) {
	f := newFixture(t)
	active, err := f.store.Config(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var cfg config.Config
	if err = json.Unmarshal(active.Config, &cfg); err != nil {
		t.Fatal(err)
	}
	cfg.Notes.MaxPerRole = 19
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	raw = append([]byte("# private-source-marker\n"), raw...)
	if _, err = f.store.ApplyConfig(context.Background(), raw, active.ConfigVersion, "test-admin"); err != nil {
		t.Fatal(err)
	}
	status, data := request(t, f, "GET", "/api/v1/config", "", "")
	v := parseResult[core.ConfigView](t, status, data)
	if v.ConfigVersion != 2 || bytes.Contains(data, []byte("private-source-marker")) {
		t.Fatal("raw source leaked", string(data))
	}
	status, data = request(t, f, "GET", "/api/v1/config/history?limit=1", "", "")
	history := parseResult[core.ConfigHistory](t, status, data)
	if !history.HasMore || history.Revisions[0].ConfigVersion != 1 {
		t.Fatal(history)
	}
	status, data = request(t, f, "GET", "/api/v1/config/history?limit=1&after_version=1", "", "")
	history = parseResult[core.ConfigHistory](t, status, data)
	if history.HasMore || history.Revisions[0].ConfigVersion != 2 || bytes.Contains(data, []byte("raw_yaml")) || bytes.Contains(data, []byte("private-source-marker")) {
		t.Fatal("history raw source leaked", string(data))
	}
	for _, path := range []string{"/api/v1/config/history?limit=0", "/api/v1/config/history?limit=201", "/api/v1/config/history?after_version=-1", "/api/v1/config/history?limit=1&limit=2", "/api/v1/config/history?unknown=x", "/api/v1/config/history?%"} {
		status, _ = request(t, f, "GET", path, "", "")
		if status != 400 {
			t.Fatalf("invalid history query %s returned %d", path, status)
		}
	}
	status, _ = request(t, f, "POST", "/api/v1/config", "", `{}`)
	if status != http.StatusMethodNotAllowed {
		t.Fatalf("remote write exposed: %d", status)
	}
	response, err := http.Get(f.server.URL + "/api/v1/config")
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != 401 {
		t.Fatal("unauthenticated config read")
	}
}

package acceptance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"agent-board/internal/core"
)

// Optional private diagnostic output contains synthetic configuration only: this
// CAS fixture mints no token. Save poststate before an exact-code assertion fails.
func recordCAS(t *testing.T, b *board, candidates []string, expected int64, observed []result, current core.ConfigView, histories string, auditBefore, auditAfter, winners int) {
	t.Helper()
	directory := os.Getenv("ACCORD_CAS_EVIDENCE_DIR")
	if directory == "" {
		return
	}
	if err := os.MkdirAll(directory, 0700); err != nil {
		t.Fatal(err)
	}
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(directory, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for i, r := range observed {
		write(fmt.Sprintf("process-%d.stdout", i), r.data)
		write(fmt.Sprintf("process-%d.stderr", i), r.stderr)
	}
	for i, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		write(fmt.Sprintf("candidate-%d.json", i), data)
	}
	commands := []string{}
	for i := range candidates {
		commands = append(commands, fmt.Sprintf("agent-board config apply --db DB --file candidate-%d.json --expected-version %d --actor qa056", i, expected))
	}
	data, err := json.MarshalIndent(struct {
		Commands                []string
		Expected                int64
		Winners                 int
		Active                  core.ConfigView
		Histories               string
		AuditBefore, AuditAfter int
	}{commands, expected, winners, current, histories, auditBefore, auditAfter}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	write("poststate.json", append(data, '\n'))
	snapshot := command(nil, "backup", "--db", b.db, "--output", filepath.Join(directory, "poststate.db"))
	if snapshot.err != nil {
		t.Fatalf("CAS diagnostic snapshot: %s", snapshot.stderr)
	}
	write("snapshot-metadata.json", snapshot.data)
}

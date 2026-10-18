package interchange

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func exportOne(r Record) NextRecord {
	done := false
	return func() (Record, error) {
		if done {
			return Record{}, io.EOF
		}
		done = true
		return r, nil
	}
}

func TestIdentityParseExportParity(t *testing.T) {
	doc, err := Parse(strings.NewReader(manifest+"\n"+task), "fixture", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"", "bad\x00id", strings.Repeat("x", 257), strings.Repeat("я", 129), "valid", strings.Repeat("x", 256), strings.Repeat("я", 128)} {
		r := doc.Records[0]
		r.ExternalID = id
		valid := id != "" && !strings.ContainsRune(id, '\x00') && len(id) <= 256
		var output bytes.Buffer
		err := Encode(&output, doc.Manifest, exportOne(r))
		if (err == nil) != valid {
			t.Fatalf("export identity length=%d valid=%v: %v", len(id), valid, err)
		}
		encoded, marshalErr := json.Marshal(r)
		if marshalErr != nil {
			t.Fatal(marshalErr)
		}
		parsed, parseErr := Parse(strings.NewReader(manifest+"\n"+string(encoded)), "fixture", nil)
		if (parseErr == nil) != valid || (parsed != nil) != valid {
			t.Fatalf("parse identity length=%d valid=%v: %v", len(id), valid, parseErr)
		}
		if valid {
			if _, err := Parse(bytes.NewReader(output.Bytes()), "fixture", nil); err != nil {
				t.Fatal("export failed round trip", err)
			}
		} else {
			path := filepath.Join(t.TempDir(), "export.jsonl")
			if err := ExportFile(path, doc.Manifest, exportOne(r)); err == nil {
				t.Fatal("invalid identity published")
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatal("invalid identity left final file", err)
			}
			entries, err := os.ReadDir(filepath.Dir(path))
			if err != nil || len(entries) != 0 {
				t.Fatal("failed export leaked staging file", entries, err)
			}
		}
	}
}

func TestNULReferencesNeverReachMappingOrExport(t *testing.T) {
	ref := &Ref{Kind: "epic", ExternalID: "bad\x00id"}
	baseTask := Task{Type: "task", Title: "Task", Description: "", Role: "developer", State: "ready"}
	taskEpic, taskTarget := baseTask, baseTask
	taskEpic.Epic = ref
	taskTarget.Target = &Ref{Kind: "task", ExternalID: ref.ExternalID}
	for _, item := range []struct {
		kind string
		data any
	}{
		{"task", taskEpic}, {"task", taskTarget},
		{"note", Note{Text: "Note", Kind: "context", Epic: ref}},
		{"history", map[string]any{"origin_ref": ref, "type": "attempt", "at": "2026-01-01T00:00:00Z"}},
		{"link", Link{From: *ref, To: Ref{Kind: "task", ExternalID: "mapped"}, Type: "blocks"}},
		{"link", Link{From: Ref{Kind: "task", ExternalID: "mapped"}, To: *ref, Type: "related"}},
	} {
		data, err := json.Marshal(item.data)
		if err != nil {
			t.Fatal(err)
		}
		r := Record{Kind: item.kind, ExternalID: "valid", Data: data}
		raw, err := json.Marshal(r)
		if err != nil {
			t.Fatal(err)
		}
		called := false
		_, err = ParseWithReferences(strings.NewReader(manifest+"\n"+string(raw)), "fixture", nil, func(Ref) bool {
			called = true
			return true
		})
		if err == nil || called {
			t.Fatal("NUL reference reached authoritative mapping", item.kind, err)
		}
		if diagnostics := ValidateReferences([]Record{r}, func(Ref) bool { return true }); len(diagnostics) == 0 {
			t.Fatal("reference-only check accepted NUL", item.kind)
		}
		var out bytes.Buffer
		if err := Encode(&out, Manifest{Format: "accord-jsonl", Version: 1, Source: "fixture"}, exportOne(r)); err == nil {
			t.Fatal("NUL reference exported", item.kind)
		}
		path := filepath.Join(t.TempDir(), "export.jsonl")
		if err := ExportFile(path, Manifest{Format: "accord-jsonl", Version: 1, Source: "fixture"}, exportOne(r)); err == nil {
			t.Fatal("NUL reference published", item.kind)
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("NUL reference left final file", item.kind, err)
		}
	}
}

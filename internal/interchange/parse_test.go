package interchange

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const manifest = `{"format":"accord-jsonl","version":1,"source":"fixture"}`
const task = `{"kind":"task","external_id":"T-1","data":{"type":"task","title":"Task","description":"","role":"developer","state":"done","priority":0}}`

func TestParseExactHashAndAuthority(t *testing.T) {
	raw := manifest + "\n" + task + "\n"
	doc, err := Parse(strings.NewReader(raw), "fixture", nil)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte(raw))
	if doc.FileSHA256 != hex.EncodeToString(sum[:]) {
		t.Fatal("hash changed bytes")
	}
	for _, field := range []string{`"lease":{}`, `"author_session":"foreign"`, `"completion_trust":"local"`, `"generation":8`, `"auth":{}`} {
		invalid := strings.Replace(task, `"priority":0`, `"priority":0,`+field, 1)
		if d, err := Parse(strings.NewReader(manifest+"\n"+invalid), "fixture", nil); err == nil || d != nil {
			t.Fatalf("authority accepted: %s", field)
		}
	}
}
func TestParseRejectsInvalidWholeFile(t *testing.T) {
	inputs := []string{
		manifest + "\n" + task + "\n" + task,
		manifest + "\n" + task + "\n{",
		strings.Replace(manifest, `"version":1`, `"version":1,"version":1`, 1) + "\n" + task,
		manifest + "\n" + strings.Replace(task, `"priority":0`, `"priority":9007199254740993.0`, 1),
		strings.Replace(manifest, `"version":1`, `"version":2`, 1),
		manifest + "\n" + strings.Replace(task, `"title":"Task"`, `"title":"`+string([]byte{0xff})+`"`, 1),
		manifest + "\n" + strings.Repeat(" ", MaxLine+1),
	}
	for i, raw := range inputs {
		if d, err := Parse(strings.NewReader(raw), "fixture", nil); err == nil || d != nil {
			t.Fatalf("case %d accepted", i)
		}
	}
	if _, err := Parse(strings.NewReader(manifest), "other", nil); err == nil {
		t.Fatal("source mismatch")
	}
}
func TestNumberPrecisionAndCanonicalHash(t *testing.T) {
	history := `{"kind":"history","external_id":"h1","data":{"origin_ref":{"kind":"task","external_id":"T-1"},"type":"attempt","at":"2026-01-01T00:00:00Z","tokens_spent":9007199254740993}}`
	doc, err := Parse(strings.NewReader(manifest+"\n"+history+"\n"+task), "fixture", nil)
	if err != nil {
		t.Fatal(err)
	}
	var h History
	if err := json.Unmarshal(doc.Records[0].Data, &h); err != nil || h.TokensSpent == nil || *h.TokensSpent != 9007199254740993 {
		t.Fatal("precision lost")
	}
	r := doc.Records[1]
	h1, _ := RecordHash(r)
	r.Data = json.RawMessage(`{ "priority":0,"state":"done","role":"developer","description":"","title":"Task","type":"task" }`)
	h2, _ := RecordHash(r)
	if h1 != h2 {
		t.Fatal("key order changed canonical hash")
	}
	negative := strings.Replace(history, "9007199254740993", "-1", 1)
	if _, err := Parse(strings.NewReader(manifest+"\n"+task+"\n"+negative), "fixture", nil); err == nil {
		t.Fatal("negative token accepted")
	}
}
func TestReferencesAndCycles(t *testing.T) {
	link := func(id, from, to string) string {
		return `{"kind":"link","external_id":"` + id + `","data":{"from":{"kind":"task","external_id":"` + from + `"},"to":{"kind":"task","external_id":"` + to + `"},"type":"blocks"}}`
	}
	task2 := strings.Replace(task, "T-1", "T-2", 1)
	for _, raw := range []string{manifest + "\n" + task + "\n" + link("l1", "T-1", "unknown"), manifest + "\n" + link("l1", "T-1", "T-2") + "\n" + link("l2", "T-2", "T-1") + "\n" + task + "\n" + task2} {
		if _, err := Parse(strings.NewReader(raw), "fixture", nil); err == nil {
			t.Fatal("invalid graph accepted")
		}
	}
}
func TestJSONBoundsAndDuplicates(t *testing.T) {
	for _, raw := range []string{`{"x":1,"x":2}`, strings.Repeat("[", 33) + "0" + strings.Repeat("]", 33), `{} {}`, `{"nested":{"a":1,"a":2}}`} {
		if CheckJSON([]byte(raw), 32) == nil {
			t.Fatal("invalid accepted", raw)
		}
	}
}
func TestExportNoPartialAndNoOverwrite(t *testing.T) {
	doc, err := Parse(strings.NewReader(manifest+"\n"+task), "fixture", nil)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "export.jsonl")
	next := func() NextRecord {
		n := 0
		return func() (Record, error) {
			if n > 0 {
				return Record{}, io.EOF
			}
			n++
			return doc.Records[0], nil
		}
	}
	if err := ExportFile(path, doc.Manifest, next()); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(path)
	if err := ExportFile(path, doc.Manifest, next()); err == nil {
		t.Fatal("overwritten")
	}
	after, _ := os.ReadFile(path)
	if !bytes.Equal(before, after) {
		t.Fatal("changed existing")
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Fatal("public file")
	}
	partial := filepath.Join(filepath.Dir(path), "partial")
	n := 0
	err = ExportFile(partial, doc.Manifest, func() (Record, error) {
		n++
		if n == 1 {
			return doc.Records[0], nil
		}
		return Record{}, errors.New("snapshot interrupted")
	})
	if err == nil {
		t.Fatal("ignored failure")
	}
	if _, err := os.Stat(partial); !os.IsNotExist(err) {
		t.Fatal("published partial")
	}
}

type validatorFunc func(Record) []Diagnostic

func (f validatorFunc) ValidateRecord(r Record) []Diagnostic { return f(r) }
func TestPolicyDiagnosticsMappingAndCaps(t *testing.T) {
	denied := validatorFunc(func(r Record) []Diagnostic { return []Diagnostic{{Path: "data.role", Code: "UNKNOWN_ROLE"}} })
	if d, err := Parse(strings.NewReader(manifest+"\n"+task), "fixture", denied); err == nil || d != nil {
		t.Fatal("policy ignored")
	}
	raw := manifest + "\n" + strings.Replace(task, `"priority":0`, `"priority":0,"epic":{"kind":"epic","external_id":"mapped"}`, 1)
	if _, err := ParseWithReferences(strings.NewReader(raw), "fixture", nil, func(ref Ref) bool { return ref.Kind == "epic" && ref.ExternalID == "mapped" }); err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{strings.Replace(task, `"title":"Task"`, `"Title":"Task"`, 1), strings.Replace(task, `,"priority":0`, "", 1), strings.Replace(task, `"priority":0`, `"priority":null`, 1)} {
		if _, err := Parse(strings.NewReader(manifest+"\n"+mutation), "fixture", nil); err == nil {
			t.Fatal("schema alias/missing/null")
		}
	}
	_, err := Parse(strings.NewReader(manifest+"\n"+strings.Repeat("{}\n", 150)), "fixture", nil)
	var invalid *Invalid
	if !errors.As(err, &invalid) || len(invalid.Diagnostics) != 100 {
		t.Fatal("diagnostic cap", err)
	}
	if _, err := Parse(io.MultiReader(strings.NewReader(manifest+"\n"), io.LimitReader(repeatReader{}, MaxBytes)), "fixture", nil); err == nil {
		t.Fatal("byte cap")
	}
}

type repeatReader struct{}

func (repeatReader) Read(b []byte) (int, error) {
	for i := range b {
		b[i] = ' '
	}
	return len(b), nil
}

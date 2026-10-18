package interchange

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"regexp"
)

var sourcePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

func ValidSource(source string) bool  { return sourcePattern.MatchString(source) }
func identity(kind, id string) string { return kind + "\x00" + id }

// Parse checks syntax and closed-file references only when validator is nil.
// It never declares the document valid against a live board/config.
// Parse consumes a bounded stream, preserving the exact file hash. A nil document
// accompanies any diagnostic; callers cannot accidentally apply a valid prefix.
func Parse(r io.Reader, source string, validator Validator) (*Document, error) {
	return ParseWithReferences(r, source, validator, nil)
}

// ParseWithReferences adds an authoritative same-source mapping lookup. Offline
// validation passes nil and requires a references-closed package. This callback
// cannot authorize apply or substitute current-board validation inside a writer lock.
func ParseWithReferences(r io.Reader, source string, validator Validator, exists func(Ref) bool) (*Document, error) {
	bad := &Invalid{}
	add := func(line int, path, code string) {
		if len(bad.Diagnostics) < 100 {
			bad.Diagnostics = append(bad.Diagnostics, Diagnostic{line, path, code})
		}
	}
	if !ValidSource(source) {
		add(1, "source", "SOURCE")
		return nil, bad
	}
	h := sha256.New()
	limited := &io.LimitedReader{R: r, N: MaxBytes + 1}
	scanner := bufio.NewScanner(io.TeeReader(limited, h))
	scanner.Buffer(make([]byte, 4096), MaxLine+2)
	doc := &Document{}
	seen := map[string]bool{}
	line := 0
	for scanner.Scan() {
		line++
		raw := scanner.Bytes()
		if len(raw) > MaxLine {
			add(line, "", "LINE_LIMIT")
			continue
		}
		if line > MaxRecords+1 {
			add(line, "", "RECORD_LIMIT")
			break
		}
		if err := CheckJSON(raw, 32); err != nil {
			add(line, "", "JSON")
			continue
		}
		if line == 1 {
			if err := strict(raw, &doc.Manifest); err != nil || doc.Manifest.Format != "accord-jsonl" || doc.Manifest.Version != 1 || doc.Manifest.Source != source || doc.Manifest.HighWater < 0 || doc.Manifest.ConfigVersion < 0 {
				add(line, "manifest", "MANIFEST")
			}
			continue
		}
		var rec Record
		if err := strict(raw, &rec); err != nil || rec.Kind == "" || rec.ExternalID == "" || len(rec.ExternalID) > 256 || bytes.IndexByte([]byte(rec.ExternalID), 0) >= 0 {
			add(line, "record", "RECORD")
			continue
		}
		rec.Line = line
		key := identity(rec.Kind, rec.ExternalID)
		if seen[key] {
			add(line, "external_id", "DUPLICATE_ID")
		}
		seen[key] = true
		if err := validateData(rec); err != nil {
			add(line, "data", "DATA")
			continue
		}
		hash, err := RecordHash(rec)
		if err != nil {
			add(line, "data", "JSON")
			continue
		}
		rec.Hash = hash
		if validator != nil {
			for _, d := range validator.ValidateRecord(rec) {
				add(line, d.Path, d.Code)
			}
		}
		doc.Records = append(doc.Records, rec)
	}
	if scanner.Err() != nil {
		add(line+1, "", "READ_OR_LINE_LIMIT")
	}
	if limited.N == 0 {
		add(line, "", "FILE_LIMIT")
	}
	if line == 0 {
		add(1, "", "MANIFEST")
	}
	for _, d := range ValidateReferences(doc.Records, exists) {
		add(d.Line, d.Path, d.Code)
	}
	if len(bad.Diagnostics) > 0 {
		return nil, bad
	}
	doc.FileSHA256 = hex.EncodeToString(h.Sum(nil))
	return doc, nil
}

// RecordHash canonicalizes object key order without converting integers to floats.
// JSON number lexemes are retained; distinct number spellings remain distinct.
func RecordHash(r Record) (string, error) {
	if err := CheckJSON(r.Data, 32); err != nil {
		return "", err
	}
	d := json.NewDecoder(bytes.NewReader(r.Data))
	d.UseNumber()
	var value any
	if err := d.Decode(&value); err != nil {
		return "", err
	}
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	canonical, err := json.Marshal(struct {
		Kind string          `json:"kind"`
		ID   string          `json:"external_id"`
		Data json.RawMessage `json:"data"`
	}{r.Kind, r.ExternalID, data})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canonical)
	return hex.EncodeToString(sum[:]), nil
}

package interchange

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
)

// NextRecord streams a sanitized snapshot supplied by the store. Config values,
// credentials and active authority have no fields in the allowed DTOs.
type NextRecord func() (Record, error)

func Encode(w io.Writer, manifest Manifest, next NextRecord) error {
	if manifest.Format != "accord-jsonl" || manifest.Version != 1 || !ValidSource(manifest.Source) || manifest.HighWater < 0 || manifest.ConfigVersion < 0 {
		return errors.New("invalid manifest")
	}
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(manifest); err != nil {
		return err
	}
	for {
		r, err := next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if r.ExternalID == "" || len(r.ExternalID) > 256 || CheckJSON(r.Data, 32) != nil || validateData(r) != nil {
			return errors.New("invalid export record")
		}
		// Re-marshal the strict DTO to ensure only the explicit portability schema is
		// emitted. validateData rejects unrecognized credential/authority fields.
		raw, err := json.Marshal(r)
		if err != nil {
			return err
		}
		if len(raw) > MaxLine {
			return errors.New("export record exceeds line limit")
		}
		if err := enc.Encode(r); err != nil {
			return err
		}
	}
}

// ExportFile atomically publishes a private staged file without replacing an
// existing path. A successful return includes file and directory fsync. Errors
// after publication may leave a complete file; callers must not claim rollback.
func ExportFile(path string, manifest Manifest, next NextRecord) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".accord-export-*")
	if err != nil {
		return err
	}
	stage := f.Name()
	defer os.Remove(stage)
	if err = f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if err = Encode(f, manifest, next); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Link(stage, path); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}

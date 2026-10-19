package web

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"
)

// Startup discards incomplete publications; it never guesses an orphan was accepted.
// Ready records are checked before readiness, so a missing or altered blob fails closed.
func (d *designStore) recover(ctx context.Context) error {
	rows, err := d.db.QueryContext(ctx, `SELECT `+designColumns+` FROM design_assets`)
	if err != nil {
		return err
	}
	var records []storedDesign
	for rows.Next() {
		v, e := scanDesign(rows)
		if e != nil {
			rows.Close()
			return e
		}
		records = append(records, v)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	keep := make(map[string]bool)
	for _, v := range records {
		if v.Status == "pending" {
			if err = d.abort(v); err != nil {
				return err
			}
			continue
		}
		f, e := designOpen(d.dir, "blobs", v.Blob, os.O_RDONLY)
		if e != nil {
			return fmt.Errorf("ready design blob unavailable: %w", e)
		}
		info, e := f.Stat()
		if e != nil {
			f.Close()
			return e
		}
		h := sha256.New()
		n, e := io.Copy(h, io.LimitReader(f, maxDesignFile+1))
		f.Close()
		if e != nil || n != v.Size || info.Size() != v.Size || hex.EncodeToString(h.Sum(nil)) != v.SHA256 {
			return fmt.Errorf("ready design blob integrity failure")
		}
		keep[v.Blob] = true
		if v.Preview != "" {
			f, e = designOpen(d.dir, "blobs", v.Preview, os.O_RDONLY)
			if e != nil {
				return e
			}
			fi, e := f.Stat()
			f.Close()
			if e != nil {
				return e
			}
			if fi.Size() != v.PreviewSize {
				return fmt.Errorf("design preview integrity failure")
			}
			keep[v.Preview] = true
		}
	}
	for _, part := range []string{"staging", "blobs"} {
		root, e := designRoot(d.dir, part)
		if e != nil {
			return e
		}
		f, e := root.Open(".")
		if e != nil {
			root.Close()
			return e
		}
		entries, e := f.ReadDir(-1)
		f.Close()
		root.Close()
		if e != nil {
			return e
		}
		for _, entry := range entries {
			if part == "blobs" && keep[entry.Name()] {
				continue
			}
			if !designStorageName(entry.Name()) {
				return fmt.Errorf("unexpected entry in private design storage")
			}
			if err = designRemove(d.dir, part, entry.Name()); err != nil {
				return err
			}
		}
	}
	d.lastCleanup = time.Now().UTC()
	d.cleanupError = ""
	return nil
}

// Live cleanup only handles old, unreferenced staging entries. Active uploads hold
// pending rows, which are excluded; ready blobs are immutable and never collected.
func (d *designStore) cleanup(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if time.Since(d.lastCleanup) < 15*time.Minute {
		return nil
	}
	err := d.cleanupStaging(ctx)
	if err != nil {
		d.cleanupError = err.Error()
		return err
	}
	d.lastCleanup = time.Now().UTC()
	d.cleanupError = ""
	return nil
}

func (d *designStore) cleanupStaging(ctx context.Context) error {
	root, err := designRoot(d.dir, "staging")
	if err != nil {
		return err
	}
	defer root.Close()
	f, err := root.Open(".")
	if err != nil {
		return err
	}
	entries, err := f.ReadDir(-1)
	f.Close()
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !designStorageName(entry.Name()) {
			return fmt.Errorf("unexpected staging entry")
		}
		fi, e := root.Lstat(entry.Name())
		if e != nil {
			return e
		}
		if fi.ModTime().After(time.Now().Add(-24 * time.Hour)) {
			continue
		}
		var count int
		e = d.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM design_assets WHERE status='pending' AND (blob=? OR blob||'.preview'=?)`, entry.Name(), entry.Name()).Scan(&count)
		if e != nil {
			return e
		}
		if count == 0 {
			if e = root.Remove(entry.Name()); e != nil {
				return e
			}
		}
	}
	return nil
}

// DesignStorageStatus exposes counts and cleanup health, never file paths or secrets.
type DesignStorageStatus struct {
	Used, Reserved int64
	LastCleanup    time.Time
	CleanupError   string
}

func (s *Server) DesignStorageStatus(ctx context.Context) (DesignStorageStatus, error) {
	d := s.designs
	d.mu.Lock()
	defer d.mu.Unlock()
	used, reserved, err := d.totals(ctx)
	return DesignStorageStatus{used, reserved, d.lastCleanup, d.cleanupError}, err
}

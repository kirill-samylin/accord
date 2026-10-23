package worker

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
)

// These local read-only helpers produce hashes for review. They never enable a
// preset, rewrite configuration, enroll a host, or change a server grant.
func AutomationIdentity(path string) (string, error) {
	if err := validateLocalPath(path); err != nil {
		return "", err
	}
	return directoryIdentity(path)
}
func AutomationDependencyDigest(ctx context.Context, directory, root string) (string, error) {
	if validateLocalPath(directory) != nil || !consentRelative(root) {
		return "", consentError()
	}
	var records []string
	count := 0
	var bytes int64
	err := filepath.WalkDir(filepath.Join(directory, root), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		count++
		if count > 100000 {
			return errors.New("dependency capacity exceeded")
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(directory, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if info.Mode()&os.ModeSymlink != 0 {
			record, _, err := dependencyLink(directory, rel, []string{root})
			if err != nil {
				return err
			}
			records = append(records, record)
			return nil
		}
		if info.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return consentError()
		}
		bytes += info.Size()
		if bytes > 1<<30 {
			return errors.New("dependency bytes capacity exceeded")
		}
		digest, err := hashRegular(path, 256<<20)
		if err != nil {
			return err
		}
		records = append(records, rel+"\x00"+strconv.FormatUint(uint64(info.Mode().Perm()), 8)+"\x00"+strconv.FormatInt(info.Size(), 10)+"\x00"+digest)
		return nil
	})
	if err != nil {
		return "", err
	}
	return digestRecords(records), nil
}

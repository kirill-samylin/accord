package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func belongs(path string, roots []string) bool {
	for _, root := range roots {
		if withinRoot(path, filepath.ToSlash(root)) {
			return true
		}
	}
	return false
}
func sourceManifest(ctx context.Context, directory string, a AutomationWorkspace, tracked map[string]gitEntry) (string, string, map[string]string, error) {
	var source, unknown []string
	deps := map[string][]string{}
	var bytes int64
	count := 0
	err := filepath.WalkDir(directory, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		rel, e := filepath.Rel(directory, path)
		if e != nil {
			return e
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if rel == ".git" {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		info, e := d.Info()
		if e != nil {
			return e
		}
		if info.Mode()&os.ModeSymlink != 0 {
			count++
			if count > 100000 {
				return errors.New("source manifest capacity exceeded")
			}
			if _, trackedLink := tracked[rel]; trackedLink || !belongs(rel, a.DependencyRoots) {
				return errors.New("source symlinks require an explicit supported boundary")
			}
			record, root, linkErr := dependencyLink(directory, rel, a.DependencyRoots)
			if linkErr != nil {
				return linkErr
			}
			deps[root] = append(deps[root], record)
			return nil
		}
		if !info.IsDir() && !info.Mode().IsRegular() {
			return errors.New("unsupported source filesystem entry")
		}
		if info.IsDir() {
			return nil
		}
		count++
		bytes += info.Size()
		if count > 100000 || bytes > 1<<30 || info.Size() > 256<<20 {
			return errors.New("source manifest capacity exceeded")
		}
		_, isTracked := tracked[rel]
		artifact := belongs(rel, a.ArtifactRoots)
		dependency := belongs(rel, a.DependencyRoots)
		if isTracked && (artifact || dependency || !belongs(rel, a.SourceRoots)) {
			return errors.New("tracked source crosses approved roots")
		}
		if artifact {
			return nil
		}
		digest, e := hashRegular(path, 256<<20)
		if e != nil {
			return e
		}
		record := rel + "\x00" + strconv.FormatUint(uint64(info.Mode().Perm()), 8) + "\x00" + strconv.FormatInt(info.Size(), 10) + "\x00" + digest
		if dependency {
			for _, root := range a.DependencyRoots {
				if withinRoot(rel, filepath.ToSlash(root)) {
					deps[root] = append(deps[root], record)
				}
			}
			return nil
		}
		source = append(source, record)
		if !isTracked {
			unknown = append(unknown, record)
		}
		return nil
	})
	if err != nil {
		return "", "", nil, err
	}
	observed := map[string]string{}
	for _, root := range a.DependencyRoots {
		digest := digestRecords(deps[root])
		if digest != a.DependencySHA256[root] {
			return "", "", nil, errors.New("approved dependency bytes changed")
		}
		observed[root] = digest
	}
	return digestRecords(source), digestRecords(unknown), observed, nil
}
func hashRegular(path string, max int64) (string, error) {
	info, e := os.Lstat(path)
	if e != nil || !info.Mode().IsRegular() || info.Size() > max {
		return "", errors.New("unsupported bounded artifact")
	}
	f, e := os.Open(path)
	if e != nil {
		return "", e
	}
	defer f.Close()
	opened, e := f.Stat()
	if e != nil || !os.SameFile(info, opened) || !opened.Mode().IsRegular() {
		return "", errors.New("artifact identity changed")
	}
	h := sha256.New()
	n, e := io.Copy(h, io.LimitReader(f, max+1))
	if e != nil || n > max {
		return "", errors.New("artifact capacity exceeded")
	}
	after, e := f.Stat()
	if e != nil || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
		return "", errors.New("artifact changed while observing")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
func unchangedProof(before, after gitProof, readOnly bool) error {
	if before.Unknown != after.Unknown {
		return errors.New("untracked or ignored source boundary changed")
	}
	if readOnly && (before.Head != after.Head || before.Manifest != after.Manifest) {
		return errors.New("read-only stage changed reviewed source")
	}
	return nil
}

// AutomationPresetDigest is for explicit local setup. It does not enable a
// preset or transmit executable paths to Accord.
func AutomationPresetDigest(workspace, identity, id string, a AutomationWorkspace, p CheckPreset) string {
	return presetConsentDigest(workspace, identity, id, a, p)
}
func safeRelativeFile(directory, relative string) (string, error) {
	if !consentRelative(relative) {
		return "", consentError()
	}
	path := filepath.Join(directory, relative)
	current := directory
	for _, component := range strings.Split(relative, string(filepath.Separator)) {
		current = filepath.Join(current, component)
		info, e := os.Lstat(current)
		if e != nil || info.Mode()&os.ModeSymlink != 0 {
			return "", errors.New("artifact path must not follow links")
		}
	}
	return path, nil
}

func dependencyLink(directory, relative string, roots []string) (string, string, error) {
	path := filepath.Join(directory, filepath.FromSlash(relative))
	target, err := os.Readlink(path)
	if err != nil || filepath.IsAbs(target) {
		return "", "", errors.New("dependency links must be relative and internal")
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", "", err
	}
	resolvedRelative, err := filepath.Rel(directory, resolved)
	if err != nil || !consentRelative(resolvedRelative) {
		return "", "", consentError()
	}
	for _, root := range roots {
		if withinRoot(relative, filepath.ToSlash(root)) && withinRoot(filepath.ToSlash(resolvedRelative), filepath.ToSlash(root)) {
			return relative + "\x00link\x00" + filepath.ToSlash(target), root, nil
		}
	}
	return "", "", errors.New("dependency link escaped its approved root")
}

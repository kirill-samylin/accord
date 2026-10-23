package worker

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"
)

// A test design is an unexecuted proposal. Its hash cannot qualify passed tests;
// a developer must apply/commit it and independent review must approve the SHA.
func artifactPatches(ctx context.Context, directory string, a AutomationWorkspace) (map[string]string, error) {
	result := map[string]string{}
	count := 0
	for _, root := range a.ArtifactRoots {
		path := filepath.Join(directory, root)
		if _, err := os.Lstat(path); os.IsNotExist(err) {
			continue
		}
		err := filepath.WalkDir(path, func(path string, d fs.DirEntry, e error) error {
			if e != nil {
				return e
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			count++
			if count > 10000 {
				return errors.New("artifact directory capacity exceeded")
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(d.Name(), ".patch") {
				return nil
			}
			if d.Type()&os.ModeSymlink != 0 || len(result) >= 128 {
				return errors.New("test design artifact boundary exceeded")
			}

			rel, e := filepath.Rel(directory, path)
			if e != nil {
				return e
			}
			hash, e := hashRegular(path, 1<<20)
			if e != nil {
				return e
			}
			result[rel] = hash
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
func validDesignPatch(raw []byte, roots []string) bool {
	if !utf8.Valid(raw) || len(raw) == 0 || strings.ContainsRune(string(raw), 0) {
		return false
	}
	headers, hunks := 0, 0
	for _, line := range strings.Split(string(raw), "\n") {
		if strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "--- ") {
			name := strings.TrimSpace(line[4:])
			if name == "/dev/null" {
				continue
			}
			if strings.ContainsAny(name, "\t\r") {
				return false
			}
			if len(name) < 3 || (name[:2] != "a/" && name[:2] != "b/") {
				return false
			}
			relative := filepath.FromSlash(name[2:])
			if !consentRelative(relative) || !belongs(filepath.ToSlash(relative), roots) {
				return false
			}
			headers++
		}
		if strings.HasPrefix(line, "@@ ") && strings.Contains(line[3:], " @@") {
			hunks++
		}
	}
	return headers > 0 && hunks > 0
}
func designArtifactRefs(before, after map[string]string) []string {
	var refs []string
	seen := map[string]bool{}
	for path, digest := range after {
		if before[path] != digest && !seen[digest] {
			refs = append(refs, "artifact:"+digest)
			seen[digest] = true
		}
	}
	sort.Strings(refs)
	return refs
}
func automationSandbox(config Config, role, workspace string) string {
	if role == "developer" {
		return config.CodexSandbox
	}
	if role == "tester" && config.Automation != nil && config.Automation.Workspaces[workspace].AllowTesterArtifactWrites && config.CodexSandbox == "workspace-write" {
		return "workspace-write"
	}
	return "read-only"
}

func qualifyingDesignRefs(ctx context.Context, directory string, a AutomationWorkspace, before, after map[string]string) []string {
	var refs []string
	seen := map[string]bool{}
	for path, digest := range after {
		if ctx.Err() != nil {
			return nil
		}
		if before[path] == digest || seen[digest] {
			continue
		}
		full, err := safeRelativeFile(directory, path)
		if err != nil {
			continue
		}
		raw, err := readBoundedReport(full)
		if err != nil || len(raw) > 1<<20 || !validDesignPatch(raw, a.SourceRoots) {
			continue
		}
		current, err := hashRegular(full, 1<<20)
		if err != nil || current != digest {
			continue
		}
		refs = append(refs, "artifact:"+digest)
		if len(refs) > 8 {
			return nil
		}
		seen[digest] = true
	}
	sort.Strings(refs)
	return refs
}

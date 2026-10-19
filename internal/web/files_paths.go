package web

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func designValidID(v string) bool {
	if len(v) != 48 {
		return false
	}
	for _, c := range v {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func designStorageName(v string) bool {
	return designValidID(v) || strings.HasSuffix(v, ".preview") && designValidID(strings.TrimSuffix(v, ".preview"))
}

func designPrivateDir(path string) error {
	if err := os.Mkdir(path, 0700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	fi, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !fi.IsDir() || fi.Mode()&os.ModeSymlink != 0 || fi.Mode().Perm() != 0700 {
		return fmt.Errorf("unsafe private design directory")
	}
	if !designOwned(fi) {
		return fmt.Errorf("design directory has unsafe owner")
	}
	return nil
}

func designRoot(dir, part string) (*os.Root, error) {
	if part != "blobs" && part != "staging" {
		return nil, fmt.Errorf("invalid design root")
	}
	path := filepath.Join(dir, part)
	if err := designPrivateDir(path); err != nil {
		return nil, err
	}
	return os.OpenRoot(path)
}

func designOpen(dir, part, name string, flags int) (*os.File, error) {
	if !designStorageName(name) {
		return nil, fmt.Errorf("invalid design storage name")
	}
	root, err := designRoot(dir, part)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	f, err := root.OpenFile(name, flags|designNoFollow, 0600)
	if err != nil {
		return nil, err
	}
	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if !fi.Mode().IsRegular() || fi.Mode().Perm() != 0600 || !designOwned(fi) {
		f.Close()
		return nil, fmt.Errorf("unsafe design file")
	}
	return f, nil
}

func designRemove(dir, part, name string) error {
	root, err := designRoot(dir, part)
	if err != nil {
		return err
	}
	defer root.Close()
	if !designStorageName(name) {
		return fmt.Errorf("invalid design storage name")
	}
	if err = root.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func designSyncDir(dir, part string) error {
	root, err := designRoot(dir, part)
	if err != nil {
		return err
	}
	defer root.Close()
	f, err := root.Open(".")
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Sync()
}

// Hard link publication is atomic and cannot overwrite an existing immutable blob.
func designPublish(dir, name string) error {
	if !designStorageName(name) {
		return fmt.Errorf("invalid design storage name")
	}
	if err := designPrivateDir(filepath.Join(dir, "staging")); err != nil {
		return err
	}
	if err := designPrivateDir(filepath.Join(dir, "blobs")); err != nil {
		return err
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		return err
	}
	defer root.Close()
	if err = root.Link("staging/"+name, "blobs/"+name); err != nil {
		return err
	}
	if err = designSyncDir(dir, "blobs"); err != nil {
		return err
	}
	if err = root.Remove("staging/" + name); err != nil {
		return err
	}
	return designSyncDir(dir, "staging")
}

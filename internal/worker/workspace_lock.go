package worker

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type workspaceLock struct{ file *os.File }

var errForeignIntent = errors.New("checkout intent belongs to another launch")

func acquireWorkspace(directory, lockRoot, intent string) (*workspaceLock, error) {
	identity, err := directoryIdentity(directory)
	if err != nil {
		return nil, errors.New("cannot establish local workspace filesystem identity")
	}
	if lockRoot == "" {
		configRoot, err := os.UserConfigDir()
		if err != nil {
			return nil, err
		}
		lockRoot = filepath.Join(configRoot, "Accord", "worker-locks")
	}
	if err = privateDirectory(lockRoot); err != nil {
		return nil, err
	}
	path := filepath.Join(lockRoot, identity+".lock")
	if _, err = os.Lstat(path); err == nil {
		f, e := privateRead(path)
		if e != nil {
			return nil, e
		}
		f.Close()
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	if err = lockFile(file); err != nil {
		file.Close()
		return nil, errors.New("this checkout is already owned by another local worker")
	}
	data, err := io.ReadAll(io.LimitReader(file, 1025))
	if err != nil || len(data) > 1024 || len(data) > 0 && string(data) != "idle" {
		unlockFile(file)
		file.Close()
		return nil, errors.New("this checkout has an unresolved previous launch; local recovery required")
	}
	lock := &workspaceLock{file: file}
	if err = lock.write("intent " + intent); err != nil {
		unlockFile(file)
		file.Close()
		return nil, err
	}
	if err = syncDirectory(lockRoot); err != nil {
		unlockFile(file)
		file.Close()
		return nil, err
	}
	return lock, nil
}

func (l *workspaceLock) write(value string) error {
	if err := l.file.Truncate(0); err != nil {
		return err
	}
	if _, err := l.file.Seek(0, 0); err != nil {
		return err
	}
	if _, err := io.WriteString(l.file, value); err != nil {
		return err
	}
	return l.file.Sync()
}

func (l *workspaceLock) release(provenExit bool) error {
	var err error
	if provenExit {
		err = l.write("idle")
	}
	unlockFile(l.file)
	closeErr := l.file.Close()
	if err != nil {
		return err
	}
	return closeErr
}

// resetWorkspaceIntent returns a checkout to idle only when its intent names
// exactly this run and no live worker holds the lock. A foreign or unreadable
// intent is never overwritten.
func resetWorkspaceIntent(directory, lockRoot, runID string) error {
	identity, err := directoryIdentity(directory)
	if err != nil {
		return errors.New("cannot establish local workspace filesystem identity")
	}
	if lockRoot == "" {
		configRoot, err := os.UserConfigDir()
		if err != nil {
			return err
		}
		lockRoot = filepath.Join(configRoot, "Accord", "worker-locks")
	}
	path := filepath.Join(lockRoot, identity+".lock")
	if _, err = os.Lstat(path); os.IsNotExist(err) {
		return nil
	}
	f, err := privateRead(path)
	if err != nil {
		return err
	}
	f.Close()
	file, err := os.OpenFile(path, os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	if err = lockFile(file); err != nil {
		file.Close()
		return errors.New("this checkout is already owned by another local worker")
	}
	lock := &workspaceLock{file: file}
	data, err := io.ReadAll(io.LimitReader(file, 1025))
	text := string(data)
	if err != nil || len(data) > 1024 || (text != "idle" && text != "" && !(strings.HasPrefix(text, "intent ") && strings.HasSuffix(text, " "+runID))) {
		_ = lock.release(false)
		return errForeignIntent
	}
	return lock.release(true)
}

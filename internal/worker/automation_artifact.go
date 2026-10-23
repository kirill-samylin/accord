package worker

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
)

func readArtifact(path string, max int64) ([]byte, string, error) {
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() > max {
		return nil, "", errors.New("bounded regular artifact required")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, "", err
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !os.SameFile(info, opened) {
		return nil, "", errors.New("artifact identity changed")
	}
	raw, err := io.ReadAll(io.LimitReader(file, max+1))
	after, statErr := file.Stat()
	if err != nil || statErr != nil || int64(len(raw)) > max || int64(len(raw)) != info.Size() || after.Size() != info.Size() || !after.ModTime().Equal(info.ModTime()) {
		return nil, "", errors.New("artifact changed during observation")
	}
	digest := sha256.Sum256(raw)
	return raw, hex.EncodeToString(digest[:]), nil
}
func (e *Engine) persistCheckArtifact(id, checkID, format string, raw []byte) error {
	e.mu.Lock()
	run := e.journal.Runs[id]
	if run == nil {
		e.mu.Unlock()
		return consentError()
	}
	dir := filepath.Join(e.config.StateDir, "runs", run.ID, "attempt-"+strconv.FormatInt(run.Generation, 10), "checks", checkID)
	e.mu.Unlock()
	if err := privateDirectory(dir); err != nil {
		return err
	}
	path := filepath.Join(dir, "report."+format)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	_, err = file.Write(raw)
	if err == nil {
		err = file.Sync()
	}
	closed := file.Close()
	if err != nil {
		return err
	}
	if closed != nil {
		return closed
	}
	return syncDirectory(dir)
}

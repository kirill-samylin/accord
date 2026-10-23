package worker

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"

	"agent-board/internal/core"
)

type RunState struct {
	Automation      *automationAttempt `json:"automation,omitempty"`
	ID              string             `json:"id"`
	Provider        string             `json:"provider"`
	Workspace       string             `json:"workspace"`
	Generation      int64              `json:"generation"`
	State           string             `json:"state"`
	SessionID       string             `json:"session_id,omitempty"`
	CommandID       string             `json:"command_id"`
	Output          string             `json:"output,omitempty"`
	Error           string             `json:"error,omitempty"`
	LocalDiagnostic string             `json:"local_diagnostic,omitempty"`
}

type Receipt struct {
	RecoveryWorkerID string `json:"recovery_worker_id,omitempty"`
	Forced           bool   `json:"forced,omitempty"`
	RequestID        string `json:"request_id"`
	CommandID        string `json:"command_id"`
	RunID            string `json:"run_id"`
	DispatchID       string `json:"dispatch_id"`
	Generation       int64  `json:"generation"`
	Phase            string `json:"phase"`
	State            string `json:"state,omitempty"`
	SessionID        string `json:"provider_session_id,omitempty"`
	Error            string `json:"error,omitempty"`
	Output           string `json:"output,omitempty"`
}

type journal struct {
	AutomationBudgets map[string]*automationBudget `json:"automation_budgets,omitempty"`
	PathPins          map[string]pathPin           `json:"path_pins"`
	OwnerPrincipalID  string                       `json:"owner_principal_id"`
	Server            string                       `json:"server"`
	Project           string                       `json:"project"`
	Machine           string                       `json:"machine"`
	Enrollment        *core.ManagedHostEnrollInput `json:"enrollment,omitempty"`
	Consent           *core.ManagedHostUpdateInput `json:"consent,omitempty"`
	HostID            string                       `json:"host_id,omitempty"`
	HostSecret        string                       `json:"host_secret"`
	EnrollRequestID   string                       `json:"enroll_request_id"`
	WorkerID          string                       `json:"worker_id"`
	Epoch             int64                        `json:"epoch"`
	Runs              map[string]*RunState         `json:"runs"`
	Receipts          map[string]*Receipt          `json:"receipts"`
	Pending           []*Receipt                   `json:"pending"`
}

type disk struct {
	dir  string
	lock *os.File
}

func openDisk(dir string) (*disk, journal, error) {
	j := journal{Runs: map[string]*RunState{}, Receipts: map[string]*Receipt{}}
	if err := privateDirectory(dir); err != nil {
		return nil, j, err
	}
	path := filepath.Join(dir, "worker.lock")
	if _, err := os.Lstat(path); err == nil {
		f, e := privateRead(path)
		if e != nil {
			return nil, j, e
		}
		f.Close()
	} else if !os.IsNotExist(err) {
		return nil, j, err
	}
	lock, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, j, err
	}
	if err = lockFile(lock); err != nil {
		lock.Close()
		return nil, j, errors.New("worker state is already in use")
	}
	d := &disk{dir: dir, lock: lock}
	file, err := privateRead(filepath.Join(dir, "state.json"))
	if err == nil {
		decoder := json.NewDecoder(io.LimitReader(file, 8<<20))
		decoder.DisallowUnknownFields()
		err = decoder.Decode(&j)
		if err == nil && decoder.Decode(new(any)) != io.EOF {
			err = errors.New("trailing journal data")
		}
		file.Close()
		if err != nil || j.Runs == nil || j.Receipts == nil {
			d.close()
			return nil, j, errors.New("invalid worker journal; manual recovery required")
		}
	} else if !os.IsNotExist(err) {
		d.close()
		return nil, j, err
	}
	// Each process lifetime has a distinct incarnation. Old dispatches cannot be
	// silently acknowledged by a restarted process with no ownership handle.
	j.WorkerID = randomID("worker_")
	if j.HostSecret == "" {
		j.HostSecret = randomID("") + randomID("")
		j.EnrollRequestID = randomID("enroll_")
	}
	for _, run := range j.Runs {
		if a := run.Automation; a != nil && (len(a.Pending) > 0 || (a.NativeExited && (!a.ExitAck || !a.LockReleased))) {
			run.State = "needs_recovery"
			run.Error = "workflow proof or checkout release was not acknowledged before restart; local recovery required"
		}
		if run.State == "accepted" || run.State == "launching" || run.State == "running" || run.State == "stopping" {
			run.State = "needs_recovery"
			run.Error = "worker restarted after a possible launch; local recovery required"
		}
	}
	if err = d.save(j); err != nil {
		d.close()
		return nil, j, err
	}
	return d, j, nil
}

func randomID(prefix string) string {
	var data [16]byte
	if _, err := rand.Read(data[:]); err != nil {
		panic("operating system randomness unavailable")
	}
	return prefix + hex.EncodeToString(data[:])
}

func (d *disk) save(j journal) error {
	data, err := json.Marshal(j)
	if err != nil {
		return err
	}
	if len(data) >= 8<<20 {
		return errors.New("private worker journal capacity reached; local maintenance required")
	}
	file, err := os.OpenFile(filepath.Join(d.dir, "state.next"), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if os.IsExist(err) {
		return errors.New("incomplete journal write; local recovery required")
	}
	if err != nil {
		return err
	}
	name := file.Name()
	if _, err = file.Write(data); err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err == nil {
		err = closeErr
	}
	if err != nil {
		os.Remove(name)
		return err
	}
	if err = replaceFile(name, filepath.Join(d.dir, "state.json")); err != nil {
		return err
	}
	return syncDirectory(d.dir)
}

func (d *disk) close() { unlockFile(d.lock); d.lock.Close() }

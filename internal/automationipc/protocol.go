// Package automationipc is a per-attempt local verification channel. It accepts
// fixed preset identifiers, never executable paths, arguments or commands.
package automationipc

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"time"

	"agent-board/internal/transport"
)

const MaxFrame = 64 << 10
const MaxResponse = 16 << 10

type Binding struct {
	WorkflowID   string `json:"workflow_id"`
	StepID       string `json:"step_id"`
	RunID        string `json:"run_id"`
	ControlEpoch int64  `json:"control_epoch"`
	Generation   int64  `json:"generation"`
}
type Config struct {
	Endpoint string  `json:"endpoint"`
	Nonce    string  `json:"nonce"`
	Binding  Binding `json:"binding"`
}
type Request struct {
	Version        int    `json:"v"`
	Nonce          string `json:"nonce"`
	RequestID      string `json:"request_id"`
	Operation      string `json:"operation"`
	WorkflowID     string `json:"workflow_id"`
	StepID         string `json:"step_id"`
	RunID          string `json:"run_id"`
	ControlEpoch   int64  `json:"control_epoch"`
	Generation     int64  `json:"generation"`
	SessionID      string `json:"session_id"`
	TaskKey        string `json:"task_key"`
	TaskGeneration int64  `json:"task_generation"`
	PresetID       string `json:"preset_id"`
}
type Result struct {
	ExitCode       *int     `json:"exit_code,omitempty"`
	Passed         int64    `json:"passed"`
	Failed         int64    `json:"failed"`
	Skipped        int64    `json:"skipped"`
	ReportKind     string   `json:"report_kind"`
	ReportSHA256   string   `json:"report_sha256,omitempty"`
	ActualHead     string   `json:"actual_head,omitempty"`
	ManifestSHA256 string   `json:"manifest_sha256,omitempty"`
	ArtifactRefs   []string `json:"artifact_refs,omitempty"`
}
type Response struct {
	CheckID string  `json:"check_id,omitempty"`
	Status  string  `json:"status"`
	Result  *Result `json:"result,omitempty"`
	Error   string  `json:"error,omitempty"`
}
type Handler func(context.Context, Request) (Response, error)

func secret() (string, error) {
	var raw [32]byte
	if _, e := rand.Read(raw[:]); e != nil {
		return "", e
	}
	return hex.EncodeToString(raw[:]), nil
}
func valid(c Config, r Request) bool {
	return c.Binding.WorkflowID != "" && c.Binding.StepID != "" && c.Binding.RunID != "" && c.Binding.ControlEpoch > 0 && c.Binding.Generation > 0 && r.Version == 1 && len(c.Nonce) == 64 && len(r.Nonce) == 64 && subtle.ConstantTimeCompare([]byte(c.Nonce), []byte(r.Nonce)) == 1 &&
		r.WorkflowID == c.Binding.WorkflowID && r.StepID == c.Binding.StepID && r.RunID == c.Binding.RunID && r.ControlEpoch == c.Binding.ControlEpoch && r.Generation == c.Binding.Generation &&
		(r.Operation == "start" || r.Operation == "status") && len(r.RequestID) > 0 && len(r.RequestID) <= 128 && len(r.PresetID) > 0 && len(r.PresetID) <= 128 && len(r.SessionID) <= 128 && len(r.TaskKey) <= 128 && r.TaskGeneration > 0
}
func readFrame(r io.Reader, value any) error { return readFrameBound(r, value, MaxFrame) }
func readFrameBound(r io.Reader, value any, max uint32) error {
	var n [4]byte
	if _, e := io.ReadFull(r, n[:]); e != nil {
		return e
	}
	size := binary.BigEndian.Uint32(n[:])
	if size == 0 || size > max {
		return errors.New("invalid local frame")
	}
	raw := make([]byte, size)
	if _, e := io.ReadFull(r, raw); e != nil {
		return e
	}
	if transport.ValidateJSONFields(raw) != nil {
		return errors.New("invalid local JSON")
	}
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if e := d.Decode(value); e != nil {
		return errors.New("invalid local frame fields")
	}
	if d.Decode(new(any)) != io.EOF {
		return errors.New("trailing local frame")
	}
	return nil
}
func writeFrame(w io.Writer, value any) error {
	raw, e := json.Marshal(value)
	if e != nil {
		return e
	}
	if len(raw) > MaxFrame {
		return errors.New("oversized local frame")
	}
	var n [4]byte
	binary.BigEndian.PutUint32(n[:], uint32(len(raw)))
	if _, e = io.Copy(w, bytes.NewReader(n[:])); e != nil {
		return e
	}
	_, e = io.Copy(w, bytes.NewReader(raw))
	return e
}

// Call performs one bounded exchange. A retry must retain RequestID and payload.
func Call(ctx context.Context, c Config, r Request) (Response, error) {
	var out Response
	r.Version = 1
	r.Nonce = c.Nonce
	r.WorkflowID, r.StepID, r.RunID = c.Binding.WorkflowID, c.Binding.StepID, c.Binding.RunID
	r.ControlEpoch, r.Generation = c.Binding.ControlEpoch, c.Binding.Generation
	if !valid(c, r) {
		return out, errors.New("invalid immutable check binding")
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	conn, e := dial(ctx, c.Endpoint)
	if e != nil {
		return out, errors.New("local check endpoint unavailable")
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if e = writeFrame(conn, r); e != nil {
		return out, errors.New("local check request failed")
	}
	if e = readFrameBound(conn, &out, MaxResponse); e != nil {
		return out, errors.New("local check response unavailable")
	}
	return out, nil
}

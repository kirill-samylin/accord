package worker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"agent-board/internal/transport"
)

type configFrames struct {
	reader        *bufio.Reader
	bytes, frames int
}

func (f *configFrames) next() (map[string]json.RawMessage, error) {
	var line []byte
	for {
		part, err := f.reader.ReadSlice('\n')
		f.bytes += len(part)
		if len(line)+len(part) > 2<<20 || f.bytes > 4<<20 {
			return nil, permissionError()
		}
		line = append(line, part...)
		if err == bufio.ErrBufferFull {
			continue
		}
		if err != nil {
			if err == io.EOF && len(line) == 0 {
				return nil, io.EOF
			}
			return nil, permissionError()
		}
		break
	}
	f.frames++
	if f.frames > 35 || transport.ValidateJSONFields(line) != nil {
		return nil, permissionError()
	}
	var m map[string]json.RawMessage
	if json.Unmarshal(line, &m) != nil || m == nil {
		return nil, permissionError()
	}
	for k := range m {
		switch k {
		case "id", "result", "error", "method", "params", "emittedAtMs":
		default:
			return nil, permissionError()
		}
	}
	if stamp, present := m["emittedAtMs"]; present {
		var milliseconds int64
		_, hasID := m["id"]
		if hasID || bytes.Equal(bytes.TrimSpace(stamp), []byte("null")) || json.Unmarshal(stamp, &milliseconds) != nil {
			return nil, permissionError()
		}
	}
	return m, nil
}

func passiveConfigNotification(m map[string]json.RawMessage) bool {
	_, hasID := m["id"]
	var method string
	return !hasID && m["result"] == nil && m["error"] == nil && json.Unmarshal(m["method"], &method) == nil &&
		(method == "configWarning" || method == "remoteControl/status/changed")
}

func (f *configFrames) response(id int) (json.RawMessage, error) {
	for {
		m, err := f.next()
		if err != nil {
			return nil, permissionError()
		}
		if _, ok := m["id"]; !ok {
			if !passiveConfigNotification(m) {
				return nil, permissionError()
			}
			continue // Never log possibly secret warning params.
		}
		var got int
		if json.Unmarshal(m["id"], &got) != nil || got != id || m["method"] != nil || m["params"] != nil || m["error"] != nil || m["result"] == nil {
			return nil, permissionError()
		}
		var object map[string]json.RawMessage
		if json.Unmarshal(m["result"], &object) != nil || object == nil {
			return nil, permissionError()
		}
		return m["result"], nil
	}
}
func configDialogue(input io.WriteCloser, output io.Reader, i configInspection) (permissionProof, error) {
	f := configFrames{reader: bufio.NewReader(output)}
	written := 0
	send := func(value any) error {
		b, err := json.Marshal(value)
		written += len(b) + 1
		if err != nil || written > 8192 {
			return permissionError()
		}
		_, err = input.Write(append(b, '\n'))
		if err != nil {
			return permissionError()
		}
		return nil
	}
	request := func(id int, method string, params any) error {
		return send(map[string]any{"id": id, "method": method, "params": params})
	}
	if request(1, "initialize", map[string]any{"clientInfo": map[string]string{"name": "accord_config_inspection", "version": "1"}}) != nil {
		return permissionProof{}, permissionError()
	}
	raw, err := f.response(1)
	if err != nil {
		return permissionProof{}, err
	}
	var init map[string]any
	if json.Unmarshal(raw, &init) != nil {
		return permissionProof{}, permissionError()
	}
	for _, key := range []string{"userAgent", "platformFamily", "platformOs", "codexHome"} {
		if v, ok := init[key].(string); !ok || v == "" {
			return permissionProof{}, permissionError()
		}
	}
	if send(map[string]any{"method": "initialized", "params": map[string]any{}}) != nil || request(2, "config/read", map[string]any{"cwd": i.Directory, "includeLayers": true}) != nil {
		return permissionProof{}, permissionError()
	}
	config, err := f.response(2)
	if err != nil {
		return permissionProof{}, err
	}
	if request(3, "configRequirements/read", map[string]any{}) != nil {
		return permissionProof{}, permissionError()
	}
	requirements, err := f.response(3)
	if err != nil {
		return permissionProof{}, err
	}
	proof, err := provePermissionConfig(config, requirements, i)
	if err != nil {
		return permissionProof{}, err
	}
	if input.Close() != nil {
		return permissionProof{}, permissionError()
	}
	// Only bounded passive status may follow stdin close. A reply/request never
	// acquires a new protocol slot, and all original frame/deadline limits remain.
	for {
		message, readErr := f.next()
		if readErr == io.EOF {
			break
		}
		if readErr != nil || !passiveConfigNotification(message) {
			return permissionProof{}, permissionError()
		}
	}
	return proof, nil
}

// The helper owns raw stdout. It can emit only the sanitized finite proof.
func inspectWithProcess(ctx context.Context, spec launchSpec, start func(launchSpec) (process, error)) (permissionProof, process, error) {
	if ctx.Err() != nil || spec.inspection == nil || spec.inspection.Directory != spec.directory {
		return permissionProof{}, nil, permissionError()
	}
	r, w := io.Pipe()
	spec.stdin = r
	child, err := start(spec)
	if err != nil {
		r.Close()
		w.Close()
		return permissionProof{}, nil, errors.Join(permissionError(), err)
	}
	type result struct {
		proof permissionProof
		err   error
	}
	finished := make(chan result, 1)
	go func() {
		proof, err := configDialogue(w, child.Output(), *spec.inspection)
		finished <- result{proof, err}
	}()
	var proof permissionProof
	select {
	case out := <-finished:
		proof, err = out.proof, out.err
	case <-ctx.Done():
		err = permissionError()
	}
	w.Close()
	r.Close()
	if err != nil || ctx.Err() != nil {
		cleanupErr := cleanupInspection(child)
		if errors.Is(cleanupErr, errOwnership) {
			return permissionProof{}, child, cleanupErr
		}
		return permissionProof{}, child, permissionError()
	}
	waited := make(chan error, 1)
	go func() { waited <- child.Wait() }()
	select {
	case err = <-waited:
	case <-ctx.Done():
		err = cleanupInspection(child)
		if !errors.Is(err, errOwnership) {
			err = permissionError()
		}
	}
	child.Output().Close()
	if errors.Is(err, errOwnership) {
		return permissionProof{}, child, err
	}
	code, known := processExitCode(child)
	if ctx.Err() != nil || err != nil || !known || code != 0 || child.Forced() {
		return permissionProof{}, child, permissionError()
	}
	return proof, child, nil
}

// Cleanup has its own finite reserve. Losing stop/reap proof never becomes an
// ordinary no-child preparation failure, even if output had looked valid.
func cleanupInspection(child process) error {
	done := make(chan error, 1)
	go func() {
		err := child.Stop()
		child.Output().Close()
		if err != nil {
			done <- errOwnership
			return
		}
		done <- child.Wait()
	}()
	select {
	case err := <-done:
		return err
	case <-time.After(7 * time.Second):
		child.Output().Close()
		return errOwnership
	}
}

func superviseConfigInspection(request supervisorRequest, lifeline io.Reader, output io.Writer) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	go func() { _, _ = io.Copy(io.Discard, lifeline); cancel() }()
	proof, child, err := inspectWithProcess(ctx, launchSpec{executable: request.Executable, directory: request.Directory, args: request.Args, env: request.Env, inspection: request.Inspection}, startProcess)
	exit := supervisorProof{Type: "accord.supervisor.exit", Nonce: request.Nonce, Proven: !errors.Is(err, errOwnership), Success: err == nil, Diagnostic: ownershipDiagnostic(err)}
	if child != nil {
		exit.Forced = child.Forced()
		if code, ok := processExitCode(child); ok {
			exit.ExitCode = &code
		}
	}
	if err == nil {
		if err = json.NewEncoder(output).Encode(proof); err != nil {
			return err
		}
	}
	return json.NewEncoder(output).Encode(exit)
}

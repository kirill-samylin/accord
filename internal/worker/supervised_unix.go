//go:build darwin || linux

package worker

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type supervisedProcess struct {
	cmd       *exec.Cmd
	input     io.WriteCloser
	output    *proofReader
	nonce     string
	done      chan struct{}
	closeOnce sync.Once
	mu        sync.Mutex
	proof     *supervisorProof
	waitError error
}

func startSupervised(spec launchSpec) (process, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(self, "worker", "supervise")
	cmd.Env = providerEnv()
	cmd.Stderr = io.Discard
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	input, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	output, writer, err := os.Pipe()
	if err != nil {
		input.Close()
		return nil, err
	}
	cmd.Stdout = writer
	p := &supervisedProcess{cmd: cmd, input: input, nonce: randomID("proof_"), done: make(chan struct{})}
	p.output = &proofReader{source: output, reader: bufio.NewReader(output), owner: p}
	if err = cmd.Start(); err != nil {
		input.Close()
		output.Close()
		writer.Close()
		return nil, err
	}
	writer.Close()
	// From this point the helper may have launched a native process. Always
	// return its ownership handle; handshake failures must go through final
	// proof/needs_recovery instead of the unambiguous no-child error path.
	request := supervisorRequest{Executable: spec.executable, Directory: spec.directory, Args: spec.args, Env: spec.env, Prompt: spec.prompt, Nonce: p.nonce}
	go func() {
		if json.NewEncoder(input).Encode(request) != nil {
			p.closeInput()
		}
	}()
	go func() { err := cmd.Wait(); p.mu.Lock(); p.waitError = err; p.mu.Unlock(); close(p.done) }()
	return p, nil
}

func (p *supervisedProcess) closeInput()           { p.closeOnce.Do(func() { p.input.Close() }) }
func (p *supervisedProcess) Output() io.ReadCloser { return p.output }
func (p *supervisedProcess) Forced() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.proof != nil && p.proof.Forced
}
func (p *supervisedProcess) Wait() error {
	defer p.closeInput()
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.proof != nil && !p.proof.Proven {
		return proofOwnershipFailure(p.proof)
	}
	if p.waitError != nil {
		return ownershipFailureAt("supervisor_helper_exit_failed")
	}
	if p.proof == nil {
		return ownershipFailureAt("supervisor_missing_nonce_proof")
	}
	if !p.proof.Success {
		return errNativeExit
	}
	return nil
}
func (p *supervisedProcess) Stop() error {
	p.closeInput()
	deadline := time.Now().Add(7 * time.Second)
	select {
	case <-p.done:
		for {
			p.mu.Lock()
			proof := p.proof
			p.mu.Unlock()
			if proof != nil {
				if proof.Proven {
					return nil
				}
				return proofOwnershipFailure(proof)
			}
			if time.Now().After(deadline) {
				return ownershipFailureAt("supervisor_stop_proof_timeout")
			}
			time.Sleep(10 * time.Millisecond)
		}
	case <-time.After(7 * time.Second):
		return ownershipFailureAt("supervisor_stop_helper_timeout")
	}
}

func proofOwnershipFailure(proof *supervisorProof) error {
	if proof.Diagnostic != "" && validDiagnostic(proof.Diagnostic) {
		return ownershipFailureAt(proof.Diagnostic)
	}
	return ownershipFailureAt("supervisor_unproven_exit")
}

// ReadSlice keeps any single allocation bounded even for malformed native
// output. Provider event framing still belongs to the ordinary bounded scanner.
type proofReader struct {
	source io.ReadCloser
	reader *bufio.Reader
	owner  *supervisedProcess
	buffer []byte
}

func (r *proofReader) Read(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if len(r.buffer) > 0 {
		n := copy(data, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}
	fragment, err := r.reader.ReadSlice('\n')
	if len(fragment) > 0 {
		var proof supervisorProof
		if len(fragment) < 1024 && json.Unmarshal(fragment, &proof) == nil && proof.Type == "accord.supervisor.exit" && proof.Nonce == r.owner.nonce {
			r.owner.mu.Lock()
			r.owner.proof = &proof
			r.owner.mu.Unlock()
		}
		n := copy(data, fragment)
		r.buffer = append(r.buffer[:0], fragment[n:]...)
		return n, nil
	}
	return 0, err
}
func (r *proofReader) Close() error { return r.source.Close() }

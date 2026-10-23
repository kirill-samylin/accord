//go:build windows

package worker

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"
)

// The outer Job owns the helper and descendants; its separate stdin remains a
// parent-lifetime guard. Raw app-server stdout is consumed only in the helper.
type windowsInspection struct {
	child  process
	input  *io.PipeWriter
	output *proofReader
	nonce  string
	mu     sync.Mutex
	proof  *supervisorProof
	done   chan struct{}
	err    error
}

func startWindowsInspection(spec launchSpec) (process, error) {
	self, err := os.Executable()
	if err != nil {
		return nil, err
	}
	r, w := io.Pipe()
	child, err := startProcess(launchSpec{executable: self, args: []string{"worker", "supervise"}, env: providerEnv(), stdin: r})
	if err != nil {
		r.Close()
		w.Close()
		return nil, err
	}
	p := &windowsInspection{child: child, input: w, nonce: randomID("proof_"), done: make(chan struct{})}
	p.output = &proofReader{source: child.Output(), reader: bufio.NewReader(child.Output()), owner: p}
	go func() {
		if json.NewEncoder(w).Encode(supervisorRequest{Executable: spec.executable, Directory: spec.directory, Args: spec.args, Env: spec.env, Nonce: p.nonce, Inspection: spec.inspection}) != nil {
			w.Close()
		}
	}()
	go func() {
		err := child.Wait()
		p.mu.Lock()
		p.err = err
		p.mu.Unlock()
		r.Close()
		close(p.done)
	}()
	return p, nil
}
func (p *windowsInspection) acceptProof(proof supervisorProof) bool {
	if proof.Nonce != p.nonce {
		return false
	}
	p.mu.Lock()
	p.proof = &proof
	p.mu.Unlock()
	return true
}
func (p *windowsInspection) Output() io.ReadCloser { return p.output }
func (p *windowsInspection) Wait() error {
	defer p.input.Close()
	<-p.done
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil || p.proof == nil || !p.proof.Proven {
		return errOwnership
	}
	if !p.proof.Success {
		return errNativeExit
	}
	return nil
}
func (p *windowsInspection) Stop() error {
	p.input.Close()
	timer := time.NewTimer(7 * time.Second)
	defer timer.Stop()
	select {
	case <-p.done:
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.err != nil || p.proof == nil || !p.proof.Proven {
			return errOwnership
		}
		return nil
	case <-timer.C:
		return errOwnership
	}
}
func (p *windowsInspection) Forced() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.proof != nil && p.proof.Forced
}
func (p *windowsInspection) ExitCode() (int, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.proof == nil || !p.proof.Proven || p.proof.ExitCode == nil {
		return 0, false
	}
	return *p.proof.ExitCode, true
}

//go:build darwin || linux

package worker

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type unixProcess struct {
	cmd      *exec.Cmd
	output   *os.File
	done     chan struct{}
	mu       sync.Mutex
	result   error
	finished bool
	forced   bool
}

func startProcess(spec launchSpec) (process, error) {
	reader, writer, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(spec.executable, spec.args...)
	cmd.Dir, cmd.Env, cmd.Stdin = spec.directory, spec.env, strings.NewReader(spec.prompt)
	cmd.Stdout, cmd.Stderr = writer, io.Discard
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err = cmd.Start(); err != nil {
		reader.Close()
		writer.Close()
		return nil, err
	}
	writer.Close()
	p := &unixProcess{cmd: cmd, output: reader, done: make(chan struct{})}
	go p.reap()
	return p, nil
}

func (p *unixProcess) Output() io.ReadCloser { return p.output }

func (p *unixProcess) reap() {
	// WNOWAIT keeps the leader's PID reserved until the owned process group is
	// killed. A fast parent exit must not leave live members in that group or
	// make a later group signal target an unrelated process after PID reuse.
	var siginfo [128]byte
	var errno syscall.Errno
	for {
		_, _, errno = syscall.Syscall6(syscall.SYS_WAITID, 1, uintptr(p.cmd.Process.Pid), uintptr(unsafe.Pointer(&siginfo[0])), syscall.WEXITED|syscall.WNOWAIT, 0, 0)
		if errno != syscall.EINTR {
			break
		}
	}
	p.mu.Lock()
	var cleanupError error
	if errno == 0 {
		if err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
			cleanupError = errOwnership
		} else {
			cleanupError = waitGroupExit(p.cmd.Process.Pid)
		}
	}
	// If the ownership wait itself fails, do not signal an unverified PID.
	p.result = p.cmd.Wait()
	if errno != 0 || cleanupError != nil {
		p.result = errOwnership
	}
	p.finished = true
	close(p.done)
	p.mu.Unlock()
}

func (p *unixProcess) Wait() error { <-p.done; p.mu.Lock(); defer p.mu.Unlock(); return p.result }

func (p *unixProcess) Stop() error {
	p.mu.Lock()
	if p.finished {
		p.mu.Unlock()
		return nil
	}
	err := syscall.Kill(-p.cmd.Process.Pid, syscall.SIGINT)
	p.mu.Unlock()
	if err != nil && err != syscall.ESRCH {
		return errors.New("cannot interrupt owned process group")
	}
	select {
	case <-p.done:
		return nil
	case <-time.After(3 * time.Second):
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.finished {
		p.forced = true
		if err = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL); err != nil && err != syscall.ESRCH {
			return errors.New("cannot stop owned process group")
		}
	}
	return nil
}

func (p *unixProcess) Forced() bool { p.mu.Lock(); defer p.mu.Unlock(); return p.forced }

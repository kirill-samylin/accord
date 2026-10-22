//go:build windows

package worker

import (
	"errors"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"unicode/utf16"
	"unsafe"

	"golang.org/x/sys/windows"
)

type windowsProcess struct {
	output   *os.File
	process  windows.Handle
	job      windows.Handle
	done     chan struct{}
	mu       sync.Mutex
	result   error
	finished bool
	forced   bool
}

func startProcess(spec launchSpec) (process, error) {
	job, err := windows.CreateJobObject(nil, nil)
	if err != nil {
		return nil, err
	}
	info := windows.JOBOBJECT_EXTENDED_LIMIT_INFORMATION{}
	info.BasicLimitInformation.LimitFlags = windows.JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE
	if _, err = windows.SetInformationJobObject(job, windows.JobObjectExtendedLimitInformation, uintptr(unsafe.Pointer(&info)), uint32(unsafe.Sizeof(info))); err != nil {
		windows.CloseHandle(job)
		return nil, err
	}
	p := &windowsProcess{job: job, done: make(chan struct{})}
	if err = p.launch(spec); err != nil {
		windows.CloseHandle(job)
		return nil, err
	}
	go p.reap()
	return p, nil
}

func (p *windowsProcess) launch(spec launchSpec) error {
	inputR, inputW, err := os.Pipe()
	if err != nil {
		return err
	}
	defer inputR.Close()
	outputR, outputW, err := os.Pipe()
	if err != nil {
		inputW.Close()
		return err
	}
	defer outputW.Close()
	attrs, err := windows.NewProcThreadAttributeList(1)
	if err != nil {
		inputW.Close()
		outputR.Close()
		return err
	}
	defer attrs.Delete()
	handles := []windows.Handle{windows.Handle(inputR.Fd()), windows.Handle(outputW.Fd())}
	for _, handle := range handles {
		if err = windows.SetHandleInformation(handle, windows.HANDLE_FLAG_INHERIT, windows.HANDLE_FLAG_INHERIT); err != nil {
			inputW.Close()
			outputR.Close()
			return err
		}
	}
	if err = attrs.Update(windows.PROC_THREAD_ATTRIBUTE_HANDLE_LIST, unsafe.Pointer(&handles[0]), uintptr(len(handles))*unsafe.Sizeof(handles[0])); err != nil {
		inputW.Close()
		outputR.Close()
		return err
	}
	si := windows.StartupInfoEx{StartupInfo: windows.StartupInfo{Cb: uint32(unsafe.Sizeof(windows.StartupInfoEx{})), Flags: windows.STARTF_USESTDHANDLES, StdInput: handles[0], StdOutput: handles[1], StdErr: handles[1]}, ProcThreadAttributeList: attrs.List()}
	executable, _ := windows.UTF16PtrFromString(spec.executable)
	command, err := windows.UTF16PtrFromString(windows.ComposeCommandLine(append([]string{spec.executable}, spec.args...)))
	if err != nil {
		inputW.Close()
		outputR.Close()
		return err
	}
	directory, err := windows.UTF16PtrFromString(spec.directory)
	if err != nil {
		inputW.Close()
		outputR.Close()
		return err
	}
	env := append([]string(nil), spec.env...)
	sort.Slice(env, func(i, j int) bool { return strings.ToUpper(env[i]) < strings.ToUpper(env[j]) })
	block := utf16.Encode([]rune(strings.Join(env, "\x00") + "\x00\x00"))
	var pi windows.ProcessInformation
	flags := uint32(windows.CREATE_SUSPENDED | windows.CREATE_UNICODE_ENVIRONMENT | windows.EXTENDED_STARTUPINFO_PRESENT | windows.CREATE_NO_WINDOW)
	err = windows.CreateProcess(executable, command, nil, nil, true, flags, &block[0], directory, &si.StartupInfo, &pi)
	if err != nil {
		inputW.Close()
		outputR.Close()
		return err
	}
	defer windows.CloseHandle(pi.Thread)
	// The process cannot execute a single instruction or spawn a descendant
	// before Job ownership is established. Failure always terminates it.
	if err = windows.AssignProcessToJobObject(p.job, pi.Process); err == nil {
		_, err = windows.ResumeThread(pi.Thread)
	}
	if err != nil {
		windows.TerminateProcess(pi.Process, 1)
		windows.CloseHandle(pi.Process)
		inputW.Close()
		outputR.Close()
		return err
	}
	p.process, p.output = pi.Process, outputR
	go func() { _, _ = io.WriteString(inputW, spec.prompt); inputW.Close() }()
	return nil
}

func (p *windowsProcess) Output() io.ReadCloser { return p.output }
func (p *windowsProcess) Wait() error           { <-p.done; p.mu.Lock(); defer p.mu.Unlock(); return p.result }
func (p *windowsProcess) reap() {
	_, err := windows.WaitForSingleObject(p.process, windows.INFINITE)
	p.mu.Lock()
	defer p.mu.Unlock()
	var code uint32
	if err == nil {
		err = windows.GetExitCodeProcess(p.process, &code)
	}
	closeProcessError := windows.CloseHandle(p.process)
	forced, treeError := finishWindowsJob(p.job)
	p.forced = p.forced || forced
	closeJobError := windows.CloseHandle(p.job)
	p.finished = true
	if err != nil || code != 0 {
		p.result = errors.New("native provider exited unsuccessfully")
	}
	if closeProcessError != nil || closeJobError != nil || treeError != nil {
		p.result = errOwnership
	}
	close(p.done)
}
func (p *windowsProcess) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.finished {
		return nil
	}
	if err := windows.TerminateJobObject(p.job, 1); err != nil {
		return errors.New("cannot stop owned process job")
	}
	p.forced = true
	return nil
}

func (p *windowsProcess) Forced() bool { p.mu.Lock(); defer p.mu.Unlock(); return p.forced }

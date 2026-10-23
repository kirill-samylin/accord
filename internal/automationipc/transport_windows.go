//go:build windows

package automationipc

import (
	"context"
	"encoding/hex"
	"golang.org/x/sys/windows"
	"io"
	"net"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type pipeAddr string

func (a pipeAddr) Network() string { return "pipe" }
func (a pipeAddr) String() string  { return string(a) }

type pipeListener struct {
	mu      sync.Mutex
	path    string
	sd      *windows.SECURITY_DESCRIPTOR
	waiting windows.Handle
	closed  bool
}

func listen(_ string) (net.Listener, error) {
	name, e := secret()
	if e != nil {
		return nil, e
	}
	user, e := windows.GetCurrentProcessToken().GetTokenUser()
	if e != nil {
		return nil, errLocalTransport
	}
	sd, e := windows.SecurityDescriptorFromString("D:P(A;;GA;;;SY)(A;;GA;;;" + user.User.Sid.String() + ")")
	if e != nil {
		return nil, errLocalTransport
	}
	l := &pipeListener{path: `\\.\pipe\accord-check-` + name[:32], sd: sd}
	h, e := l.newHandle(true)
	if e != nil {
		return nil, errLocalTransport
	}
	l.waiting = h
	return l, nil
}
func (l *pipeListener) newHandle(first bool) (windows.Handle, error) {
	ptr, e := windows.UTF16PtrFromString(l.path)
	if e != nil {
		return 0, e
	}
	flags := uint32(windows.PIPE_ACCESS_DUPLEX | windows.FILE_FLAG_OVERLAPPED)
	if first {
		flags |= windows.FILE_FLAG_FIRST_PIPE_INSTANCE
	}
	sa := windows.SecurityAttributes{Length: uint32(unsafe.Sizeof(windows.SecurityAttributes{})), SecurityDescriptor: l.sd}
	return windows.CreateNamedPipe(ptr, flags, windows.PIPE_TYPE_BYTE|windows.PIPE_READMODE_BYTE|windows.PIPE_WAIT|windows.PIPE_REJECT_REMOTE_CLIENTS, 16, MaxFrame+4, MaxFrame+4, 0, &sa)
}
func (l *pipeListener) Addr() net.Addr { return pipeAddr(l.path) }
func (l *pipeListener) Accept() (net.Conn, error) {
	l.mu.Lock()
	if l.closed {
		l.mu.Unlock()
		return nil, net.ErrClosed
	}
	h := l.waiting
	l.mu.Unlock()
	event, e := windows.CreateEvent(nil, 1, 0, nil)
	if e != nil {
		return nil, e
	}
	defer windows.CloseHandle(event)
	over := windows.Overlapped{HEvent: event}
	e = windows.ConnectNamedPipe(h, &over)
	if e == windows.ERROR_IO_PENDING {
		_, e = windows.WaitForSingleObject(event, windows.INFINITE)
		if e == nil {
			var n uint32
			e = windows.GetOverlappedResult(h, &over, &n, false)
		}
	} else if e == windows.ERROR_PIPE_CONNECTED {
		e = nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil, net.ErrClosed
	}
	if e != nil {
		return nil, e
	}
	next, e := l.newHandle(false)
	if e != nil {
		windows.CloseHandle(h)
		l.closed = true
		return nil, e
	}
	l.waiting = next
	return &pipeConn{handle: h}, nil
}
func (l *pipeListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	_ = windows.CancelIoEx(l.waiting, nil)
	return windows.CloseHandle(l.waiting)
}
func dial(ctx context.Context, path string) (net.Conn, error) {
	suffix, ok := strings.CutPrefix(path, `\\.\pipe\accord-check-`)
	decoded, decodeErr := hex.DecodeString(suffix)
	if !ok || decodeErr != nil || len(decoded) != 16 {
		return nil, errLocalTransport
	}
	ptr, e := windows.UTF16PtrFromString(path)
	if e != nil {
		return nil, e
	}
	for {
		h, e := windows.CreateFile(ptr, windows.GENERIC_READ|windows.GENERIC_WRITE, 0, nil, windows.OPEN_EXISTING, windows.FILE_FLAG_OVERLAPPED, 0)
		if e == nil {
			return &pipeConn{handle: h}, nil
		}
		if e != windows.ERROR_PIPE_BUSY && e != windows.ERROR_FILE_NOT_FOUND {
			return nil, e
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

type pipeConn struct {
	handle          windows.Handle
	mu              sync.Mutex
	readMu, writeMu sync.Mutex
	deadline        time.Time
	closed          bool
}

func (c *pipeConn) Read(p []byte) (int, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()
	return c.transfer(p, false)
}
func (c *pipeConn) Write(p []byte) (int, error) {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	return c.transfer(p, true)
}
func (c *pipeConn) transfer(p []byte, write bool) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	c.mu.Lock()
	h, d, closed := c.handle, c.deadline, c.closed
	c.mu.Unlock()
	if closed {
		return 0, net.ErrClosed
	}
	event, e := windows.CreateEvent(nil, 1, 0, nil)
	if e != nil {
		return 0, e
	}
	defer windows.CloseHandle(event)
	over := windows.Overlapped{HEvent: event}
	var n uint32
	if write {
		e = windows.WriteFile(h, p, &n, &over)
	} else {
		e = windows.ReadFile(h, p, &n, &over)
	}
	if e == windows.ERROR_IO_PENDING {
		timeout := uint32(windows.INFINITE)
		if !d.IsZero() {
			remaining := time.Until(d)
			if remaining <= 0 {
				remaining = time.Millisecond
			}
			timeout = uint32(max(1, remaining.Milliseconds()))
		}
		state, waitErr := windows.WaitForSingleObject(event, timeout)
		if waitErr != nil || state == uint32(windows.WAIT_TIMEOUT) {
			_ = windows.CancelIoEx(h, &over)
			_ = windows.GetOverlappedResult(h, &over, &n, true)
			return 0, context.DeadlineExceeded
		}
		e = windows.GetOverlappedResult(h, &over, &n, false)
	}
	if e == windows.ERROR_BROKEN_PIPE {
		return int(n), io.EOF
	}
	return int(n), e
}
func (c *pipeConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	_ = windows.CancelIoEx(c.handle, nil)
	return windows.CloseHandle(c.handle)
}
func (c *pipeConn) LocalAddr() net.Addr  { return pipeAddr("local") }
func (c *pipeConn) RemoteAddr() net.Addr { return pipeAddr("local") }
func (c *pipeConn) SetDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deadline = t
	return nil
}
func (c *pipeConn) SetReadDeadline(t time.Time) error  { return c.SetDeadline(t) }
func (c *pipeConn) SetWriteDeadline(t time.Time) error { return c.SetDeadline(t) }

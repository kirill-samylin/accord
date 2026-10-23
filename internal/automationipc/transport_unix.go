//go:build darwin || linux

package automationipc

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
)

type privateListener struct {
	net.Listener
	directory string
}

func (l *privateListener) Close() error {
	e := l.Listener.Close()
	if l.directory != "" {
		e = errors.Join(e, os.Remove(l.directory))
	}
	return e
}
func privateIPCFolder(path string) error {
	info, e := os.Lstat(path)
	if e != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return errLocalTransport
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return errLocalTransport
	}
	return nil
}
func listen(directory string) (net.Listener, error) {
	if privateIPCFolder(directory) != nil {
		return nil, errLocalTransport
	}
	name, e := secret()
	if e != nil {
		return nil, e
	}
	base := "c-" + name[:24] + ".sock"
	path := filepath.Join(directory, base)
	owned := ""
	limit := 107
	if runtime.GOOS == "darwin" {
		limit = 103
	}
	if len(path) > limit {
		home, e := os.UserHomeDir()
		if e != nil {
			return nil, errLocalTransport
		}
		root := filepath.Join(home, ".accord-ipc")
		if e = os.Mkdir(root, 0700); e != nil && !os.IsExist(e) {
			return nil, errLocalTransport
		}
		if privateIPCFolder(root) != nil {
			return nil, errLocalTransport
		}
		owned = filepath.Join(root, name[24:40])
		if len(filepath.Join(owned, base)) > limit {
			return nil, errors.New("private IPC path is too long; a shorter owner HOME is required")
		}
		if e = os.Mkdir(owned, 0700); e != nil {
			return nil, errLocalTransport
		}
		path = filepath.Join(owned, base)
	}
	l, e := net.Listen("unix", path)
	if e != nil {
		if owned != "" {
			_ = os.Remove(owned)
		}
		return nil, errLocalTransport
	}
	if e = os.Chmod(path, 0600); e != nil {
		l.Close()
		if owned != "" {
			_ = os.Remove(owned)
		}
		return nil, errLocalTransport
	}
	return &privateListener{l, owned}, nil
}
func dial(ctx context.Context, path string) (net.Conn, error) {
	info, e := os.Lstat(path)
	if e != nil || info.Mode()&os.ModeSocket == 0 || info.Mode().Perm()&0077 != 0 {
		return nil, errLocalTransport
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok || stat.Uid != uint32(os.Geteuid()) {
		return nil, errLocalTransport
	}
	return (&net.Dialer{}).DialContext(ctx, "unix", path)
}

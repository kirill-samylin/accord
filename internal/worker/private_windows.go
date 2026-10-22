//go:build windows

package worker

import (
	"errors"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Windows mode bits do not enforce confidentiality. Check the actual owner and
// DACL instead, allowing only this user, SYSTEM and local administrators.
func checkPrivateACL(path string) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return err
	}
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.OWNER_SECURITY_INFORMATION|windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		return err
	}
	owner, _, err := sd.Owner()
	if err != nil || !owner.Equals(user.User.Sid) {
		return errors.New("worker path must be owned by current user")
	}
	acl, _, err := sd.DACL()
	if err != nil || acl == nil || acl.AceCount == 0 {
		return errors.New("worker path requires a private DACL")
	}
	for i := uint32(0); i < uint32(acl.AceCount); i++ {
		var ace *windows.ACCESS_ALLOWED_ACE
		if err = windows.GetAce(acl, i, &ace); err != nil {
			return err
		}
		if ace.Header.AceType == windows.ACCESS_DENIED_ACE_TYPE {
			continue
		}
		if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE {
			return errors.New("unsupported worker path ACL")
		}
		sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
		if !sid.Equals(user.User.Sid) && !sid.IsWellKnown(windows.WinLocalSystemSid) && !sid.IsWellKnown(windows.WinBuiltinAdministratorsSid) {
			return errors.New("worker path is accessible to another account")
		}
	}
	return nil
}

func privateRead(path string) (*os.File, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return nil, err
	}
	if err != nil || !info.Mode().IsRegular() {
		return nil, errors.New("worker file must be regular and not a symlink")
	}
	if err = checkPrivateACL(path); err != nil {
		return nil, err
	}
	return os.Open(path)
}

func privateDirectory(path string) error {
	_, err := os.Lstat(path)
	if os.IsNotExist(err) {
		if err = os.MkdirAll(path, 0700); err != nil {
			return err
		}
		user, err := windows.GetCurrentProcessToken().GetTokenUser()
		if err != nil {
			return err
		}
		sd, err := windows.SecurityDescriptorFromString("D:P(A;OICI;FA;;;" + user.User.Sid.String() + ")")
		if err != nil {
			return err
		}
		acl, _, err := sd.DACL()
		if err != nil {
			return err
		}
		if err = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, acl, nil); err != nil {
			return err
		}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("worker state must be a directory and not a symlink")
	}
	return checkPrivateACL(path)
}

func lockFile(file *os.File) error {
	return windows.LockFileEx(windows.Handle(file.Fd()), windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY, 0, 1, 0, &windows.Overlapped{})
}
func unlockFile(file *os.File) {
	_ = windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &windows.Overlapped{})
}

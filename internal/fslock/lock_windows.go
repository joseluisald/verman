//go:build windows

package fslock

import (
	"os"
	"syscall"
	"unsafe"
)

var lockFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("LockFileEx")

func tryLock(f *os.File, shared bool) (bool, error) {
	flags := uintptr(1)
	if !shared {
		flags |= 2
	}
	var overlapped syscall.Overlapped
	ok, _, err := lockFileEx.Call(f.Fd(), flags, 0, 1, 0, uintptr(unsafe.Pointer(&overlapped)))
	if ok != 0 {
		return true, nil
	}
	if err == syscall.Errno(33) || err == syscall.Errno(997) {
		return false, nil
	}
	return false, err
}

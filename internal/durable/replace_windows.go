//go:build windows

package durable

import (
	"syscall"
	"unsafe"
)

var moveFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

func replace(from, to string) error {
	source, err := syscall.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	dest, err := syscall.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	ok, _, err := moveFileEx.Call(uintptr(unsafe.Pointer(source)), uintptr(unsafe.Pointer(dest)), 9)
	if ok == 0 {
		return err
	}
	return nil
}

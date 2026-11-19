//go:build windows

package durable

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestLockedDestinationPreservesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte("OLD"), 0600); err != nil {
		t.Fatal(err)
	}
	wide, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	handle, err := syscall.CreateFile(wide, syscall.GENERIC_READ, syscall.FILE_SHARE_READ, nil, syscall.OPEN_EXISTING, syscall.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	if err := Write(path, []byte("NEW"), 0600); err == nil {
		syscall.CloseHandle(handle)
		t.Fatal("arquivo bloqueado substituído")
	}
	syscall.CloseHandle(handle)
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "OLD" {
		t.Fatal("original perdido")
	}
}

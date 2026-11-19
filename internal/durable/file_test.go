package durable

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestPartialWriteDiskFullPreservesPreviousFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.toml")
	if err := os.WriteFile(target, []byte("OLD"), 0600); err != nil {
		t.Fatal(err)
	}
	err := writeContext(context.Background(), target, []byte("NEW"), 0600, func(f *os.File, data []byte) (int, error) {
		n, err := f.Write(data[:1])
		if err != nil {
			return n, err
		}
		return n, syscall.ENOSPC
	})
	if !errors.Is(err, syscall.ENOSPC) {
		t.Fatalf("erro de disco perdido: %v", err)
	}
	if data, err := os.ReadFile(target); err != nil || string(data) != "OLD" {
		t.Fatalf("arquivo anterior alterado: %q %v", data, err)
	}
	items, err := os.ReadDir(dir)
	if err != nil || len(items) != 1 {
		t.Fatalf("temporário não limpo: %v %v", items, err)
	}
}

func TestFailedReplacementPreservesExistingDirectory(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "config.toml")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(target, "preserve")
	if err := os.WriteFile(marker, []byte("OLD"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Write(target, []byte("NEW"), 0600); err == nil {
		t.Fatal("substituição impossível retornou sucesso")
	}
	if data, err := os.ReadFile(marker); err != nil || string(data) != "OLD" {
		t.Fatal("dados existentes alterados")
	}
	items, err := os.ReadDir(dir)
	if err != nil || len(items) != 1 {
		t.Fatalf("temporário não limpo: %v %v", items, err)
	}
}

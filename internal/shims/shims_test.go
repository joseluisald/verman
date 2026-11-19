package shims

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestSetupOwnershipAndRepeatedInstallation(t *testing.T) {
	dir := t.TempDir()
	source := filepath.Join(t.TempDir(), "source.exe")
	if err := os.WriteFile(source, []byte("V1"), 0700); err != nil {
		t.Fatal(err)
	}
	foreign := filepath.Join(dir, "php.exe")
	if err := os.WriteFile(foreign, []byte("FOREIGN"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := Setup(context.Background(), source, dir); err == nil {
		t.Fatal("arquivo externo substituído")
	}
	if data, err := os.ReadFile(foreign); err != nil || string(data) != "FOREIGN" {
		t.Fatal("arquivo externo alterado")
	}
	if _, err := os.Stat(filepath.Join(dir, "node.exe")); !os.IsNotExist(err) {
		t.Fatal("setup alterou diretório antes do preflight")
	}
	if err := os.Remove(foreign); err != nil {
		t.Fatal(err)
	}
	for _, version := range []string{"V1", "V1", "V2"} {
		if err := os.WriteFile(source, []byte(version), 0700); err != nil {
			t.Fatal(err)
		}
		if err := Setup(context.Background(), source, dir); err != nil {
			t.Fatal(err)
		}
		if !IsDirectory(dir) {
			t.Fatal("manifesto ausente")
		}
		for _, name := range Names {
			data, err := os.ReadFile(filepath.Join(dir, name+".exe"))
			if err != nil || string(data) != version {
				t.Fatalf("shim %s: %q %v", name, data, err)
			}
		}
	}
}

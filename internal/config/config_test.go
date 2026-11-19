package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRejectUnsafeInputs(t *testing.T) {
	for _, pair := range [][2]string{{"../node", "22.1.0"}, {"ruby", "3.1.0"}, {"node", "../../escape"}, {"php", `C:\tmp`}, {"python", "latest"}, {"node", "22.1"}, {"node", "22.1.0\n"}} {
		if _, err := InstallDir(pair[0], pair[1]); err == nil {
			t.Errorf("entrada aceita: %q", pair)
		}
		if err := SaveProjectTool(t.TempDir(), pair[0], pair[1]); err == nil {
			t.Errorf("configuração aceitou %q", pair)
		}
	}
}

func TestInvalidConfigPreserved(t *testing.T) {
	for _, data := range []string{"[tools\nnode =", "[tools]\nnode = 123", "[tools]\nnode = '../../escape'"} {
		dir := t.TempDir()
		path := filepath.Join(dir, ProjectFileName)
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
		if err := SaveProjectTool(dir, "php", "8.4.1"); err == nil {
			t.Fatal("esperava erro")
		}
		got, err := os.ReadFile(path)
		if err != nil || string(got) != data {
			t.Fatalf("arquivo alterado: %q, %v", got, err)
		}
		if _, err := LoadProjectConfig(dir); err == nil {
			t.Fatal("leitura deveria falhar")
		}
	}
}

func TestSaveAndInheritance(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(parent, ProjectFileName)
	if err := os.WriteFile(path, []byte("title = 'preservar'\n[tools]\nnode = '22.11.0'\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := SaveProjectTool(parent, "php", "8.4.1"); err != nil {
		t.Fatal(err)
	}
	// Exercita também a substituição de um arquivo existente no Windows.
	if err := SaveProjectTool(parent, "php", "8.4.2"); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadProjectConfig(child)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Tools["node"] != "22.11.0" || cfg.Tools["php"] != "8.4.2" {
		t.Fatalf("configuração: %#v", cfg)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "preservar") {
		t.Fatalf("chave perdida: %s, %v", data, err)
	}
	if err := SaveProjectTool(child, "python", "3.13.0"); err != nil {
		t.Fatal(err)
	}
	cfg, err = LoadProjectConfig(child)
	if err != nil || len(cfg.Tools) != 1 || cfg.Tools["python"] != "3.13.0" {
		t.Fatalf("configurações foram mescladas: %#v, %v", cfg, err)
	}
	entries, err := filepath.Glob(filepath.Join(parent, ".verman-*.tmp"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporários residuais: %v, %v", entries, err)
	}
}

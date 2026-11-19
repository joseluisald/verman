package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGlobalPrecedenceAndRemoval(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	project := t.TempDir()
	if err := SaveGlobalTool("node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	if err := SaveGlobalTool("php", "8.4.27"); err != nil {
		t.Fatal(err)
	}
	if err := SaveProjectTool(project, "node", "22.12.0"); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadEffectiveConfig(project)
	if err != nil || cfg.Tools["node"] != "22.12.0" || cfg.Tools["php"] != "8.4.27" {
		t.Fatalf("precedência: %#v %v", cfg, err)
	}
	global, err := LoadGlobalConfig()
	if err != nil || global.Tools["node"] != "22.11.0" {
		t.Fatal("projeto alterou global")
	}
	if err := RemoveGlobalTool("node"); err != nil {
		t.Fatal(err)
	}
	global, err = LoadGlobalConfig()
	if err != nil || len(global.Tools) != 1 {
		t.Fatalf("remoção global: %#v %v", global, err)
	}
	base, err := BaseDir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(base, GlobalFileName)
	bad := "[tools\nnode ="
	if err := os.WriteFile(path, []byte(bad), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadEffectiveConfig(project); err == nil {
		t.Fatal("global inválido ignorado")
	}
	if err := SaveGlobalTool("node", "22.11.0"); err == nil {
		t.Fatal("global inválido sobrescrito")
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != bad {
		t.Fatal("arquivo global alterado")
	}
}

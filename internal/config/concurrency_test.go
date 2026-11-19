package config

import (
	"os"
	"os/exec"
	"testing"
)

func TestConcurrentProcessesPreserveTools(t *testing.T) {
	for _, scope := range []string{"project", "global"} {
		t.Run(scope, func(t *testing.T) { testConcurrentScope(t, scope) })
	}
}

func testConcurrentScope(t *testing.T, scope string) {
	dir := t.TempDir()
	t.Setenv("USERPROFILE", dir)
	t.Setenv("HOME", dir)
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	commands := []*exec.Cmd{}
	for _, pair := range [][2]string{{"node", "22.11.0"}, {"php", "8.5.12"}, {"python", "3.14.10"}} {
		child := exec.Command(binary, "-test.run=^TestConfigWorker$")
		child.Env = append(os.Environ(), "VERMAN_CONFIG_SCOPE="+scope, "VERMAN_CONFIG_DIR="+dir, "VERMAN_CONFIG_LANG="+pair[0], "VERMAN_CONFIG_VERSION="+pair[1])
		child.Stdout = os.Stdout
		child.Stderr = os.Stderr
		if err := child.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, child)
	}
	for _, child := range commands {
		if err := child.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	cfg, err := LoadProjectConfig(dir)
	if scope == "global" {
		cfg, err = LoadGlobalConfig()
	}
	if err != nil || len(cfg.Tools) != 3 {
		t.Fatalf("atualizações perdidas: %#v %v", cfg, err)
	}
}
func TestConfigWorker(t *testing.T) {
	if dir := os.Getenv("VERMAN_CONFIG_DIR"); dir != "" {
		var err error
		if os.Getenv("VERMAN_CONFIG_SCOPE") == "global" {
			err = SaveGlobalTool(os.Getenv("VERMAN_CONFIG_LANG"), os.Getenv("VERMAN_CONFIG_VERSION"))
		} else {
			err = SaveProjectTool(dir, os.Getenv("VERMAN_CONFIG_LANG"), os.Getenv("VERMAN_CONFIG_VERSION"))
		}
		if err != nil {
			t.Fatal(err)
		}
	}
}

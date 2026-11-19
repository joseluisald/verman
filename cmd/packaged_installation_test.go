package cmd

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/joseluisald/verman/internal/config"
)

func TestPackagedToolsAndProfileInstallations(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("pacote Windows amd64")
	}
	binary, project, home := step9Setup(t)
	app := filepath.Dir(binary)
	if err := os.WriteFile(filepath.Join(app, ".verman-app"), []byte("Verman"), 0600); err != nil {
		t.Fatal(err)
	}

	legacy := filepath.Join(home, ".verman", "installs", "node")
	packaged := filepath.Join(app, "tools", "node")
	if err := os.MkdirAll(packaged, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(legacy, "22.12.0"), filepath.Join(packaged, "22.12.0")); err != nil {
		t.Fatal(err)
	}
	run := func(executable string, args ...string) (string, error) {
		command := exec.Command(executable, args...)
		command.Dir = project
		output, err := command.CombinedOutput()
		return string(output), err
	}
	check := func(executable string, expected []string, args ...string) {
		t.Helper()
		output, err := run(executable, args...)
		if err != nil {
			t.Fatalf("%v: %s (%v)", args, output, err)
		}
		for _, value := range expected {
			if !strings.Contains(output, value) {
				t.Fatalf("%v: %q ausente em %s", args, value, output)
			}
		}
	}

	check(binary, []string{"22.11.0", "22.12.0"}, "list", "node")
	checkEnv := func(path string) {
		t.Helper()
		encoded, err := json.Marshal(path)
		if err != nil {
			t.Fatal(err)
		}
		check(binary, []string{strings.ReplaceAll(string(encoded), "'", "''")}, "env")
	}
	checkEnv(filepath.Join(packaged, "22.12.0"))
	if err := config.SaveProjectTool(project, "node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	checkEnv(filepath.Join(legacy, "22.11.0"))

	for _, bin := range []string{filepath.Join(app, "bin"), t.TempDir()} {
		check(binary, nil, "setup", "--bin-dir", bin)
		check(filepath.Join(bin, "verman.exe"), []string{"22.11.0", "22.12.0"}, "list", "node")
	}

	journal := config.InstallJournalName("22.10.0")
	if err := os.WriteFile(filepath.Join(legacy, journal), []byte("inválido"), 0600); err != nil {
		t.Fatal(err)
	}
	if output, err := run(binary, "recover", "node", "22.10.0"); err == nil || !strings.Contains(output, journal) {
		t.Fatalf("recover não encontrou o journal no perfil: %s (%v)", output, err)
	}

	if output, err := run(binary, "--offline", "install", "bun", "1.2.3"); err == nil {
		t.Fatalf("instalação sem cache em modo offline: %s", output)
	}
	if _, err := os.Stat(filepath.Join(app, "tools", "bun", ".lock-1.2.3.lck")); err != nil {
		t.Fatalf("instalação não usou a pasta tools: %v", err)
	}

	check(binary, nil, "uninstall", "node", "22.12.0")
	if _, err := os.Stat(filepath.Join(packaged, "22.12.0")); !os.IsNotExist(err) {
		t.Fatalf("versão removida ainda presente: %v", err)
	}
	check(binary, []string{"22.11.0"}, "list", "node")
}

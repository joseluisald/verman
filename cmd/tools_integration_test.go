package cmd

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/joseluisald/verman/internal/installer"
)

// Somente distribuições oficiais, em um perfil descartável; não instala pacotes
// Chocolatey/vcpkg, não eleva e não altera PATH/perfis/registro do computador.
func TestOfficialAdditionalToolsWorkflow(t *testing.T) {
	if os.Getenv("VERMAN_INTEGRATION") != "1" {
		t.Skip("defina VERMAN_INTEGRATION=1")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows/amd64")
	}
	root := t.TempDir()
	home := filepath.Join(root, "perfil espaço")
	project := filepath.Join(root, "projeto")
	for _, dir := range []string{home, project, filepath.Join(home, "AppData", "Roaming"), filepath.Join(home, "AppData", "Local")} {
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for key, value := range map[string]string{"HOME": home, "USERPROFILE": home, "APPDATA": filepath.Join(home, "AppData", "Roaming"), "LOCALAPPDATA": filepath.Join(home, "AppData", "Local"), "VERMAN_PATH_ENTRIES": "", "VERMAN_ENV_BACKUP": "", "PSModuleAnalysisCachePath": filepath.Join(home, "ps-cache"), "VCPKG_DISABLE_METRICS": "1"} {
		t.Setenv(key, value)
	}
	binary := filepath.Join(root, "verman.exe")
	if out, err := exec.Command("go", "build", "-o", binary, "..").CombinedOutput(); err != nil {
		t.Fatalf("build: %s %v", out, err)
	}
	run := func(tb *testing.T, executable string, args ...string) string {
		tb.Helper()
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, executable, args...)
		cmd.Dir = project
		out, err := cmd.CombinedOutput()
		if err != nil {
			tb.Fatalf("%v: %v\n%s", args, err, out)
		}
		return string(out)
	}
	// Dependências são selecionadas apenas no projeto temporário.
	run(t, binary, "use", "node", "22.12.0")
	run(t, binary, "use", "php", "latest")
	shims := filepath.Join(root, "shims")
	run(t, binary, "setup", "--bin-dir", shims)
	for _, language := range []string{"bun", "go", "deno", "git", "rust", "composer", "yarn", "pnpm", "vcpkg", "chocolatey"} {
		t.Run(language, func(t *testing.T) {
			t.Logf("catálogo e instalação real: %s", language)
			catalog, err := installer.Available(context.Background(), language, true)
			if err != nil {
				t.Fatal(err)
			}
			if len(catalog.Catalog.Artifacts) == 0 {
				t.Fatal("catálogo vazio")
			}
			a := catalog.Catalog.Artifacts[0]
			t.Logf("versão %s build %s", a.Version, a.Build)
			run(t, binary, "install", language, a.Version, "--progress", "never")
			run(t, binary, "use", language, a.Version)
			run(t, binary, "verify", language, a.Version)
			command := language
			flag := "--version"
			expected := a.Version
			switch language {
			case "rust":
				command = "rustc"
			case "chocolatey":
				command = "choco"
			case "go":
				flag = "version"
			case "vcpkg":
				flag = "version"
				expected = a.Build
			}
			out := run(t, binary, "exec", "--", command, flag)
			if !strings.Contains(out, expected) {
				t.Fatalf("versão: %s", out)
			}
			out = run(t, filepath.Join(shims, command+".exe"), flag)
			if !strings.Contains(out, expected) {
				t.Fatalf("shim: %s", out)
			}
			switch language {
			case "rust":
				run(t, filepath.Join(shims, "cargo.exe"), "--version")
			case "go":
				run(t, filepath.Join(shims, "gofmt.exe"), "-h")
			case "yarn":
				run(t, filepath.Join(shims, "yarnpkg.exe"), "--version")
			}
		})
	}
}

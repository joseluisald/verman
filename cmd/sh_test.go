package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestShManualActivation(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("alvo validado: Git sh no Windows")
	}
	shell := `C:\Program Files\Git\usr\bin\sh.exe`
	if _, err := os.Stat(shell); err != nil {
		t.Skip("Git sh indisponível")
	}
	binary, project, home := step9Setup(t)
	quote := func(v string) string { return "'" + strings.ReplaceAll(filepath.ToSlash(v), "'", "'\"'\"'") + "'" }
	script := `set -eu
PATH=/usr/bin:/bin:$PATH
export PATH
exe=$(cygpath -u -- ` + quote(binary) + `)
cd "$(cygpath -u -- ` + quote(project) + `)"
code=$("$exe" env --shell sh)
eval "$code"
expected=$(cygpath -u -- ` + quote(filepath.Join(home, ".verman", "installs", "node", "22.12.0")) + `)
case "$PATH" in "$expected":*) ;; *) exit 9;; esac
code=$("$exe" env --shell sh --clear)
eval "$code"
test -z "${VERMAN_PATH_ENTRIES-}"
printf '%s\n' SH_OK
`
	output, err := exec.Command(shell, "-c", script).CombinedOutput()
	if err != nil || !strings.Contains(string(output), "SH_OK") {
		t.Fatalf("sh: %s %v", output, err)
	}
}

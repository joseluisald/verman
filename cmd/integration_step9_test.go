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
	"github.com/joseluisald/verman/internal/installer"
)

func step9Setup(t *testing.T) (string, string, string) {
	t.Helper()
	home := filepath.Join(t.TempDir(), "home O'Brien $cash & test")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("VERMAN_PATH_ENTRIES", "")
	t.Setenv("VERMAN_ENV_BACKUP", "")
	// Evite gravar o cache do PowerShell fora do diretório temporário do teste.
	t.Setenv("PSModuleAnalysisCachePath", filepath.Join(home, "powershell-cache"))
	binary := filepath.Join(t.TempDir(), "verman.exe")
	if runtime.GOOS != "windows" {
		binary = strings.TrimSuffix(binary, ".exe")
	}
	if out, err := exec.Command("go", "build", "-o", binary, "..").CombinedOutput(); err != nil {
		t.Fatalf("build: %s %v", out, err)
	}
	project := t.TempDir()
	for _, version := range []string{"22.11.0", "22.12.0"} {
		root, err := config.InstallDir("node", version)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(root, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "node.exe"), []byte("fixture"), 0700); err != nil {
			t.Fatal(err)
		}
		data, err := json.Marshal(installer.Metadata{SchemaVersion: 1, Language: "node", Version: version, OS: runtime.GOOS, Arch: runtime.GOARCH, Source: "https://example.test/node.zip", Executables: map[string]string{"node": "node.exe"}})
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, installer.MetadataFileName), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := config.SaveProjectTool(project, "node", "22.12.0"); err != nil {
		t.Fatal(err)
	}
	return binary, project, home
}

func TestPowerShellHookAndGlobal(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell Windows")
	}
	binary, project, home := step9Setup(t)
	if err := config.SaveGlobalTool("node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Fatal(err)
	}
	quote := func(v string) string { return "'" + strings.ReplaceAll(v, "'", "''") + "'" }
	script := `$ErrorActionPreference = 'Stop'
$exe = ` + quote(binary) + `
Set-Location ` + quote(home) + `
function global:prompt { 'ORIGINAL_PROMPT' }
$code = & $exe init --shell powershell
if ($LASTEXITCODE -ne 0) { throw 'init falhou' }
$code | Out-String | Invoke-Expression
$code | Out-String | Invoke-Expression
if (-not $env:VERMAN_PATH_ENTRIES) { throw 'init nao aplicou ambiente imediatamente' }
Set-Location ` + quote(project) + `
$global:LASTEXITCODE = 7
if ((prompt) -ne 'ORIGINAL_PROMPT') { throw 'prompt original perdido' }
if ($global:LASTEXITCODE -ne 7) { throw 'codigo de saida perdido' }
if (($env:PATH -split ';')[0] -ne ` + quote(filepath.Join(home, ".verman", "installs", "node", "22.12.0")) + `) { throw 'projeto nao aplicado' }
Set-Location ` + quote(home) + `
$null = prompt
if (($env:PATH -split ';')[0] -ne ` + quote(filepath.Join(home, ".verman", "installs", "node", "22.11.0")) + `) { throw 'global nao aplicado' }
# Configuracao quebrada deve limpar a selecao anterior, sem instalar nada.
Set-Location ` + quote(project) + `
Set-Content -LiteralPath .verman.toml -Value '[broken' -Encoding ASCII
$null = prompt
if ($env:VERMAN_PATH_ENTRIES) { throw 'ambiente anterior permaneceu em erro' }
if ($ErrorActionPreference -ne 'Stop') { throw 'preferencia de erro alterada' }
$code = & $exe init --shell powershell --disable
if ($LASTEXITCODE -ne 0) { throw 'disable falhou' }
$code | Out-String | Invoke-Expression
if ((prompt) -ne 'ORIGINAL_PROMPT') { throw 'prompt nao restaurado' }
Write-Output 'HOOK_OK'
`
	path := filepath.Join(t.TempDir(), "test.ps1")
	if err := os.WriteFile(path, append([]byte{0xef, 0xbb, 0xbf}, []byte(script)...), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path).CombinedOutput()
	if err != nil || !strings.Contains(string(output), "HOOK_OK") {
		t.Fatalf("PowerShell: %s %v", output, err)
	}
}

func TestBashHookAndPOSIX(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("alvo validado: Git Bash no Windows")
	}
	bash := `C:\Program Files\Git\bin\bash.exe`
	if _, err := os.Stat(bash); err != nil {
		t.Skip("Git Bash não disponível")
	}
	binary, project, home := step9Setup(t)
	if err := config.SaveGlobalTool("node", "22.11.0"); err != nil {
		t.Fatal(err)
	}
	quote := func(v string) string { return "'" + strings.ReplaceAll(filepath.ToSlash(v), "'", "'\"'\"'") + "'" }
	script := `set -e
exe=$(cygpath -u -- ` + quote(binary) + `)
project=$(cygpath -u -- ` + quote(project) + `)
home=$(cygpath -u -- ` + quote(home) + `)
cd "$home"
PROMPT_COMMAND='printf original >/dev/null'
eval "$("$exe" init --shell bash)"
eval "$("$exe" init --shell bash)"
[[ -n ${VERMAN_PATH_ENTRIES-} ]]
[[ ${#PROMPT_COMMAND[@]} -eq 2 ]]
cd "$project"
__verman_autoenv
expected=$(cygpath -u -- ` + quote(filepath.Join(home, ".verman", "installs", "node", "22.12.0")) + `)
[[ ${PATH%%:*} == "$expected" ]]
__verman_autoenv
[[ ${PATH%%:*} == "$expected" ]]
cd "$home"
__verman_autoenv
expected=$(cygpath -u -- ` + quote(filepath.Join(home, ".verman", "installs", "node", "22.11.0")) + `)
[[ ${PATH%%:*} == "$expected" ]]
code=$("$exe" env --shell sh --clear)
eval "$code"
[[ -z ${VERMAN_PATH_ENTRIES-} ]]
eval "$("$exe" init --shell bash --disable)"
[[ ${#PROMPT_COMMAND[@]} -eq 1 ]]
printf '%s\n' BASH_OK
`
	path := filepath.Join(t.TempDir(), "test.sh")
	if err := os.WriteFile(path, []byte(script), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(bash, "--noprofile", "--norc", "-c", "source "+quote(path)).CombinedOutput()
	if err != nil || !strings.Contains(string(output), "BASH_OK") {
		t.Fatalf("Bash: %s %v", output, err)
	}
}

func TestExecPreservesArgumentsAndExitCode(t *testing.T) {
	binary, project, _ := step9Setup(t)
	// Execute o binário de teste como filho nativo; nenhum shell interpreta os argumentos.
	testBinary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(binary, "exec", "--", testBinary, "-test.run=^TestExecChildHelper$", "--", "space value", "$cash;&'quote")
	command.Dir = project
	command.Env = append(os.Environ(), "VERMAN_EXEC_TEST=1")
	output, err := command.CombinedOutput()
	exit, ok := err.(*exec.ExitError)
	if !ok || exit.ExitCode() != 7 || !strings.Contains(string(output), "ARGS_OK") {
		t.Fatalf("exec: %s %v", output, err)
	}
}

func TestExecChildHelper(t *testing.T) {
	if os.Getenv("VERMAN_EXEC_TEST") != "1" {
		return
	}
	args := os.Args
	if len(args) < 2 || args[len(args)-2] != "space value" || args[len(args)-1] != "$cash;&'quote" {
		os.Exit(9)
	}
	if !strings.Contains(strings.Split(os.Getenv("PATH"), string(os.PathListSeparator))[0], "22.12.0") {
		os.Exit(8)
	}
	fmtMessage := "ARGS_OK\n"
	_, _ = os.Stdout.WriteString(fmtMessage)
	os.Exit(7)
}

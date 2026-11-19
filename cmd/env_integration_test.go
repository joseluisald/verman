package cmd

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/joseluisald/verman/internal/installer"
)

// Este teste requer rede e valida a CLI e a resolução real do PATH no PowerShell.
func TestOfficialPowerShellWorkflow(t *testing.T) {
	if os.Getenv("VERMAN_INTEGRATION") != "1" {
		t.Skip("defina VERMAN_INTEGRATION=1 para validar runtimes oficiais")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("alvo Windows/amd64")
	}
	t.Setenv("PSModuleAnalysisCachePath", filepath.Join(t.TempDir(), "powershell-cache"))
	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(t.TempDir(), "teste espaço O'Brien $cash & (test)")
	if err := os.MkdirAll(workspace, 0700); err != nil {
		t.Fatal(err)
	}
	home := filepath.Join(workspace, "home")
	if err := os.Mkdir(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("VERMAN_PATH_ENTRIES", "")
	binary := filepath.Join(workspace, "verman.exe")
	if out, err := exec.Command("go", "build", "-o", binary, "..").CombinedOutput(); err != nil {
		t.Fatalf("build: %s %v", out, err)
	}
	for _, version := range []string{"22.11.0", "22.12.0"} {
		if out, err := exec.Command(binary, "install", "node", version).CombinedOutput(); err != nil {
			t.Fatalf("instalar: %s %v", out, err)
		}
	}
	for _, alias := range []string{"latest", "lts"} {
		version, err := installer.ResolveVersion(context.Background(), "node", alias, io.Discard)
		if err != nil || version == "" || version == alias {
			t.Fatalf("alias %s: %s %v", alias, version, err)
		}
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	projectA, projectB := filepath.Join(workspace, "project-a"), filepath.Join(workspace, "project-b")
	for _, project := range []string{projectA, projectB} {
		if err := os.Mkdir(project, 0700); err != nil {
			t.Fatal(err)
		}
	}
	bash := `C:\Program Files\Git\bin\bash.exe`
	if _, err := os.Stat(bash); err == nil {
		bashQuote := func(s string) string { return "'" + strings.ReplaceAll(filepath.ToSlash(s), "'", "'\"'\"'") + "'" }
		bashCode := `set -e
exe=$(cygpath -u -- ` + bashQuote(binary) + `)
cd "$(cygpath -u -- ` + bashQuote(projectA) + `)"
"$exe" use node 22.11.0
code=$("$exe" env --shell sh)
eval "$code"
[[ $(node --version) == v22.11.0 ]]
npm --version
npx --version
[[ $("$exe" exec -- node --version) == v22.11.0 ]]
"$exe" exec -- npm --version
printf '%s\n' REAL_BASH_OK
`
		output, err := exec.Command(bash, "--noprofile", "--norc", "-c", bashCode).CombinedOutput()
		if err != nil || !strings.Contains(string(output), "REAL_BASH_OK") {
			t.Fatalf("Node Git Bash: %s %v", output, err)
		}
	}
	script := `$ErrorActionPreference = 'Stop'
$exe = ` + quote(binary) + `
& $exe global node 22.12.0
if ($LASTEXITCODE -ne 0) { throw 'global falhou' }
function Activate-Verman {
    $code = & $exe env --shell powershell
    if ($LASTEXITCODE -ne 0) { throw 'falha em env' }
    $code | Out-String | Invoke-Expression
}
Set-Location ` + quote(projectA) + `
& $exe use node 22.11.0
if ($LASTEXITCODE -ne 0) { throw 'falha em use A' }
Activate-Verman
Activate-Verman
if ((node --version) -ne 'v22.11.0') { throw 'versão A incorreta' }
$parts = @($env:PATH -split ';' | Where-Object { $_ -eq ` + quote(filepath.Join(home, ".verman", "installs", "node", "22.11.0")) + ` })
if ($parts.Count -ne 1) { throw 'PATH duplicado' }
npm --version
if ($LASTEXITCODE -ne 0) { throw 'npm A falhou' }
npx --version
if ($LASTEXITCODE -ne 0) { throw 'npx A falhou' }
if ((& $exe exec -- node --version) -ne 'v22.11.0') { throw 'exec usou versao errada' }
& $exe exec -- npm --version
if ($LASTEXITCODE -ne 0) { throw 'exec npm falhou' }
& $exe exec -- npx --version
if ($LASTEXITCODE -ne 0) { throw 'exec npx falhou' }
Set-Location ` + quote(projectB) + `
& $exe use node 22.12.0
if ($LASTEXITCODE -ne 0) { throw 'falha em use B' }
Activate-Verman
if ((node --version) -ne 'v22.12.0') { throw 'versão B incorreta' }
if (($env:PATH -split ';') -contains ` + quote(filepath.Join(home, ".verman", "installs", "node", "22.11.0")) + `) { throw 'versão A ficou no PATH' }
npm --version
if ($LASTEXITCODE -ne 0) { throw 'npm B falhou' }
npx --version
if ($LASTEXITCODE -ne 0) { throw 'npx B falhou' }
$code = & $exe env --shell powershell --clear
$code | Out-String | Invoke-Expression
$shimDir = Join-Path ` + quote(workspace) + ` 'shims'
& $exe setup --bin-dir $shimDir
if ($LASTEXITCODE -ne 0) { throw 'setup shims Node falhou' }
$env:PATH = $shimDir + ';' + $env:PATH
Set-Location ` + quote(projectA) + `
if ((node --version) -ne 'v22.11.0') { throw 'shim Node A falhou' }
npm --version
if ($LASTEXITCODE -ne 0) { throw 'shim npm falhou' }
npx --version
if ($LASTEXITCODE -ne 0) { throw 'shim npx falhou' }
Set-Location ` + quote(projectB) + `
if ((node --version) -ne 'v22.12.0') { throw 'shim Node B falhou' }
$ErrorActionPreference = 'Continue'
& $exe uninstall node 22.12.0 2>$null
if ($LASTEXITCODE -eq 0) { throw 'remoção de selecionada permitida sem opção' }
$ErrorActionPreference = 'Stop'
& $exe uninstall node 22.11.0
if ($LASTEXITCODE -ne 0) { throw 'remoção A falhou' }
& $exe uninstall node 22.12.0 --allow-selected
if ($LASTEXITCODE -ne 0) { throw 'remoção B falhou' }
& $exe list node
if ($LASTEXITCODE -ne 0) { throw 'list falhou' }
Write-Output 'WORKFLOW_OK'
`
	path := filepath.Join(workspace, "workflow.ps1")
	if err := os.WriteFile(path, append([]byte{0xef, 0xbb, 0xbf}, []byte(script)...), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "WORKFLOW_OK") {
		t.Fatalf("workflow PowerShell: %s, %v", out, err)
	}
}

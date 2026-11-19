package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWindowsBuildNpmResolutionAndNativeInvocation(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell Windows")
	}
	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Fatal(err)
	}
	helper, err := filepath.Abs(filepath.Join("..", "..", "scripts", "build-tools.ps1"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	t.Setenv("PSModuleAnalysisCachePath", filepath.Join(root, "ps-cache"))
	t.Setenv("VERMAN_BUILD_HELPERS", helper)
	t.Setenv("VERMAN_BUILD_TEST_ROOT", root)
	script := `$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
. $env:VERMAN_BUILD_HELPERS
$originalPath=$env:PATH
$nativeShell=Join-Path $PSHOME 'powershell.exe'
$bin=Join-Path $env:VERMAN_BUILD_TEST_ROOT "npm com espaço O'Brien & teste"
New-Item -ItemType Directory -Path $bin | Out-Null
$cmd=Join-Path $bin 'npm.cmd'
$exe=Join-Path $bin 'npm.exe'
try {
    # Somente resolucao: fixtures nunca sao executadas.
    $env:PATH=$bin
    [IO.File]::WriteAllText($cmd,'fixture: nunca executar')
    if ((Resolve-BuildNpm) -cne $cmd) { throw 'npm.cmd tradicional nao reconhecido' }
    [IO.File]::WriteAllText($exe,'fixture: nunca executar')
    if ((Resolve-BuildNpm) -cne $exe) { throw 'npm.exe nao teve prioridade' }
    Remove-Item -LiteralPath $cmd
    if ((Resolve-BuildNpm) -cne $exe) { throw 'exigiu npm.cmd mesmo com npm.exe presente' }
    Remove-Item -LiteralPath $exe
    [IO.File]::WriteAllText((Join-Path $bin 'npm.ps1'),"throw 'nao executar npm.ps1'")
    try { Resolve-BuildNpm; throw 'deveria falhar sem npm nativo' }
    catch { if ($_.Exception.Message -notlike '*verman use node lts*') { throw } }
} finally { $env:PATH=$originalPath }
$echoScript=Join-Path $env:VERMAN_BUILD_TEST_ROOT 'args com espaço.ps1'
[IO.File]::WriteAllText($echoScript,'param([string]$Value) Write-Output $Value')
$argument='espaço O''Brien & literal; dollar$(nao-executar)'
$output=@(Invoke-BuildTool -Executable $nativeShell -CommandArgs @('-NoProfile','-NonInteractive','-ExecutionPolicy','Bypass','-File',$echoScript,$argument) -Failure 'argumentos')
if (($output -join '') -cne $argument) { throw 'argumentos alterados ou interpretados' }
try {
    Invoke-BuildTool -Executable $nativeShell -CommandArgs @('-NoProfile','-NonInteractive','-Command','exit 7') -Failure 'falha esperada'
    throw 'codigo de saida ignorado'
} catch { if ($_.Exception.Message -ne 'falha esperada (codigo 7)') { throw } }
if ($ErrorActionPreference -ne 'Stop') { throw 'preferencia de erro nao restaurada' }
Write-Output 'OK'
`
	path := filepath.Join(root, "build-tools-test.ps1")
	if err := os.WriteFile(path, append([]byte{0xef, 0xbb, 0xbf}, []byte(script)...), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command(shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path).CombinedOutput()
	if err != nil || strings.TrimSpace(string(output)) != "OK" {
		t.Fatalf("regressao do build: %v\n%s", err, output)
	}
}

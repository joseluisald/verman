package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/joseluisald/verman/internal/installer"
)

// Downloads reais são opt-in; inclui execução após mover a instalação para seu destino.
func TestOfficialPHPPythonWorkflow(t *testing.T) {
	if os.Getenv("VERMAN_INTEGRATION") != "1" {
		t.Skip("defina VERMAN_INTEGRATION=1 para validar PHP/Python reais")
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("MVP Windows/amd64")
	}
	t.Setenv("PSModuleAnalysisCachePath", filepath.Join(t.TempDir(), "powershell-cache"))
	shell, err := exec.LookPath("powershell.exe")
	if err != nil {
		t.Fatal(err)
	}
	workspace := filepath.Join(t.TempDir(), "espaço O'Brien $cash & test")
	home := filepath.Join(workspace, "home")
	if err := os.MkdirAll(home, 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("VERMAN_PATH_ENTRIES", "")
	t.Setenv("VERMAN_ENV_BACKUP", "")
	binary := filepath.Join(workspace, "verman.exe")
	if out, err := exec.Command("go", "build", "-o", binary, "..").CombinedOutput(); err != nil {
		t.Fatalf("build: %s %v", out, err)
	}
	versions := map[string][]string{}
	for _, language := range []string{"php", "python"} {
		catalog, err := installer.Available(context.Background(), language, true)
		if err != nil {
			t.Fatal(err)
		}
		if len(catalog.Catalog.Artifacts) < 2 {
			t.Fatalf("catálogo %s sem duas versões", language)
		}
		for _, artifact := range catalog.Catalog.Artifacts[:2] {
			versions[language] = append(versions[language], artifact.Version)
			t.Logf("instalando %s %s (%s)", language, artifact.Version, artifact.Build)
			process := exec.Command(binary, "install", language, artifact.Version, "--json", "--progress", "always")
			var diagnostics bytes.Buffer
			process.Stderr = &diagnostics
			out, err := process.Output()
			if err != nil {
				t.Fatalf("instalar: %s %s %v", out, &diagnostics, err)
			}
			var result struct {
				Schema  int                `json:"schema"`
				Success bool               `json:"success"`
				Data    installer.Metadata `json:"data"`
			}
			if err := json.Unmarshal(out, &result); err != nil || result.Schema != 2 || !result.Success || result.Data.Language != language || result.Data.Version != artifact.Version {
				t.Fatalf("JSON de instalação: %s %v", out, err)
			}
			if !bytes.Contains(diagnostics.Bytes(), []byte("MiB")) || bytes.Contains(diagnostics.Bytes(), []byte("\r")) {
				t.Fatalf("progresso em pipe: %s", &diagnostics)
			}
		}
	}
	type projectCase struct {
		Project string `json:"project"`
		PHP     string `json:"php"`
		Python  string `json:"python"`
	}
	cases := []projectCase{}
	for i := 0; i < 2; i++ {
		project := filepath.Join(workspace, []string{"project-a", "project-b"}[i])
		if err := os.Mkdir(project, 0700); err != nil {
			t.Fatal(err)
		}
		cases = append(cases, projectCase{project, versions["php"][i], versions["python"][i]})
	}
	data, err := json.Marshal(cases)
	if err != nil {
		t.Fatal(err)
	}
	quote := func(s string) string { return "'" + strings.ReplaceAll(s, "'", "''") + "'" }
	script := `$ErrorActionPreference = 'Stop'
$exe = ` + quote(binary) + `
$cases = ConvertFrom-Json -InputObject ` + quote(string(data)) + `
$env:PHPRC = 'original-php-config'
$env:PYTHONHOME = 'invalid-python-home'
$env:PYTHONPATH = 'original-python-path'
$externalPHPDir = Join-Path $env:USERPROFILE ('.verman/installs/php/' + $cases[1].php)
$externalPHP = Join-Path $externalPHPDir 'php.exe'
$env:PATH = $externalPHPDir + ';' + $env:PATH
$firstCase = $true
function Activate-Verman {
    $code = & $exe env --shell powershell
    if ($LASTEXITCODE -ne 0) { throw 'env falhou' }
    $code | Out-String | Invoke-Expression
}
foreach ($case in $cases) {
    Set-Location $case.project
    & $exe use php $case.php
    if ($LASTEXITCODE -ne 0) { throw 'use php falhou' }
    & $exe use python $case.python
    if ($LASTEXITCODE -ne 0) { throw 'use python falhou' }
    if ($firstCase) {
        if ((php -r 'echo PHP_VERSION;') -ne $cases[1].php) { throw 'o cenário sem ativação não foi reproduzido' }
        $ErrorActionPreference = 'Continue'
        $rawDiagnosis = & $exe doctor php --json 2>$null
        $diagnosticExit = $LASTEXITCODE
        $ErrorActionPreference = 'Stop'
		$diagnosis = ($rawDiagnosis | Out-String | ConvertFrom-Json).data
        if ($diagnosticExit -ne 1 -or $diagnosis.tools[0].status -ne 'activation_required') { throw 'doctor não identificou o conflito' }
        if ((& $exe exec -- php -r 'echo PHP_VERSION;') -ne $case.php) { throw 'exec dependeu de ativação' }
        $hook = & $exe init --shell powershell
        if ($LASTEXITCODE -ne 0) { throw 'init falhou' }
        $hook | Out-String | Invoke-Expression
        if ((php -r 'echo PHP_VERSION;') -ne $case.php) { throw 'init não ativou imediatamente' }
        Set-Alias -Name php -Value $externalPHP -Scope Global
        if ((php -r 'echo PHP_VERSION;') -ne $cases[1].php) { throw 'conflito por alias não reproduzido' }
        $aliasDiagnosis = ((& $exe doctor php --json) | Out-String | ConvertFrom-Json).data
        if ($LASTEXITCODE -ne 0 -or $aliasDiagnosis.shell_aliases_inspected -ne $false -or $aliasDiagnosis.tools[0].path_matches -ne $true) { throw 'doctor alegou inspecionar o shell pai' }
        Remove-Item Alias:php
        $firstCase = $false
    }
    Activate-Verman
    Activate-Verman
    if ((php -r 'echo PHP_VERSION;') -ne $case.php) { throw 'PHP errado' }
    if ($LASTEXITCODE -ne 0) { throw 'php falhou' }
    $phpVersionOutput = php -v
    if (($phpVersionOutput | Out-String) -notlike ('*PHP ' + $case.php + '*')) { throw 'php -v não corresponde à seleção' }
    $checkPHP = Join-Path $case.project 'check.php'
    Set-Content -LiteralPath $checkPHP -Encoding ASCII -Value '<?php if (!php_ini_loaded_file() || basename(php_ini_loaded_file()) !== "php.ini") { exit(1); } echo "INI_OK";'
    $iniResult = php $checkPHP
    if ($iniResult -ne 'INI_OK' -or $LASTEXITCODE -ne 0) { throw "php.ini da instalação não foi carregado: $iniResult" }
    Set-Content -LiteralPath $checkPHP -Encoding ASCII -Value '<?php if (!extension_loaded("curl")) { exit(1); } echo "EXT_OK";'
    if ((php -d extension=curl $checkPHP) -ne 'EXT_OK' -or $LASTEXITCODE -ne 0) { throw 'extension_dir da instalação inválido' }
    if ((python -c 'import platform; print(platform.python_version())') -ne $case.python) { throw 'Python errado' }
    if ($LASTEXITCODE -ne 0) { throw 'python falhou' }
    python -m pip --version
    if ($LASTEXITCODE -ne 0) { throw 'pip falhou' }
    & $exe exec -- php --version
    if ($LASTEXITCODE -ne 0) { throw 'exec php falhou' }
    & $exe exec -- python --version
    if ($LASTEXITCODE -ne 0) { throw 'exec python falhou' }
    & $exe exec -- pip --version
    if ($LASTEXITCODE -ne 0) { throw 'exec pip falhou' }
    pip --version
    if ($LASTEXITCODE -ne 0) { throw 'wrapper pip falhou' }
    python -m venv .venv
    if ($LASTEXITCODE -ne 0) { throw 'venv falhou após publicação' }
    & ./.venv/Scripts/python.exe -m pip --version
    if ($LASTEXITCODE -ne 0) { throw 'pip do venv falhou' }
    & ./.venv/Scripts/python.exe -c 'import sys; assert sys.prefix != sys.base_prefix'
    if ($LASTEXITCODE -ne 0) { throw 'venv não isolado' }
    & $exe list php
    & $exe list python
}
# O ambiente virtual anterior continua vinculado à versão usada ao criá-lo.
$oldVenv = Join-Path $cases[0].project '.venv/Scripts/python.exe'
if ((& $oldVenv -c 'import platform; print(platform.python_version())') -ne $cases[0].python) { throw 'venv anterior mudou de versão' }
$clearCode = & $exe env --shell powershell --clear
$clearCode | Out-String | Invoke-Expression
$shimDir = Join-Path ` + quote(workspace) + ` 'shims'
& $exe setup --bin-dir $shimDir
if ($LASTEXITCODE -ne 0) { throw 'setup de shims falhou' }
$env:PATH = $shimDir + ';' + $env:PATH
foreach ($case in $cases) {
    Set-Location $case.project
    & $exe use php $case.php
    if ($LASTEXITCODE -ne 0) { throw 'use por shim falhou' }
    if ((php -r 'echo PHP_VERSION;') -ne $case.php) { throw 'shim PHP usou versao errada' }
    if ((python -c 'import platform; print(platform.python_version())') -ne $case.python) { throw 'shim Python usou versao errada' }
    pip --version
    if ($LASTEXITCODE -ne 0) { throw 'shim pip falhou' }
    $diagnosis = ((& $exe doctor php --json) | Out-String | ConvertFrom-Json).data
    if ($LASTEXITCODE -ne 0 -or -not $diagnosis.tools[0].uses_shim) { throw 'doctor nao reconheceu shim' }
    & $exe verify php $case.php
    if ($LASTEXITCODE -ne 0) { throw 'manifesto PHP invalido' }
    & $exe verify python $case.python
    if ($LASTEXITCODE -ne 0) { throw 'manifesto Python invalido' }
}
$processes = @()
foreach ($case in $cases) {
    $process = New-Object Diagnostics.Process
    $process.StartInfo.FileName = Join-Path $shimDir 'php.exe'
    $process.StartInfo.Arguments = '-r "echo PHP_VERSION;"'
    $process.StartInfo.WorkingDirectory = $case.project
    $process.StartInfo.UseShellExecute = $false
    $process.StartInfo.RedirectStandardOutput = $true
    [void]$process.Start()
    $processes += $process
}
for ($i = 0; $i -lt $cases.Count; $i++) {
    $version = $processes[$i].StandardOutput.ReadToEnd()
    $processes[$i].WaitForExit()
    if ($processes[$i].ExitCode -ne 0 -or $version -ne $cases[$i].php) { throw 'processos concorrentes misturaram selecoes' }
    $processes[$i].Dispose()
}
$customIni = Join-Path $env:USERPROFILE ('.verman/installs/php/' + $cases[0].php + '/php.ini')
Add-Content -LiteralPath $customIni -Value ';CUSTOM_PRESERVED'
& $exe repair php $cases[0].php
if ($LASTEXITCODE -ne 0) { throw 'repair PHP falhou' }
if ((Get-Content -LiteralPath $customIni -Raw) -notlike '*CUSTOM_PRESERVED*') { throw 'repair apagou configuracao' }
& $exe verify php $cases[0].php
if ($LASTEXITCODE -ne 0) { throw 'integridade apos repair falhou' }
Set-Location ` + quote(workspace) + `
Activate-Verman
if ($env:PHPRC -ne 'original-php-config' -or $env:PYTHONHOME -ne 'invalid-python-home' -or $env:PYTHONPATH -ne 'original-python-path') { throw 'variáveis anteriores não restauradas' }
& $exe global php $cases[1].php
if ($LASTEXITCODE -ne 0) { throw 'global com shim falhou' }
if ((php -r 'echo PHP_VERSION;') -ne $cases[1].php) { throw 'shim global usou versao errada' }
& $exe global --unset php
if ($LASTEXITCODE -ne 0) { throw 'unset global falhou' }
foreach ($case in $cases) {
    & $exe uninstall php $case.php
    if ($LASTEXITCODE -ne 0) { throw 'uninstall php falhou' }
    & $exe uninstall python $case.python
    if ($LASTEXITCODE -ne 0) { throw 'uninstall python falhou' }
}
Write-Output 'LANGUAGES_OK'
`
	path := filepath.Join(workspace, "languages.ps1")
	if err := os.WriteFile(path, append([]byte{0xef, 0xbb, 0xbf}, []byte(script)...), 0600); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(shell, "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-File", path).CombinedOutput()
	if err != nil || !strings.Contains(string(out), "LANGUAGES_OK") {
		t.Fatalf("PHP/Python PowerShell: %s %v", out, err)
	}
	for _, language := range []string{"php", "python"} {
		items, err := installer.ListInstalled(language)
		if err != nil || len(items) != 0 {
			t.Fatalf("remoção %s: %v %v", language, items, err)
		}
	}
}

package environment

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joseluisald/verman/internal/config"
	"github.com/joseluisald/verman/internal/installer"
)

// PowerShell gera apenas código: nenhum ambiente é alterado no processo Go.
// Um script block mantém as variáveis auxiliares fora do escopo do chamador.
func PowerShell(dir string) (string, error) {
	paths, variables, err := selectedEnvironment(dir)
	if err != nil {
		return "", err
	}
	return renderPowerShell(paths, variables)
}

func selectedEnvironment(dir string) ([]string, map[string]*string, error) {
	cfg, err := config.LoadEffectiveConfig(dir)
	if err != nil {
		return nil, nil, err
	}
	languages := make([]string, 0, len(cfg.Tools))
	for language := range cfg.Tools {
		languages = append(languages, language)
	}
	sort.Strings(languages)
	paths := []string{}
	variables := map[string]*string{}
	for _, language := range languages {
		dirs, err := installer.BinDirs(language, cfg.Tools[language])
		if err != nil {
			return nil, nil, fmt.Errorf("ativar %s %s: instalação ausente ou inválida; execute verman install %s %s: %w", language, cfg.Tools[language], language, cfg.Tools[language], err)
		}
		for _, path := range dirs {
			if strings.ContainsAny(path, ";\r\n\x00") {
				return nil, nil, fmt.Errorf("diretório não representável no PATH: %q", path)
			}
			paths = append(paths, path)
		}
		if language == "php" {
			root, err := config.InstallDir(language, cfg.Tools[language])
			if err != nil {
				return nil, nil, err
			}
			scan := filepath.Join(root, "conf.d")
			variables["PHPRC"] = &root
			variables["PHP_INI_SCAN_DIR"] = &scan
		}
		if language == "python" {
			variables["PYTHONHOME"] = nil
			variables["PYTHONPATH"] = nil
		}
		root, err := config.InstallDir(language, cfg.Tools[language])
		if err != nil {
			return nil, nil, err
		}
		for key, value := range installer.ToolVariables(language, cfg.Tools[language], root) {
			copy := value
			variables[key] = &copy
		}
	}
	return paths, variables, nil
}

func RenderPowerShell(paths []string) (string, error) {
	return renderPowerShell(paths, map[string]*string{})
}

func renderPowerShell(paths []string, variables map[string]*string) (string, error) {
	if paths == nil {
		paths = []string{}
	}
	data, err := json.Marshal(paths)
	if err != nil {
		return "", err
	}
	// JSON é passado como string literal PowerShell; aspas simples são duplicadas.
	literal := strings.ReplaceAll(string(data), "'", "''")
	variableData, err := json.Marshal(variables)
	if err != nil {
		return "", err
	}
	variableLiteral := strings.ReplaceAll(string(variableData), "'", "''")
	return `& {
    $ErrorActionPreference = 'Stop'
    $next = ConvertFrom-Json -InputObject '` + literal + `'
    $next = @($next)
    $old = @()
    if ($env:VERMAN_PATH_ENTRIES) {
        try { $old = ConvertFrom-Json -InputObject $env:VERMAN_PATH_ENTRIES; $old = @($old) }
        catch { throw 'VERMAN_PATH_ENTRIES invalido; o PATH nao foi alterado.' }
        foreach ($entry in $old) {
            if ($entry -isnot [string]) { throw 'VERMAN_PATH_ENTRIES invalido; o PATH nao foi alterado.' }
        }
    }
    $separator = [string][IO.Path]::PathSeparator
    $remaining = @()
    if ($null -ne $env:PATH) {
        $remaining = @($env:PATH -split [regex]::Escape($separator) | Where-Object {
            $old -notcontains $_ -and $next -notcontains $_
        })
    }
    $updated = (@($next) + @($remaining)) -join $separator
    $marker = ConvertTo-Json -InputObject @($next) -Compress
    $nextVars = ConvertFrom-Json -InputObject '` + variableLiteral + `'
    $oldVars = @{}
    if ($env:VERMAN_ENV_BACKUP) {
        try {
            $parsed = ConvertFrom-Json -InputObject $env:VERMAN_ENV_BACKUP
            if ($parsed -isnot [pscustomobject]) { throw 'invalid' }
            foreach ($property in $parsed.PSObject.Properties) {
                if (@('PHPRC','PHP_INI_SCAN_DIR','PYTHONHOME','PYTHONPATH','GOROOT','GOTOOLCHAIN','CARGO_HOME','RUSTUP_HOME','RUSTUP_TOOLCHAIN','VCPKG_ROOT','CHOCOLATEYINSTALL') -notcontains $property.Name) { throw 'invalid' }
                if ($null -ne $property.Value -and $property.Value -isnot [string]) { throw 'invalid' }
                $oldVars[$property.Name] = $property.Value
            }
        } catch { throw 'VERMAN_ENV_BACKUP invalido; o ambiente nao foi alterado.' }
    }
    $savedVars = @{}
    foreach ($property in $nextVars.PSObject.Properties) {
        if ($oldVars.ContainsKey($property.Name)) { $savedVars[$property.Name] = $oldVars[$property.Name] }
        else { $savedVars[$property.Name] = [Environment]::GetEnvironmentVariable($property.Name, 'Process') }
    }
    $backup = ConvertTo-Json -InputObject $savedVars -Compress
    foreach ($name in $oldVars.Keys) { [Environment]::SetEnvironmentVariable($name, $oldVars[$name], 'Process') }
    foreach ($property in $nextVars.PSObject.Properties) { [Environment]::SetEnvironmentVariable($property.Name, $property.Value, 'Process') }
    if ($savedVars.Count -gt 0) { $env:VERMAN_ENV_BACKUP = $backup }
    else { Remove-Item Env:VERMAN_ENV_BACKUP -ErrorAction SilentlyContinue }
    $env:PATH = $updated
    if ($next.Count -gt 0) { $env:VERMAN_PATH_ENTRIES = $marker }
    else { Remove-Item Env:VERMAN_PATH_ENTRIES -ErrorAction SilentlyContinue }
}
`, nil
}

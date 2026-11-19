package environment

import (
	"fmt"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var temporaryExecutable = regexp.MustCompile(`(?i)(^|[/\\])(go-build[0-9]+[/\\]|go-build[/\\][a-f0-9]{2}[/\\][a-f0-9]{64}-d[/\\])`)

func IsTemporaryExecutable(path string) bool { return temporaryExecutable.MatchString(path) }

func Hook(shell, executable string, disable bool) (string, error) {
	if !disable && IsTemporaryExecutable(executable) {
		return "", fmt.Errorf("init requer um executável permanente; go run usa um binário temporário. Compile com go build -o verman.exe . e execute .\\verman.exe init --shell %s", shell)
	}
	if shell == "powershell" {
		if disable {
			return `if ($global:__VermanHookInstalled) {
    $code = & $global:__VermanExecutable env --shell powershell --clear
    if ($LASTEXITCODE -eq 0) { $code | Out-String | Invoke-Expression }
    Set-Item Function:global:prompt -Value $global:__VermanOriginalPrompt
    Remove-Item Function:global:__VermanSyncEnvironment -ErrorAction SilentlyContinue
    Remove-Variable __VermanHookInstalled,__VermanOriginalPrompt,__VermanExecutable -Scope Global -ErrorAction SilentlyContinue
}
`, nil
		}
		literal := "'" + strings.ReplaceAll(executable, "'", "''") + "'"
		return `if (-not (Get-Variable __VermanHookInstalled -Scope Global -ErrorAction SilentlyContinue)) {
    $global:__VermanExecutable = ` + literal + `
    $global:__VermanOriginalPrompt = (Get-Item Function:prompt).ScriptBlock
    $global:__VermanHookInstalled = $true
    function global:__VermanSyncEnvironment {
        $savedExit = Get-Variable LASTEXITCODE -Scope Global -ValueOnly -ErrorAction SilentlyContinue
        $savedPreference = $ErrorActionPreference
        $ErrorActionPreference = 'Continue'
        try {
            $code = & $global:__VermanExecutable env --shell powershell
            if ($LASTEXITCODE -ne 0) { $code = & $global:__VermanExecutable env --shell powershell --clear }
            if ($LASTEXITCODE -eq 0) { $code | Out-String | Invoke-Expression }
        } catch { Write-Warning $_ }
        finally { $global:LASTEXITCODE = $savedExit; $ErrorActionPreference = $savedPreference }
    }
    function global:prompt {
        __VermanSyncEnvironment
        & $global:__VermanOriginalPrompt
    }
}
__VermanSyncEnvironment
`, nil
	}
	if shell != "bash" {
		return "", fmt.Errorf("hook disponível para powershell ou bash (Bash 5+); sh usa env manualmente")
	}
	if disable {
		return `__verman_prompt_commands=()
for __verman_command in "${PROMPT_COMMAND[@]}"; do
    if [[ "$__verman_command" != '__verman_autoenv' ]]; then __verman_prompt_commands+=("$__verman_command"); fi
done
PROMPT_COMMAND=("${__verman_prompt_commands[@]}")
if [[ -n "${__verman_executable-}" ]]; then
    if __verman_code=$("$__verman_executable" env --shell bash --clear); then eval "$__verman_code"; fi
fi
unset -f __verman_autoenv
unset __verman_prompt_commands __verman_command __verman_code __verman_executable
`, nil
	}
	path := ShellQuote(filepath.ToSlash(executable))
	assignment := "__verman_executable=" + path + "\n"
	if runtime.GOOS == "windows" {
		assignment = "__verman_executable=$(command cygpath -u -- " + path + ") || return 1\n"
	}
	return `if (( BASH_VERSINFO[0] < 5 )); then printf '%s\n' 'Verman: o hook requer Bash 5+.' >&2; return 1; fi
` + assignment + `__verman_autoenv() {
    local previous=$? code
    if code=$("$__verman_executable" env --shell bash); then eval "$code"
    elif code=$("$__verman_executable" env --shell bash --clear); then eval "$code"; fi
    return "$previous"
}
if [[ $(declare -p PROMPT_COMMAND 2>/dev/null) != 'declare -a'* ]]; then
    PROMPT_COMMAND=("${PROMPT_COMMAND-}")
fi
__verman_has_hook=0
for __verman_command in "${PROMPT_COMMAND[@]}"; do
    if [[ "$__verman_command" == '__verman_autoenv' ]]; then __verman_has_hook=1; fi
done
if (( ! __verman_has_hook )); then PROMPT_COMMAND+=(__verman_autoenv); fi
unset __verman_has_hook __verman_command
__verman_autoenv
`, nil
}

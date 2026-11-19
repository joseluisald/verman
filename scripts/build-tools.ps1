# Funcoes compartilhadas do build; carregar este arquivo nao executa ferramentas.
function Resolve-BuildNpm {
    # npm.exe e o shim nativo do Verman. npm.cmd atende instalacoes comuns do Node.
    # Nao use npm.ps1: ele depende da politica de execucao do PowerShell.
    foreach ($candidate in @('npm.exe', 'npm.cmd')) {
        $command = Get-Command -Name $candidate -CommandType Application -ErrorAction SilentlyContinue | Select-Object -First 1
        if ($command) { return $command.Path }
    }
    throw "Node/npm nao encontrado no PATH. O build aceita npm.exe (Verman) ou npm.cmd (Node.js). Se usar Verman, execute 'verman use node lts' na raiz do projeto e confira 'npm --version'. Reabra o terminal depois de configurar o PATH."
}

function Invoke-BuildTool {
    param([string]$Executable,[string[]]$CommandArgs,[string]$Failure)
    $command = Get-Command -Name $Executable -CommandType Application -ErrorAction Stop | Select-Object -First 1
    $previousPreference = $ErrorActionPreference
    try {
        # PowerShell 5 trata stderr nativo como ErrorRecord sob redirecionamento.
        # Avisos com exit 0 nao sao falhas do build.
        $ErrorActionPreference = 'Continue'
        & $command.Path @CommandArgs 2>&1 | ForEach-Object { Write-Output $_.ToString() }
        $toolExit = $LASTEXITCODE
    } finally { $ErrorActionPreference = $previousPreference }
    if ($toolExit -ne 0) { throw "$Failure (codigo $toolExit)" }
}

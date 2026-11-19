param(
    [ValidatePattern('^\d+\.\d+\.\d+$')][string]$Version='0.5.2',
    [string]$InnoCompiler,
    [switch]$SkipNpmInstall,
    [switch]$SkipInstaller
)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
. (Join-Path $PSScriptRoot 'build-tools.ps1')
$root=(Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$payload=Join-Path $root 'dist\windows\payload'
$output=Join-Path $root 'dist'
New-Item -ItemType Directory -Force -Path $payload | Out-Null
Push-Location $root
try {
    Push-Location (Join-Path $root 'desktop\frontend')
    try {
        $npmExecutable=Resolve-BuildNpm
        Write-Output "npm do build: $npmExecutable"
        if (-not $SkipNpmInstall) { Invoke-BuildTool -Executable $npmExecutable -CommandArgs @('ci','--ignore-scripts','--no-audit','--no-fund') -Failure 'npm ci falhou' }
        Invoke-BuildTool -Executable $npmExecutable -CommandArgs @('run','build') -Failure 'Build da interface falhou'
    } finally { Pop-Location }
    Invoke-BuildTool -Executable go -CommandArgs @('run','./tools/windowsicon','-out',(Join-Path $payload 'assets')) -Failure 'Exportacao do logo Windows falhou'
    Invoke-BuildTool -Executable go -CommandArgs @('build','-trimpath','-buildvcs=false','-ldflags',"-s -w -X github.com/joseluisald/verman/cmd.Version=$Version",'-o',(Join-Path $payload 'verman.exe'),'.') -Failure 'Build da CLI falhou'
    Invoke-BuildTool -Executable go -CommandArgs @('build','-trimpath','-buildvcs=false','-tags','desktop,production','-ldflags',"-s -w -H windowsgui -X main.Version=$Version",'-o',(Join-Path $payload 'verman-desktop.exe'),'./desktop') -Failure 'Build desktop falhou'
    Invoke-BuildTool -Executable go -CommandArgs @('run','./tools/windowsicon','-exe',(Join-Path $payload 'verman-desktop.exe'),'-icon',(Join-Path $payload 'assets\verman.ico')) -Failure 'Incorporacao do icone do desktop falhou'
    Invoke-BuildTool -Executable (Join-Path $payload 'verman.exe') -CommandArgs @('setup','--bin-dir',(Join-Path $payload 'bin')) -Failure 'Preparacao dos shims falhou'
    Invoke-BuildTool -Executable go -CommandArgs @('run','./tools/windowsbundle','-out',$payload) -Failure 'Documentacao offline falhou'
    if (-not $SkipInstaller) {
        if (-not $InnoCompiler) {
            foreach ($candidate in @((Join-Path $root 'dist\tooling\inno\ISCC.exe'),(Join-Path ${env:ProgramFiles(x86)} 'Inno Setup 6\ISCC.exe'),(Join-Path $env:ProgramFiles 'Inno Setup 7\ISCC.exe'))) { if (Test-Path -LiteralPath $candidate -PathType Leaf) { $InnoCompiler=$candidate; break } }
        }
        if (-not $InnoCompiler) { throw 'Informe -InnoCompiler com o caminho do ISCC.exe (Inno Setup). O payload ja foi gerado.' }
        Invoke-BuildTool -Executable $InnoCompiler -CommandArgs @("/DProductVersion=$Version","/DPayloadDir=$payload","/DOutputDir=$output",(Join-Path $root 'installer\windows\verman.iss')) -Failure 'Compilacao do instalador falhou'
        $installer=Join-Path $output "Verman-Setup-$Version-windows-amd64.exe"
        $hash=(Get-FileHash -LiteralPath $installer -Algorithm SHA256).Hash.ToLowerInvariant()
        [IO.File]::WriteAllText(($installer+'.sha256'),($hash+'  '+[IO.Path]::GetFileName($installer)+[Environment]::NewLine))
        Write-Output "Instalador: $installer"
    }
    Write-Output "CLI, desktop e documentacao: $payload"
} finally { Pop-Location }

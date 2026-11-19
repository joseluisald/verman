param(
    [ValidateSet('User','Machine')][string]$Scope = 'User',
    [string]$VermanExecutable,
    [string]$BinDir,
    [switch]$AllowMachinePathConflicts,
    [switch]$Preview
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
if (-not $VermanExecutable) {
    $VermanExecutable = Join-Path $PSScriptRoot 'verman.exe'
    if (-not (Test-Path -LiteralPath $VermanExecutable -PathType Leaf)) { $VermanExecutable = Join-Path $PSScriptRoot '..\verman.exe' }
}
if (-not $BinDir) {
    if ($Scope -eq 'Machine') { $BinDir = Join-Path $env:ProgramFiles 'Verman\bin' }
    else { $BinDir = Join-Path $env:USERPROFILE '.verman\bin' }
}
$bin = [IO.Path]::GetFullPath($BinDir).TrimEnd('\')
$source = (Resolve-Path -LiteralPath $VermanExecutable).Path
$machinePath = [Environment]::GetEnvironmentVariable('Path','Machine')
$conflicts = @()
foreach ($entry in ($machinePath -split ';')) {
    if (-not $entry) { continue }
    $directory = [Environment]::ExpandEnvironmentVariables($entry.Trim('"'))
    if (Test-Path -LiteralPath (Join-Path $directory '.verman-shims.json')) { continue }
    foreach ($tool in @('php.exe','node.exe','python.exe','bun.exe','bunx.exe','go.exe','gofmt.exe','deno.exe','git.exe','rust.exe','rustc.exe','rustdoc.exe','cargo.exe','rustup.exe','composer.exe','yarn.exe','yarnpkg.exe','pnpm.exe','vcpkg.exe','chocolatey.exe','choco.exe')) {
        if (Test-Path -LiteralPath (Join-Path $directory $tool) -PathType Leaf) { $conflicts += (Join-Path $directory $tool) }
    }
}
Write-Output "Escopo: $Scope; shims: $bin"
if ($Scope -eq 'User' -and $conflicts.Count -gt 0) {
    Write-Output ('Conflitos no PATH do sistema: ' + ($conflicts -join ', '))
    if (-not $AllowMachinePathConflicts) {
        throw 'O PATH do usuario nao tem prioridade sobre o do sistema. Use -Scope Machine em PowerShell administrador, ou configure a prioridade manualmente. Nenhuma alteracao foi aplicada.'
    }
    Write-Warning 'Comandos no PATH do sistema podem ter prioridade sobre os shims do usuario. Use verman exec para executar a versao selecionada.'
}
if ($Preview) { Write-Output 'Previa concluida; arquivos e PATH nao foram alterados.'; return }
if ($Scope -eq 'Machine') {
    $principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
    if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) { throw 'Abra o PowerShell como administrador para configurar o PATH do sistema.' }
    $programFilesRoot = [IO.Path]::GetFullPath($env:ProgramFiles).TrimEnd('\') + '\'
    if (-not $bin.StartsWith($programFilesRoot,[StringComparison]::OrdinalIgnoreCase)) { throw 'No escopo Machine use um destino protegido dentro de Program Files.' }
}
& $source setup --bin-dir $bin
if ($LASTEXITCODE -ne 0) { throw 'Falha ao instalar shims; PATH preservado.' }
if ($Scope -eq 'Machine') { $key = [Microsoft.Win32.Registry]::LocalMachine.OpenSubKey('SYSTEM\CurrentControlSet\Control\Session Manager\Environment',$true) }
else { $key = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment') }
try {
    $oldPath = [string]$key.GetValue('Path','',[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    try { $kind = $key.GetValueKind('Path') } catch { $kind = [Microsoft.Win32.RegistryValueKind]::ExpandString }
    $remaining = @($oldPath -split ';' | Where-Object { $_.Trim('"').TrimEnd('\') -ine $bin })
    $newPath = (@($bin) + $remaining) -join ';'
    $backup = @{ scope=$Scope; path=$oldPath; kind=[string]$kind; bin=$bin } | ConvertTo-Json
    $backupFile = Join-Path $bin ('path-backup-' + [Guid]::NewGuid().ToString('N') + '.json')
    [IO.File]::WriteAllText($backupFile,$backup)
    $key.SetValue('Path',$newPath,$kind)
} finally { $key.Dispose() }
if (-not ('VermanPathNotify' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class VermanPathNotify {
 [DllImport("user32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
 public static extern IntPtr SendMessageTimeout(IntPtr h,uint m,UIntPtr w,string l,uint f,uint t,out UIntPtr r);
}
'@
}
$result = [UIntPtr]::Zero
[void][VermanPathNotify]::SendMessageTimeout([IntPtr]0xffff,0x1a,[UIntPtr]::Zero,'Environment',2,5000,[ref]$result)
$env:PATH = $bin + ';' + ((@($env:PATH -split ';' | Where-Object { $_.Trim('"').TrimEnd('\') -ine $bin })) -join ';')
Write-Output "PATH configurado. Backup: $backupFile"
Write-Output 'A sessao que executou o script ja usa os shims. Reabra outros terminais/editores uma vez. Depois, use troca a selecao sem reaplicar env. Instalacoes anteriores foram preservadas.'

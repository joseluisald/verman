param(
    [Parameter(Mandatory=$true)][string]$BinDir,
    [ValidateSet('User','Machine')][string]$Scope = 'Machine',
    [switch]$Preview
)
$ErrorActionPreference='Stop'
Set-StrictMode -Version Latest
$bin=[IO.Path]::GetFullPath($BinDir).TrimEnd('\')
if ([IO.Path]::GetFileName($bin) -ine 'bin') { throw 'Destino de desinstalacao deve ser a pasta bin dos shims.' }
if ($Scope -eq 'Machine') {
    $protectedRoot=[IO.Path]::GetFullPath($env:ProgramFiles).TrimEnd('\')+'\'
    if (-not $bin.StartsWith($protectedRoot,[StringComparison]::OrdinalIgnoreCase)) { throw 'No escopo Machine use a pasta bin protegida dentro de Program Files.' }
    $hive=[Microsoft.Win32.Registry]::LocalMachine
    $keyPath='SYSTEM\CurrentControlSet\Control\Session Manager\Environment'
} else {
    $hive=[Microsoft.Win32.Registry]::CurrentUser
    $keyPath='Environment'
}
if (-not $Preview -and -not (Test-Path -LiteralPath (Join-Path $bin '.verman-shims.json') -PathType Leaf)) { throw 'Manifesto do Verman ausente; PATH preservado.' }
$key=$hive.OpenSubKey($keyPath, -not $Preview)
if ($null -eq $key) { return }
try {
    $oldPath=[string]$key.GetValue('Path','',[Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    $parts=@($oldPath -split ';' | Where-Object { $_.Trim('"').TrimEnd('\') -ine $bin })
    $nextPath=$parts -join ';'
    if ($Preview) { Write-Output 'Previa: somente a entrada exata dos shims seria removida. Nenhuma alteracao aplicada.'; return }
    if ($oldPath -ceq $nextPath) { return }
    $kind=$key.GetValueKind('Path')
    $backup=@{path=$oldPath;kind=[string]$kind;scope=$Scope;bin=$bin} | ConvertTo-Json
    [IO.File]::WriteAllText((Join-Path $bin ('path-uninstall-backup-'+[Guid]::NewGuid().ToString('N')+'.json')),$backup)
    $key.SetValue('Path',$nextPath,$kind)
} finally { $key.Dispose() }
if (-not ('VermanUninstallNotify' -as [type])) {
    Add-Type -TypeDefinition @'
using System;
using System.Runtime.InteropServices;
public static class VermanUninstallNotify {
 [DllImport("user32.dll", CharSet=CharSet.Unicode, SetLastError=true)]
 public static extern IntPtr SendMessageTimeout(IntPtr h,uint m,UIntPtr w,string l,uint f,uint t,out UIntPtr r);
}
'@
}
$result=[UIntPtr]::Zero
[void][VermanUninstallNotify]::SendMessageTimeout([IntPtr]0xffff,0x1a,[UIntPtr]::Zero,'Environment',2,5000,[ref]$result)
Write-Output 'Entrada dos shims removida. Outros caminhos, runtimes e configuracoes foram preservados.'

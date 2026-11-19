param([string]$BinDir = (Join-Path $env:LOCALAPPDATA 'verman\bin'))
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$resolvedBin = [IO.Path]::GetFullPath($BinDir).TrimEnd('\')
$target = [IO.Path]::GetFullPath((Join-Path $resolvedBin 'verman.exe'))
if ([IO.Path]::GetDirectoryName($target) -ne $resolvedBin) { throw 'Destino fora do diretorio bin.' }
if (Test-Path -LiteralPath $target) { Remove-Item -LiteralPath $target }
Write-Output 'Executavel removido. Runtimes e configuracoes em ~/.verman foram preservados.'
Write-Output 'Remova do perfil as linhas de init do Verman e o diretorio bin do PATH, se os adicionou manualmente.'

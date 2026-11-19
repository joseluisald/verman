param(
    [Parameter(Mandatory = $true)][string]$Archive,
    [Parameter(Mandatory = $true)][string]$Checksums,
    [string]$BinDir = (Join-Path $env:LOCALAPPDATA 'verman\bin')
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
$archivePath = (Resolve-Path -LiteralPath $Archive).Path
$checksumPath = (Resolve-Path -LiteralPath $Checksums).Path
$archiveName = [IO.Path]::GetFileName($archivePath)
$matchesFound = @(Get-Content -LiteralPath $checksumPath | ForEach-Object {
    if ($_ -match '^([a-fA-F0-9]{64})\s+\*?(.+)$' -and $Matches[2] -ceq $archiveName) { $Matches[1].ToLowerInvariant() }
})
if ($matchesFound.Count -ne 1) { throw 'Checksum ausente ou duplicado para o pacote informado.' }
if ((Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant() -ne $matchesFound[0]) { throw 'SHA-256 do pacote nao confere.' }
$resolvedBin = [IO.Path]::GetFullPath($BinDir)
if ($resolvedBin.TrimEnd('\') -eq [IO.Path]::GetPathRoot($resolvedBin).TrimEnd('\')) { throw 'Escolha um diretorio bin, nao a raiz do volume.' }
New-Item -ItemType Directory -Path $resolvedBin -Force | Out-Null
$resolvedBin = (Get-Item -LiteralPath $resolvedBin).FullName
$target = [IO.Path]::GetFullPath((Join-Path $resolvedBin 'verman.exe'))
if ([IO.Path]::GetDirectoryName($target) -ne $resolvedBin.TrimEnd('\')) { throw 'Destino fora do diretorio bin.' }
if ((Test-Path -LiteralPath $target) -and ((Get-Item -LiteralPath $target).Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw 'Destino do executavel e um link; substituicao recusada.' }
$temporary = Join-Path $resolvedBin ('.verman-' + [Guid]::NewGuid().ToString('N') + '.tmp')
$backup = Join-Path $resolvedBin ('.verman-' + [Guid]::NewGuid().ToString('N') + '.previous')
Add-Type -AssemblyName System.IO.Compression.FileSystem
$zip = [IO.Compression.ZipFile]::OpenRead($archivePath)
try {
    $entries = @($zip.Entries | Where-Object { $_.FullName -ceq 'verman.exe' })
    if ($entries.Count -ne 1 -or $entries[0].Length -gt 134217728) { throw 'Pacote sem executavel unico valido.' }
    $inputStream = $entries[0].Open()
    try {
        $outputStream = [IO.File]::Open($temporary, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write)
        try { $inputStream.CopyTo($outputStream) } finally { $outputStream.Dispose() }
    } finally { $inputStream.Dispose() }
    if ((Get-Item -LiteralPath $temporary).Length -ne $entries[0].Length) { throw 'Executavel extraido com tamanho incorreto.' }
    if (Test-Path -LiteralPath $target) { [IO.File]::Replace($temporary, $target, $backup) }
    else { [IO.File]::Move($temporary, $target) }
    if (Test-Path -LiteralPath $backup) { Remove-Item -LiteralPath $backup }
} finally {
    $zip.Dispose()
    if (Test-Path -LiteralPath $temporary) { Remove-Item -LiteralPath $temporary }
}
Write-Output "Verman instalado/atualizado em $target"
Write-Output 'Adicione esse diretorio ao PATH do usuario ou execute pelo caminho completo. Nenhum perfil ou PATH persistente foi alterado.'

$ErrorActionPreference = 'Stop'
$projectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$watcher = Join-Path $PSScriptRoot 'watch-qr-network.ps1'
$pidFile = Join-Path $projectRoot 'backend/data/qr-network-watcher.pid'
$addressFile = Join-Path $projectRoot 'backend/data/qr-lan-url.txt'

$running = $false
if (Test-Path -LiteralPath $pidFile) {
    $oldProcessId = Get-Content -LiteralPath $pidFile -ErrorAction SilentlyContinue
    if ($oldProcessId -match '^\d+$') {
        $fresh = (Test-Path -LiteralPath $addressFile) -and (((Get-Date) - (Get-Item -LiteralPath $addressFile).LastWriteTime).TotalSeconds -lt 30)
        $running = $fresh -and [bool](Get-Process -Id ([int]$oldProcessId) -ErrorAction SilentlyContinue)
    }
}
if (-not $running) {
    & $watcher -Once
    $arguments = "-NoProfile -NonInteractive -ExecutionPolicy Bypass -File `"$watcher`""
    $watcherProcess = Start-Process -FilePath (Get-Command powershell.exe).Source -ArgumentList $arguments -WindowStyle Hidden -PassThru
    [System.IO.File]::WriteAllText($pidFile, [string]$watcherProcess.Id)
}

Push-Location $projectRoot
try {
    docker compose up -d
    Write-Output 'Proyecto: http://localhost:3001'
    Write-Output "Dirección para el celular: $(Get-Content -LiteralPath $addressFile -Raw)"
} finally {
    Pop-Location
}

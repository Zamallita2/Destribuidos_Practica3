param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('mongo', 'america', 'europa')]
    [string]$Role,

    [ValidateSet('database', 'app', 'all')]
    [string]$Stage = 'all',

    [switch]$AllowDegraded
)

$ErrorActionPreference = 'Stop'
$projectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$configPath = Join-Path $projectRoot '.env.distributed'
if (-not (Test-Path -LiteralPath $configPath)) {
    throw 'Falta .env.distributed. Copia .env.distributed.example y completa las IP y los secretos.'
}

$settings = @{}
foreach ($line in Get-Content -LiteralPath $configPath) {
    $item = $line.Trim()
    if ($item.Length -eq 0 -or $item.StartsWith('#')) { continue }
    $parts = $item.Split('=', 2)
    if ($parts.Count -ne 2 -or [string]::IsNullOrWhiteSpace($parts[0])) {
        throw "Linea invalida en .env.distributed: $item"
    }
    $settings[$parts[0].Trim()] = $parts[1].Trim()
}
foreach ($key in @('PC_MONGO_IP', 'PC_AM_IP', 'PC_EU_IP', 'POSTGRES_PASSWORD', 'MONGO_PASSWORD', 'BOARDING_PASS_SECRET')) {
    if (-not $settings.ContainsKey($key) -or $settings[$key] -match '^CAMBIAR_') {
        throw "Configura $key en .env.distributed"
    }
}
foreach ($key in @('POSTGRES_PASSWORD', 'MONGO_PASSWORD')) {
    if ($settings[$key] -notmatch '^[A-Za-z0-9]{16,}$') {
        throw "$key debe tener al menos 16 caracteres alfanumericos (sin caracteres especiales en URI/DSN)."
    }
}
if ($settings['BOARDING_PASS_SECRET'] -notmatch '^[A-Za-z0-9]{32,}$') {
    throw 'BOARDING_PASS_SECRET debe tener por lo menos 32 caracteres alfanumericos.'
}
foreach ($key in @('PC_MONGO_IP', 'PC_AM_IP', 'PC_EU_IP')) {
    $parsed = $null
    if (-not [System.Net.IPAddress]::TryParse($settings[$key], [ref]$parsed) -or
        $parsed.AddressFamily -ne [System.Net.Sockets.AddressFamily]::InterNetwork -or
        [System.Net.IPAddress]::IsLoopback($parsed)) {
        throw "$key debe ser una IPv4 LAN valida."
    }
}

$roleConfig = @{
    mongo = @{ IP = 'PC_MONGO_IP'; Service = 'mongodb'; Node = 'pc_mongo' }
    america = @{ IP = 'PC_AM_IP'; Service = 'postgres_am'; Node = 'pc_am' }
    europa = @{ IP = 'PC_EU_IP'; Service = 'postgres_eu'; Node = 'pc_eu' }
}
$mine = $roleConfig[$Role]
$localIP = $settings[$mine.IP]
$assignedIPs = @(Get-NetIPAddress -AddressFamily IPv4 -ErrorAction Stop | Select-Object -ExpandProperty IPAddress)
if ($localIP -notin $assignedIPs) {
    throw "La IP $localIP ($($mine.IP)) no esta asignada a esta PC. Revisa el Wi-Fi y .env.distributed."
}

function Test-TcpPort([string]$Address, [int]$Port) {
    $client = [System.Net.Sockets.TcpClient]::new()
    try {
        $attempt = $client.ConnectAsync($Address, $Port)
        return $attempt.Wait(1500) -and $client.Connected
    } catch {
        return $false
    } finally {
        $client.Dispose()
    }
}

$env:NODE_ID = $mine.Node
$env:PUBLIC_BASE_URL = "http://${localIP}:3001"
$env:CLUSTER_BOOTSTRAP = if ($Role -eq 'mongo') { 'true' } else { 'false' }
$env:PG_AM_HOST = if ($Role -eq 'america') { 'postgres_am' } else { $settings['PC_AM_IP'] }
$env:PG_EU_HOST = if ($Role -eq 'europa') { 'postgres_eu' } else { $settings['PC_EU_IP'] }
$env:PG_EU_PORT = if ($Role -eq 'europa') { '5432' } else { '5435' }
$env:MONGO_HOST = if ($Role -eq 'mongo') { 'mongodb' } else { $settings['PC_MONGO_IP'] }
$composeArgs = @('compose', '--env-file', $configPath, '-f', (Join-Path $projectRoot 'docker-compose.distributed.yml'), '-p', 'airres_distributed', '--profile', $Role)

Push-Location $projectRoot
try {
    if ($Stage -in @('database', 'all')) {
        & docker @composeArgs up -d $mine.Service
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo iniciar la base de datos local.' }
        Write-Output "Base de datos $($mine.Service) iniciada en $localIP."
    }

    if ($Stage -in @('app', 'all')) {
        if ($Role -eq 'mongo') {
            $reachable = (Test-TcpPort $settings['PC_AM_IP'] 5432) -and
                (Test-TcpPort $settings['PC_EU_IP'] 5435) -and
                (Test-TcpPort $settings['PC_MONGO_IP'] 27017)
            if (-not $reachable) {
                if (-not $AllowDegraded) {
                    throw 'Para la carga inicial, primero inicia las tres bases de datos y comprueba los puertos LAN. Usa -AllowDegraded solo para reiniciar durante una caida.'
                }
                $env:CLUSTER_BOOTSTRAP = 'false'
                Write-Warning 'Iniciando sin carga inicial porque alguna base de datos no responde.'
            }
        }
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
        & docker @composeArgs up -d --build backend frontend
        if ($LASTEXITCODE -ne 0) { throw 'No se pudo iniciar la API y el frontend.' }
        Write-Output "Aplicacion de esta PC: http://${localIP}:3001/sincronizacion"
    }
} finally {
    Pop-Location
}

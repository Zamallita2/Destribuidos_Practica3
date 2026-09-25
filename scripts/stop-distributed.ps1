param(
    [Parameter(Mandatory = $true)]
    [ValidateSet('mongo', 'america', 'europa')]
    [string]$Role,

    [switch]$AppOnly
)

$ErrorActionPreference = 'Stop'
$projectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$configPath = Join-Path $projectRoot '.env.distributed'
if (-not (Test-Path -LiteralPath $configPath)) { throw 'Falta .env.distributed.' }

# Compose interpolates these variables even for stop/down; no connection is made.
$env:NODE_ID = 'stopping'
$env:PUBLIC_BASE_URL = 'http://localhost:3001'
$env:PG_AM_HOST = 'localhost'
$env:PG_EU_HOST = 'localhost'
$env:PG_EU_PORT = '5432'
$env:MONGO_HOST = 'localhost'
$composeArgs = @('compose', '--env-file', $configPath, '-f', (Join-Path $projectRoot 'docker-compose.distributed.yml'), '-p', 'airres_distributed', '--profile', $Role)

Push-Location $projectRoot
try {
    if ($AppOnly) {
        & docker @composeArgs stop backend frontend
    } else {
        & docker @composeArgs down
    }
    if ($LASTEXITCODE -ne 0) { throw 'No se pudieron detener los contenedores distribuidos.' }
} finally {
    Pop-Location
}

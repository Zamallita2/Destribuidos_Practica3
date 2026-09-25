param(
    [switch]$Once,
    [int]$IntervalSeconds = 10
)

$ErrorActionPreference = 'Stop'
$projectRoot = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$target = Join-Path $projectRoot 'backend/data/qr-lan-url.txt'

function Get-CurrentLanURL {
    $adapters = @(Get-NetIPConfiguration | Where-Object {
        $_.NetAdapter -and $_.NetAdapter.Status -eq 'Up' -and
        $_.IPv4DefaultGateway -and $_.IPv4Address -and
        $_.InterfaceAlias -notmatch 'vEthernet|WSL|Docker|Virtual|VPN|Tailscale'
    })
    $preferred = @($adapters | Where-Object { $_.InterfaceAlias -match 'Wi-?Fi|WLAN|Wireless' })
    if ($preferred.Count -gt 0) { $adapters = $preferred }
    foreach ($adapter in $adapters) {
        foreach ($address in @($adapter.IPv4Address)) {
            $ip = $address.IPAddress
            if ($ip -and $ip -notmatch '^(127\.|169\.254\.|0\.)') {
                return "http://${ip}:3001"
            }
        }
    }
    return ''
}

do {
    try {
        $url = Get-CurrentLanURL
        $previous = if (Test-Path -LiteralPath $target) { (Get-Content -LiteralPath $target -Raw).Trim() } else { $null }
        $temporary = "$target.tmp"
        [System.IO.File]::WriteAllText($temporary, $url)
        Move-Item -LiteralPath $temporary -Destination $target -Force
        if ($url -ne $previous) {
            Write-Output "Dirección del QR: $(if ($url) { $url } else { 'sin red local' })"
        }
    } catch {
        Write-Error "No se pudo detectar la red local: $_" -ErrorAction Continue
    }
    if (-not $Once) { Start-Sleep -Seconds ([Math]::Max(3, $IntervalSeconds)) }
} while (-not $Once)

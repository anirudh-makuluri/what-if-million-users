# Port-forward all services for url-shortener Kubernetes deployment
# Usage: .\port-forward.ps1

param(
    [switch]$Stop,
    [switch]$List
)

$namespace = "url-shortener"

# Define port-forwards
$portForwards = @(
    @{ Service = "url-shortener"; LocalPort = 8083; RemotePort = 8080; Name = "App API" }
    @{ Service = "prometheus"; LocalPort = 9091; RemotePort = 9090; Name = "Prometheus" }
    @{ Service = "grafana"; LocalPort = 3000; RemotePort = 3000; Name = "Grafana" }
    @{ Service = "dynamodb-local"; LocalPort = 8000; RemotePort = 8000; Name = "DynamoDB" }
    @{ Service = "kafka"; LocalPort = 9092; RemotePort = 9092; Name = "Kafka" }
    @{ Service = "redis"; LocalPort = 6379; RemotePort = 6379; Name = "Redis" }
)

if ($List) {
    Write-Host "Available port-forwards:" -ForegroundColor Cyan
    $portForwards | ForEach-Object {
        Write-Host "  $($_.Name): localhost:$($_.LocalPort) -> $($_.Service):$($_.RemotePort)"
    }
    exit 0
}

if ($Stop) {
    Write-Host "Stopping all port-forwards..." -ForegroundColor Yellow
    Get-Process kubectl | Where-Object { $_.CommandLine -match "port-forward" } | ForEach-Object {
        Write-Host "  Killing kubectl process: $($_.ProcessName) (PID: $($_.Id))"
        Stop-Process -Id $_.Id -Force
    }
    Write-Host "All port-forwards stopped." -ForegroundColor Green
    exit 0
}

Write-Host "Starting Kubernetes port-forwards for $namespace namespace..." -ForegroundColor Cyan
Write-Host ""

# Start each port-forward in the background
$portForwards | ForEach-Object {
    $command = "kubectl port-forward svc/$($_.Service) $($_.LocalPort):$($_.RemotePort) -n $namespace"
    Write-Host "  $($_.Name)" -ForegroundColor Green
    Write-Host "    localhost:$($_.LocalPort) -> $($_.Service):$($_.RemotePort)"
    
    # Run in background job
    Start-Job -ScriptBlock {
        param($cmd)
        Invoke-Expression $cmd
    } -ArgumentList $command | Out-Null
}

Write-Host ""
Write-Host "Port-forwards started! You can access:" -ForegroundColor Green
Write-Host ""
$portForwards | ForEach-Object {
    Write-Host "  $($_.Name):`t`thttp://localhost:$($_.LocalPort)" -ForegroundColor White
}
Write-Host ""
Write-Host "To stop all port-forwards, run:" -ForegroundColor Yellow
Write-Host "  .\port-forward.ps1 -Stop" -ForegroundColor Cyan
Write-Host ""
Write-Host "To list available services, run:" -ForegroundColor Yellow
Write-Host "  .\port-forward.ps1 -List" -ForegroundColor Cyan

# Script to test rate limiting locally on Windows

Write-Host "Waiting for services to start..."
Start-Sleep -Seconds 5

Write-Host "Testing rate limiting..."
Write-Host ""

# Test 1: First request (should be allowed)
Write-Host "[Test 1] First request - should be allowed"
$body1 = @{
    client_id = "test-client-1"
    action = "create_post"
} | ConvertTo-Json

curl -X POST http://localhost:8080/api/request `
  -H "Content-Type: application/json" `
  -Body $body1
Write-Host ""
Write-Host ""

Start-Sleep -Seconds 1

# Test 2: Another request from same client
Write-Host "[Test 2] Second request - should be allowed"
$body2 = @{
    client_id = "test-client-1"
    action = "update_post"
} | ConvertTo-Json

curl -X POST http://localhost:8080/api/request `
  -H "Content-Type: application/json" `
  -Body $body2
Write-Host ""
Write-Host ""

# Test 3: Check metrics
Write-Host "[Test 3] Check Prometheus metrics"
curl http://localhost:8080/metrics
Write-Host ""
Write-Host ""

# Test 4: Health check
Write-Host "[Test 4] Health check"
curl http://localhost:8080/health
Write-Host ""
Write-Host ""

Write-Host "Done!"

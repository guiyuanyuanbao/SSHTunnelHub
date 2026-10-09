Write-Host "=========================================" -ForegroundColor Cyan
Write-Host "  Building SSHTunnelHub (Frontend + Backend)  " -ForegroundColor Cyan
Write-Host "=========================================" -ForegroundColor Cyan

# 1. Build Frontend
Write-Host "`n[1/3] Building Vue 3 Frontend..." -ForegroundColor Yellow
Set-Location -Path "frontend"
pnpm run build
if ($LASTEXITCODE -ne 0) {
    Write-Host "Frontend build failed!" -ForegroundColor Red
    Set-Location -Path ".."
    exit 1
}
Set-Location -Path ".."

# 2. Sync dist to backend embed folder
Write-Host "`n[2/3] Syncing dist to backend webui embed folder..." -ForegroundColor Yellow
if (!(Test-Path "backend\internal\webui\dist")) {
    New-Item -ItemType Directory -Force -Path "backend\internal\webui\dist" | Out-Null
}
Copy-Item -Recurse -Force "frontend\dist\*" "backend\internal\webui\dist"

# 3. Build Backend Binary
Write-Host "`n[3/3] Compiling Go integrated executable..." -ForegroundColor Yellow
Set-Location -Path "backend"
go build -ldflags "-s -w" -o "..\sshtunnelhub.exe" ./cmd/server
if ($LASTEXITCODE -ne 0) {
    Write-Host "Backend build failed!" -ForegroundColor Red
    Set-Location -Path ".."
    exit 1
}
Set-Location -Path ".."

Write-Host "`n=========================================" -ForegroundColor Green
Write-Host "Build complete! Output: sshtunnelhub.exe" -ForegroundColor Green
Write-Host "Run with: .\sshtunnelhub.exe" -ForegroundColor Green
Write-Host "Access UI: http://127.0.0.1:9090" -ForegroundColor Green
Write-Host "=========================================" -ForegroundColor Green

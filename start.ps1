# Gemini Image Web App 启动脚本
# 使用方法: 右键 -> 使用 PowerShell 运行，或在 PowerShell 中执行 .\start.ps1

param(
    [string]$ApiKey = $env:GEMINI_API_KEY,
    [string]$Port = "8080"
)

if (-not $ApiKey) {
    Write-Host "❌ 未找到 API Key！" -ForegroundColor Red
    Write-Host ""
    Write-Host "请通过以下方式之一提供 API Key：" -ForegroundColor Yellow
    Write-Host "  方式1（推荐）: 设置环境变量后运行" -ForegroundColor Cyan
    Write-Host '    $env:GEMINI_API_KEY = "your-api-key"' -ForegroundColor White
    Write-Host "    .\start.ps1" -ForegroundColor White
    Write-Host ""
    Write-Host "  方式2: 直接传参" -ForegroundColor Cyan
    Write-Host '    .\start.ps1 -ApiKey "your-api-key"' -ForegroundColor White
    Write-Host ""
    $ApiKey = Read-Host "或直接在此输入你的 API Key"
    if (-not $ApiKey) {
        Write-Host "未提供 API Key，退出。" -ForegroundColor Red
        exit 1
    }
}

$env:GEMINI_API_KEY = $ApiKey

Write-Host ""
Write-Host "🚀 启动 Gemini Image Web App..." -ForegroundColor Green
Write-Host "   端口: $Port" -ForegroundColor Cyan
Write-Host "   访问: http://localhost:$Port" -ForegroundColor Cyan
Write-Host "   按 Ctrl+C 停止服务" -ForegroundColor Gray
Write-Host ""

# 切换到 webapp 目录并运行
Set-Location $PSScriptRoot
.\gemini-webapp.exe

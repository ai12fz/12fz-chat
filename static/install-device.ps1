# ============================================================
# 12FZ 设备一键接入(Windows) — 自动识别已装 agent,注册并部署 bridge
# ============================================================
# 用法(在 PowerShell 里执行,由后台"复制安装命令"按钮提供):
#   powershell -ExecutionPolicy Bypass -File install-device.ps1 -Code dev-xxx
#   可选参数:
#     -Name 设备名    默认 <agent>-win-<hostname>(自动唯一)
#     -Agent hermes|12fzclaw|openclaw   强制指定,默认自动识别
#     -Ws ai.12fz.com 服务器主机(默认 ai.12fz.com)
#
# 自动识别逻辑(按优先级):
#   1. 命令存在: 12fzclaw > claw > hermes
#   2. 目录存在: ~\.12fzclaw > ~\.openclaw > ~\.hermes
#   3. 都没探测到: 默认 hermes
# 幂等: 已装过(12fz-bridge.json 存在)则跳过注册,只更新 agent_type 并重启计划任务
# 守护: schtasks 计划任务 Hermes12FZBridge 跑 watchdog,崩溃自动重启
# ============================================================
param(
  [string]$Code = "",
  [string]$Name = "",
  [string]$Agent = "",
  [string]$Ws = "ai.12fz.com"
)

$ErrorActionPreference = "Stop"

if (-not $Code) { Write-Host "!! 缺少注册码,请到 ai.12fz.com 后台设备管理生成" -ForegroundColor Red; exit 1 }

# ---------- 自动识别 agent_type ----------
function Detect-Agent {
  if ($Agent) { return $Agent }
  if (Get-Command 12fzclaw -ErrorAction SilentlyContinue) { return "12fzclaw" }
  if (Get-Command claw -ErrorAction SilentlyContinue) { return "openclaw" }
  if (Get-Command hermes -ErrorAction SilentlyContinue) { return "hermes" }
  if (Test-Path "$env:USERPROFILE\.12fzclaw") { return "12fzclaw" }
  if (Test-Path "$env:USERPROFILE\.openclaw") { return "openclaw" }
  if (Test-Path "$env:USERPROFILE\.hermes") { return "hermes" }
  return "hermes"
}

$AgentType = Detect-Agent
$HN = $env:COMPUTERNAME
if (-not $Name) { $Name = "$AgentType-win-$HN" }
Write-Host "== 12FZ 接入 agent_type=$AgentType 设备=$Name 码=$Code 端点=$Ws =="

$BridgeDir = "$env:USERPROFILE\12fz-bridge"
$Config = "$env:USERPROFILE\.hermes\12fz-bridge.json"
$BridgePy = "$BridgeDir\hermes-bridge.py"
$Watchdog = "$BridgeDir\bridge_watchdog.bat"

# ---------- 幂等: 已装过则跳过注册 ----------
if (Test-Path $Config) {
  try {
    $old = (Get-Content $Config -Raw | ConvertFrom-Json).agent_type
  } catch { $old = "" }
  if ($old -and $old -ne $AgentType) {
    Write-Host "== 已安装(agent=$old),更新为 $AgentType ..."
    $c = Get-Content $Config -Raw | ConvertFrom-Json
    $c.agent_type = $AgentType
    $c | ConvertTo-Json | Set-Content $Config -Encoding UTF8
    schtasks /End /TN "Hermes12FZBridge" 2>$null | Out-Null
    schtasks /Run /TN "Hermes12FZBridge" 2>$null | Out-Null
  } else {
    Write-Host "== 已安装(agent=$old),无需变更"
  }
  Write-Host "== 完成! 查看状态: schtasks /Query /TN Hermes12FZBridge"
  exit 0
}

New-Item -ItemType Directory -Force -Path $BridgeDir | Out-Null
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.hermes" | Out-Null

# ---------- 1) 注册设备 ----------
Write-Host "==> 注册设备..."
$body = @{ name = $Name; device_key = $Code; os = "windows"; agent_type = $AgentType } | ConvertTo-Json
try {
  $reg = Invoke-RestMethod -Uri "https://$Ws/api/devices/register" -Method Post -ContentType "application/json; charset=utf-8" -Body $body
} catch {
  Write-Host "!! 注册失败: $($_.Exception.Message)" -ForegroundColor Red
  exit 1
}
$DevToken = $reg.token
if (-not $DevToken) { Write-Host "!! 注册失败: $($reg | ConvertTo-Json -Compress)" -ForegroundColor Red; exit 1 }

# ---------- 2) 获取 sk-dev key ----------
Write-Host "==> 获取 sk-dev key..."
$setupResp = Invoke-RestMethod -Uri "https://$Ws/api/devices/setup?token=$DevToken"
$SkKey = $setupResp.key
if (-not $SkKey) { Write-Host "!! setup 失败: $($setupResp | ConvertTo-Json -Compress)" -ForegroundColor Red; exit 1 }

# ---------- 3) 写 bridge 配置 ----------
Write-Host "==> 写 bridge 配置 $Config"
$cfg = @{
  token = $DevToken
  bot_id = $Name
  ws_host = $Ws
  user_id = "1"
  agent_type = $AgentType
  doc_dir = "$env:USERPROFILE\research"
} | ConvertTo-Json
Set-Content -Path $Config -Value $cfg -Encoding UTF8

# ---------- 4) 配置 hermes 指向 12fz 代理 ----------
$envFile = "$env:USERPROFILE\.hermes\.env"
if (Test-Path $envFile) {
  $content = Get-Content $envFile -Raw
  if ($content -match "OPENAI_API_KEY=") {
    $content = $content -replace "(?m)^OPENAI_API_KEY=.*$", "OPENAI_API_KEY=$SkKey"
    Set-Content $envFile $content -Encoding UTF8
  } else {
    Add-Content $envFile "OPENAI_API_KEY=$SkKey" -Encoding UTF8
  }
} else {
  Set-Content $envFile "OPENAI_API_KEY=$SkKey" -Encoding UTF8
}

# ---------- 5) 下载 bridge 脚本 ----------
Write-Host "==> 下载 bridge 脚本..."
try {
  Invoke-WebRequest -Uri "https://$Ws/static/hermes-bridge-v8.py" -OutFile $BridgePy -UseBasicParsing
} catch {
  Write-Host "!! 下载 bridge 失败: $($_.Exception.Message)" -ForegroundColor Red
  exit 1
}

# ---------- 6) 写 watchdog + 注册计划任务 ----------
Write-Host "==> 注册计划任务 Hermes12FZBridge ..."
$py = (Get-Command python -ErrorAction SilentlyContinue).Source
if (-not $py) { $py = "python" }
@"
@echo off
:restart
"$py" -u "$BridgePy" >> "$env:USERPROFILE\bridge_v8.log" 2>&1
echo %date% %time% bridge exited with code %errorlevel%, restarting in 5s... >> "$env:USERPROFILE\bridge_watchdog.log"
timeout /t 5 /nobreak >nul
goto restart
"@ | Set-Content $Watchdog -Encoding ASCII

schtasks /Create /F /TN "Hermes12FZBridge" /TR "cmd /c `"$Watchdog`"" /SC ONSTART /RL HIGHEST 2>$null | Out-Null
schtasks /Run /TN "Hermes12FZBridge" 2>$null | Out-Null

Write-Host ""
Write-Host "==============================================" -ForegroundColor Green
Write-Host "  部署完成!" -ForegroundColor Green
Write-Host "  设备: $Name (agent_type=$AgentType)"
Write-Host "  服务: schtasks /Query /TN Hermes12FZBridge"
Write-Host "  日志: $env:USERPROFILE\bridge_v8.log"
Write-Host "  配置: $Config"
Write-Host "==============================================" -ForegroundColor Green

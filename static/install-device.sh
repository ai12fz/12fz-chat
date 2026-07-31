#!/bin/bash
# ============================================================
# 12FZ 设备一键接入 — 自动识别已装 agent,注册并部署 bridge
# ============================================================
# 用法(Linux/macOS):
#   curl -s https://ai.12fz.com/install-device.sh | bash -s -- --code=dev-xxx
#   可选参数:
#     --name=设备名       默认 <agent>-linux-<hostname>(自动唯一)
#     --agent=hermes|12fzclaw|openclaw   强制指定类型,默认自动识别
#     --ws=ai.12fz.com    服务器主机(默认 ai.12fz.com)
# Windows 设备用 PowerShell 版(前端按钮复制,install-device.ps1)
#
# 自动识别逻辑(按优先级):
#   1. 命令存在: 12fzclaw > claw > hermes
#   2. 目录存在: ~/.12fzclaw > ~/.openclaw > ~/.hermes
#   3. 都没探测到: 默认 hermes
# 幂等: 已装过(12fz-bridge.json 存在)则跳过注册,只更新 agent_type 并重启服务
# ============================================================
set -e

REG_CODE=""; BOT_ID=""; WS_HOST="ai.12fz.com"; AGENT_OVERRIDE=""
while [[ $# -gt 0 ]]; do
  case $1 in
    --code=*) REG_CODE="${1#*=}"; shift ;;
    --name=*) BOT_ID="${1#*=}"; shift ;;
    --ws=*)   WS_HOST="${1#*=}"; shift ;;
    --agent=*) AGENT_OVERRIDE="${1#*=}"; shift ;;
    *) echo "未知参数: $1 (支持 --code --name --agent --ws)"; exit 1 ;;
  esac
done
[ -z "$REG_CODE" ] && { echo "!! 缺少注册码,请到 ai.12fz.com 后台设备管理生成"; exit 1; }

# ---------- 自动识别 agent_type ----------
detect_agent() {
  [ -n "$AGENT_OVERRIDE" ] && { echo "$AGENT_OVERRIDE"; return; }
  command -v 12fzclaw >/dev/null 2>&1 && { echo "12fzclaw"; return; }
  command -v claw >/dev/null 2>&1 && { echo "openclaw"; return; }
  command -v hermes >/dev/null 2>&1 && { echo "hermes"; return; }
  [ -d "$HOME/.12fzclaw" ] && { echo "12fzclaw"; return; }
  [ -d "$HOME/.openclaw" ] && { echo "openclaw"; return; }
  [ -d "$HOME/.hermes" ] && { echo "hermes"; return; }
  echo "hermes"
}
AGENT_TYPE="$(detect_agent)"
[ "$AGENT_TYPE" = "claw" ] && AGENT_TYPE="openclaw"
HN=$(hostname 2>/dev/null | cut -d. -f1 || echo host)
[ -z "$BOT_ID" ] && BOT_ID="${AGENT_TYPE}-linux-${HN}"
echo "== 12FZ 接入 agent_type=$AGENT_TYPE 设备=$BOT_ID 码=$REG_CODE 端点=$WS_HOST =="

CONFIG="$HOME/.hermes/12fz-bridge.json"

# ---------- 幂等: 已装过则跳过注册 ----------
if [ -f "$CONFIG" ]; then
  OLD_AGENT=$(python3 -c "import json;print(json.load(open('$CONFIG')).get('agent_type',''))" 2>/dev/null || echo "")
  if [ -n "$OLD_AGENT" ] && [ "$OLD_AGENT" != "$AGENT_TYPE" ]; then
    echo "== 已安装(agent=$OLD_AGENT),更新为 $AGENT_TYPE ..."
    python3 - "$CONFIG" "$AGENT_TYPE" <<'PYEOF'
import json, sys
p, a = sys.argv[1], sys.argv[2]
c = json.load(open(p, encoding="utf-8"))
c["agent_type"] = a
json.dump(c, open(p, "w", encoding="utf-8"), ensure_ascii=False)
print("agent_type ->", a)
PYEOF
    sudo systemctl restart 12fz-bridge 2>/dev/null || systemctl --user restart 12fz-bridge 2>/dev/null || true
  else
    echo "== 已安装(agent=$OLD_AGENT),无需变更"
  fi
  echo "== 完成! 查看状态: systemctl status 12fz-bridge"
  exit 0
fi

# ---------- 全新安装: 下载并执行 setup ----------
echo "== 下载 setup 脚本..."
SETUP=$(mktemp)
curl -fsSL "https://$WS_HOST/static/12fz-bridge-setup.sh" -o "$SETUP" \
  || { echo "!! 下载 setup 失败,请检查网络/服务器"; rm -f "$SETUP"; exit 1; }
bash "$SETUP" "$BOT_ID" "$AGENT_TYPE" "$REG_CODE"
rm -f "$SETUP"

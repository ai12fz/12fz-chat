#!/bin/bash
# 12FZ Linux Hermes Bridge — 一键部署脚本
# 用法: bash setup-12fz-bridge.sh [设备名] [agent_type]
# 设备名默认 hermes-linux-qiuming,可改成你想要的(唯一即可)
# agent_type 可选: hermes / 12fzclaw / openclaw,默认 hermes
set -e

BOT_ID="${1:-hermes-linux-qiuming}"
AGENT_TYPE="${2:-${AGENT_TYPE:-hermes}}"   # 安装类型: hermes / 12fzclaw / openclaw
REG_CODE="dev-LQztUhv3ASEz"          # 预生成的注册码(已登记在服务器,一次性)
WS_HOST="ai.12fz.com"
USER_ID="1"
BRIDGE_DIR="$HOME/12fz-bridge"
HERMES_HOME="${HERMES_HOME:-$HOME/.hermes}"
CONFIG="$HOME/.hermes/12fz-bridge.json"

echo "==> 设备: $BOT_ID  注册码: $REG_CODE"
mkdir -p "$BRIDGE_DIR"

# 1) 注册设备 + 拿 sk-dev key
echo "==> 注册设备..."
REG=$(curl -s -X POST "https://$WS_HOST/api/devices/register" \
  -H "Content-Type: application/json" \
  -d "{\"name\":\"$BOT_ID\",\"device_key\":\"$REG_CODE\",\"os\":\"linux\",\"agent_type\":\"$AGENT_TYPE\"}")
echo "$REG"
DEV_TOKEN=$(echo "$REG" | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])" 2>/dev/null || true)
if [ -z "$DEV_TOKEN" ]; then
  echo "!! 注册失败,请检查注册码是否已被使用"
  exit 1
fi

echo "==> 获取 sk-dev key..."
SETUP=$(curl -s "https://$WS_HOST/api/devices/setup?token=$DEV_TOKEN")
echo "$SETUP"
SK_KEY=$(echo "$SETUP" | python3 -c "import sys,json; print(json.load(sys.stdin)['key'])" 2>/dev/null || true)
if [ -z "$SK_KEY" ]; then
  echo "!! setup 失败,响应: $SETUP"
  exit 1
fi

# 2) 写 bridge 配置
echo "==> 写 bridge 配置 $CONFIG"
cat > "$CONFIG" <<EOF
{
  "token": "$DEV_TOKEN",
  "bot_id": "$BOT_ID",
  "ws_host": "$WS_HOST",
  "user_id": "$USER_ID",
  "agent_type": "$AGENT_TYPE",
  "doc_dir": "$HOME/research"
}
EOF
chmod 600 "$CONFIG"

# 3) 配置 hermes 指向 12fz 代理
echo "==> 配置 hermes (config.yaml + .env)"
ENV_FILE="$HERMES_HOME/.env"
if ! grep -q "OPENAI_API_KEY" "$ENV_FILE" 2>/dev/null; then
  echo "OPENAI_API_KEY=$SK_KEY" >> "$ENV_FILE"
else
  sed -i "s|^OPENAI_API_KEY=.*|OPENAI_API_KEY=$SK_KEY|" "$ENV_FILE"
fi

CFG_FILE="$HERMES_HOME/config.yaml"
if [ -f "$CFG_FILE" ]; then
  python3 - "$CFG_FILE" "$SK_KEY" <<'PYEOF'
import sys, yaml
cfg_path, sk = sys.argv[1], sys.argv[2]
with open(cfg_path, encoding="utf-8") as f:
    cfg = yaml.safe_load(f) or {}
model = cfg.setdefault("model", {})
model["base_url"] = "https://ai.12fz.com/v1"
model["default"] = model.get("default", "deepseek-v4-flash")
if "provider" in model and str(model["provider"]).startswith("custom"):
    model["provider"] = "custom"
with open(cfg_path, "w", encoding="utf-8") as f:
    yaml.dump(cfg, f, default_flow_style=False, allow_unicode=True)
print("config.yaml updated: base_url=https://ai.12fz.com/v1")
PYEOF
else
  echo "!! 未找到 $CFG_FILE — 请先 hermes setup 初始化"
fi

# 4) 拷贝 bridge 脚本
echo "==> 拷贝 bridge.py -> $BRIDGE_DIR/hermes-bridge.py"
cp "$(dirname "$0")/hermes-bridge.py" "$BRIDGE_DIR/hermes-bridge.py" 2>/dev/null \
  || curl -fsSL -o "$BRIDGE_DIR/hermes-bridge.py" "https://raw.githubusercontent.com/qiu/ai-chat/master/static/hermes-bridge-v8.py"

# 5) 安装依赖
echo "==> 安装 Python 依赖..."
pip install --quiet websocket-client requests pyyaml 2>/dev/null \
  || pip3 install --quiet websocket-client requests pyyaml

# 6) 注册 systemd 服务
echo "==> 注册 systemd 服务..."
cat > /tmp/12fz-bridge.service <<EOF
[Unit]
Description=12FZ Hermes Bridge ($BOT_ID)
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$(whoami)
WorkingDirectory=$BRIDGE_DIR
ExecStart=$(which python3) $BRIDGE_DIR/hermes-bridge.py
Restart=always
RestartSec=5
Environment=PYTHONUNBUFFERED=1
EnvironmentFile=$CONFIG 2>/dev/null || true

[Install]
WantedBy=multi-user.target
EOF
# EnvironmentFile 对 json 无效,去掉
sed -i '/EnvironmentFile/d' /tmp/12fz-bridge.service
sudo cp /tmp/12fz-bridge.service /etc/systemd/system/12fz-bridge.service
sudo systemctl daemon-reload
sudo systemctl enable 12fz-bridge
sudo systemctl restart 12fz-bridge

echo ""
echo "=============================================="
echo "✅ 部署完成!"
echo "   设备: $BOT_ID"
echo "   服务: systemctl status 12fz-bridge"
echo "   日志: journalctl -u 12fz-bridge -f"
echo "   配置: $CONFIG"
echo "=============================================="

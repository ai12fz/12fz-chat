#!/bin/bash
# 12FZ Agent Bridge — 一键安装脚本
# curl -s https://ai.12fz.com/agent-bridge-install.sh | bash -s -- <agent_bot_id> <token>

set -e

BOT_ID="${1:-}"
TOKEN="${2:-}"
WS_HOST="${3:-ai.12fz.com}"

if [ -z "$BOT_ID" ] || [ -z "$TOKEN" ]; then
  echo "用法: curl -s https://ai.12fz.com/agent-bridge-install.sh | bash -s -- <agent_bot_id> <token>"
  echo ""
  echo "到 https://ai.12fz.com/chat/ → 管理 → Agent管理 → 新建Agent 获取"
  echo "创建时选择「API类型」，生成 token"
  exit 1
fi

echo "=== 12FZ Agent Bridge 安装 ==="
echo "Agent: ${BOT_ID}"
echo ""

# 1. 下载桥接脚本
mkdir -p ~/.12fz
curl -sk -o ~/.12fz/bridge.py https://ai.12fz.com/hermes-bridge.py
chmod +x ~/.12fz/bridge.py

# 2. 配置
mkdir -p ~/.hermes
cat > ~/.hermes/12fz-bridge.json << EOF
{"bot_id":"${BOT_ID}","token":"${TOKEN}","ws_host":"${WS_HOST}","user_id":"1"}
EOF
echo "✓ 配置完成"

# 3. 安装 systemd 服务（如果可用）
if command -v systemctl &>/dev/null && [ -d /etc/systemd/system ]; then
  sudo tee /etc/systemd/system/12fz-bridge.service << EOF > /dev/null
[Unit]
Description=12FZ Agent Bridge
After=network.target

[Service]
Type=simple
User=${USER}
ExecStart=/usr/bin/python3 -u ${HOME}/.12fz/bridge.py
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF
  sudo systemctl daemon-reload
  sudo systemctl enable 12fz-bridge
  sudo systemctl start 12fz-bridge
  echo "✓ systemd 服务已启动，开机自启"
else
  # fallback: nohup + crontab
  pkill -f "bridge.py" 2>/dev/null || true
  nohup python3 -u ~/.12fz/bridge.py > ~/.12fz/bridge.log 2>&1 &
  (crontab -l 2>/dev/null | grep -v bridge.py; echo "@reboot sleep 30 && nohup python3 -u ${HOME}/.12fz/bridge.py > ${HOME}/.12fz/bridge.log 2>&1 &") | crontab -
  echo "✓ 已启动（nohup + crontab 自启）"
fi

echo ""
echo "=== 安装完成 ==="
echo "Agent ${BOT_ID} 已连接"
echo "查看日志: tail -f ~/.12fz/bridge.log"

#!/bin/bash
# 12FZ 设备WebSocket连接安装脚本
# 用法: curl -s https://ai.12fz.com/install-device-ws.sh | bash -s -- --bot-id=<设备ID> --token=<设备Token>

set -e

BOT_ID=""
TOKEN=""
WS_HOST="ai.12fz.com"

while [[ $# -gt 0 ]]; do
  case $1 in
    --bot-id=*) BOT_ID="${1#*=}"; shift ;;
    --bot-id) BOT_ID="$2"; shift 2 ;;
    --token=*) TOKEN="${1#*=}"; shift ;;
    --token) TOKEN="$2"; shift 2 ;;
    *) echo "未知参数: $1"; exit 1 ;;
  esac
done

if [ -z "$BOT_ID" ] || [ -z "$TOKEN" ]; then
  echo "用法: curl -s https://ai.12fz.com/install-device-ws.sh | bash -s -- --bot-id=<设备ID> --token=<设备Token>"
  echo ""
  echo "到 https://ai.12fz.com/ → 设备管理 → 查看设备Token"
  exit 1
fi

echo "=== 12FZ 设备 WebSocket 连接安装 ==="
echo "设备ID: $BOT_ID"

# 安装依赖
if command -v pip3 &>/dev/null; then
  pip3 install websocket-client -q 2>/dev/null || true
elif command -v pip &>/dev/null; then
  pip install websocket-client -q 2>/dev/null || true
fi

# 创建配置目录
mkdir -p /etc/12fz-agent

# 写配置文件
cat > /etc/12fz-agent/device.json << EOF
{
  "bot_id": "$BOT_ID",
  "token": "$TOKEN",
  "host": "$WS_HOST"
}
EOF

# 写连接脚本
cat > /usr/local/bin/12fz-device-ws.py << 'PYEOF'
#!/usr/bin/env python3
"""12FZ Device WebSocket 持久连接"""
import json, threading, time, requests, os, signal, sys

CONFIG_FILE = "/etc/12fz-agent/device.json"

def load_config():
    with open(CONFIG_FILE) as f:
        return json.load(f)

def heartbeat_loop(cfg):
    while True:
        try:
            requests.post(f"https://{cfg['host']}/api/devices/heartbeat",
                headers={"Authorization": f"Bearer {cfg['token']}"}, timeout=5)
        except:
            pass
        time.sleep(30)

def ws_connect(cfg):
    import websocket
    def on_msg(ws, msg):
        try:
            d = json.loads(msg)
            if d.get("type") == "hello":
                print(f"[ws] connected as {cfg['bot_id']}")
        except:
            pass
    def on_error(ws, err):
        print(f"[ws] error: {err}", flush=True)
    def on_close(ws, *a):
        print(f"[ws] closed", flush=True)
    ws = websocket.WebSocketApp(
        f"wss://{cfg['host']}/ws?token={cfg['token']}",
        on_message=on_msg, on_error=on_error, on_close=on_close)
    ws.run_forever(ping_interval=30, ping_timeout=10)

def main():
    cfg = load_config()
    threading.Thread(target=heartbeat_loop, args=(cfg,), daemon=True).start()
    while True:
        try:
            ws_connect(cfg)
        except Exception as e:
            print(f"[ws] conn err: {e}", flush=True)
        time.sleep(5)

if __name__ == "__main__":
    main()
PYEOF

chmod +x /usr/local/bin/12fz-device-ws.py

# 写systemd服务
cat > /etc/systemd/system/12fz-device-ws.service << EOF
[Unit]
Description=12FZ Device WebSocket Connection
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
Group=root
ExecStart=/usr/bin/python3 /usr/local/bin/12fz-device-ws.py
Restart=always
RestartSec=10
StandardOutput=append:/var/log/12fz-device-ws.log
StandardError=append:/var/log/12fz-device-ws.log

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable 12fz-device-ws
systemctl start 12fz-device-ws

echo "✅ 安装完成！正在连接..."
sleep 3
systemctl status 12fz-device-ws --no-pager -l | head -5

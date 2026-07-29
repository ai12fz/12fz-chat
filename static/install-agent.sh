#!/bin/bash
# 12FZ API Agent 一键安装脚本
set -e

BOT_ID=""
TOKEN=""
API_URL="https://ai.12fz.com/v1"
CONFIG_DIR="$HOME/.12fz-agent"

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
  echo "用法: $0 --bot-id=xxx --token=xxx"
  exit 1
fi

echo "=== 12FZ Agent 安装 ==="
echo "Bot ID: $BOT_ID"

mkdir -p "$CONFIG_DIR"

# 写配置文件
cat > "$CONFIG_DIR/config.json" << EOF
{
  "bot_id": "$BOT_ID",
  "token": "$TOKEN",
  "api_url": "$API_URL"
}
EOF

echo "✅ 配置已写入 $CONFIG_DIR/config.json"
echo ""
echo "=== 使用方式 ==="
echo "直接调用 API："
echo "  curl -s $API_URL/chat/completions \\"
echo "    -H \"Authorization: Bearer $TOKEN\" \\"
echo "    -H \"Content-Type: application/json\" \\"
echo "    -d '{\"model\":\"deepseek-v4-pro\",\"messages\":[{\"role\":\"user\",\"content\":\"你好\"}]}'"
echo ""
echo "或在代码中配置 provider："
echo "  base_url: $API_URL"
echo "  api_key: $TOKEN"

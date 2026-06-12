#!/bin/bash
# Restart all bot processors for chat.12fz.com
# Usage: bash restart-bot-processors.sh

echo "=== Killing old processors ==="

# Kill on Aliyun
sshpass -p 'Cx99w06020354' ssh -o StrictHostKeyChecking=no root@8.138.235.183 'pkill -9 -f chat-bot-processor 2>/dev/null; sleep 2; echo "Aliyun cleared"'

# Kill on Vultr
pkill -9 -f chaogu-chat-processor 2>/dev/null
sleep 2
echo "Vultr cleared"

echo "=== Starting processors ==="

# Start on Aliyun: 服务器技术 + 高级工程师
sshpass -p 'Cx99w06020354' ssh -o StrictHostKeyChecking=no root@8.138.235.183 '
cd /root/.hermes
BOT_ID=服务器技术 nohup python3 /usr/local/bin/chat-bot-processor-v8.py > /tmp/bot-serv.log 2>&1 &
BOT_ID=高级工程师 nohup python3 /usr/local/bin/chat-bot-processor-v8.py > /tmp/bot-senior.log 2>&1 &
sleep 3
echo "Aliyun started"
'

# Start on Vultr: chaogu-ai
cd /root/.hermes
nohup python3 chaogu-chat-processor.py > /tmp/chaogu-chat.log 2>&1 &
sleep 3
echo "Vultr started"

echo "=== Verifying ==="
sleep 5
sshpass -p 'Cx99w06020354' ssh -o StrictHostKeyChecking=no root@8.138.235.183 'grep ws_online /tmp/bot-serv.log; grep ws_online /tmp/bot-senior.log'
grep ws_online /tmp/chaogu-chat.log
echo "=== All done ==="

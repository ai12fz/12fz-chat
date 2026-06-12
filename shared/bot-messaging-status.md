# Bot消息互通 — 实现方案

## 架构

```
用户在 chat.12fz.com 群发消息
  → Go Server 存DB + WS广播给所有在线客户端
  → Bot Processor (阿里云/v8) 收到消息
    → Hermes CLI 判断「相关/不相关」
      → 相关：AI生成回复 → REST API 发回群
      → 不相关：静默忽略
      → 如果是@其他bot：标记skipped（v8智能路由）
```

## 当前状态 (2026-06-12)

| Bot | 服务器 | 状态 | 备注 |
|-----|--------|------|------|
| 服务器技术 | 阿里云 | ✅ 正常运行 | v8 processor，回复服务器状态 |
| 高级工程师 | 阿里云 | ✅ 正常运行 | v8 processor，智能路由 |
| chaogu-ai | Vultr | ⚠️ 能连WS但无法回复 | Hermes CLI `-q`模式有signal handler bug |
| gong3 | 101机器 | ❓ 未验证 | 需SSH到内网检查 |

## 已知问题
1. **chaogu-ai Hermes CLI bug**: `hermes chat -q` 在单查询模式下触发 `KeyboardInterrupt`，导致无输出。
   症状：`_signal_handler_q` → `logger.debug()` 后 `raise KeyboardInterrupt()`。
   临时候选方案：改用非 `-q` 模式 + 解析输出。
2. **WS断连**: bot processor时不时断连重连，疑似ping/pong超时。
   已从v5升级到v8（后台线程处理消息，不阻塞WS事件循环）。
3. **Go服务密码不统一**: admin密码 `admin123` 和 `Cx99w06020354` 混用。
   systemd service 用 `admin123`，但旧文档写 `Cx99w06020354`。

## 重启命令
```bash
bash /root/12fz-chat/scripts/restart-bot-processors.sh
```

## 查看日志
```bash
# 阿里云
tail -20 /tmp/bot-serv.log     # 服务器技术
tail -20 /tmp/bot-senior.log   # 高级工程师

# Vultr
tail -20 /tmp/chaogu-chat.log  # chaogu-ai

# Go服务
journalctl -u chat-server --no-pager -n 50
```

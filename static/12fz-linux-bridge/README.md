# 12FZ Linux Hermes Bridge 部署说明

在另一台 Linux 机器上让 Hermes(最新版)接入 ai.12fz.com 聊天互通。

## 包内文件

- `hermes-bridge-v8.py` — 跨平台桥接脚本(Windows/Linux 自适应路径)
- `setup-12fz-bridge.sh` — Linux 一键部署脚本

## 前置条件(在 Linux 上)

1. 已安装最新版 Hermes:
   ```bash
   curl -fsSL https://hermes-agent.nousresearch.com/install.sh | bash
   ```
   安装后确认 `hermes` 可用(`~/.local/bin/hermes` 应在 PATH 中)。
   至少跑过一次 `hermes setup` 或 `hermes` 初始化出 `~/.hermes/config.yaml`。

2. 需要 `curl` 和 `python3` + pip(一般自带)。

## 部署步骤

把本目录整个拷到 Linux(如 `/root/12fz-bridge/` 或任意目录),然后:

```bash
cd 12fz-bridge
bash setup-12fz-bridge.sh                 # 用默认设备名 hermes-linux-qiuming
# 或指定名字:
bash setup-12fz-bridge.sh hermes-linux-02
```

脚本会自动完成:

1. 用预注册码 `dev-LQztUhv3ASEz` 注册设备(一次性的,只可用一次)
   → 拿到设备 token `d_xxx`
2. 调 `/api/devices/setup` 拿到 LLM key `sk-dev-xxx`
3. 写 `~/.hermes/12fz-bridge.json`(token / bot_id / ws_host / user_id)
4. 更新 `~/.hermes/.env` 的 `OPENAI_API_KEY=sk-dev-xxx`
5. 更新 `~/.hermes/config.yaml`:`model.base_url = https://ai.12fz.com/v1`
6. 安装依赖(websocket-client / requests / pyyaml)
7. 注册 systemd 服务 `12fz-bridge.service` 并启动

## 验证

```bash
systemctl status 12fz-bridge          # active (running)
journalctl -u 12fz-bridge -f          # 看日志
```

日志里看到 `connected as hermes-linux-qiuming` 即成功。
之后在 ai.12fz.com 管理面板里应该能看到新设备,并向它发消息测试。

## 手动测试(不装 systemd)

```bash
python3 hermes-bridge-v8.py
```

## 常见问题

- **注册失败 invalid registration code**:注册码是一次性的,若已被使用,
  需在管理面板生成新注册码,改脚本里 `REG_CODE` 变量后重试。
- **hermes: command not found**:重新登录 shell,或把
  `export PATH=$PATH:$HOME/.local/bin` 加入 `~/.bashrc`。
- **改模型**:在 ai.12fz.com 管理面板改设备模型,bridge 每 55s 自动同步
  到 `~/.hermes/config.yaml`,无需手动改。

## 配置说明

`~/.hermes/12fz-bridge.json`:

```json
{
  "token": "d_...",
  "bot_id": "hermes-linux-qiuming",
  "ws_host": "ai.12fz.com",
  "user_id": "1",
  "doc_dir": "/root/research"
}
```

- `doc_dir`:hermes 生成的文件(截图/文档)放这里会自动上传,回复里附 `[doc:id]` 卡片
- `user_id`:默认 1(管理员)。回复固定发给这个用户。

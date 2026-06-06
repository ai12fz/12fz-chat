#!/usr/bin/env python3
"""
chat-bot-processor.py v6 — Intelligent response selection per bot, w/ name mention
Each bot evaluates message relevance and only replies when relevant.
v6: Added all bot roles, mention detection, proper systemd support.
"""
import json, os, sys, time, subprocess, urllib.request, re, signal

CHAT_HOST = "chat.12fz.com"
BOT_ID = os.environ.get("BOT_ID", "unknown")
HTTP_PROTO = "https"
WS_PROTO = "wss"
HERMES_HOME = os.path.expanduser("~/.hermes")

BOT_TOKENS = {
    "服务器技术": "chat-token-server-2026",
    "高级工程师": "chat-token-senior-2026",
    "chaogu-ai": "chat-token-chaogu-2026",
    "gong3": "chat-token-gong3-2026",
}
BOT_TOKEN = BOT_TOKENS.get(BOT_ID, "")
WS_URL = f"{WS_PROTO}://{CHAT_HOST}/ws?token={BOT_TOKEN}"

# Role profiles — determines what each bot is responsible for
BOT_ROLES = {
    "服务器技术": "服务器运维（Nginx / MySQL / Redis / PHP）、阿里云管理、WordPress插件开发、站点部署配置、SSL证书、基础设施故障排查、Linux系统问题、数据库备份恢复、监控告警、网络配置。也参与Go语言开发和数据库迁移。",
    "高级工程师": "Go语言开发、数据库迁移（Oracle→PostgreSQL）、12FZ聊天系统开发、代码审核、API接口开发、架构设计、bug修复、全量重构、编程技术方案讨论。",
    "chaogu-ai": "AI集成、代码审核、全量重构协调、技术方案评审、Go聊天系统开发参与、技术文档沉淀、团队协作协调、飞书消息处理、多Agent调度。",
    "gong3": "前端Vue3开发、Go聊天系统WebSocket开发、SSO用户系统开发、UI界面设计和实现、系统架构讨论。",
}

running = True

def signal_handler(sig, frame):
    global running
    print(f"[bot] shutting down...", flush=True)
    running = False

signal.signal(signal.SIGTERM, signal_handler)
signal.signal(signal.SIGINT, signal_handler)

def api_post(path, data):
    url = f"{HTTP_PROTO}://{CHAT_HOST}{path}"
    req = urllib.request.Request(url, data=json.dumps(data).encode(),
        headers={"Content-Type":"application/json","Authorization":f"Bearer {BOT_TOKEN}"})
    try:
        resp = urllib.request.urlopen(req, timeout=10)
        return json.loads(resp.read())
    except Exception as e:
        print(f"[bot] api_err: {e}", flush=True)
        return None

def extract_reply(text):
    """Extract reply from Hermes output.
    Format expected:
      相关
      @reply text here...
    Or:
      不相关
    """
    lines = text.split("\n")
    
    # Find avatar
    avatar_idx = -1
    for i, line in enumerate(lines):
        if "⚕" in line and "hermes" in line.lower():
            avatar_idx = i
            break
    if avatar_idx < 0:
        return None
    
    # Collect text lines inside the box
    content_lines = []
    for line in lines[avatar_idx+1:]:
        s = line.strip()
        # Detect closing border
        if s and re.match(r'^[\u2500-\u257f\s]+$', s) and len(s) > 15:
            break
        if not s or s.startswith("⚠") or '┊' in s or s.startswith('<'):
            continue
        if any(x in s for x in ['Iteration', 'Initializing', 'Duration:', 'Messages:', 'Session:', 'Resume', 'hermes --resume']):
            continue
        if 'asking model' in s.lower() or ('summary' in s.lower() and 'requesting' in s.lower()):
            continue
        content_lines.append(s)
    
    if not content_lines:
        return None
    
    first = content_lines[0].strip()
    
    if "不相关" in first or "IGNORE" in first.upper():
        return None
    
    if "相关" in first:
        reply_lines = content_lines[1:]
        reply = " ".join(line for line in reply_lines if line.strip()).strip()
        if reply:
            return reply
        return None
    
    # No prefix - treat entire content as reply
    reply = " ".join(content_lines).strip()
    if reply and "不相关" not in reply and "IGNORE" not in reply.upper()[:10]:
        return reply
    return None

def generate_reply(content, force_reply=False):
    """Use hermes CLI to decide relevance + generate reply"""
    role_desc = BOT_ROLES.get(BOT_ID, "")
    env = {**os.environ, "HERMES_HOME": HERMES_HOME}
    try:
        if force_reply:
            prompt = (
                f"你是{BOT_ID}，职责范围：{role_desc}\n\n"
                f"用户提到了你（@了你），消息：「{content[:400]}」\n\n"
                f"请直接回复（1-2句话），以@开头。\n"
                f"不要写「相关」或「不相关」，直接回复内容。"
            )
        else:
            prompt = (
                f"你是{BOT_ID}，职责范围：{role_desc}\n\n"
                f"用户说：「{content[:400]}」\n\n"
                f"回复格式（严格执行）：\n"
                f"第一行写「相关」或「不相关」\n"
                f"如果相关：第二行开始以@开头简短回复（1-2句话），不用工具\n"
                f"如果不相关：只写「不相关」，不要写其他内容"
            )
        result = subprocess.run(
            ["hermes", "chat", "-q", prompt,
             "--model", "deepseek-v4-flash", "--provider", "deepseek",
             "-t", "", "--max-turns", "1"],
            capture_output=True, text=True, timeout=30, env=env
        )
        out = result.stdout
        reply = extract_reply(out)
        return reply
    except subprocess.TimeoutExpired:
        return None
    except Exception as e:
        print(f"[bot] hermes_err: {e}", flush=True)
        return None

def on_message(ws, raw):
    try:
        msg = json.loads(raw)
        if msg.get("type") != "message":
            return
        d = msg.get("data", {})
        s = d.get("sender_id", "")
        mid = d.get("id", 0)
        gid = d.get("group_id", 0)
        content = d.get("content", "")

        # Don't respond to own messages or other bots' messages (prevents loops)
        if s == BOT_ID or mid in last_msg_ids:
            return
        # Quick check: bot IDs are simple Chinese/English names
        if s in BOT_TOKENS:
            return
        last_msg_ids.add(mid)
        if len(last_msg_ids) > 500:
            last_msg_ids.clear()

        # 如果消息里@了别的bot，不回复（避免抢答）
        for other_id in BOT_TOKENS:
            if other_id != BOT_ID and (f"@{other_id}" in content or other_id in content):
                print(f"[bot] G{gid} {s}: skipped (@{other_id} is the target)", flush=True)
                return

        # Check if this bot is explicitly mentioned
        mentioned = f"@{BOT_ID}" in content or BOT_ID in content

        print(f"[bot] G{gid} {s}: {content[:80]}{' [MENTIONED]' if mentioned else ''}", flush=True)
        reply = generate_reply(content, force_reply=mentioned)
        if reply:
            result = api_post("/api/messages", {"group_id": gid, "content": reply})
            print(f"[bot] replied: {reply[:80]}", flush=True)
        else:
            print(f"[bot] ignored (not relevant)", flush=True)
    except Exception as e:
        print(f"[bot] err: {e}", flush=True)

last_msg_ids = set()

def run():
    global running
    print(f"[bot] {BOT_ID} starting v6 (all bots + mention)...", flush=True)
    while running:
        try:
            ws_app = __import__("websocket").WebSocketApp(WS_URL, on_message=on_message,
                on_error=lambda w,e: print(f"[bot] ws_err: {e}", flush=True),
                on_close=lambda w,*a: print("[bot] ws_closed", flush=True),
                on_open=lambda w: print(f"[bot] ws_online as {BOT_ID}", flush=True))
            ws_app.run_forever(ping_interval=30, ping_timeout=10)
        except Exception as e:
            print(f"[bot] conn_err: {e}", flush=True)
        if running:
            time.sleep(5)

if __name__ == "__main__":
    run()

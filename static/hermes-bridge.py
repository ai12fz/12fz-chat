#!/usr/bin/env python3
"""12FZ Hermes Bridge — 连接 WebSocket，收发消息"""
import json, time, struct, threading, subprocess, os, sys, signal

CONFIG_PATH = os.path.expanduser("~/.hermes/12fz-bridge.json")

def load_config():
    if not os.path.exists(CONFIG_PATH):
        print(f"请先配置 {CONFIG_PATH}:")
        print('{"bot_id":"xxx","token":"d_xxx","ws_url":"wss://ai.12fz.com/ws"}')
        sys.exit(1)
    with open(CONFIG_PATH) as f:
        return json.load(f)

cfg = load_config()
TOKEN = cfg["token"]
BOT_ID = cfg["bot_id"]
WS_URL = cfg.get("ws_url", "wss://ai.12fz.com/ws")

import socket, ssl

def connect():
    """连接 WebSocket"""
    host = WS_URL.replace("wss://","").replace("ws://","").rstrip("/")
    port = 443 if WS_URL.startswith("wss") else 80
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(30)
    if WS_URL.startswith("wss"):
        ctx = ssl.create_default_context()
        sock = ctx.wrap_socket(sock, server_hostname=host)
    sock.connect((host, port))
    sock.send(f"GET /ws?token={TOKEN} HTTP/1.1\r\nHost: {host}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: x3JJHMbDL1EzLkh9GBhXDw==\r\nSec-WebSocket-Version: 13\r\n\r\n".encode())
    resp = sock.recv(4096)
    if b"101" not in resp:
        raise Exception(f"WebSocket handshake failed: {resp[:200]}")
    return sock

def ws_send(ws, text):
    """发送 masked WebSocket 文本帧"""
    data = text.encode()
    l = len(data)
    if l < 126:
        header = bytes([0x81, 0x80 | l])
    elif l < 65536:
        header = bytes([0x81, 0x80 | 126]) + struct.pack(">H", l)
    else:
        header = bytes([0x81, 0x80 | 127]) + struct.pack(">Q", l)
    mask = os.urandom(4)
    masked = bytearray(data)
    for i in range(l):
        masked[i] ^= mask[i % 4]
    ws.send(header + mask + bytes(masked))

def ws_recv(ws):
    """接收 WebSocket 帧，返回文本"""
    h = ws.recv(2)
    if len(h) < 2:
        raise EOFError()
    op = h[0] & 0x0F
    if op == 8:  # close
        return None
    if op == 9:  # ping
        ws_send_pong(ws, h[1:])
        return ws_recv(ws)
    masked = h[1] & 0x80
    plen = h[1] & 0x7F
    if plen == 126:
        plen = struct.unpack(">H", ws.recv(2))[0]
    elif plen == 127:
        plen = struct.unpack(">Q", ws.recv(8))[0]
    if masked:
        mask_bytes = ws.recv(4)
    data = bytearray()
    while len(data) < plen:
        chunk = ws.recv(min(plen - len(data), 4096))
        if not chunk:
            raise EOFError()
        data.extend(chunk)
    if masked:
        for i in range(len(data)):
            data[i] ^= mask_bytes[i % 4]
    return data.decode(errors="replace")

def ws_send_pong(ws, payload):
    ws.send(bytes([0x8A, len(payload)]) + payload)

def send_agent_status(ws, phase, tool="", detail="{}"):
    """发送工具执行状态"""
    msg = json.dumps({
        "type": "agent_status",
        "data": {
            "from": BOT_ID,
            "phase": phase,
            "tool": tool,
            "detail": detail
        }
    })
    try:
        ws_send(ws, msg)
    except:
        pass

# ── Heartbeat ──
def heartbeat_loop(ws_ref):
    while True:
        time.sleep(30)
        try:
            if ws_ref[0]:
                ws_send(ws_ref[0], json.dumps({"type":"ping"}))
        except:
            pass

# ── 消息处理 ──
def process_message(ws, text):
    """用 Hermes 处理消息并回复"""
    send_agent_status(ws, "tool_start", "hermes_reply", json.dumps({"msg": text[:100]}))
    try:
        result = subprocess.run(
            ["hermes", "chat", "-q", text, "--quiet", "--no-banner"],
            capture_output=True, text=True, timeout=300,
            env={**os.environ, "HERMES_YOLO_MODE": "1"}
        )
        reply = result.stdout.strip()
        if not reply:
            reply = result.stderr.strip()[:500] or "(Hermes 无输出)"
    except subprocess.TimeoutExpired:
        reply = "⏰ Hermes 处理超时"
    except FileNotFoundError:
        reply = "❌ 未安装 Hermes Agent，请运行: curl -fsSL https://raw.githubusercontent.com/NousResearch/hermes-agent/main/scripts/install.sh | bash"
    except Exception as e:
        reply = f"❌ {e}"

    send_agent_status(ws, "tool_end", "hermes_reply", json.dumps({"reply": reply[:100]}))

    # 发回复
    rmsg = json.dumps({
        "type": "message",
        "data": {
            "from": BOT_ID,
            "to": cfg.get("user_id", "1"),
            "content": reply
        }
    })
    try:
        ws_send(ws, rmsg)
    except:
        pass

# ── 主循环 ──
ws = [None]  # mutable ref for heartbeat thread

def main():
    ws[0] = connect()
    print(f"[bridge] connected as {BOT_ID}")
    threading.Thread(target=heartbeat_loop, args=(ws,), daemon=True).start()
    
    while True:
        try:
            raw = ws_recv(ws[0])
            if raw is None:
                raise EOFError("close frame")
            msg = json.loads(raw)
            if msg.get("type") == "message":
                data = msg.get("data", {})
                content = data.get("content", "")
                if content:
                    print(f"[bridge] 收到: {content[:80]}")
                    process_message(ws[0], content)
            elif msg.get("type") == "event":
                evt = msg.get("data", {}).get("event", "")
                print(f"[bridge] 事件: {evt}")
        except (EOFError, ConnectionError, OSError) as e:
            print(f"[bridge] 断开: {e}, 3s 后重连...")
            time.sleep(3)
            try:
                ws[0] = connect()
                print(f"[bridge] 重连成功")
            except Exception as e2:
                print(f"[bridge] 重连失败: {e2}")
                time.sleep(5)
        except json.JSONDecodeError:
            pass

if __name__ == "__main__":
    main()

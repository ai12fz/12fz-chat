#!/usr/bin/env python3
"""12FZ Hermes Bridge — WebSocket ↔ Hermes 消息桥接"""
import json, time, struct, threading, subprocess, os, sys

CONFIG_PATH = os.path.expanduser("~/.hermes/12fz-bridge.json")

def load_config():
    if not os.path.exists(CONFIG_PATH):
        print(f"需要配置文件 {CONFIG_PATH}")
        sys.exit(1)
    with open(CONFIG_PATH) as f:
        return json.load(f)

cfg = load_config()
TOKEN = cfg["token"]
BOT_ID = cfg["bot_id"]
WS_HOST = cfg.get("ws_host", "ai.12fz.com")

import socket, ssl

def connect():
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    sock.settimeout(30)
    ctx = ssl.create_default_context()
    ssock = ctx.wrap_socket(sock, server_hostname=WS_HOST)
    ssock.connect((WS_HOST, 443))
    req = f"GET /ws?token={TOKEN} HTTP/1.1\r\nHost: {WS_HOST}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: x3JJHMbDL1EzLkh9GBhXDw==\r\nSec-WebSocket-Version: 13\r\n\r\n"
    ssock.send(req.encode())
    resp = ssock.recv(4096)
    if b"101" not in resp:
        raise Exception(f"WS handshake failed: {resp[:200]}")
    return ssock

def ws_send(ws, text):
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
    h = ws.recv(2)
    if len(h) < 2:
        raise EOFError()
    op = h[0] & 0x0F
    if op == 8:
        return None
    if op == 9:
        ws.send(bytes([0x8A, h[1] & 0x7F]) + ws.recv(h[1] & 0x7F))
        return ws_recv(ws)
    plen = h[1] & 0x7F
    if plen == 126:
        plen = struct.unpack(">H", ws.recv(2))[0]
    elif plen == 127:
        plen = struct.unpack(">Q", ws.recv(8))[0]
    mask_bytes = ws.recv(4)
    data = bytearray()
    while len(data) < plen:
        chunk = ws.recv(min(plen - len(data), 4096))
        if not chunk:
            raise EOFError()
        data.extend(chunk)
    for i in range(len(data)):
        data[i] ^= mask_bytes[i % 4]
    return data.decode(errors="replace")

def hb_loop(ws_ref):
    while True:
        time.sleep(30)
        try:
            if ws_ref[0]:
                ws_send(ws_ref[0], json.dumps({"type":"ping"}))
        except:
            pass

def process_message(ws, text):
    try:
        result = subprocess.run(
            ["hermes", "chat", "-q", text, "--quiet"],
            capture_output=True, text=True, timeout=300,
            env={**os.environ, "HERMES_YOLO_MODE": "1"}
        )
        reply = result.stdout.strip() or result.stderr.strip()[:500] or "(空)"
    except subprocess.TimeoutExpired:
        reply = "⏰ 超时"
    except FileNotFoundError:
        reply = "❌ 未安装 hermes"
    except Exception as e:
        reply = f"❌ {e}"

    rmsg = json.dumps({"type":"message","data":{"from":BOT_ID,"to":cfg.get("user_id","1"),"content":reply}})
    try:
        ws_send(ws, rmsg)
    except:
        pass

ws = [None]

def main():
    for i in range(10):
        try:
            ws[0] = connect()
            print(f"[bridge] connected as {BOT_ID}")
            break
        except Exception as e:
            print(f"[bridge] connect {i+1}/10: {e}")
            time.sleep(3)
    else:
        print("[bridge] all connect attempts failed")
        sys.exit(1)

    threading.Thread(target=hb_loop, args=(ws,), daemon=True).start()
    while True:
        try:
            raw = ws_recv(ws[0])
            if raw is None:
                raise EOFError("close")
            msg = json.loads(raw)
            if msg.get("type") == "message":
                content = msg.get("data", {}).get("content", "")
                if content:
                    print(f"[bridge] msg: {content[:80]}")
                    process_message(ws[0], content)
        except (EOFError, ConnectionError, OSError) as e:
            print(f"[bridge] disconnect: {e}, retrying...")
            time.sleep(3)
            try:
                ws[0] = connect()
                print("[bridge] reconnected")
            except Exception as e2:
                print(f"[bridge] reconnect failed: {e2}")
                time.sleep(5)

if __name__ == "__main__":
    main()

#!/usr/bin/env python3
"""12FZ Hermes Bridge v7 — streaming STDOUT parser + capability IDs"""
import json, threading, subprocess, os, sys, time, requests, websocket, re

CONFIG_PATH = os.path.expanduser("~/.hermes/12fz-bridge.json")
cfg = json.load(open(CONFIG_PATH))
TOKEN = cfg["token"]
BOT_ID = cfg["bot_id"]
WS_HOST = cfg.get("ws_host", "ai.12fz.com")
USER_ID = cfg.get("user_id", "1")

API_BASE = "https://%s" % WS_HOST
HEADERS = {"Authorization": "Bearer " + TOKEN, "Content-Type": "application/json"}

# ── capability map: name→{id,icon,desc} ──
CAP_MAP = {}

def load_capabilities():
    global CAP_MAP
    try:
        r = requests.get(API_BASE + "/api/capabilities", headers=HEADERS, timeout=10)
        if r.status_code == 200:
            for c in r.json():
                CAP_MAP[c["name"]] = {"id": c["id"], "icon": c["icon"], "desc": c["description"]}
            print("[bridge] loaded %d capabilities" % len(CAP_MAP), flush=True)
        else:
            print("[bridge] capabilities http %d" % r.status_code, flush=True)
    except Exception as e:
        print("[bridge] capabilities err: %s" % e, flush=True)

# ── tool detection patterns ──
TOOL_PATTERNS = [
    (r"preparing terminal",   "terminal"),
    (r"\$ .+",                "terminal"),     # shell command
    (r"reading file",         "read_file"),
    (r"writing file",         "write_file"),
    (r"patching file",        "write_file"),
    (r"searching",            "web_search"),
    (r"fetching",             "web_search"),
    (r"browsing",             "browser"),
    (r"running code",         "code_exec"),
    (r"memorizing",           "memory"),
    (r"preparing (desktop|computer)", "computer_use"),
]

def detect_tool(line):
    """Detect tool name from hermes stdout line"""
    line = line.strip()
    for pattern, tool in TOOL_PATTERNS:
        if re.search(pattern, line, re.IGNORECASE):
            # extract command detail for terminal
            if tool == "terminal":
                m = re.search(r"\$ (.+)  [0-9.]+s", line)
                if m:
                    return tool, m.group(1)[:60]
            return tool, ""
    return None, None

def load_env():
    if "HERMES_CUSTOM_TK_12FZ_COM_API_KEY" in os.environ:
        return
    for p in [
        os.path.expanduser("~/AppData/Local/hermes/.env"),
        "C:\\Users\\Administrator\\AppData\\Local\\hermes\\.env",
    ]:
        if os.path.exists(p):
            with open(p, encoding="utf-8") as f:
                for line in f:
                    line = line.strip()
                    if "=" in line and not line.startswith("#"):
                        k, v = line.split("=", 1)
                        os.environ[k.strip()] = v.strip().strip("'\"")
            break

def send_ws(ws, msg):
    try:
        ws.send(json.dumps(msg))
    except:
        pass

def on_message(ws, raw):
    try:
        msg = json.loads(raw)
    except json.JSONDecodeError:
        return
    if msg.get("type") != "message":
        return
    content = msg.get("data", {}).get("content", "")
    if not content:
        return
    print("[bridge] msg: %s" % content[:80], flush=True)

    # ── streaming hermes subprocess ──
    def agent_status(p, t, d):
        send_ws(ws, {"type":"agent_status","data":{"p":p,"t":t,"d":d}})
        print("[bridge] status %s c=%s %s" % (p, t, d[:30] if d else ""), flush=True)

    agent_status("s", 0, "处理中...")

    # WS keepalive thread
    stop_ping = threading.Event()
    def ping_loop():
        while not stop_ping.wait(20):
            send_ws(ws, {"type":"ping"})
    tp = threading.Thread(target=ping_loop, daemon=True)
    tp.start()

    reply = ""
    try:
        load_env()
        env = os.environ.copy()
        env["PYTHONIOENCODING"] = "utf-8"
        env["PYTHONUTF8"] = "1"
        env["LC_ALL"] = "C.UTF-8"

        proc = subprocess.Popen(
            [sys.executable, os.path.join(os.path.dirname(sys.executable), "..", "..", "hermes"),
             "chat", "-q", content],
            stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            env=env)
        reply_bytes = []

        # Stream stdout line by line
        last_tool = None
        for line_bytes in iter(proc.stdout.readline, b""):
            line = line_bytes.decode("utf-8", errors="replace")
            # Detect tool usage from "┊" lines
            if "┊" in line:
                tool, detail = detect_tool(line)
                if tool and tool in CAP_MAP:
                    cap = CAP_MAP[tool]
                    agent_status("s", cap["id"], detail or cap["desc"])
                    last_tool = tool
            reply_bytes.append(line_bytes)

        proc.wait(timeout=600)
        stop_ping.set()
        agent_status("d", 0, "")

        # Read reply (last non-empty section)
        full_out = b"".join(reply_bytes).decode("utf-8", errors="replace")
        # Extract response after the last tool section
        parts = full_out.split("╭─")
        reply = parts[-1].strip() if len(parts) > 1 else full_out.strip()
        if not reply:
            err = proc.stderr.read().decode("utf-8", errors="replace")[:500]
            reply = err if err else "（处理完成但未返回文本）"

    except subprocess.TimeoutExpired:
        proc.kill()
        stop_ping.set()
        reply = "处理超时，请重试"
        agent_status("d", 0, "")
    except FileNotFoundError:
        stop_ping.set()
        reply = "hermes not installed"
    except Exception as e:
        stop_ping.set()
        reply = "err: %s" % e

    try:
        resp = requests.post(
            API_BASE + "/api/friend-messages",
            headers=HEADERS,
            json={"friend_id": USER_ID, "content": reply},
            timeout=15)
        print("[bridge] reply http %d" % resp.status_code, flush=True)
    except Exception as e:
        print("[bridge] http err: %s" % e, flush=True)

def on_error(ws, err):
    print("[bridge] ws error: %s" % err, flush=True)
def on_close(ws, *a):
    print("[bridge] ws closed", flush=True)
def on_open(ws):
    print("[bridge] connected as %s" % BOT_ID, flush=True)
    # Load capabilities on connect
    threading.Thread(target=load_capabilities, daemon=True).start()

def hb_loop():
    while True:
        time.sleep(55)
        try:
            requests.post(API_BASE + "/api/devices/heartbeat", headers=HEADERS, timeout=10)
            # ── sync model config from server ──
            sync_model_config()
        except Exception:
            pass

def sync_model_config():
    """Fetch device model config and update hermes config.yaml if changed"""
    import yaml
    try:
        r = requests.get(API_BASE + "/api/devices/" + BOT_ID + "/model", headers=HEADERS, timeout=10)
        if r.status_code != 200:
            return
        cfg = r.json()
        model_name = cfg.get("model_name", "deepseek-v4-flash")
        model_provider = cfg.get("model_provider", "custom:deepseek-v4-flash(12fz)")

        config_path = os.path.expanduser("~/AppData/Local/hermes/config.yaml")
        if not os.path.exists(config_path):
            return

        with open(config_path, "r", encoding="utf-8") as f:
            hermes_cfg = yaml.safe_load(f)

        current_model = hermes_cfg.get("model", {}).get("default", "")
        current_provider = hermes_cfg.get("model", {}).get("provider", "")

        if current_model != model_name or current_provider != model_provider:
            if "model" not in hermes_cfg:
                hermes_cfg["model"] = {}
            hermes_cfg["model"]["default"] = model_name
            hermes_cfg["model"]["provider"] = model_provider
            with open(config_path, "w", encoding="utf-8") as f:
                yaml.dump(hermes_cfg, f, default_flow_style=False, allow_unicode=True)
            print("[bridge] model changed: %s (%s)" % (model_name, model_provider), flush=True)
    except Exception as e:
        print("[bridge] model sync err: %s" % e, flush=True)

def main():
    threading.Thread(target=hb_loop, daemon=True).start()
    url = "wss://%s/ws?token=%s" % (WS_HOST, TOKEN)
    while True:
        ws = websocket.WebSocketApp(
            url, on_message=on_message, on_error=on_error,
            on_close=on_close, on_open=on_open)
        ws.run_forever(ping_interval=30, ping_timeout=10)
        print("[bridge] reconnecting...", flush=True)
        time.sleep(3)

if __name__ == "__main__":
    main()

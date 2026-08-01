#!/usr/bin/env python3
"""12FZ Hermes Bridge v9 — multi-agent (one device, many agents, per-agent Hermes profile)
Each agent in cfg["agents"] gets its own WS connection (own token), its own Hermes
profile (HERMES_PROFILE env → isolated memories/skills/config.yaml), its own
heartbeat to /api/agents/{bot_id}/heartbeat, and model sync from /api/agents/{bot_id}.
Backwards compatible: a flat cfg with bot_id/token is treated as a single agent.
"""
import json, threading, subprocess, os, sys, time, shutil, requests, websocket, re, platform, socket

def find_config():
    """Locate 12fz-bridge.json. When run under SYSTEM (schtasks watchdog) expanduser('~')
    points at systemprofile, so fall back to the real user profile."""
    cands = [
        os.environ.get("12FZ_BRIDGE_CONFIG"),
        os.path.expanduser("~/.hermes/12fz-bridge.json"),
        os.path.join(os.environ.get("USERPROFILE", ""), ".hermes", "12fz-bridge.json"),
        r"C:\Users\Administrator\.hermes\12fz-bridge.json",
        "/root/.hermes/12fz-bridge.json",
    ]
    for c in cands:
        if c and os.path.exists(c):
            return c
    raise FileNotFoundError("12fz-bridge.json not found in: %s" % cands)

CONFIG_PATH = find_config()
cfg = json.load(open(CONFIG_PATH, encoding="utf-8"))
WS_HOST = cfg.get("ws_host", "ai.12fz.com")
USER_ID = cfg.get("user_id", "1")
DEVICE_TOKEN = cfg.get("token", "")
DEVICE_ID = cfg.get("bot_id", "")

# ── agents: explicit list, or single-agent fallback from flat config ──
if isinstance(cfg.get("agents"), list) and cfg["agents"]:
    AGENTS = cfg["agents"]
else:
    AGENTS = [{"bot_id": DEVICE_ID, "token": DEVICE_TOKEN,
               "profile": cfg.get("profile", ""), "display_name": cfg.get("display_name", DEVICE_ID)}]

API_BASE = "https://%s" % WS_HOST
IS_WIN = platform.system() == "Windows"

# ── path resolution ──
def hermes_bin():
    """Locate the hermes CLI executable."""
    if cfg.get("hermes_bin"):
        return cfg["hermes_bin"]
    if IS_WIN:
        return os.path.abspath(os.path.join(os.path.dirname(sys.executable), "..", "..", "hermes"))
    # Linux/macOS: prefer HERMES_BIN env, then PATH, then ~/.local/bin/hermes
    for cand in [os.environ.get("HERMES_BIN"), shutil.which("hermes"),
                 os.path.expanduser("~/.local/bin/hermes"),
                 os.path.expanduser("~/.hermes/hermes-agent/hermes")]:
        if cand and os.path.exists(cand):
            return cand
    return "hermes"  # last resort: rely on PATH

HERMES_BIN = hermes_bin()

def hermes_home():
    """Hermes data dir: $HERMES_HOME on Linux, AppData on Windows."""
    if os.environ.get("HERMES_HOME"):
        return os.environ["HERMES_HOME"]
    if IS_WIN:
        # SYSTEM (schtasks) has no user profile — fall back to real user
        for base in [os.path.expanduser("~"), os.environ.get("USERPROFILE", ""), r"C:\Users\Administrator"]:
            if base and os.path.isdir(os.path.join(base, "AppData", "Local", "hermes")):
                return os.path.join(base, "AppData", "Local", "hermes")
        return os.path.expanduser("~/AppData/Local/hermes")
    return os.path.expanduser("~/.hermes")

HERMES_HOME = hermes_home()

def agent_headers(token):
    return {"Authorization": "Bearer " + token, "Content-Type": "application/json"}

# ── document delivery (outbox dir → /api/documents) ──
DOC_DIR = cfg.get("doc_dir", os.path.expanduser("~/research") if not IS_WIN
                  else r"C:\Users\Administrator\research")
DOC_EXT = {".md", ".pdf", ".docx", ".xlsx", ".pptx", ".txt", ".csv", ".zip", ".gz", ".png", ".jpg", ".jpeg", ".webp"}
DOC_STATE = os.path.join(HERMES_HOME, "docs_uploaded.json")

def _load_doc_state():
    try:
        with open(DOC_STATE, encoding="utf-8") as f:
            return json.load(f)
    except Exception:
        return []

def _save_doc_state(state):
    try:
        with open(DOC_STATE, "w", encoding="utf-8") as f:
            json.dump(state, f, ensure_ascii=False)
    except Exception as e:
        print("[bridge] doc state save err: %s" % e, flush=True)

def upload_new_docs(reply, token):
    """Upload new files from DOC_DIR to /api/documents; append [doc:id] markers."""
    try:
        if not os.path.isdir(DOC_DIR):
            return reply
        state = set(_load_doc_state())
        markers = []
        for name in sorted(os.listdir(DOC_DIR)):
            path = os.path.join(DOC_DIR, name)
            if not os.path.isfile(path):
                continue
            ext = os.path.splitext(name)[1].lower()
            if ext not in DOC_EXT:
                continue
            st = os.stat(path)
            key = "%s|%d|%d" % (path, st.st_size, int(st.st_mtime))
            if key in state:
                continue
            try:
                title = os.path.splitext(name)[0]
                with open(path, "rb") as fh:
                    r = requests.post(
                        API_BASE + "/api/documents",
                        headers={"Authorization": "Bearer " + token},
                        files={"file": (name, fh)},
                        data={"title": title, "user_id": USER_ID},
                        timeout=60)
                if r.status_code in (200, 201):
                    doc = r.json()
                    markers.append("[doc:%s] %s" % (doc.get("id"), doc.get("title") or name))
                    state.add(key)
                    print("[bridge] doc uploaded id=%s %s (%dB)" % (doc.get("id"), name, st.st_size), flush=True)
                else:
                    print("[bridge] doc upload %s -> HTTP %d" % (name, r.status_code), flush=True)
            except Exception as e:
                print("[bridge] doc upload err %s: %s" % (name, e), flush=True)
        if markers:
            _save_doc_state(list(state))
            return reply.rstrip() + "\n\n📎 文档已生成，可点击上方卡片下载：\n" + "\n".join(markers)
        return reply
    except Exception as e:
        print("[bridge] upload_new_docs err: %s" % e, flush=True)
        return reply

# ── capability map: name→{id,icon,desc} ──
CAP_MAP = {}

def load_capabilities(token):
    global CAP_MAP
    try:
        r = requests.get(API_BASE + "/api/capabilities", headers=agent_headers(token), timeout=10)
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
        os.path.join(HERMES_HOME, ".env"),
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

def ensure_profile(profile):
    """Ensure the Hermes profile directory exists; create it (cloning default) if missing."""
    if not profile:
        return
    prof_dir = os.path.join(HERMES_HOME, "profiles", profile)
    if os.path.isdir(prof_dir):
        return
    try:
        subprocess.run([HERMES_BIN, "profile", "create", profile, "--clone", "--no-alias"],
                       env=os.environ.copy(), capture_output=True, timeout=120)
        print("[bridge] profile created: %s" % profile, flush=True)
    except Exception as e:
        print("[bridge] profile create %s err: %s" % (profile, e), flush=True)

class AgentWorker:
    """One agent = one WS connection + one Hermes profile + own heartbeat/model sync."""

    def __init__(self, agent):
        self.bot_id = agent["bot_id"]
        self.token = agent["token"]
        self.profile = agent.get("profile") or self.bot_id
        self.display_name = agent.get("display_name") or self.bot_id
        self.headers = agent_headers(self.token)
        self.ws = None

    def on_message(self, ws, raw):
        try:
            msg = json.loads(raw)
        except json.JSONDecodeError:
            return
        if msg.get("type") != "message":
            return
        content = msg.get("data", {}).get("content", "")
        if not content:
            return
        print("[bridge:%s] msg: %s" % (self.bot_id, content[:80]), flush=True)

        # ── streaming hermes subprocess ──
        def agent_status(p, t, d):
            send_ws(ws, {"type": "agent_status", "data": {"p": p, "t": t, "d": d}})
            print("[bridge:%s] status %s c=%s %s" % (self.bot_id, p, t, d[:30] if d else ""), flush=True)

        agent_status("s", 0, "处理中...")

        # WS keepalive thread
        stop_ping = threading.Event()
        def ping_loop():
            while not stop_ping.wait(20):
                send_ws(ws, {"type": "ping"})
        tp = threading.Thread(target=ping_loop, daemon=True)
        tp.start()

        reply = ""
        try:
            load_env()
            env = os.environ.copy()
            env["PYTHONIOENCODING"] = "utf-8"
            env["PYTHONUTF8"] = "1"
            env["LC_ALL"] = "C.UTF-8"
            if self.profile:
                env["HERMES_PROFILE"] = self.profile
            # sync model config before answering so per-agent model is honored
            try:
                self.sync_model_config()
            except Exception:
                pass

            proc = subprocess.Popen(
                [HERMES_BIN, "chat", "-q", content],
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

            # Read reply
            full_out = bytes().join(reply_bytes).decode("utf-8", errors="replace")
            # Strip ANSI escape codes and CR
            full_out = re.sub(r'\x1b(?:[@-Z\\-_]|\[[0-?]*[ -/]*[@-~])', '', full_out)
            full_out = full_out.replace('\r', '')

            # Extract AI response from box structure: split on ╭ (U+256d)
            parts = full_out.split('\u256d')
            reply = parts[-1].strip() if len(parts) > 1 else full_out.strip()
            # Split into lines, remove header/footer/debug
            lines = reply.split('\n')
            while lines and (not lines[0].strip() or
                             '\u256e' in lines[0] or  # ╮ header end
                             '\u255a' in lines[0]):   # ╚ box corner
                lines.pop(0)
            footers = {'Resume', 'hermes --resume', 'Session:', 'Duration:', 'Messages:'}
            while lines and (not lines[-1].strip() or
                             '\u2570' in lines[-1] or  # ╰ footer start
                             any(lines[-1].strip().startswith(f) for f in footers)):
                lines.pop()
            # Strip 4-space indent (box padding)
            reply = '\n'.join(l[4:] if l.startswith('    ') else l for l in lines).strip()
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
            reply = "hermes not installed (HERMES_BIN=%s)" % HERMES_BIN
        except Exception as e:
            stop_ping.set()
            reply = "err: %s" % e

        # Upload new deliverable files and append doc markers to the reply
        reply = upload_new_docs(reply, self.token)

        try:
            resp = requests.post(
                API_BASE + "/api/friend-messages",
                headers=self.headers,
                json={"friend_id": USER_ID, "content": reply},
                timeout=15)
            print("[bridge:%s] reply http %d" % (self.bot_id, resp.status_code), flush=True)
        except Exception as e:
            print("[bridge:%s] http err: %s" % (self.bot_id, e), flush=True)

    def on_error(self, ws, err):
        print("[bridge:%s] ws error: %s" % (self.bot_id, err), flush=True)
    def on_close(self, ws, *a):
        print("[bridge:%s] ws closed" % self.bot_id, flush=True)
    def on_open(self, ws):
        self.ws = ws
        print("[bridge:%s] connected as %s (profile=%s)" % (self.bot_id, self.bot_id, self.profile), flush=True)
        threading.Thread(target=load_capabilities, args=(self.token,), daemon=True).start()

    def get_local_ip(self):
        """Best-effort local IP (no external calls)."""
        try:
            s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
            s.connect(("8.8.8.8", 80))
            ip = s.getsockname()[0]
            s.close()
            return ip
        except Exception:
            try:
                return socket.gethostbyname(socket.gethostname())
            except Exception:
                return ""

    def heartbeat(self):
        """Device heartbeat + per-agent heartbeat, then model sync."""
        try:
            ip = self.get_local_ip()
            # per-agent heartbeat
            body = {"ip": ip} if ip else {}
            agent_type = cfg.get("agent_type", "hermes")
            if agent_type:
                body["agent_type"] = agent_type
            r = requests.post(API_BASE + "/api/agents/%s/heartbeat" % self.bot_id,
                              headers=self.headers, json=body, timeout=10)
            if r.status_code not in (200, 404):
                print("[bridge:%s] agent heartbeat http %d" % (self.bot_id, r.status_code), flush=True)
            self.sync_model_config()
        except Exception as e:
            print("[bridge:%s] heartbeat err: %s" % (self.bot_id, e), flush=True)

    def sync_model_config(self):
        """Fetch agent model config from server and update the agent's profile config.yaml."""
        import yaml
        try:
            r = requests.get(API_BASE + "/api/agents/%s" % self.bot_id, headers=self.headers, timeout=10)
            if r.status_code != 200:
                return
            ad = r.json()
            model_name = ad.get("model", "deepseek-v4-flash")
            model_provider = ad.get("model_provider", "") or ""
            if not model_provider:
                return

            if self.profile:
                config_path = os.path.join(HERMES_HOME, "profiles", self.profile, "config.yaml")
            else:
                config_path = os.path.join(HERMES_HOME, "config.yaml")
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
                print("[bridge:%s] model changed: %s (%s)" % (self.bot_id, model_name, model_provider), flush=True)
        except Exception as e:
            print("[bridge:%s] model sync err: %s" % (self.bot_id, e), flush=True)

    def hb_loop(self):
        while True:
            time.sleep(55)
            try:
                self.heartbeat()
            except Exception:
                pass

    def run(self):
        print("[bridge:%s] platform=%s hermes=%s home=%s" % (self.bot_id, platform.system(), HERMES_BIN, HERMES_HOME), flush=True)
        if self.profile:
            ensure_profile(self.profile)
        threading.Thread(target=self.hb_loop, daemon=True).start()
        url = "wss://%s/ws?token=%s" % (WS_HOST, self.token)
        while True:
            ws = websocket.WebSocketApp(
                url, on_message=self.on_message, on_error=self.on_error,
                on_close=self.on_close, on_open=self.on_open)
            self.ws = ws
            ws.run_forever(ping_interval=30, ping_timeout=10)
            print("[bridge:%s] reconnecting..." % self.bot_id, flush=True)
            time.sleep(3)

def main():
    print("[bridge] v9 multi-agent: %d agent(s), host=%s" % (len(AGENTS), WS_HOST), flush=True)
    threads = []
    for agent in AGENTS:
        t = threading.Thread(target=AgentWorker(agent).run, daemon=True)
        t.start()
        threads.append(t)
    while True:
        time.sleep(10)

if __name__ == "__main__":
    main()

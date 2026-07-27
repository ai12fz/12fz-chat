#!/bin/bash
P="$HOME/.12fzclaw"
[ ! -f "$P/device-token" ] && exit 1
echo "Starting 12fzclaw (push)..."
python3 -u > "$P/bot.log" 2>&1 << 'PYEOF'
#!/usr/bin/env python3
"""Minimal WebSocket client - no dependencies, push mode for 12fzclaw"""
import json,os,time,subprocess,socket,ssl,hashlib,base64,struct,threading,threading

P=os.path.expanduser("~/.12fzclaw")
T=open(P+"/device-token").read().strip()
H,N="ai.12fz.com",os.uname().nodename

def curl(method,path,data=None):
    cmd=["curl","-sk","-m30",f"https://{H}{path}","-H",f"Authorization: Bearer {T}"]
    if data: cmd+=["-H","Content-Type: application/json","-d",json.dumps(data)]
    r=subprocess.run(cmd,capture_output=True,text=True)
    if not r.stdout: return {}
    try: return json.loads(r.stdout)
    except: return {}

def sh(cmd):
    try:
        r=subprocess.run(cmd,shell=True,capture_output=True,text=True,timeout=15)
        return (r.stdout+r.stderr).strip()[:3000] or "ok"
    except: return "error"

# ---- TOOLS (same as before) ----
TOOLS=[
    {"type":"function","function":{"name":"shell","description":"执行Linux命令","parameters":{"type":"object","properties":{"cmd":{"type":"string"}},"required":["cmd"]}}},
    {"type":"function","function":{"name":"system_info","description":"系统概览","parameters":{"type":"object","properties":{}}}},
    {"type":"function","function":{"name":"processes","description":"进程列表","parameters":{"type":"object","properties":{}}}},
    {"type":"function","function":{"name":"disk_usage","description":"磁盘空间","parameters":{"type":"object","properties":{}}}},
    {"type":"function","function":{"name":"memory_info","description":"内存使用","parameters":{"type":"object","properties":{}}}},
    {"type":"function","function":{"name":"network","description":"网络端口","parameters":{"type":"object","properties":{}}}},
    {"type":"function","function":{"name":"ping_test","description":"Ping测试","parameters":{"type":"object","properties":{"host":{"type":"string"}},"required":["host"]}}},
    {"type":"function","function":{"name":"read_file","description":"读取文件","parameters":{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}}},
    {"type":"function","function":{"name":"write_file","description":"写入文件","parameters":{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}}},
    {"type":"function","function":{"name":"web_request","description":"HTTP请求","parameters":{"type":"object","properties":{"url":{"type":"string"},"method":{"type":"string"}},"required":["url"]}}},
    {"type":"function","function":{"name":"remember","description":"记住信息","parameters":{"type":"object","properties":{"key":{"type":"string"},"value":{"type":"string"}},"required":["key","value"]}}},
    {"type":"function","function":{"name":"recall","description":"回忆信息","parameters":{"type":"object","properties":{"key":{"type":"string"}},"required":["key"]}}},
]

TOOL_MAP={
    "shell": lambda a: sh(a["cmd"]),
    "system_info": lambda a: sh("echo HOST=$(hostname); cat /etc/os-release|grep PRETTY|cut -d= -f2; uptime; free -h|head -2; df -h /|tail -1; echo IP=$(hostname -I)"),
    "processes": lambda a: sh("ps aux --sort=-%mem|head -20"),
    "disk_usage": lambda a: sh("df -h"),
    "memory_info": lambda a: sh("free -h"),
    "network": lambda a: sh("ss -tlnp"),
    "ping_test": lambda a: sh("ping -c 3 "+a.get("host","")+" 2>/dev/null"),
    "read_file": lambda a: open(a["path"]).read()[:3000] if os.path.isfile(a["path"]) else "not found",
    "write_file": lambda a: open(a["path"],"w").write(a["content"]) or "written",
    "web_request": lambda a: sh("curl -sk -X %s '%s' 2>/dev/null"%(a.get("method","GET"),a["url"])),
    "remember": lambda a: [open(P+"/memory.txt","a").write(a["key"]+"="+a["value"]+"\n"),"remembered"][1],
    "recall": lambda a: sh("grep '^"+a["key"]+"=' "+P+"/memory.txt 2>/dev/null|head -1|cut -d= -f2-") or "not found",
}

def llm(msgs,tools=None):
    b={"model":"deepseek-v4-pro","messages":msgs,"max_tokens":2000}
    if tools: b["tools"]=tools
    return curl("POST","/v1/chat/completions",b)

SYS="你是12fzclaw运行在%s上的设备代理。用工具干活，中文作答。"%N

def agent(msg):
    msgs=[{"role":"system","content":SYS},{"role":"user","content":msg}]
    reply=""
    for _ in range(5):
        resp=llm(msgs,TOOLS)
        m=resp.get("choices",[{}])[0].get("message",{})
        reply=m.get("content","") or reply
        tcs=m.get("tool_calls") or []
        if not tcs: return reply or "ok"
        msgs.append(m)
        for tc in tcs:
            fn=tc["function"]; name=fn["name"]
            args=json.loads(fn["arguments"])
            if name in TOOL_MAP:
                try:
                    result=TOOL_MAP[name](args)
                    print("@"+name,flush=True)
                    curl("POST","/api/devices/activity",{"action":name,"detail":str(result)[:100]})
                except Exception as e:
                    result="error: "+str(e)
                msgs.append({"role":"tool","tool_call_id":tc["id"],"content":str(result)})
    resp=llm(msgs,TOOLS)
    return resp.get("choices",[{}])[0].get("message",{}).get("content",reply) or str(reply)[:100] or "ok"

# ---- WebSocket client (built-in) ----
def ws_connect(token):
    """Connect to WS server and return raw socket"""
    import urllib.parse
    key = base64.b64encode(os.urandom(16)).decode()
    path = f"/ws?token={token}"
    
    sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    ctx = ssl.create_default_context()
    ws = ctx.wrap_socket(sock, server_hostname=H)
    ws.connect((H, 443))
    
    req = f"GET {path} HTTP/1.1\r\nHost: {H}\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: {key}\r\nSec-WebSocket-Version: 13\r\n\r\n"
    ws.send(req.encode())
    
    resp = b""
    while b"\r\n\r\n" not in resp:
        resp += ws.recv(1024)
    
    if b"101" not in resp:
        raise Exception("WS handshake failed")
    
    return ws

def ws_recv(ws):
    """Read one WebSocket frame, return payload"""
    b1 = ws.recv(1)
    if not b1: raise Exception("EOF")
    b2 = ws.recv(1)
    if not b2: raise Exception("EOF")
    
    opcode = b1[0] & 0x0F
    length = b2[0] & 0x7F
    
    if length == 126:
        length = struct.unpack(">H", ws.recv(2))[0]
    elif length == 127:
        length = struct.unpack(">Q", ws.recv(8))[0]
    
    # Server frames are NOT masked
    payload = ws.recv(length)
    
    if opcode == 0x8:  # Close
        raise Exception("WS closed")
    elif opcode == 0x9:  # Ping
        ws.send(b"\x8a\x00")  # Pong
        return ws_recv(ws)
    
    return payload.decode()

def ws_send(ws, text):
    """Send a text WebSocket frame"""
    data = text.encode()
    frame = b"\x81"
    length = len(data)
    if length < 126:
        frame += bytes([length])
    elif length < 65536:
        frame += b"\x7e" + struct.pack(">H", length)
    else:
        frame += b"\x7f" + struct.pack(">Q", length)
    frame += data
    ws.send(frame)

# ---- Main loop (push mode) ----
print("12fzclaw@%s push-mode" % N, flush=True)

# Sync cloud skills
try:
    skills = curl("GET","/api/skills?org_id=00000000-0000-0000-0000-000000000000")
    if isinstance(skills, list):
        for s in skills:
            name = s.get("name","")
            if name and name not in [t["function"]["name"] for t in TOOLS]:
                td = s.get("tool_definition")
                if isinstance(td, str): td = json.loads(td)
                props = {}; required = []
                for pn, pt in td.get("parameters",{}).get("properties",{}).items():
                    props[pn] = {"type":"string","description":str(pt)}
                    if pn in td.get("parameters",{}).get("required",[]): required.append(pn)
                TOOLS.append({"type":"function","function":{"name":name,"description":s.get("description",""),"parameters":{"type":"object","properties":props,"required":required}}})
                handler = s.get("handler",""); pk = list(props.keys())
                def make_fn(h=handler, pk=pk):
                    def fn(a):
                        cmd = h
                        for p in pk: cmd = cmd.replace("%"+p+"%", a.get(p,""))
                        if "%s" in cmd:
                            vals = [a.get(p,"") for p in sorted(pk)]
                            try: cmd = cmd % tuple(vals)
                            except: pass
                        return sh(cmd)
                    return fn
                TOOL_MAP[name] = make_fn()
        print(f"cloud synced {len(skills)}", flush=True)
except Exception as ex:
    print(f"sync err: {ex}", flush=True)

# Connect WebSocket and listen for messages
def hb_loop():
    while True:
        time.sleep(30)
        try:
            curl("POST","/api/devices/heartbeat",{})
        except:
            pass
threading.Thread(target=hb_loop, daemon=True).start()
lhb = time.time()
while True:
    try:
        ws = ws_connect(T)
        print("ws connected", flush=True)
        curl("POST","/api/devices/heartbeat",{})
        lhb = time.time()
        
        while True:
            data = ws_recv(ws)
            try:
                pkt = json.loads(data)
                # Expected format: {"type":"friend_msg","data":{"id":...,"from_id":"1","content":"..."}}
                if pkt.get("type") in ("message","friend_msg"):
                    d = pkt["data"]
                    c = d.get("content","")
                    print("Q:"+c[:40], flush=True)
                    r = agent(c)
                    curl("POST","/api/friend-messages",{"friend_id":"1","content":r})
                    print("A:"+r[:50], flush=True)
                elif pkt.get("type") == "hello":
                    print("ws hello: "+str(pkt.get("data","")), flush=True)
            except Exception as e:
                print("ws msg err: "+str(e)[:80], flush=True)
            
            # Heartbeat
            if time.time() - lhb > 30:
                curl("POST","/api/devices/heartbeat",{})
                lhb = time.time()
                
    except Exception as e:
        print("ws err: "+str(e)[:80]+", reconnect in 5s", flush=True)
        time.sleep(5)
PYEOF

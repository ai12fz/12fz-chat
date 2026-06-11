# 12FZ AI Service
# 独立 AI 服务，通过 webhook 与 chat-core 通信
# 未来可独立部署

from fastapi import FastAPI, Header, HTTPException
from pydantic import BaseModel
import asyncio, json, os

app = FastAPI(title="12FZ AI Service")

BOT_TOKEN = os.getenv("AI_BOT_TOKEN", "dev-bot-token")
CHAT_CORE_URL = os.getenv("CHAT_CORE_URL", "http://localhost:8080")

class BotMessage(BaseModel):
    to: str          # "user:xxx" or "group:xxx"
    content: str
    msgId: str

class BotBroadcast(BaseModel):
    to: str
    content: str

# ---- Bot 处理器注册 ----

class BaseBot:
    """所有 bot 的基类"""
    name: str = ""
    bot_id: str = ""
    
    async def on_message(self, msg: BotMessage) -> str | None:
        """收到消息时处理，返回回复内容"""
        raise NotImplementedError

    async def on_tick(self):
        """定时触发（如心跳检查），默认空"""
        pass

# 注册中心
_bots: dict[str, BaseBot] = {}

def register_bot(bot: BaseBot):
    _bots[bot.bot_id] = bot

# ---- Webhook 端点 ----

@app.post("/webhook/message")
async def handle_message(msg: BotMessage, x_bot_token: str = Header(None)):
    if x_bot_token != BOT_TOKEN:
        raise HTTPException(401, "Invalid bot token")
    
    # 路由到对应 bot 处理
    bot_id = msg.to.split(":")[0] if ":" in msg.to else "default"
    bot = _bots.get(bot_id) or _bots.get("default")
    if not bot:
        raise HTTPException(404, f"Bot {bot_id} not found")
    
    reply = await bot.on_message(msg)
    if reply:
        # 通过 webhook 回写到 chat-core
        await send_to_chat_core(msg.to, reply, bot.bot_id)
    return {"code": 200, "msg": "ok"}

@app.post("/webhook/broadcast")
async def handle_broadcast(bc: BotBroadcast, x_bot_token: str = Header(None)):
    if x_bot_token != BOT_TOKEN:
        raise HTTPException(401, "Invalid bot token")
    
    reply = f"[{bc.content}]"
    await send_to_chat_core(bc.to, reply, "system")
    return {"code": 200, "msg": "ok"}

@app.get("/health")
async def health():
    return {"status": "ok", "bots": list(_bots.keys())}

# ---- 工具函数 ----

async def send_to_chat_core(to: str, content: str, bot_id: str):
    """通过 webhook 发送消息回 chat-core"""
    import httpx
    async with httpx.AsyncClient() as client:
        await client.post(
            f"{CHAT_CORE_URL}/api/v1/bot/send",
            json={"to": to, "content": content, "botId": bot_id},
            headers={"X-Bot-Token": BOT_TOKEN}
        )

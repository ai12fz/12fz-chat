# 12FZ Chat 系统 — API 契约
# chat-core (Go) ↔ ai-service ↔ frontend (Vue)

## 一、约定

- 基础路径：`/api/v1`
- 认证：JWT Bearer Token，前端存 localStorage
- 响应格式：`{code: number, msg: string, data: any}`
- WS 端点：`/ws?token={jwt}`

## 二、认证

```
POST /api/v1/auth/login
  Body: {username, password}
  Response: {code:200, data:{token, user:{id, username, avatar}}}

POST /api/v1/auth/refresh
  Header: Authorization: Bearer {token}
  Response: {code:200, data:{token}}
```

## 三、消息

```
WS /ws?token={jwt}
  消息格式:
  {type:"message", to:"user|group:xxx", content:"text", msgId:"uuid"}
  {type:"ack", msgId:"uuid"}
  {type:"typing", to:"user:xxx"}
  
GET /api/v1/messages?target={user|group:xxx}&before={msgId}&limit=50
  Response: {code:200, data:{messages:[...], hasMore:bool}}
```

## 四、群组/联系人

```
GET /api/v1/groups
  Response: {code:200, data:[{id, name, avatar, unread, lastMsg}]}

GET /api/v1/contacts
  Response: {code:200, data:[{id, username, avatar, online, unread}]}
  
GET /api/v1/contacts/search?q=xxx
  Response: {code:200, data:[...]}
```

## 五、机器人消息（AI ↔ Chat-core）

```
ai-service → chat-core (Webhook)
  POST /api/v1/bot/send
    Header: X-Bot-Token: {preSharedToken}
    Body: {to:"user|group:xxx", content:"text", botId:"bot001"}
    
  POST /api/v1/bot/broadcast  
    Header: X-Bot-Token: {preSharedToken}
    Body: {to:"group:xxx", content:"text", botId:"bot001"}
```

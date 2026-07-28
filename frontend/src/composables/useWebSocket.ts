import { ref, onUnmounted } from 'vue'
import { useChatStore } from '../stores/chat'
import type { BackendMessage } from '../stores/chat'

export function useWebSocket() {
  let ws: WebSocket | null = null
  const store = useChatStore()
  let connectedHandlers: Array<()=>void> = []
  const reconnectTimer = ref<number>()

  function connect(token: string) {
    if (ws) disconnect()

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${location.host}/ws?token=${token}`

    ws = new WebSocket(url)

    ws.onopen = () => {
      console.log('[ws] connected')
      store.setConnected(true)
      clearTimeout(reconnectTimer.value)
    }

    ws.onmessage = (e) => {
      try {
        const pkt = JSON.parse(e.data)
        // Backend protocol:
        //   {type:"hello", data:{bot_id, msg}}
        //   {type:"message", data:{id,group_id,sender_id,content,msg_type,send_at}}
        //   {type:"event", data:{event, bot_id}}
        switch (pkt.type) {
          case 'hello':
            console.log('[ws] hello:', pkt.data)
            break
          case 'message': { console.log('[ws-debug] GOT message', pkt.data);
            const d = pkt.data
            if (d.from && d.to && !d.group_id) {
              // Only process messages TO me (ignore self-echo)
              const tok = localStorage.getItem('token') || ''
              const myId = tok.startsWith('session-') ? tok.slice(8) : tok
              const isFromMe = (d.from === myId || d.from === tok)
              if (isFromMe) { /* echo, skip */ }
              else {
                const fid = d.from
                const sid = 'friend:' + fid
                let s = store.sessions.find(function(x: any){ return x.id === sid })
                if (!s) {
                  s = { id: sid, name: fid, type: 'friend', messages: [] as any[], members: [], lastMsg: '', lastMsgAt: '' } as any
                  store.sessions.push(s)
                }
                if (!s.messages.some(function(m: any){ return m.content === d.content && m.sender_id === fid })) {
                  s.messages.push({
                    id: d.id || Date.now(), group_id: 0, sender_id: fid,
                    content: d.content, msg_type: 'text',
                    created_at: d.created_at || new Date().toISOString()
                  })
                }
                s.lastMsg = d.content
                s.lastMsgAt = d.created_at || new Date().toISOString()
                if (sid !== store.activeId) { s.unread = (s.unread||0) + 1 }
              }
            } else {
              handleMessage(pkt.data)
            }
            break
          }
          case 'agent_status':
            handleAgentStatus(pkt.data, pkt.from)
            break
          case 'event':
            handleEvent(pkt.data)
            break
        }
      } catch (err) {
        console.error('[ws] parse error:', err)
      }
    }

    ws.onclose = (e) => {
        console.log('[ws] onclose code=' + e.code + ' reason=' + e.reason + ' wasClean=' + e.wasClean)
      console.log('[ws] disconnected, reconnecting in 3s...')
      store.setConnected(false)
      reconnectTimer.value = window.setTimeout(() => connect(token), 3000)
    }

    ws.onerror = () => {
      console.error('[ws] error')
      ws?.close()
    }
  }

  function handleMessage(data: BackendMessage) {
    store.receiveMessage(data)
  }

  function handleEvent(data: { event: string; bot_id: string }) {
    if (data.event === 'user_online' || data.event === 'user_offline') {
      console.log(`[ws] ${data.bot_id} ${data.event}`)
    }
  }

  function handleAgentStatus(data: any, from: string) {
    store.addAgentStatus(from, data)
  }

  /** Send a text message via WebSocket */
  function sendMessage(groupId: number, content: string) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    ws.send(JSON.stringify({
      type: 'message',
      // Backend expects { group_id, content } in data
      data: {
        group_id: groupId,
        content,
      },
    }))
  }

  function disconnect() {
    ws?.close()
    ws = null
    clearTimeout(reconnectTimer.value)
  }

  onUnmounted(disconnect)

  return { connect, sendMessage, disconnect }
}

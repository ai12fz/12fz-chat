import { ref, onUnmounted } from 'vue'
import { useChatStore } from '../stores/chat'
import type { BackendMessage } from '../stores/chat'

export function useWebSocket() {
  let ws: WebSocket | null = null
  const store = useChatStore()
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
          case 'message':
            handleMessage(pkt.data)
            break
          case 'event':
            handleEvent(pkt.data)
            break
        }
      } catch (err) {
        console.error('[ws] parse error:', err)
      }
    }

    ws.onclose = () => {
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

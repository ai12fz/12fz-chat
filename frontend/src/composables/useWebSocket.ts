import { ref, onUnmounted } from 'vue'
import { useChatStore } from '../stores/chat'

export function useWebSocket() {
  let ws: WebSocket | null = null
  const store = useChatStore()
  const reconnectTimer = ref<number>()

  function connect(token: string) {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${location.host}/ws?token=${token}`

    ws = new WebSocket(url)

    ws.onopen = () => {
      store.setConnected(true)
      clearTimeout(reconnectTimer.value)
    }

    ws.onmessage = (e) => {
      try {
        const msg = JSON.parse(e.data)
        handleMessage(msg)
      } catch { }
    }

    ws.onclose = () => {
      store.setConnected(false)
      reconnectTimer.value = window.setTimeout(() => connect(token), 3000)
    }
  }

  function handleMessage(msg: any) {
    if (msg.type === 'message') {
      store.addMessage(msg.from, {
        id: msg.msgId,
        from: msg.fromId,
        fromName: msg.fromName,
        content: msg.content,
        timestamp: msg.timestamp || Date.now(),
        type: msg.msgType || 'text',
      })
    }
  }

  function send(to: string, content: string) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return
    ws.send(JSON.stringify({
      type: 'message',
      to,
      content,
      msgId: crypto.randomUUID(),
    }))
  }

  function disconnect() {
    ws?.close()
    ws = null
    clearTimeout(reconnectTimer.value)
  }

  onUnmounted(disconnect)

  return { connect, send, disconnect }
}

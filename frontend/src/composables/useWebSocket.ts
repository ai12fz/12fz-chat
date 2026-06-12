import { ref } from 'vue'
import { useChatStore } from '../stores/chat'
import { markRead } from '../api'
import type { BackendMessage } from '../stores/chat'

// ── Singleton: one WebSocket instance shared by all components ──
let ws: WebSocket | null = null
let reconnectTimer = 0
let currentToken = ''
let connectedRef = ref(false)

export function useWebSocket() {
  const store = useChatStore()

  function connect(token: string) {
    currentToken = token
    if (ws) {
      // Already connected or connecting — skip
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) return
      disconnect()
    }

    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const url = `${protocol}//${location.host}/ws?token=${token}`

    ws = new WebSocket(url)

    ws.onopen = () => {
      console.log('[ws] connected')
      store.setConnected(true)
      connectedRef.value = true
      clearTimeout(reconnectTimer)
    }

    ws.onmessage = (e) => {
      try {
        const pkt = JSON.parse(e.data)
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
      connectedRef.value = false
      ws = null
      reconnectTimer = window.setTimeout(() => connect(currentToken), 3000)
    }

    ws.onerror = () => {
      console.error('[ws] error')
      ws?.close()
    }
  }

  function handleMessage(data: BackendMessage & { send_at?: string }) {
    // Normalize: WS sends send_at, REST returns created_at
    if (data.send_at && !data.created_at) {
      (data as any).created_at = data.send_at
    }
    const store = useChatStore()
    store.receiveMessage(data as BackendMessage)
    // Real-time markRead: if the current active session is this group
    const activeId = store.activeId
    if (activeId && activeId === `group:${data.group_id}`) {
      markRead(data.group_id, data.id).catch(() => {/* non-critical */})
    }
  }

  function handleEvent(data: { event: string; bot_id: string }) {
    if (data.event === 'user_online' || data.event === 'user_offline') {
      console.log(`[ws] ${data.bot_id} ${data.event}`)
    }
  }

  function sendMessage(groupId: number, content: string) {
    if (!ws || ws.readyState !== WebSocket.OPEN) {
      console.warn('[ws] cannot send: not connected')
      return
    }
    ws.send(JSON.stringify({
      type: 'message',
      data: {
        group_id: groupId,
        content,
      },
    }))
  }

  function disconnect() {
    ws?.close()
    ws = null
    clearTimeout(reconnectTimer)
  }

  return { connect, sendMessage, disconnect, connected: connectedRef }
}

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { getMessages } from '../api'

export interface ChatMessage {
  id: string
  from: string
  fromName: string
  content: string
  timestamp: number
  type: 'text' | 'image' | 'system'
}

export interface ChatSession {
  id: string
  name: string
  type: 'user' | 'group' | 'bot'
  unread: number
  online?: boolean
  lastMsg?: string
  messages: ChatMessage[]
}

export const useChatStore = defineStore('chat', () => {
  const sessions = ref<ChatSession[]>([])
  const activeId = ref<string>('')
  const connected = ref(false)

  const activeSession = computed(() =>
    sessions.value.find(s => s.id === activeId.value)
  )

  function addSession(session: ChatSession) {
    if (!sessions.value.find(s => s.id === session.id)) {
      sessions.value.push(session)
    }
  }

  function setActive(id: string) {
    const s = sessions.value.find(s => s.id === id)
    if (s) {
      s.unread = 0
      activeId.value = id
    }
  }

  function addMessage(sessionId: string, msg: ChatMessage) {
    const s = sessions.value.find(s => s.id === sessionId)
    if (s) {
      s.messages.push(msg)
      s.lastMsg = msg.content
      if (sessionId !== activeId.value) s.unread++
    }
  }

  async function loadHistory(sessionId: string, before?: string) {
    const res = await getMessages(sessionId, before)
    const s = sessions.value.find(s => s.id === sessionId)
    if (s) {
      s.messages = [...res.messages, ...s.messages]
    }
    return res.hasMore
  }

  function setConnected(val: boolean) {
    connected.value = val
  }

  return {
    sessions, activeId, activeSession, connected,
    addSession, setActive, addMessage, loadHistory, setConnected
  }
})

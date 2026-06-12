import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

// ── Types matching backend API ──

export interface GroupInfo {
  id: number
  name: string
  created_by: string
  created_at: string
  last_msg_at?: string
}

export interface GroupMember {
  group_id: number
  bot_id: string
  role: string
  joined_at: string
  last_read_msg_id?: number
}

export interface FriendInfo {
  user_id: string
  friend_id: string
  status: string
  created_at: string
}

export interface BackendMessage {
  id: number
  group_id: number
  sender_id: string
  content: string
  msg_type: string
  created_at: string
  send_at?: string     // WS broadcasts use this (Go ChatMessage)
}

export interface ChatSession {
  id: string           // "group:123" or "user:abc"
  name: string
  type: 'group' | 'user'
  unread: number
  lastMsg?: string
  lastMsgAt?: string
  messages: BackendMessage[]
  members?: GroupMember[]
}

// ── Store ──

export const useChatStore = defineStore('chat', () => {
  const sessions = ref<ChatSession[]>([])
  const activeId = ref<string>('')
  const connected = ref(false)
  // 已处理标记：session_id → true
  const doneSessions = ref<Record<string, boolean>>({})

  const activeSession = computed(() =>
    sessions.value.find(s => s.id === activeId.value)
  )

  /** Get session id for group */
  function groupSessionId(groupId: number): string {
    return `group:${groupId}`
  }

  /** Get or create a group session */
  function ensureGroupSession(group: GroupInfo): ChatSession {
    const id = groupSessionId(group.id)
    let s = sessions.value.find(s => s.id === id)
    if (!s) {
      s = {
        id,
        name: group.name,
        type: 'group',
        unread: 0,
        lastMsg: undefined,
        messages: [],
      }
      sessions.value.push(s)
    }
    // Update metadata
    s.lastMsgAt = group.last_msg_at
    return s
  }

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

  /** Add a received message to a session */
  function receiveMessage(msg: BackendMessage) {
    const sessionId = groupSessionId(msg.group_id)
    const s = sessions.value.find(s => s.id === sessionId)
    if (!s) return

    // Dedup
    if (s.messages.some(m => m.id === msg.id)) return

    s.messages.push({
      id: msg.id,
      group_id: msg.group_id,
      sender_id: msg.sender_id,
      content: msg.content,
      msg_type: msg.msg_type || 'text',
      created_at: msg.created_at,
    })
    s.lastMsg = msg.content
    s.lastMsgAt = msg.created_at
    if (sessionId !== activeId.value) {
      s.unread++
      // 有新消息 → 自动移出已处理
      delete doneSessions.value[sessionId]
    }
  }

  /** Load historical messages into a session */
  function loadMessages(sessionId: string, msgs: BackendMessage[]) {
    const s = sessions.value.find(s => s.id === sessionId)
    if (!s) return
    // Prepend older messages
    const existingIds = new Set(s.messages.map(m => m.id))
    const newMsgs = msgs.filter(m => !existingIds.has(m.id))
    s.messages = [...newMsgs, ...s.messages]
  }

  function setConnected(val: boolean) {
    connected.value = val
  }

  function setMembers(groupId: number, members: GroupMember[]) {
    const s = sessions.value.find(s => s.id === groupSessionId(groupId))
    if (s) s.members = members
  }

  // 待处理会话：有未读消息
  const pendingSessions = computed(() =>
    sessions.value.filter(s => s.unread > 0)
  )

  // 已处理会话：手动标记、无新消息
  const doneSessionList = computed(() =>
    sessions.value.filter(s => s.unread === 0 && doneSessions.value[s.id])
  )

  // 标记已处理
  function markDone(id: string) {
    const s = sessions.value.find(s => s.id === id)
    if (s) {
      s.unread = 0
      doneSessions.value[id] = true
    }
  }

  return {
    sessions, activeId, activeSession, connected, doneSessions,
    pendingSessions, doneSessionList,
    groupSessionId, ensureGroupSession,
    addSession, setActive, receiveMessage, loadMessages,
    setConnected, setMembers, markDone,
  }
})

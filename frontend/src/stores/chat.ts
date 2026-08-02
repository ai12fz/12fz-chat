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
  user_id: number
  role: string
  joined_at: string
  last_read_msg_id?: number
}

export interface FriendInfo {
  user_id: string
  friend_id: number
  status: string
  created_at: string
}

export interface BackendMessage {
  id: number
  group_id: number
  sender_id: number
  content: string
  msg_type: string
  created_at: string
}

export interface ChatSession {
  id: string           // "group:123" or "user:abc"
  name: string
  type: 'group' | 'friend'
  unread: number
  lastMsg?: string
  lastMsgAt?: string
  lastReadMsgId?: number
  messages: BackendMessage[]
  members?: GroupMember[]
}

// ── Store ──

export const useChatStore = defineStore('chat', () => {
  const sessions = ref<ChatSession[]>([])
  const activeId = ref<string>('')
  const connected = ref(false)
  // Poll REST API for agent status
  setInterval(async () => {
    try {
      // Use the friends list to get agent IDs
      const ids = ['agent-1785267337061', 'hermes-win-qiuming']
      for (const botId of ids) {
        const resp = await fetch('/api/agent-status?bot_id=' + botId, { headers: { Authorization: 'Bearer ' + (localStorage.getItem('token') || '') } })
        if (!resp.ok) continue
        const data = await resp.json()
        if (data.status === 'online' || data.message) {
          addAgentStatus(botId, data)
        }
      }
    } catch(e) {}
  }, 5000)

  const agentStatuses = ref<Record<string, any[]>>({})  // deviceId -> status entries
  // Live online/offline map driven by WS user_online/user_offline events (botId -> bool)
  const onlineMap = ref<Record<string, boolean>>({})

  function setOnline(botId: string, online: boolean) {
    onlineMap.value[botId] = online
  }

  function addAgentStatus(from: string, data: any) {
    // {p:"d"} = done/clear
    if (data.p === 'd') {
      delete agentStatuses.value[from]
      return
    }
    if (!agentStatuses.value[from]) {
      agentStatuses.value[from] = []
    }
    agentStatuses.value[from].push({
      ...data,
      time: Date.now()
    })
    // Keep last 20 entries
    if (agentStatuses.value[from].length > 20) {
      agentStatuses.value[from].shift()
    }
  }

  function clearAgentStatus(from?: string) {
    if (from) {
      delete agentStatuses.value[from]
    } else {
      agentStatuses.value = {}
    }
  }

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
        members: [],
      }
      sessions.value.push(s)
    }
    // Update metadata
    s.lastMsgAt = group.last_msg_at
    s.lastReadMsgId = group.last_read_msg_id || 0
    s.unread = group.unread || 0
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
      localStorage.setItem('hermes_active_session', id)
    }
  }

  // Clear stale active session on page load; auto-select will set the right one
  localStorage.removeItem('hermes_active_session')

  /** Add a received message to a session */
  function receiveMessage(msg: BackendMessage) {
    // Friend message (from WS: {from, to, content, timestamp})
    if ((msg as any).from && (msg as any).to && !msg.group_id) {
      const friendId = (msg as any).from
      const sessionId = 'friend:' + friendId
      let s = sessions.value.find(x => x.id === sessionId)
      if (!s) {
        s = { id: sessionId, name: friendId, type: 'friend', messages: [] as any[], members: [], lastMsg: '', lastMsgAt: '' } as any
        sessions.value.push(s)
      }
      // Only add if not duplicate
      if (!s.messages.some(function(m: any){ return m.content === msg.content && m.sender_id === friendId })) {
        s.messages.push({
          id: (msg as any).id || Date.now(),
          group_id: 0,
          sender_id: friendId,
          content: msg.content,
          msg_type: 'text',
          created_at: (msg as any).created_at || new Date().toISOString(),
        })
      }
      s.lastMsg = msg.content
      s.lastMsgAt = (msg as any).created_at || new Date().toISOString()
      if (sessionId !== activeId.value) s.unread++
      return
    }
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
    if (sessionId !== activeId.value) s.unread++
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

  return {
    agentStatuses,
    setConnected,
    addAgentStatus,
    clearAgentStatus,
    sessions, activeId, activeSession, connected,
    groupSessionId, ensureGroupSession,
    addSession, setActive, receiveMessage, loadMessages,
    setConnected, setMembers,
  }
})

<template>
  <div class="chat-layout">
    <SidebarLeft />
    <ChatContent />
    <AgentStatusPanel />
  </div>
</template>

<script setup lang="ts">
import AgentStatusPanel from '../components/AgentStatusPanel.vue'
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { useWebSocket } from '../composables/useWebSocket'
import { getMyGroups, getMessages } from '../api'
import axiosApi from '../api'
import SidebarLeft from '../components/SidebarLeft.vue'
import ChatContent from '../components/ChatContent.vue'
// import AgentStatusPanel from '../components/SidebarRight.vue'

const router = useRouter()
const auth = useAuthStore()
const chat = useChatStore()
const ws = useWebSocket()

onMounted(async () => {
  if (!auth.token) {
    router.push('/login')
    return
  }

  // Connect WebSocket
  ws.connect(auth.token)

  // Fetch user identity from go.12fz.com
  await auth.fetchWhoAmI()

  try {
    // Load groups
    const groups = await getMyGroups()
    if (groups && Array.isArray(groups)) groups.forEach((g: any) => {
      chat.ensureGroupSession(g)
    })

    // Load friend sessions from stored friend messages
    try {
      const resp = await axiosApi.get('/friend-messages', { params: { with: 'hermes-win-qiuming', limit: 1 } })
      const msgs = resp.data
      if (msgs && msgs.length > 0) {
        const last = msgs[msgs.length - 1]
        const friendId = typeof last.to_id === 'string' && last.to_id !== '1' ? last.to_id : last.from_id
        const sid = 'friend:' + friendId
        let s = chat.sessions.find((x: any) => x.id === sid)
        if (!s) {
          s = { id: sid, name: friendId, type: 'friend', messages: msgs.reverse(), members: [], lastMsg: last.content, lastMsgAt: last.created_at, unread: 0 }
          chat.sessions.push(s)
        }
      }
    } catch(e) { console.error('[init] load friend session:', e) }

    // Load messages for all groups immediately
    for (const g of groups) {
      try {
        const sid = chat.groupSessionId(g.id)
        const msgs = await getMessages(g.id, 50, 0)
        if (msgs && msgs.length > 0) {
          const sessionMsgs = msgs
          const idx = chat.sessions.findIndex(function(x){ return x.id === sid })
          if (idx >= 0) {
            chat.sessions[idx].messages = sessionMsgs
            chat.sessions[idx].unread = 0
          }
        }
      } catch(e) { console.error("load error:", e) }
    }
    // Select first session
    // Auto-select handled by SidebarLeft after friends load
  } catch (err) {
    console.error('Failed to load chat data:', err)
  }
})
</script>

<style scoped>
.chat-layout {
  display: flex;
  height: 100%;
  background: #fff;
}
</style>

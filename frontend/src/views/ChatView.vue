<template>
  <div class="chat-layout">
    <SidebarLeft />
    <ChatContent />
    <SidebarRight />
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { useWebSocket } from '../composables/useWebSocket'
import { getMyGroups, getMessages } from '../api'
import SidebarLeft from '../components/SidebarLeft.vue'
import ChatContent from '../components/ChatContent.vue'
import SidebarRight from '../components/SidebarRight.vue'

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

  try {
    // Load groups
    const groups = await getMyGroups()
    let firstId = ''

    if (groups && Array.isArray(groups)) groups.forEach((g: any) => {
      chat.ensureGroupSession(g)
      if (!firstId) firstId = chat.groupSessionId(g.id)
    })

    // Load unread counts
    try {
      const unreads = await getUnreadCounts()
      for (const [groupIdStr, count] of Object.entries(unreads)) {
        const sid = chat.groupSessionId(Number(groupIdStr))
        const s = chat.sessions.find(s => s.id === sid)
        if (s) s.unread = count as number
      }
    } catch { /* unread endpoint may not have data */ }

    // Load messages for all groups immediately
    for (const g of groups) {
      try {
        const sid = chat.groupSessionId(g.id)
        const msgs = await getMessages(g.id, 50, 0)
        if (msgs && msgs.length > 0) {
          const sessionMsgs = [...msgs].reverse()
          const idx = chat.sessions.findIndex(function(x){ return x.id === sid })
          if (idx >= 0) {
            chat.sessions[idx].messages = sessionMsgs
          }
        }
      } catch(e) { console.error("load error:", e) }
    }
    // Select first session
    if (firstId) chat.setActive(firstId)
  } catch (err) {
    console.error('Failed to load chat data:', err)
  }
})
</script>

<style scoped>
.chat-layout {
  display: flex;
  height: 100vh;
  background: #fff;
}
</style>

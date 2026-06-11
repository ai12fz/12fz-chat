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
import { getGroups, getContacts } from '../api'
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

  ws.connect(auth.token)

  const [groups, contacts] = await Promise.all([
    getGroups(),
    getContacts(),
  ])

  groups.forEach((g: any) => chat.addSession({
    id: `group:${g.id}`, name: g.name, type: 'group',
    unread: g.unread || 0, lastMsg: g.lastMsg, messages: [],
  }))

  contacts.forEach((c: any) => chat.addSession({
    id: `user:${c.id}`, name: c.username, type: 'user',
    unread: c.unread || 0, online: c.online, messages: [],
  }))

  if (groups.length) chat.setActive(`group:${groups[0].id}`)
})
</script>

<style scoped>
.chat-layout {
  display: flex;
  height: 100vh;
  background: #fff;
}
</style>

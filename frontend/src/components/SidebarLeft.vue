<template>
  <aside class="sidebar-left">
    <div class="sidebar-header">
      <div class="user-info" @click="handleLogout" title="点击退出">
        <span class="avatar">{{ displayName[0] }}</span>
        <span class="name">{{ displayName }}</span>
      </div>
    </div>
    <div class="search-box">
      <input v-model="search" placeholder="搜索聊天..." />
    </div>
    <nav class="session-list">
      <!-- Group sessions -->
      <div class="section-title">群聊</div>
      <div v-for="s in filteredSessions.filter(x => x.type !== 'friend')" :key="s.id" class="session-item" :class="{ active: s.id === chat.activeId }" @click="chat.setActive(s.id)">
        <span class="avatar sm" :style="{ background: avatarColor(s) }">{{ s.name[0] }}</span>
        <div class="session-info">
          <div class="session-top">
            <span class="session-name">{{ s.name }}</span>
            <span class="session-badge">群</span>
          </div>
          <span class="session-msg">{{ s.lastMsg || '暂无消息' }}</span>
        </div>
        <span v-if="s.unread > 0" class="unread-badge">{{ s.unread > 99 ? '99+' : s.unread }}</span>
      </div>

      <!-- Friend sessions -->
      <div class="section-title">好友</div>
      <div v-for="f in friends" :key="f.friend_id" class="session-item" :class="{ active: chat.activeId === 'friend:' + f.friend_id }" @click="openFriendChat(f.friend_id, f.display_name)">
        <span class="avatar sm" :style="{ background: avatarColor({name: f.display_name}) }">{{ f.display_name[0] }}</span>
        <div class="session-info">
          <div class="session-top">
            <span class="session-name">{{ f.display_name }}</span>
            <span class="session-badge">友</span>
          </div>
          <span class="session-msg">{{ f.status || '暂无消息' }}</span>
        </div>
      </div>

      <div class="session-item add-friend-btn" @click="showAddFriend = true">
        <span class="avatar sm" style="background: #52c41a">+</span>
        <div class="session-info"><span class="session-name">添加好友</span></div>
      </div>

      <div v-if="filteredSessions.filter(x => x.type !== 'friend').length === 0 && friends.length === 0" class="empty-hint">暂无会话</div>
    </nav>

    <AddFriendDialog v-if="showAddFriend" @close="showAddFriend = false; loadFriends()" />
  </aside>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { getFriends } from '../api'
import AddFriendDialog from './AddFriendDialog.vue'

const router = useRouter()
const auth = useAuthStore()
const chat = useChatStore()
const search = ref('')

const friends = ref<any[]>([])
const showAddFriend = ref(false)

const displayName = computed(() => auth.user?.username || auth.user?.bot_id || '用户')

const filteredSessions = computed(() => {
  if (!search.value) return chat.sessions
  const q = search.value.toLowerCase()
  return chat.sessions.filter((s: any) => s.name.toLowerCase().includes(q))
})

function avatarColor(s: { name: string }) {
  const colors = ['#1890ff', '#52c41a', '#fa8c16', '#eb2f96', '#722ed1', '#13c2c2', '#f5222d']
  let hash = 0
  for (let i = 0; i < s.name.length; i++) hash = s.name.charCodeAt(i) + ((hash << 5) - hash)
  return colors[Math.abs(hash) % colors.length]
}

async function loadFriends() {
  const token = localStorage.getItem('token') || ''
  const userId = token.startsWith('session-') ? token.slice(8) : auth.user?.bot_id || auth.user?.username
  if (!userId) return
  try {
    const res = await getFriends(userId)
    friends.value = Array.isArray(res) ? res : []
  } catch(e) { console.error('Failed to load friends:', e) }
}

function openFriendChat(friendId: string, displayName: string) {
  const sid = 'friend:' + friendId
  let session = chat.sessions.find((s: any) => s.id === sid)
  if (!session) {
    session = { id: sid, name: displayName, type: 'friend', messages: [], members: [] }
    chat.sessions.push(session)
  }
  chat.activeId = sid
}

function handleLogout() {
  auth.logout()
  router.push('/login')
}

loadFriends()
</script>

<style scoped>
/* keep existing styles */
.section-title {
  padding: 8px 16px 4px;
  font-size: 11px;
  color: #999;
  text-transform: uppercase;
}
.add-friend-btn {
  border-top: 1px solid #f0f0f0;
  margin-top: 8px;
  padding-top: 10px;
}
</style>

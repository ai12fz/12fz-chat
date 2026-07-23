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
      <div v-for="s in (filteredSessions || []).filter(function(x){ return x.type !== 'friend' })" :key="s.id" class="session-item" :class="{ active: s.id === chat.activeId }" @click="chat.setActive(s.id)">
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
      <div v-for="f in (friends || [])" :key="f.friend_id" class="session-item" :class="{ active: chat.activeId === 'friend:' + f.friend_id }" @click="openFriendChat(f.friend_id, f.friend_id)">
        <span class="avatar sm" :style="{ background: avatarColor({name: f.friend_id}) }">{{ f.friend_id[0] }}</span>
        <div class="session-info">
          <div class="session-top">
            <span class="session-name">{{ f.friend_id }}</span>
            <span class="session-badge">友</span>
          </div>
          <span class="session-msg">{{ f.status || '暂无消息' }}</span>
        </div>
      </div>

      <div class="session-item add-friend-btn" @click="showAddFriend = true">
        <span class="avatar sm" style="background: #52c41a">+</span>
        <div class="session-info"><span class="session-name">添加好友</span></div>
      </div>

      <div v-if="(filteredSessions || []).filter(function(x){ return x.type !== 'friend' }).length === 0 && (friends || []).length === 0" class="empty-hint">暂无会话</div>
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
    friends.value = Array.isArray(res) ? res.map(function(f){ if (!f.display_name) f.display_name = f.friend_id; return f }) : []
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
.sidebar-left {
  width: 280px;
  border-right: 1px solid #e8e8e8;
  display: flex;
  flex-direction: column;
  background: #fafafa;
  flex-shrink: 0;
}
.sidebar-header {
  padding: 12px 16px;
  border-bottom: 1px solid #e8e8e8;
}
.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}
.avatar {
  width: 36px; height: 36px;
  border-radius: 50%;
  background: #1890ff;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  flex-shrink: 0;
}
.avatar.sm { width: 32px; height: 32px; font-size: 12px; }
.name { font-weight: 500; font-size: 14px; }
.search-box { padding: 8px 12px; }
.search-box input {
  width: 100%;
  padding: 6px 10px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
  box-sizing: border-box;
  outline: none;
}
.search-box input:focus { border-color: #1890ff; }
.session-list { flex: 1; overflow-y: auto; padding: 4px 0; }
.session-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  cursor: pointer;
  position: relative;
  transition: background .15s;
}
.session-item:hover { background: #f0f0f0; }
.session-item.active { background: #e6f7ff; }
.session-info { flex: 1; min-width: 0; }
.session-top {
  display: flex;
  align-items: center;
  gap: 4px;
}
.session-name { font-size: 14px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.session-badge {
  font-size: 10px;
  background: #e8e8e8;
  padding: 0 4px;
  border-radius: 2px;
  flex-shrink: 0;
}
.session-msg {
  font-size: 12px;
  color: #888;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
  margin-top: 2px;
}
.unread-badge {
  background: #f5222d;
  color: #fff;
  font-size: 11px;
  min-width: 18px;
  height: 18px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 4px;
  flex-shrink: 0;
}
.empty-hint {
  text-align: center;
  color: #bbb;
  padding: 32px 16px;
  font-size: 13px;
}

.section-title { padding: 8px 16px 4px; font-size: 11px; color: #999; text-transform: uppercase; }
.add-friend-btn { border-top: 1px solid #f0f0f0; margin-top: 8px; padding-top: 10px; }
</style>

<template>
  <aside class="sidebar-left">
    <div class="sidebar-header">
      <div class="user-info" @click="showProfile = true" title="查看/编辑个人信息">
        <span class="avatar">{{ displayName[0] }}</span>
        <span class="name">{{ displayName }}</span>
      </div>
      <button class="logout-btn" @click="handleLogout" title="退出登录">退出</button>
      <button class="admin-btn" @click="$router.push('/admin/agents')" title="Agent管理">⚙ 管理</button>
    </div>

    <!-- Tab 切换 -->
    <div class="tab-bar">
      <div class="tab" :class="{ active: tab === 'msg' }" @click="tab = 'msg'">消息</div>
      <div class="tab" :class="{ active: tab === 'friends' }" @click="tab = 'friends'">好友</div>
    </div>

    <!-- 消息 tab -->
    <div v-show="tab === 'msg'" class="tab-content">
      <div class="search-box">
        <input v-model="search" placeholder="搜索聊天..." />
      </div>
      <nav class="session-list">
        <div v-for="s in filteredSessions" :key="s.id"
          class="session-item"
          :class="{ active: s.id === chat.activeId }"
          @click="chat.setActive(s.id)">
          <span class="avatar sm" :style="{ background: avatarColor(s) }">{{ s.name[0] }}</span>
          <div class="session-info">
            <div class="session-top">
              <span class="session-name">{{ s.name }}</span>
              <span class="session-badge">{{ s.type === 'group' ? '群' : '友' }}</span>
            </div>
            <span class="session-msg">{{ s.lastMsg || '暂无消息' }}</span>
          </div>
          <span v-if="s.unread > 0" class="unread-badge">{{ s.unread > 99 ? '99+' : s.unread }}</span>
        </div>
        <div v-if="filteredSessions.length === 0" class="empty-hint">暂无会话</div>
      </nav>
    </div>

    <!-- 好友 tab -->
    <div v-show="tab === 'friends'" class="tab-content">
      <nav class="session-list">
        <div v-for="f in (friends || [])" :key="f.friend_id"
          class="session-item"
          :class="{ active: chat.activeId === 'friend:' + f.friend_id }"
          @click="openFriendChat(f.friend_id, f.friend_id)">
          <span class="avatar sm" :style="{ background: avatarColor({name: f.friend_id}) }">{{ f.friend_id[0] }}</span>
          <div class="session-info">
            <span class="session-name">{{ f.friend_id }}</span>
          </div>
        </div>
        <div v-if="friends.length === 0" class="empty-hint">暂无好友</div>
      </nav>
      <div class="add-friend-bar">
        <div class="session-item" @click="showAddFriend = true">
          <span class="avatar sm" style="background: #52c41a">+</span>
          <div class="session-info"><span class="session-name">添加好友</span></div>
        </div>
      </div>
    </div>

    <!-- 个人信息弹窗 -->
    <div v-if="showProfile" class="profile-overlay" @click.self="showProfile = false">
      <div class="profile-card">
        <div class="profile-avatar" :style="{ background: avatarColor({name: displayName}) }">{{ displayName[0] }}</div>
        <h3>{{ displayName }}</h3>
        <div class="profile-form">
          <label>用户名</label>
          <input v-model="profileName" placeholder="输入新名称" />
          <button class="save-btn" @click="saveProfile">保存</button>
          <p v-if="profileMsg" class="profile-msg">{{ profileMsg }}</p>
        </div>
        <button class="close-btn" @click="showProfile = false">关闭</button>
      </div>
    </div>

    <AddFriendDialog :visible="showAddFriend" @close="showAddFriend = false; loadFriends()" />
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
const tab = ref('msg')

const friends = ref<any[]>([])
const showAddFriend = ref(false)
const showProfile = ref(false)
const profileName = ref('')
const profileMsg = ref('')

const displayName = computed(() => auth.user?.username || auth.user?.bot_id || '用户')

const filteredSessions = computed(() => {
  const sortFn = (a: any, b: any) => { const ta = a.lastMsgAt || '', tb = b.lastMsgAt || ''; return ta > tb ? -1 : ta < tb ? 1 : 0 }
  const activeId = chat.activeId
  let list: any[] = []
  if (!search.value) {
    list = chat.sessions.slice().sort(sortFn)
  } else {
    const q = search.value.toLowerCase()
    list = chat.sessions.filter((s: any) => s.name.toLowerCase().includes(q)).sort(sortFn)
  }
  // Active session always first
  if (activeId) {
    const idx = list.findIndex((s: any) => s.id === activeId)
    if (idx > 0) {
      const [active] = list.splice(idx, 1)
      list.unshift(active)
    }
  }
  return list
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
    friends.value = Array.isArray(res) ? res.map(function(f: any){ return f }) : []
  } catch(e) { console.error('Failed to load friends:', e) }
}

function openFriendChat(friendId: string, displayName: string) {
  const sid = 'friend:' + friendId
  let session = chat.sessions.find((s: any) => s.id === sid)
  if (!session) {
    session = { id: sid, name: displayName, type: 'friend', messages: [], members: [], lastMsgAt: new Date().toISOString() }
    chat.sessions.push(session)
  }
  chat.activeId = sid
  tab.value = 'msg'
}

function handleLogout() {
  auth.logout()
  router.push('/login')
}

function saveProfile() {
  if (!profileName.value.trim()) {
    profileMsg.value = '名称不能为空'
    return
  }
  // Update in auth store and localStorage
  const botId = profileName.value.trim()
  auth.botId = botId
  localStorage.setItem('bot_id', botId)
  profileMsg.value = '已保存'
  setTimeout(() => { profileMsg.value = '' }, 2000)
}

// Init profile name on open
const openProfileHandler = () => {
  profileName.value = displayName.value
}

// Watch showProfile to init name
import { watch } from 'vue'
watch(showProfile, (v) => {
  if (v) profileName.value = displayName.value
})

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
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  flex: 1;
  min-width: 0;
}
.logout-btn {
  background: none;
  border: 1px solid #e8e8e8;
  border-radius: 4px;
  font-size: 16px;
  cursor: pointer;
  padding: 4px 8px;
  color: #999;
  flex-shrink: 0;
  margin-left: 8px;
}
.logout-btn:hover { color: #f5222d; border-color: #f5222d; }
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
.name { font-weight: 500; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* Tabs */
.tab-bar {
  display: flex;
  border-bottom: 1px solid #e8e8e8;
}
.tab {
  flex: 1;
  text-align: center;
  padding: 10px 0;
  font-size: 14px;
  cursor: pointer;
  color: #666;
  border-bottom: 2px solid transparent;
  transition: all .15s;
}
.tab:hover { color: #333; }
.tab.active { color: #1890ff; border-bottom-color: #1890ff; }

.tab-content {
  display: flex;
  flex-direction: column;
  flex: 1;
  overflow: hidden;
}

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

.add-friend-bar {
  border-top: 1px solid #e8e8e8;
  padding: 4px 0;
}

/* Profile overlay */
.profile-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,.4);
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
}
.profile-card {
  background: #fff;
  border-radius: 12px;
  padding: 32px;
  width: 320px;
  text-align: center;
  box-shadow: 0 8px 32px rgba(0,0,0,.15);
}
.profile-avatar {
  width: 64px; height: 64px;
  border-radius: 50%;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 24px;
  margin: 0 auto 12px;
}
.profile-card h3 { margin: 0 0 16px; font-size: 18px; }
.profile-form { text-align: left; }
.profile-form label { display: block; font-size: 13px; color: #888; margin-bottom: 4px; }
.profile-form input {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 14px;
  box-sizing: border-box;
}
.save-btn {
  margin-top: 12px;
  width: 100%;
  padding: 8px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 14px;
}
.save-btn:hover { background: #40a9ff; }
.profile-msg { font-size: 12px; color: #52c41a; margin: 8px 0 0; text-align: center; }
.close-btn {
  margin-top: 12px;
  background: none;
  border: 1px solid #d9d9d9;
  padding: 6px 24px;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
}
.close-btn:hover { background: #f5f5f5; }
</style>

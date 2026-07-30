<template>
  <aside class="sidebar-left">
    <div class="sidebar-header">
      <div class="user-info" @click="showProfile = true" title="查看/编辑个人信息">
        <span class="avatar">{{ displayName[0] }}</span>
        <span class="name">{{ displayName }}</span>
      </div>
      <button class="logout-btn" @click="handleLogout" title="退出登录">退出</button>
      <button class="admin-btn" @click="goAdmin" title="Agent管理">⚙ 管理</button>
    </div>
    <div class="tab-bar">
      <div class="tab" :class="{ active: activeTab === 'msg' }" @click="activeTab = 'msg'">消息</div>
      <div class="tab" :class="{ active: activeTab === 'friends' }" @click="activeTab = 'friends'">好友</div>
    </div>

    <div class="tab-content" v-show="activeTab === 'msg'">
      <div class="search-box">
        <input v-model="search" placeholder="搜索聊天..." />
      </div>
      <nav class="session-list">
        <div v-for="s in sortedSessions" :key="s.id" class="session-item" :class="{ active: s.id === chat.activeId }" @click="chat.setActive(s.id)">
          <span class="avatar sm" :style="{ background: avatarColor(s) }">{{ s.name[0] }}</span>
          <div class="session-info">
            <div class="session-top">
              <span class="session-name">{{ s.name }}</span>
              <span v-if="s.type === 'group'" class="session-badge">群</span>
              <span v-else-if="s.userType === 'agent'" class="session-badge badge-agent">🤖 Agent</span>
              <span v-else-if="s.userType === 'device'" class="session-badge badge-device">🖥 主机</span>
              <span v-else class="session-badge badge-human">👤 好友</span>
            </div>
            <span class="session-msg">{{ s.lastMsg || '暂无消息' }}</span>
          </div>
          <span v-if="s.unread > 0" class="unread-badge">{{ s.unread > 99 ? '99+' : s.unread }}</span>
        </div>
        <div v-if="sortedSessions.length === 0" class="empty-hint">暂无会话</div>
      </nav>
    </div>



<div class="tab-content" v-show="activeTab === 'friends'">
      <nav class="session-list">
        <div v-for="f in friends" :key="f.friend_id" class="session-item" :class="{ active: chat.activeId === 'friend:' + f.friend_id }" @click="openFriendChat(f.friend_id, f.friend_id, f.user_type)">
          <span class="avatar sm" :style="{ background: avatarColor({name: f.name || f.friend_id}) }">{{ (f.name || f.friend_id)[0] }}</span>
          <div class="session-info">
            <div class="session-top">
              <span class="session-name">{{ f.name || f.friend_id }}</span>
              <span v-if="f.user_type === 'agent' || f.user_type === 'api'" class="session-badge badge-agent">🤖 Agent</span>
              <span v-else-if="f.user_type === 'human'" class="session-badge badge-human">👤 好友</span>
              <span v-else-if="f.user_type === 'device'" class="session-badge badge-device">🖥 主机</span>
              <span v-else class="session-badge badge-human">👤 好友</span>
            </div>
            <span class="session-msg">{{ f.status || '暂无消息' }}</span>
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

    <div v-if="showProfile" class="profile-overlay" @click.self="showProfile = false">
      <div class="profile-card">
        <div class="profile-avatar" :style="{ background: avatarColor({name: displayName}) }">{{ displayName[0] }}</div>
        <h3>{{ displayName }}</h3>
        <div class="profile-form">
          <span class="user-id-tag">ID: {{ userId || '—' }}</span>
          <label>昵称</label>
          <input v-model="newNickname" placeholder="输入新名称" />
          <button class="save-btn" @click="saveNickname">保存</button>
          <p v-if="profileMsg" class="profile-msg">{{ profileMsg }}</p>
        </div>
        <button class="close-btn" @click="showProfile = false">关闭</button>
      </div>
    </div>

    <AddFriendDialog v-if="showAddFriend" @close="showAddFriend = false; loadFriends()" />
  </aside>
</template>

<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { getFriends } from '../api'
import AddFriendDialog from './AddFriendDialog.vue'

const router = useRouter()
const auth = useAuthStore()
const chat = useChatStore()
const search = ref('')
const activeTab = ref('msg')

const friends = ref<any[]>([])
const nonAgentFriends = computed(() => friends.value)
const showAddFriend = ref(false)
const showProfile = ref(false)
const newNickname = ref('')
const profileMsg = ref('')

const displayName = computed(() => auth.user?.nickname || auth.user?.username || '用户')
const userId = computed(() => localStorage.getItem('user_id') || auth.user?.user_id || '')

const sortedSessions = computed(() => {
  return [...chat.sessions]
    .filter(function(s: any) {
      // Always show friend/device sessions; groups only if they have messages
      if (s.type === 'friend') return true
      var hasContent = s.messages && s.messages.length > 0
      var hasLastMsg = s.lastMsgAt && s.lastMsgAt !== ''
      return hasContent || hasLastMsg
    })
    .sort(function(a,b){
      // Active session always at top
      if (a.id === chat.activeId) return -1
      if (b.id === chat.activeId) return 1
      // Rest sorted by lastMsgAt descending (most recent first)
      var at = a.lastMsgAt || ''
      var bt = b.lastMsgAt || ''
      if (at > bt) return -1
      if (at < bt) return 1
      return 0
    })
})

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

function goAdmin() { router.push('/admin/agents') }

async function loadFriends() {
  const token = localStorage.getItem('token') || ''
  const userId = token.startsWith('session-') ? token.slice(8) : auth.user?.bot_id || auth.user?.username
  if (!userId) return
  try {
    const res = await getFriends(userId)
    friends.value = Array.isArray(res) ? res.map(function(f){ if (!f.display_name) f.display_name = f.friend_id; return f }) : []
    // Auto-create sessions for all friends so sidebar shows them after refresh
    if (Array.isArray(friends.value)) {
      friends.value.forEach(function(f: any, i: number) {
        const sid = 'friend:' + f.friend_id
        if (!chat.sessions.find(function(s: any) { return s.id === sid })) {
          // Stagger timestamps: last friend in list gets newest time
          const d = new Date()
          d.setSeconds(d.getSeconds() - (friends.value.length - 1 - i) * 10)
          chat.sessions.push({
            id: sid, name: f.display_name || f.friend_id, type: 'friend',
            userType: f.user_type || 'human',
            messages: [], members: [], lastMsg: '', lastMsgAt: d.toISOString()
          })
        }
      })
    }
    // Auto-select topmost session; clear stale localStorage
    nextTick(() => {
      const top = chat.sessions.filter(s => s.type !== "group")
        .sort((a: any, b: any) => (a.lastMsgAt > b.lastMsgAt) ? -1 : 1)
      if (top.length > 0) chat.setActive(top[0].id)
    })
  } catch(e) { console.error('Failed to load friends:', e) }
}

function openFriendChat(friendId: string, displayName: string, userType: string) {
  const sid = 'friend:' + friendId
  let session = chat.sessions.find((s: any) => s.id === sid)
  if (!session) {
    session = { id: sid, name: displayName, type: 'friend', userType: userType || 'human', messages: [], members: [], lastMsg: '', lastMsgAt: new Date().toISOString() }
    chat.sessions.push(session)
  }
  chat.activeId = sid
  activeTab.value = 'msg'
}

function handleLogout() { auth.logout() }

function saveNickname() {
  if (!newNickname.value.trim()) { profileMsg.value = '名称不能为空'; return }
  localStorage.setItem('nickname', newNickname.value.trim())
  profileMsg.value = '已保存'
  setTimeout(() => { profileMsg.value = '' }, 2000)
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
  display: flex;
  align-items: center;
  gap: 4px;
}
.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  flex: 1;
  min-width: 0;
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
.name { font-weight: 500; font-size: 14px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.logout-btn, .admin-btn {
  font-size: 12px;
  padding: 2px 6px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  background: #fff;
  cursor: pointer;
  flex-shrink: 0;
}
.logout-btn:hover, .admin-btn:hover { background: #f0f0f0; }

.tab-bar {
  display: flex;
  border-bottom: 1px solid #e8e8e8;
}
.tab {
  flex: 1;
  text-align: center;
  padding: 8px 0;
  font-size: 13px;
  cursor: pointer;
  color: #666;
  border-bottom: 2px solid transparent;
}
.tab.active {
  color: #1890ff;
  border-bottom-color: #1890ff;
}

.tab-content { display: flex; flex-direction: column; flex: 1; min-height: 0; }

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
.session-badge.badge-agent { background: #e6f7ff; color: #1890ff; border: 1px solid #91d5ff; }
.session-badge.badge-device { background: #fff7e6; color: #fa8c16; border: 1px solid #ffd591; }
.category-filter { display: flex; flex-wrap: wrap; gap: 4px; padding: 6px 8px; background: #fafafa; border-bottom: 1px solid #eee; }
.category-filter button { font-size: 11px; padding: 2px 8px; border-radius: 10px; border: 1px solid #d9d9d9; background: #fff; cursor: pointer; color: #666; }
.category-filter button.active { background: #1890ff; color: #fff; border-color: #1890ff; }
.category-filter button:hover { border-color: #1890ff; }
.session-badge.badge-human { background: #f6ffed; color: #52c41a; border: 1px solid #b7eb8f; }
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

.add-friend-bar { border-top: 1px solid #f0f0f0; margin-top: auto; }

.profile-overlay {
  position: fixed; top: 0; left: 0; width: 100%; height: 100%;
  background: rgba(0,0,0,.3); z-index: 1000;
  display: flex; align-items: center; justify-content: center;
}
.profile-card {
  background: #fff; border-radius: 8px; padding: 24px;
  width: 300px; text-align: center;
}
.profile-avatar {
  width: 64px; height: 64px; border-radius: 50%; margin: 0 auto 12px;
  color: #fff; display: flex; align-items: center; justify-content: center;
  font-size: 24px;
}
.profile-form { text-align: left; margin-top: 12px; }
.profile-form label { font-size: 12px; color: #888; display: block; margin-top: 8px; }
.profile-form input {
  width: 100%; padding: 6px 8px; border: 1px solid #d9d9d9;
  border-radius: 4px; font-size: 13px; box-sizing: border-box;
}
.user-id-tag { font-size: 12px; color: #888; }
.save-btn {
  margin-top: 8px; padding: 4px 12px; background: #1890ff; color: #fff;
  border: none; border-radius: 4px; cursor: pointer; font-size: 13px;
}
.close-btn {
  margin-top: 12px; padding: 4px 16px; background: #f5f5f5;
  border: 1px solid #d9d9d9; border-radius: 4px; cursor: pointer;
}
.profile-msg { font-size: 12px; color: #52c41a; margin-top: 4px; }
</style>

<template>
  <aside class="sidebar-left">
    <div class="sidebar-header">
      <div class="user-info" @click="showProfile = true" title="查看/编辑个人信息">
        <span class="avatar">{{ displayName[0] }}</span>
        <span class="name">{{ displayName }}</span>
      </div>
      <button class="logout-btn" @click="handleLogout" title="退出登录">退出</button>
    </div>
    <div class="tab-bar">
      <div class="tab" :class="{ active: activeTab === 'msg' }" @click="activeTab = 'msg'">消息</div>
      <div class="tab" :class="{ active: activeTab === 'friends' }" @click="activeTab = 'friends'">好友</div>
      <div class="tab" :class="{ active: activeTab === 'docs' }" @click="activeTab = 'docs'; loadDocs()">文档</div>
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
      <div class="friend-tabs">
        <div class="ftab" :class="{ active: friendTab === 'human' }" @click="friendTab = 'human'">👤 好友</div>
        <div class="ftab" :class="{ active: friendTab === 'device' }" @click="friendTab = 'device'">🖥 主机</div>
        <div class="ftab" :class="{ active: friendTab === 'agent' }" @click="friendTab = 'agent'">🤖 Agent</div>
      </div>
      <nav class="session-list">
        <div v-for="f in filteredFriends" :key="f.friend_id" class="session-item" :class="{ active: chat.activeId === 'friend:' + f.friend_id }" @click="openFriendChat(f.friend_id, f.name || f.friend_id, f.user_type)">
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
          <button v-if="f.user_type === 'device' || f.user_type === 'agent'" class="grant-btn" title="授权给员工" @click.stop="openGrant(f)">授权</button>
        </div>
        <div v-if="filteredFriends.length === 0" class="empty-hint">{{ friendTab === 'human' ? '暂无好友' : friendTab === 'device' ? '暂无主机' : '暂无Agent' }}</div>
      </nav>
      <div class="add-friend-bar" v-if="friendTab === 'human'">
        <div class="session-item" @click="showAddFriend = true">
          <span class="avatar sm" style="background: #52c41a">+</span>
          <div class="session-info"><span class="session-name">添加好友</span></div>
        </div>
      </div>
    </div>

    <div v-if="grant.show" class="profile-overlay" @click.self="grant.show = false">
      <div class="profile-card grant-card">
        <h3>授权给员工</h3>
        <p class="grant-target">目标: {{ grant.friend?.name || grant.friend?.friend_id }}</p>
        <div class="grant-list">
          <label v-for="s in staff" :key="s.user_id" class="grant-item">
            <input type="checkbox" :value="String(s.user_id)" v-model="grant.checked" />
            <span>{{ s.nickname || s.phone || s.user_id }} <em v-if="s.phone">({{ s.phone }})</em></span>
          </label>
          <div v-if="staff.length === 0" class="empty-hint">暂无员工可授权</div>
        </div>
        <div class="grant-actions">
          <button class="save-btn" :disabled="grant.loading" @click="doGrant">{{ grant.loading ? '提交中...' : '确认授权' }}</button>
          <button class="close-btn" @click="grant.show = false">取消</button>
        </div>
        <p v-if="grant.msg" class="profile-msg">{{ grant.msg }}</p>
      </div>
    </div>

    <div class="tab-content" v-show="activeTab === 'docs'">
      <div class="doc-list-header">
        <span class="doc-list-title">📄 最近文档</span>
        <button class="refresh-btn" @click="loadDocs" title="刷新">⟳</button>
      </div>
      <nav class="session-list">
        <div v-for="d in docs" :key="d.id" class="session-item" @click="previewDoc(d)">
          <span class="avatar sm" style="background: #fa8c16">📄</span>
          <div class="session-info">
            <div class="session-top">
              <span class="session-name doc-name">{{ d.title }}</span>
            </div>
            <span class="session-msg">{{ formatSize(d.size) }} · {{ formatTime(d.created_at) }}</span>
          </div>
          <button class="doc-dl-btn" @click.stop="downloadDoc(d.id)">下载</button>
          <button class="doc-view-btn" @click.stop="previewDoc(d)">查看</button>
        </div>
        <div v-if="docs.length === 0" class="empty-hint">暂无文档</div>
      </nav>
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

    <div v-if="preview.show" class="preview-overlay" @click.self="closePreview">
      <div class="preview-card">
        <div class="preview-header">
          <span class="preview-title" :title="preview.doc?.title">{{ preview.doc?.title || '文档预览' }}</span>
          <div class="preview-actions">
            <button class="preview-dl-btn" @click="downloadDoc(preview.doc?.id)">下载</button>
            <button class="preview-close-btn" @click="closePreview">✕</button>
          </div>
        </div>
        <div class="preview-body">
          <div v-if="preview.loading" class="preview-hint">加载中...</div>
          <div v-else-if="preview.error" class="preview-hint preview-error">{{ preview.error }}</div>
          <iframe v-else-if="preview.kind === 'pdf'" class="preview-frame" :src="preview.url"></iframe>
          <img v-else-if="preview.kind === 'image'" class="preview-img" :src="preview.url" alt="文档图片" />
          <div v-else-if="preview.kind === 'text'" class="preview-text">{{ preview.text }}</div>
          <div v-else class="preview-hint">
            该格式暂不支持在线预览,请下载后查看
            <br /><button class="preview-dl-btn" @click="downloadDoc(preview.doc?.id)">下载文档</button>
          </div>
        </div>
      </div>
    </div>

    <AddFriendDialog v-if="showAddFriend" @close="showAddFriend = false; loadFriends()" />
  </aside>
</template>

<script setup lang="ts">
import { ref, computed, nextTick } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { getFriends, listDocuments, listOrgStaff, grantFriend } from '../api'
import AddFriendDialog from './AddFriendDialog.vue'

const auth = useAuthStore()
const chat = useChatStore()
const search = ref('')
const activeTab = ref('msg')
const docs = ref<any[]>([])

const friends = ref<any[]>([])
const friendTab = ref<'human' | 'device' | 'agent'>('human')
const filteredFriends = computed(() => {
  return friends.value.filter((f: any) => {
    const t = f.user_type || 'human'
    if (friendTab.value === 'human') return t === 'human' || t === 'api'
    return t === friendTab.value
  })
})
const nonAgentFriends = computed(() => friends.value)
const showAddFriend = ref(false)
const showProfile = ref(false)
const newNickname = ref('')
const profileMsg = ref('')
const staff = ref<any[]>([])
const grant = ref({ show: false, friend: null as any, checked: [] as string[], loading: false, msg: '' })

async function openGrant(f: any) {
  grant.value = { show: true, friend: f, checked: [], loading: false, msg: '' }
  try {
    const res = await listOrgStaff()
    staff.value = Array.isArray(res) ? res : []
  } catch (e) {
    staff.value = []
    grant.value.msg = '加载员工列表失败'
  }
}

async function doGrant() {
  if (!grant.value.friend || grant.value.checked.length === 0) {
    grant.value.msg = '请选择员工'
    return
  }
  grant.value.loading = true
  try {
    await grantFriend(grant.value.friend.friend_id, grant.value.checked)
    grant.value.msg = '授权成功'
    setTimeout(() => { grant.value.show = false }, 800)
  } catch (e: any) {
    grant.value.msg = '授权失败: ' + (e?.response?.data?.error || e?.message || e)
  } finally {
    grant.value.loading = false
  }
}

// ── Document preview ──
const preview = ref({ show: false, doc: null as any, url: '', kind: '', text: '', loading: false, error: '' })

function previewKind(d: any): string {
  const name = (d.filename || d.title || '').toLowerCase()
  if (name.endsWith('.pdf')) return 'pdf'
  if (/\.(png|jpe?g|gif|webp|bmp|svg)$/.test(name)) return 'image'
  if (/\.(txt|md|markdown|json|log|csv)$/.test(name)) return 'text'
  return 'unsupported'
}

async function previewDoc(d: any) {
  const token = localStorage.getItem('token') || ''
  const kind = previewKind(d)
  preview.value = { show: true, doc: d, url: '', kind, text: '', loading: kind !== 'unsupported', error: '' }
  if (kind === 'unsupported') return
  try {
    const res = await fetch('/api/documents/' + d.id + '/preview', {
      headers: { Authorization: 'Bearer ' + token }
    })
    if (!res.ok) { preview.value.error = '预览失败: HTTP ' + res.status; preview.value.loading = false; return }
    const blob = await res.blob()
    if (kind === 'text') {
      preview.value.text = await blob.text()
    } else {
      preview.value.url = URL.createObjectURL(blob)
    }
    preview.value.loading = false
  } catch (e) {
    preview.value.error = '预览失败: ' + e
    preview.value.loading = false
  }
}

function closePreview() {
  if (preview.value.url) URL.revokeObjectURL(preview.value.url)
  preview.value = { show: false, doc: null, url: '', kind: '', text: '', loading: false, error: '' }
}

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

async function loadDocs() {
  try {
    const res = await listDocuments(50)
    docs.value = Array.isArray(res) ? res : []
  } catch (e) {
    docs.value = []
  }
}

async function downloadDoc(id: string) {
  const token = localStorage.getItem('token') || ''
  try {
    const res = await fetch('/api/documents/' + id + '/download', {
      headers: { Authorization: 'Bearer ' + token }
    })
    if (!res.ok) { alert('下载失败: HTTP ' + res.status); return }
    const blob = await res.blob()
    let filename = 'doc-' + id
    const cd = res.headers.get('Content-Disposition') || ''
    const m = cd.match(/filename\*=UTF-8''([^;]+)/)
    if (m) filename = decodeURIComponent(m[1])
    else { const m2 = cd.match(/filename="([^"]+)"/); if (m2) filename = m2[1] }
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url; a.download = filename
    document.body.appendChild(a); a.click(); document.body.removeChild(a)
    setTimeout(() => URL.revokeObjectURL(url), 5000)
  } catch (e) { alert('下载失败: ' + e) }
}

function formatSize(bytes: number) {
  if (!bytes && bytes !== 0) return ''
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / 1024 / 1024).toFixed(1) + ' MB'
}

function formatTime(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' }) + ' ' +
    d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
}

async function loadFriends() {
  // Resolve the numeric uid first. The old code read auth.user synchronously, but
  // whoami runs async in ChatView's onMounted AFTER this component's setup, so the
  // fallback {username:'用户'} won the race and we queried /friends/用户 (empty).
  let userId = ''
  const token = localStorage.getItem('token') || ''
  if (token.startsWith('session-')) {
    userId = token.slice(8)
  } else {
    if (!auth.userInfo && token) {
      try { await auth.fetchWhoAmI() } catch (e) { /* keep fallback */ }
    }
    userId = auth.userInfo?.user_id || auth.user?.bot_id || auth.user?.username || ''
  }
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

.friend-tabs {
  display: flex;
  border-bottom: 1px solid #e8e8e8;
  background: #fafafa;
}
.ftab {
  flex: 1;
  text-align: center;
  padding: 7px 0;
  font-size: 12px;
  color: #666;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  user-select: none;
}
.ftab.active {
  color: #1890ff;
  font-weight: 600;
  border-bottom-color: #1890ff;
  background: #fff;
}
.grant-btn {
  flex-shrink: 0;
  font-size: 11px;
  padding: 2px 8px;
  border: 1px solid #1890ff;
  border-radius: 10px;
  background: #fff;
  color: #1890ff;
  cursor: pointer;
}
.grant-btn:hover { background: #e6f7ff; }

.grant-card { width: 320px; text-align: left; }
.grant-card h3 { font-size: 15px; margin: 0 0 8px; }
.grant-target { font-size: 12px; color: #888; margin: 0 0 12px; }
.grant-list {
  max-height: 240px;
  overflow-y: auto;
  border: 1px solid #eee;
  border-radius: 4px;
  padding: 4px 8px;
  margin-bottom: 12px;
}
.grant-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
  font-size: 13px;
  cursor: pointer;
}
.grant-item em { font-style: normal; color: #999; font-size: 12px; }
.grant-actions { display: flex; gap: 8px; }
.grant-actions .save-btn { margin-top: 0; }
.grant-actions .close-btn { margin-top: 0; }

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

.doc-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  font-size: 13px;
  color: #666;
}
.doc-list-title { font-weight: 600; }
.refresh-btn {
  border: none;
  background: none;
  font-size: 15px;
  cursor: pointer;
  color: #1890ff;
}
.doc-name {
  max-width: 150px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.doc-dl-btn {
  flex-shrink: 0;
  margin-left: 6px;
  padding: 2px 10px;
  font-size: 12px;
  border: 1px solid #1890ff;
  color: #1890ff;
  background: #fff;
  border-radius: 4px;
  cursor: pointer;
}
.doc-dl-btn:hover { background: #e6f7ff; }
.doc-view-btn {
  flex-shrink: 0;
  margin-left: 4px;
  padding: 2px 10px;
  font-size: 12px;
  border: 1px solid #52c41a;
  color: #52c41a;
  background: #fff;
  border-radius: 4px;
  cursor: pointer;
}
.doc-view-btn:hover { background: #f6ffed; }

.preview-overlay {
  position: fixed; top: 0; left: 0; width: 100%; height: 100%;
  background: rgba(0,0,0,.5); z-index: 2000;
  display: flex; align-items: center; justify-content: center;
}
.preview-card {
  background: #fff; border-radius: 8px;
  width: min(900px, 92vw); height: min(720px, 88vh);
  display: flex; flex-direction: column; overflow: hidden;
}
.preview-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 14px; border-bottom: 1px solid #f0f0f0;
}
.preview-title {
  font-size: 14px; font-weight: 600;
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  flex: 1; min-width: 0;
}
.preview-actions { display: flex; gap: 8px; flex-shrink: 0; }
.preview-dl-btn {
  padding: 3px 12px; font-size: 12px;
  border: 1px solid #1890ff; color: #1890ff; background: #fff;
  border-radius: 4px; cursor: pointer;
}
.preview-dl-btn:hover { background: #e6f7ff; }
.preview-close-btn {
  padding: 3px 10px; font-size: 13px;
  border: 1px solid #d9d9d9; color: #666; background: #fff;
  border-radius: 4px; cursor: pointer;
}
.preview-close-btn:hover { background: #f0f0f0; }
.preview-body { flex: 1; min-height: 0; background: #fafafa; display: flex; flex-direction: column; }
.preview-frame { width: 100%; height: 100%; border: none; flex: 1; }
.preview-img { max-width: 100%; max-height: 100%; object-fit: contain; margin: auto; display: block; }
.preview-text {
  flex: 1; overflow: auto; margin: 12px; padding: 16px;
  background: #fff; border: 1px solid #eee; border-radius: 4px;
  font-size: 13px; line-height: 1.7; white-space: pre-wrap;
  word-break: break-word; font-family: 'Microsoft YaHei', sans-serif;
}
.preview-hint {
  margin: auto; text-align: center; color: #888; font-size: 13px;
  line-height: 2.2;
}
.preview-error { color: #f5222d; }
</style>

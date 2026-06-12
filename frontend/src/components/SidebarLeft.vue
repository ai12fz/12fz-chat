<template>
  <aside id="chat-side">
    <!-- 顶部Tab: 💬 消息 / 👤 通讯录 (老ERP风格) -->
    <ul class="chat-side1">
      <li :class="{ active: activeTab === 'msg' }" @click="activeTab = 'msg'" title="消息">
        <i class="fa fa-comments"></i>
      </li>
      <li :class="{ active: activeTab === 'contact' }" @click="activeTab = 'contact'" title="通讯录">
        <i class="fa fa-address-book"></i>
      </li>
    </ul>

    <!-- 搜索栏 -->
    <div class="chat-search">
      <select v-if="activeTab === 'msg'" v-model="searchType">
        <option value="userNick">昵称</option>
        <option value="remark">备注</option>
        <option value="name">姓名</option>
        <option value="phone">手机号</option>
      </select>
      <input
        v-model="keyword"
        type="text"
        :placeholder="activeTab === 'contact' ? '按Enter键搜索群名称' : '按Enter键搜索'"
        @keyup.enter="doSearch"
      />
    </div>

    <!-- 子Tab (仅消息模式有: 待处理/已处理) -->
    <div class="chatside-wrap" v-if="activeTab === 'msg'">
      <ul class="side_ul">
        <li :class="{ active: subTab === 'pending' }" @click="subTab = 'pending'">待处理</li>
        <li :class="{ active: subTab === 'done' }" @click="subTab = 'done'">已处理</li>
      </ul>
      <div class="chatside_list">
        <div class="chatside_inner">
          <div
            v-for="s in pendingList"
            :key="s.id"
            class="session-item"
            :class="{ active: s.id === chat.activeId }"
            @click="chat.setActive(s.id)"
          >
            <span class="session-avatar">{{ s.name[0] }}</span>
            <div class="session-info">
              <div class="session-top">
                <span class="session-name text-overflow">{{ s.name }}</span>
                <span v-if="s.type === 'user' && isBot(s.name)" class="bot-tag">🤖</span>
                <span class="session-type-tag">{{ s.type === 'group' ? '群' : '友' }}</span>
              </div>
              <span class="session-last-msg text-overflow">{{ s.lastMsg || '暂无消息' }}</span>
            </div>
            <div class="session-right">
              <span v-if="s.unread > 0" class="unread-badge">{{ s.unread > 99 ? '99+' : s.unread }}</span>
              <span v-else-if="s.lastMsgAt" class="session-time">{{ formatAgo(s.lastMsgAt) }}</span>
            </div>
          </div>
          <div
            v-for="s in doneList"
            :key="s.id"
            class="session-item"
            :class="{ active: s.id === chat.activeId }"
            @click="chat.setActive(s.id)"
          >
            <span class="session-avatar">{{ s.name[0] }}</span>
            <div class="session-info">
              <div class="session-top">
                <span class="session-name text-overflow">{{ s.name }}</span>
                <span v-if="s.type === 'user' && isBot(s.name)" class="bot-tag">🤖</span>
                <span class="session-type-tag">{{ s.type === 'group' ? '群' : '友' }}</span>
              </div>
              <span class="session-last-msg text-overflow">{{ s.lastMsg || '暂无消息' }}</span>
            </div>
            <div class="session-right">
              <span v-if="s.lastMsgAt" class="session-time">{{ formatAgo(s.lastMsgAt) }}</span>
            </div>
          </div>
          <div v-if="subTab === 'pending' && pendingList.length === 0" class="empty-hint">暂无待处理</div>
          <div v-if="subTab === 'done' && doneList.length === 0" class="empty-hint">暂无已处理</div>
        </div>
      </div>
    </div>

    <!-- 通讯录模式 -->
    <div class="chatside-wrap" v-else>
      <ul class="side_ul">
        <li :class="{ active: subTab === 'friends' }" @click="subTab = 'friends'">好友</li>
        <li :class="{ active: subTab === 'groups' }" @click="subTab = 'groups'">我的群</li>
      </ul>
      <div class="chatside_list">
        <div class="chatside_inner">
          <div v-if="subTab === 'groups'">
            <div
              v-for="s in chat.sessions"
              :key="s.id"
              class="session-item"
              :class="{ active: s.id === chat.activeId }"
              @click="chat.setActive(s.id)"
            >
              <span class="session-avatar">{{ s.name[0] }}</span>
              <div class="session-info">
                <div class="session-top">
                  <span class="session-name text-overflow">{{ s.name }}</span>
                  <span v-if="isBot(s.name)" class="bot-tag">🤖</span>
                  <span class="session-type-tag">群</span>
                </div>
                <span class="session-last-msg text-overflow">{{ s.lastMsg || '暂无消息' }}</span>
              </div>
            </div>
          </div>
          <div v-else>
            <div
              v-for="f in friendsList"
              :key="f.id"
              class="session-item"
              @click="openFriendChat(f)"
            >
              <span class="session-avatar">{{ f.name[0] }}</span>
              <div class="session-info">
                <div class="session-top">
                  <span class="session-name text-overflow">{{ f.name }}</span>
                  <span v-if="isBot(f.name)" class="bot-tag">🤖</span>
                  <span class="session-type-tag">友</span>
                </div>
              </div>
            </div>
            <div v-if="friendsList.length === 0" class="empty-hint">暂无好友</div>
            <!-- 添加好友 -->
            <div class="add-friend-bar">
              <input
                v-model="newFriendName"
                placeholder="输入用户名添加好友"
                @keyup.enter="handleAddFriend"
              />
              <button @click="handleAddFriend">添加</button>
            </div>
          </div>
          <div v-if="subTab === 'groups' && filteredSessions.length === 0" class="empty-hint">暂无群聊</div>
        </div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useChatStore } from '../stores/chat'
import { useAuthStore } from '../stores/auth'
import { getFriends, addFriend } from '../api'

const chat = useChatStore()
const auth = useAuthStore()

// Bots list — show 🤖 icon next to these
const botNames = ['chaogu-ai', '服务器技术', '高级工程师']
function isBot(name: string) { return botNames.includes(name) }

// Friends list
const friendsList = ref<{ id: string; name: string }[]>([])
const newFriendName = ref('')

async function loadFriends() {
  const username = auth.user?.username
  if (!username) return
  try {
    const res = await getFriends(username)
    const arr = res || []
    const accepted = Array.isArray(arr) ? arr.filter((f: { status?: string }) => f.status === 'accepted' || f.status === 'online' || f.status === 'pending') : []
    friendsList.value = accepted.map((f: { friend_id: string }) => ({
      id: f.friend_id,
      name: f.friend_id,
    }))
  } catch (e) {
    console.error('Failed to load friends', e)
  }
}

async function handleAddFriend() {
  const name = newFriendName.value.trim()
  if (!name) return
  const username = auth.user?.username
  if (!username) return
  try {
    await addFriend(username, name)
    newFriendName.value = ''
    await loadFriends()
  } catch (e: any) {
    alert(e?.response?.data?.error || '添加好友失败')
  }
}

onMounted(loadFriends)

function openFriendChat(f: { id: string; name: string }) {
  const id = `user:${f.id}`
  // Find existing session or create one
  let s = chat.sessions.find(s => s.id === id)
  if (!s) {
    s = {
      id,
      name: f.name,
      type: 'user',
      unread: 0,
      messages: [],
    }
    chat.addSession(s)
  }
  chat.setActive(id)
  // 自动跳转到消息Tab待处理状态
  activeTab.value = 'msg'
  subTab.value = 'pending'
}

const activeTab = ref<'msg' | 'contact'>('msg')
const subTab = ref<'pending' | 'done' | 'friends' | 'groups'>('pending')
const keyword = ref('')
const searchType = ref('userNick')

const filteredSessions = computed(() => {
  if (!keyword.value) return chat.sessions
  const q = keyword.value.toLowerCase()
  return chat.sessions.filter(s => s.name.toLowerCase().includes(q))
})

// 待处理：有未读消息，或未标记已处理
const pendingList = computed(() =>
  [...chat.sessions].filter(s => s.unread > 0 || !chat.doneSessions[s.id])
    .sort((a, b) => {
    if (a.lastMsgAt && b.lastMsgAt) return b.lastMsgAt.localeCompare(a.lastMsgAt)
    if (a.lastMsgAt) return -1
    if (b.lastMsgAt) return 1
    return a.name.localeCompare(b.name)
  })
)

// 已处理：手动标记且无新消息
const doneList = computed(() =>
  [...chat.doneSessionList].sort((a, b) => {
    if (a.lastMsgAt && b.lastMsgAt) return b.lastMsgAt.localeCompare(a.lastMsgAt)
    if (a.lastMsgAt) return -1
    if (b.lastMsgAt) return 1
    return a.name.localeCompare(b.name)
  })
)

function doSearch() {
  // Triggered by Enter key - filtering is computed, so just doing a side-effect
  // The computed will auto-update
}

function formatAgo(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  const now = new Date()
  const diff = Math.floor((now.getTime() - d.getTime()) / 60000)
  if (diff < 1) return '刚刚'
  if (diff < 60) return diff + '分钟前'
  if (diff < 1440) return Math.floor(diff / 60) + '小时前'
  return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' })
}
</script>

<style scoped>
#chat-side {
  width: 250px;
  min-width: 250px;
  display: flex;
  flex-direction: column;
  background: transparent;
  overflow: hidden;
  flex-shrink: 0;
}

/* Tab bar */
.chat-side1 {
  display: flex;
  padding-top: 10px;
  list-style: none;
}
.chat-side1 > li {
  width: 50%;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: rgba(255,255,255,0.8);
  border-top-left-radius: 4px;
  border-top-right-radius: 4px;
  font-size: 20px;
  cursor: pointer;
}
.chat-side1 > li.active {
  background-color: rgba(255,255,255,.5);
  color: rgba(0,0,0,.6);
}

/* Search */
.chat-search {
  display: flex;
  padding: 10px 0;
  gap: 4px;
}
.chat-search select {
  background: #fff;
  padding-left: 4px;
  height: 30px;
  line-height: 30px;
  width: 30%;
  border: 0;
  border-radius: 4px;
  border-top-right-radius: 0;
  border-bottom-right-radius: 0;
  font-size: 14px;
  outline: none;
  cursor: pointer;
}
.chat-search input {
  flex: 1;
  height: 30px;
  line-height: 30px;
  border: 0;
  border-radius: 4px;
  border-top-left-radius: 0;
  border-bottom-left-radius: 0;
  padding-left: 6px;
  font-size: 15px;
  outline: none;
}

/* Sub tabs + list area */
.chatside-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.side_ul {
  display: flex;
  list-style: none;
  margin-bottom: 6px;
  gap: 0;
}
.side_ul > li {
  flex: 1;
  border-top: 1px solid rgba(255,255,255,0.2);
  border-bottom: 1px solid rgba(255,255,255,0.2);
  cursor: pointer;
  font-size: 15px;
  color: #fff;
  height: 30px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.side_ul > li.active {
  background-color: rgba(255,255,255,.8);
  color: #555;
}

.chatside_list {
  flex: 1;
  background-color: rgba(255,255,255,.1);
  border-radius: 4px;
  overflow: hidden;
}
.chatside_inner {
  height: 100%;
  overflow-y: auto;
  padding: 4px 0;
}

/* Session items */
.session-item {
  display: flex;
  align-items: center;
  padding: 8px 10px;
  cursor: pointer;
  gap: 8px;
  border-top: 1px solid rgba(255,255,255,0.05);
  transition: background .15s;
}
.session-item:hover { background: rgba(255,255,255,.1); }
.session-item.active { background: rgba(255,255,255,.25); }

.session-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: rgba(255,255,255,.25);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  flex-shrink: 0;
}

.session-info {
  flex: 1;
  min-width: 0;
}
.session-top {
  display: flex;
  align-items: center;
  gap: 4px;
}
.session-name {
  font-size: 15px;
  color: #fff;
  max-width: 120px;
}
.bot-tag {
  font-size: 14px;
  line-height: 1;
}
.session-type-tag {
  font-size: 12px;
  background: rgba(255,255,255,.2);
  color: rgba(255,255,255,.8);
  padding: 0 4px;
  border-radius: 2px;
  flex-shrink: 0;
}
.session-last-msg {
  font-size: 13px;
  color: rgba(255,255,255,.6);
  margin-top: 2px;
  max-width: 130px;
}

.session-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  flex-shrink: 0;
}
.session-time {
  font-size: 12px;
  color: rgba(255,255,255,.5);
  white-space: nowrap;
}
.unread-badge {
  background: #f5222d;
  color: #fff;
  font-size: 12px;
  min-width: 20px;
  height: 20px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2px 6px;
}

.empty-hint {
  text-align: center;
  color: rgba(255,255,255,.5);
  padding: 32px 16px;
  font-size: 15px;
}

/* 添加好友栏 */
.add-friend-bar {
  display: flex;
  padding: 8px 10px;
  gap: 6px;
  border-top: 1px solid rgba(255,255,255,.1);
}
.add-friend-bar input {
  flex: 1;
  height: 28px;
  border: 1px solid rgba(255,255,255,.2);
  border-radius: 3px;
  padding: 0 6px;
  font-size: 13px;
  background: rgba(255,255,255,.15);
  color: #fff;
  outline: none;
}
.add-friend-bar input::placeholder {
  color: rgba(255,255,255,.4);
}
.add-friend-bar button {
  height: 28px;
  padding: 0 10px;
  background: rgba(45,108,240,.8);
  color: #fff;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  font-size: 13px;
  white-space: nowrap;
}
.add-friend-bar button:hover {
  background: rgba(45,108,240,1);
}
</style>

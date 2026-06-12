<template>
  <div class="chat-layout">
    <!-- 顶部导航栏 (old ERP style: rgba(39,53,131,.6), 50px) -->
    <div id="chat-nav">
      <ul class="chat-nav-l">
        <li class="nav-avatar" @click="handleLogout" :title="'点击退出：' + displayName">
          <span class="nav-avatar-circle">{{ displayName[0] }}</span>
        </li>
        <li><span class="nav-username">{{ displayName }}</span></li>
        <li>
          <select class="nav-state-select" v-model="onlineState">
            <option value="1">在线</option>
            <option value="2">忙碌</option>
            <option value="3">隐身</option>
          </select>
        </li>
        <li><i class="fa fa-bell nav-bell" :class="bellOn ? 'bell-active' : 'bell-muted'" @click="bellOn = !bellOn"></i></li>
      </ul>
      <ul class="chat-nav-r">
        <li v-for="item in navIcons" :key="item.label" class="nav-icon-item" :title="item.label">
          <div class="nav-icon-wrap">
            <i class="fa" :class="item.icon"></i>
            <span>{{ item.label }}</span>
          </div>
        </li>
      </ul>
    </div>

    <!-- 主体：侧栏 (28%) + 内容 (72%) -->
    <div class="chat-body">
      <SidebarLeft />
      <ChatContent />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { useWebSocket } from '../composables/useWebSocket'
import { getMyGroups, getUnreadCounts, getMessages, markRead } from '../api'
import SidebarLeft from '../components/SidebarLeft.vue'
import ChatContent from '../components/ChatContent.vue'

const router = useRouter()
const auth = useAuthStore()
const chat = useChatStore()
const ws = useWebSocket()

const displayName = computed(() => auth.user?.username || auth.user?.bot_id || '用户')
const bellOn = ref(true)
const onlineState = ref('1')

// 导航栏右侧图标 (匹配老ERP: 设置中心/工单管理/统计报表)
const navIcons = [
  { icon: 'fa-cog', label: '设置' },
  { icon: 'fa-tasks', label: '工单' },
  { icon: 'fa-bar-chart', label: '统计' },
]

function handleLogout() {
  auth.logout()
  router.push('/login')
}

onMounted(async () => {
  if (!auth.token) {
    router.push('/login')
    return
  }
  ws.connect(auth.token)
  try {
    const groups = await getMyGroups()
    groups.forEach((g: any) => chat.ensureGroupSession(g))
    try {
      const unreads = await getUnreadCounts()
      for (const [groupIdStr, count] of Object.entries(unreads)) {
        const sid = chat.groupSessionId(Number(groupIdStr))
        const s = chat.sessions.find(s => s.id === sid)
        if (s) s.unread = count as number
      }
    } catch { /* ok */ }
    if (chat.sessions.length > 0) chat.setActive(chat.sessions[0].id)
  } catch (err) {
    console.error('Failed to load chat data:', err)
  }
})

// 切换会话时加载历史消息 + 标记已读
watch(() => chat.activeId, async (newId) => {
  if (!newId) return
  const match = newId.match(/^group:(\d+)$/)
  if (!match) return
  const groupId = parseInt(match[1])
  try {
    const msgs = await getMessages(groupId, 20, 0)
    chat.loadMessages(newId, msgs)
    // 标记已读：取最新消息ID
    if (msgs.length > 0) {
      const lastId = msgs[msgs.length - 1].id
      markRead(groupId, lastId).catch(() => {/* non-critical */})
    }
  } catch (err) {
    console.error('Failed to load messages:', err)
  }
})
</script>

<style scoped>
.chat-layout {
  display: flex;
  flex-direction: column;
  height: 100vh;
  background: #05aeee;
  overflow: hidden;
}

/* ===== 顶部导航栏 ===== */
#chat-nav {
  background-color: rgba(39, 53, 131, 0.6);
  height: 50px;
  padding: 0 10px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  z-index: 10;
}

#chat-nav > ul {
  display: flex;
  align-items: center;
  list-style: none;
}

.chat-nav-l {
  gap: 4px;
}
.chat-nav-l > li {
  display: flex;
  align-items: center;
}

.nav-avatar-circle {
  display: inline-flex;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: rgba(255,255,255,.3);
  color: #fff;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  cursor: pointer;
  transition: .2s;
}
.nav-avatar-circle:hover { background: rgba(255,255,255,.5); }

.nav-username {
  color: #fff;
  padding: 0 10px;
  cursor: pointer;
  font-size: 16px;
}

.nav-state-select {
  background: transparent;
  color: rgba(255,255,255,.85);
  border: 1px solid rgba(255,255,255,.3);
  border-radius: 3px;
  padding: 2px 4px;
  font-size: 12px;
  cursor: pointer;
  outline: none;
}
.nav-state-select option { color: #333; background: #fff; }

.nav-bell {
  font-size: 16px;
  margin-left: 8px;
  cursor: pointer;
  padding: 4px;
}
.bell-active { color: #e4ac40; }
.bell-muted { color: rgba(255,255,255,.6); }

.chat-nav-r {
  gap: 2px;
}
.nav-icon-item {
  width: 50px;
  height: 50px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}
.nav-icon-wrap {
  display: flex;
  flex-direction: column;
  align-items: center;
  color: rgba(255,255,255,.8);
  transform: scale(.8);
}
.nav-icon-wrap > i { font-size: 22px; }
.nav-icon-wrap > span { font-size: 13px; white-space: nowrap; }
.nav-icon-item:hover .nav-icon-wrap { color: #fff; }
.nav-icon-item:hover .nav-icon-wrap > i { transform: scale(1.1); }

/* ===== 主体 ===== */
.chat-body {
  flex: 1;
  display: flex;
  overflow: hidden;
  padding: 0 10px 10px;
  gap: 4px;
}
</style>

<template>
  <div class="sidebar-right-wrapper">
    <!-- 折叠时的切换按钮 -->
    <div v-if="!expanded && isAgentChat" class="collapsed-tab" @click="togglePanel">
      <span class="toggle-icon">◀</span>
      <span class="toggle-text">状态</span>
    </div>

    <!-- 展开的侧栏 -->
    <aside v-if="expanded" class="sidebar-right">
      <div class="panel-header">
        <div class="header-row">
          <h3>{{ session?.name || '详情' }}</h3>
          <button class="collapse-btn" @click="togglePanel" title="折叠">▶</button>
        </div>
      </div>

      <!-- Agent 状态面板 -->
      <div v-if="isAgentChat" class="panel-body">
        <div class="status-card" :class="agentStatus.status">
          <div class="status-dot"></div>
          <span class="status-text">{{ statusText }}</span>
        </div>

        <div class="info-section" v-if="agentStatus.current_task_title">
          <div class="info-label">当前任务</div>
          <div class="info-value">{{ agentStatus.current_task_title }}</div>
        </div>

        <div class="info-section" v-if="agentStatus.message">
          <div class="info-label">状态消息</div>
          <div class="info-value">{{ agentStatus.message }}</div>
        </div>

        <div class="info-section">
          <div class="info-label">最后活动</div>
          <div class="info-value">{{ formatTime(agentStatus.updated_at || agentStatus.heartbeat_at) }}</div>
        </div>
      </div>

      <!-- 普通会话信息 -->
      <div v-else class="panel-body">
        <div class="info-section">
          <div class="info-row">
            <span class="label">类型</span>
            <span>{{ session?.type === 'group' ? '群聊' : '好友' }}</span>
          </div>
        </div>

        <div class="member-section" v-if="session?.members?.length">
          <div class="section-title">群成员（{{ session.members.length }}）</div>
          <div v-for="m in session.members" :key="m.bot_id" class="member-item">
            <span class="member-avatar" :style="{ background: nameColor(m.bot_id) }">{{ m.bot_id[0] }}</span>
            <span class="member-name">{{ m.bot_id }}</span>
            <span v-if="m.role === 'admin'" class="member-badge">群主</span>
          </div>
        </div>
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onUnmounted } from 'vue'
import { useChatStore } from '../stores/chat'

const chat = useChatStore()
const session = computed(() => chat.activeSession)
const expanded = ref(true)
const agentStatus = ref<any>({ status: 'offline' })
let pollTimer: ReturnType<typeof setInterval> | null = null

const isAgentChat = computed(() => {
  const s = session.value
  if (!s) return false
  // Friend chats where the friend is a bot agent
  return s.id.startsWith('friend:') && isBotAgent(s.name)
})

function isBotAgent(name: string) {
  // Known bot agents
  const bots = ['chaogu-ai', 'gong3', '服务器技术', '高级工程师']
  return bots.includes(name)
}

const statusText = computed(() => {
  const map: Record<string, string> = {
    online: '在线',
    busy: '忙碌中',
    offline: '离线',
    idle: '空闲'
  }
  return map[agentStatus.value.status] || agentStatus.value.status || '未知'
})

function togglePanel() {
  expanded.value = !expanded.value
  if (expanded.value) {
    startPolling()
  } else {
    stopPolling()
  }
}

async function fetchStatus() {
  if (!session.value || !isAgentChat.value) return
  const botId = session.value.name
  const token = localStorage.getItem('token') || ''
  try {
    const resp = await fetch(`/api/agent-status?bot_id=${encodeURIComponent(botId)}`, {
      headers: { 'Authorization': `Bearer ${token}` }
    })
    if (resp.ok) {
      agentStatus.value = await resp.json()
    }
  } catch { /* ignore */ }
}

function startPolling() {
  stopPolling()
  fetchStatus()
  pollTimer = setInterval(fetchStatus, 5000)
}

function stopPolling() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function formatTime(t: string) {
  if (!t) return '--'
  const d = new Date(t)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return `${Math.floor(diff / 60000)} 分钟前`
  return d.toLocaleString('zh-CN', { hour: '2-digit', minute: '2-digit', month: 'short', day: 'numeric' })
}

function nameColor(name: string) {
  const colors = ['#1890ff', '#52c41a', '#fa8c16', '#eb2f96', '#722ed1', '#13c2c2', '#f5222d']
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash)
  return colors[Math.abs(hash) % colors.length]
}

// Auto-start when session changes to agent
watch(() => session.value?.id, (newId, oldId) => {
  if (newId !== oldId) {
    if (isAgentChat.value && expanded.value) {
      startPolling()
    } else {
      stopPolling()
    }
  }
}, { immediate: true })

onUnmounted(() => stopPolling())
</script>

<style scoped>
.sidebar-right-wrapper {
  position: relative;
  flex-shrink: 0;
}

.collapsed-tab {
  width: 32px;
  height: 100vh;
  background: #fafafa;
  border-left: 1px solid #e8e8e8;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  writing-mode: vertical-rl;
  color: #999;
  font-size: 12px;
  transition: background .15s;
}
.collapsed-tab:hover { background: #f0f0f0; color: #1890ff; }
.toggle-icon { font-size: 14px; }
.toggle-text { letter-spacing: 2px; }

.sidebar-right {
  width: 260px;
  border-left: 1px solid #e8e8e8;
  background: #fafafa;
  flex-shrink: 0;
  overflow-y: auto;
  height: 100vh;
}
.panel-header {
  padding: 10px 16px;
  border-bottom: 1px solid #e8e8e8;
}
.header-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.header-row h3 { margin: 0; font-size: 15px; font-weight: 600; }
.collapse-btn {
  background: none;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  cursor: pointer;
  padding: 2px 8px;
  font-size: 12px;
  color: #999;
}
.collapse-btn:hover { background: #f0f0f0; }
.panel-body { padding: 0; }

/* Status card */
.status-card {
  margin: 12px 16px;
  padding: 10px 14px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 500;
}
.status-card.online { background: #f6ffed; border: 1px solid #b7eb8f; color: #52c41a; }
.status-card.busy { background: #fff7e6; border: 1px solid #ffd591; color: #fa8c16; }
.status-card.offline { background: #f5f5f5; border: 1px solid #d9d9d9; color: #999; }
.status-card.idle { background: #e6f7ff; border: 1px solid #91d5ff; color: #1890ff; }
.status-dot {
  width: 8px; height: 8px;
  border-radius: 50%;
  background: currentColor;
}

.info-section { padding: 10px 16px; border-bottom: 1px solid #f0f0f0; }
.info-label { font-size: 12px; color: #888; margin-bottom: 4px; }
.info-value { font-size: 13px; color: #333; word-break: break-all; }
.info-row {
  display: flex;
  justify-content: space-between;
  padding: 4px 0;
  font-size: 13px;
}
.label { color: #888; }

.member-section { padding: 12px 16px; }
.section-title {
  font-size: 13px;
  font-weight: 500;
  color: #555;
  margin-bottom: 8px;
  padding-bottom: 6px;
  border-bottom: 1px solid #f0f0f0;
}
.member-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 0;
}
.member-avatar {
  width: 28px; height: 28px;
  border-radius: 50%;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 11px;
  flex-shrink: 0;
}
.member-name { font-size: 13px; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.member-badge {
  font-size: 10px;
  background: #e6f7ff;
  color: #1890ff;
  padding: 1px 5px;
  border-radius: 3px;
  flex-shrink: 0;
}
</style>

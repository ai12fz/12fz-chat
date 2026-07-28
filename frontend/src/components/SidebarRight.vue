<template>
  <div class="sidebar-right-wrapper">
    <div v-if="!expanded && isDeviceChat" class="collapsed-tab" @click="togglePanel">
      <span class="toggle-icon">◀</span>
      <span class="toggle-text">状态</span>
    </div>

    <aside v-if="expanded" class="sidebar-right">
      <div class="panel-header">
        <div class="header-row">
          <h3>🤖 {{ deviceName }} 实时工作</h3>
          <button class="collapse-btn" @click="togglePanel" title="折叠">▶</button>
        </div>
      </div>

      <div v-if="isDeviceChat" class="panel-body">
        <div v-if="statuses.length === 0" class="empty-hint">等待任务...</div>
        <div class="timeline">
          <div v-for="(item, idx) in statuses" :key="idx" class="tl-item" :class="item.phase">
            <span class="tl-icon">{{ iconFor(item) }}</span>
            <div class="tl-content">
              <span class="tl-tool">{{ item.tool }}</span>
              <span class="tl-detail">{{ item.detail?.slice(0, 100) }}</span>
              <span class="tl-time">{{ fmtMs(item.time) }}</span>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="panel-body">
        <div class="info-section">
          <div class="info-row">
            <span class="label">类型</span>
            <span>{{ session?.type === 'group' ? '群聊' : '好友' }}</span>
          </div>
        </div>
        <div class="member-section" v-if="session?.members?.length">
          <div class="section-title">群成员（{{ session.members.length }}）</div>
          <div v-for="m in session.members" :key="m.user_id" class="member-item">
            <span class="member-avatar" :style="{ background: nameColor(m.user_id) }">{{ m.user_id[0] }}</span>
            <span class="member-name">{{ m.user_id }}</span>
            <span v-if="m.role === 'admin'" class="member-badge">群主</span>
          </div>
        </div>
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useChatStore } from '../stores/chat'

const chat = useChatStore()
const session = computed(() => chat.activeSession)
const expanded = ref(true)

const deviceName = computed(() => {
  const id = chat.activeId
  if (id && id.startsWith('friend:')) return id.slice(7)
  return null
})

const isDeviceChat = computed(() => {
  const s = session.value
  if (!s) return false
  return s.id.startsWith('friend:') && /^\d+$/.test(s.name)
})

const statuses = computed(() => {
  const id = deviceName.value
  if (!id) return []
  return chat.agentStatuses[id] || []
})

function iconFor(item: any) {
  if (item.phase === 'tool_start') return '⚙'
  if (item.phase === 'tool_end') return '✅'
  return '📌'
}

function fmtMs(ms: number) {
  const d = new Date(ms)
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
}

function togglePanel() {
  expanded.value = !expanded.value
}

function nameColor(name: string) {
  const colors = ['#1890ff', '#52c41a', '#fa8c16', '#eb2f96', '#722ed1', '#13c2c2', '#f5222d']
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash)
  return colors[Math.abs(hash) % colors.length]
}
</script>

<style scoped>
.sidebar-right-wrapper { position: relative; flex-shrink: 0; }
.collapsed-tab {
  width: 32px; height: 100%; background: #fafafa; border-left: 1px solid #e8e8e8;
  cursor: pointer; display: flex; flex-direction: column; align-items: center;
  justify-content: center; gap: 4px; writing-mode: vertical-rl; color: #999; font-size: 12px;
}
.collapsed-tab:hover { background: #f0f0f0; color: #1890ff; }
.toggle-icon { font-size: 14px; }
.toggle-text { letter-spacing: 2px; }
.sidebar-right {
  width: 280px; border-left: 1px solid #e8e8e8; background: #fafafa;
  flex-shrink: 0; overflow-y: auto; height: 100%;
}
.panel-header { padding: 10px 16px; border-bottom: 1px solid #e8e8e8; }
.header-row { display: flex; align-items: center; justify-content: space-between; }
.header-row h3 { margin: 0; font-size: 15px; font-weight: 600; }
.collapse-btn { background: none; border: 1px solid #d9d9d9; border-radius: 4px; cursor: pointer; padding: 2px 8px; font-size: 12px; color: #999; }
.panel-body { padding: 0; }
.empty-hint { padding: 30px; text-align: center; color: #999; font-size: 13px; }
.timeline { flex: 1; overflow-y: auto; padding: 8px; }
.tl-item { display: flex; gap: 6px; padding: 5px 8px; font-size: 12px; border-bottom: 1px solid #f0f0f0; }
.tl-icon { flex-shrink: 0; margin-top: 1px; }
.tl-content { display: flex; flex-direction: column; gap: 1px; flex: 1; min-width: 0; }
.tl-tool { font-weight: 600; color: #333; }
.tl-detail { color: #666; word-break: break-all; font-size: 11px; }
.tl-time { color: #aaa; font-size: 10px; }
.tl-item.tool_start .tl-icon { color: #1890ff; }
.tl-item.tool_end .tl-icon { color: #52c41a; }
.info-section { padding: 10px 16px; border-bottom: 1px solid #f0f0f0; }
.info-row { display: flex; justify-content: space-between; padding: 4px 0; font-size: 13px; }
.label { color: #888; }
.member-section { padding: 12px 16px; }
.section-title { font-size: 13px; font-weight: 500; color: #555; margin-bottom: 8px; padding-bottom: 6px; border-bottom: 1px solid #f0f0f0; }
.member-item { display: flex; align-items: center; gap: 8px; padding: 6px 0; }
.member-avatar { width: 28px; height: 28px; border-radius: 50%; color: #fff; display: flex; align-items: center; justify-content: center; font-size: 11px; flex-shrink: 0; }
.member-name { font-size: 13px; flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.member-badge { font-size: 10px; background: #e6f7ff; color: #1890ff; padding: 1px 5px; border-radius: 3px; flex-shrink: 0; }
</style>
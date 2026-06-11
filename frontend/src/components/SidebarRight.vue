<template>
  <aside class="sidebar-right" v-if="session">
    <div class="panel-header">
      <h3>{{ session.name }}</h3>
    </div>
    <div class="panel-body">
      <!-- Session info -->
      <div class="info-section">
        <div class="info-row">
          <span class="label">类型</span>
          <span>{{ session.type === 'group' ? '群聊' : '好友' }}</span>
        </div>
      </div>

      <!-- Group members -->
      <div class="member-section" v-if="session.members && session.members.length">
        <div class="section-title">
          群成员（{{ session.members.length }}）
        </div>
        <div
          v-for="m in session.members"
          :key="m.bot_id"
          class="member-item"
        >
          <span class="member-avatar" :style="{ background: nameColor(m.bot_id) }">
            {{ m.bot_id[0] }}
          </span>
          <span class="member-name">{{ m.bot_id }}</span>
          <span v-if="m.role === 'admin'" class="member-badge">群主</span>
        </div>
      </div>
      <div class="member-section" v-else-if="session.type === 'group'">
        <div class="section-title">群成员</div>
        <div class="loading-hint">加载中...</div>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '../stores/chat'

const chat = useChatStore()
const session = computed(() => chat.activeSession)

function nameColor(name: string) {
  const colors = ['#1890ff', '#52c41a', '#fa8c16', '#eb2f96', '#722ed1', '#13c2c2', '#f5222d']
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash)
  return colors[Math.abs(hash) % colors.length]
}
</script>

<style scoped>
.sidebar-right {
  width: 260px;
  border-left: 1px solid #e8e8e8;
  background: #fafafa;
  flex-shrink: 0;
  overflow-y: auto;
}
.panel-header {
  padding: 12px 16px;
  border-bottom: 1px solid #e8e8e8;
  background: #fafafa;
}
.panel-header h3 { margin: 0; font-size: 15px; font-weight: 600; }
.panel-body { padding: 0; }

.info-section { padding: 12px 16px; border-bottom: 1px solid #f0f0f0; }
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
.loading-hint { font-size: 12px; color: #bbb; text-align: center; padding: 12px 0; }
</style>

<template>
  <aside class="sidebar-right" v-if="session">
    <div class="panel-header">
      <h3>{{ session.name }}</h3>
    </div>
    <div class="panel-body">
      <div class="info-row">
        <span class="label">类型</span>
        <span>{{ session.type === 'group' ? '群聊' : session.type === 'bot' ? 'AI 客服' : '好友' }}</span>
      </div>
      <div class="info-row" v-if="session.type === 'user'">
        <span class="label">状态</span>
        <span :class="session.online ? 'online' : 'offline'">
          {{ session.online ? '在线' : '离线' }}
        </span>
      </div>
    </div>
  </aside>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '../stores/chat'

const chat = useChatStore()
const session = computed(() => chat.activeSession)
</script>

<style scoped>
.sidebar-right {
  width: 260px;
  border-left: 1px solid #e8e8e8;
  background: #fafafa;
}
.panel-header {
  padding: 12px 16px;
  border-bottom: 1px solid #e8e8e8;
}
.panel-header h3 { margin: 0; font-size: 15px; }
.panel-body { padding: 12px 16px; }
.info-row {
  display: flex;
  justify-content: space-between;
  padding: 8px 0;
  font-size: 13px;
}
.label { color: #888; }
.online { color: #52c41a; }
.offline { color: #888; }
</style>

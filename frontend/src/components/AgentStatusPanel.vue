<template>
  <div class="agent-status-panel" v-if="currentStatuses.length > 0">
    <div class="panel-header">
      <span>🤖 {{ activeDevice }} 工作状态</span>
      <button class="clear-btn" @click="() => clearFromChat(activeDevice)">✕</button>
    </div>
    <div class="timeline">
      <div v-for="(item, idx) in currentStatuses" :key="idx" class="tl-item" :class="item.phase">
        <span class="tl-icon">{{ iconFor(item) }}</span>
        <div class="tl-content">
          <span class="tl-tool">{{ item.tool }}</span>
          <span class="tl-detail">{{ item.detail?.slice(0, 80) }}</span>
        </div>
      </div>
    </div>
  </div>
  <div class="agent-status-panel empty" v-else-if="activeDevice">
    <div class="panel-header"><span>🤖 {{ activeDevice }}</span></div>
    <p class="empty-hint">等待任务...</p>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '../stores/chat'
const chat = useChatStore()
const activeDevice = computed(() => {
  const id = chat.activeId
  if (id.startsWith('friend:')) return id.slice(7)
  return null
})
const currentStatuses = computed(() => {
  if (!activeDevice.value) return []
  return chat.agentStatuses[activeDevice.value] || []
})
function iconFor(item: any) {
  if (item.phase === 'tool_start') return '⚙'
  if (item.phase === 'tool_end') return '✅'
  return '📌'
}
function clearFromChat(device: string) {
  if (device) chat.clearAgentStatus(device)
}
</script>
<style scoped>
.agent-status-panel {
  width: 280px; background: #f8f9fa; border-left: 1px solid #e0e0e0;
  display: flex; flex-direction: column; height: 100%; overflow: hidden;
}
.panel-header { padding: 10px 12px; font-size: 13px; font-weight: 600; border-bottom: 1px solid #e0e0e0; display: flex; justify-content: space-between; align-items: center; background: #fff; }
.clear-btn { background: none; border: none; cursor: pointer; font-size: 14px; color: #999; }
.timeline { flex: 1; overflow-y: auto; padding: 8px; }
.tl-item { display: flex; gap: 6px; padding: 4px 0; font-size: 12px; border-bottom: 1px dashed #e8e8e8; }
.tl-icon { flex-shrink: 0; }
.tl-content { display: flex; flex-direction: column; gap: 1px; }
.tl-tool { font-weight: 600; color: #333; }
.tl-detail { color: #666; word-break: break-all; }
.tl-item.tool_start .tl-icon { color: #1890ff; }
.tl-item.tool_end .tl-icon { color: #52c41a; }
.empty-hint { padding: 20px; text-align: center; color: #999; font-size: 13px; }
.empty { display: flex; justify-content: center; }
</style>

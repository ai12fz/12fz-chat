<template>
  <div style="width:280px;background:#f8f9fa;border-left:1px solid #e0e0e0;padding:10px;height:100%;overflow-y:auto">
    <div style="font-weight:600;margin-bottom:8px">🤖 {{ deviceName }} 工作</div>
    <div v-if="items.length === 0" style="color:#999;font-size:13px">等待任务...</div>
    <div v-for="(it,idx) in items" :key="idx" style="padding:4px 0;font-size:12px;border-bottom:1px dashed #eee">
      <span :style="{color:it.phase==='tool_start'?'#1890ff':'#52c41a'}">{{ it.phase==='tool_start'?'⚙':'✅' }}</span>
      <b>{{ it.tool }}</b>
      <span style="color:#666;margin-left:4px">{{ it.detail?.slice(0,60) }}</span>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useChatStore } from '../stores/chat'
const chat = useChatStore()
const deviceName = computed(() => { const id = chat.activeId; return id?.startsWith('friend:') ? id.slice(7) : '?' })
const items = computed(() => chat.agentStatuses[deviceName.value] || [])
</script>

<template>
  <div style="width:280px;background:#f8f9fa;border-left:1px solid #e0e0e0;padding:10px;height:100%;overflow-y:auto">
    <div style="font-weight:600;margin-bottom:8px">📡 {{ deviceName }} 工作状态</div>
    <div v-if="items.length === 0" style="color:#999;font-size:13px">空闲</div>
    <div v-for="(it,idx) in items" :key="idx" style="padding:4px 0;font-size:12px;border-bottom:1px dashed #eee">
      <span :style="{color:it.p==='s'?'#1890ff':'#52c41a'}">{{ it.p==='s'?'⚙':'✅' }}</span>
      <b>{{ capIcon(it.t) }} {{ capName(it.t) }}</b>
      <span style="color:#666;margin-left:4px">{{ it.d?.slice(0,60) }}</span>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, ref, onMounted } from 'vue'
import { useChatStore } from '../stores/chat'
const chat = useChatStore()

const caps = ref<Record<number, any>>({})

onMounted(async () => {
  try {
    const r = await fetch('/api/capabilities')
    const list = await r.json()
    const m: Record<number, any> = {}
    list.forEach((c: any) => { m[c.id] = c })
    caps.value = m
  } catch(e) { console.error('[panel] caps err', e) }
})

const deviceName = computed(() => {
  const id = chat.activeId
  return id?.startsWith('friend:') ? id.slice(7) : '?'
})
const items = computed(() => chat.agentStatuses[deviceName.value] || [])

function capIcon(id: number) { return caps.value[id]?.icon || '🔧' }
function capName(id: number) { return caps.value[id]?.name || ('#'+id) }
</script>

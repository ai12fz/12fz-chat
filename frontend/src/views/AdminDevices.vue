<template>
  <div class="admin-devices">
    <div class="admin-header">
      <h2>设备管理</h2>
      <div class="admin-nav">
        <router-link to="/admin/devices">设备管理</router-link>
        <router-link to="/admin/proxy">中转站</router-link>
        <router-link to="/admin/agents">Agent管理</router-link>
      </div>
      <div class="header-actions">
        <button class="btn-primary" @click="showAdd = true">+ 添加设备</button>
        <button class="btn-outline" @click="loadData">刷新</button>
      </div>
    </div>

    <div v-if="deviceKey" class="key-card">
      <span>设备注册密钥：<code>{{ deviceKey }}</code></span>
      <button @click="copyKey">复制密钥</button>
    </div>

    <table v-if="devices.length" class="table">
      <thead><tr><th>设备名</th><th>状态</th><th>Token</th><th>系统</th><th>最后上线</th><th>操作</th></tr></thead>
      <tbody>
        <tr v-for="d in devices" :key="d.id">
          <td><span @dblclick="startRename(d)" title="双击改名">{{ d.name }}</span></td>
          <td><span class="tag" :class="d.status">{{ d.status || 'unknown' }}</span></td>
          <td><code>{{ (d.token||'').slice(0, 12) }}...</code></td>
          <td>{{ d.os || '—' }}</td>
          <td>{{ fmt(d.last_seen) }}</td>
          <td><button class="btn-danger" @click="del(d)">删除</button></td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">暂无设备</div>

    <div v-if="showAdd" class="overlay" @click.self="showAdd = false">
      <div class="modal">
        <h3>添加设备</h3>
        <p>在目标设备上运行以下命令：</p>
        <div class="install-box">
          <pre @click="copy('linux')">curl -s https://ai.12fz.com/install-device.sh | bash -s -- --code={{ deviceKey }}</pre>
          <pre @click="copy('win')"># Windows PowerShell
irm https://ai.12fz.com/install-device.ps1 | iex -Code {{ deviceKey }}</pre>
        </div>
        <p class="tip">密钥 24 小时内有效，运行后设备自动出现在列表中</p>
        <button class="btn-outline" style="width:100%" @click="showAdd = false; loadData()">关闭</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const devices = ref<any[]>([])
const deviceKey = ref('')
const showAdd = ref(false)

async function request(url: string, opts: any = {}) {
  const token = localStorage.getItem('token') || ''
  return (await fetch('/api' + url, { headers: { 'Content-Type': 'application/json', Authorization: token }, ...opts })).json()
}

onMounted(loadData)

async function loadData() {
  const d = await request('/devices')
  devices.value = d.devices || []
  deviceKey.value = d.device_key || ''
}

async function startRename(d: any) {
  const name = prompt('新名称', d.name)
  if (name && name !== d.name) {
    d.name = name
    alert('已修改（仅本地显示，需后端支持 rename 接口）')
  }
}

async function del(d: any) {
  if (!confirm('删除 ' + d.name + '？')) return
  await request('/devices/' + encodeURIComponent(d.id), { method: 'DELETE' })
  await loadData()
}

function copyKey() {
  navigator.clipboard.writeText(deviceKey.value)
  alert('已复制')
}

function copy(os: string) {
  const cmd = os === 'linux'
    ? 'curl -s https://ai.12fz.com/install-device.sh | bash -s -- --code=' + deviceKey.value
    : 'irm https://ai.12fz.com/install-device.ps1 | iex -Code ' + deviceKey.value
  navigator.clipboard.writeText(cmd)
  alert('已复制，粘贴到目标设备终端执行')
}

function fmt(t: string) {
  if (!t || t.startsWith('0001')) return '—'
  return new Date(t).toLocaleString()
}
</script>

<style scoped>
.admin-devices { padding: 20px; }
.admin-header { margin-bottom: 16px; }
.admin-nav { display: flex; gap: 12px; margin-bottom: 12px; }
.admin-nav a { color: #1890ff; text-decoration: none; font-size: 14px; }
.header-actions { display: flex; gap: 8px; margin-bottom: 16px; }
.btn-primary { padding: 6px 16px; background: #1890ff; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 13px; }
.btn-outline { padding: 6px 16px; background: #fff; color: #1890ff; border: 1px solid #1890ff; border-radius: 4px; cursor: pointer; font-size: 13px; }
.btn-danger { padding: 4px 10px; background: #fff; color: #f5222d; border: 1px solid #f5222d; border-radius: 4px; cursor: pointer; font-size: 12px; }
.key-card { background: #e6f7ff; padding: 12px; border-radius: 6px; margin-bottom: 16px; display: flex; align-items: center; gap: 12px; }
.key-card code { background: #fff; padding: 4px 8px; border-radius: 3px; font-size: 13px; }
.table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; }
.table th,.table td { padding: 10px 14px; border-bottom: 1px solid #f0f0f0; font-size: 13px; text-align: left; }
.table th { background: #fafafa; font-weight: 500; }
.tag { padding: 2px 8px; border-radius: 10px; font-size: 12px; }
.tag.online { background: #f6ffed; color: #52c41a; }
.tag.offline,.tag.unknown { background: #fff2f0; color: #f5222d; }
.empty { text-align: center; color: #999; padding: 48px; }
.overlay { position: fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,.3); z-index:1000; display:flex; align-items:center; justify-content:center; }
.modal { background:#fff; border-radius:8px; padding:24px; width:550px; max-width:90vw; }
.modal h3 { margin:0 0 16px; }
.install-box pre { background:#1e1e1e; color:#d4d4d4; padding:12px; border-radius:4px; font-size:12px; overflow-x:auto; white-space:pre-wrap; word-break:break-all; margin:8px 0; }
.tip { color: #888; font-size: 12px; margin-top: 8px; }
</style>

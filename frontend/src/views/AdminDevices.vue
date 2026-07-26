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
        <button class="btn-primary btn-outline" @click="loadDevices">刷新</button>
      </div>
    </div>

    <table v-if="devices.length" class="device-table">
      <thead>
        <tr><th>设备名</th><th>系统</th><th>状态</th><th>Token</th><th>最后心跳</th><th>操作</th></tr>
      </thead>
      <tbody>
        <tr v-for="d in devices" :key="d.id">
          <td>
            <input v-if="editingId === d.id" v-model="editName" @blur="saveRename(d)" @keyup.enter="saveRename(d)" style="width:120px" autofocus />
            <span v-else @dblclick="startRename(d)" style="cursor:pointer" title="双击改名">{{ d.name }}</span>
          </td>
          <td>{{ d.os || '—' }}</td>
          <td><span class="status-tag" :class="d.status">{{ d.status }}</span></td>
          <td><code class="token-text">{{ (d.token || '').slice(0, 12) }}...</code></td>
          <td>{{ formatTime(d.last_seen) }}</td>
          <td><button class="btn-danger" @click="deleteDevice(d)">删除</button></td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">暂无设备，点击「添加设备」生成安装命令</div>

    <div v-if="showAdd" class="modal-overlay" @click.self="showAdd = false">
      <div class="modal">
        <h3>添加设备</h3>
        <div v-if="installCode">
          <p class="code-info">注册码：<code>{{ installCode }}</code>（24小时内有效）</p>
          <div class="install-box">
            <h4>Linux / macOS</h4>
            <pre @click="copyCmd('linux')" style="cursor:pointer">{{ installLinux }}</pre>
            <h4>Windows</h4>
            <pre @click="copyCmd('win')" style="cursor:pointer">{{ installWindows }}</pre>
          </div>
          <button class="btn-primary btn-outline" style="width:100%" @click="showAdd = false; loadDevices()">关闭</button>
        </div>
        <div v-else>
          <div class="form-group">
            <label>设备名称</label>
            <input v-model="newName" placeholder="如：办公电脑" @keyup.enter="generateCode" />
          </div>
          <button class="btn-primary" style="width:100%;margin-top:12px" @click="generateCode">生成安装命令</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../api'

const devices = ref<any[]>([])
const showAdd = ref(false)
const newName = ref('')
const installCode = ref('')
const installLinux = ref('')
const installWindows = ref('')
const editingId = ref('')
const editName = ref('')

onMounted(loadDevices)

async function loadDevices() {
  try { const { data } = await api.get('/devices'); devices.value = data.devices || [] } catch(e: any) { alert('加载失败: ' + (e.response?.data?.error || e.message)) }
}

async function generateCode() {
  try {
    const { data } = await api.post('/devices/generate-code', {}, { params: { name: newName.value || '新设备' } })
    installCode.value = data.code
    installLinux.value = data.install_linux
    installWindows.value = data.install_windows
  } catch(e: any) { alert('生成失败: ' + (e.response?.data?.error || e.message)) }
}

function copyCmd(os: string) {
  const cmd = os === 'linux' ? installLinux.value : installWindows.value
  navigator.clipboard.writeText(cmd)
  alert('已复制！在设备终端粘贴执行')
}

function startRename(d: any) { editingId.value = d.id; editName.value = d.name }
async function saveRename(d: any) {
  if (editName.value && editName.value !== d.name) {
    try { await api.put(`/devices/${encodeURIComponent(d.id)}/rename`, { name: editName.value }); d.name = editName.value } catch(e: any) { alert('改名失败: ' + (e.response?.data?.error || e.message)) }
  }
  editingId.value = ''
}

async function deleteDevice(d: any) {
  if (confirm(`删除 "${d.name}"？`)) {
    try { await api.delete(`/devices/${encodeURIComponent(d.id)}`); await loadDevices() } catch(e: any) { alert('删除失败: ' + (e.response?.data?.error || e.message)) }
  }
}

function formatTime(t: string) {
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
.btn-primary.btn-outline { background: #fff; color: #1890ff; border: 1px solid #1890ff; }
.btn-danger { padding: 4px 10px; background: #fff; color: #f5222d; border: 1px solid #f5222d; border-radius: 4px; cursor: pointer; font-size: 12px; }
.device-table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; }
.device-table th, .device-table td { padding: 10px 14px; border-bottom: 1px solid #f0f0f0; text-align: left; font-size: 13px; }
.device-table th { background: #fafafa; font-weight: 500; }
.status-tag { padding: 2px 8px; border-radius: 10px; font-size: 12px; }
.status-tag.online { background: #f6ffed; color: #52c41a; }
.status-tag.offline { background: #fff2f0; color: #f5222d; }
.token-text { font-size: 12px; background: #f5f5f5; padding: 2px 6px; border-radius: 3px; }
.empty { text-align: center; color: #999; padding: 48px; }
.modal-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,.3); z-index: 1000; display: flex; align-items: center; justify-content: center; }
.modal { background: #fff; border-radius: 8px; padding: 24px; width: 500px; max-width: 90vw; max-height: 80vh; overflow-y: auto; }
.modal h3 { margin: 0 0 16px; }
.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 13px; color: #666; margin-bottom: 4px; }
.form-group input { width: 100%; padding: 8px; border: 1px solid #d9d9d9; border-radius: 4px; font-size: 13px; box-sizing: border-box; }
.code-info { color: #52c41a; margin: 8px 0; }
.install-box pre { background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; font-size: 12px; overflow-x: auto; white-space: pre-wrap; word-break: break-all; }
.install-box h4 { font-size: 14px; margin: 12px 0 4px; }
</style>

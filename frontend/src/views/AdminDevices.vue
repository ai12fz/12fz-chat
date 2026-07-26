<template>
  <div class="admin-devices">
    <div class="admin-header">
      <h2>设备管理</h2>
      <button class="btn-primary" @click="refresh">刷新</button>
    </div>

    <div class="key-box" v-if="deviceKey">
      <span class="key-label">商户设备密钥：</span>
      <code>{{ deviceKey }}</code>
      <button class="btn-copy" @click="copyKey">复制</button>
      <span class="copy-hint" v-if="copied">✓ 已复制</span>
    </div>

    <table class="device-table" v-if="devices.length">
      <thead>
        <tr>
          <th>设备名</th>
          <th>系统</th>
          <th>状态</th>
          <th>Token</th>
          <th>最后心跳</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="d in devices" :key="d.id">
          <td>{{ d.name }}</td>
          <td>{{ d.os || '—' }}</td>
          <td><span :class="['status-tag', d.status]">{{ d.status }}</span></td>
          <td><code class="token-text">{{ d.token?.slice(0,12) }}...</code></td>
          <td>{{ formatTime(d.last_seen) }}</td>
          <td>
            <button class="btn-danger" @click="delDevice(d)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">暂无设备。在客户端运行注册脚本以添加设备。</div>

    <div class="install-box">
      <h3>设备安装命令</h3>
      <pre>Linux:   curl -s https://dl.12fz.com/install-device.sh | bash -s -- --key={{ deviceKey }}
Windows: powershell -c "iwr https://dl.12fz.com/install-device.ps1 | iex" -args "-key {{ deviceKey }}"</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../api'

interface Device {
  id: string
  name: string
  os: string
  status: string
  token: string
  last_seen: string
}

const devices = ref<Device[]>([])
const deviceKey = ref('')
const copied = ref(false)

async function refresh() {
  try {
    const { data } = await api.get('/devices')
    devices.value = data.devices || []
    deviceKey.value = data.device_key || ''
  } catch {}
}

function copyKey() {
  navigator.clipboard.writeText(deviceKey.value)
  copied.value = true
  setTimeout(() => copied.value = false, 2000)
}

function formatTime(t: string) {
  if (!t || t.startsWith('0001')) return '—'
  return new Date(t).toLocaleString()
}

async function delDevice(d: Device) {
  if (!confirm(`删除设备 "${d.name}"？`)) return
  try {
    await api.delete(`/devices/${encodeURIComponent(d.id)}`)
    await refresh()
  } catch {}
}

onMounted(refresh)
</script>

<style scoped>
.admin-devices { padding: 24px; max-width: 900px; margin: 0 auto; }
.admin-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.admin-header h2 { margin: 0; font-size: 20px; }

.key-box { background: #f6ffed; border: 1px solid #b7eb8f; border-radius: 6px; padding: 10px 16px; margin-bottom: 20px; display: flex; align-items: center; gap: 8px; }
.key-label { font-size: 13px; color: #333; }
.key-box code { font-size: 14px; background: #fff; padding: 2px 8px; border-radius: 3px; }
.btn-copy { padding: 4px 12px; border: 1px solid #52c41a; border-radius: 4px; background: #fff; color: #52c41a; cursor: pointer; font-size: 12px; }
.btn-copy:hover { background: #52c41a; color: #fff; }
.copy-hint { font-size: 12px; color: #52c41a; }

.device-table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,.1); }
.device-table th, .device-table td { padding: 10px 14px; text-align: left; border-bottom: 1px solid #f0f0f0; font-size: 14px; }
.device-table th { background: #fafafa; font-weight: 600; }
.token-text { font-size: 12px; color: #666; }
.btn-primary { padding: 8px 16px; border: none; border-radius: 6px; background: #1890ff; color: #fff; cursor: pointer; font-size: 14px; }
.btn-primary:hover { background: #40a9ff; }
.btn-danger { padding: 4px 10px; border: 1px solid #ff4d4f; border-radius: 4px; background: #fff; color: #ff4d4f; cursor: pointer; font-size: 12px; }
.btn-danger:hover { background: #fff1f0; }

.status-tag { padding: 2px 8px; border-radius: 4px; font-size: 12px; }
.status-tag.online { background: #f6ffed; color: #52c41a; border: 1px solid #b7eb8f; }
.status-tag.offline { background: #fff2f0; color: #ff4d4f; border: 1px solid #ffccc7; }

.empty { padding: 60px; text-align: center; color: #999; font-size: 16px; }

.install-box { margin-top: 30px; background: #fafafa; border-radius: 6px; padding: 16px; }
.install-box h3 { margin: 0 0 8px; font-size: 14px; }
.install-box pre { background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 4px; font-size: 12px; overflow-x: auto; white-space: pre-wrap; }
</style>

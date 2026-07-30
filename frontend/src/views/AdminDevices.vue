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
        <button class="btn-primary" @click="genCode">+ 生成注册码</button>
        <button class="btn-outline" @click="loadData">刷新</button>
      </div>
    </div>

    <div class="key-card">
      <h3>注册码（每设备独立，一次一码）</h3>
      <table v-if="regCodes.length" class="table">
        <thead><tr><th>注册码</th><th>状态</th><th>设备</th><th>创建</th><th>技能安装</th><th>软件安装</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="rc in regCodes" :key="rc.code">
            <td><code>{{ rc.code }}</code></td>
            <td>{{ rc.status === 'active' ? '有效' : rc.status === 'used' ? '已用' : '已撤销' }}</td>
            <td>{{ rc.device_id || '—' }}</td>
            <td>{{ fmt(rc.created_at) }}</td>
            <td>
              <button class="btn-sm" @click="copyText('curl -s https://ai.12fz.com/install-device.sh | bash -s -- --code=' + rc.code)">复制安装命令</button>
              <button v-if="rc.status==='active'" class="btn-danger btn-sm" @click="revokeCode(rc.code)">撤销</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty">暂无注册码</div>
    </div>

    <table v-if="devices.length" class="table">
      <thead><tr><th>设备名</th><th>状态</th><th>模型</th><th>Token</th><th>系统</th><th>最后上线</th><th>技能安装</th><th>软件安装</th><th>操作</th></tr></thead>
      <tbody>
        <tr v-for="d in devices" :key="d.id">
          <td><span @dblclick="startRename(d)" title="双击改名">{{ d.name }}</span></td>
          <td><span class="status-dot" :class="d.status" :title="d.status"></span> {{ d.status === 'online' ? '在线' : d.status === 'offline' ? '离线' : (d.status || '未知') }}</td>
          <td>
            <select :value="d.model_name || 'deepseek-v4-flash'" @change="changeModel(d, ($event.target as HTMLSelectElement).value)" class="model-select">
              <option v-for="m in models" :key="m.name" :value="m.name">{{ m.display_name || m.name }}</option>
            </select>
          </td>
          <td><code>{{ (d.token||'').slice(0, 12) }}...</code></td>
          <td>{{ d.os || '—' }}</td>
          <td>{{ fmt(d.last_seen) }}</td>
          <td><label class="switch"><input type="checkbox" :checked="d.allow_install_skills" @change="toggleSkills(d)"><span class="slider"></span></label></td>
          <td><label class="switch"><input type="checkbox" :checked="d.allow_install_software" @change="toggleSoftware(d)"><span class="slider"></span></label></td>
          <td><button class="btn-danger" @click="del(d)">删除</button></td>
        </tr>
      </tbody>
    </table>
    <div v-if="!devices.length" class="empty">暂无设备</div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const devices = ref<any[]>([])
const regCodes = ref<any[]>([])
const models = ref<any[]>([])

async function request(url: string, opts: any = {}) {
  const token = localStorage.getItem('token') || ''
  const h: any = { Authorization: 'Bearer ' + token }
  if (opts.body) h['Content-Type'] = 'application/json'
  const resp = await fetch(url, { ...opts, headers: { ...h, ...opts.headers } })
  return resp.json()
}

onMounted(loadData)
onMounted(async () => {
  try {
    const r = await request('/admin/proxy/models')
    if (Array.isArray(r)) models.value = r
    else if (r.data && Array.isArray(r.data)) models.value = r.data
  } catch(_) {}
})

async function loadData() {
  try {
    const d = await request('/api/devices')
    if (d && d.devices) devices.value = d.devices
  } catch(_) {}
  try {
    const rc = await request('/api/device-reg-codes')
    if (Array.isArray(rc)) regCodes.value = rc
  } catch(_) {}
}

async function genCode() {
  try {
    const r = await request('/api/device-reg-codes', { method: 'POST' })
    if (r.code) alert('注册码: ' + r.code)
    else if (r.error) alert('错误: ' + r.error)
    else alert('未知响应: ' + JSON.stringify(r))
  } catch(e: any) { alert('异常: ' + (e.message||String(e)) + ' | token=' + (localStorage.getItem('token')||'none')) }
  loadData()
}

async function revokeCode(code: string) {
  await request('/api/device-reg-codes/' + encodeURIComponent(code), { method: 'DELETE' })
  loadData()
}

function copyText(t: string) {
  var ta = document.createElement('textarea')
  ta.value = t
  ta.style.position = 'fixed'
  ta.style.left = '-9999px'
  document.body.appendChild(ta)
  ta.select()
  try { document.execCommand('copy'); alert('已复制') } catch(e) { prompt('请手动复制：', t) }
  document.body.removeChild(ta)
}

async function startRename(d: any) {
  const name = prompt('新名称', d.name)
  if (name && name !== d.name) {
    await request('/api/devices/' + encodeURIComponent(d.id), { method: 'PATCH', body: JSON.stringify({ name }) })
    loadData()
  }
}

async function del(d: any) {
  await request('/api/devices/' + encodeURIComponent(d.id), { method: 'DELETE' })
  loadData()
}

async function changeModel(d: any, modelName: string) {
  await request('/api/devices/' + encodeURIComponent(d.id) + '/model', {
    method: 'PUT',
    body: JSON.stringify({ model_name: modelName, model_provider: '' })
  })
  d.model_name = modelName
}

function fmt(t: string) {
  return t ? new Date(t).toLocaleString('zh-CN') : '—'
}
</script>

<style scoped>
.status-dot {
  display: inline-block; width: 10px; height: 10px; border-radius: 50%%; margin-right: 6px; vertical-align: middle;
}
.status-dot.online { background: #22c55e; box-shadow: 0 0 6px #22c55e; }
.status-dot.offline { background: #ef4444; }
.status-dot:not(.online):not(.offline) { background: #9ca3af; }
.admin-devices { padding: 20px; }
.admin-header { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 12px; }
.admin-nav { display: flex; gap: 12px; }
.admin-nav a { color: #666; text-decoration: none; }
.admin-nav a.router-link-active { color: #6366f1; font-weight: bold; }
.header-actions { display: flex; gap: 8px; }
.btn-primary { background: #6366f1; color: #fff; border: none; padding: 8px 16px; border-radius: 6px; cursor: pointer; }
.btn-outline { background: #fff; border: 1px solid #ddd; padding: 8px 16px; border-radius: 6px; cursor: pointer; }
.btn-danger { background: #ef4444; color: #fff; border: none; padding: 4px 10px; border-radius: 4px; cursor: pointer; }
.btn-sm { background: #f0f0f0; border: 1px solid #ddd; padding: 3px 8px; border-radius: 4px; cursor: pointer; font-size: 12px; }
.key-card { background: #f0f7ff; border: 1px solid #bfdbfe; border-radius: 8px; padding: 16px; margin-bottom: 20px; }
.key-card h3 { margin-top: 0; }
.table { width: 100%; border-collapse: collapse; }
.table th, .table td { padding: 8px 12px; border-bottom: 1px solid #eee; text-align: left; }
.table th { background: #f9fafb; font-size: 13px; color: #666; }
.tag.online { background: #dcfce7; color: #16a34a; padding: 2px 8px; border-radius: 10px; font-size: 12px; }
.tag.offline { background: #f3f4f6; color: #9ca3af; padding: 2px 8px; border-radius: 10px; font-size: 12px; }
.empty { color: #999; padding: 20px; }
.model-select { padding: 2px 4px; border: 1px solid #d9d9d9; border-radius: 4px; font-size: 12px; max-width: 130px; }
.switch { position: relative; display: inline-block; width: 40px; height: 22px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background: #ccc; border-radius: 22px; transition: .3s; }
.slider:before { content: ""; position: absolute; height: 16px; width: 16px; left: 3px; bottom: 3px; background: white; border-radius: 50%; transition: .3s; }
input:checked + .slider { background: #22c55e; }
input:checked + .slider:before { transform: translateX(18px); }
</style>

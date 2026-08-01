<template>
  <div class="admin-devices">
    <div class="admin-header">
      <h2>设备管理</h2>
      <div class="admin-nav">
        <router-link to="/admin/devices">设备管理</router-link>
        <router-link to="/admin/proxy">中转站</router-link>
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
          <tr v-for="rc in visibleCodes" :key="rc.code">
            <td><code>{{ rc.code }}</code></td>
            <td>{{ rc.status === 'active' ? '有效' : rc.status === 'used' ? '已用' : '已撤销' }}</td>
            <td>{{ rc.device_id || '—' }}</td>
            <td>{{ fmt(rc.created_at) }}</td>
            <td>
              <button class="btn-sm" :disabled="rc.status!=='active'" @click="copyText(linuxCmd(rc.code))">🐧 Linux 命令</button>
              <button class="btn-sm" :disabled="rc.status!=='active'" @click="copyText(winCmd(rc.code))">🪟 Windows 命令</button>
              <button v-if="rc.status==='active'" class="btn-danger btn-sm" @click="revokeCode(rc.code)">撤销</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="empty">暂无注册码</div>
      <div v-if="regCodes.length > defaultShowCount" class="expand-bar">
        <button class="btn-link" @click="showAllCodes = !showAllCodes">
          {{ showAllCodes ? '收起' : `展开全部（共 ${regCodes.length} 条）` }}
        </button>
      </div>
    </div>

    <div class="key-card">
      <h3>📄 文档配额（商户保留文档数，默认 20，阶梯收费用）</h3>
      <div class="quota-row">
        <label>商户ID(org_id)</label>
        <input v-model="quotaMerchantId" placeholder="00000000-0000-0000-0000-000000000000" class="quota-input" />
        <label>保留份数</label>
        <input v-model.number="quotaLimit" type="number" min="1" max="1000" class="quota-input" style="width:80px" />
        <button class="btn-primary btn-sm" @click="saveQuota">保存</button>
        <button class="btn-outline btn-sm" @click="loadQuota">查询</button>
        <span v-if="quotaMsg" class="quota-msg">{{ quotaMsg }}</span>
      </div>
    </div>

    <table v-if="devices.length" class="table">
      <thead><tr><th style="width:30px"></th><th>设备名</th><th>状态</th><th>模型</th><th>Token</th><th>系统</th><th>安装</th><th>本地IP</th><th>最后上线</th><th>技能安装</th><th>软件安装</th><th>操作</th></tr></thead>
      <tbody>
        <template v-for="d in devices" :key="d.id">
          <tr :class="{ 'device-row': true, expanded: expandedDevices[d.id] }">
            <td>
              <button class="btn-link expand-btn" @click="toggleExpand(d)">{{ expandedDevices[d.id] ? '▾' : '▸' }}</button>
            </td>
            <td><span @dblclick="startRename(d)" title="双击改名">{{ d.name }}</span></td>
            <td><span class="status-dot" :class="d.status" :title="d.status"></span> {{ d.status === 'online' ? '在线' : d.status === 'offline' ? '离线' : (d.status || '未知') }}</td>
            <td>
              <select :value="d.model_name || 'deepseek-v4-flash'" @change="changeModel(d, ($event.target as HTMLSelectElement).value)" class="model-select">
                <option v-for="m in models" :key="m.name" :value="m.name">{{ m.display_name || m.name }}</option>
              </select>
            </td>
            <td>
              <code>{{ (d.token||'').slice(0, 12) }}...</code>
              <button class="btn-sm copy-btn" title="复制完整 token" @click="copyText(d.token||'')">📋 复制</button>
            </td>
            <td>{{ d.os || '—' }}</td>
            <td><span class="agent-badge" :class="'agent-' + (d.agent_type || 'hermes')">{{ agentLabel(d.agent_type) }}</span></td>
            <td>{{ d.local_ip || '—' }}</td>
            <td>{{ fmt(d.last_seen) }}</td>
            <td><label class="switch"><input type="checkbox" :checked="d.allow_install_skills" @change="toggleSkills(d)"><span class="slider"></span></label></td>
            <td><label class="switch"><input type="checkbox" :checked="d.allow_install_software" @change="toggleSoftware(d)"><span class="slider"></span></label></td>
            <td>
              <button class="btn-primary btn-sm" @click="openAddAgent(d)">+ Agent</button>
              <button class="btn-danger" @click="del(d)">删除</button>
            </td>
          </tr>
          <tr v-if="expandedDevices[d.id]" class="agent-subrow">
            <td colspan="12" class="agent-subrow-td">
              <div class="agent-subtable-wrap">
                <div class="agent-subtable-header">
                  <span>Agent 列表（{{ (d._agents || []).length }}）</span>
                  <button class="btn-primary btn-sm" @click="openAddAgent(d)">+ 添加 Agent</button>
                </div>
                <table v-if="d._agents && d._agents.length" class="table agent-subtable">
                  <thead><tr><th>Bot ID</th><th>名称</th><th>类型</th><th>模型</th><th>状态</th><th>技能安装</th><th>软件安装</th><th>心跳</th><th>操作</th></tr></thead>
                  <tbody>
                    <tr v-for="a in d._agents" :key="a.bot_id">
                      <td><code>{{ a.bot_id }}</code></td>
                      <td><span class="agent-name" @dblclick="startRenameAgent(a, d)" title="双击改名">{{ a.display_name }}</span></td>
                      <td><span :class="'agent-badge ' + (a.agent_type === 'hermes' ? 'agent-hermes' : 'agent-api')">{{ a.agent_type === 'hermes' ? 'Hermes' : 'API' }}</span></td>
                      <td>
                        <select :value="a.model || 'deepseek-v4-flash'" @change="changeAgentModel(a, d, ($event.target as HTMLSelectElement).value)" class="model-select">
                          <option v-for="m in models" :key="m.name" :value="m.name">{{ m.display_name || m.name }}</option>
                        </select>
                      </td>
                      <td>{{ a.status === 'active' ? '启用' : '停用' }}</td>
                      <td><label class="switch"><input type="checkbox" :checked="!!a.allow_install_skills" @change="toggleAgentSkill(a, d)"><span class="slider"></span></label></td>
                      <td><label class="switch"><input type="checkbox" :checked="!!a.allow_install_software" @change="toggleAgentSoftware(a, d)"><span class="slider"></span></label></td>
                      <td>
                        <span v-if="isAgentOnline(a)" class="status-dot online" style="background:#22c55e;box-shadow:0 0 6px #22c55e"></span>
                        <span v-else class="status-dot offline" style="background:#ef4444"></span>
                        {{ isAgentOnline(a) ? '在线' : '离线' }}
                      </td>
                      <td>
                        <button class="btn-danger btn-sm" @click="delAgent(a, d)">删除</button>
                      </td>
                    </tr>
                  </tbody>
                </table>
                <div v-else class="empty" style="padding:8px">该设备下暂无 Agent，点击「+ 添加 Agent」创建</div>
              </div>
            </td>
          </tr>
        </template>
      </tbody>
    </table>
    <div v-if="!devices.length" class="empty">暂无设备</div>

    <!-- 添加/编辑 Agent 弹窗 -->
    <div v-if="showAgentModal" class="modal-mask" @click.self="showAgentModal = false">
      <div class="modal">
        <h3>{{ editAgent.bot_id ? '编辑 Agent' : '添加 Agent — ' + (currentDevice?.name || '') }}</h3>
        <div class="form-group"><label>显示名称 *</label><input v-model="editAgent.display_name" placeholder="如：客服小助手" /></div>
        <div class="form-group">
          <label>类型</label>
          <select v-model="editAgent.agent_type">
            <option value="hermes">Hermes（本机运行）</option>
            <option value="api">API（走中转站）</option>
          </select>
        </div>
        <div class="form-group">
          <label>模型</label>
          <select v-model="editAgent.model">
            <option v-for="m in models" :key="m.name" :value="m.name">{{ m.display_name || m.name }}</option>
          </select>
        </div>
        <div class="form-group">
          <label>权限</label>
          <div class="perm-row">
            <label class="perm-check"><input type="checkbox" v-model="editAgent.allow_install_skills" /> 允许安装技能</label>
            <label class="perm-check"><input type="checkbox" v-model="editAgent.allow_install_software" /> 允许安装软件</label>
          </div>
        </div>
        <div v-if="editAgent.bot_id" class="form-group"><label>Bot ID</label><input v-model="editAgent.bot_id" disabled /></div>
        <div class="modal-actions">
          <button class="btn-outline" @click="showAgentModal = false">取消</button>
          <button class="btn-primary" @click="saveAgent" :disabled="saving">{{ saving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'

const devices = ref<any[]>([])
const regCodes = ref<any[]>([])
const models = ref<any[]>([])
const showAllCodes = ref(false)
const quotaMerchantId = ref('00000000-0000-0000-0000-000000000000')
const quotaLimit = ref(20)
const quotaMsg = ref('')
const defaultShowCount = 2
const expandedDevices = ref<Record<string, boolean>>({})
const showAgentModal = ref(false)
const currentDevice = ref<any>(null)
const saving = ref(false)
const editAgent = ref<any>({ bot_id: '', display_name: '', model: 'deepseek-v4-flash', agent_type: 'hermes', allow_install_skills: false, allow_install_software: false })

const visibleCodes = computed(() => {
  if (showAllCodes.value) return regCodes.value
  return regCodes.value.slice(0, defaultShowCount)
})

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
    const r = await request('/api/admin/proxy/models')
    if (Array.isArray(r)) models.value = r
    else if (r.data && Array.isArray(r.data)) models.value = r.data
  } catch(_) {}
})

async function loadData() {
  try {
    const d = await request('/api/devices')
    if (d && d.devices) {
      devices.value = d.devices
      // 保留已展开设备的 agents
      for (const dev of devices.value) {
        if (expandedDevices.value[dev.id] && dev._agents) {
          // refresh agents silently
          loadAgents(dev, false)
        }
      }
    }
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
  if (navigator.clipboard && navigator.clipboard.writeText) {
    navigator.clipboard.writeText(t).then(() => alert('已复制')).catch(() => fallbackCopy(t))
  } else {
    fallbackCopy(t)
  }
}
function fallbackCopy(t: string) {
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
  if (!confirm(`确认删除设备 "${d.name}"？`)) return
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

async function toggleSkills(d: any) {
  const r = await request('/api/devices/' + encodeURIComponent(d.id) + '/toggle-skills', { method: 'POST' })
  if (r && r.allow_install_skills !== undefined) d.allow_install_skills = r.allow_install_skills
  else d.allow_install_skills = !d.allow_install_skills
}

async function toggleSoftware(d: any) {
  const r = await request('/api/devices/' + encodeURIComponent(d.id) + '/toggle-software', { method: 'POST' })
  if (r && r.allow_install_software !== undefined) d.allow_install_software = r.allow_install_software
  else d.allow_install_software = !d.allow_install_software
}

// ── Agent 子表 ──

function isAgentOnline(a: any): boolean {
  if (!a.heartbeat_at) return false
  const hb = new Date(a.heartbeat_at).getTime()
  return Date.now() - hb < 90 * 1000
}

async function toggleExpand(d: any) {
  const willExpand = !expandedDevices.value[d.id]
  expandedDevices.value[d.id] = willExpand
  if (willExpand && !d._agents) {
    await loadAgents(d, true)
  }
}

async function loadAgents(d: any, spinner: boolean) {
  try {
    const r = await request('/api/agents?device_id=' + encodeURIComponent(d.id))
    if (Array.isArray(r)) d._agents = r
  } catch(_) {
    if (spinner) alert('加载 Agent 列表失败')
  }
}

function openAddAgent(d: any) {
  currentDevice.value = d
  editAgent.value = { bot_id: '', display_name: '', model: (d.model_name || 'deepseek-v4-flash'), agent_type: d.agent_type === 'api' ? 'api' : 'hermes', allow_install_skills: !!d.allow_install_skills, allow_install_software: !!d.allow_install_software }
  showAgentModal.value = true
}

async function saveAgent() {
  if (!editAgent.value.display_name) { alert('请输入显示名称'); return }
  saving.value = true
  try {
    if (editAgent.value.bot_id) {
      // 编辑
      await request('/api/agents/' + encodeURIComponent(editAgent.value.bot_id), { method: 'PUT', body: JSON.stringify(editAgent.value) })
    } else {
      // 新建:bot_id = device-{ts},自动挂设备
      const a = {
        bot_id: 'agent-' + Date.now(),
        display_name: editAgent.value.display_name,
        device_id: currentDevice.value.id,
        model: editAgent.value.model,
        agent_type: editAgent.value.agent_type,
        allow_install_skills: editAgent.value.allow_install_skills,
        allow_install_software: editAgent.value.allow_install_software,
        status: 'active'
      }
      const r = await request('/api/agents', { method: 'POST', body: JSON.stringify(a) })
      if (r && r.bot_id) {
        alert('已创建 Agent: ' + r.bot_id + '\n' + (r.token ? 'Token: ' + r.token : ''))
      }
    }
    showAgentModal.value = false
    expandedDevices.value[currentDevice.value.id] = true
    await loadAgents(currentDevice.value, false)
  } catch(e: any) {
    alert('保存失败: ' + (e.message || e))
  } finally {
    saving.value = false
  }
}

async function startRenameAgent(a: any, d: any) {
  const name = prompt('新名称', a.display_name)
  if (name && name !== a.display_name) {
    await request('/api/agents/' + encodeURIComponent(a.bot_id), { method: 'PUT', body: JSON.stringify({ display_name: name }) })
    await loadAgents(d, false)
  }
}

async function changeAgentModel(a: any, d: any, modelName: string) {
  await request('/api/agents/' + encodeURIComponent(a.bot_id), { method: 'PUT', body: JSON.stringify({ model: modelName }) })
  a.model = modelName
}

async function toggleAgentSkill(a: any, d: any) {
  await request('/api/agents/' + encodeURIComponent(a.bot_id), { method: 'PUT', body: JSON.stringify({ allow_install_skills: !a.allow_install_skills }) })
  a.allow_install_skills = !a.allow_install_skills
}

async function toggleAgentSoftware(a: any, d: any) {
  await request('/api/agents/' + encodeURIComponent(a.bot_id), { method: 'PUT', body: JSON.stringify({ allow_install_software: !a.allow_install_software }) })
  a.allow_install_software = !a.allow_install_software
}

async function delAgent(a: any, d: any) {
  if (!confirm(`确认删除 Agent "${a.display_name}"？`)) return
  try {
    await request('/api/agents/' + encodeURIComponent(a.bot_id), { method: 'DELETE' })
    await loadAgents(d, false)
  } catch(e: any) { alert(e.message || '删除失败') }
}

function fmt(t: string) {
  return t ? new Date(t).toLocaleString('zh-CN') : '—'
}

function agentLabel(t?: string) {
  const m: Record<string, string> = { '12fzclaw': '12fzclaw', openclaw: 'openclaw', hermes: 'Hermes' }
  return m[t || ''] || (t || '—')
}

function linuxCmd(code: string) {
  return 'curl -s https://ai.12fz.com/static/install-device.sh | bash -s -- --code=' + code
}

function winCmd(code: string) {
  // PowerShell 一键安装: 下载 ps1 → 执行 -Code 注册
  const dl = 'iwr https://ai.12fz.com/static/install-device.ps1 -OutFile $env:TEMP\\install-device.ps1'
  const ex = '& $env:TEMP\\install-device.ps1 -Code ' + code
  return 'powershell -ExecutionPolicy Bypass -Command "' + dl + '; ' + ex + '"'
}
</script>

<style scoped>
.status-dot {
  display: inline-block; width: 10px; height: 10px; border-radius: 50%; margin-right: 6px; vertical-align: middle;
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
.agent-badge { display: inline-block; padding: 2px 10px; border-radius: 10px; font-size: 12px; font-weight: 500; white-space: nowrap; }
.agent-hermes { background: #e0e7ff; color: #4338ca; }
.agent-12fzclaw { background: #d1fae5; color: #047857; }
.agent-openclaw { background: #ffedd5; color: #c2410c; }
.agent-api { background: #fef3c7; color: #92400e; }
.agent- { background: #f3f4f6; color: #6b7280; }
.empty { color: #999; padding: 20px; }
.expand-bar { text-align: center; padding: 8px 0; }
.btn-link { background: none; border: none; color: #6366f1; cursor: pointer; font-size: 13px; padding: 4px 8px; }
.btn-link:hover { text-decoration: underline; }
.model-select { padding: 2px 4px; border: 1px solid #d9d9d9; border-radius: 4px; font-size: 12px; max-width: 130px; }
.copy-btn { margin-left: 6px; white-space: nowrap; }
.switch { position: relative; display: inline-block; width: 40px; height: 22px; }
.switch input { opacity: 0; width: 0; height: 0; }
.slider { position: absolute; cursor: pointer; top: 0; left: 0; right: 0; bottom: 0; background: #ccc; border-radius: 22px; transition: .3s; }
.slider:before { content: ""; position: absolute; height: 16px; width: 16px; left: 3px; bottom: 3px; background: white; border-radius: 50%; transition: .3s; }
input:checked + .slider { background: #22c55e; }
input:checked + .slider:before { transform: translateX(18px); }

.quota-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-top: 8px;
}
.quota-row label { font-size: 13px; color: #666; }
.quota-input {
  padding: 5px 8px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
  width: 280px;
}
.quota-msg { font-size: 12px; color: #52c41a; margin-left: 6px; }

.device-row.expanded { background: #f5f7ff; }
.expand-btn { font-size: 12px; padding: 2px 6px; }
.agent-subrow-td { padding: 0 !important; background: #fafbff; }
.agent-subtable-wrap { padding: 12px 16px 12px 40px; }
.agent-subtable-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px; font-weight: 600; color: #444; }
.agent-subtable { border: 1px solid #e5e7eb; border-radius: 6px; overflow: hidden; }
.agent-subtable th { background: #eef2ff; font-size: 12px; }
.agent-subtable td { font-size: 13px; }
.agent-name { cursor: pointer; }
.agent-name:hover { color: #6366f1; text-decoration: underline; }

.modal-mask { position: fixed; inset: 0; background: rgba(0,0,0,.45); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: #fff; border-radius: 10px; padding: 24px; width: 420px; max-width: 92vw; box-shadow: 0 10px 40px rgba(0,0,0,.2); }
.modal h3 { margin-top: 0; }
.form-group { margin-bottom: 14px; }
.form-group label { display: block; font-size: 13px; color: #555; margin-bottom: 4px; }
.form-group input, .form-group select { width: 100%; padding: 7px 10px; border: 1px solid #d9d9d9; border-radius: 6px; font-size: 14px; box-sizing: border-box; }
.perm-row { display: flex; gap: 20px; }
.perm-check { display: flex !important; align-items: center; gap: 6px; font-size: 13px !important; color: #333 !important; margin-bottom: 0 !important; }
.perm-check input { width: auto !important; }
.modal-actions { display: flex; justify-content: flex-end; gap: 10px; margin-top: 18px; }
</style>

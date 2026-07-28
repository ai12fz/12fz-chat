<template>
  <div class="admin-agents">
    <div class="admin-header">
      <h2>Agent 管理</h2>
      <div class="admin-nav">
        <router-link to="/admin/devices">设备管理</router-link>
        <router-link to="/admin/proxy">中转站</router-link>
        <router-link to="/admin/agents">Agent管理</router-link>
      </div>
      <button class="btn-primary" @click="openAgent()">+ 新建 Agent</button>
    </div>

    <table v-if="agents.length">
      <thead><tr><th>Bot ID</th><th>显示名</th><th>类型</th><th>模型</th><th>状态</th><th>群聊</th><th>操作</th></tr></thead>
      <tbody>
        <tr v-for="a in agents" :key="a.bot_id">
          <td>{{ a.bot_id }}</td><td>{{ a.display_name }}</td>
          <td><span :class="'type-tag type-' + (a.agent_type || 'api')">{{ a.agent_type === 'hermes' ? 'Hermes' : 'API' }}</span></td>
          <td>
            <select :value="a.model" @change="changeModel(a, $event.target.value)" style="font-size:12px;padding:2px 4px;border:1px solid #d9d9d9;border-radius:4px">
              <option v-for="m in proxyModels" :key="m.id" :value="m.name">{{ m.display_name || m.name }}</option>
            </select>
          </td>
          <td><span class="status-tag" :class="a.status">{{ a.status }}</span></td>
          <td>{{ a.group_count || 0 }} 个群</td>
          <td>
            <button @click="openAgent(a)">编辑</button>
            <button @click="openGroups(a)">群聊</button>
            <button class="btn-danger" @click="deleteAgent(a)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">暂无 Agent，点击上方按钮创建</div>

    <!-- Agent Modal -->
    <div v-if="showAgent" class="modal-overlay" @click.self="showAgent = false">
      <div class="modal" style="max-width:560px">
        <h3>{{ editAgent.bot_id ? '编辑' : '新建' }} Agent</h3>
        
        <!-- Tab bar -->
        <div style="display:flex;gap:0;margin-bottom:12px;border-bottom:2px solid #eee">
          <div :class="'tab-btn' + (agentTab==='quick'?' active':'')" @click="agentTab='quick'" style="padding:6px 16px;cursor:pointer;border-bottom:2px solid transparent;margin-bottom:-2px;font-size:14px">⚡ 快速配置</div>
          <div :class="'tab-btn' + (agentTab==='custom'?' active':'')" @click="agentTab='custom'" style="padding:6px 16px;cursor:pointer;border-bottom:2px solid transparent;margin-bottom:-2px;font-size:14px;color:#999">🔧 自定义配置</div>
        </div>

        <!-- Quick Config -->
        <div v-show="agentTab==='quick'">
          <div class="form-group"><label>显示名称 *</label><input v-model="editAgent.display_name" placeholder="如：销售助手" /></div>
          <div class="form-group"><label>类型 *</label>
            <select v-model="editAgent.agent_type">
              <option value="api">API — 中转站标准 Agent（走模型计费）</option>
              <option value="hermes">Hermes — 商户自建 Agent（Hermes 桥接）</option>
            </select>
          </div>
          <div class="form-group"><label>分类 *</label>
            <select v-model="editAgent.category" @change="applyTemplate">
              <option value="">选择分类</option>
              <option v-for="c in categories" :key="c" :value="c">{{ c }}</option>
            </select>
          </div>
          <div v-if="editAgent.category" style="background:#f0f9ff;padding:8px 12px;border-radius:6px;margin:8px 0;font-size:13px;color:#555">
            ✅ 将使用「{{ editAgent.category }}」模板预设的能力和提示词
          </div>
          <div v-if="error" class="error">{{ error }}</div>
          <div class="modal-btns" style="margin-top:15px">
            <button @click="showAgent = false">取消</button>
            <button class="btn-primary" @click="saveQuickAgent" :disabled="saving">{{ saving ? '创建中...' : '创建 Agent' }}</button>
          </div>
          <div v-if="installCmd" style="margin-top:12px;background:#f6ffed;border:1px solid #b7eb8f;border-radius:6px;padding:12px">
            <div style="font-size:13px;color:#52c41a;margin-bottom:6px">✅ Agent 创建成功！</div>
            <div v-if="createdToken" style="margin-bottom:8px">
              <div style="font-size:12px;color:#555;margin-bottom:4px">🔑 Token（请妥善保存，仅显示一次）：</div>
              <code style="display:block;background:#fff;padding:8px;border-radius:4px;font-size:12px;word-break:break-all;overflow-x:auto">{{ createdToken }}</code>
              <button style="margin-top:4px;font-size:12px" @click="copyText(createdToken)">📋 复制 Token</button>
            </div>
            <div style="font-size:12px;color:#555;margin-bottom:4px">📦 {{ editAgent.agent_type === 'hermes' ? 'Hermes 桥接安装命令：' : '安装命令：' }}</div>
            <code style="display:block;background:#fff;padding:8px;border-radius:4px;font-size:12px;word-break:break-all;overflow-x:auto">{{ installCmd }}</code>
            <button style="margin-top:4px;font-size:12px" @click="copyText(installCmd)">📋 复制命令</button>
          </div>
        </div>

        <!-- Custom Config -->
        <div v-show="agentTab==='custom'">
          <div class="form-group"><label>Bot ID</label><input v-model="editAgent.bot_id" :disabled="!!editAgent.bot_id" placeholder="英文标识，如 my-agent" /></div>
          <div class="form-group"><label>显示名</label><input v-model="editAgent.display_name" placeholder="显示名称" /></div>
          <div class="form-group"><label>类型</label>
            <select v-model="editAgent.agent_type">
              <option value="api">API — 中转站标准 Agent</option>
              <option value="hermes">Hermes — 商户自建 Agent</option>
            </select>
          </div>
          <div class="form-group"><label>分类</label>
            <select v-model="editAgent.category" @change="applyTemplate">
              <option value="">选择分类模板</option>
              <option v-for="c in categories" :key="c" :value="c">{{ c }}</option>
            </select>
          </div>
          <div class="form-group"><label>模型</label>
            <select v-model="editAgent.model">
              <option v-for="m in proxyModels" :key="m.id" :value="m.name">{{ m.display_name || m.name }}</option>
            </select>
          </div>
          <div class="form-group"><label>System Prompt</label><textarea v-model="editAgent.system_prompt" rows="4" placeholder="定义 Agent 的角色和行为..."></textarea></div>
          <div class="form-group"><label>API Key</label><input v-model="editAgent.api_key" type="password" placeholder="API 密钥" /></div>
          <div class="form-group"><label>API URL</label><input v-model="editAgent.api_url" placeholder="如 https://ai.12fz.com/v1" /></div>
          <div class="form-group"><label>能力</label>
            <div class="checkbox-group">
              <label v-for="c in ['code','search','terminal','file','web','vision']" :key="c"><input type="checkbox" :value="c" v-model="editAgent.capabilities" /> {{ c }}</label>
            </div>
          </div>
          <div class="form-group"><label>状态</label><select v-model="editAgent.status"><option value="active">启用</option><option value="inactive">停用</option></select></div>
          <div v-if="editAgent.token" style="background:#f6ffed;border:1px solid #b7eb8f;border-radius:6px;padding:8px;margin-top:8px">
            <div style="font-size:12px;color:#52c41a">🔑 Token: <code style="background:#fff;padding:2px 6px;border-radius:3px">{{ editAgent.token }}</code></div>
          </div>
          <div v-if="error" class="error">{{ error }}</div>
          <div class="modal-btns">
            <button @click="showAgent = false">取消</button>
            <button class="btn-primary" @click="saveAgent" :disabled="saving">{{ saving ? '保存中...' : '保存' }}</button>
          </div>
        </div>
      </div>
    </div>

    <!-- Groups Modal -->
    <div v-if="showGroups" class="modal-overlay" @click.self="showGroups = false">
      <div class="modal">
        <h3>{{ selectedAgent?.display_name }} — 群聊绑定</h3>
        <div v-if="groups.length" class="group-list">
          <label v-for="g in groups" :key="g.id" class="group-item">
            <input type="checkbox" :value="g.id" v-model="selectedGroups" @change="saveGroupBinding" /> {{ g.name }}
          </label>
        </div>
        <div v-else>暂无群聊</div>
        <div class="modal-btns"><button @click="showGroups = false">关闭</button></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
async function request(url: string, opts: any = {}) {
  const token = localStorage.getItem('token') || localStorage.token || 'session-1'
  opts.headers = { 'Content-Type': 'application/json', ...opts.headers, Authorization: 'Bearer ' + token }
  const r = await fetch(url, opts)
  if (!r.ok) {
    const j = await r.json().catch(() => ({}))
    throw new Error(j.error || `HTTP ${r.status}`)
  }
  return r.json()
}

async function apiGet(path: string) {
  const token = localStorage.getItem('token') || ''
  const r = await fetch('/api' + path, { headers: { Authorization: 'Bearer ' + token } })
  if (!r.ok) {
    if (r.status === 401) throw new Error('未登录，请先登录后再访问')
    const j = await r.json().catch(() => ({}))
    throw new Error(j.error || r.statusText)
  }
  return r.json()
}
async function apiPut(path: string, body: any) {
  const token = localStorage.getItem('token') || ''
  const r = await fetch('/api' + path, { method: 'PUT', headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token }, body: JSON.stringify(body) })
  if (!r.ok) {
    if (r.status === 401) throw new Error('未登录，请先登录后再访问')
    const j = await r.json().catch(() => ({}))
    throw new Error(j.error || r.statusText)
  }
  return r.json()
}
async function apiPost(path: string, body: any) {
  const token = localStorage.getItem('token') || ''
  const r = await fetch('/api' + path, { method: 'POST', headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token }, body: JSON.stringify(body) })
  if (!r.ok) {
    if (r.status === 401) throw new Error('未登录，请先登录后再访问')
    const j = await r.json().catch(() => ({}))
    throw new Error(j.error || r.statusText)
  }
  return r.json()
}
async function apiDel(path: string) {
  const token = localStorage.getItem('token') || ''
  const r = await fetch('/api' + path, { method: 'DELETE', headers: { Authorization: 'Bearer ' + token } })
  if (!r.ok) {
    if (r.status === 401) throw new Error('未登录，请先登录后再访问')
    const j = await r.json().catch(() => ({}))
    throw new Error(j.error || r.statusText)
  }
  return r.json()
}

const agents = ref<any[]>([])
const groups = ref<any[]>([])
const showAgent = ref(false)
const showGroups = ref(false)
const selectedAgent = ref<any>(null)
const selectedGroups = ref<number[]>([])
const saving = ref(false)
const error = ref('')
const proxyModels = ref<any[]>([])
async function loadProxyModels() {
  try {
    const token = localStorage.getItem('token') || ''
    const resp = await fetch('/admin/proxy/models', { headers: { Authorization: 'Bearer ' + token } })
    if (resp.ok) proxyModels.value = await resp.json()
  } catch {}
}
const agentTab = ref('quick')
const installCmd = ref('')
const createdToken = ref('')

const editAgent = ref<any>({ bot_id: '', display_name: '', model: 'deepseek-v4-flash', system_prompt: '', capabilities: [], status: 'active', api_key: '', api_url: 'https://ai.12fz.com/v1', agent_type: 'api' })

onMounted(async () => {
  await loadProxyModels()
  await loadAgents()
  try { const { data } = await apiGet('/groups/my'); groups.value = data || [] } catch {}
})

async function changeModel(agent: any, newModel: string) {
  try {
    await apiPut('/agents/' + agent.bot_id, { ...agent, model: newModel })
    agent.model = newModel
  } catch(e: any) { alert('修改失败: ' + (e.message || '未知错误')) }
}

async function loadAgents() {
  try { const data = await apiGet('/agents'); agents.value = data || []
} catch(e: any) { error.value = e.message || '加载失败' }
}

const categories = ['办公','日常','销售','生产','编程','旅游','财务','策划运营','客服','农业','科技','教育','医疗','法律','设计','营销','物流','招聘','餐饮','房产','税务']
const categoryTemplates: Record<string,{p:string;c:string[]}> = {'办公':{p:'你是企业办公助手。帮助处理文档、会议室、日程、报销等。中文作答。',c:['chat','tools','memory']},'日常':{p:'你是通用日常助手。聊天、查资料、处理简单任务。中文回答。',c:['chat','tools']},'销售':{p:'你是金牌销售助手。客户管理、跟进线索、报价、数据分析。',c:['chat','tools','memory','search']},'生产':{p:'你是生产管理助手。排期、物料、质检、产能分析。',c:['chat','tools','terminal','file']},'编程':{p:'你是高级编程助手。代码审查、debug、架构设计。',c:['code','search','terminal','file','web']},'旅游':{p:'你是旅行规划助手。目的地推荐、行程规划、预算估算。',c:['chat','search','web']},'财务':{p:'你是财务分析助手。报表、预算、成本分析。',c:['chat','tools','file','memory']},'策划运营':{p:'你是策划运营助手。活动策划、内容运营、社群管理。',c:['chat','tools','search','web']},'客服':{p:'你是智能客服。礼貌耐心、解决用户问题、投诉升级。',c:['chat','memory']},'农业':{p:'你是智慧农业助手。种植管理、病虫害、气象预警。',c:['chat','tools','search']},'科技':{p:'你是科技资讯助手。科技动态、论文解读、专利分析。',c:['chat','search','web','tools']},'教育':{p:'你是教育辅导助手。备课、出题、答疑、学习规划。',c:['chat','tools','file','search']},'医疗':{p:'你是医疗知识助手。医学百科、症状自查(不替代医生)。',c:['chat','search']},'法律':{p:'你是法律顾问助手。合同审查、法规查询(不替代律师)。',c:['chat','search','tools','file']},'设计':{p:'你是设计助理。UI方案、配色建议、素材推荐。',c:['chat','web','tools']},'营销':{p:'你是营销策划助手。文案、投放策略、SEO、竞品分析。',c:['chat','tools','web','search']},'物流':{p:'你是物流管理助手。路径优化、仓储管理、运单跟踪。',c:['chat','tools','memory']},'招聘':{p:'你是招聘助手。JD撰写、简历筛选、面试题库。',c:['chat','tools','search','file']},'餐饮':{p:'你是餐饮运营助手。菜单设计、成本核算、排班管理。',c:['chat','tools']},'房产':{p:'你是房产顾问助手。房源匹配、价值评估、政策解读。',c:['chat','search','web','tools']},'税务':{p:'你是税务助手。申报流程、税收筹划、发票管理(不替代税务师)。',c:['chat','tools','file','search']}}
function applyTemplate() {
  const t = categoryTemplates[editAgent.value.category]
  if (t) { editAgent.value.system_prompt = t.p; editAgent.value.capabilities = t.c.slice() }
}

async function saveQuickAgent() {
  const a = editAgent.value
  if (!a.display_name) { error.value = '请输入显示名称'; return }
  if (!a.category) { error.value = '请选择分类'; return }
  if (!a.agent_type) { error.value = '请选择类型'; return }
  error.value = ''; saving.value = true
  const t = categoryTemplates[a.category]
  const body = {
    bot_id: a.bot_id || 'agent-' + Date.now(),
    display_name: a.display_name,
    model: a.model || 'deepseek-v4-flash',
    system_prompt: t ? t.p : (a.system_prompt || ''),
    capabilities: t ? t.c : (a.capabilities || []),
    category: a.category,
    status: 'active',
    api_key: a.api_key || '',
    api_url: a.api_url || 'https://ai.12fz.com/v1',
    agent_type: a.agent_type || 'api'
  }
  try {
    const res = await request('/api/agents', { method: 'POST', body: JSON.stringify(body) })
    // 不关闭弹窗，先展示 Token 和安装命令
    createdToken.value = res.token || ''
    if (res.agent_type === 'hermes') {
      installCmd.value = `sudo curl -s -o /usr/local/bin/12fz-hermes-bridge https://ai.12fz.com/hermes-bridge.py && sudo chmod +x /usr/local/bin/12fz-hermes-bridge && mkdir -p ~/.hermes && cat > ~/.hermes/12fz-bridge.json << 'HEREOF'\n{"bot_id":"${res.bot_id}","token":"${res.token}","ws_url":"wss://ai.12fz.com/ws","user_id":"1"}\nHEREOF\n/usr/local/bin/12fz-hermes-bridge &`
    } else {
      installCmd.value = `curl -s https://ai.12fz.com/install-agent.sh | bash -s -- --bot-id=${res.bot_id} --token=${res.token}`
    }
    await loadAgents()
  } catch(e: any) { error.value = e.message || '创建失败' }
  saving.value = false
}

function copyText(t: string) { navigator.clipboard.writeText(t).then(() => alert('已复制')) }

function openAgent(a?: any) {
  editAgent.value = a ? { ...a } : { bot_id: '', display_name: '', model: 'deepseek-v4-flash', system_prompt: '', capabilities: [], status: 'active', api_key: '', api_url: 'https://ai.12fz.com/v1', agent_type: 'api' }
  showAgent.value = true; error.value = ''; installCmd.value = ''; createdToken.value = ''
}

async function saveAgent() {
  saving.value = true; error.value = ''
  try {
    if (editAgent.value.bot_id) await apiPut(`/agents/${encodeURIComponent(editAgent.value.bot_id)}`, editAgent.value)
    else await apiPost('/agents', editAgent.value)
    showAgent.value = false; await loadAgents()
  } catch(e: any) { error.value = e.message || '保存失败' }
  finally { saving.value = false }
}

async function deleteAgent(a: any) {
  if (confirm(`确认删除 Agent "${a.display_name}"？`)) {
    try { await apiDel(`/agents/${encodeURIComponent(a.bot_id)}`); await loadAgents() } catch(e: any) { alert(e.message || '删除失败') }
  }
}

async function openGroups(a: any) {
  selectedAgent.value = a
  try { const { data } = await apiGet(`/agents/${encodeURIComponent(a.bot_id)}/groups`); selectedGroups.value = (data || []).map((g: any) => g.id) } catch { selectedGroups.value = [] }
  showGroups.value = true
}

async function saveGroupBinding() {
  if (!selectedAgent.value) return
  try { await apiPut(`/agents/${encodeURIComponent(selectedAgent.value.bot_id)}/groups`, { group_ids: selectedGroups.value }) } catch(e: any) { alert(e.response?.data?.error || '保存群聊绑定失败') }
}
</script>

<style scoped>
.admin-agents { padding: 20px; }
.admin-header { margin-bottom: 16px; }
.admin-nav { display: flex; gap: 12px; margin-bottom: 12px; }
.admin-nav a { color: #1890ff; text-decoration: none; font-size: 14px; }
.btn-primary { padding: 6px 16px; background: #1890ff; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 13px; }
.btn-primary:disabled { opacity: 0.5; }
.btn-danger { padding: 4px 10px; background: #fff; color: #f5222d; border: 1px solid #f5222d; border-radius: 4px; cursor: pointer; font-size: 12px; }
table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; margin-top: 12px; }
th, td { padding: 10px 14px; border-bottom: 1px solid #f0f0f0; text-align: left; font-size: 13px; }
th { background: #fafafa; font-weight: 500; }
.status-tag { padding: 2px 8px; border-radius: 10px; font-size: 12px; }
.status-tag.active { background: #f6ffed; color: #52c41a; }
.status-tag.inactive { background: #fff2f0; color: #f5222d; }
.type-tag { padding: 2px 8px; border-radius: 10px; font-size: 12px; }
.type-tag.type-api { background: #e6f7ff; color: #1890ff; }
.type-tag.type-hermes { background: #f9f0ff; color: #722ed1; }
.empty { text-align: center; color: #999; padding: 48px; }
.modal-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,.3); z-index: 1000; display: flex; align-items: center; justify-content: center; }
.modal { background: #fff; border-radius: 8px; padding: 24px; width: 520px; max-width: 90vw; max-height: 80vh; overflow-y: auto; }
.modal h3 { margin: 0 0 16px; }
.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 13px; color: #666; margin-bottom: 4px; }
.form-group input, .form-group select, .form-group textarea { width: 100%; padding: 6px 8px; border: 1px solid #d9d9d9; border-radius: 4px; font-size: 13px; box-sizing: border-box; }
.checkbox-group { display: flex; flex-wrap: wrap; gap: 8px; }
.checkbox-group label { font-size: 13px; cursor: pointer; }

.tab-btn.active { color: #1890ff !important; border-bottom-color: #1890ff !important; font-weight: 600; }
.tab-btn:not(.active):hover { color: #1890ff; }

.modal-btns { display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px; }
.error { color: #f5222d; font-size: 13px; margin-top: 8px; }
.group-list { max-height: 300px; overflow-y: auto; }
.group-item { display: block; padding: 6px 0; font-size: 13px; cursor: pointer; }
</style>

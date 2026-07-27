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
      <thead><tr><th>Bot ID</th><th>显示名</th><th>模型</th><th>状态</th><th>群聊</th><th>操作</th></tr></thead>
      <tbody>
        <tr v-for="a in agents" :key="a.bot_id">
          <td>{{ a.bot_id }}</td><td>{{ a.display_name }}</td><td>{{ a.model }}</td>
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
      <div class="modal">
        <h3>{{ editAgent.bot_id ? '编辑' : '新建' }} Agent</h3>
        <div class="form-group"><label>Bot ID</label><input v-model="editAgent.bot_id" :disabled="!!editAgent.bot_id" placeholder="英文标识，如 my-agent" /></div>
        <div class="form-group"><label>显示名</label><input v-model="editAgent.display_name" placeholder="显示名称" /></div>
        <div class="form-group"><label>分类</label>
          <select v-model="editAgent.category" @change="applyTemplate">
            <option value="">选择分类模板</option>
            <option v-for="c in categories" :key="c" :value="c">{{ c }}</option>
          </select>
        </div>
        <div class="form-group"><label>模型</label>
          <select v-model="editAgent.model">
            <option value="tk.12fz.com">tk.12fz.com (中转)</option>
            <option value="gpt-4o">GPT-4o</option>
            <option value="gpt-3.5-turbo">GPT-3.5</option>
            <option value="claude-sonnet-4">Claude Sonnet 4</option>
            <option value="deepseek-v3">DeepSeek V3</option>
            <option value="qwen-max">Qwen Max</option>
          </select>
        </div>
        <div class="form-group"><label>System Prompt</label><textarea v-model="editAgent.system_prompt" rows="4" placeholder="定义 Agent 的角色和行为..."></textarea></div>
        <div class="form-group"><label>API Key</label><input v-model="editAgent.api_key" type="password" placeholder="API 密钥" /></div>
        <div class="form-group"><label>API URL</label><input v-model="editAgent.api_url" placeholder="如 https://tk.12fz.com/v1" /></div>
        <div class="form-group"><label>能力</label>
          <div class="checkbox-group">
            <label v-for="c in ['code','search','terminal','file','web','vision']" :key="c"><input type="checkbox" :value="c" v-model="editAgent.capabilities" /> {{ c }}</label>
          </div>
        </div>
        <div class="form-group"><label>状态</label><select v-model="editAgent.status"><option value="active">启用</option><option value="inactive">停用</option></select></div>
        <div v-if="error" class="error">{{ error }}</div>
        <div class="modal-btns">
          <button @click="showAgent = false">取消</button>
          <button class="btn-primary" @click="saveAgent" :disabled="saving">{{ saving ? '保存中...' : '保存' }}</button>
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
import api from '../api'

const agents = ref<any[]>([])
const groups = ref<any[]>([])
const showAgent = ref(false)
const showGroups = ref(false)
const selectedAgent = ref<any>(null)
const selectedGroups = ref<number[]>([])
const saving = ref(false)
const error = ref('')

const editAgent = ref<any>({ bot_id: '', display_name: '', model: 'tk.12fz.com', system_prompt: '', capabilities: [], status: 'active', api_key: '', api_url: '' })

onMounted(async () => {
  await loadAgents()
  try { const { data } = await api.get('/groups/my'); groups.value = data || [] } catch {}
})

async function loadAgents() {
  try { const { data } = await api.get('/agents'); agents.value = data || [] } catch(e: any) { error.value = e.response?.data?.error || '加载失败' }
}

const categories = ['办公','日常','销售','生产','编程','旅游','财务','策划运营','客服','农业','科技','教育','医疗','法律','设计','营销','物流','招聘','餐饮','房产','税务']
const categoryTemplates: Record<string,{p:string;c:string[]}> = {'办公':{p:'你是企业办公助手。帮助处理文档、会议室、日程、报销等。中文作答。',c:['chat','tools','memory']},'日常':{p:'你是通用日常助手。聊天、查资料、处理简单任务。中文回答。',c:['chat','tools']},'销售':{p:'你是金牌销售助手。客户管理、跟进线索、报价、数据分析。',c:['chat','tools','memory','search']},'生产':{p:'你是生产管理助手。排期、物料、质检、产能分析。',c:['chat','tools','terminal','file']},'编程':{p:'你是高级编程助手。代码审查、debug、架构设计。',c:['code','search','terminal','file','web']},'旅游':{p:'你是旅行规划助手。目的地推荐、行程规划、预算估算。',c:['chat','search','web']},'财务':{p:'你是财务分析助手。报表、预算、成本分析。',c:['chat','tools','file','memory']},'策划运营':{p:'你是策划运营助手。活动策划、内容运营、社群管理。',c:['chat','tools','search','web']},'客服':{p:'你是智能客服。礼貌耐心、解决用户问题、投诉升级。',c:['chat','memory']},'农业':{p:'你是智慧农业助手。种植管理、病虫害、气象预警。',c:['chat','tools','search']},'科技':{p:'你是科技资讯助手。科技动态、论文解读、专利分析。',c:['chat','search','web','tools']},'教育':{p:'你是教育辅导助手。备课、出题、答疑、学习规划。',c:['chat','tools','file','search']},'医疗':{p:'你是医疗知识助手。医学百科、症状自查(不替代医生)。',c:['chat','search']},'法律':{p:'你是法律顾问助手。合同审查、法规查询(不替代律师)。',c:['chat','search','tools','file']},'设计':{p:'你是设计助理。UI方案、配色建议、素材推荐。',c:['chat','web','tools']},'营销':{p:'你是营销策划助手。文案、投放策略、SEO、竞品分析。',c:['chat','tools','web','search']},'物流':{p:'你是物流管理助手。路径优化、仓储管理、运单跟踪。',c:['chat','tools','memory']},'招聘':{p:'你是招聘助手。JD撰写、简历筛选、面试题库。',c:['chat','tools','search','file']},'餐饮':{p:'你是餐饮运营助手。菜单设计、成本核算、排班管理。',c:['chat','tools']},'房产':{p:'你是房产顾问助手。房源匹配、价值评估、政策解读。',c:['chat','search','web','tools']},'税务':{p:'你是税务助手。申报流程、税收筹划、发票管理(不替代税务师)。',c:['chat','tools','file','search']}}
function applyTemplate() {
  const t = categoryTemplates[editAgent.value.category]
  if (t) { editAgent.value.system_prompt = t.p; editAgent.value.capabilities = t.c.slice() }
}

function openAgent(a?: any) {
  editAgent.value = a ? { ...a } : { bot_id: '', display_name: '', model: 'tk.12fz.com', system_prompt: '', capabilities: [], status: 'active', api_key: '', api_url: '' }
  showAgent.value = true; error.value = ''
}

async function saveAgent() {
  saving.value = true; error.value = ''
  try {
    if (editAgent.value.bot_id) await api.put(`/agents/${encodeURIComponent(editAgent.value.bot_id)}`, editAgent.value)
    else await api.post('/agents', editAgent.value)
    showAgent.value = false; await loadAgents()
  } catch(e: any) { error.value = e.response?.data?.error || '保存失败' }
  finally { saving.value = false }
}

async function deleteAgent(a: any) {
  if (confirm(`确认删除 Agent "${a.display_name}"？`)) {
    try { await api.delete(`/agents/${encodeURIComponent(a.bot_id)}`); await loadAgents() } catch(e: any) { alert(e.response?.data?.error || '删除失败') }
  }
}

async function openGroups(a: any) {
  selectedAgent.value = a
  try { const { data } = await api.get(`/agents/${encodeURIComponent(a.bot_id)}/groups`); selectedGroups.value = (data || []).map((g: any) => g.id) } catch { selectedGroups.value = [] }
  showGroups.value = true
}

async function saveGroupBinding() {
  if (!selectedAgent.value) return
  try { await api.put(`/agents/${encodeURIComponent(selectedAgent.value.bot_id)}/groups`, { group_ids: selectedGroups.value }) } catch(e: any) { alert(e.response?.data?.error || '保存群聊绑定失败') }
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
.empty { text-align: center; color: #999; padding: 48px; }
.modal-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,.3); z-index: 1000; display: flex; align-items: center; justify-content: center; }
.modal { background: #fff; border-radius: 8px; padding: 24px; width: 520px; max-width: 90vw; max-height: 80vh; overflow-y: auto; }
.modal h3 { margin: 0 0 16px; }
.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 13px; color: #666; margin-bottom: 4px; }
.form-group input, .form-group select, .form-group textarea { width: 100%; padding: 6px 8px; border: 1px solid #d9d9d9; border-radius: 4px; font-size: 13px; box-sizing: border-box; }
.checkbox-group { display: flex; flex-wrap: wrap; gap: 8px; }
.checkbox-group label { font-size: 13px; cursor: pointer; }
.modal-btns { display: flex; gap: 8px; justify-content: flex-end; margin-top: 16px; }
.error { color: #f5222d; font-size: 13px; margin-top: 8px; }
.group-list { max-height: 300px; overflow-y: auto; }
.group-item { display: block; padding: 6px 0; font-size: 13px; cursor: pointer; }
</style>

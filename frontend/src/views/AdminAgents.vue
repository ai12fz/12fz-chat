<template>
  <div class="admin-agents">
    <div class="admin-header">
      <h2>Agent 管理</h2>
      <button class="btn-primary" @click="openCreate">+ 新建 Agent</button>
    </div>

    <table class="agent-table" v-if="agents.length">
      <thead>
        <tr>
          <th>Bot ID</th>
          <th>显示名</th>
          <th>模型</th>
          <th>状态</th>
          <th>群聊</th>
          <th>操作</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="a in agents" :key="a.bot_id">
          <td>{{ a.bot_id }}</td>
          <td>{{ a.display_name }}</td>
          <td>{{ a.model }}</td>
          <td><span :class="['status-tag', a.status]">{{ a.status }}</span></td>
          <td>{{ a.group_count || 0 }} 个群</td>
          <td class="actions">
            <button @click="openEdit(a)">编辑</button>
            <button @click="openGroups(a)">群聊</button>
            <button class="btn-danger" @click="delAgent(a)">删除</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">暂无 Agent，点击上方按钮创建</div>

    <!-- Edit/Create Modal -->
    <div class="modal-overlay" v-if="showForm" @click.self="showForm = false">
      <div class="modal">
        <h3>{{ editing ? '编辑' : '新建' }} Agent</h3>
        <div class="form-group">
          <label>Bot ID</label>
          <input v-model="form.bot_id" :disabled="!!editing" placeholder="英文标识，如 my-agent" />
        </div>
        <div class="form-group">
          <label>显示名</label>
          <input v-model="form.display_name" placeholder="显示名称" />
        </div>
        <div class="form-group">
          <label>模型</label>
          <select v-model="form.model">
            <option value="gpt-4">GPT-4</option>
            <option value="gpt-4o">GPT-4o</option>
            <option value="gpt-3.5-turbo">GPT-3.5</option>
            <option value="claude-sonnet-4">Claude Sonnet 4</option>
            <option value="deepseek-v3">DeepSeek V3</option>
            <option value="qwen-max">Qwen Max</option>
            <option value="tk.12fz.com">tk.12fz.com (中转)</option>
          </select>
        </div>
        <div class="form-group">
          <label>System Prompt</label>
          <textarea v-model="form.system_prompt" rows="4" placeholder="定义 Agent 的角色和行为..." />
        </div>
        <div class="form-group">
          <label>能力</label>
          <div class="checkbox-group">
            <label v-for="c in allCaps" :key="c"><input type="checkbox" :value="c" v-model="form.capabilities" /> {{ c }}</label>
          </div>
        </div>
        <div class="form-group">
          <label>状态</label>
          <select v-model="form.status">
            <option value="active">启用</option>
            <option value="inactive">停用</option>
          </select>
        </div>
        <div class="form-group" v-if="error">
          <span class="error-msg">{{ error }}</span>
        </div>
        <div class="modal-btns">
          <button @click="showForm = false">取消</button>
          <button class="btn-primary" @click="save" :disabled="saving">{{ saving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>

    <!-- Group Binding Modal -->
    <div class="modal-overlay" v-if="showGroups" @click.self="showGroups = false">
      <div class="modal">
        <h3>{{ groupAgent?.display_name }} — 群聊绑定</h3>
        <div class="group-list" v-if="allGroups.length">
          <label v-for="g in allGroups" :key="g.id" class="group-item">
            <input type="checkbox" :value="g.id" v-model="selectedGroups" @change="saveGroups" />
            {{ g.name }}
          </label>
        </div>
        <div v-else>暂无群聊</div>
        <div class="modal-btns">
          <button @click="showGroups = false">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import api from '../api'

interface Agent {
  id: number
  bot_id: string
  display_name: string
  model: string
  system_prompt: string
  capabilities: string[]
  status: string
  group_count?: number
}

interface Group {
  id: number
  name: string
}

const agents = ref<Agent[]>([])
const allGroups = ref<Group[]>([])
const allCaps = ['code', 'search', 'terminal', 'file', 'web', 'vision']
const showForm = ref(false)
const showGroups = ref(false)
const editing = ref<Agent | null>(null)
const groupAgent = ref<Agent | null>(null)
const selectedGroups = ref<number[]>([])
const saving = ref(false)
const error = ref('')

const form = ref<Agent>({
  id: 0, bot_id: '', display_name: '', model: 'gpt-4', system_prompt: '', capabilities: [], status: 'active'
})

async function load() {
  try {
    const { data } = await api.get('/agents')
    // Also load group counts per agent
    for (const a of data) {
      try {
        const { data: gs } = await api.get(`/agents/${encodeURIComponent(a.bot_id)}/groups`)
        a.group_count = gs.length
      } catch { a.group_count = 0 }
    }
    agents.value = data
  } catch (e: any) {
    error.value = e.response?.data?.error || '加载失败'
  }
}

async function loadGroups() {
  try {
    const { data } = await api.get('/groups/my')
    allGroups.value = data || []
  } catch {}
}

onMounted(() => { load(); loadGroups() })

function openCreate() {
  editing.value = null
  form.value = { id: 0, bot_id: '', display_name: '', model: 'gpt-4', system_prompt: '', capabilities: [], status: 'active' }
  error.value = ''
  showForm.value = true
}

function openEdit(a: Agent) {
  editing.value = a
  form.value = { ...a }
  error.value = ''
  showForm.value = true
}

async function save() {
  saving.value = true
  error.value = ''
  try {
    if (editing.value) {
      await api.put(`/agents/${encodeURIComponent(editing.value.bot_id)}`, form.value)
    } else {
      await api.post('/agents', form.value)
    }
    showForm.value = false
    await load()
  } catch (e: any) {
    error.value = e.response?.data?.error || '保存失败'
  } finally {
    saving.value = false
  }
}

async function delAgent(a: Agent) {
  if (!confirm(`确认删除 Agent "${a.display_name}"？`)) return
  try {
    await api.delete(`/agents/${encodeURIComponent(a.bot_id)}`)
    await load()
  } catch (e: any) {
    alert(e.response?.data?.error || '删除失败')
  }
}

async function openGroups(a: Agent) {
  groupAgent.value = a
  try {
    const { data } = await api.get(`/agents/${encodeURIComponent(a.bot_id)}/groups`)
    selectedGroups.value = data.map((g: Group) => g.id)
  } catch {
    selectedGroups.value = []
  }
  showGroups.value = true
}

async function saveGroups() {
  if (!groupAgent.value) return
  try {
    await api.put(`/agents/${encodeURIComponent(groupAgent.value.bot_id)}/groups`, {
      group_ids: selectedGroups.value
    })
  } catch (e: any) {
    alert(e.response?.data?.error || '保存群聊绑定失败')
  }
}
</script>

<style scoped>
.admin-agents { padding: 24px; max-width: 900px; margin: 0 auto; }
.admin-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 20px; }
.admin-header h2 { margin: 0; font-size: 20px; }

.agent-table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,.1); }
.agent-table th, .agent-table td { padding: 10px 14px; text-align: left; border-bottom: 1px solid #f0f0f0; font-size: 14px; }
.agent-table th { background: #fafafa; font-weight: 600; }
.agent-table .actions button { margin-right: 6px; padding: 4px 10px; border: 1px solid #d9d9d9; border-radius: 4px; background: #fff; cursor: pointer; font-size: 12px; }
.agent-table .actions button:hover { border-color: #1890ff; color: #1890ff; }
.btn-danger { color: #ff4d4f !important; border-color: #ff4d4f !important; }
.btn-danger:hover { background: #fff1f0 !important; }
.btn-primary { padding: 8px 16px; border: none; border-radius: 6px; background: #1890ff; color: #fff; cursor: pointer; font-size: 14px; }
.btn-primary:hover { background: #40a9ff; }

.status-tag { padding: 2px 8px; border-radius: 4px; font-size: 12px; }
.status-tag.active { background: #f6ffed; color: #52c41a; border: 1px solid #b7eb8f; }
.status-tag.inactive { background: #fff2f0; color: #ff4d4f; border: 1px solid #ffccc7; }

.empty { padding: 60px; text-align: center; color: #999; font-size: 16px; }

.modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,.45); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.modal { background: #fff; border-radius: 8px; padding: 24px; width: 480px; max-height: 80vh; overflow-y: auto; box-shadow: 0 4px 12px rgba(0,0,0,.15); }
.modal h3 { margin: 0 0 16px; font-size: 16px; }
.form-group { margin-bottom: 14px; }
.form-group label { display: block; margin-bottom: 4px; font-size: 13px; color: #555; }
.form-group input, .form-group select, .form-group textarea { width: 100%; padding: 8px 10px; border: 1px solid #d9d9d9; border-radius: 4px; font-size: 14px; box-sizing: border-box; }
.checkbox-group { display: flex; flex-wrap: wrap; gap: 12px; }
.checkbox-group label { display: flex; align-items: center; gap: 4px; font-size: 13px; cursor: pointer; }
.error-msg { color: #ff4d4f; font-size: 13px; }
.modal-btns { display: flex; justify-content: flex-end; gap: 8px; margin-top: 20px; }
.modal-btns button { padding: 6px 16px; border: 1px solid #d9d9d9; border-radius: 4px; background: #fff; cursor: pointer; }

.group-list { display: flex; flex-direction: column; gap: 8px; max-height: 300px; overflow-y: auto; }
.group-item { display: flex; align-items: center; gap: 8px; padding: 8px; border-radius: 4px; cursor: pointer; font-size: 14px; }
.group-item:hover { background: #f5f5f5; }
</style>

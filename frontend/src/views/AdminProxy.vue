<template>
  <div class="admin-proxy">
    <div class="admin-header">
      <h2>中转站管理</h2>
      <div class="admin-nav">
        <router-link to="/admin/devices">设备管理</router-link>
        <router-link to="/admin/proxy">中转站</router-link>
      </div>
      <div class="tabs">
        <button v-for="t in tabs" :key="t.id" :class="{ active: tab === t.id }" @click="tab = t.id">{{ t.name }}</button>
      </div>
    </div>

    <!-- Dashboard -->
    <div v-if="tab === 'dashboard'">
      <div style="margin-bottom:12px;display:flex;align-items:center;gap:8px">
        <label style="font-size:13px;color:#666">按Key过滤：</label>
        <select v-model="selectedKey" @change="loadDashboard" style="padding:4px 8px;border:1px solid #d9d9d9;border-radius:4px;font-size:13px">
          <option value="">全部Key</option>
          <option v-for="k in keys" :key="k.id" :value="k.id">{{ k.name || k.key_text?.slice(0,16) }}</option>
        </select>
      </div>
      <div class="cards">
        <div class="card" v-for="(v, k) in stats" :key="k">
          <div class="card-label">{{ k }}</div>
          <div class="card-num">{{ v.tokens }}<small> tk</small></div>
          <div class="card-sub">{{ v.calls }} 次 · ¥{{ (v.cost||0).toFixed(2) }}</div>
        </div>
      </div>
      <div class="chart-box" v-if="chartDates.length">
        <h4>近30天 Token 消耗（按模型）</h4>
        <div class="chart-wrap" style="position:relative">
          
          <div class="bar-chart">
            <div class="bar-col" v-for="d in chartDates" :key="d">
              <div class="bar-stack">
                <div v-for="(seg, i) in (chartMap[d]||[])" :key="i"
                  class="bar-seg" :style="{ height: segH(seg.tokens), background: seg.color }"
                  :title="seg.model+': '+seg.tokens+' tk'"></div>
              </div>
              <span class="bar-label">{{ d.slice(5) }}</span>
              <span v-if="d === chartDates[chartDates.length-1]" class="bar-ruler" :style="{ bottom: rulerY + 'px' }">{{ rulerLabel }}</span>
            </div>
          </div>
        </div>
      </div>
      <!-- 近30天消耗金额 -->
      <div class="chart-box" v-if="costDates.length">
        <h4>近30天 消耗金额（按模型）</h4>
        <div class="chart-wrap" style="position:relative">
          <div class="bar-chart">
            <div class="bar-col" v-for="d in costDates" :key="'c'+d">
              <div class="bar-stack">
                <div v-for="(seg, i) in (costMap[d]||[])" :key="i"
                  class="bar-seg" :style="{ height: ((seg.cost / costMax) * 120).toFixed(1) + 'px', background: seg.color }"
                  :title="seg.model+': ¥'+seg.cost.toFixed(2)"></div>
              </div>
              <span class="bar-label">{{ d.slice(5) }}</span>
            </div>
          </div>
        </div>
      </div>

    </div>

    <!-- Models -->
    <div v-if="tab === 'models'">
      <button class="btn-primary" @click="openModel()">添加模型</button>
      <table v-if="models.length">
        <thead><tr><th>名称</th><th>提供商</th><th>状态</th><th>优先级</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="m in models" :key="m.id">
            <td>{{ m.display_name }}<br/><small>{{ m.name }}</small></td>
            <td>{{ m.provider }}</td>
            <td>{{ m.status }}</td>
            <td>{{ m.priority }}</td>
            <td>
              <button @click="openModel(m)">编辑</button>
              <button @click="deleteModel(m)">删除</button>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-if="showModel" class="modal-overlay" @click.self="showModel = false">
        <div class="modal">
          <h3>{{ editModel.id ? '编辑' : '添加' }}模型</h3>
          <div class="form-group" v-for="field in ['name','display_name','provider','endpoint','api_key']" :key="field">
            <label>{{ field }}</label>
            <input v-model="editModel[field]" :type="field === 'api_key' ? 'password' : 'text'" />
          </div>
          <div class="form-group"><label>状态</label><select v-model="editModel.status"><option>active</option><option>disabled</option></select></div>
          <div class="form-group"><label>优先级</label><input type="number" v-model.number="editModel.priority" /></div>
          <div class="form-group"><label>RPM</label><input type="number" v-model.number="editModel.max_rpm" /></div>
          <button class="btn-primary" @click="saveModel">保存</button>
          <button @click="showModel = false">取消</button>
        </div>
      </div>
    </div>

    <!-- Pricing -->
    <div v-if="tab === 'pricing'">
      <div class="multibar">
        <span>倍数：<strong>{{ multiplier }}x</strong></span>
        <input type="range" min="1.2" max="10" step="0.1" v-model.number="multiplier" style="width:200px" />
        <button class="btn-primary" @click="applyMultiplier">应用倍数</button>
      </div>
      <table v-if="pricing.length">
        <thead><tr><th>科目</th><th>名称</th><th>官方价</th><th>售价</th><th>启用</th></tr></thead>
        <tbody>
          <tr v-for="p in pricing" :key="p.key">
            <td>{{ p.key }}</td><td>{{ p.name }}</td><td>¥{{ (p.official_amount||0).toFixed(4) }}</td>
            <td><input type="number" v-model.number="p.amount" style="width:80px" /></td>
            <td><input type="checkbox" v-model="p.active" /></td>
          </tr>
        </tbody>
      </table>
      <button class="btn-primary" @click="savePricing">保存全部</button>
    </div>

    <!-- Merchants -->
    <div v-if="tab === 'merchants'">
      <table v-if="merchants.length">
        <thead><tr><th>商户ID</th><th>余额</th><th>累计消费</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="m in merchants" :key="m.org_id">
            <td>{{ m.name }}<br/><small>{{ m.org_id }}</small></td>
            <td>¥{{ (m.balance||0).toFixed(2) }}</td>
            <td>¥{{ (m.total_used||0).toFixed(2) }}</td>
            <td><button @click="recharge(m)">充值</button><button @click="showLedger(m)">流水</button></td>
          </tr>
        </tbody>
      </table>
      <div v-if="ledger.length" class="modal-overlay" @click.self="ledger = []">
        <div class="modal"><h3>流水</h3><table><tr v-for="l in ledger"><td>{{ l.direction }}</td><td>¥{{ l.amount_cny }}</td><td>{{ l.biz_type }}</td><td>{{ (l.created_at||'').slice(0,16) }}</td></tr></table><button @click="ledger = []">关闭</button></div>
      </div>
    </div>

    <!-- Keys -->
    <div v-if="tab === 'keys'">
      <div class="form-row">
        <input v-model="keyOrgId" placeholder="商户ID" />
        <input v-model="keyName" placeholder="名称" />
        <button class="btn-primary" @click="createKey">创建Key</button>
      </div>
      <table v-if="keys.length">
        <thead><tr><th>Key</th><th>设备</th><th>状态</th><th>操作</th></tr></thead>
        <tbody>
          <tr v-for="k in keys" :key="k.id">
            <td><code>{{ (k.key_text||'').slice(0,16) }}...</code></td>
            <td>{{ k.device_id || '通用' }}</td>
            <td>{{ k.status }}</td>
            <td><button @click="revokeKey(k)">吊销</button></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import api from '../api'

const tabs = [{id:'dashboard',name:'看板'},{id:'models',name:'模型管理'},{id:'pricing',name:'定价配置'},{id:'merchants',name:'商户管理'},{id:'keys',name:'Key管理'}]
const tab = ref('dashboard')

const stats = ref<any>({})
const keys = ref<any[]>([])
const selectedKey = ref('')
const dailyData = ref<any[]>([])
const chartDates = ref<string[]>([])
const chartMap = ref<Record<string,any[]>>({})
const costDates = ref<string[]>([])
const costMap = ref<Record<string,any[]>>({})
const costMax = ref(1)
function buildCostChart(daily) {
  const mc = {}
  let ci = 0
  const dateMap = {}
  for (const r of daily) {
    if (!mc[r.model]) mc[r.model] = MODEL_COLORS[ci++ % MODEL_COLORS.length]
    if (!dateMap[r.date]) dateMap[r.date] = []
    dateMap[r.date].push({ model: r.model, cost: r.cost || 0, color: mc[r.model] })
  }
  costDates.value = Object.keys(dateMap).sort()
  costMap.value = dateMap
  costMax.value = Math.max(0.01, ...daily.map(function(r) { return r.cost || 0 }))
}
const chartMax = ref(1)
const refValue = ref(0)
const refLineY = ref(40)
const rulerY = ref(0)
const rulerLabel = ref("0")
const MODEL_COLORS = ["#1890ff","#52c41a","#fa8c16","#eb2f96","#722ed1","#13c2c2","#f5222d","#faad14"]
const models = ref<any[]>([])
const pricing = ref<any[]>([])
const multiplier = ref(2)
const merchants = ref<any[]>([])
const ledger = ref<any[]>([])

const showModel = ref(false)
const editModel = ref<any>({name:'',display_name:'',provider:'',endpoint:'',api_key:'',status:'active',priority:0,max_rpm:60})
const keyOrgId = ref('00000000-0000-0000-0000-000000000000')
const keyName = ref('')

async function call(url: string, opts: any = {}) {
  return (await fetch(url, { headers: { 'Content-Type': 'application/json', Authorization: localStorage.getItem('token') || '' }, ...opts })).json()
}

watch(tab, async t => {
  if (t === 'dashboard') {
    keys.value = await call('/api/admin/proxy/keys')
    await loadDashboard()
  }
  if (t === 'models') models.value = await call('/api/admin/proxy/models')
  if (t === 'pricing') { const d = await call('/api/admin/proxy/pricing'); pricing.value = d.items || []; multiplier.value = d.multiplier || 2 }
  if (t === 'merchants') merchants.value = await call('/api/admin/proxy/merchants')
  if (t === 'keys') keys.value = await call('/api/admin/proxy/keys?org_id=' + keyOrgId.value)
}, { immediate: true })

async function loadDashboard() {
  const q = selectedKey.value ? '?key_id=' + selectedKey.value : ''
  const d = await call('/api/admin/proxy/dashboard' + q)
  stats.value = { '今日': d.today, '本月': d.month }
  dailyData.value = d.daily || []
  // Build chart data grouped by date+model
  const dateMap: Record<string,Record<string,number>> = {}
  const models = new Set<string>()
  for (const r of dailyData.value) {
    if (!dateMap[r.date]) dateMap[r.date] = {}
    dateMap[r.date][r.model||'unknown'] = (dateMap[r.date][r.model||'unknown']||0) + r.tokens
    models.add(r.model||'unknown')
  }
  chartDates.value = Object.keys(dateMap).sort()
  chartMap.value = {}
  const colors = [...MODEL_COLORS]
  const modelColor: Record<string,string> = {}
  let ci = 0
  for (const m of models) { modelColor[m] = colors[ci++ % colors.length] }
  chartMax.value = 1
  for (const date of chartDates.value) {
    chartMap.value[date] = Object.entries(dateMap[date]).map(([model, tokens]) => ({
      model, tokens, color: modelColor[model]
    })).sort((a,b) => b.tokens - a.tokens)
    const total = Object.values(dateMap[date]).reduce((s,v)=>s+v,0)
    if (total > chartMax.value) chartMax.value = total
  }
  // Reference line = average daily tokens
  const totals = chartDates.value.map(d => Object.values(dateMap[d]).reduce((s,v)=>s+v,0))
  refValue.value = totals.length ? totals.reduce((s,v)=>s+v,0) / totals.length : 0
  refLineY.value = 34 + (refValue.value / chartMax.value) * 140
  // Ruler: show total tokens on last bar
  const lastDate = chartDates.value[chartDates.value.length-1]
  if (lastDate && chartMap.value[lastDate]) {
    const lastTotal = chartMap.value[lastDate].reduce((s:number, x:any) => s + x.tokens, 0)
    rulerLabel.value = lastTotal >= 1000 ? (lastTotal/1000).toFixed(1)+'k' : String(lastTotal)
    rulerY.value = 34 + (lastTotal / chartMax.value) * 140
  }
  buildCostChart(dailyData.value)
}
  if (t === 'models') models.value = await call('/api/admin/proxy/models')
  if (t === 'pricing') { const d = await call('/api/admin/proxy/pricing'); pricing.value = d.items || []; multiplier.value = d.multiplier || 2 }
  if (t === 'merchants') merchants.value = await call('/api/admin/proxy/merchants')

function openModel(m?: any) { editModel.value = m ? { ...m } : { name: '', display_name: '', provider: '', endpoint: '', api_key: '', status: 'active', priority: 0, max_rpm: 60 }; showModel.value = true }
async function saveModel() {
  const m = editModel.value
  if (m.id) await call('/api/admin/proxy/models/' + m.id, { method: 'PUT', body: JSON.stringify(m) })
  else await call('/api/admin/proxy/models', { method: 'POST', body: JSON.stringify(m) })
  showModel.value = false; models.value = await call('/api/admin/proxy/models')
}
async function deleteModel(m: any) { if (confirm('删除 ' + m.display_name + '?')) { await call('/api/admin/proxy/models/' + m.id, { method: 'DELETE' }); models.value = await call('/api/admin/proxy/models') } }
async function savePricing() { for (const p of pricing.value) if (p.key !== 'pricing_multiplier') await call('/api/admin/proxy/pricing/' + p.key, { method: 'PUT', body: JSON.stringify({ amount: p.amount, active: p.active }) }); alert('已保存') }
async function applyMultiplier() { await call('/api/admin/proxy/pricing/multiplier', { method: 'PUT', body: JSON.stringify({ multiplier: multiplier.value }) }); pricing.value = (await call('/api/admin/proxy/pricing')).items || []; alert('倍数已应用') }
async function recharge(m: any) { const amount = prompt('充值金额(元)'); if (amount) { await call('/api/admin/proxy/merchants/' + m.org_id + '/recharge', { method: 'POST', body: JSON.stringify({ amount: parseFloat(amount) }) }); merchants.value = await call('/api/admin/proxy/merchants') } }
async function showLedger(m: any) { ledger.value = await call('/api/admin/proxy/merchants/' + m.org_id + '/ledger') }
async function createKey() { const r = await call('/api/admin/proxy/keys', { method: 'POST', body: JSON.stringify({ org_id: keyOrgId.value, name: keyName.value }) }); alert('Key: ' + (r as any).key_text); keys.value = await call('/api/admin/proxy/keys?org_id=' + keyOrgId.value) }
async function revokeKey(k: any) { await call('/api/admin/proxy/keys/' + k.id + '/revoke', { method: 'POST' }); keys.value = await call('/api/admin/proxy/keys?org_id=' + k.org_id) }
function segH(v: number) { return Math.max((v/chartMax.value)*140, 2)+'px' }

</script>

<style scoped>
.admin-proxy { padding: 20px; }
.admin-header { margin-bottom: 16px; }
.admin-nav { display: flex; gap: 12px; margin-bottom: 12px; }
.admin-nav a { color: #1890ff; text-decoration: none; font-size: 14px; }
.tabs { display: flex; gap: 4px; margin-bottom: 16px; }
.tabs button { padding: 6px 14px; border: 1px solid #d9d9d9; background: #fff; border-radius: 4px; cursor: pointer; font-size: 13px; }
.tabs button.active { background: #1890ff; color: #fff; border-color: #1890ff; }
.btn-primary { padding: 6px 16px; background: #1890ff; color: #fff; border: none; border-radius: 4px; cursor: pointer; font-size: 13px; margin-right: 8px; }
table { width: 100%; border-collapse: collapse; background: #fff; border-radius: 8px; overflow: hidden; margin-top: 12px; }
th, td { padding: 10px 14px; border-bottom: 1px solid #f0f0f0; text-align: left; font-size: 13px; }
th { background: #fafafa; font-weight: 500; }
.cards { display: flex; gap: 12px; flex-wrap: wrap; }
.card-label { font-size: 13px; color: #888; margin-bottom: 4px; }
.card-num { font-size: 32px; font-weight: 400; line-height: 1.1; }
.card-num small { font-size: 14px; font-weight: 400; color: #999; }
.card-sub { font-size: 13px; color: #666; margin-top: 4px; }
.chart-box { margin-top: 24px; background: #fff; border-radius: 8px; padding: 20px; }
.chart-box h4 { margin: 0 0 16px; font-size: 14px; color: #666; }
.chart-wrap { position: relative; padding-bottom: 30px; }
.ref-line { position: absolute; left: 0; right: 0; border-top: 2px dashed #f5222d; font-size: 11px; color: #999; text-align: right; padding-right: 4px; z-index: 1; pointer-events: none; }
.bar-chart { display: flex; align-items: flex-end; gap: 1px; height: 180px; padding: 0 0 4px; }
.bar-col { flex: 1; display: flex; flex-direction: column; align-items: center; min-width: 0; height: 100%; justify-content: flex-end; }
.bar-stack { width: 100%; max-width: 14px; display: flex; flex-direction: column-reverse; }
.bar-seg { width: 100%; border-radius: 1px; transition: opacity .2s; }
.bar-seg:hover { opacity: 0.7; }
.bar-ruler { position: absolute; right: -8px; font-size: 10px; color: #999; font-weight: 600; white-space: nowrap; pointer-events: none; }
.bar-label { font-size: 8px; color: #999; margin-top: 4px; transform: rotate(-60deg); white-space: nowrap; transform-origin: top left; }
.card { padding: 20px; font-size: 24px; font-weight: 600; line-height: 1.6; background: #fff; border-radius: 8px; flex: 1; min-width: 200px; }
.form-group { margin-bottom: 12px; }
.form-group label { display: block; font-size: 13px; color: #666; margin-bottom: 4px; }
.form-group input, .form-group select { width: 100%; padding: 6px 8px; border: 1px solid #d9d9d9; border-radius: 4px; font-size: 13px; box-sizing: border-box; }
.modal-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,.3); z-index: 1000; display: flex; align-items: center; justify-content: center; }
.modal { background: #fff; border-radius: 8px; padding: 24px; width: 500px; max-width: 90vw; max-height: 80vh; overflow-y: auto; }
.multibar { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.form-row { display: flex; gap: 8px; margin-bottom: 12px; }
.form-row input { padding: 6px 8px; border: 1px solid #d9d9d9; border-radius: 4px; font-size: 13px; flex: 1; }
</style>

package handler

import (
	"encoding/json"
	"net/http"
)

// TrackEvent records a client-side analytics event (merged from the old
// standalone 3009 analytics service into chat-token).
func (h *HTTPHandler) TrackEvent(w http.ResponseWriter, r *http.Request) {
	var data map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&data); err != nil {
		jsonError(w, "bad request", 400)
		return
	}
	ip := clientIP(r)
	app, _ := data["app"].(string)
	if app == "" {
		app = "web"
	}
	event, _ := data["event"].(string)
	page, _ := data["page"].(string)
	title, _ := data["title"].(string)
	source, _ := data["source"].(string)
	sourceType, _ := data["source_type"].(string)
	userID, _ := data["user_id"].(string)
	sessionID, _ := data["session_id"].(string)
	extra, _ := data["extra"].(map[string]interface{})
	if extra == nil {
		extra = map[string]interface{}{}
	}

	if err := h.db.InsertAnalyticsEvent(r.Context(), app, event, page, title, source, sourceType, userID, sessionID, ip, extra); err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, map[string]interface{}{"ok": true}, 200)
}

// AnalyticsOverview returns aggregate stats for the dashboard.
func (h *HTTPHandler) AnalyticsOverview(w http.ResponseWriter, r *http.Request) {
	days := 7
	if r.URL.Query().Get("range") == "30d" {
		days = 30
	}
	stats, err := h.db.AnalyticsOverview(r.Context(), days)
	if err != nil {
		jsonError(w, err.Error(), 500)
		return
	}
	jsonResp(w, stats, 200)
}

// AnalyticsPage renders the embedded stats dashboard (same UI as the old service).
func (h *HTTPHandler) AnalyticsPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(analyticsHTML))
}

func clientIP(r *http.Request) string {
	ip := r.Header.Get("X-Forwarded-For")
	if ip == "" {
		ip = r.RemoteAddr
	}
	if len(ip) > 45 {
		ip = ip[:45]
	}
	return ip
}

const analyticsHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>统计分析</title>
<script src="https://cdn.jsdelivr.net/npm/chart.js@4"></script>
<style>
body{font-family:-apple-system,BlinkMacSystemFont,'Segoe UI',sans-serif;background:#f5f6fa;color:#333;margin:0}
.header{background:linear-gradient(135deg,#4a6cf7,#6a3de8);color:#fff;padding:24px 32px}
.header h1{font-size:22px;font-weight:600;margin:0 0 4px}
.header p{font-size:13px;opacity:.8;margin:0}
.container{max-width:1200px;margin:0 auto;padding:20px}
.row{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:16px;margin-bottom:24px}
.card{background:#fff;border-radius:12px;padding:20px;box-shadow:0 2px 8px rgba(0,0,0,.06)}
.card .label{font-size:13px;color:#888;margin-bottom:4px}
.card .value{font-size:28px;font-weight:700}
.card .value.pv{color:#4a6cf7}.card .value.uv{color:#6a3de8}.card .value.tools{color:#f97316}
.charts{display:grid;grid-template-columns:1fr 1fr;gap:16px;margin-bottom:24px}
.charts .card{grid-column:span 2}
.charts .card.half{grid-column:span 1}
table{width:100%;border-collapse:collapse;font-size:14px}
th,td{text-align:left;padding:10px 12px;border-bottom:1px solid #eee}
th{color:#888;font-weight:500;font-size:12px;text-transform:uppercase}
td:last-child{text-align:right;font-weight:600}
.bar{height:6px;border-radius:3px;background:#eef0f7;overflow:hidden;margin-top:6px}
.bar .fill{height:100%;border-radius:3px;background:linear-gradient(90deg,#4a6cf7,#6a3de8)}
.toolbar{display:flex;gap:8px;margin-bottom:16px}
.toolbar button{padding:6px 16px;border:1px solid #ddd;border-radius:6px;background:#fff;cursor:pointer;font-size:13px}
.toolbar button.active{border-color:#4a6cf7;background:#eef1ff;color:#4a6cf7;font-weight:600}
.empty{text-align:center;padding:40px;color:#999;font-size:14px}
</style>
</head>
<body>
<div class="header"><h1>📊 统计分析</h1><p id="subtitle">加载中…</p></div>
<div class="container">
<div class="toolbar"><button class="active" onclick="loadData('7d')">近 7 天</button><button onclick="loadData('30d')">近 30 天</button></div>
<div class="row">
<div class="card"><div class="label">页面浏览量 (PV)</div><div class="value pv" id="pv">-</div></div>
<div class="card"><div class="label">访客数 (UV)</div><div class="value uv" id="uv">-</div></div>
<div class="card"><div class="label">工具调用</div><div class="value tools" id="tools">-</div></div>
</div>
<div class="charts">
<div class="card half"><h3 style="font-size:15px;margin-bottom:12px">来源分布</h3><canvas id="sourceChart" height="200"></canvas></div>
<div class="card half"><h3 style="font-size:15px;margin-bottom:12px">热门页面</h3><div id="topPages"><div class="empty">暂无数据</div></div></div>
</div>
</div>
<script>
let sourceChart=null;
async function loadData(range){
document.querySelectorAll('.toolbar button').forEach(b=>b.classList.remove('active'));
(event?.target)?.classList?.add('active')||document.querySelector('.toolbar button')?.classList?.add('active');
try{
const r=await fetch('/api/v1/analytics/overview?range='+(range||'7d'));
const d=await r.json();
const o=d.overview||{};
document.getElementById('pv').textContent=o.pv??'-';
document.getElementById('uv').textContent=o.uv??'-';
document.getElementById('tools').textContent=o.tools??'-';
document.getElementById('subtitle').textContent='共 '+(o.pv??0)+' 次访问 · '+(range==='30d'?'近 30 天':'近 7 天');
const pagesEl=document.getElementById('topPages');
if(d.top_pages&&d.top_pages.length){
const maxV=Math.max(...d.top_pages.map(p=>p.views));
pagesEl.innerHTML='<table><tr><th>页面</th><th style="text-align:right">访问量</th></tr>'+d.top_pages.map(p=>'<tr><td>'+p.page+'<div class="bar"><div class="fill" style="width:'+Math.round(p.views/maxV*100)+'%"></div></div></td><td>'+p.views+'</td></tr>').join('')+'</table>';
}else pagesEl.innerHTML='<div class="empty">暂无数据</div>';
const srcEl=document.getElementById('sourceChart');
if(sourceChart){sourceChart.destroy();sourceChart=null}
if(d.sources&&d.sources.length){
const labels=d.sources.map(s=>s.type||'未知');
const vals=d.sources.map(s=>s.count);
const colors=['#4a6cf7','#6a3de8','#f97316','#10b981','#f43f5e','#8b5cf6','#06b6d4','#eab308'];
sourceChart=new Chart(srcEl,{type:'doughnut',data:{labels,datasets:[{data:vals,backgroundColor:colors.slice(0,labels.length),borderWidth:0}]},options:{responsive:true,plugins:{legend:{position:'bottom',labels:{padding:12,font:{size:12}}}}}});
}
}catch(e){document.getElementById('subtitle').textContent='加载失败'}
}
loadData('7d');
</script>
</body>
</html>`

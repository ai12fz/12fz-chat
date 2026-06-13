<template>
  <div class="login-page">
    <!-- 背景星光 -->
    <div class="bg-stars"></div>
    <!-- 浮动AI图标 -->
    <div class="ai-icons">
      <div class="ai-icon" style="top:10%;left:13%;animation-delay:0s;width:72px;height:72px;color:#00e5ff">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="14" rx="3"/><circle cx="8" cy="10" r="1.5" fill="currentColor"/><circle cx="16" cy="10" r="1.5" fill="currentColor"/><path d="M6 17v3h12v-3"/><line x1="9" y1="3" x2="9" y2="1"/><line x1="15" y1="3" x2="15" y2="1"/></svg>
        <span class="icon-label">AI Chat</span>
      </div>
      <div class="ai-icon" style="top:18%;right:7%;animation-delay:0.5s;width:60px;height:60px;color:#00e5a0">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="4" y="4" width="16" height="16" rx="4"/><path d="M9 10h6"/><path d="M9 14h6"/><circle cx="8" cy="10" r="1" fill="currentColor"/></svg>
        <span class="icon-label">智能</span>
      </div>
      <div class="ai-icon" style="bottom:18%;left:11%;animation-delay:1s;width:66px;height:66px;color:#40a9ff">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="8" r="4"/><path d="M20 21c0-4.418-3.582-8-8-8s-8 3.582-8 8"/><path d="M5 5L3 7"/><path d="M19 5l2 2"/><path d="M12 2V0"/></svg>
        <span class="icon-label">协作</span>
      </div>
      <div class="ai-icon" style="bottom:25%;right:3%;animation-delay:0.3s;width:54px;height:54px;color:#1e90ff">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>
        <span class="icon-label">极速</span>
      </div>
      <div class="ai-icon" style="top:42%;left:5%;animation-delay:0.8s;width:48px;height:48px;color:#6c5ce7">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 2a3 3 0 0 0-3 3v1H8a3 3 0 0 0-3 3v1H4a3 3 0 0 0-3 3v2a3 3 0 0 0 3 3h1v1a3 3 0 0 0 3 3h1v1a3 3 0 0 0 3 3h2a3 3 0 0 0 3-3v-1h1a3 3 0 0 0 3-3v-1h1a3 3 0 0 0 3-3v-2a3 3 0 0 0-3-3h-1v-1a3 3 0 0 0-3-3h-1V5a3 3 0 0 0-3-3h-2z"/><circle cx="12" cy="12" r="3" fill="currentColor"/><line x1="12" y1="2" x2="12" y2="5"/><line x1="12" y1="19" x2="12" y2="22"/><line x1="2" y1="12" x2="5" y2="12"/><line x1="19" y1="12" x2="22" y2="12"/></svg>
        <span class="icon-label">AI</span>
      </div>
      <div class="ai-icon" style="top:50%;right:0%;animation-delay:1.2s;width:57px;height:57px;color:#a855f7">
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="5" width="18" height="16" rx="3"/><circle cx="8" cy="11" r="1.5" fill="currentColor"/><circle cx="16" cy="11" r="1.5" fill="currentColor"/><path d="M8 16c2 1.5 6 1.5 8 0"/><line x1="12" y1="3" x2="12" y2="5"/></svg>
        <span class="icon-label">数据</span>
      </div>
    </div>

    <div class="user-login">
      <div class="user-login-logo">
        <svg viewBox="0 0 48 48" width="85" height="85" fill="none">
          <rect x="4" y="8" width="32" height="28" rx="4" stroke="rgba(255,255,255,0.9)" stroke-width="2" fill="rgba(255,255,255,0.05)"/>
          <circle cx="14" cy="22" r="2" fill="rgba(255,255,255,0.7)"/>
          <circle cx="22" cy="22" r="2" fill="rgba(255,255,255,0.7)"/>
          <line x1="30" y1="20" x2="36" y2="20" stroke="rgba(255,255,255,0.5)" stroke-width="1.5"/>
          <line x1="30" y1="26" x2="36" y2="26" stroke="rgba(255,255,255,0.5)" stroke-width="1.5"/>
          <path d="M10 36l-4 4v-4" stroke="rgba(255,255,255,0.6)" stroke-width="1.5" fill="none"/>
        </svg>
      </div>
      <div class="user-login-main">
        <form @submit.prevent="handleLogin" class="data-form">
          <h6>12FZ Chat · 登录</h6>
          <div class="data-form-item">
            <div class="form-input-item form-input-item-white">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="#999" stroke-width="2" style="margin:6px;float:left"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
              <input v-model="username" placeholder="登录用户名" autocomplete="username" />
            </div>
          </div>
          <div class="data-form-item">
            <div class="form-input-item form-input-item-white">
              <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="#999" stroke-width="2" style="margin:6px;float:left"><rect x="3" y="11" width="18" height="11" rx="2" ry="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
              <input v-model="password" type="password" placeholder="登录密码" autocomplete="current-password" />
            </div>
          </div>
          <div class="data-form-item">
            <div class="form-input-item form-input-item-white" style="display:flex;align-items:center;gap:4px;padding:0 6px">
              <input v-model="captchaAnswer" placeholder="验证码" style="flex:1;height:30px;border:none;outline:none;font-size:14px;background:transparent;min-width:0" />
              <div v-html="captchaSVG" @click="loadCaptcha" style="cursor:pointer;flex-shrink:0;border-left:1px solid #eee;padding-left:6px;line-height:0;transform:scale(0.5);transform-origin:left center;width:80px;height:30px;overflow:hidden" title="点击刷新"></div>
            </div>
          </div>
          <div v-if="error" class="error">{{ error }}</div>
          <div class="data-form-item" style="margin-top:10px">
            <div class="form-input-item form-input-item-normal">
              <label class="login-btn" :class="{ disabled: loading }" @click="handleLogin">
                {{ loading ? '登录中...' : '登录' }}
              </label>
            </div>
          </div>
        </form>
      </div>
    </div>
    <div class="bottom-copyright">粤ICP备19115165号</div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const captchaAnswer = ref('')
const captchaID = ref('')
const captchaSVG = ref('')
const loading = ref(false)
const error = ref('')

async function loadCaptcha() {
  try {
    const res = await fetch('/api/captcha')
    const data = await res.json()
    captchaID.value = data.captcha_id
    captchaSVG.value = data.svg
  } catch {
    captchaSVG.value = '<span style="color:#fff;font-size:12px">加载失败，点击重试</span>'
  }
}

async function handleLogin() {
  if (loading.value) return
  error.value = ''
  if (!captchaAnswer.value.trim()) {
    error.value = '请输入验证码'
    return
  }
  loading.value = true
  try {
    await auth.login(username.value, password.value, captchaID.value, parseInt(captchaAnswer.value))
    router.push('/chat')
  } catch (e: any) {
    error.value = e.response?.data?.error || '登录失败'
    loadCaptcha() // Refresh captcha on failure
    captchaAnswer.value = ''
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadCaptcha()
})
</script>

<style scoped>
.login-page {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #27cccd, #31add5, #3f81ed, #4b59e9, #4b41db);
  position: relative;
  overflow: hidden;
}

/* 背景星光粒子 */
.bg-stars {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  pointer-events: none;
  z-index: 0;
  background:
    radial-gradient(2px 2px at 15% 25%, rgba(255,255,255,0.12), transparent),
    radial-gradient(2px 2px at 35% 55%, rgba(255,255,255,0.08), transparent),
    radial-gradient(3px 3px at 65% 15%, rgba(255,255,255,0.1), transparent),
    radial-gradient(2px 2px at 80% 70%, rgba(255,255,255,0.06), transparent),
    radial-gradient(2px 2px at 45% 85%, rgba(255,255,255,0.08), transparent),
    radial-gradient(3px 3px at 10% 80%, rgba(255,255,255,0.1), transparent),
    radial-gradient(2px 2px at 90% 40%, rgba(255,255,255,0.06), transparent),
    radial-gradient(2px 2px at 55% 10%, rgba(255,255,255,0.08), transparent);
}

/* 浮动AI图标 */
.ai-icons {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  pointer-events: none;
  z-index: 1;
}
.ai-icon {
  position: absolute;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 0.3s ease;
  filter: drop-shadow(0 4px 12px rgba(0,0,0,0.3));
}
.ai-icon svg {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.ai-icon .icon-label {
  position: absolute;
  bottom: -22px;
  left: 50%;
  transform: translateX(-50%);
  font-size: 12px;
  color: rgba(255,255,255,0.5);
  white-space: nowrap;
  font-weight: 600;
  letter-spacing: 0.5px;
}

@keyframes float1 {
  0%   { transform: translate(0px, 0px); }
  25%  { transform: translate(-12px, -8px); }
  50%  { transform: translate(8px, -20px); }
  75%  { transform: translate(-6px, -10px); }
  100% { transform: translate(0px, 0px); }
}
@keyframes float2 {
  0%   { transform: translate(0px, 0px); }
  25%  { transform: translate(10px, -12px); }
  50%  { transform: translate(-14px, -15px); }
  75%  { transform: translate(6px, -6px); }
  100% { transform: translate(0px, 0px); }
}
@keyframes float3 {
  0%   { transform: translate(0px, 0px); }
  25%  { transform: translate(-8px, -18px); }
  50%  { transform: translate(16px, -25px); }
  75%  { transform: translate(-10px, -12px); }
  100% { transform: translate(0px, 0px); }
}
.ai-icon:nth-child(1) { animation: float1 6s ease-in-out infinite; }
.ai-icon:nth-child(2) { animation: float2 5s ease-in-out infinite 0.5s; }
.ai-icon:nth-child(3) { animation: float3 7s ease-in-out infinite 1s; }
.ai-icon:nth-child(4) { animation: float1 5.5s ease-in-out infinite 0.3s; }
.ai-icon:nth-child(5) { animation: float2 6.5s ease-in-out infinite 0.8s; }
.ai-icon:nth-child(6) { animation: float3 4.8s ease-in-out infinite 1.2s; }

@media (max-width: 768px) {
  .ai-icon { display: none; }
}

/* ===== 登录卡片 ===== */
.user-login {
  width: 350px;
  margin: 0 auto;
  position: relative;
  z-index: 2;
}
.user-login-logo {
  text-align: center;
  margin-bottom: 10px;
}
.user-login-logo svg {
  filter: drop-shadow(0 2px 8px rgba(0,0,0,0.2));
}

.user-login-main {
  padding: 20px 24px;
  background-color: rgba(230,230,230,0.85);
  box-shadow: 1px 1px 15px 2px rgba(0,0,0,0.3);
  border-radius: 5px;
  margin-top: 20px;
  backdrop-filter: blur(4px);
}
.user-login-main h6 {
  margin: 6px 0 18px;
  font-size: 20px;
  color: #555;
  font-weight: bold;
  text-align: center;
}

.data-form {
  display: block;
}
.data-form-item {
  display: block;
  overflow: auto;
  zoom: 1;
}

.form-input-item {
  display: block;
  width: 220px;
  border-radius: 5px;
  line-height: 32px;
  height: 36px;
  margin: 0 auto;
}
.form-input-item-white {
  background: #fff;
  margin: 8px auto;
  border: 1px solid #ddd;
}
.form-input-item-white:focus-within {
  border-color: #337ab7;
  box-shadow: 0 0 0 2px rgba(51,122,183,0.15);
}
.form-input-item input {
  line-height: 30px;
  height: 30px;
  padding: 0 10px 0 2px;
  margin-top: 3px;
  vertical-align: middle;
  border: none;
  background: transparent;
  width: calc(100% - 32px);
  font-size: 14px;
  outline: none;
}
.form-input-item-normal {
  margin: 8px auto;
  text-align: center;
}

.login-btn {
  color: #fff;
  background-color: #337ab7;
  border-color: #2e6da4;
  display: inline-block;
  font-weight: 400;
  text-align: center;
  white-space: nowrap;
  vertical-align: middle;
  border: 1px solid transparent;
  background-image: none;
  border-radius: 4px;
  padding: 6px 30px;
  font-size: 16px;
  cursor: pointer;
  line-height: 26px;
  min-width: 120px;
  transition: background-color 0.2s;
}
.login-btn:hover {
  background-color: #2e6da4;
}
.login-btn.disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.error {
  color: #f5222d;
  font-size: 15px;
  margin-bottom: 8px;
  text-align: center;
}

.bottom-copyright {
  position: fixed;
  bottom: 0;
  text-align: center;
  line-height: 20px;
  padding: 30px 0;
  width: 100%;
  color: rgba(200,200,200,0.5);
  z-index: 2;
  font-size: 12px;
}
</style>

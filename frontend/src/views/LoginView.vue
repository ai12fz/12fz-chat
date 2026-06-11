<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <h1>12FZ Chat</h1>
        <p>登录到聊天系统</p>
      </div>
      <form @submit.prevent="handleLogin">
        <div class="field">
          <input v-model="username" placeholder="用户名" autocomplete="username" />
        </div>
        <div class="field">
          <input v-model="password" type="password" placeholder="密码" autocomplete="current-password" />
        </div>
        <div v-if="error" class="error">{{ error }}</div>
        <button type="submit" :disabled="loading">
          {{ loading ? '登录中...' : '登录' }}
        </button>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const router = useRouter()
const auth = useAuthStore()

const username = ref('')
const password = ref('')
const loading = ref(false)
const error = ref('')

async function handleLogin() {
  error.value = ''
  loading.value = true
  try {
    await auth.login(username.value, password.value)
    router.push('/chat')
  } catch (e: any) {
    error.value = e.response?.data?.msg || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f0f2f5;
}
.login-card {
  width: 380px;
  padding: 40px;
  background: #fff;
  border-radius: 8px;
  box-shadow: 0 2px 12px rgba(0,0,0,.08);
}
.login-header {
  text-align: center;
  margin-bottom: 32px;
}
.login-header h1 { font-size: 24px; margin: 0 0 8px; }
.login-header p { color: #888; margin: 0; }
.field { margin-bottom: 16px; }
.field input {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 14px;
  box-sizing: border-box;
}
.field input:focus {
  border-color: #1890ff;
  outline: none;
  box-shadow: 0 0 0 2px rgba(24,144,255,.2);
}
.error { color: #f5222d; font-size: 13px; margin-bottom: 12px; }
button {
  width: 100%;
  padding: 10px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 4px;
  font-size: 15px;
  cursor: pointer;
}
button:disabled { opacity: .6; cursor: not-allowed; }
button:hover:not(:disabled) { background: #40a9ff; }
</style>

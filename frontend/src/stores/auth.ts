import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as apiLogin } from '../api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const botId = ref(localStorage.getItem('bot_id') || '')
  const expire = ref(Number(localStorage.getItem('expire') || '0'))

  const user = computed(() => ({
    username: botId.value,
    bot_id: botId.value,
  }))

  async function login(username: string, password: string, captchaId?: string, captchaAnswer?: number) {
    const res = await apiLogin(username, password, captchaId, captchaAnswer)
    // Backend response: { token, bot_id, expire }
    token.value = res.token
    botId.value = res.bot_id
    expire.value = res.expire
    localStorage.setItem('token', res.token)
    localStorage.setItem('bot_id', res.bot_id)
    localStorage.setItem('expire', String(res.expire))
  }

  function logout() {
    token.value = ''
    botId.value = ''
    expire.value = 0
    localStorage.removeItem('token')
    localStorage.removeItem('bot_id')
    localStorage.removeItem('expire')
  }

  return { token, botId, expire, user, login, logout }
})

import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as apiLogin } from '../api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const botId = ref(localStorage.getItem('bot_id') || '')
  const expire = ref(Number(localStorage.getItem('expire') || '0'))
  const merchantId = ref(localStorage.getItem('merchant_id') || '')

  
  // Read token from URL query param (for email links)
  const urlParams = new URLSearchParams(window.location.search)
  const urlToken = urlParams.get('token')
  if (urlToken && !token.value) {
    token.value = urlToken
    botId.value = urlToken.startsWith('session-') ? urlToken.slice(8) : urlToken
    localStorage.setItem('token', urlToken)
    localStorage.setItem('bot_id', botId.value)
    // Clean URL
    const cleanUrl = window.location.pathname
    window.history.replaceState({}, '', cleanUrl)
  }

  const user = computed(() => ({
    username: botId.value,
    bot_id: botId.value,
    merchant_id: merchantId.value,
  }))

  async function login(username: string, password: string) {
    const res = await apiLogin(username, password)
    // Backend response: { token, bot_id, expire }
    token.value = res.token
    botId.value = res.bot_id
    expire.value = res.expire
    localStorage.setItem('token', res.token)
    localStorage.setItem('bot_id', res.bot_id)
    localStorage.setItem('expire', String(res.expire))
    if (res.merchant_id) { merchantId.value = res.merchant_id; localStorage.setItem('merchant_id', res.merchant_id) }
  }

  function logout() {
    token.value = ''
    botId.value = ''
    expire.value = 0
    localStorage.removeItem('token')
    localStorage.removeItem('bot_id')
    merchantId.value = ''; localStorage.removeItem('merchant_id'); localStorage.removeItem('expire')
  }

  return { token, botId, expire, merchantId, user, login, logout }
})

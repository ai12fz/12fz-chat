import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as apiLogin } from '../api'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const userId = ref(localStorage.getItem('user_id') || '')
  const expire = ref(Number(localStorage.getItem('expire') || '0'))
  const merchantId = ref(localStorage.getItem('merchant_id') || '')

  
  // Read token from URL query param (for email links)
  const urlParams = new URLSearchParams(window.location.search)
  const urlToken = urlParams.get('token')
  if (urlToken && !token.value) {
    token.value = urlToken
    userId.value = urlToken.startsWith('session-') ? parseInt(urlToken.slice(8)) : parseInt(urlToken) || urlToken
    localStorage.setItem('token', urlToken)
    localStorage.setItem('user_id', userId.value)
    // Clean URL
    const cleanUrl = window.location.pathname
    window.history.replaceState({}, '', cleanUrl)
  }

  const userInfo = ref<{user_id?: number; nickname?: string; phone?: string}>({})

  const user = computed(() => ({
    user_id: userInfo.value.user_id || userId.value,
    nickname: userInfo.value.nickname || userId.value,
    username: userInfo.value.nickname || userId.value,
    bot_id: userId.value,
    merchant_id: merchantId.value,
  }))

  async function fetchWhoAmI() {
    if (!token.value) return
    try {
      const res = await fetch('/api/whoami', { headers: { Authorization: `Bearer ${token.value}` } })
      if (res.ok) {
        const data = await res.json()
        userInfo.value = data
      }
    } catch {}
  }

  async function login(username: string, password: string) {
    const res = await apiLogin(username, password)
    // Backend response: { token, bot_id, expire }
    token.value = res.token
    userId.value = res.user_id
    expire.value = res.expire
    localStorage.setItem('token', res.token)
    localStorage.setItem('user_id', res.user_id)
    localStorage.setItem('expire', String(res.expire))
    if (res.merchant_id) { merchantId.value = res.merchant_id; localStorage.setItem('merchant_id', res.merchant_id) }
  }

  function logout() {
    token.value = ''
    userId.value = ''
    expire.value = 0
    localStorage.removeItem('token')
    localStorage.removeItem('user_id')
    merchantId.value = ''; localStorage.removeItem('merchant_id'); localStorage.removeItem('expire')
  }

  
  /** Resolve botId to username via /api/whoami */
  async function resolveUsername() {
    try {
      const res = await fetch('/api/whoami', {
        headers: { Authorization: `Bearer ${token.value}` }
      })
      if (res.ok) {
        const data = await res.json()
        userId.value = data.username
        localStorage.setItem(bot_id, data.username)
        return data.username
      }
    } catch {}
    return userId.value
  }
return { token, userId, expire, merchantId, user, userInfo, login, logout, fetchWhoAmI }
})

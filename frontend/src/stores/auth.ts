import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

const GO_URL = 'https://go.12fz.com'

export const useAuthStore = defineStore('auth', () => {
  const token = ref(localStorage.getItem('token') || '')
  const userInfo = ref<any>(null)

  const user = computed(() => {
    if (userInfo.value) {
      const u = userInfo.value
      return { username: u.nickname || String(u.user_id || ''), user_id: u.user_id, nickname: u.nickname, ...u }
    }
    const t = token.value
    if (!t) return { username: '', bot_id: '' }
    return { username: t.startsWith('session-') ? t.slice(8) : '用户', bot_id: '' }
  })

  async function fetchWhoAmI() {
    if (!token.value) return
    try {
      const res = await fetch('/api/whoami', {
        headers: { Authorization: token.value }
      })
      if (res.ok) {
        const data = await res.json()
        if (data.user_id) {
          userInfo.value = data
          localStorage.setItem('whoami', JSON.stringify(data))
        }
      }
    } catch {}
  }

  function logout() {
    token.value = ''
    userInfo.value = null
    localStorage.removeItem('token')
    localStorage.removeItem('whoami')
    window.location.href = GO_URL
  }

  return { token, user, userInfo, fetchWhoAmI, logout }
})

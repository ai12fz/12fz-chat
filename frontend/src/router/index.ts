import { createRouter, createWebHistory } from 'vue-router'
import ChatView from '../views/ChatView.vue'
import { resolveChatToken } from '../utils/authToken'

const GO_URL = 'https://go.12fz.com'

const router = createRouter({
  history: createWebHistory('/chat/'),
  routes: [
    { path: '/', name: 'chat', component: ChatView },
    { path: '/login', redirect: () => { window.location.href = GO_URL; return '/' } },
    { path: '/admin/proxy', name: 'admin-proxy', component: () => import('../views/AdminProxy.vue') },
    { path: '/admin/devices', name: 'admin-devices', component: () => import('../views/AdminDevices.vue') },
  ],
})

router.beforeEach(async (to, _from, next) => {
  const urlToken = new URLSearchParams(window.location.search).get('token')
  if (urlToken) {
    // ChatPanel (go.12fz.com) 注入的是 marketplace token,必须先换成本地 JWT
    const resolved = await resolveChatToken(urlToken)
    localStorage.setItem('token', resolved)
    window.history.replaceState({}, '', window.location.pathname)
  }
  const token = localStorage.getItem('token')
  if (!token && !to.path.startsWith('/admin')) {
    window.location.href = GO_URL
    return
  }
  next()
})

export default router

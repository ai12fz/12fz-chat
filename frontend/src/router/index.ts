import { createRouter, createWebHistory } from 'vue-router'
import ChatView from '../views/ChatView.vue'

const GO_URL = 'https://go.12fz.com'

const router = createRouter({
  history: createWebHistory('/chat/'),
  routes: [
    { path: '/', name: 'chat', component: ChatView },
    { path: '/login', redirect: () => { window.location.href = GO_URL; return '/' } },
    { path: '/admin/proxy', name: 'admin-proxy', component: () => import('../views/AdminProxy.vue') },
    { path: '/admin/devices', name: 'admin-devices', component: () => import('../views/AdminDevices.vue') },
    { path: '/admin/agents', name: 'admin-agents', component: () => import('../views/AdminAgents.vue') },
  ],
})

router.beforeEach((to, _from, next) => {
  const urlToken = new URLSearchParams(window.location.search).get('token')
  if (urlToken) {
    localStorage.setItem('token', urlToken)
    window.history.replaceState({}, '', window.location.pathname)
  }
  const token = localStorage.getItem('token')
  if (!token) {
    window.location.href = GO_URL
    return
  }
  next()
})

export default router

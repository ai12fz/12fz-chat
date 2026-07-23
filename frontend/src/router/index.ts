import { createRouter, createWebHistory } from 'vue-router'
import LoginView from '../views/LoginView.vue'
import ChatView from '../views/ChatView.vue'
import AdminAgents from '../views/AdminAgents.vue'

const router = createRouter({
  history: createWebHistory(import.meta.env.BASE_URL),
  routes: [
    { path: '/', redirect: '/chat' },
    { path: '/login', name: 'login', component: LoginView },
    { path: '/chat', name: 'chat', component: ChatView, meta: { requiresAuth: true } },
    { path: '/admin/agents', name: 'admin-agents', component: AdminAgents, meta: { requiresAuth: true } },
  ],
})

router.beforeEach((to, _from, next) => {
  // Handle token from URL query param
  const urlToken = to.query.token as string
  if (urlToken) {
    localStorage.setItem('token', urlToken)
    const userId = urlToken.startsWith('session-') ? parseInt(urlToken.slice(8)) : parseInt(urlToken)
    localStorage.setItem('user_id', String(userId))
    // Clean URL
    const cleanUrl = to.path + (to.hash || '')
    window.history.replaceState({}, '', cleanUrl)
  }

  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) {
    next('/login')
  } else {
    next()
  }
})

export default router

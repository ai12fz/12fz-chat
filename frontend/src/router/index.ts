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
  // Handle token from URL query param (for email links)
  const urlToken = to.query.token as string
  if (urlToken && !localStorage.getItem('token')) {
    localStorage.setItem('token', urlToken)
    const botId = urlToken.startsWith('session-') ? urlToken.slice(8) : urlToken
    localStorage.setItem('bot_id', botId)
    // Redirect to clean URL
    next({ path: to.path, replace: true })
    return
  }

  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) {
    next('/login')
  } else {
    next()
  }
})

export default router

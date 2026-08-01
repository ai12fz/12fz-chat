import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'
import './assets/main.css'
import { resolveChatToken } from './utils/authToken'

// Listen for auth token from parent iframe (go.12fz.com ChatPanel)
window.addEventListener('message', async (e) => {
  if (e.data?.type === 'auth' && e.data?.token) {
    const resolved = await resolveChatToken(e.data.token)
    localStorage.setItem('token', resolved)
  }
})

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')

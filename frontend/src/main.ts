import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'
import './assets/main.css'

// Listen for auth token from parent iframe (go.12fz.com ChatPanel)
window.addEventListener('message', (e) => {
  if (e.data?.type === 'auth' && e.data?.token) {
    localStorage.setItem('token', e.data.token)
  }
})

const app = createApp(App)
app.use(createPinia())
app.use(router)
app.mount('#app')

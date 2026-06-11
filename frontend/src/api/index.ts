import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 10000,
})

api.interceptors.request.use(config => {
  const token = localStorage.getItem('token')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

api.interceptors.response.use(
  res => res,
  err => {
    if (err.response?.status === 401) {
      localStorage.removeItem('token')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export default api

export async function login(username: string, password: string) {
  const { data } = await api.post('/auth/login', { username, password })
  return data.data
}

export async function getGroups() {
  const { data } = await api.get('/groups')
  return data.data
}

export async function getContacts() {
  const { data } = await api.get('/contacts')
  return data.data
}

export async function getMessages(target: string, before?: string) {
  const params: any = { target }
  if (before) params.before = before
  const { data } = await api.get('/messages', { params })
  return data.data
}

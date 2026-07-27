import axios from 'axios'

const GO_URL = 'https://go.12fz.com'

const api = axios.create({
  baseURL: '/api',
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
      localStorage.removeItem('whoami')
      window.location.href = GO_URL
    }
    return Promise.reject(err)
  }
)

export default api

// ── Groups ──

export async function getMyGroups() {
  const { data } = await api.get('/groups/my')
  return data
}

// ── Messages ──

export async function getUnreadCounts() {
  const { data } = await api.get('/messages/unread')
  return data
}

// ── Friends ──

export async function getFriends(userId: string) {
  const { data } = await api.get(`/friends/${userId}`)
  return data
}

export async function addFriend(userId: string, friendId: string) {
  const { data } = await api.post('/friends', { user_id: userId, friend_id: friendId })
  return data
}

export async function getFriendMessages(userId: string, otherId: string) {
  const token = localStorage.getItem('token') || ''
  const res = await fetch('/api/friend-messages?with=' + otherId, {
    headers: { Authorization: 'Bearer ' + token }
  })
  return res.json()
}

export async function sendFriendMessage(friendId: string, content: string) {
  const token = localStorage.getItem('token') || ''
  const res = await fetch('/api/friend-messages', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: 'Bearer ' + token },
    body: JSON.stringify({ friend_id: friendId, content })
  })
  return res.json()
}

// ── Messages (used by ChatContent) ──

export async function getMessages(groupId: number, limit = 50, offset = 0) {
  const { data } = await api.get('/messages', { params: { group_id: groupId, limit, offset } })
  return data
}

export async function sendMessage(groupId: number, content: string) {
  const { data } = await api.post('/messages', { group_id: groupId, content })
  return data
}

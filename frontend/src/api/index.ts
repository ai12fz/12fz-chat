import axios from 'axios'

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
      localStorage.removeItem('user')
      window.location.href = '/login'
    }
    return Promise.reject(err)
  }
)

export default api

// ── Auth ──

export async function login(username: string, password: string, captchaId?: string, captchaAnswer?: number) {
  const body: any = { username, password }
  if (captchaId) body.captcha_id = captchaId
  if (captchaAnswer !== undefined) body.captcha_answer = captchaAnswer
  const { data } = await api.post('/login', body)
  return data // { token, bot_id, expire }
}

// ── Groups ──

export async function getMyGroups() {
  const { data } = await api.get('/groups/my')
  // data = [{ id, name, created_by, created_at, last_msg_at }]
  return data
}

export async function getGroupMembers(groupId: number) {
  const { data } = await api.get(`/groups/${groupId}/members`)
  // data = [{ group_id, bot_id, role, joined_at }]
  return data
}

export async function createGroup(name: string) {
  const { data } = await api.post('/groups', { name })
  return data
}

// ── Messages ──

export async function getMessages(groupId: number, limit = 50, offset = 0) {
  const { data } = await api.get('/messages', {
    params: { group_id: groupId, limit, offset },
  })
  // data = [{ id, group_id, sender_id, content, msg_type, created_at }]
  return data
}

export async function sendMessage(groupId: number, content: string) {
  const { data } = await api.post('/messages', { group_id: groupId, content })
  return data
}

export async function getUnreadCounts() {
  const { data } = await api.get('/messages/unread')
  // data = { <group_id>: <count>, ... }
  return data
}

export async function markRead(groupId: number, lastReadMsgId: number) {
  const { data } = await api.post('/messages/read', {
    group_id: groupId,
    last_read_msg_id: lastReadMsgId,
  })
  return data
}

// ── Friends ──

export async function getFriends(userId: string) {
  const { data } = await api.get(`/friends/${userId}`)
  // data = [{ user_id, friend_id, status, created_at }]
  return data
}

export async function addFriend(userId: string, friendId: string) {
  const { data } = await api.post('/friends', { user_id: userId, friend_id: friendId })
  return data
}

// ── DM (Direct Message) Group ──

export async function createDMGroup(friendId: string) {
  const { data } = await api.post('/groups/dm', { friend_id: friendId })
  // data = { id, name, created_by, created_at }
  return data
}

// ── Image Upload ──

export async function uploadImage(file: File, groupId: number) {
  const formData = new FormData()
  formData.append('image', file)
  formData.append('group_id', String(groupId))
  const token = localStorage.getItem('token')
  const res = await fetch('/api/upload', {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: formData,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'upload failed' }))
    throw new Error(err.error || `HTTP ${res.status}`)
  }
  return res.json()
}

export async function uploadAvatar(file: File) {
  const formData = new FormData()
  formData.append('avatar', file)
  const token = localStorage.getItem('token')
  const res = await fetch('/api/avatar', {
    method: 'POST',
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: formData,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: 'upload failed' }))
    throw new Error(err.error || `HTTP ${res.status}`)
  }
  return res.json()
}

<template>
  <div class="chat-layout">
    <!-- 顶部导航栏 -->
    <div id="chat-nav">
      <ul class="chat-nav-l">
        <li class="nav-avatar" @click="handleAvatarClick" :title="'点击上传头像'">
          <img v-if="avatarUrl" :src="avatarUrl" class="nav-avatar-img" />
          <span v-else class="nav-avatar-circle">{{ displayName[0] }}</span>
        </li>
        <li>
          <span class="nav-username">{{ displayName }}</span>
        </li>
        <li>
          <select class="nav-state-select" v-model="onlineState">
            <option value="1">在线</option>
            <option value="2">忙碌</option>
            <option value="3">隐身</option>
          </select>
        </li>
        <li><i class="fa fa-bell nav-bell" :class="bellOn ? 'bell-active' : 'bell-muted'" @click="bellOn = !bellOn"></i></li>
      </ul>
      <ul class="chat-nav-r">
        <li v-for="item in navIcons" :key="item.label" class="nav-icon-item" :title="item.label" @click.stop="item.onClick">
          <div class="nav-icon-wrap">
            <i class="fa" :class="item.icon"></i>
            <span>{{ item.label }}</span>
          </div>
        </li>
      </ul>
      <!-- 设置下拉菜单 -->
      <div v-if="showSettings" class="settings-dropdown" @click.stop>
        <div class="settings-dropdown-item" @click="handleLogout">
          <i class="fa fa-sign-out"></i> 退出登录
        </div>
      </div>
    </div>

    <!-- 隐藏的文件上传 input -->
    <input ref="avatarFileInput" type="file" accept="image/*" style="display:none" @change="handleAvatarUpload" />

    <!-- 主体：侧栏 + 内容 -->
    <div class="chat-body">
      <SidebarLeft />
      <ChatContent />
      <SidebarRight />
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { useWebSocket } from '../composables/useWebSocket'
import { getMyGroups, getUnreadCounts, getMessages, markRead, uploadAvatar } from '../api'
import SidebarLeft from '../components/SidebarLeft.vue'
import ChatContent from '../components/ChatContent.vue'
import SidebarRight from '../components/SidebarRight.vue'

const router = useRouter()
const auth = useAuthStore()
const chat = useChatStore()
const ws = useWebSocket()

const displayName = computed(() => auth.user?.username || auth.user?.bot_id || '用户')
const bellOn = ref(true)
const onlineState = ref('1')
const showSettings = ref(false)
const avatarFileInput = ref<HTMLInputElement>()
const avatarUrl = ref(localStorage.getItem('avatar_url') || '')
const navIcons = [
  { icon: 'fa-cog', label: '设置', onClick: () => { showSettings.value = !showSettings.value } },
  { icon: 'fa-tasks', label: '工单', onClick: () => {} },
  { icon: 'fa-bar-chart', label: '中台', onClick: () => window.open('https://go.12fz.com/', '_blank') },
]

function handleLogout() {
  showSettings.value = false
  auth.logout()
  router.push('/login')
}

function handleAvatarClick() {
  avatarFileInput.value?.click()
}

async function handleAvatarUpload(event: Event) {
  const input = event.target as HTMLInputElement
  if (!input.files || input.files.length === 0) return
  const file = input.files[0]
  try {
    const res = await uploadAvatar(file)
    if (res.avatar_url) {
      avatarUrl.value = res.avatar_url
      localStorage.setItem('avatar_url', res.avatar_url)
    }
  } catch (err) {
    console.error('avatar upload failed:', err)
    alert('上传头像失败')
  }
  input.value = ''
}

onMounted(async () => {
  if (!auth.token) {
    router.push('/login')
    return
  }
  ws.connect(auth.token)
  try {
    const groups = await getMyGroups()
    groups.forEach((g: any) => chat.ensureGroupSession(g))
    try {
      const unreads = await getUnreadCounts()
      for (const [groupIdStr, count] of Object.entries(unreads)) {
        const sid = chat.groupSessionId(Number(groupIdStr))
        const s = chat.sessions.find(s => s.id === sid)
        if (s) s.unread = count as number
      }
    } catch { /* ok */ }
    if (chat.sessions.length > 0) chat.setActive(chat.sessions[0].id)
  } catch (err) {
    console.error('Failed to load chat data:', err)
  }
})

// 点击其他地方关闭下拉
document.addEventListener('click', () => {
  showSettings.value = false
})

// 切换会话时加载历史消息 + 标记已读
watch(() => chat.activeId, async (newId) => {
  if (!newId) return
  const match = newId.match(/^group:(\d+)$/)
  if (!match) return
  const groupId = parseInt(match[1])
  try {
    const msgs = await getMessages(groupId, 20, 0)
    chat.loadMessages(newId, msgs)
    if (msgs.length > 0) {
      const lastId = msgs[msgs.length - 1].id
      markRead(groupId, lastId).catch(() => {/* non-critical */})
    }
  } catch (err) {
    console.error('Failed to load messages:', err)
  }
})
</script>

<style scoped>
.chat-layout { display: flex; flex-direction: column; height: 100vh; background: #05aeee; }
#chat-nav {
  display: flex; align-items: center; justify-content: space-between;
  height: 50px; padding: 0 16px;
  background: rgba(39,53,131,.6); color: #fff; flex-shrink: 0;
  position: relative; z-index: 50;
}
.chat-nav-l { display: flex; align-items: center; list-style: none; margin: 0; padding: 0; gap: 8px; }
.chat-nav-l li { display: flex; align-items: center; }
.nav-avatar { width: 34px; height: 34px; border-radius: 50%; overflow: hidden; cursor: pointer; display: flex; align-items: center; justify-content: center; background: rgba(255,255,255,.2); }
.nav-avatar-img { width: 100%; height: 100%; object-fit: cover; }
.nav-avatar-circle { font-size: 16px; font-weight: 700; color: #fff; }
.nav-username { font-size: 15px; font-weight: 500; cursor: default; }
.nav-state-select {
  background: transparent; color: #fff; border: 1px solid rgba(255,255,255,.3);
  border-radius: 3px; padding: 2px 4px; font-size: 12px; cursor: pointer;
  outline: none;
}
.nav-state-select option { color: #333; background: #fff; }
.nav-bell { font-size: 18px; cursor: pointer; opacity: .8; transition: opacity .2s; }
.nav-bell:hover { opacity: 1; }
.bell-active { opacity: 1; }
.bell-muted { opacity: .4; color: #ff6b6b; }
.chat-nav-r { display: flex; align-items: center; list-style: none; margin: 0; padding: 0; gap: 2px; }
.nav-icon-item { cursor: pointer; padding: 4px 8px; border-radius: 4px; transition: background .15s; }
.nav-icon-item:hover { background: rgba(255,255,255,.15); }
.nav-icon-wrap { display: flex; align-items: center; gap: 4px; color: rgba(255,255,255,.85); font-size: 13px; }
.nav-icon-wrap i { font-size: 16px; }
.settings-dropdown {
  position: absolute; right: 16px; top: 52px;
  background: #fff; border: 1px solid #e8e8e8; border-radius: 6px;
  box-shadow: 0 4px 12px rgba(0,0,0,.12); min-width: 120px; z-index: 100;
  overflow: hidden;
}
.settings-dropdown-item {
  padding: 10px 16px; font-size: 14px; color: #333; cursor: pointer;
  transition: background .15s; white-space: nowrap; display: flex; align-items: center; gap: 6px;
}
.settings-dropdown-item:hover { background: #f5f5f5; color: #f5222d; }
.chat-body { display: flex; flex: 1; overflow: hidden; }
</style>

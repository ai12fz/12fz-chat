<template>
  <aside class="sidebar-left">
    <div class="sidebar-header">
      <div class="user-info" @click="auth.logout(); router.push('/login')">
        <span class="avatar">{{ (auth.user?.username || '?')[0] }}</span>
        <span class="name">{{ auth.user?.username }}</span>
      </div>
    </div>
    <div class="search-box">
      <input v-model="search" placeholder="搜索联系人..." />
    </div>
    <nav class="session-list">
      <div
        v-for="s in filteredSessions"
        :key="s.id"
        class="session-item"
        :class="{ active: s.id === chat.activeId }"
        @click="chat.setActive(s.id)"
      >
        <span class="avatar sm">{{ s.name[0] }}</span>
        <div class="session-info">
          <div class="session-top">
            <span class="session-name">{{ s.name }}</span>
            <span class="session-badge" v-if="s.type === 'group'">群</span>
          </div>
          <span class="session-msg">{{ s.lastMsg || '暂无消息' }}</span>
        </div>
        <span v-if="s.unread" class="unread-badge">{{ s.unread }}</span>
      </div>
    </nav>
  </aside>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'

const router = useRouter()
const auth = useAuthStore()
const chat = useChatStore()
const search = ref('')

const filteredSessions = computed(() => {
  if (!search.value) return chat.sessions
  const q = search.value.toLowerCase()
  return chat.sessions.filter(s => s.name.toLowerCase().includes(q))
})
</script>

<style scoped>
.sidebar-left {
  width: 280px;
  border-right: 1px solid #e8e8e8;
  display: flex;
  flex-direction: column;
  background: #fafafa;
}
.sidebar-header {
  padding: 12px 16px;
  border-bottom: 1px solid #e8e8e8;
}
.user-info {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}
.avatar {
  width: 36px; height: 36px;
  border-radius: 50%;
  background: #1890ff;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  flex-shrink: 0;
}
.avatar.sm { width: 32px; height: 32px; font-size: 12px; }
.name { font-weight: 500; }
.search-box { padding: 8px 12px; }
.search-box input {
  width: 100%;
  padding: 6px 10px;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  font-size: 13px;
  box-sizing: border-box;
}
.session-list { flex: 1; overflow-y: auto; }
.session-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 16px;
  cursor: pointer;
  position: relative;
}
.session-item:hover { background: #f0f0f0; }
.session-item.active { background: #e6f7ff; }
.session-info { flex: 1; min-width: 0; }
.session-top {
  display: flex;
  align-items: center;
  gap: 4px;
}
.session-name { font-size: 14px; font-weight: 500; }
.session-badge {
  font-size: 10px;
  background: #e8e8e8;
  padding: 0 4px;
  border-radius: 2px;
}
.session-msg {
  font-size: 12px;
  color: #888;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
}
.unread-badge {
  background: #f5222d;
  color: #fff;
  font-size: 11px;
  min-width: 18px;
  height: 18px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 4px;
}
</style>

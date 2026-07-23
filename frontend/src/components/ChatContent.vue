<template>
  <main class="chat-content">
    <template v-if="session">
      <!-- Header -->
      <div class="chat-header">
        <span class="chat-title">{{ session.name }}</span>
        <span class="chat-type">{{ session.type === 'group' ? '群聊' : '好友' }}</span>
        <span class="header-meta" v-if="session.members">
          {{ session.members.length }} 人
        </span>
      </div>

      <!-- Messages -->
      <div class="message-list" ref="msgListRef">
        <div v-for="msg in session.messages" :key="msg.id" class="message-row">
          <div v-if="msg.sender_id === auth.user?.username || msg.sender_id === auth.user?.bot_id" class="message self">
            <div class="msg-body">
              <div class="msg-content self-msg">{{ msg.content }}</div>
              <div class="msg-time">{{ formatTime(msg.created_at) }}</div>
            </div>
          </div>
          <div v-else class="message other">
            <span class="msg-avatar" :style="{ background: nameColor(msg.sender_id) }">{{ msg.sender_id[0] }}</span>
            <div class="msg-body">
              <div class="msg-sender">{{ msg.sender_id }}</div>
              <div class="msg-content other-msg">{{ msg.content }}</div>
              <div class="msg-time">{{ formatTime(msg.created_at) }}</div>
            </div>
          </div>
        </div>
        <div v-if="session.messages.length === 0" class="empty-msg">
          暂无消息，开始聊天吧
        </div>
      </div>

      <!-- Chat Input -->
      <div class="chat-input-area">
        <div class="toolbar">
          <button class="tool-btn" @click="showEmoji = !showEmoji" title="表情">😊</button>
          <button class="tool-btn" @click="selectFile" title="文件">📎</button>
        </div>
        <div v-if="showEmoji" class="emoji-picker">
          <span v-for="emoji in emojis" :key="emoji" class="emoji-item" @click="insertEmoji(emoji)">{{ emoji }}</span>
        </div>
        <textarea
          ref="inputRef"
          v-model="text"
          placeholder="输入消息..."
          @keydown="handleKeydown"
          rows="2"
        ></textarea>
        <div class="input-actions">
          <span class="connection-status" :class="{ online: chat.connected }">
            {{ chat.connected ? '已连接' : '重连中...' }}
          </span>
          <div class="action-right">
            <input
              ref="fileInputRef"
              type="file"
              style="display:none"
              @change="handleFile"
            />
            <button @click="handleSend" :disabled="!text.trim()">发送</button>
          </div>
        </div>
      </div>
    </template>

    <!-- Empty state -->
    <div v-else class="empty-state">
      <div class="empty-icon">💬</div>
      <p>选择一个会话开始聊天</p>
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref, watch, nextTick, computed } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { useWebSocket } from '../composables/useWebSocket'

const auth = useAuthStore()
const chat = useChatStore()
const ws = useWebSocket()

const text = ref('')
const showEmoji = ref(false)
const msgListRef = ref<HTMLElement>()
const inputRef = ref<HTMLTextAreaElement>()
const fileInputRef = ref<HTMLInputElement>()

const session = computed(() => chat.activeSession)

const emojis = ['😊', '😂', '🤣', '❤️', '👍', '😍', '🥰', '😘', '😜', '😎',
  '🤔', '🙄', '😏', '😴', '🥱', '😭', '😤', '😡', '🤬', '👋',
  '✌️', '🤞', '🤝', '🙏', '💪', '🔥', '✨', '🌟', '⭐', '🎉',
  '🎊', '🎈', '🎁', '💡', '📌', '✅', '❌', '⚠️', '🚀', '💰']

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  }
}

function handleSend() {
  const content = text.value.trim()
  if (!content || !session.value) return
  // Parse group id from session id "group:123"
  const match = session.value.id.match(/^group:(\d+)$/)
  if (match) {
    ws.sendMessage(parseInt(match[1]), content)
  }
  text.value = ''
  showEmoji.value = false
}

function insertEmoji(emoji: string) {
  text.value += emoji
  inputRef.value?.focus()
}

function selectFile() {
  fileInputRef.value?.click()
}

function handleFile(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || !session.value) return
  // For now, just notify - file upload via REST later
  const match = session.value.id.match(/^group:(\d+)$/)
  if (match) {
    ws.sendMessage(parseInt(match[1]), `[文件] ${file.name}`)
  }
  input.value = '' // reset
}

function formatTime(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  const now = new Date()
  const isToday = d.toDateString() === now.toDateString()
  const time = d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  if (isToday) return time
  return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' }) + ' ' + time
}

function nameColor(name: string) {
  const colors = ['#1890ff', '#52c41a', '#fa8c16', '#eb2f96', '#722ed1', '#13c2c2', '#f5222d', '#faad14']
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = name.charCodeAt(i) + ((hash << 5) - hash)
  return colors[Math.abs(hash) % colors.length]
}

// Auto scroll to bottom on new messages
watch(
  () => session.value?.messages.length,
  async () => {
    await nextTick()
    if (msgListRef.value) {
      msgListRef.value.scrollTop = msgListRef.value.scrollHeight
    }
  }
)

// Scroll when switching sessions
watch(() => chat.activeId, async () => {
  await nextTick()
  if (msgListRef.value) {
    msgListRef.value.scrollTop = msgListRef.value.scrollHeight
  }
})
</script>

<style scoped>
.chat-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  background: #fff;
}

/* Header */
.chat-header {
  padding: 12px 20px;
  border-bottom: 1px solid #e8e8e8;
  display: flex;
  align-items: center;
  gap: 8px;
  background: #fafafa;
}
.chat-title { font-size: 16px; font-weight: 600; }
.chat-type { font-size: 11px; background: #e8e8e8; padding: 2px 6px; border-radius: 3px; }
.header-meta { font-size: 12px; color: #888; margin-left: auto; }

/* Message list */
.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
  background: #f5f5f5;
}
.message-row { margin-bottom: 8px; }
.message {
  display: flex;
  gap: 8px;
  max-width: 80%;
}
.message.self {
  flex-direction: row-reverse;
  margin-left: auto;
}
.message.other {
  margin-right: auto;
}

.msg-avatar {
  width: 32px; height: 32px;
  border-radius: 50%;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  flex-shrink: 0;
  margin-top: 4px;
}

.msg-body { max-width: 70%; }
.msg-sender { font-size: 11px; color: #888; margin-bottom: 2px; }
.msg-content {
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;
  white-space: pre-wrap;
}
.self-msg {
  background: #1890ff;
  color: #fff;
  border-bottom-right-radius: 2px;
}
.other-msg {
  background: #fff;
  color: #333;
  border-bottom-left-radius: 2px;
  box-shadow: 0 1px 2px rgba(0,0,0,.06);
}
.msg-time {
  font-size: 10px;
  color: #aaa;
  margin-top: 2px;
  text-align: right;
}
.message.other .msg-time { text-align: left; }

.empty-msg {
  text-align: center;
  color: #bbb;
  padding: 48px 16px;
  font-size: 14px;
}

/* Chat input */
.chat-input-area {
  border-top: 1px solid #e8e8e8;
  padding: 0 20px 12px;
  background: #fff;
}
.toolbar {
  display: flex;
  gap: 4px;
  padding: 6px 0;
}
.tool-btn {
  background: none;
  border: none;
  cursor: pointer;
  font-size: 18px;
  padding: 4px 6px;
  border-radius: 4px;
  transition: background .15s;
}
.tool-btn:hover { background: #f0f0f0; }

.emoji-picker {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 8px;
  background: #fafafa;
  border-radius: 6px;
  margin-bottom: 6px;
  max-height: 120px;
  overflow-y: auto;
}
.emoji-item {
  cursor: pointer;
  font-size: 20px;
  padding: 2px;
  border-radius: 3px;
  transition: background .15s;
}
.emoji-item:hover { background: #e8e8e8; }

textarea {
  width: 100%;
  border: 1px solid #d9d9d9;
  border-radius: 6px;
  padding: 8px 10px;
  font-size: 14px;
  resize: none;
  box-sizing: border-box;
  outline: none;
  font-family: inherit;
  transition: border-color .15s;
}
textarea:focus { border-color: #1890ff; }

.input-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 6px;
}
.connection-status { font-size: 12px; color: #888; }
.connection-status.online { color: #52c41a; }
.action-right { display: flex; gap: 8px; align-items: center; }
button {
  padding: 6px 24px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 13px;
  transition: background .15s;
}
button:hover:not(:disabled) { background: #40a9ff; }
button:disabled { opacity: .5; cursor: not-allowed; }

/* Empty state */
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: #888;
}
.empty-icon { font-size: 48px; margin-bottom: 12px; }
</style>

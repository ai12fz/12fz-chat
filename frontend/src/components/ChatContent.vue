<template>
  <main class="chat-content">
    <template v-if="session">
      <div class="chat-header">
        <span class="chat-title">{{ session.name }}</span>
        <span class="chat-type">{{ session.type === 'group' ? '群聊' : session.type === 'bot' ? 'AI' : '好友' }}</span>
      </div>

      <div class="message-list" ref="msgList">
        <div v-for="msg in session.messages" :key="msg.id" class="message" :class="{ self: msg.from === auth.user?.id }">
          <span class="msg-avatar">{{ msg.fromName[0] }}</span>
          <div class="msg-body">
            <div class="msg-sender">{{ msg.fromName }}</div>
            <div class="msg-content">{{ msg.content }}</div>
          </div>
        </div>
      </div>

      <div class="chat-input">
        <textarea
          v-model="text"
          placeholder="输入消息..."
          @keydown.enter.prevent="sendMessage"
          rows="3"
        ></textarea>
        <div class="input-actions">
          <span class="connection-status" :class="{ online: chat.connected }">
            {{ chat.connected ? '已连接' : '重连中...' }}
          </span>
          <button @click="sendMessage" :disabled="!text.trim()">发送</button>
        </div>
      </div>
    </template>
    <div v-else class="empty-state">
      <div class="empty-icon">💬</div>
      <p>选择一个会话开始聊天</p>
    </div>
  </main>
</template>

<script setup lang="ts">
import { ref, watch, nextTick } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { useWebSocket } from '../composables/useWebSocket'

const auth = useAuthStore()
const chat = useChatStore()
const ws = useWebSocket()

const text = ref('')
const msgList = ref<HTMLElement>()
const session = chat.activeSession

function sendMessage() {
  const content = text.value.trim()
  if (!content || !session) return
  ws.send(session.id, content)
  text.value = ''
}

watch(() => chat.activeId, async () => {
  await nextTick()
  if (msgList.value) msgList.value.scrollTop = msgList.value.scrollHeight
})
</script>

<style scoped>
.chat-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.chat-header {
  padding: 12px 20px;
  border-bottom: 1px solid #e8e8e8;
  display: flex;
  align-items: center;
  gap: 8px;
}
.chat-title { font-size: 16px; font-weight: 600; }
.chat-type { font-size: 11px; background: #e8e8e8; padding: 2px 6px; border-radius: 3px; }

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
}
.message {
  display: flex;
  gap: 8px;
  margin-bottom: 16px;
}
.message.self { flex-direction: row-reverse; }
.msg-avatar {
  width: 32px; height: 32px;
  border-radius: 50%;
  background: #1890ff;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  flex-shrink: 0;
}
.msg-body { max-width: 60%; }
.msg-sender { font-size: 12px; color: #888; margin-bottom: 4px; }
.message.self .msg-sender { text-align: right; }
.msg-content {
  background: #f0f0f0;
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;
}
.message.self .msg-content {
  background: #1890ff;
  color: #fff;
}

.chat-input {
  border-top: 1px solid #e8e8e8;
  padding: 12px 20px;
}
.chat-input textarea {
  width: 100%;
  border: 1px solid #d9d9d9;
  border-radius: 4px;
  padding: 8px;
  font-size: 14px;
  resize: none;
  box-sizing: border-box;
}
.input-actions {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
}
.connection-status { font-size: 12px; color: #888; }
.connection-status.online { color: #52c41a; }
button {
  padding: 6px 20px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}
button:disabled { opacity: .5; cursor: not-allowed; }

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

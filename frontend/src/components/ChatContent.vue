<template>
  <div id="chat-content">
    <div class="content-inner">
      <template v-if="session">
        <!-- 标题栏: 好友显示昵称, 群聊显示群名+邀请 -->
        <div v-if="session.type === 'group'" class="sz_othersTitle flex-center">
          <h3>{{ session.name }}</h3>
          <i class="fa fa-user-plus invite-icon" @click="inviteFriends" title="邀请好友"></i>
        </div>
        <div v-else class="sz_othersTitle flex-center">
          <h3>{{ session.name }}</h3>
        </div>

        <!-- 消息列表 -->
    <div ref="msgListRef" class="sz_chatcontent" @scroll="onScroll">
      <!-- 未读消息分隔线 -->
      <div v-if="unreadDividerIndex >= 0" class="unread-divider">
        <span>—— 未读消息 ——</span>
      </div>
      <div v-for="(msg, idx) in session.messages" :key="msg.id" class="message-row">
        <div v-if="unreadDividerIndex >= 0 && idx === unreadDividerIndex" class="unread-mark"></div>
        <!-- 自己发的 -->
        <div v-if="msg.sender_id === myName" class="message self">
          <div class="msg-body">
            <div class="msg-sender self-sender">{{ msg.sender_id }}</div>
            <div class="msg-content self-msg">
              <img v-if="msg.msg_type === 'image'" :src="parseMsg(msg.content).main" class="msg-image" alt="图片" @click="previewImage(parseMsg(msg.content).main)" />
              <div v-else class="msg-text">{{ parseMsg(msg.content).main }}</div>
              <details v-if="parseMsg(msg.content).exec" class="exec-log">
                <summary>📋 执行过程 ▾</summary>
                <pre class="exec-steps">{{ parseMsg(msg.content).exec }}</pre>
              </details>
            </div>
            <div class="msg-time">{{ formatTime(msg.created_at) }}</div>
          </div>
        </div>
        <!-- 别人发的 -->
        <div v-else class="message other">
          <span class="msg-avatar">{{ msg.sender_id[0] }}</span>
          <div class="msg-body">
            <div class="msg-sender">
              {{ msg.sender_id }}
              <span v-if="isBot(msg.sender_id)" class="bot-tag">🤖</span>
            </div>
            <div class="msg-content other-msg">
              <img v-if="msg.msg_type === 'image'" :src="parseMsg(msg.content).main" class="msg-image" alt="图片" @click="previewImage(parseMsg(msg.content).main)" />
              <div v-else class="msg-text">{{ parseMsg(msg.content).main }}</div>
              <details v-if="parseMsg(msg.content).exec" class="exec-log">
                <summary>📋 执行过程 ▾</summary>
                <pre class="exec-steps">{{ parseMsg(msg.content).exec }}</pre>
              </details>
            </div>
            <div class="msg-time">{{ formatTime(msg.created_at) }}</div>
          </div>
        </div>
      </div>
      <div v-if="session.messages.length === 0" class="empty-msg">暂无消息，开始聊天吧</div>
    </div>

        <!-- 工具栏 (sysfun) -->
        <div class="sysfun flex-center">
          <ul class="sysfun-left">
            <li v-for="btn in sysfunBtns" :key="btn.id" :title="btn.title" @click="handleToolbar(btn)">
              <i class="fa" :class="[btn.icon, btn.extraClass || '']"></i>
            </li>
          </ul>
          <ul class="sysfun-right">
            <li v-for="btn in sysfunBtns2" :key="btn.id" :title="btn.title" @click="handleToolbar(btn)">
              <i class="fa" :class="btn.icon"></i>
            </li>
          </ul>
        </div>

        <!-- @提及面板 -->
        <div v-if="showMention" class="mention-panel">
          <div class="mention-search">
            <i class="fa fa-search"></i>
            <input
              ref="mentionSearchInput"
              v-model="mentionFilter"
              placeholder="搜索成员..."
              @keydown="handleMentionKeydown"
              @keydown.escape="closeMention"
            />
          </div>
          <div class="mention-list">
            <div
              v-for="(item, i) in filteredMentionList"
              :key="item.id"
              class="mention-item"
              :class="{ active: mentionIndex === i }"
              @click="selectMention(item)"
              @mouseenter="mentionIndex = i"
            >
              <span class="mention-avatar">{{ item.name[0] }}</span>
              <div class="mention-info">
                <span class="mention-name">{{ item.name }}</span>
                <span v-if="isBot(item.name)" class="bot-tag">🤖</span>
                <span class="mention-note" v-if="item.note">{{ item.note }}</span>
              </div>
            </div>
            <div v-if="filteredMentionList.length === 0" class="mention-empty">无匹配成员</div>
          </div>
        </div>

        <!-- contenteditable 输入框 -->
        <div
          class="chat_input"
          contenteditable="true"
          ref="inputRef"
          @keydown="handleKeydown"
          @paste="handlePaste"
          @input="handleInput"
          data-placeholder="输入消息..."
        ></div>

        <!-- 发送栏 -->
        <div class="chat_enter">
          <div class="enter-mode flex-center">
            <span class="flex-center" @click="sendMode = 1">
              <i class="fa" :class="sendMode === 1 ? 'fa-circle' : 'fa-circle-thin'"></i>
              <span>按Enter发送</span>
            </span>
            <span class="flex-center" @click="sendMode = 2">
              <i class="fa" :class="sendMode === 2 ? 'fa-circle' : 'fa-circle-thin'"></i>
              <span>按Ctrl+Enter发送</span>
            </span>
          </div>
          <div class="send-area flex-center">
            <button class="send-btn" @click="handleSend">发送</button>
            <i class="fa fa-send shortcut-btn" @click="openShortcut" title="快捷语"></i>
          </div>
        </div>

        <!-- 工具函数对应的隐藏元素 -->
        <div style="display:none">
          <input ref="fileInput" type="file" @change="handleFileSelect" />
          <input ref="imgInput" type="file" accept="image/*" @change="handleImgSelect" />
        </div>
      </template>

      <!-- 未选择会话 -->
      <div v-else class="empty-state">
        <div class="empty-icon">💬</div>
        <p>选择一个会话开始聊天</p>
      </div>

    </div>
    <!-- 快捷语面板（放在 content-inner 外面，避免被 overflow:hidden 裁剪） -->
    <div v-if="showShortcut" class="shortcut-panel" @click.stop>
      <!-- 分类Tab -->
      <div class="sc-cats">
        <span
          v-for="cat in scData"
          :key="cat.id"
          class="sc-cat"
          :class="{ active: scActiveCat === cat.id }"
          @click="scActiveCat = cat.id"
          @dblclick="scStartEditCat"
        >{{ cat.name }}</span>
        <span class="sc-cat-add" @click="scAddCat" title="添加分类">＋</span>
      </div>
      <!-- 编辑模式：重命名分类 -->
      <div v-if="scEditingCat" class="sc-cat-edit">
        <input v-model="scEditingCatName" @keyup.enter="scSaveCatRename" @keyup.escape="scCancelCatEdit" ref="scCatInput" />
        <button @click="scSaveCatRename">确定</button>
        <button @click="scCancelCatEdit">取消</button>
        <button v-if="scData.length > 1" class="sc-del-btn" @click="scDeleteCat">删除此分类</button>
      </div>
      <!-- 快捷语列表 -->
      <div class="sc-items">
        <div
          v-for="item in currentItems"
          :key="item.id"
          class="sc-item"
        >
          <span class="sc-item-text" @click="selectShortcut(item.text)">{{ item.text }}</span>
          <span class="sc-item-ops">
            <i class="fa fa-pencil" @click="scStartEdit(item)" title="编辑"></i>
            <i class="fa fa-trash" @click="scDeleteItem(item)" title="删除"></i>
          </span>
        </div>
      </div>
      <!-- 添加新快捷语 -->
      <div class="sc-add-bar">
        <input v-model="scNewText" placeholder="输入新快捷语" @keyup.enter="scAddItem" />
        <button @click="scAddItem">添加</button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useChatStore } from '../stores/chat'
import { useWebSocket } from '../composables/useWebSocket'
import { getGroupMembers, getFriends, getMessages, uploadImage } from '../api'

const auth = useAuthStore()
const chat = useChatStore()
const ws = useWebSocket()

// Bots list — show 🤖 icon next to these
const botNames = ['chaogu-ai', '服务器技术', '高级工程师']
function isBot(name: string) { return botNames.includes(name) }

// 解析bot消息：分离正文和"━━━ 执行过程 ━━━"后面的内容
const execSep = '━━━ 执行过程 ━━━'
function parseMsg(content: string) {
  const idx = content.indexOf(execSep)
  if (idx === -1) return { main: content, exec: '' }
  return {
    main: content.substring(0, idx).trim(),
    exec: content.substring(idx + execSep.length).trim(),
  }
}

const myName = computed(() => auth.user?.username || auth.user?.bot_id || '')
const session = computed(() => chat.activeSession)
const sendMode = ref(1) // 1=Enter 2=Ctrl+Enter
const inputRef = ref<HTMLDivElement>()
const msgListRef = ref<HTMLElement | null>(null)
const fileInput = ref<HTMLInputElement>()
const imgInput = ref<HTMLInputElement>()

// ── @提及 ──
const showMention = ref(false)
const mentionFilter = ref('')
const mentionIndex = ref(0)
const mentionSearchInput = ref<HTMLInputElement>()
const cachedMembers = ref<{ id: string; name: string; note?: string }[]>([])
const cachedFriends = ref<{ id: string; name: string; note?: string }[]>([])

interface MentionItem {
  id: string
  name: string
  note?: string
  lastMsgTime?: string
}

// Build mention list: for group → members, for user → friends
// Sort by most recent message sender in current session first
const mentionList = computed<MentionItem[]>(() => {
  if (!session.value) return []

  if (session.value.type === 'group') {
    // Build a map of last message time per sender_id
    const lastMsgMap = new Map<string, string>()
    if (session.value.messages) {
      for (const msg of session.value.messages) {
        if (msg.sender_id !== myName.value) {
          // Use the latest message time for each sender
          if (!lastMsgMap.has(msg.sender_id) || msg.created_at > lastMsgMap.get(msg.sender_id)!) {
            lastMsgMap.set(msg.sender_id, msg.created_at)
          }
        }
      }
    }

    const members = cachedMembers.value
    const sorted = [...members].map(m => ({
      id: m.id,
      name: m.name,
      note: m.note,
      lastMsgTime: lastMsgMap.get(m.id),
    }))
    // Sort: recently active first, then alphabetically
    sorted.sort((a, b) => {
      if (a.lastMsgTime && b.lastMsgTime) return b.lastMsgTime.localeCompare(a.lastMsgTime)
      if (a.lastMsgTime) return -1
      if (b.lastMsgTime) return 1
      return a.name.localeCompare(b.name)
    })
    return sorted
  } else {
    // User chat: just show friends
    return cachedFriends.value.map(f => ({ id: f.id, name: f.name, note: f.note }))
  }
})

const filteredMentionList = computed(() => {
  const q = mentionFilter.value.toLowerCase().trim()
  if (!q) return mentionList.value
  return mentionList.value.filter(m =>
    m.name.toLowerCase().includes(q) || m.note?.toLowerCase().includes(q)
  )
})

async function openMention() {
  mentionFilter.value = ''
  mentionIndex.value = 0

  if (!session.value) return

  if (session.value.type === 'group') {
    // Fetch group members if not cached
    const match = session.value.id.match(/^group:(\d+)$/)
    if (match) {
      const groupId = parseInt(match[1])
      if (cachedMembers.value.length === 0) {
        try {
          const members = await getGroupMembers(groupId)
          // members = [{ group_id, bot_id, role, joined_at }]
          cachedMembers.value = members.map((m: { bot_id: string; role: string }) => ({
            id: m.bot_id,
            name: m.bot_id,
            note: m.role === 'admin' ? '管理员' : undefined,
          }))
        } catch (e) {
          console.error('Failed to fetch group members', e)
        }
      }
    }
  } else {
    // Fetch friends
    if (cachedFriends.value.length === 0) {
      try {
        const friends = await getFriends(myName.value)
        // friends = [{ user_id, friend_id, status, created_at }]
        cachedFriends.value = (friends || []).map((f: { friend_id: string; status?: string }) => ({
          id: f.friend_id,
          name: f.friend_id,
          note: f.status === 'online' ? '在线' : undefined,
        }))
      } catch (e) {
        console.error('Failed to fetch friends', e)
      }
    }
  }

  showMention.value = true
  await nextTick()
  mentionSearchInput.value?.focus()
}

function closeMention() {
  showMention.value = false
  mentionFilter.value = ''
  mentionIndex.value = 0
  nextTick(() => inputRef.value?.focus())
}

// 点击面板外部 → 关闭@列表或快捷语面板
function handleClickOutside(e: MouseEvent) {
  const target = e.target as Node

  // 关闭@提及面板
  if (showMention.value) {
    const panel = document.querySelector('.mention-panel')
    if (panel && !panel.contains(target)) {
      closeMention()
    }
  }

  // 关闭快捷语面板（排除点击快捷语按钮本身）
  if (showShortcut.value) {
    const panel = document.querySelector('.shortcut-panel')
    const btn = document.querySelector('.shortcut-btn')
    if (panel && !panel.contains(target) && btn && !btn.contains(target)) {
      showShortcut.value = false
    }
  }
}

onMounted(() => {
  document.addEventListener('mousedown', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('mousedown', handleClickOutside)
})

function selectMention(item: MentionItem) {
  // Focus the chat input first, then insert @name
  if (inputRef.value) {
    inputRef.value.focus()
    nextTick(() => {
      document.execCommand('insertText', false, `@${item.name} `)
    })
  }
  closeMention()
}

function handleMentionKeydown(e: KeyboardEvent) {
  const list = filteredMentionList.value
  if (list.length === 0) return

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    mentionIndex.value = (mentionIndex.value + 1) % list.length
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    mentionIndex.value = (mentionIndex.value - 1 + list.length) % list.length
  } else if (e.key === 'Enter' || e.key === 'Tab') {
    e.preventDefault()
    const item = list[mentionIndex.value]
    if (item) selectMention(item)
  }
}

function handleInput() {
  // Detect "@" typed in input → open mention panel
  const text = inputRef.value?.innerText || ''
  // Check if last char is @
  if (text.endsWith('@') && !showMention.value) {
    openMention()
  }
}

// 工具栏按钮 — 匹配老ERP: sysfun (10个) + sysfun2 (2个)
const sysfunBtns = [
  { id: 'li-at', val: 'at', icon: 'fa-at', title: '@提及' },
  { id: 'li-exp', val: 'exp', icon: 'fa-smile-o', title: '表情' },
  { id: 'li-pic', val: 'pic', icon: 'fa-photo', title: '图片上传' },
  { id: 'li-fol', val: 'fol', icon: 'fa-folder-o', title: '文件上传' },
  { id: 'li-tag', val: 'tag', icon: 'fa-tag', title: '标签设置' },
  { id: 'li-gen', val: 'gen', icon: 'fa-pencil-square-o', title: '备注设置' },
  { id: 'li-cut', val: 'cut', icon: 'fa-cut', title: '裁剪' },
  { id: 'li-mic', val: 'mic', icon: 'fa-microphone', title: '语音' },
  { id: 'li-rmb', val: 'rmb', icon: 'fa-rmb', title: '付款', extraClass: 'fagetrmb' },
  { id: 'li-lis', val: 'lis', icon: 'fa-list', title: '付款记录' },
  { id: 'li-alt', val: 'alt', icon: 'fa-list-alt', title: '工单' },
  { id: 'li-done', val: 'done', icon: 'fa-check-circle', title: '标记已处理' },
]
const sysfunBtns2 = [
  { id: 'li-exc', val: 'exc', icon: 'fa-exchange', title: '客服转接' },
  { id: 'li-new', val: 'new', icon: 'fa-newspaper-o', title: '聊天记录' },
]

// 表情列表 (保留供后续表情面板使用)
// eslint-disable-next-line @typescript-eslint/no-unused-vars
const emojis = ['😊', '😂', '🤣', '❤️', '👍', '😍', '🥰', '😘', '😜', '😎',
  '🤔', '🙄', '😏', '😴', '🥱', '😭', '😤', '😡', '🤬', '👋',
  '✌️', '🤞', '🤝', '🙏', '💪', '🔥', '✨', '🌟', '⭐', '🎉',
  '🎊', '🎈', '🎁', '💡', '📌', '✅', '❌', '⚠️', '🚀', '💰']

function handleKeydown(e: KeyboardEvent) {
  if (showMention.value) {
    // Let mention panel handle its own keydown
    return
  }
  if (sendMode.value === 1 && e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    handleSend()
  } else if (sendMode.value === 2 && e.key === 'Enter' && e.ctrlKey) {
    e.preventDefault()
    handleSend()
  }
}

function handlePaste(e: ClipboardEvent) {
  // Handle rich text paste as plain text
  e.preventDefault()
  const text = e.clipboardData?.getData('text/plain') || ''
  document.execCommand('insertText', false, text)
}

function getInputText(): string {
  return inputRef.value?.innerText?.trim() || ''
}

function clearInput() {
  if (inputRef.value) inputRef.value.innerHTML = ''
}

function handleSend() {
  const content = getInputText()
  if (!content || !session.value) return
  const match = session.value.id.match(/^group:(\d+)$/)
  if (match) {
    ws.sendMessage(parseInt(match[1]), content)
  }
  clearInput()
}

function handleToolbar(btn: { val: string }) {
  switch (btn.val) {
    case 'at':
      openMention()
      break
    case 'exp':
      // Emoji picker: insert a random emoji into the input
      if (inputRef.value) {
        const emoji = emojis[Math.floor(Math.random() * emojis.length)]
        document.execCommand('insertText', false, emoji)
      }
      break
    case 'pic':
      imgInput.value?.click()
      break
    case 'fol':
      fileInput.value?.click()
      break
    case 'tag':
      alert('标签设置 - 开发中')
      break
    case 'gen':
      alert('备注设置 - 开发中')
      break
    case 'cut':
      // Crop functionality - placeholder
      break
    case 'mic':
      alert('语音 - 开发中')
      break
    case 'rmb':
      alert('付款 - 需与ERP对接')
      break
    case 'lis':
      alert('付款记录 - 需与ERP对接')
      break
    case 'alt':
      alert('工单 - 开发中')
      break
    case 'exc':
      alert('客服转接 - 开发中')
      break
    case 'new':
      alert('聊天记录 - 开发中')
      break
    case 'done':
      if (session.value) {
        chat.markDone(session.value.id)
      }
      break
  }
}

function handleImgSelect(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || !session.value) return
  const match = session.value.id.match(/^group:(\d+)$/)
  if (match) {
    const groupId = parseInt(match[1])
    uploadImage(file, groupId).catch(err => {
      console.error('Upload failed:', err)
      alert('图片上传失败: ' + err.message)
    })
  }
  input.value = ''
}

function handleFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file || !session.value) return
  const match = session.value.id.match(/^group:(\d+)$/)
  if (match) {
    ws.sendMessage(parseInt(match[1]), `[文件] ${file.name}`)
  }
  input.value = ''
}

// 图片预览：在新窗口打开
function previewImage(url: string) {
  window.open(url, '_blank')
}

function inviteFriends() {
  alert('邀请好友 - 开发中')
}

// ── 快捷语（分类+增删改，localStorage持久化）──
const STORAGE_KEY = 'chat12fz-shortcuts'

interface SCItem { id: string; text: string }
interface SCCat { id: string; name: string; items: SCItem[] }

function defaultSCData(): SCCat[] {
  return [
    { id: 'common', name: '常用', items: [
      { id: 's1', text: '收到' },
      { id: 's2', text: '好的，马上处理' },
      { id: 's3', text: '了解' },
    ]},
    { id: 'reply', name: '回复', items: [
      { id: 's4', text: '已处理' },
      { id: 's5', text: '请查看' },
      { id: 's6', text: '没问题' },
    ]},
    { id: 'cs', name: '客服', items: [
      { id: 's7', text: '稍等一下' },
      { id: 's8', text: '收到，我来处理' },
    ]},
  ]
}

function loadSCData(): SCCat[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return JSON.parse(raw)
  } catch {}
  return defaultSCData()
}

function saveSCData() {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(scData.value))
}

function genId(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 6)
}

const scData = ref<SCCat[]>(loadSCData())
const scActiveCat = ref(scData.value[0]?.id || '')
const scNewText = ref('')
const showShortcut = ref(false)
const scEditingCat = ref(false)
const scEditingCatName = ref('')
const scCatInput = ref<HTMLInputElement>()

const currentItems = computed(() => {
  const cat = scData.value.find(c => c.id === scActiveCat.value)
  return cat?.items || []
})

function openShortcut() {
  showShortcut.value = !showShortcut.value
  scEditingCat.value = false
}

function selectShortcut(text: string) {
  if (inputRef.value) {
    inputRef.value.focus()
    document.execCommand('insertText', false, text + ' ')
  }
  showShortcut.value = false
}

// 分类操作
function scAddCat() {
  const name = prompt('输入分类名称：')
  if (!name?.trim()) return
  const id = genId()
  scData.value.push({ id, name: name.trim(), items: [] })
  scActiveCat.value = id
  saveSCData()
}

function scDeleteCat() {
  if (!confirm('确定删除此分类及其所有快捷语？')) return
  scData.value = scData.value.filter(c => c.id !== scActiveCat.value)
  if (scData.value.length > 0) scActiveCat.value = scData.value[0].id
  scEditingCat.value = false
  saveSCData()
}

function scStartEditCat() {
  const cat = scData.value.find(c => c.id === scActiveCat.value)
  if (!cat) return
  scEditingCat.value = true
  scEditingCatName.value = cat.name
  nextTick(() => scCatInput.value?.focus())
}

function scSaveCatRename() {
  const name = scEditingCatName.value.trim()
  if (!name) return
  const cat = scData.value.find(c => c.id === scActiveCat.value)
  if (cat) cat.name = name
  scEditingCat.value = false
  saveSCData()
}

function scCancelCatEdit() {
  scEditingCat.value = false
}

// 快捷语操作
function scAddItem() {
  const text = scNewText.value.trim()
  if (!text) return
  const cat = scData.value.find(c => c.id === scActiveCat.value)
  if (!cat) return
  cat.items.push({ id: genId(), text })
  scNewText.value = ''
  saveSCData()
}

function scStartEdit(item: SCItem) {
  const newText = prompt('编辑快捷语：', item.text)
  if (newText?.trim()) {
    item.text = newText.trim()
    saveSCData()
  }
}

function scDeleteItem(item: SCItem) {
  const cat = scData.value.find(c => c.id === scActiveCat.value)
  if (!cat) return
  cat.items = cat.items.filter(i => i.id !== item.id)
  saveSCData()
}

// ── 格式化时间 ──
function formatTime(iso: string) {
  if (!iso) return ''
  const d = new Date(iso)
  const now = new Date()
  const isToday = d.toDateString() === now.toDateString()
  const time = d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' })
  if (isToday) return time
  return d.toLocaleDateString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' }) + ' ' + time
}

// ── 未读消息分割索引 ──
// 消息依 id 升序排列（最旧在上最新在下）
// unreadDividerIndex = messages.length - unread
// 指向第一条未读消息的索引
const unreadDividerIndex = computed(() => {
  const s = session.value
  if (!s || !s.messages.length || s.unread <= 0) return -1
  const idx = s.messages.length - s.unread
  return idx >= 0 && idx < s.messages.length ? idx : -1
})

// 标记：切换会话后等待消息加载完再滚到底
let pendingScrollToBottom = false

// 消息列表变化：切会话强制滚到底，新消息只在底部才滚
watch(() => session.value?.messages.length, async () => {
  await nextTick()
  if (!msgListRef.value) return
  if (pendingScrollToBottom) {
    msgListRef.value.scrollTop = msgListRef.value.scrollHeight
    pendingScrollToBottom = false
  } else {
    const el = msgListRef.value
    const nearBottom = el.scrollHeight - el.scrollTop - el.clientHeight < 100
    if (nearBottom) {
      el.scrollTop = el.scrollHeight
    }
  }
})

// 切换会话：重置状态 + 滚到底（flush:'post'确保DOM已更新）
watch(() => chat.activeId, async () => {
  await nextTick()
  loadedCount.value = 0
  hasMore.value = true
  isLoadingMore.value = false
  cachedMembers.value = []
  cachedFriends.value = []
  showMention.value = false
  showShortcut.value = false
  // 滚到底
  if (msgListRef.value) {
    msgListRef.value.scrollTop = msgListRef.value.scrollHeight
  }
  pendingScrollToBottom = true
}, { flush: 'post' })

// ── 滚动加载更多历史消息 ──
const loadedCount = ref(0)   // 当前已加载的消息数（作为offset）
const isLoadingMore = ref(false)
const hasMore = ref(true)

// 加载更多历史消息（每次20条）
async function loadMoreMessages() {
  if (isLoadingMore.value || !hasMore.value || !session.value) return
  const match = session.value.id.match(/^group:(\d+)$/)
  if (!match) return
  const groupId = parseInt(match[1])
  isLoadingMore.value = true
  try {
    const msgs = await getMessages(groupId, 20, loadedCount.value)
    if (msgs.length < 20) hasMore.value = false
    if (msgs.length > 0) {
      // 保留滚动位置：记录加载前的高度
      const list = msgListRef.value
      const prevHeight = list?.scrollHeight || 0
      chat.loadMessages(session.value.id, msgs)
      loadedCount.value += msgs.length
      // 加载后修正滚动位置（新消息加在顶部，会撑高）
      await nextTick()
      if (list) {
        list.scrollTop = list.scrollHeight - prevHeight
      }
    }
  } catch (e) {
    console.error('Failed to load more messages:', e)
  } finally {
    isLoadingMore.value = false
  }
}

// 监听滚动事件：滚到顶部时加载更多
function onScroll() {
  const list = msgListRef.value
  if (!list) return
  if (list.scrollTop <= 10 && !isLoadingMore.value && hasMore.value) {
    loadMoreMessages()
  }
}

</script><style scoped>
#chat-content {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}
.content-inner {
  flex: 1;
  display: flex;
  flex-direction: column;
  background-color: rgba(255,255,255,.8);
  border-radius: 4px;
  overflow: hidden;
  height: 100%;
}

/* 标题栏 */
.sz_othersTitle {
  padding: 8px 16px;
  border-bottom: 1px solid rgba(0,0,0,.06);
  position: relative;
  background: #f8f9fa;
}
.sz_othersTitle h3 {
  font-size: 17px;
  font-weight: 600;
  margin: 0;
  color: #333;
}
.invite-icon {
  position: absolute;
  right: 16px;
  font-size: 16px;
  color: #666;
  cursor: pointer;
  padding: 4px;
}
.invite-icon:hover { color: #333; }

/* 消息列表 */
.sz_chatcontent {
  flex: 1;
  overflow-y: auto;
  padding: 12px 16px;
  display: block;
}
.message-row { margin-bottom: 10px; }
.message {
  display: flex;
  gap: 8px;
  max-width: 80%;
}
.message.self {
  flex-direction: row-reverse;
  margin-left: auto;
}
.message.other { margin-right: auto; }

.msg-avatar {
  width: 32px; height: 32px;
  border-radius: 50%;
  background: #2d6cf0;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  flex-shrink: 0;
  margin-top: 4px;
}
.msg-body { max-width: 75%; }
.msg-sender { font-size: 13px; color: #888; margin-bottom: 2px; }
.msg-sender .bot-tag { font-size: 12px; margin-left: 2px; }
.self-sender { text-align: right; }
.msg-content {
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 16px;
  line-height: 1.6;
  word-break: break-word;
  white-space: pre-wrap;
}
.msg-image {
  max-width: 100%;
  max-height: 400px;
  border-radius: 4px;
  cursor: pointer;
  display: block;
}
.self-msg {
  background: #2d6cf0;
  color: #fff;
  border-bottom-right-radius: 2px;
}
.other-msg {
  background: #fff;
  color: #333;
  border-bottom-left-radius: 2px;
  box-shadow: 0 1px 2px rgba(0,0,0,.08);
}
.msg-time {
  font-size: 12px;
  color: #aaa;
  margin-top: 2px;
  text-align: right;
}
.message.other .msg-time { text-align: left; }
.empty-msg {
  text-align: center;
  color: #bbb;
  padding: 48px 16px;
  font-size: 16px;
}
.unread-divider {
  text-align: center;
  padding: 8px 0;
  position: relative;
}
.unread-divider span {
  background: #ffd700;
  color: #333;
  font-size: 12px;
  padding: 2px 12px;
  border-radius: 10px;
  position: relative;
  z-index: 1;
}

/* 工具栏 */
.sysfun {
  padding: 0 12px;
  height: 36px;
  justify-content: space-between !important;
  border-top: 1px solid rgba(0,0,0,.05);
  background: #f5f5f5;
}
.sysfun-left {
  display: flex;
  align-items: center;
  list-style: none;
  gap: 2px;
}
.sysfun-right {
  display: flex;
  align-items: center;
  list-style: none;
  gap: 2px;
}
.sysfun > ul > li {
  width: 32px;
  height: 32px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  border-radius: 3px;
}
.sysfun > ul > li:hover { background: rgba(0,0,0,.05); }
.sysfun > ul > li > i {
  color: rgba(80,80,80,.8);
  font-size: 18px;
  transition: transform .15s;
}
.sysfun > ul > li > i:hover { transform: scale(1.15); }
.fagetrmb {
  background: #fbd55c;
  border: 1px solid #f1eaaa;
  display: flex;
  justify-content: center;
  align-items: center;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  font-size: 13px;
  color: #e4ac40 !important;
  padding-top: 1px;
}

/* @提及面板 */
.mention-panel {
  position: relative;
  margin: 0 12px;
  background: #fff;
  border: 1px solid #ddd;
  border-radius: 6px;
  box-shadow: 0 -2px 12px rgba(0,0,0,.12);
  max-height: 200px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  z-index: 20;
}
.mention-search {
  display: flex;
  align-items: center;
  padding: 6px 10px;
  border-bottom: 1px solid #eee;
  background: #fafafa;
  gap: 6px;
}
.mention-search > i {
  color: #999;
  font-size: 14px;
}
.mention-search input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 14px;
  background: transparent;
  color: #333;
}
.mention-list {
  overflow-y: auto;
  flex: 1;
  padding: 4px 0;
}
.mention-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  cursor: pointer;
  transition: background .12s;
}
.mention-item:hover,
.mention-item.active {
  background: #e8f0fe;
}
.mention-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: #2d6cf0;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  flex-shrink: 0;
}
.mention-info {
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.mention-name {
  font-size: 14px;
  color: #333;
}
.mention-name .bot-tag,
.menton-info .bot-tag {
  margin-left: 2px;
}
.mention-note {
  font-size: 12px;
  color: #999;
}
.mention-empty {
  text-align: center;
  color: #bbb;
  padding: 16px;
  font-size: 14px;
}

/* 输入框 */
.chat_input {
  min-height: 80px;
  max-height: 120px;
  padding: 8px 12px;
  margin: 0 12px;
  background: #fff;
  border: 1px solid #e0e0e0;
  border-radius: 4px;
  overflow-y: auto;
  font-size: 16px;
  line-height: 1.6;
  outline: none;
  color: #333;
}
.chat_input:focus { border-color: #2d6cf0; }
.chat_input[contenteditable]:empty:before {
  content: attr(data-placeholder);
  color: #bbb;
}

/* 发送栏 */
.chat_enter {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 4px 12px 8px;
  color: #666;
  background: #f5f5f5;
  border-top: 1px solid rgba(0,0,0,.03);
}
.enter-mode {
  gap: 12px;
}
.enter-mode > span {
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 4px;
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 14px;
}
.enter-mode > span > i.fa-circle { color: #67c23a; font-size: 12px; }
.enter-mode > span > i.fa-circle-thin { font-size: 12px; color: #999; }
.enter-mode > span:hover { background: rgba(0,0,0,.04); }
.send-area {
  gap: 6px;
}
.send-btn {
  height: 32px;
  padding: 0 20px;
  color: #fff;
  background: #337ab7;
  border: 1px solid #2e6da4;
  border-radius: 3px;
  cursor: pointer;
  font-size: 15px;
}
.send-btn:active { background: #96c8f4; }
.shortcut-btn {
  font-size: 18px;
  cursor: pointer;
  padding: 4px;
  color: #666;
}
.shortcut-btn:hover { color: #333; }

/* 快捷语面板 */
.shortcut-panel {
  position: absolute;
  bottom: 48px;
  right: 12px;
  width: 280px;
  background: #fff;
  border: 1px solid #ddd;
  border-radius: 6px;
  box-shadow: 0 -2px 12px rgba(0,0,0,.12);
  z-index: 30;
  display: flex;
  flex-direction: column;
  max-height: 300px;
}
/* 分类Tab */
.sc-cats {
  display: flex;
  gap: 0;
  border-bottom: 1px solid #eee;
  background: #fafafa;
  padding: 0;
  flex-shrink: 0;
  overflow-x: auto;
}
.sc-cat {
  padding: 6px 12px;
  font-size: 13px;
  color: #666;
  cursor: pointer;
  white-space: nowrap;
  border-bottom: 2px solid transparent;
  user-select: none;
}
.sc-cat.active {
  color: #2d6cf0;
  border-bottom-color: #2d6cf0;
  background: #fff;
}
.sc-cat:hover { color: #333; }
.sc-cat-add {
  padding: 6px 10px;
  font-size: 14px;
  color: #999;
  cursor: pointer;
  flex-shrink: 0;
  user-select: none;
}
.sc-cat-add:hover { color: #2d6cf0; }
/* 分类编辑 */
.sc-cat-edit {
  display: flex;
  gap: 4px;
  padding: 6px 8px;
  border-bottom: 1px solid #eee;
  background: #fff;
}
.sc-cat-edit input {
  flex: 1;
  height: 26px;
  border: 1px solid #ddd;
  border-radius: 3px;
  padding: 0 6px;
  font-size: 13px;
  outline: none;
}
.sc-cat-edit input:focus { border-color: #2d6cf0; }
.sc-cat-edit button {
  height: 26px;
  padding: 0 8px;
  font-size: 12px;
  border: 1px solid #ddd;
  border-radius: 3px;
  background: #fff;
  cursor: pointer;
  color: #333;
}
.sc-cat-edit button:hover { background: #f5f5f5; }
.sc-del-btn { color: #f5222d !important; border-color: #f5222d !important; }
/* 快捷语列表 */
.sc-items {
  flex: 1;
  overflow-y: auto;
  padding: 4px 0;
}
.sc-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 5px 12px;
  cursor: pointer;
  transition: background .12s;
}
.sc-item:hover { background: #f5f7fa; }
.sc-item-text {
  font-size: 14px;
  color: #333;
  flex: 1;
}
.sc-item-ops {
  display: none;
  gap: 4px;
  flex-shrink: 0;
}
.sc-item:hover .sc-item-ops { display: flex; }
.sc-item-ops i {
  font-size: 12px;
  color: #999;
  padding: 2px;
  cursor: pointer;
}
.sc-item-ops i:hover { color: #2d6cf0; }
.sc-item-ops .fa-trash:hover { color: #f5222d; }
/* 添加快捷语 */
.sc-add-bar {
  display: flex;
  gap: 4px;
  padding: 6px 8px;
  border-top: 1px solid #eee;
  background: #fafafa;
}
.sc-add-bar input {
  flex: 1;
  height: 28px;
  border: 1px solid #ddd;
  border-radius: 3px;
  padding: 0 6px;
  font-size: 13px;
  outline: none;
}
.sc-add-bar input:focus { border-color: #2d6cf0; }
.sc-add-bar button {
  height: 28px;
  padding: 0 12px;
  background: #2d6cf0;
  color: #fff;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  font-size: 13px;
}
.sc-add-bar button:hover { background: #1a5cd6; }

/* 执行过程折叠块 */
.exec-log {
  margin-top: 6px;
  border-top: 1px solid rgba(0,0,0,.06);
  padding-top: 4px;
  font-size: 13px;
}
.exec-log summary {
  cursor: pointer;
  color: #888;
  font-size: 12px;
  user-select: none;
  padding: 2px 0;
}
.exec-log summary:hover { color: #555; }
.exec-steps {
  margin: 4px 0 0;
  padding: 6px 8px;
  background: rgba(0,0,0,.03);
  border-radius: 4px;
  font-size: 12px;
  line-height: 1.5;
  white-space: pre-wrap;
  font-family: inherit;
  color: #666;
  overflow-x: auto;
}
.self-msg .exec-steps {
  background: rgba(255,255,255,.12);
  color: rgba(255,255,255,.8);
}

/* 工具函数对应的隐藏元素 */
</style>

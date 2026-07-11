<script setup>
import { ref, reactive, computed, nextTick, watch, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  Promotion,
  ChatLineRound,
  WarningFilled,
  Loading,
  Picture,
  Document,
  Download,
  VideoCamera,
} from '@element-plus/icons-vue'
import { useChatStore } from '../store/chat'
import { useUserStore } from '../store/user'
import { api } from '../api'
import { buildFileMsg, parseFileMsg, humanSize } from '../utils/filemsg'
import { buildStickerMsg, parseStickerMsg, stickerSrc, loadStickers } from '../utils/sticker'
import Avatar from './Avatar.vue'

const chat = useChatStore()
const user = useUserStore()

const conv = computed(() => chat.activeConversation)
const messages = computed(() => chat.activeMessages)

const draft = ref('')
const scroller = ref(null)

const membersVisible = ref(false)
const members = ref([])
const loadingMembers = ref(false)
const joinRequests = ref([])
const roleLabels = { 0: '成员', 1: '管理员', 2: '群主' }
const inviteUid = ref('')
const inviting = ref(false)
const groupInfo = ref(null)
const annVisible = ref(false)
const annDraft = ref('')

const myRole = computed(() => {
  const me = members.value.find((m) => m.uid === user.uid)
  return me ? me.role : -1
})
const isOwner = computed(() => myRole.value === 2)
const isAdmin = computed(() => myRole.value >= 1)

function isMuted(mem) {
  if (!mem || !mem.mute_until) return false
  const t = new Date(mem.mute_until).getTime()
  return !Number.isNaN(t) && t > Date.now()
}
function canManage(mem) {
  if (mem.uid === user.uid) return false
  if (mem.role === 2) return false
  if (mem.role === 1 && !isOwner.value) return false
  return isAdmin.value
}

const imgInput = ref(null)
const fileInput = ref(null)
const uploadingFile = ref(false)
const fileUrls = reactive({}) // file_id -> dataUrl（图片消息渲染）

function imageUrl(fileId) {
  if (fileUrls[fileId] === undefined) {
    fileUrls[fileId] = ''
    api
      .getAvatar(user.token, fileId)
      .then((u) => {
        fileUrls[fileId] = u || ''
      })
      .catch(() => {
        fileUrls[fileId] = ''
      })
  }
  return fileUrls[fileId]
}

function fileOf(content) {
  return parseFileMsg(content)
}

const nameCache = reactive({})
function senderName(uid) {
  if (!uid) return ''
  if (uid === user.uid) return user.name || uid
  const f = chat.friends.find((x) => x.uid === uid)
  if (f) return f.name
  if (nameCache[uid] === undefined) {
    nameCache[uid] = uid
    api
      .userInfo(user.token, uid)
      .then((u) => {
        if (u && u.name) nameCache[uid] = u.name
      })
      .catch(() => {})
  }
  return nameCache[uid]
}

function canRecall(m) {
  return m.self && m.msgId && Date.now() - m.time < 2 * 60 * 1000
}
async function recall(m) {
  try {
    await api.messageRecall(user.token, m.msgId)
    chat.markRecalled(m.msgId)
  } catch (e) {
    ElMessage.error('撤回失败：' + String(e?.message || e))
  }
}

function stickerOf(content) {
  return parseStickerMsg(content)
}

const stickers = ref([])
onMounted(async () => {
  stickers.value = await loadStickers()
})

async function sendSticker(name) {
  const c = conv.value
  if (!c) return
  if (!chat.connected) {
    ElMessage.warning('未连接到服务器')
    return
  }
  const content = buildStickerMsg(name)
  try {
    const key = c.isGroup
      ? await api.sendGroupText(c.uid, content, [])
      : await api.sendText(c.uid, content)
    chat.addOutgoing(c.uid, content, Number(key))
    scrollToBottom()
  } catch (e) {
    ElMessage.error('发送失败：' + String(e?.message || e))
  }
}

function pickImage() {
  if (imgInput.value) imgInput.value.click()
}
function pickFile() {
  if (fileInput.value) fileInput.value.click()
}

function readFileAsDataURL(file) {
  return new Promise((resolve, reject) => {
    const r = new FileReader()
    r.onload = () => resolve(r.result)
    r.onerror = reject
    r.readAsDataURL(file)
  })
}

async function sendFile(kind, file, inputEl) {
  const c = conv.value
  if (!c) return
  if (!chat.connected) {
    ElMessage.warning('未连接到服务器')
    return
  }
  if (file.size > 10 * 1024 * 1024) {
    ElMessage.warning('文件不能超过 10MB')
    return
  }
  uploadingFile.value = true
  try {
    const dataUrl = await readFileAsDataURL(file)
    const base64 = dataUrl.split(',')[1] || ''
    const fileId = await api.uploadFile(
      user.token,
      base64,
      file.type || 'application/octet-stream'
    )
    const content = buildFileMsg(kind, fileId, file.name, file.size, file.type)
    const key = c.isGroup
      ? await api.sendGroupText(c.uid, content, [])
      : await api.sendText(c.uid, content)
    chat.addOutgoing(c.uid, content, Number(key))
    scrollToBottom()
  } catch (e) {
    ElMessage.error('发送失败：' + String(e?.message || e))
  } finally {
    uploadingFile.value = false
    if (inputEl) inputEl.value = ''
  }
}

function onImageSelected(e) {
  const f = e.target.files && e.target.files[0]
  if (f) sendFile('image', f, e.target)
}
function onFileSelected(e) {
  const f = e.target.files && e.target.files[0]
  if (f) sendFile('file', f, e.target)
}

async function downloadFile(fmsg) {
  try {
    const dataUrl = await api.getAvatar(user.token, fmsg.file_id)
    const base64 = (dataUrl || '').split(',')[1] || ''
    if (!base64) {
      ElMessage.warning('文件不存在')
      return
    }
    const path = await api.saveFile(fmsg.name || 'download', base64)
    if (path) ElMessage.success('已保存到：' + path)
  } catch (e) {
    ElMessage.error('下载失败：' + String(e?.message || e))
  }
}

async function doInvite() {
  const c = conv.value
  const uid = inviteUid.value.trim()
  if (!c || !uid) {
    ElMessage.warning('请输入对方 UID')
    return
  }
  inviting.value = true
  try {
    await api.groupInvite(user.token, c.uid, uid)
    ElMessage.success('已邀请入群')
    inviteUid.value = ''
    members.value = (await api.groupMembers(user.token, c.uid)) || []
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  } finally {
    inviting.value = false
  }
}

const mentions = ref([]) // [{uid, name}]
const mentionVisible = ref(false)

async function openMention() {
  const c = conv.value
  if (!c || !c.isGroup) return
  if (!members.value.length) {
    try {
      members.value = (await api.groupMembers(user.token, c.uid)) || []
    } catch (e) {
      /* ignore */
    }
  }
  mentionVisible.value = true
}

function pickMention(mem) {
  const name = mem.nickname || mem.uid
  const sep = draft.value && !draft.value.endsWith(' ') ? ' ' : ''
  draft.value += sep + '@' + name + ' '
  if (!mentions.value.find((m) => m.uid === mem.uid)) {
    mentions.value.push({ uid: mem.uid, name })
  }
  mentionVisible.value = false
}

async function showMembers() {
  const c = conv.value
  if (!c) return
  loadingMembers.value = true
  try {
    members.value = (await api.groupMembers(user.token, c.uid)) || []
    try {
      groupInfo.value = await api.groupInfo(user.token, c.uid)
    } catch (e) {
      groupInfo.value = null
    }
    membersVisible.value = true
    // 群主拉取入群申请
    const me = members.value.find((m) => m.uid === user.uid)
    if (me && me.role === 2) {
      try {
        joinRequests.value = (await api.groupJoinRequests(user.token, c.uid)) || []
      } catch (e) {
        joinRequests.value = []
      }
    } else {
      joinRequests.value = []
    }
  } catch (e) {
    ElMessage.error('获取成员失败：' + String(e?.message || e))
  } finally {
    loadingMembers.value = false
  }
}

async function approveJoin(req) {
  const c = conv.value
  if (!c) return
  try {
    await api.groupApprove(user.token, c.uid, req.uid)
    joinRequests.value = joinRequests.value.filter((r) => r.uid !== req.uid)
    members.value = (await api.groupMembers(user.token, c.uid)) || []
    ElMessage.success('已通过')
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}

async function rejectJoin(req) {
  const c = conv.value
  if (!c) return
  try {
    await api.groupReject(user.token, c.uid, req.uid)
    joinRequests.value = joinRequests.value.filter((r) => r.uid !== req.uid)
    ElMessage.success('已拒绝')
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}

async function refreshMembers() {
  const c = conv.value
  if (!c) return
  try {
    members.value = (await api.groupMembers(user.token, c.uid)) || []
  } catch (e) {
    /* ignore */
  }
}

async function kickMember(mem) {
  const c = conv.value
  if (!c) return
  try {
    await ElMessageBox.confirm(
      `确定将 ${mem.nickname || mem.uid} 移出群聊？`,
      '踢出成员',
      { type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await api.groupKick(user.token, c.uid, mem.uid)
    members.value = members.value.filter((m) => m.uid !== mem.uid)
    ElMessage.success('已移出')
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}

async function muteMember(mem) {
  const c = conv.value
  if (!c) return
  let minutes
  try {
    const { value } = await ElMessageBox.prompt('禁言时长（分钟，0 表示解除禁言）', '禁言', {
      inputValue: '10',
      inputPattern: /^\d+$/,
      inputErrorMessage: '请输入非负整数',
    })
    minutes = parseInt(value, 10)
  } catch {
    return
  }
  try {
    await api.groupMute(user.token, c.uid, mem.uid, minutes)
    await refreshMembers()
    ElMessage.success(minutes > 0 ? '已禁言' : '已解除禁言')
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}

async function unmuteMember(mem) {
  const c = conv.value
  if (!c) return
  try {
    await api.groupMute(user.token, c.uid, mem.uid, 0)
    await refreshMembers()
    ElMessage.success('已解除禁言')
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}

async function transferOwner(mem) {
  const c = conv.value
  if (!c) return
  try {
    await ElMessageBox.confirm(
      `确定将群主转让给 ${mem.nickname || mem.uid}？转让后你将成为普通成员。`,
      '转让群主',
      { type: 'warning' }
    )
  } catch {
    return
  }
  try {
    await api.groupTransfer(user.token, c.uid, mem.uid)
    await refreshMembers()
    if (groupInfo.value) groupInfo.value.owner_uid = mem.uid
    ElMessage.success('已转让群主')
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}

async function leaveGroup() {
  const c = conv.value
  if (!c) return
  try {
    await ElMessageBox.confirm('确定退出该群聊？', '退群', { type: 'warning' })
  } catch {
    return
  }
  try {
    await api.groupLeave(user.token, c.uid)
    membersVisible.value = false
    chat.removeConversation(c.uid)
    ElMessage.success('已退出群聊')
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}

async function disbandGroup() {
  const c = conv.value
  if (!c) return
  try {
    await ElMessageBox.confirm('确定解散该群聊？解散后不可恢复。', '解散群', {
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await api.groupDisband(user.token, c.uid)
    membersVisible.value = false
    chat.removeConversation(c.uid)
    ElMessage.success('群聊已解散')
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}

function openAnnounce() {
  annDraft.value = (groupInfo.value && groupInfo.value.announcement) || ''
  annVisible.value = true
}

async function saveAnnounce() {
  const c = conv.value
  if (!c) return
  try {
    await api.groupAnnounce(user.token, c.uid, annDraft.value)
    if (groupInfo.value) groupInfo.value.announcement = annDraft.value
    annVisible.value = false
    ElMessage.success('公告已发布')
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}

function scrollToBottom() {
  nextTick(() => {
    const el = scroller.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

// 仅在切换会话或“尾部新增”消息时滚到底；前插历史不改变尾部 id，因此不会触发，避免位置跳动。
watch(
  () => [
    chat.activeUid,
    messages.value.length ? messages.value[messages.value.length - 1].id : '',
  ],
  () => scrollToBottom(),
  { flush: 'post' }
)

const loadingMore = ref(false)
const historyExhausted = ref(false)
const HISTORY_PAGE = 30

async function loadMore() {
  const c = conv.value
  if (!c || loadingMore.value || historyExhausted.value) return
  const el = scroller.value
  const prevHeight = el ? el.scrollHeight : 0
  const prevTop = el ? el.scrollTop : 0
  const before = chat.oldestMessageTime(c.uid)
  loadingMore.value = true
  try {
    const list =
      (await api.messageHistory(
        user.token,
        c.isGroup ? '' : c.uid,
        c.isGroup ? c.uid : '',
        before,
        HISTORY_PAGE
      )) || []
    const added = chat.prependHistory(c.uid, list)
    if (list.length < HISTORY_PAGE) historyExhausted.value = true
    if (added > 0) {
      await nextTick()
      const el2 = scroller.value
      if (el2) el2.scrollTop = el2.scrollHeight - prevHeight + prevTop
    }
  } catch (e) {
    /* ignore */
  } finally {
    loadingMore.value = false
  }
}

function onMessagesScroll(e) {
  if (e.target.scrollTop <= 40) loadMore()
}

// 切换会话时重置翻页状态；若本地无历史则从服务器拉首页。
watch(
  conv,
  (c) => {
    historyExhausted.value = false
    loadingMore.value = false
    if (!c) return
    const uid = c.uid
    setTimeout(() => {
      if (
        conv.value &&
        conv.value.uid === uid &&
        messages.value.length === 0 &&
        !loadingMore.value
      ) {
        loadMore()
      }
    }, 250)
  },
  { immediate: true }
)

// 正在输入：发送端节流（每 2.5s 最多一次）；接收端用 ticking now 驱动到期隐藏。
let lastTypingSent = 0
function onInput() {
  const c = conv.value
  if (!c || !chat.connected) return
  const now = Date.now()
  if (now - lastTypingSent < 2500) return
  lastTypingSent = now
  api.sendTyping(c.isGroup ? '' : c.uid, c.isGroup ? c.uid : '').catch(() => {})
}

const nowTick = ref(Date.now())
let typingTimer = null
onMounted(() => {
  typingTimer = setInterval(() => {
    nowTick.value = Date.now()
  }, 1000)
})
onUnmounted(() => {
  if (typingTimer) clearInterval(typingTimer)
})
const peerTyping = computed(() => {
  const c = conv.value
  if (!c) return false
  const exp = chat.typing[c.uid]
  return !!exp && exp > nowTick.value
})

// 已读回执：当前正在查看的会话有新消息或切换会话时，向对端/群发送已读到的最新时间。
function sendReadReceipt() {
  const c = conv.value
  if (!c || !chat.connected) return
  const arr = messages.value
  if (!arr.length) return
  let upTo = 0
  for (const m of arr) if (m.time && m.time > upTo) upTo = m.time
  if (!upTo) return
  api
    .sendRead(c.isGroup ? '' : c.uid, c.isGroup ? c.uid : '', upTo)
    .catch(() => {})
}

watch(
  () => [
    chat.activeUid,
    messages.value.length ? messages.value[messages.value.length - 1].time : 0,
  ],
  () => sendReadReceipt(),
  { flush: 'post' }
)

async function send() {
  const text = draft.value.trim()
  if (!text) return
  const c = conv.value
  if (!c) return
  if (!chat.connected) {
    ElMessage.warning('未连接到服务器')
    return
  }
  const fullText = draft.value
  draft.value = ''
  try {
    let key
    if (c.isGroup) {
      const ids = mentions.value
        .filter((m) => fullText.includes('@' + m.name))
        .map((m) => m.uid)
      key = await api.sendGroupText(c.uid, text, Array.from(new Set(ids)))
      mentions.value = []
    } else {
      key = await api.sendText(c.uid, text)
    }
    chat.addOutgoing(c.uid, text, Number(key))
    scrollToBottom()
  } catch (e) {
    ElMessage.error('发送失败：' + String(e?.message || e))
  }
}


function startVideoCall() {
  const c = conv.value
  if (!c || c.isGroup) return
  if (!chat.connected) {
    ElMessage.warning('未连接到服务器')
    return
  }
  chat.requestVideoCall(c.uid, c.name)
}
function onKeydown(e) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    send()
  }
}
</script>

<template>
  <div class="chat-panel">
    <div v-if="!conv" class="empty-state">
      <el-icon :size="64" color="#d6d6d6"><ChatLineRound /></el-icon>
      <p>选择一个会话开始聊天</p>
    </div>

    <template v-else>
      <div class="chat-header">
        <span class="title">
          <span v-if="conv.isGroup" class="gtag">群</span>{{ conv.name }}
          <span v-if="peerTyping" class="typing-tip">
            {{ conv.isGroup ? '有人正在输入…' : '对方正在输入…' }}
          </span>
        </span>
        <div class="header-actions">
          <el-button
            v-if="!conv.isGroup"
            link
            class="video-btn"
            :icon="VideoCamera"
            title="视频通话"
            @click="startVideoCall"
          />
          <el-button
            v-if="conv.isGroup"
            link
            class="members-btn"
            :loading="loadingMembers"
            @click="showMembers"
          >
            成员
          </el-button>
        </div>
      </div>

      <div ref="scroller" class="messages selectable" @scroll="onMessagesScroll">
        <div v-if="loadingMore" class="history-tip">加载中…</div>
        <div v-else-if="historyExhausted && messages.length" class="history-tip">
          没有更多消息了
        </div>
        <div
          v-for="m in messages"
          :key="m.id"
          class="msg-row"
          :class="{ self: m.self }"
        >
          <Avatar
            :uid="m.fromUid || (m.self ? user.uid : conv.uid)"
            :name="senderName(m.fromUid || (m.self ? user.uid : conv.uid))"
            :size="38"
            class="clickable-avatar"
            @click="chat.viewUser(m.fromUid || (m.self ? user.uid : conv.uid))"
          />
          <div class="bubble-wrap">
            <template v-if="m.recalled">
              <div class="bubble recalled">
                {{ m.self ? '你撤回了一条消息' : (conv.isGroup ? senderName(m.fromUid) + ' 撤回了一条消息' : '对方撤回了一条消息') }}
              </div>
            </template>
            <template v-else>
              <div v-if="conv.isGroup && !m.self" class="sender-name">
                {{ senderName(m.fromUid) }}
              </div>
              <img
                v-if="stickerOf(m.content)"
                :src="stickerSrc(stickerOf(m.content).sticker)"
                class="bubble-sticker"
                alt=""
              />
              <template v-else-if="fileOf(m.content)">
                <el-image
                  v-if="fileOf(m.content).kind === 'image'"
                  :src="imageUrl(fileOf(m.content).file_id)"
                  :preview-src-list="[imageUrl(fileOf(m.content).file_id)]"
                  fit="cover"
                  class="bubble-img"
                  preview-teleported
                  hide-on-click-modal
                />
                <div
                  v-else
                  class="bubble file-card"
                  :class="{ self: m.self }"
                  @click="downloadFile(fileOf(m.content))"
                >
                  <el-icon :size="28" class="fc-icon"><Document /></el-icon>
                  <div class="fc-info">
                    <div class="fc-name">{{ fileOf(m.content).name }}</div>
                    <div class="fc-size">{{ humanSize(fileOf(m.content).size) }}</div>
                  </div>
                  <el-icon class="fc-dl"><Download /></el-icon>
                </div>
              </template>
              <div v-else class="bubble" :class="{ self: m.self }">{{ m.content }}</div>
              <el-button
                v-if="canRecall(m)"
                class="recall-btn"
                text
                size="small"
                @click="recall(m)"
              >
                撤回
              </el-button>
              <div v-if="m.self" class="msg-status">
                <el-icon v-if="m.status === 'sending'" class="spin"><Loading /></el-icon>
                <el-icon v-else-if="m.status === 'failed'" color="#fa5151"><WarningFilled /></el-icon>
                <span v-else-if="conv.isGroup && m.readers && m.readers.length" class="read-tag">
                  {{ m.readers.length }}人已读
                </span>
                <span v-else-if="!conv.isGroup" class="read-tag" :class="{ read: m.status === 'read' }">
                  {{ m.status === 'read' ? '已读' : '已送达' }}
                </span>
              </div>
            </template>
          </div>
        </div>
        <div v-if="!messages.length" class="no-msg">还没有消息，发送第一条吧</div>
      </div>

      <div class="input-area">
        <textarea
          v-model="draft"
          class="input-box"
          placeholder="输入消息，Enter 发送，Shift+Enter 换行"
          @keydown="onKeydown"
          @input="onInput"
        ></textarea>
        <div class="send-row">
          <div class="tools">
            <el-popover trigger="click" :width="280" placement="top">
              <template #reference>
                <el-button class="tool-btn emoji-btn" text>😀</el-button>
              </template>
              <div class="sticker-panel">
                <img
                  v-for="s in stickers"
                  :key="s"
                  :src="stickerSrc(s)"
                  class="sticker-item"
                  @click="sendSticker(s)"
                />
                <div v-if="!stickers.length" class="sticker-empty">
                  暂无表情，把图片放到 public/stickers/ 并在 index.json 列出
                </div>
              </div>
            </el-popover>
            <el-button
              class="tool-btn"
              text
              :icon="Picture"
              :loading="uploadingFile"
              title="发送图片"
              @click="pickImage"
            />
            <el-button class="tool-btn" text :icon="Document" title="发送文件" @click="pickFile" />
            <el-button v-if="conv.isGroup" class="tool-btn at-btn" text @click="openMention">
              @
            </el-button>
          </div>
          <el-button
            class="send-btn"
            color="#07c160"
            :icon="Promotion"
            @click="send"
          >
            发送
          </el-button>
          <input
            ref="imgInput"
            type="file"
            accept="image/*"
            style="display: none"
            @change="onImageSelected"
          />
          <input ref="fileInput" type="file" style="display: none" @change="onFileSelected" />
        </div>
      </div>

      <el-dialog v-model="membersVisible" title="群成员" width="360px" align-center>
        <div class="ann-box">
          <div class="ann-head">
            <span class="ann-title">群公告</span>
            <el-button v-if="isAdmin" link size="small" @click="openAnnounce">编辑</el-button>
          </div>
          <div class="ann-text">
            {{ (groupInfo && groupInfo.announcement) || '暂无公告' }}
          </div>
        </div>
        <div v-if="joinRequests.length" class="join-reqs">
          <div class="jr-title">入群申请 ({{ joinRequests.length }})</div>
          <div v-for="req in joinRequests" :key="req.uid" class="jr-row">
            <Avatar :uid="req.uid" :name="req.uid" :size="30" />
            <span class="jr-uid">{{ req.uid }}</span>
            <el-button size="small" type="primary" color="#07c160" @click="approveJoin(req)">
              通过
            </el-button>
            <el-button size="small" @click="rejectJoin(req)">拒绝</el-button>
          </div>
        </div>
        <div v-for="mem in members" :key="mem.uid" class="member-row">
          <Avatar
            :uid="mem.uid"
            :name="mem.nickname || mem.uid"
            :size="34"
            class="clickable-avatar"
            @click="chat.viewUser(mem.uid)"
          />
          <div class="member-info">
            <div class="member-name">
              {{ mem.nickname || mem.uid }}
              <span v-if="isMuted(mem)" class="member-mute-tag">禁言中</span>
            </div>
            <div class="member-uid">UID: {{ mem.uid }}</div>
          </div>
          <span class="member-role">{{ roleLabels[mem.role] || '成员' }}</span>
          <div v-if="canManage(mem)" class="member-actions">
            <el-button
              v-if="isMuted(mem)"
              link
              size="small"
              @click="unmuteMember(mem)"
            >
              解禁
            </el-button>
            <el-button v-else link size="small" @click="muteMember(mem)">禁言</el-button>
            <el-button link size="small" @click="kickMember(mem)">踢出</el-button>
            <el-button
              v-if="isOwner"
              link
              size="small"
              @click="transferOwner(mem)"
            >
              转让
            </el-button>
          </div>
        </div>
        <div v-if="!members.length" class="member-empty">暂无成员</div>
        <div class="invite-box">
          <el-input
            v-model="inviteUid"
            size="small"
            placeholder="输入对方 UID 邀请入群"
            @keyup.enter="doInvite"
          />
          <el-button
            size="small"
            type="primary"
            color="#07c160"
            :loading="inviting"
            @click="doInvite"
          >
            邀请
          </el-button>
        </div>
        <div class="group-actions">
          <el-button v-if="isOwner" type="danger" plain size="small" @click="disbandGroup">
            解散群聊
          </el-button>
          <el-button v-else size="small" @click="leaveGroup">退出群聊</el-button>
        </div>
      </el-dialog>

      <el-dialog v-model="annVisible" title="群公告" width="360px" align-center>
        <el-input
          v-model="annDraft"
          type="textarea"
          :rows="5"
          maxlength="1024"
          show-word-limit
          placeholder="输入群公告内容"
        />
        <template #footer>
          <el-button @click="annVisible = false">取消</el-button>
          <el-button type="primary" color="#07c160" @click="saveAnnounce">发布</el-button>
        </template>
      </el-dialog>

      <el-dialog v-model="mentionVisible" title="@ 群成员" width="300px" align-center>
        <div
          v-for="mem in members"
          :key="mem.uid"
          class="member-row pick"
          @click="pickMention(mem)"
        >
          <Avatar :uid="mem.uid" :name="mem.nickname || mem.uid" :size="32" />
          <div class="member-info">
            <div class="member-name">{{ mem.nickname || mem.uid }}</div>
          </div>
        </div>
        <div v-if="!members.length" class="member-empty">暂无成员</div>
      </el-dialog>
    </template>
  </div>
</template>

<style scoped>
.chat-panel {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--wx-bg);
  min-width: 0;
}
.empty-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--wx-text-sub);
  gap: 12px;
}
.chat-header {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  border-bottom: 1px solid var(--wx-border);
  background: var(--wx-header-bg);
}
.chat-header .title {
  font-size: 16px;
  font-weight: 500;
}
.typing-tip {
  margin-left: 8px;
  font-size: 12px;
  font-weight: 400;
  color: var(--wx-green);
}
.messages {
  flex: 1;
  overflow-y: auto;
  padding: 18px 20px;
}
.msg-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  margin-bottom: 18px;
}
.msg-row.self {
  flex-direction: row-reverse;
}
.msg-avatar {
  width: 38px;
  height: 38px;
  border-radius: 5px;
  color: #fff;
  font-size: 15px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.bubble-wrap {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  max-width: 60%;
}
.sender-name {
  flex-basis: 100%;
  font-size: 11px;
  color: var(--wx-text-sub);
  margin-bottom: 2px;
}
.clickable-avatar {
  cursor: pointer;
}
.bubble.recalled {
  background: transparent;
  color: var(--wx-text-sub);
  font-size: 12px;
  box-shadow: none;
}
.bubble.recalled::after {
  display: none;
}
.recall-btn {
  color: var(--wx-text-sub);
  padding: 0 4px;
}
.msg-row.self .bubble-wrap {
  flex-direction: row-reverse;
}
.bubble {
  position: relative;
  padding: 9px 13px;
  border-radius: 5px;
  background: #fff;
  font-size: 14px;
  line-height: 1.5;
  word-break: break-word;
  white-space: pre-wrap;
  box-shadow: 0 1px 1px rgba(0, 0, 0, 0.03);
}
.bubble.self {
  background: var(--wx-bubble-self);
}
.bubble::after {
  content: '';
  position: absolute;
  top: 12px;
  left: -5px;
  border: 5px solid transparent;
  border-right-color: #fff;
}
.bubble.self::after {
  left: auto;
  right: -5px;
  border-right-color: transparent;
  border-left-color: var(--wx-bubble-self);
}
.msg-status {
  display: flex;
  align-items: center;
}
.read-tag {
  font-size: 11px;
  color: var(--wx-text-sub);
  white-space: nowrap;
}
.read-tag.read {
  color: var(--wx-green);
}
.spin {
  animation: spin 1s linear infinite;
  color: var(--wx-text-sub);
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
.no-msg {
  text-align: center;
  color: var(--wx-text-sub);
  font-size: 12px;
  margin-top: 30px;
}
.history-tip {
  text-align: center;
  color: var(--wx-text-sub);
  font-size: 12px;
  padding: 6px 0;
}
.input-area {
  border-top: 1px solid var(--wx-border);
  background: var(--wx-bg);
  padding: 8px 16px 12px;
  display: flex;
  flex-direction: column;
}
.input-box {
  width: 100%;
  height: 80px;
  resize: none;
  border: none;
  outline: none;
  background: transparent;
  font-family: inherit;
  font-size: 14px;
  line-height: 1.5;
  color: var(--wx-text);
  padding: 6px 2px;
}
.send-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.tools {
  display: flex;
  align-items: center;
  gap: 2px;
}
.tool-btn {
  color: var(--wx-text-sub);
  font-size: 18px;
}
.send-btn {
  color: #fff;
}
.gtag {
  display: inline-block;
  font-size: 11px;
  color: #fff;
  background: var(--wx-green);
  border-radius: 2px;
  padding: 0 4px;
  margin-right: 6px;
  vertical-align: middle;
}
.header-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 6px;
}
.video-btn {
  color: var(--wx-text-sub);
  font-size: 18px;
}
.members-btn {
  margin-left: 0;
}
.member-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 2px;
}
.member-avatar {
  width: 34px;
  height: 34px;
  border-radius: 5px;
  color: #fff;
  font-size: 14px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.member-info {
  flex: 1;
  min-width: 0;
}
.member-name {
  font-size: 14px;
}
.member-uid {
  font-size: 11px;
  color: var(--wx-text-sub);
}
.member-role {
  font-size: 12px;
  color: var(--wx-green);
}
.member-empty {
  text-align: center;
  color: var(--wx-text-sub);
  font-size: 12px;
  padding: 20px;
}
.join-reqs {
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--wx-border);
}
.jr-title {
  font-size: 12px;
  color: var(--wx-text-sub);
  margin-bottom: 6px;
}
.jr-row {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
}
.jr-uid {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.invite-box {
  display: flex;
  gap: 6px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--wx-border);
}
.at-btn {
  font-size: 18px;
  color: var(--wx-text-sub);
}
.bubble-img {
  max-width: 180px;
  max-height: 200px;
  border-radius: 5px;
  cursor: pointer;
  display: block;
}
.bubble-sticker {
  width: 100px;
  height: 100px;
  object-fit: contain;
  display: block;
}
.emoji-btn {
  font-size: 17px;
}
.sticker-panel {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  max-height: 220px;
  overflow-y: auto;
}
.sticker-item {
  width: 56px;
  height: 56px;
  object-fit: contain;
  border-radius: 4px;
  cursor: pointer;
}
.sticker-item:hover {
  background: var(--wx-list-hover);
}
.sticker-empty {
  font-size: 12px;
  color: var(--wx-text-sub);
  padding: 16px 8px;
  text-align: center;
}
.file-card {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 180px;
  max-width: 240px;
  cursor: pointer;
}
.fc-icon {
  color: #5b8def;
  flex-shrink: 0;
}
.fc-info {
  flex: 1;
  min-width: 0;
}
.fc-name {
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fc-size {
  font-size: 11px;
  color: var(--wx-text-sub);
}
.fc-dl {
  color: var(--wx-text-sub);
  flex-shrink: 0;
}
.member-row.pick {
  cursor: pointer;
  border-radius: 4px;
}
.member-row.pick:hover {
  background: var(--wx-list-hover);
}
.ann-box {
  margin-bottom: 10px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--wx-border);
}
.ann-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.ann-title {
  font-size: 12px;
  color: var(--wx-text-sub);
}
.ann-text {
  font-size: 13px;
  white-space: pre-wrap;
  word-break: break-word;
  margin-top: 2px;
}
.member-mute-tag {
  margin-left: 6px;
  font-size: 11px;
  color: #fa5151;
  border: 1px solid #fa5151;
  border-radius: 3px;
  padding: 0 3px;
}
.member-actions {
  display: flex;
  gap: 2px;
}
.group-actions {
  margin-top: 12px;
  text-align: center;
}
</style>

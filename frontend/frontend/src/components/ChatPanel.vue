<script setup>
import { ref, computed, nextTick, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { Promotion, ChatLineRound, WarningFilled, Loading } from '@element-plus/icons-vue'
import { useChatStore } from '../store/chat'
import { useUserStore } from '../store/user'
import { api } from '../api'
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
const roleLabels = { 0: '成员', 1: '管理员', 2: '群主' }
const inviteUid = ref('')
const inviting = ref(false)

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

async function showMembers() {
  const c = conv.value
  if (!c) return
  loadingMembers.value = true
  try {
    members.value = (await api.groupMembers(user.token, c.uid)) || []
    membersVisible.value = true
  } catch (e) {
    ElMessage.error('获取成员失败：' + String(e?.message || e))
  } finally {
    loadingMembers.value = false
  }
}

function scrollToBottom() {
  nextTick(() => {
    const el = scroller.value
    if (el) el.scrollTop = el.scrollHeight
  })
}

watch(
  () => [chat.activeUid, messages.value.length],
  () => scrollToBottom(),
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
  draft.value = ''
  try {
    const key = c.isGroup
      ? await api.sendGroupText(c.uid, text)
      : await api.sendText(c.uid, text)
    chat.addOutgoing(c.uid, text, Number(key))
    scrollToBottom()
  } catch (e) {
    ElMessage.error('发送失败：' + String(e?.message || e))
  }
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
        </span>
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

      <div ref="scroller" class="messages selectable">
        <div
          v-for="m in messages"
          :key="m.id"
          class="msg-row"
          :class="{ self: m.self }"
        >
          <Avatar
            :uid="m.self ? user.uid : conv.uid"
            :name="m.self ? user.name : conv.name"
            :group="!m.self && conv.isGroup"
            :size="38"
          />
          <div class="bubble-wrap">
            <div class="bubble" :class="{ self: m.self }">{{ m.content }}</div>
            <div v-if="m.self && m.status !== 'sent'" class="msg-status">
              <el-icon v-if="m.status === 'sending'" class="spin"><Loading /></el-icon>
              <el-icon v-else-if="m.status === 'failed'" color="#fa5151"><WarningFilled /></el-icon>
            </div>
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
        ></textarea>
        <div class="send-row">
          <el-button
            class="send-btn"
            color="#07c160"
            :icon="Promotion"
            @click="send"
          >
            发送
          </el-button>
        </div>
      </div>

      <el-dialog v-model="membersVisible" title="群成员" width="320px" align-center>
        <div v-for="mem in members" :key="mem.uid" class="member-row">
          <Avatar :uid="mem.uid" :name="mem.nickname || mem.uid" :size="34" />
          <div class="member-info">
            <div class="member-name">{{ mem.nickname || mem.uid }}</div>
            <div class="member-uid">UID: {{ mem.uid }}</div>
          </div>
          <span class="member-role">{{ roleLabels[mem.role] || '成员' }}</span>
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
  gap: 6px;
  max-width: 60%;
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
  justify-content: flex-end;
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
.members-btn {
  margin-left: auto;
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
.invite-box {
  display: flex;
  gap: 6px;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--wx-border);
}
</style>

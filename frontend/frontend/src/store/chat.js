import { defineStore } from 'pinia'

let _mid = 0
function nextId() {
  _mid += 1
  return `${Date.now()}_${_mid}`
}

function parseTime(t) {
  if (!t) return Date.now()
  const ms = new Date(t).getTime()
  return Number.isNaN(ms) ? Date.now() : ms
}

export const useChatStore = defineStore('chat', {
  state: () => ({
    selfUid: '',
    conversations: [], // {uid, name, avatar, last, time, unread}
    messages: {}, // uid -> [{id, content, time, self, status, key}]
    activeUid: '',
    connected: false,
  }),

  getters: {
    sortedConversations: (s) =>
      [...s.conversations].sort((a, b) => (b.time || 0) - (a.time || 0)),
    activeConversation: (s) =>
      s.conversations.find((c) => c.uid === s.activeUid) || null,
    activeMessages: (s) => s.messages[s.activeUid] || [],
    totalUnread: (s) =>
      s.conversations.reduce((sum, c) => sum + (c.unread || 0), 0),
  },

  actions: {
    init(selfUid) {
      this.selfUid = selfUid
    },

    setConnected(v) {
      this.connected = v
    },

    ensureConversation(uid, name) {
      let conv = this.conversations.find((c) => c.uid === uid)
      if (!conv) {
        conv = { uid, name: name || uid, avatar: '', last: '', time: 0, unread: 0 }
        this.conversations.push(conv)
        this.messages[uid] = this.messages[uid] || []
      } else if (name && conv.name === conv.uid) {
        conv.name = name
      }
      return conv
    },

    setActive(uid) {
      this.activeUid = uid
      const conv = this.conversations.find((c) => c.uid === uid)
      if (conv) conv.unread = 0
    },

    _touch(uid, content, time, incrUnread) {
      const conv = this.conversations.find((c) => c.uid === uid)
      if (!conv) return
      conv.last = content
      conv.time = time
      if (incrUnread && uid !== this.activeUid) {
        conv.unread = (conv.unread || 0) + 1
      }
    },

    _push(uid, msg) {
      if (!this.messages[uid]) this.messages[uid] = []
      this.messages[uid].push(msg)
    },

    // 本地发出的消息
    addOutgoing(uid, content, key) {
      const time = Date.now()
      const msg = { id: nextId(), content, time, self: true, status: 'sending', key }
      this.ensureConversation(uid)
      this._push(uid, msg)
      this._touch(uid, content, time, false)
      return msg
    },

    // ack/nack 回执更新发送状态
    markStatus(key, status) {
      for (const uid of Object.keys(this.messages)) {
        const m = this.messages[uid].find((x) => x.key === key)
        if (m) {
          m.status = status
          return
        }
      }
    },

    // 实时收到的文本：后端路由帧不含发送者，归入当前会话。
    receiveText(content) {
      const uid = this.activeUid || '__unknown__'
      const time = Date.now()
      this.ensureConversation(uid, uid === '__unknown__' ? '新消息' : undefined)
      this._push(uid, { id: nextId(), content, time, self: false, status: 'recv' })
      this._touch(uid, content, time, true)
    },

    // 离线同步的消息：含 from_uid / to_uid，可正确归属。
    receiveOffline(m) {
      const from = m.from_uid
      const to = m.to_uid
      const self = from === this.selfUid
      const peer = self ? to : from
      if (!peer) return
      const time = parseTime(m.created_at)
      this.ensureConversation(peer)
      this._push(peer, {
        id: m.msg_id || nextId(),
        content: m.content || '',
        time,
        self,
        status: self ? 'sent' : 'recv',
      })
      this._touch(peer, m.content || '', time, !self)
    },

    removeConversation(uid) {
      this.conversations = this.conversations.filter((c) => c.uid !== uid)
      delete this.messages[uid]
      if (this.activeUid === uid) this.activeUid = ''
    },

    reset() {
      this.conversations = []
      this.messages = {}
      this.activeUid = ''
      this.connected = false
    },
  },
})

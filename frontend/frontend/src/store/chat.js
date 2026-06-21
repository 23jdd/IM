import { defineStore } from 'pinia'
import { parseFileMsg } from '../utils/filemsg'
import { parseStickerMsg } from '../utils/sticker'
import { api } from '../api'

let _mid = 0
function nextId() {
  _mid += 1
  return `${Date.now()}_${_mid}`
}

function preview(content) {
  if (parseStickerMsg(content)) return '[表情]'
  const f = parseFileMsg(content)
  if (f) return f.kind === 'image' ? '[图片]' : '[文件]'
  return content
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
    friends: [], // {uid, name, avatar}
    avatarCache: {}, // uid -> dataUrl ('' 表示无头像)
    friendRequests: [], // 收到的好友申请
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

    setAvatarCache(uid, url) {
      if (uid) this.avatarCache[uid] = url || ''
    },

    setFriendRequests(list) {
      this.friendRequests = list || []
    },

    setFriends(list) {
      this.friends = (list || []).map((f) => ({
        uid: f.uid,
        name: f.remark || f.name || f.uid,
        avatar: f.avatar || '',
      }))
      // 同步好友名到已有会话（此前仅有 uid 的占位名）
      for (const f of this.friends) {
        const conv = this.conversations.find((c) => c.uid === f.uid)
        if (conv && conv.name === conv.uid) conv.name = f.name
      }
    },

    loadConversations(list) {
      for (const c of list || []) {
        if (!c.peer) continue
        const friend = this.friends.find((f) => f.uid === c.peer)
        const conv = this.ensureConversation(c.peer, friend ? friend.name : undefined)
        const t = parseTime(c.time)
        if (!conv.time || t >= conv.time) {
          conv.last = c.content || conv.last
          conv.time = t
        }
      }
    },

    ensureConversation(uid, name, isGroup) {
      let conv = this.conversations.find((c) => c.uid === uid)
      if (!conv) {
        let displayName = name
        if (!displayName && !isGroup) {
          const f = this.friends.find((x) => x.uid === uid)
          if (f) displayName = f.name
        }
        conv = {
          uid,
          name: displayName || uid,
          avatar: '',
          last: '',
          time: 0,
          unread: 0,
          isGroup: !!isGroup,
        }
        this.conversations.push(conv)
        this.messages[uid] = this.messages[uid] || []
      } else {
        if (name && conv.name === conv.uid) conv.name = name
        if (isGroup) conv.isGroup = true
      }
      return conv
    },

    loadGroups(list) {
      for (const g of list || []) {
        if (!g.group_id) continue
        this.ensureConversation(g.group_id, g.name || '群聊', true)
      }
    },

    setActive(uid) {
      this.activeUid = uid
      const conv = this.conversations.find((c) => c.uid === uid)
      if (conv) conv.unread = 0
      if (!this.messages[uid] || this.messages[uid].length === 0) {
        this.loadHistory(uid)
      }
    },

    _touch(uid, content, time, incrUnread) {
      const conv = this.conversations.find((c) => c.uid === uid)
      if (!conv) return
      conv.last = preview(content)
      conv.time = time
      if (incrUnread && uid !== this.activeUid) {
        conv.unread = (conv.unread || 0) + 1
      }
    },

    _push(uid, msg) {
      if (!this.messages[uid]) this.messages[uid] = []
      this.messages[uid].push(msg)
      this._persist(uid, msg)
    },

    _persist(uid, msg) {
      if (!uid || uid === '__unknown__') return
      const fromUid = msg.fromUid || (msg.self ? this.selfUid : uid)
      try {
        api
          .localSave(
            uid,
            String(msg.id || ''),
            fromUid,
            msg.content || '',
            !!msg.self,
            msg.status || '',
            msg.time || Date.now()
          )
          .catch(() => {})
      } catch {
        /* ignore */
      }
    },

    async loadHistory(uid) {
      if (!uid || uid === '__unknown__') return
      if (this.messages[uid] && this.messages[uid].length) return
      try {
        const rows = await api.localLoad(uid, 200)
        if (!rows || !rows.length) return
        if (this.messages[uid] && this.messages[uid].length) return
        this.messages[uid] = rows.map((r) => ({
          id: r.msg_id || nextId(),
          fromUid: r.from_uid,
          content: r.content,
          time: r.ts,
          self: r.self,
          status: r.status || (r.self ? 'sent' : 'recv'),
        }))
      } catch {
        /* ignore */
      }
    },

    // 本地发出的消息
    addOutgoing(uid, content, key) {
      const time = Date.now()
      const msg = {
        id: nextId(),
        fromUid: this.selfUid,
        content,
        time,
        self: true,
        status: 'sending',
        key,
      }
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

    // 实时收到的文本：单聊按 from_uid 归属；群聊按 group_id 归属。
    receiveText(payload) {
      const content = (payload && payload.content) || ''
      const from = (payload && payload.from_uid) || ''
      const groupId = (payload && payload.group_id) || ''
      let uid
      let self
      if (groupId) {
        uid = groupId
        self = from === this.selfUid
      } else {
        uid = from || this.activeUid || '__unknown__'
        self = false
      }
      const time = Date.now()
      const placeholderName = groupId
        ? '群聊 ' + groupId
        : uid === '__unknown__'
        ? '新消息'
        : undefined
      this.ensureConversation(uid, placeholderName, !!groupId)
      this._push(uid, {
        id: (payload && payload.msg_id) || nextId(),
        fromUid: from || uid,
        content,
        time,
        self,
        status: self ? 'sent' : 'recv',
      })
      this._touch(uid, content, time, !self)
    },

    // 离线同步的消息：含 from_uid / to_uid / group_id，可正确归属。
    receiveOffline(m) {
      const from = m.from_uid
      const self = from === this.selfUid
      let peer
      let isGroup = false
      if (m.group_id) {
        peer = m.group_id
        isGroup = true
      } else {
        peer = self ? m.to_uid : from
      }
      if (!peer) return
      const time = parseTime(m.created_at)
      this.ensureConversation(peer, undefined, isGroup)
      this._push(peer, {
        id: m.msg_id || nextId(),
        fromUid: from,
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
      this.friends = []
      this.avatarCache = {}
      this.friendRequests = []
    },
  },
})

import { defineStore } from 'pinia'
import { api } from '../api'

export const useUserStore = defineStore('user', {
  state: () => ({
    token: '',
    uid: '',
    name: '',
    profile: null,
    avatarUrl: '', // 头像 data URL（不持久化，登录后按 avatar id 解析）
  }),
  actions: {
    // 从本地 SQLite 的 session 记录恢复登录态（在 app 启动时调用）。
    restore(s) {
      if (!s) return
      this.token = s.token || ''
      this.uid = s.uid || ''
      this.name = s.name || ''
      try {
        this.profile = s.profile ? JSON.parse(s.profile) : null
      } catch {
        this.profile = null
      }
    },

    _persist() {
      try {
        api
          .saveSession(
            this.token,
            this.uid,
            this.name,
            this.profile ? JSON.stringify(this.profile) : ''
          )
          .catch(() => {})
      } catch {
        /* ignore */
      }
    },

    setLogin({ token, uid, name }) {
      this.token = token
      this.uid = uid
      this.name = name
      this._persist()
    },

    setProfile(p) {
      this.profile = p
      if (p && p.name) this.name = p.name
      this._persist()
    },

    setAvatarUrl(url) {
      this.avatarUrl = url || ''
    },

    logout() {
      this.token = ''
      this.uid = ''
      this.name = ''
      this.profile = null
      this.avatarUrl = ''
      try {
        api.clearSession().catch(() => {})
      } catch {
        /* ignore */
      }
    },
  },
})

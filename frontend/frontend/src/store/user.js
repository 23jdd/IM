import { defineStore } from 'pinia'

const KEY = 'im_user'

function load() {
  try {
    return JSON.parse(localStorage.getItem(KEY)) || {}
  } catch {
    return {}
  }
}

export const useUserStore = defineStore('user', {
  state: () => {
    const saved = load()
    return {
      token: saved.token || '',
      uid: saved.uid || '',
      name: saved.name || '',
      profile: saved.profile || null,
    }
  },
  actions: {
    persist() {
      localStorage.setItem(
        KEY,
        JSON.stringify({
          token: this.token,
          uid: this.uid,
          name: this.name,
          profile: this.profile,
        })
      )
    },
    setLogin({ token, uid, name }) {
      this.token = token
      this.uid = uid
      this.name = name
      this.persist()
    },
    setProfile(p) {
      this.profile = p
      if (p && p.name) this.name = p.name
      this.persist()
    },
    logout() {
      this.token = ''
      this.uid = ''
      this.name = ''
      this.profile = null
      localStorage.removeItem(KEY)
    },
  },
})

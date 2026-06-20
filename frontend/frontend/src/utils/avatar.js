import { useUserStore } from '../store/user'
import { useChatStore } from '../store/chat'
import { api } from '../api'

// 模块级 in-flight 表：同一 uid 的并发解析只发一次请求。
const inflight = new Map()

// resolveAvatar 按 uid 解析头像 data URL，结果缓存到 chat store。
// 无头像/失败返回空串（前端退化为首字母占位）。
export async function resolveAvatar(uid) {
  if (!uid) return ''
  const chat = useChatStore()
  const cached = chat.avatarCache[uid]
  if (cached !== undefined) return cached
  if (inflight.has(uid)) return inflight.get(uid)

  const user = useUserStore()
  const p = api
    .getAvatarByUid(user.token, uid)
    .then((url) => {
      const v = url || ''
      chat.setAvatarCache(uid, v)
      inflight.delete(uid)
      return v
    })
    .catch(() => {
      chat.setAvatarCache(uid, '')
      inflight.delete(uid)
      return ''
    })
  inflight.set(uid, p)
  return p
}

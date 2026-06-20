// 时间格式化：今天显示 HH:mm，昨天显示“昨天”，更早显示 MM/DD
export function formatTime(ts) {
  if (!ts) return ''
  const d = new Date(ts)
  const now = new Date()
  const pad = (n) => String(n).padStart(2, '0')
  const hm = `${pad(d.getHours())}:${pad(d.getMinutes())}`
  const sameDay =
    d.getFullYear() === now.getFullYear() &&
    d.getMonth() === now.getMonth() &&
    d.getDate() === now.getDate()
  if (sameDay) return hm

  const yest = new Date(now)
  yest.setDate(now.getDate() - 1)
  const isYest =
    d.getFullYear() === yest.getFullYear() &&
    d.getMonth() === yest.getMonth() &&
    d.getDate() === yest.getDate()
  if (isYest) return '昨天'

  return `${pad(d.getMonth() + 1)}/${pad(d.getDate())}`
}

const COLORS = ['#07c160', '#10aeff', '#ffc300', '#f56c6c', '#9b59b6', '#576b95']

export function avatarColor(seed) {
  const s = String(seed || '')
  let sum = 0
  for (let i = 0; i < s.length; i++) sum += s.charCodeAt(i)
  return COLORS[sum % COLORS.length]
}

export function avatarText(name) {
  const s = String(name || '?').trim()
  if (!s) return '?'
  return s.slice(0, 1).toUpperCase()
}

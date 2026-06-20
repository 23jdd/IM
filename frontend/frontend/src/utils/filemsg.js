// 文件消息：复用文本消息通道，content 为带 t:'f' 标记的 JSON。
export function buildFileMsg(kind, fileId, name, size, mime) {
  return JSON.stringify({ t: 'f', kind, file_id: fileId, name, size, mime })
}

export function parseFileMsg(content) {
  if (!content || content[0] !== '{') return null
  try {
    const o = JSON.parse(content)
    if (o && o.t === 'f' && o.file_id) return o
  } catch {
    /* not a file message */
  }
  return null
}

export function humanSize(n) {
  if (!n) return ''
  if (n < 1024) return n + ' B'
  if (n < 1024 * 1024) return (n / 1024).toFixed(1) + ' KB'
  return (n / 1024 / 1024).toFixed(1) + ' MB'
}

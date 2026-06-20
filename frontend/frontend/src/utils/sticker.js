// 表情消息：表情是预置静态资源（public/stickers/），消息只传文件名，
// 复用文本通道（后端无需改），接收端用相同的本地资源显示。
export function buildStickerMsg(name) {
  return JSON.stringify({ t: 's', sticker: name })
}

export function parseStickerMsg(content) {
  if (!content || content[0] !== '{') return null
  try {
    const o = JSON.parse(content)
    if (o && o.t === 's' && o.sticker) return o
  } catch {
    /* not a sticker message */
  }
  return null
}

// 表情资源 URL（public/stickers/ 下的图片）。
export function stickerSrc(name) {
  return '/stickers/' + name
}

// 加载表情清单（public/stickers/index.json，内容为文件名数组）。
export async function loadStickers() {
  try {
    const res = await fetch('/stickers/index.json')
    if (!res.ok) return []
    const list = await res.json()
    return Array.isArray(list) ? list : []
  } catch {
    return []
  }
}

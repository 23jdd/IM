<script setup>
import { ref, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Star, StarFilled, ChatLineSquare, Picture } from '@element-plus/icons-vue'
import { useUserStore } from '../store/user'
import { useChatStore } from '../store/chat'
import { api } from '../api'
import Avatar from './Avatar.vue'
import { formatTime } from '../utils/format'

const user = useUserStore()
const chat = useChatStore()

const moments = ref([])
const loading = ref(false)
const publishing = ref(false)

const draft = ref('')
const newImages = ref([]) // data URL 列表
const fileInput = ref(null)

const imageUrls = reactive({}) // imageId -> dataUrl
const commentDraft = reactive({}) // momentId -> text
const commentOpen = reactive({}) // momentId -> bool

function nameOf(uid) {
  if (uid === user.uid) return user.name || uid
  const f = chat.friends.find((x) => x.uid === uid)
  return f ? f.name : uid
}

function timeText(s) {
  const t = new Date(s).getTime()
  return Number.isNaN(t) ? '' : formatTime(t)
}

async function resolveImages(list) {
  for (const m of list) {
    for (const id of m.images || []) {
      if (imageUrls[id] !== undefined) continue
      imageUrls[id] = ''
      try {
        imageUrls[id] = await api.getAvatar(user.token, id)
      } catch {
        imageUrls[id] = ''
      }
    }
  }
}

async function loadTimeline() {
  loading.value = true
  try {
    const list = (await api.momentTimeline(user.token)) || []
    moments.value = list
    await resolveImages(list)
  } catch (e) {
    ElMessage.error('加载失败：' + String(e?.message || e))
  } finally {
    loading.value = false
  }
}

function pickImages() {
  if (fileInput.value) fileInput.value.click()
}

function readAsDataURL(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result)
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

async function onFilesChange(e) {
  const files = Array.from(e.target.files || [])
  for (const file of files) {
    if (!file.type.startsWith('image/')) continue
    if (file.size > 2 * 1024 * 1024) {
      ElMessage.warning('单张图片不能超过 2MB')
      continue
    }
    if (newImages.value.length >= 9) break
    newImages.value.push(await readAsDataURL(file))
  }
  if (fileInput.value) fileInput.value.value = ''
}

function removeNewImage(i) {
  newImages.value.splice(i, 1)
}

async function publish() {
  const content = draft.value.trim()
  if (!content && newImages.value.length === 0) {
    ElMessage.warning('说点什么或选张图片吧')
    return
  }
  publishing.value = true
  try {
    await api.momentPublish(user.token, content, newImages.value)
    draft.value = ''
    newImages.value = []
    ElMessage.success('已发布')
    await loadTimeline()
  } catch (e) {
    ElMessage.error('发布失败：' + String(e?.message || e))
  } finally {
    publishing.value = false
  }
}

function likedByMe(m) {
  return (m.likes || []).includes(user.uid)
}

async function toggleLike(m) {
  try {
    const liked = await api.momentLike(user.token, m.moment_id)
    if (!m.likes) m.likes = []
    const idx = m.likes.indexOf(user.uid)
    if (liked && idx < 0) m.likes.push(user.uid)
    if (!liked && idx >= 0) m.likes.splice(idx, 1)
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}

async function submitComment(m) {
  const text = (commentDraft[m.moment_id] || '').trim()
  if (!text) return
  try {
    await api.momentComment(user.token, m.moment_id, text)
    if (!m.comments) m.comments = []
    m.comments.push({ uid: user.uid, content: text, created_at: new Date().toISOString() })
    commentDraft[m.moment_id] = ''
    commentOpen[m.moment_id] = false
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}

function previewList(m) {
  return (m.images || []).map((id) => imageUrls[id]).filter(Boolean)
}

async function removeMoment(m) {
  try {
    await ElMessageBox.confirm('确定删除这条动态吗？', '提示', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await api.momentDelete(user.token, m.moment_id)
    moments.value = moments.value.filter((x) => x.moment_id !== m.moment_id)
    ElMessage.success('已删除')
  } catch (e) {
    ElMessage.error('删除失败：' + String(e?.message || e))
  }
}

onMounted(loadTimeline)
</script>

<template>
  <div class="moments">
    <div class="mo-header">朋友圈</div>

    <div class="mo-scroll">
      <div class="publish">
        <textarea v-model="draft" class="pub-input" placeholder="分享新鲜事..."></textarea>
        <div v-if="newImages.length" class="pub-imgs">
          <div v-for="(img, i) in newImages" :key="i" class="pub-img">
            <img :src="img" alt="" />
            <span class="rm" @click="removeNewImage(i)">×</span>
          </div>
        </div>
        <div class="pub-bar">
          <el-button :icon="Picture" text @click="pickImages">图片</el-button>
          <el-button
            type="primary"
            color="#07c160"
            :loading="publishing"
            @click="publish"
          >
            发表
          </el-button>
          <input
            ref="fileInput"
            type="file"
            accept="image/*"
            multiple
            style="display: none"
            @change="onFilesChange"
          />
        </div>
      </div>

      <div v-if="loading" class="tip">加载中...</div>
      <div v-else-if="!moments.length" class="tip">还没有动态，发布第一条吧</div>

      <div v-for="m in moments" :key="m.moment_id" class="moment selectable">
        <Avatar :uid="m.uid" :name="nameOf(m.uid)" :size="40" />
        <div class="m-body">
          <div class="m-name">{{ nameOf(m.uid) }}</div>
          <div v-if="m.content" class="m-content">{{ m.content }}</div>
          <div v-if="m.images && m.images.length" class="m-imgs">
            <el-image
              v-for="(id, idx) in m.images"
              :key="id"
              :src="imageUrls[id] || ''"
              :preview-src-list="previewList(m)"
              :initial-index="idx"
              fit="cover"
              class="m-img"
              preview-teleported
              hide-on-click-modal
            />
          </div>
          <div class="m-meta">
            <span class="m-time">{{ timeText(m.created_at) }}</span>
            <div class="m-actions">
              <span class="m-act" @click="toggleLike(m)">
                <el-icon :color="likedByMe(m) ? '#07c160' : ''">
                  <component :is="likedByMe(m) ? StarFilled : Star" />
                </el-icon>
                {{ (m.likes && m.likes.length) || 0 }}
              </span>
              <span class="m-act" @click="commentOpen[m.moment_id] = !commentOpen[m.moment_id]">
                <el-icon><ChatLineSquare /></el-icon>
                {{ (m.comments && m.comments.length) || 0 }}
              </span>
              <span
                v-if="m.uid === user.uid"
                class="m-act m-del"
                @click="removeMoment(m)"
              >
                删除
              </span>
            </div>
          </div>

          <div v-if="m.likes && m.likes.length" class="m-likes">
            <el-icon :size="13"><StarFilled /></el-icon>
            {{ m.likes.map(nameOf).join('，') }}
          </div>

          <div v-if="m.comments && m.comments.length" class="m-comments">
            <div v-for="(cm, i) in m.comments" :key="i" class="m-comment">
              <span class="c-name">{{ nameOf(cm.uid) }}：</span>{{ cm.content }}
            </div>
          </div>

          <div v-if="commentOpen[m.moment_id]" class="m-comment-box">
            <el-input
              v-model="commentDraft[m.moment_id]"
              size="small"
              placeholder="评论"
              @keyup.enter="submitComment(m)"
            />
            <el-button size="small" color="#07c160" type="primary" @click="submitComment(m)">
              发送
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.moments {
  flex: 1;
  display: flex;
  flex-direction: column;
  height: 100%;
  background: var(--wx-bg);
  min-width: 0;
}
.mo-header {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  font-size: 16px;
  font-weight: 500;
  border-bottom: 1px solid var(--wx-border);
  background: var(--wx-header-bg);
}
.mo-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
}
.publish {
  background: #fff;
  border-radius: 8px;
  padding: 12px;
  margin-bottom: 16px;
}
.pub-input {
  width: 100%;
  height: 60px;
  border: none;
  outline: none;
  resize: none;
  font-family: inherit;
  font-size: 14px;
}
.pub-imgs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin: 8px 0;
}
.pub-img {
  position: relative;
  width: 72px;
  height: 72px;
}
.pub-img img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 4px;
}
.pub-img .rm {
  position: absolute;
  top: -6px;
  right: -6px;
  width: 18px;
  height: 18px;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  border-radius: 50%;
  text-align: center;
  line-height: 18px;
  cursor: pointer;
  font-size: 14px;
}
.pub-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  border-top: 1px solid var(--wx-border);
  padding-top: 8px;
}
.tip {
  text-align: center;
  color: var(--wx-text-sub);
  font-size: 13px;
  padding: 30px;
}
.moment {
  display: flex;
  gap: 10px;
  padding: 14px 4px;
  border-bottom: 1px solid var(--wx-border);
}
.m-body {
  flex: 1;
  min-width: 0;
}
.m-name {
  font-size: 14px;
  color: #576b95;
  font-weight: 500;
}
.m-content {
  font-size: 14px;
  margin-top: 4px;
  white-space: pre-wrap;
  word-break: break-word;
}
.m-imgs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
}
.m-img {
  width: 90px;
  height: 90px;
  object-fit: cover;
  border-radius: 4px;
  background: #eee;
}
.m-meta {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
}
.m-time {
  font-size: 12px;
  color: var(--wx-text-sub);
}
.m-actions {
  display: flex;
  gap: 14px;
}
.m-act {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 12px;
  color: var(--wx-text-sub);
  cursor: pointer;
}
.m-del {
  color: #fa5151;
}
.m-likes {
  display: flex;
  align-items: center;
  gap: 4px;
  background: #f7f7f7;
  border-radius: 4px;
  padding: 5px 8px;
  margin-top: 8px;
  font-size: 12px;
  color: #576b95;
}
.m-comments {
  background: #f7f7f7;
  border-radius: 4px;
  padding: 5px 8px;
  margin-top: 6px;
}
.m-comment {
  font-size: 13px;
  line-height: 1.6;
}
.c-name {
  color: #576b95;
}
.m-comment-box {
  display: flex;
  gap: 6px;
  margin-top: 8px;
}
</style>

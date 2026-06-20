<script setup>
import { ref, computed } from 'vue'
import { Search, Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useChatStore } from '../store/chat'
import { formatTime, avatarColor, avatarText } from '../utils/format'

const chat = useChatStore()
const keyword = ref('')

const list = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  const items = chat.sortedConversations
  if (!kw) return items
  return items.filter(
    (c) =>
      (c.name || '').toLowerCase().includes(kw) ||
      (c.uid || '').toLowerCase().includes(kw)
  )
})

const dialogVisible = ref(false)
const newChat = ref({ uid: '', name: '' })

function openNew() {
  newChat.value = { uid: '', name: '' }
  dialogVisible.value = true
}

function confirmNew() {
  const uid = newChat.value.uid.trim()
  if (!uid) {
    ElMessage.warning('请输入对方账号(UID)')
    return
  }
  chat.ensureConversation(uid, newChat.value.name.trim() || uid)
  chat.setActive(uid)
  dialogVisible.value = false
}

function select(uid) {
  chat.setActive(uid)
}
</script>

<template>
  <div class="conv-list">
    <div class="search-bar">
      <el-input
        v-model="keyword"
        placeholder="搜索"
        size="default"
        :prefix-icon="Search"
        clearable
      />
      <el-button class="add-btn" :icon="Plus" circle @click="openNew" title="发起聊天" />
    </div>

    <div class="items">
      <div
        v-for="c in list"
        :key="c.uid"
        class="conv-item"
        :class="{ active: c.uid === chat.activeUid }"
        @click="select(c.uid)"
      >
        <el-badge :value="c.unread" :hidden="!c.unread" class="badge">
          <div class="avatar" :style="{ background: avatarColor(c.uid) }">
            {{ avatarText(c.name) }}
          </div>
        </el-badge>
        <div class="meta">
          <div class="row1">
            <span class="name">{{ c.name }}</span>
            <span class="time">{{ formatTime(c.time) }}</span>
          </div>
          <div class="last">{{ c.last }}</div>
        </div>
      </div>

      <div v-if="!list.length" class="empty">暂无会话，点击右上角发起聊天</div>
    </div>

    <el-dialog v-model="dialogVisible" title="发起聊天" width="320px" align-center>
      <el-form label-position="top">
        <el-form-item label="对方账号 (UID)">
          <el-input v-model="newChat.uid" placeholder="请输入对方 UID" />
        </el-form-item>
        <el-form-item label="备注名 (可选)">
          <el-input v-model="newChat.name" placeholder="备注名" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" color="#07c160" @click="confirmNew">确定</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.conv-list {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.search-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 12px;
  border-bottom: 1px solid var(--wx-border);
}
.add-btn {
  flex-shrink: 0;
}
.items {
  flex: 1;
  overflow-y: auto;
}
.conv-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 14px;
  cursor: pointer;
}
.conv-item:hover {
  background: var(--wx-list-hover);
}
.conv-item.active {
  background: var(--wx-list-active);
}
.avatar {
  width: 40px;
  height: 40px;
  border-radius: 5px;
  color: #fff;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.meta {
  flex: 1;
  min-width: 0;
}
.row1 {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.name {
  font-size: 14px;
  color: var(--wx-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.time {
  font-size: 11px;
  color: var(--wx-text-sub);
  flex-shrink: 0;
  margin-left: 6px;
}
.last {
  font-size: 12px;
  color: var(--wx-text-sub);
  margin-top: 3px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.empty {
  text-align: center;
  color: var(--wx-text-sub);
  font-size: 12px;
  padding: 40px 16px;
}
.badge :deep(.el-badge__content) {
  border: none;
}
</style>

<script setup>
import { ref, computed } from 'vue'
import { Search, Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useChatStore } from '../store/chat'
import { useUserStore } from '../store/user'
import { api } from '../api'
import { formatTime } from '../utils/format'
import Avatar from './Avatar.vue'

const chat = useChatStore()
const user = useUserStore()
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
const mode = ref('create') // 'create' | 'join'
const loading = ref(false)
const form = ref({ uid: '', name: '', groupName: '', desc: '', groupId: '' })

const dialogTitle = computed(() =>
  mode.value === 'create' ? '创建群聊' : '申请加群'
)

function openDialog(m) {
  mode.value = m
  form.value = { uid: '', name: '', groupName: '', desc: '', groupId: '' }
  dialogVisible.value = true
}

async function confirm() {
  if (mode.value === 'create') {
    const name = form.value.groupName.trim()
    if (!name) {
      ElMessage.warning('请输入群名称')
      return
    }
    loading.value = true
    try {
      const g = await api.groupCreate(user.token, name, form.value.desc.trim())
      chat.ensureConversation(g.group_id, g.name, true)
      chat.setActive(g.group_id)
      ElMessage.success('群创建成功，群号：' + g.group_id)
      dialogVisible.value = false
    } catch (e) {
      ElMessage.error(String(e?.message || e))
    } finally {
      loading.value = false
    }
    return
  }

  // join request（需群主审批）
  const gid = form.value.groupId.trim()
  if (!gid) {
    ElMessage.warning('请输入群号')
    return
  }
  loading.value = true
  try {
    await api.groupJoin(user.token, gid)
    ElMessage.success('申请已发送，等待群主同意')
    dialogVisible.value = false
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  } finally {
    loading.value = false
  }
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
      <el-dropdown trigger="click" @command="openDialog">
        <el-button class="add-btn" :icon="Plus" circle title="新建" />
        <template #dropdown>
          <el-dropdown-menu>
            <el-dropdown-item command="create">创建群聊</el-dropdown-item>
            <el-dropdown-item command="join">申请加群</el-dropdown-item>
          </el-dropdown-menu>
        </template>
      </el-dropdown>
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
          <Avatar :uid="c.uid" :name="c.name" :group="c.isGroup" :size="40" />
        </el-badge>
        <div class="meta">
          <div class="row1">
            <span class="name">
              <span v-if="c.isGroup" class="tag">群</span>{{ c.name }}
            </span>
            <span class="time">{{ formatTime(c.time) }}</span>
          </div>
          <div class="last">{{ c.last }}</div>
        </div>
      </div>

      <div v-if="!list.length" class="empty">暂无会话，点击右上角新建</div>
    </div>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="320px" align-center>
      <el-form label-position="top">
        <template v-if="mode === 'create'">
          <el-form-item label="群名称">
            <el-input v-model="form.groupName" placeholder="请输入群名称" />
          </el-form-item>
          <el-form-item label="群简介 (可选)">
            <el-input v-model="form.desc" placeholder="群简介" />
          </el-form-item>
        </template>

        <template v-else>
          <el-form-item label="群号">
            <el-input v-model="form.groupId" placeholder="请输入群号" />
          </el-form-item>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" color="#07c160" :loading="loading" @click="confirm">
          确定
        </el-button>
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
.tag {
  display: inline-block;
  font-size: 10px;
  color: #fff;
  background: var(--wx-green);
  border-radius: 2px;
  padding: 0 3px;
  margin-right: 4px;
  vertical-align: middle;
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

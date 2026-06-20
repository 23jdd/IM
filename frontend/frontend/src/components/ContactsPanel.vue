<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Bell } from '@element-plus/icons-vue'
import { useChatStore } from '../store/chat'
import { useUserStore } from '../store/user'
import { api } from '../api'
import Avatar from './Avatar.vue'

const chat = useChatStore()
const user = useUserStore()
const emit = defineEmits(['open-chat'])

const groups = computed(() =>
  chat.conversations
    .filter((c) => c.isGroup)
    .sort((a, b) => (a.name || '').localeCompare(b.name || ''))
)

const contacts = computed(() =>
  [...chat.friends].sort((a, b) => (a.name || '').localeCompare(b.name || ''))
)

const requests = computed(() => chat.friendRequests)

async function loadRequests() {
  try {
    chat.setFriendRequests((await api.friendRequests(user.token)) || [])
  } catch {
    /* ignore */
  }
}

async function refreshFriends() {
  try {
    const friends = await api.getFriends(user.token)
    chat.setFriends(friends || [])
  } catch {
    /* ignore */
  }
}

// 添加好友
const addVisible = ref(false)
const addForm = ref({ uid: '', remark: '' })
const adding = ref(false)

function openAdd() {
  addForm.value = { uid: '', remark: '' }
  addVisible.value = true
}

async function submitAdd() {
  const uid = addForm.value.uid.trim()
  if (!uid) {
    ElMessage.warning('请输入对方账号(UID)')
    return
  }
  adding.value = true
  try {
    await api.friendRequest(user.token, uid, addForm.value.remark.trim())
    ElMessage.success('好友申请已发送')
    addVisible.value = false
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  } finally {
    adding.value = false
  }
}

// 新的朋友（申请列表）
const reqVisible = ref(false)

async function openRequests() {
  await loadRequests()
  reqVisible.value = true
}

async function accept(req) {
  try {
    await api.friendAccept(user.token, req.uid)
    chat.setFriendRequests(chat.friendRequests.filter((r) => r.uid !== req.uid))
    await refreshFriends()
    ElMessage.success('已添加为好友')
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}

function openFriend(c) {
  chat.ensureConversation(c.uid, c.name)
  emit('open-chat', c.uid)
}

function openGroup(g) {
  emit('open-chat', g.uid)
}

onMounted(loadRequests)
</script>

<template>
  <div class="contacts">
    <div class="head">
      <span>通讯录</span>
      <div class="head-actions">
        <el-badge :value="requests.length" :hidden="!requests.length" :max="99">
          <el-button :icon="Bell" circle size="small" title="新的朋友" @click="openRequests" />
        </el-badge>
        <el-button :icon="Plus" circle size="small" title="添加好友" @click="openAdd" />
      </div>
    </div>

    <div class="list">
      <div class="section-title">我的群聊 ({{ groups.length }})</div>
      <div v-for="g in groups" :key="g.uid" class="item" @click="openGroup(g)">
        <Avatar :uid="g.uid" :name="g.name" :group="true" :size="38" />
        <div class="info">
          <div class="name">{{ g.name }}</div>
          <div class="uid">群号: {{ g.uid }}</div>
        </div>
      </div>
      <div v-if="!groups.length" class="empty-line">暂无群聊</div>

      <div class="section-title">好友 ({{ contacts.length }})</div>
      <div v-for="c in contacts" :key="c.uid" class="item" @click="openFriend(c)">
        <Avatar :uid="c.uid" :name="c.name" :size="38" />
        <div class="info">
          <div class="name">{{ c.name }}</div>
          <div class="uid">UID: {{ c.uid }}</div>
        </div>
      </div>
      <div v-if="!contacts.length" class="empty-line">暂无好友</div>
    </div>

    <el-dialog v-model="addVisible" title="添加好友" width="320px" align-center>
      <el-form label-position="top">
        <el-form-item label="对方账号 (UID)">
          <el-input v-model="addForm.uid" placeholder="请输入对方 UID" />
        </el-form-item>
        <el-form-item label="备注 (可选)">
          <el-input v-model="addForm.remark" placeholder="验证信息/备注" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="addVisible = false">取消</el-button>
        <el-button type="primary" color="#07c160" :loading="adding" @click="submitAdd">
          发送申请
        </el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="reqVisible" title="新的朋友" width="340px" align-center>
      <div v-for="req in requests" :key="req.uid" class="req-row">
        <Avatar :uid="req.uid" :name="req.uid" :size="36" />
        <div class="req-info">
          <div class="req-name">{{ req.uid }}</div>
          <div class="req-remark">{{ req.remark || '请求添加你为好友' }}</div>
        </div>
        <el-button size="small" type="primary" color="#07c160" @click="accept(req)">
          接受
        </el-button>
      </div>
      <div v-if="!requests.length" class="empty-line">暂无新的好友申请</div>
    </el-dialog>
  </div>
</template>

<style scoped>
.contacts {
  display: flex;
  flex-direction: column;
  height: 100%;
}
.head {
  height: 49px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 16px;
  font-size: 15px;
  font-weight: 500;
  border-bottom: 1px solid var(--wx-border);
}
.head-actions {
  display: flex;
  align-items: center;
  gap: 10px;
}
.list {
  flex: 1;
  overflow-y: auto;
}
.section-title {
  font-size: 12px;
  color: var(--wx-text-sub);
  padding: 10px 14px 4px;
  background: var(--wx-list-bg);
}
.item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 11px 14px;
  cursor: pointer;
}
.item:hover {
  background: var(--wx-list-hover);
}
.name {
  font-size: 14px;
}
.uid {
  font-size: 11px;
  color: var(--wx-text-sub);
  margin-top: 2px;
}
.empty-line {
  color: var(--wx-text-sub);
  font-size: 12px;
  padding: 8px 16px;
}
.req-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 2px;
}
.req-info {
  flex: 1;
  min-width: 0;
}
.req-name {
  font-size: 14px;
}
.req-remark {
  font-size: 12px;
  color: var(--wx-text-sub);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>

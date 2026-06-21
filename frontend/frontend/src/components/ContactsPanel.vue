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

// 添加好友（搜索 uid 查到用户再添加）
const addVisible = ref(false)
const searchUid = ref('')
const searched = ref(null) // {uid, name, avatar}
const searching = ref(false)
const addRemark = ref('')
const adding = ref(false)

function openAdd() {
  searchUid.value = ''
  searched.value = null
  addRemark.value = ''
  addVisible.value = true
}

async function doSearch() {
  const uid = searchUid.value.trim()
  if (!uid) {
    ElMessage.warning('请输入对方账号(UID)')
    return
  }
  searching.value = true
  searched.value = null
  try {
    searched.value = await api.userInfo(user.token, uid)
  } catch (e) {
    ElMessage.warning('未找到该用户')
  } finally {
    searching.value = false
  }
}

async function submitAdd() {
  if (!searched.value) return
  if (searched.value.uid === user.uid) {
    ElMessage.warning('不能添加自己为好友')
    return
  }
  adding.value = true
  try {
    await api.friendRequest(user.token, searched.value.uid, addRemark.value.trim())
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
        <Avatar
          :uid="c.uid"
          :name="c.name"
          :size="38"
          style="cursor: pointer"
          @click.stop="chat.viewUser(c.uid)"
        />
        <div class="info">
          <div class="name">{{ c.name }}</div>
        </div>
      </div>
      <div v-if="!contacts.length" class="empty-line">暂无好友</div>
    </div>

    <el-dialog v-model="addVisible" title="添加好友" width="340px" align-center>
      <div class="search-row">
        <el-input
          v-model="searchUid"
          placeholder="输入对方账号(UID)搜索"
          @keyup.enter="doSearch"
        />
        <el-button :loading="searching" @click="doSearch">搜索</el-button>
      </div>

      <div v-if="searched" class="search-result">
        <Avatar :uid="searched.uid" :name="searched.name" :size="44" />
        <div class="sr-info">
          <div class="sr-name">{{ searched.name || searched.uid }}</div>
          <div class="sr-uid">UID: {{ searched.uid }}</div>
        </div>
      </div>
      <el-input
        v-if="searched"
        v-model="addRemark"
        placeholder="备注 (可选)"
        size="small"
        style="margin-top: 10px"
      />

      <template #footer>
        <el-button @click="addVisible = false">取消</el-button>
        <el-button
          type="primary"
          color="#07c160"
          :loading="adding"
          :disabled="!searched"
          @click="submitAdd"
        >
          添加好友
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
.search-row {
  display: flex;
  gap: 8px;
}
.search-result {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 14px;
  padding: 10px;
  background: var(--wx-list-bg);
  border-radius: 6px;
}
.sr-info {
  flex: 1;
  min-width: 0;
}
.sr-name {
  font-size: 15px;
}
.sr-uid {
  font-size: 12px;
  color: var(--wx-text-sub);
  margin-top: 2px;
}
</style>

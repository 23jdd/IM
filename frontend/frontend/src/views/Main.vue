<script setup>
import { ref, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, ElMessageBox, ElNotification } from 'element-plus'
import SideNav from '../components/SideNav.vue'
import ConversationList from '../components/ConversationList.vue'
import ContactsPanel from '../components/ContactsPanel.vue'
import ChatPanel from '../components/ChatPanel.vue'
import ProfileDialog from '../components/ProfileDialog.vue'
import Moments from '../components/Moments.vue'
import UserCard from '../components/UserCard.vue'
import { api, onEvent, EVT } from '../api'
import { useUserStore } from '../store/user'
import { useChatStore } from '../store/chat'

const router = useRouter()
const user = useUserStore()
const chat = useChatStore()

const activeView = ref('chats')
const profileVisible = ref(false)

let unsubs = []

function bindEvents() {
  unsubs.push(
    onEvent(EVT.STATUS, (data) => {
      const ok = data === 'connected'
      chat.setConnected(ok)
      if (typeof data === 'string' && data.startsWith('error:')) {
        ElMessage.error('连接错误：' + data.slice(6))
      }
    })
  )
  unsubs.push(onEvent(EVT.TEXT, (d) => chat.receiveText(d || {})))
  unsubs.push(onEvent(EVT.OFFLINE, (d) => chat.receiveOffline(d || {})))
  unsubs.push(onEvent(EVT.ACK, (d) => chat.markStatus(Number(d?.key), 'sent')))
  unsubs.push(onEvent(EVT.NACK, (d) => chat.markStatus(Number(d?.key), 'failed')))
  unsubs.push(onEvent(EVT.NOTIFY, (d) => handleNotify(d)))
}

async function handleNotify(d) {
  if (!d || !d.event) return
  if (d.event === 'friend_request') {
    try {
      const reqs = await api.friendRequests(user.token)
      chat.setFriendRequests(reqs || [])
    } catch (e) {
      /* ignore */
    }
    ElNotification({
      title: '新的朋友',
      message: `${d.from_uid || ''} 请求添加你为好友`,
      type: 'info',
    })
  } else if (d.event === 'friend_accepted') {
    try {
      const friends = await api.getFriends(user.token)
      chat.setFriends(friends || [])
    } catch (e) {
      /* ignore */
    }
    ElNotification({
      title: '好友',
      message: `${d.from_uid || ''} 接受了你的好友申请`,
      type: 'success',
    })
  } else if (d.event === 'group_invite') {
    try {
      const groups = await api.groupList(user.token)
      chat.loadGroups(groups || [])
    } catch (e) {
      /* ignore */
    }
    ElNotification({
      title: '群聊邀请',
      message: `${d.from_uid || ''} 邀请你加入了群聊`,
      type: 'info',
    })
  } else if (d.event === 'mention') {
    ElNotification({
      title: '有人@你',
      message: `${d.from_uid || ''} 在群聊里@了你`,
      type: 'warning',
    })
  } else if (d.event === 'group_join_request') {
    ElNotification({
      title: '入群申请',
      message: `${d.from_uid || ''} 申请加入你的群聊，请在群成员里审批`,
      type: 'info',
    })
  } else if (d.event === 'group_join_approved') {
    try {
      const groups = await api.groupList(user.token)
      chat.loadGroups(groups || [])
    } catch (e) {
      /* ignore */
    }
    ElNotification({
      title: '加群成功',
      message: '群主已通过你的入群申请',
      type: 'success',
    })
  } else if (d.event === 'group_join_rejected') {
    ElNotification({
      title: '加群被拒绝',
      message: '群主拒绝了你的入群申请',
      type: 'warning',
    })
  }
}

async function connectFlow() {
  try {
    await api.connect()
    await api.authTcp(user.token)
    // TCP 有序：认证帧先于同步帧被服务端处理，稍后拉取离线消息。
    setTimeout(() => api.sync().catch(() => {}), 300)
  } catch (e) {
    ElMessage.error('连接服务器失败：' + String(e?.message || e))
  }
}

async function loadInitialData() {
  try {
    const profile = await api.getProfile(user.token)
    if (profile) {
      user.setProfile(profile)
      if (profile.avatar) {
        const url = await api.getAvatar(user.token, profile.avatar)
        user.setAvatarUrl(url)
        chat.setAvatarCache(user.uid, url || '')
      }
    }
  } catch (e) {
    /* 资料/头像加载失败不阻断 */
  }
  try {
    const friends = await api.getFriends(user.token)
    chat.setFriends(friends || [])
  } catch (e) {
    /* 好友接口失败不阻断 */
  }
  try {
    const groups = await api.groupList(user.token)
    chat.loadGroups(groups || [])
  } catch (e) {
    /* 群列表失败不阻断 */
  }
  try {
    const convs = await api.getConversations(user.token)
    chat.loadConversations(convs || [])
  } catch (e) {
    /* 会话接口失败不阻断 */
  }
}

function onChangeView(v) {
  activeView.value = v
}

function onOpenChat(payload) {
  const uid = typeof payload === 'string' ? payload : payload && payload.uid
  const name = typeof payload === 'string' ? undefined : payload && payload.name
  if (!uid) return
  chat.ensureConversation(uid, name)
  chat.setActive(uid)
  activeView.value = 'chats'
}

async function onLogout() {
  try {
    await ElMessageBox.confirm('确定要退出登录吗？', '提示', {
      confirmButtonText: '退出',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return
  }
  cleanup()
  api.disconnect()
  chat.reset()
  user.logout()
  router.replace('/login')
}

function cleanup() {
  unsubs.forEach((u) => {
    try {
      u && u()
    } catch {
      /* ignore */
    }
  })
  unsubs = []
}

onMounted(async () => {
  chat.init(user.uid)
  bindEvents()
  try {
    await api.localInit(user.uid)
  } catch (e) {
    /* 本地库不可用不阻断 */
  }
  connectFlow()
  loadInitialData()
})

onUnmounted(() => {
  cleanup()
})
</script>

<template>
  <div class="wx-shell">
    <SideNav
      :active-view="activeView"
      :connected="chat.connected"
      :name="user.name"
      :uid="user.uid"
      :avatar-url="user.avatarUrl"
      @change-view="onChangeView"
      @open-profile="profileVisible = true"
      @logout="onLogout"
    />

    <div class="list-col" v-show="activeView !== 'moments'">
      <ConversationList v-show="activeView === 'chats'" />
      <ContactsPanel v-show="activeView === 'contacts'" @open-chat="onOpenChat" />
    </div>

    <ChatPanel class="main-col" v-show="activeView !== 'moments'" />
    <Moments v-if="activeView === 'moments'" class="main-col" />

    <ProfileDialog v-model="profileVisible" />
    <UserCard
      :model-value="chat.userCardVisible"
      :uid="chat.userCardUid"
      @update:model-value="chat.userCardVisible = $event"
      @open-chat="onOpenChat"
    />
  </div>
</template>

<style scoped>
.wx-shell {
  display: flex;
  height: 100%;
  background: var(--wx-bg);
}
.list-col {
  width: 280px;
  flex-shrink: 0;
  background: var(--wx-list-bg);
  border-right: 1px solid var(--wx-border);
  height: 100%;
  overflow: hidden;
}
.list-col > * {
  height: 100%;
}
.main-col {
  flex: 1;
  min-width: 0;
}

@media (max-width: 820px) {
  .list-col {
    width: 230px;
  }
}
@media (max-width: 640px) {
  .list-col {
    width: 180px;
  }
}
</style>

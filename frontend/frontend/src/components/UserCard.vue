<script setup>
import { ref, computed, watch } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { api } from '../api'
import { useUserStore } from '../store/user'
import { useChatStore } from '../store/chat'
import Avatar from './Avatar.vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  uid: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue', 'open-chat'])

const user = useUserStore()
const chat = useChatStore()
const info = ref(null)
const loading = ref(false)

const genderText = { 0: '', 1: '男', 2: '女' }

const isFriend = computed(
  () => info.value && chat.friends.some((f) => f.uid === info.value.uid)
)

watch(
  () => props.modelValue,
  async (v) => {
    if (!v || !props.uid) return
    loading.value = true
    info.value = null
    try {
      info.value = await api.userInfo(user.token, props.uid)
    } catch (e) {
      info.value = null
    } finally {
      loading.value = false
    }
  }
)

function close() {
  emit('update:modelValue', false)
}

function startChat() {
  if (!info.value) return
  if (info.value.uid === user.uid) {
    ElMessage.warning('不能和自己聊天')
    return
  }
  emit('open-chat', { uid: info.value.uid, name: info.value.name })
  close()
}

async function removeFriend() {
  if (!info.value) return
  try {
    await ElMessageBox.confirm('确定删除该好友吗？', '提示', {
      confirmButtonText: '删除',
      cancelButtonText: '取消',
      type: 'warning',
    })
  } catch {
    return
  }
  try {
    await api.friendRemove(user.token, info.value.uid)
    const friends = await api.getFriends(user.token)
    chat.setFriends(friends || [])
    ElMessage.success('已删除好友')
    close()
  } catch (e) {
    ElMessage.error(String(e?.message || e))
  }
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    title="个人信息"
    width="300px"
    align-center
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div v-if="info" class="uc">
      <Avatar :uid="info.uid" :name="info.name" :size="64" />
      <div class="uc-name">{{ info.name || info.uid }}</div>
      <div class="uc-uid">UID: {{ info.uid }}</div>
      <div v-if="genderText[info.gender]" class="uc-row">
        性别：{{ genderText[info.gender] }}
      </div>
      <div v-if="info.signature" class="uc-sig">{{ info.signature }}</div>
      <el-button
        class="uc-btn"
        type="primary"
        color="#07c160"
        :disabled="info.uid === user.uid"
        @click="startChat"
      >
        发消息
      </el-button>
      <el-button
        v-if="isFriend"
        class="uc-btn"
        type="danger"
        plain
        @click="removeFriend"
      >
        删除好友
      </el-button>
    </div>
    <div v-else-if="loading" class="uc-tip">加载中...</div>
    <div v-else class="uc-tip">用户不存在</div>
  </el-dialog>
</template>

<style scoped>
.uc {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
}
.uc-name {
  font-size: 17px;
  font-weight: 500;
  margin-top: 4px;
}
.uc-uid {
  font-size: 12px;
  color: var(--wx-text-sub);
}
.uc-row {
  font-size: 13px;
}
.uc-sig {
  font-size: 13px;
  color: var(--wx-text-sub);
  text-align: center;
  max-width: 240px;
}
.uc-btn {
  margin-top: 8px;
  width: 140px;
}
.uc-tip {
  text-align: center;
  color: var(--wx-text-sub);
  padding: 20px;
}
</style>

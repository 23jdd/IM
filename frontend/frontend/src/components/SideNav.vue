<script setup>
import { computed } from 'vue'
import { ChatDotRound, Avatar, SwitchButton, Camera } from '@element-plus/icons-vue'
import { avatarColor, avatarText } from '../utils/format'

const props = defineProps({
  activeView: { type: String, default: 'chats' },
  connected: { type: Boolean, default: false },
  name: { type: String, default: '' },
  uid: { type: String, default: '' },
  avatarUrl: { type: String, default: '' },
})
const emit = defineEmits(['change-view', 'open-profile', 'logout'])

const initial = computed(() => avatarText(props.name))
const bg = computed(() => avatarColor(props.uid || props.name))
</script>

<template>
  <div class="side-nav">
    <div class="nav-top">
      <div
        class="nav-avatar"
        :style="{ background: avatarUrl ? 'transparent' : bg }"
        title="个人资料"
        @click="emit('open-profile')"
      >
        <img v-if="avatarUrl" :src="avatarUrl" class="nav-avatar-img" alt="" />
        <template v-else>{{ initial }}</template>
        <span class="status-dot" :class="{ on: connected }"></span>
      </div>

      <div
        class="nav-item"
        :class="{ active: activeView === 'chats' }"
        title="聊天"
        @click="emit('change-view', 'chats')"
      >
        <el-icon :size="24"><ChatDotRound /></el-icon>
      </div>
      <div
        class="nav-item"
        :class="{ active: activeView === 'contacts' }"
        title="通讯录"
        @click="emit('change-view', 'contacts')"
      >
        <el-icon :size="24"><Avatar /></el-icon>
      </div>
      <div
        class="nav-item"
        :class="{ active: activeView === 'moments' }"
        title="朋友圈"
        @click="emit('change-view', 'moments')"
      >
        <el-icon :size="24"><Camera /></el-icon>
      </div>
    </div>

    <div class="nav-bottom">
      <div class="nav-item" title="退出登录" @click="emit('logout')">
        <el-icon :size="22"><SwitchButton /></el-icon>
      </div>
    </div>
  </div>
</template>

<style scoped>
.side-nav {
  width: 60px;
  background: var(--wx-navbar);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  align-items: center;
  padding: 18px 0 16px;
  flex-shrink: 0;
}
.nav-top {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 18px;
}
.nav-avatar {
  position: relative;
  width: 38px;
  height: 38px;
  border-radius: 6px;
  color: #fff;
  font-size: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  margin-bottom: 6px;
}
.nav-avatar-img {
  width: 100%;
  height: 100%;
  border-radius: 6px;
  object-fit: cover;
}
.status-dot {
  position: absolute;
  right: -2px;
  bottom: -2px;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 2px solid var(--wx-navbar);
  background: #8b8b8b;
}
.status-dot.on {
  background: var(--wx-green);
}
.nav-item {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 6px;
  color: #9a9a9a;
  cursor: pointer;
  transition: color 0.15s, background 0.15s;
}
.nav-item:hover {
  color: #fff;
}
.nav-item.active {
  color: var(--wx-green);
}
</style>

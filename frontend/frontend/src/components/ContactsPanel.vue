<script setup>
import { computed } from 'vue'
import { useChatStore } from '../store/chat'
import { avatarColor, avatarText } from '../utils/format'

const chat = useChatStore()
const emit = defineEmits(['open-chat'])

const contacts = computed(() =>
  [...chat.conversations]
    .filter((c) => c.uid !== '__unknown__')
    .sort((a, b) => (a.name || '').localeCompare(b.name || ''))
)

function open(uid) {
  emit('open-chat', uid)
}
</script>

<template>
  <div class="contacts">
    <div class="head">通讯录</div>
    <div class="list">
      <div v-for="c in contacts" :key="c.uid" class="item" @click="open(c.uid)">
        <div class="avatar" :style="{ background: avatarColor(c.uid) }">
          {{ avatarText(c.name) }}
        </div>
        <div class="info">
          <div class="name">{{ c.name }}</div>
          <div class="uid">UID: {{ c.uid }}</div>
        </div>
      </div>
      <div v-if="!contacts.length" class="empty">
        暂无联系人，去“聊天”发起会话即可添加
      </div>
    </div>
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
  padding: 0 16px;
  font-size: 15px;
  font-weight: 500;
  border-bottom: 1px solid var(--wx-border);
}
.list {
  flex: 1;
  overflow-y: auto;
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
.avatar {
  width: 38px;
  height: 38px;
  border-radius: 5px;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 15px;
}
.name {
  font-size: 14px;
}
.uid {
  font-size: 11px;
  color: var(--wx-text-sub);
  margin-top: 2px;
}
.empty {
  text-align: center;
  color: var(--wx-text-sub);
  font-size: 12px;
  padding: 40px 16px;
}
</style>

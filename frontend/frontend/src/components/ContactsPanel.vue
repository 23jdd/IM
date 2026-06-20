<script setup>
import { computed } from 'vue'
import { useChatStore } from '../store/chat'
import Avatar from './Avatar.vue'

const chat = useChatStore()
const emit = defineEmits(['open-chat'])

const groups = computed(() =>
  chat.conversations
    .filter((c) => c.isGroup)
    .sort((a, b) => (a.name || '').localeCompare(b.name || ''))
)

const contacts = computed(() =>
  [...chat.friends].sort((a, b) => (a.name || '').localeCompare(b.name || ''))
)

function openFriend(c) {
  chat.ensureConversation(c.uid, c.name)
  emit('open-chat', c.uid)
}

function openGroup(g) {
  emit('open-chat', g.uid)
}
</script>

<template>
  <div class="contacts">
    <div class="head">通讯录</div>
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
.empty-line {
  color: var(--wx-text-sub);
  font-size: 12px;
  padding: 8px 16px;
}
</style>

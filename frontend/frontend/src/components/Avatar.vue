<script setup>
import { ref, watch, onMounted } from 'vue'
import { avatarColor, avatarText } from '../utils/format'
import { resolveAvatar } from '../utils/avatar'

const props = defineProps({
  uid: { type: String, default: '' },
  name: { type: String, default: '' },
  size: { type: Number, default: 40 },
  group: { type: Boolean, default: false },
})

const url = ref('')

async function load() {
  url.value = ''
  if (!props.uid || props.group) return
  url.value = await resolveAvatar(props.uid)
}

watch(() => props.uid, load)
onMounted(load)
</script>

<template>
  <div
    class="avatar"
    :style="{
      width: size + 'px',
      height: size + 'px',
      fontSize: size * 0.4 + 'px',
      background: url ? 'transparent' : avatarColor(uid || name),
    }"
  >
    <img v-if="url" :src="url" class="avatar-img" alt="" />
    <template v-else>{{ avatarText(name) }}</template>
  </div>
</template>

<style scoped>
.avatar {
  border-radius: 5px;
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  flex-shrink: 0;
}
.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>

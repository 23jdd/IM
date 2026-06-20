<script setup>
import { ref, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { useUserStore } from '../store/user'
import { avatarText, avatarColor } from '../utils/format'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const user = useUserStore()
const tab = ref('info')
const loading = ref(false)
const uploading = ref(false)
const avatarUrl = ref('')
const fileInput = ref(null)

const form = reactive({
  uid: '',
  name: '',
  avatar: '',
  gender: 0,
  birthday: '',
  signature: '',
  email: '',
  phone: '',
  status: 0,
})

const pwd = reactive({ old: '', neo: '', confirm: '' })

watch(
  () => props.modelValue,
  async (v) => {
    if (!v) return
    tab.value = 'info'
    pwd.old = pwd.neo = pwd.confirm = ''
    try {
      const p = await api.getProfile(user.token)
      Object.assign(form, p)
      avatarUrl.value = user.avatarUrl || ''
      if (!avatarUrl.value && form.avatar) {
        try {
          avatarUrl.value = await api.getAvatar(user.token, form.avatar)
        } catch (e) {
          /* ignore */
        }
      }
    } catch (e) {
      ElMessage.error('加载资料失败：' + String(e?.message || e))
    }
  }
)

function close() {
  emit('update:modelValue', false)
}

function onPickFile() {
  if (fileInput.value) fileInput.value.click()
}

function readAsDataURL(file) {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => resolve(reader.result)
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

async function onFileChange(e) {
  const file = e.target.files && e.target.files[0]
  if (!file) return
  if (!file.type.startsWith('image/')) {
    ElMessage.warning('请选择图片文件')
    return
  }
  if (file.size > 2 * 1024 * 1024) {
    ElMessage.warning('图片不能超过 2MB')
    return
  }
  uploading.value = true
  try {
    const dataUrl = await readAsDataURL(file)
    const base64 = dataUrl.split(',')[1] || ''
    const id = await api.uploadAvatar(user.token, base64, file.type)
    form.avatar = id
    avatarUrl.value = dataUrl
    user.setAvatarUrl(dataUrl)
    user.setProfile({ ...form })
    ElMessage.success('头像已更新')
  } catch (err) {
    ElMessage.error('上传失败：' + String(err?.message || err))
  } finally {
    uploading.value = false
    if (fileInput.value) fileInput.value.value = ''
  }
}

async function saveProfile() {
  loading.value = true
  try {
    await api.updateProfile(user.token, {
      uid: form.uid,
      name: form.name,
      avatar: form.avatar,
      gender: Number(form.gender),
      birthday: form.birthday || '',
      signature: form.signature || '',
      email: form.email || '',
      phone: form.phone || '',
      status: Number(form.status),
    })
    user.setProfile({ ...form })
    ElMessage.success('资料已更新')
    close()
  } catch (e) {
    ElMessage.error('保存失败：' + String(e?.message || e))
  } finally {
    loading.value = false
  }
}

async function savePassword() {
  if (!pwd.old || !pwd.neo) {
    ElMessage.warning('请输入密码')
    return
  }
  if (pwd.neo.length < 6) {
    ElMessage.warning('新密码至少 6 位')
    return
  }
  if (pwd.neo !== pwd.confirm) {
    ElMessage.warning('两次新密码不一致')
    return
  }
  loading.value = true
  try {
    await api.changePassword(user.token, pwd.old, pwd.neo)
    ElMessage.success('密码已修改')
    close()
  } catch (e) {
    ElMessage.error('修改失败：' + String(e?.message || e))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    title="个人资料"
    width="420px"
    align-center
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-tabs v-model="tab">
      <el-tab-pane label="资料" name="info">
        <div class="avatar-box">
          <div
            class="avatar-preview"
            :style="{ background: avatarUrl ? 'transparent' : avatarColor(form.uid) }"
          >
            <img v-if="avatarUrl" :src="avatarUrl" class="avatar-img" alt="" />
            <template v-else>{{ avatarText(form.name) }}</template>
          </div>
          <el-button size="small" :loading="uploading" @click="onPickFile">更换头像</el-button>
          <input
            ref="fileInput"
            type="file"
            accept="image/*"
            style="display: none"
            @change="onFileChange"
          />
        </div>
        <el-form label-width="72px">
          <el-form-item label="账号">
            <el-input :model-value="form.uid" disabled />
          </el-form-item>
          <el-form-item label="昵称">
            <el-input v-model="form.name" />
          </el-form-item>
          <el-form-item label="性别">
            <el-radio-group v-model="form.gender">
              <el-radio :value="0">未知</el-radio>
              <el-radio :value="1">男</el-radio>
              <el-radio :value="2">女</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item label="签名">
            <el-input v-model="form.signature" type="textarea" :rows="2" />
          </el-form-item>
          <el-form-item label="邮箱">
            <el-input v-model="form.email" />
          </el-form-item>
          <el-form-item label="手机号">
            <el-input v-model="form.phone" />
          </el-form-item>
        </el-form>
        <div class="actions">
          <el-button @click="close">取消</el-button>
          <el-button type="primary" color="#07c160" :loading="loading" @click="saveProfile">
            保存
          </el-button>
        </div>
      </el-tab-pane>

      <el-tab-pane label="修改密码" name="pwd">
        <el-form label-width="88px">
          <el-form-item label="原密码">
            <el-input v-model="pwd.old" type="password" show-password />
          </el-form-item>
          <el-form-item label="新密码">
            <el-input v-model="pwd.neo" type="password" show-password />
          </el-form-item>
          <el-form-item label="确认新密码">
            <el-input v-model="pwd.confirm" type="password" show-password />
          </el-form-item>
        </el-form>
        <div class="actions">
          <el-button @click="close">取消</el-button>
          <el-button type="primary" color="#07c160" :loading="loading" @click="savePassword">
            确定
          </el-button>
        </div>
      </el-tab-pane>
    </el-tabs>
  </el-dialog>
</template>

<style scoped>
.actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 8px;
}
.avatar-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  margin-bottom: 14px;
}
.avatar-preview {
  width: 72px;
  height: 72px;
  border-radius: 8px;
  color: #fff;
  font-size: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
}
.avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>

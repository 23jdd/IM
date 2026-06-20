<script setup>
import { reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { ChatDotRound } from '@element-plus/icons-vue'
import { api } from '../api'
import { useUserStore } from '../store/user'

const router = useRouter()
const userStore = useUserStore()

const tab = ref('login')
const loading = ref(false)

const loginForm = reactive({ uid: '', password: '' })
const registerForm = reactive({
  name: '',
  password: '',
  confirm: '',
  email: '',
  phone: '',
})

async function onLogin() {
  if (!loginForm.uid || !loginForm.password) {
    ElMessage.warning('请输入账号和密码')
    return
  }
  loading.value = true
  try {
    const res = await api.login(loginForm.uid, loginForm.password)
    userStore.setLogin({ token: res.token, uid: res.uid, name: res.name })
    router.replace('/')
  } catch (e) {
    ElMessage.error(String(e?.message || e || '登录失败'))
  } finally {
    loading.value = false
  }
}

async function onRegister() {
  if (!registerForm.name || !registerForm.password) {
    ElMessage.warning('请输入昵称和密码')
    return
  }
  if (registerForm.password.length < 6) {
    ElMessage.warning('密码至少 6 位')
    return
  }
  if (registerForm.password !== registerForm.confirm) {
    ElMessage.warning('两次密码不一致')
    return
  }
  loading.value = true
  try {
    const res = await api.register(
      registerForm.name,
      registerForm.password,
      registerForm.email,
      registerForm.phone
    )
    ElMessage.success(`注册成功，您的账号(UID)：${res.uid}`)
    loginForm.uid = res.uid
    tab.value = 'login'
  } catch (e) {
    ElMessage.error(String(e?.message || e || '注册失败'))
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-bg">
    <div class="login-card">
      <div class="brand">
        <div class="brand-logo">
          <el-icon :size="30"><ChatDotRound /></el-icon>
        </div>
        <div class="brand-title">WeChatIM</div>
        <div class="brand-sub">连接你我</div>
      </div>

      <el-tabs v-model="tab" stretch class="login-tabs">
        <el-tab-pane label="登录" name="login">
          <el-form label-position="top" @submit.prevent>
            <el-form-item label="账号 (UID/手机/邮箱)">
              <el-input
                v-model="loginForm.uid"
                placeholder="请输入账号"
                size="large"
                clearable
              />
            </el-form-item>
            <el-form-item label="密码">
              <el-input
                v-model="loginForm.password"
                type="password"
                placeholder="请输入密码"
                size="large"
                show-password
                @keyup.enter="onLogin"
              />
            </el-form-item>
            <el-button
              class="submit-btn"
              type="primary"
              size="large"
              :loading="loading"
              @click="onLogin"
            >
              登录
            </el-button>
          </el-form>
        </el-tab-pane>

        <el-tab-pane label="注册" name="register">
          <el-form label-position="top" @submit.prevent>
            <el-form-item label="昵称">
              <el-input v-model="registerForm.name" placeholder="请输入昵称" size="large" clearable />
            </el-form-item>
            <el-form-item label="密码">
              <el-input
                v-model="registerForm.password"
                type="password"
                placeholder="至少 6 位"
                size="large"
                show-password
              />
            </el-form-item>
            <el-form-item label="确认密码">
              <el-input
                v-model="registerForm.confirm"
                type="password"
                placeholder="再次输入密码"
                size="large"
                show-password
              />
            </el-form-item>
            <el-form-item label="邮箱 (可选)">
              <el-input v-model="registerForm.email" placeholder="邮箱" size="large" clearable />
            </el-form-item>
            <el-form-item label="手机号 (可选)">
              <el-input v-model="registerForm.phone" placeholder="手机号" size="large" clearable />
            </el-form-item>
            <el-button
              class="submit-btn"
              type="primary"
              size="large"
              :loading="loading"
              @click="onRegister"
            >
              注册
            </el-button>
          </el-form>
        </el-tab-pane>
      </el-tabs>
    </div>
  </div>
</template>

<style scoped>
.login-bg {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #e8f7ee 0%, #f5f5f5 60%);
}

.login-card {
  width: 360px;
  background: #fff;
  border-radius: 12px;
  padding: 32px 28px 28px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.08);
}

.brand {
  text-align: center;
  margin-bottom: 8px;
}
.brand-logo {
  width: 60px;
  height: 60px;
  margin: 0 auto 10px;
  border-radius: 16px;
  background: var(--wx-green);
  color: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}
.brand-title {
  font-size: 20px;
  font-weight: 600;
  color: var(--wx-text);
}
.brand-sub {
  font-size: 12px;
  color: var(--wx-text-sub);
  margin-top: 2px;
}

.login-tabs {
  margin-top: 8px;
}

.submit-btn {
  width: 100%;
  margin-top: 6px;
  background: var(--wx-green);
  border-color: var(--wx-green);
}
.submit-btn:hover {
  background: var(--wx-green-hover);
  border-color: var(--wx-green-hover);
}

:deep(.el-tabs__active-bar) {
  background-color: var(--wx-green);
}
:deep(.el-tabs__item.is-active) {
  color: var(--wx-green);
}
:deep(.el-tabs__item:hover) {
  color: var(--wx-green-hover);
}
</style>

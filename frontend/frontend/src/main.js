import { createApp } from 'vue'
import { createPinia } from 'pinia'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'

import App from './App.vue'
import router from './router'
import { api } from './api'
import { useUserStore } from './store/user'

async function bootstrap() {
  const app = createApp(App)
  const pinia = createPinia()
  app.use(pinia)

  // 从本地 SQLite 恢复登录态（替代 localStorage），需在路由挂载前完成
  try {
    const s = await api.loadSession()
    useUserStore(pinia).restore(s)
  } catch (e) {
    /* 无本地登录态或读取失败，进入登录页 */
  }

  for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
    app.component(key, component)
  }
  app.use(router)
  app.use(ElementPlus)
  app.mount('#app')
}

bootstrap()

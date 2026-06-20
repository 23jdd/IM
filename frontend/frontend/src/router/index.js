import { createRouter, createWebHashHistory } from 'vue-router'
import { useUserStore } from '../store/user'

const routes = [
  { path: '/login', name: 'login', component: () => import('../views/Login.vue') },
  { path: '/', name: 'main', component: () => import('../views/Main.vue') },
]

const router = createRouter({
  history: createWebHashHistory(),
  routes,
})

router.beforeEach((to) => {
  const user = useUserStore()
  if (to.name !== 'login' && !user.token) {
    return { name: 'login' }
  }
  if (to.name === 'login' && user.token) {
    return { name: 'main' }
  }
  return true
})

export default router

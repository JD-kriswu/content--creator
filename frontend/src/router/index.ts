import { createRouter, createWebHistory } from 'vue-router'
import Login from '@/views/Login.vue'
import Home from '@/views/Home.vue'

const router = createRouter({
  history: createWebHistory('/creator/'),
  routes: [
    { path: '/login', component: Login },
    { path: '/', component: Home, meta: { requiresAuth: true } },
    { path: '/:pathMatch(.*)*', redirect: '/' }
  ]
})

router.beforeEach(to => {
  const token = localStorage.getItem('token')
  if (to.meta.requiresAuth && !token) return '/login'
  if (to.path === '/login' && token) return '/'
  return true
})

export default router

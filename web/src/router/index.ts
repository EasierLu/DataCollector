import { createRouter, createWebHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import { useAuthStore } from '@/stores/auth'

const AdminLayout = () => import('@/layouts/AdminLayout.vue')

const routes: RouteRecordRaw[] = [
  {
    path: '/login',
    name: 'Login',
    component: () => import('@/views/LoginView.vue'),
    meta: { guest: true },
  },
  {
    path: '/setup',
    name: 'Setup',
    component: () => import('@/views/SetupView.vue'),
    meta: { guest: true },
  },
  {
    path: '/',
    component: AdminLayout,
    meta: { requiresAuth: true },
    children: [
      { path: '', redirect: '/dashboard' },
      { path: 'dashboard', name: 'Dashboard', component: () => import('@/views/DashboardView.vue') },
      { path: 'sources', name: 'Sources', component: () => import('@/views/SourcesView.vue') },
      { path: 'sources/:id', name: 'SourceDetail', component: () => import('@/views/SourceDetailView.vue') },
      { path: 'data', name: 'Data', component: () => import('@/views/DataView.vue') },
      { path: 'settings', name: 'Settings', component: () => import('@/views/SettingsView.vue') },
      { path: 'api-docs', name: 'ApiDocs', component: () => import('@/views/ApiDocsView.vue') },
    ],
  },
  {
    path: '/:pathMatch(.*)*',
    name: 'NotFound',
    component: () => import('@/views/NotFoundView.vue'),
  },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach(async (to) => {
  const authStore = useAuthStore()

  if (to.meta.guest) {
    // 已登录用户访问 guest 页面（login/setup）时重定向到仪表板
    if (authStore.token && !authStore.isTokenExpired()) {
      return { name: 'Dashboard' }
    }
    return true
  }

  if (to.meta.requiresAuth || to.matched.some(r => r.meta.requiresAuth)) {
    if (!authStore.token || authStore.isTokenExpired()) {
      authStore.clearToken()
      return { name: 'Login', query: { redirect: to.fullPath } }
    }
  }

  return true
})

export default router

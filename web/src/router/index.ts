import { createRouter, createWebHistory } from 'vue-router'
import AdminLayout from '@/layouts/AdminLayout.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/login',
      name: 'Login',
      component: () => import('@/views/LoginView.vue'),
    },
    {
      path: '/setup',
      name: 'Setup',
      component: () => import('@/views/SetupView.vue'),
    },
    {
      path: '/',
      component: AdminLayout,
      meta: { requiresAuth: true },
      children: [
        {
          path: '',
          redirect: '/dashboard',
        },
        {
          path: 'dashboard',
          name: 'Dashboard',
          component: () => import('@/views/DashboardView.vue'),
        },
        {
          path: 'sources',
          name: 'Sources',
          component: () => import('@/views/SourcesView.vue'),
        },
        {
          path: 'sources/:id',
          name: 'SourceDetail',
          component: () => import('@/views/SourceDetailView.vue'),
        },
        {
          path: 'data',
          name: 'Data',
          component: () => import('@/views/DataView.vue'),
        },
        {
          path: 'settings',
          name: 'Settings',
          component: () => import('@/views/SettingsView.vue'),
        },
        {
          path: 'api-docs',
          name: 'ApiDocs',
          component: () => import('@/views/ApiDocsView.vue'),
        },
      ],
    },
    {
      path: '/:pathMatch(.*)*',
      redirect: '/dashboard',
    },
  ],
})

router.beforeEach((to) => {
  const token = localStorage.getItem('jwt_token')

  if (to.path === '/login' && token) {
    return '/dashboard'
  }

  if (to.matched.some((r) => r.meta.requiresAuth) && !token) {
    return '/login'
  }
})

export default router

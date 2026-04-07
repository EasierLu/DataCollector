import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi } from '@/api/auth'
import type { LoginRequest } from '@/api/auth'
import router from '@/router'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('jwt_token'))

  const isLoggedIn = computed(() => !!token.value)

  async function login(data: LoginRequest) {
    const res = await loginApi(data)
    token.value = res.token
    localStorage.setItem('jwt_token', res.token)
  }

  function logout() {
    token.value = null
    localStorage.removeItem('jwt_token')
    router.push('/login')
  }

  return { token, isLoggedIn, login, logout }
})

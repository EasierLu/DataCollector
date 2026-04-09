import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { login as loginApi } from '@/api/auth'
import type { LoginRequest } from '@/api/auth'
import router from '@/router'

const TOKEN_KEY = 'jwt_token'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem(TOKEN_KEY))

  const isLoggedIn = computed(() => !!token.value)

  function setToken(newToken: string) {
    token.value = newToken
    localStorage.setItem(TOKEN_KEY, newToken)
  }

  function clearToken() {
    token.value = null
    localStorage.removeItem(TOKEN_KEY)
  }

  function isTokenExpired(): boolean {
    if (!token.value) return true
    try {
      const payload = JSON.parse(atob(token.value.split('.')[1]))
      return payload.exp * 1000 <= Date.now()
    } catch {
      return true
    }
  }

  async function login(data: LoginRequest) {
    const res = await loginApi(data)
    setToken(res.token)
  }

  function logout() {
    clearToken()
    router.push('/login')
  }

  return { token, isLoggedIn, setToken, clearToken, isTokenExpired, login, logout }
})

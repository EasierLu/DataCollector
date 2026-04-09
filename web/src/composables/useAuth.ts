import { onMounted, onUnmounted } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { refreshToken } from '@/api/auth'

export function useTokenRefresh() {
  const authStore = useAuthStore()
  let timer: ReturnType<typeof setInterval> | null = null

  function start() {
    timer = setInterval(async () => {
      if (!authStore.token) return
      try {
        const payload = JSON.parse(atob(authStore.token.split('.')[1]))
        const exp = payload.exp * 1000
        const remaining = exp - Date.now()
        if (remaining < 2 * 60 * 60 * 1000 && remaining > 0) {
          const result = await refreshToken()
          if (result?.token) {
            authStore.setToken(result.token)
          }
        }
      } catch {
        // ignore
      }
    }, 5 * 60 * 1000)
  }

  function stop() {
    if (timer) {
      clearInterval(timer)
      timer = null
    }
  }

  onMounted(start)
  onUnmounted(stop)
}

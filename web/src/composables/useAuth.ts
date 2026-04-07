import { onMounted, onUnmounted } from 'vue'
import { refreshToken } from '@/api/auth'

export function useTokenRefresh() {
  let timer: ReturnType<typeof setInterval> | null = null

  function start() {
    timer = setInterval(async () => {
      const token = localStorage.getItem('jwt_token')
      if (!token) return
      try {
        const payload = JSON.parse(atob(token.split('.')[1]))
        const exp = payload.exp * 1000
        const remaining = exp - Date.now()
        if (remaining < 2 * 60 * 60 * 1000 && remaining > 0) {
          const result = await refreshToken()
          if (result?.token) {
            localStorage.setItem('jwt_token', result.token)
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

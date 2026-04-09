import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useAuthStore } from '@/stores/auth'

export const useWebSocketStore = defineStore('websocket', () => {
  const connected = ref(false)
  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let messageHandlers: Set<(data: any) => void> = new Set()
  let reconnectDelay = 1000
  const maxReconnectDelay = 60000

  function getUrl(): string {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    return `${protocol}//${location.host}/api/v1/admin/ws/monitor`
  }

  function cleanup() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  function connect() {
    cleanup()
    const authStore = useAuthStore()
    if (!authStore.token) return

    try {
      ws = new WebSocket(getUrl(), [`access_token.${authStore.token}`])

      ws.onopen = () => {
        connected.value = true
        reconnectDelay = 1000
      }

      ws.onmessage = (event) => {
        try {
          const message = JSON.parse(event.data)
          messageHandlers.forEach((handler) => handler(message))
        } catch {
          // ignore parse errors
        }
      }

      ws.onclose = () => {
        connected.value = false
        reconnectTimer = setTimeout(connect, reconnectDelay)
        reconnectDelay = Math.min(reconnectDelay * 2, maxReconnectDelay)
      }

      ws.onerror = () => {
        connected.value = false
      }
    } catch {
      connected.value = false
    }
  }

  function disconnect() {
    cleanup()
    if (ws) {
      ws.close()
      ws = null
    }
    connected.value = false
    messageHandlers.clear()
    reconnectDelay = 1000
  }

  function reconnect() {
    disconnect()
    connect()
  }

  function onMessage(handler: (data: any) => void) {
    messageHandlers.add(handler)
    return () => {
      messageHandlers.delete(handler)
    }
  }

  return {
    connected,
    connect,
    disconnect,
    reconnect,
    onMessage,
  }
})

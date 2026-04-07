import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useWebSocketStore = defineStore('websocket', () => {
  const connected = ref(false)
  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let messageHandlers: Set<(data: any) => void> = new Set()

  function getUrl(): string {
    const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
    const token = localStorage.getItem('jwt_token')
    return `${protocol}//${location.host}/api/v1/admin/ws/monitor?token=${token}`
  }

  function cleanup() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  function connect() {
    cleanup()
    try {
      ws = new WebSocket(getUrl())

      ws.onopen = () => {
        connected.value = true
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
        reconnectTimer = setTimeout(connect, 5000)
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
  }

  function reconnect() {
    disconnect()
    connect()
  }

  function onMessage(handler: (data: any) => void) {
    messageHandlers.add(handler)
    // 返回取消订阅函数
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

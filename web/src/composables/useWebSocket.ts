import { ref, onUnmounted } from 'vue'

export function useWebSocket(getUrl: () => string) {
  const connected = ref(false)
  let ws: WebSocket | null = null
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let messageHandler: ((data: any) => void) | null = null

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
          messageHandler?.(message)
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

  function cleanup() {
    if (reconnectTimer) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  function onMessage(handler: (data: any) => void) {
    messageHandler = handler
  }

  onUnmounted(() => {
    disconnect()
  })

  return { connected, connect, disconnect, onMessage }
}

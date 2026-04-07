import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAppStore = defineStore('app', () => {
  const sidebarOpen = ref(window.innerWidth >= 768)

  function toggleSidebar() {
    sidebarOpen.value = !sidebarOpen.value
  }

  return { sidebarOpen, toggleSidebar }
})

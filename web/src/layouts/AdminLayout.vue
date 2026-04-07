<template>
  <div class="admin-layout">
    <!-- 侧边栏 -->
    <aside class="sidebar" :class="{ collapsed: !appStore.sidebarOpen }">
      <div class="sidebar-header">
        <el-icon :size="28" color="#a5b4fc"><Monitor /></el-icon>
        <h1 v-show="appStore.sidebarOpen" class="sidebar-title">DataCollector</h1>
      </div>
      <nav class="sidebar-nav">
        <router-link
          v-for="item in menuItems"
          :key="item.path"
          :to="item.path"
          class="nav-item"
          :class="{ active: route.path === item.path || (item.path === '/sources' && route.path.startsWith('/sources')) }"
        >
          <el-icon :size="20"><component :is="item.icon" /></el-icon>
          <span v-show="appStore.sidebarOpen" class="nav-label">{{ item.label }}</span>
        </router-link>
      </nav>
      <!-- WebSocket 状态 -->
      <div class="sidebar-footer">
        <div class="ws-status" :class="{ collapsed: !appStore.sidebarOpen }">
          <div class="ws-status-main">
            <span class="ws-dot" :class="wsStore.connected ? 'connected' : 'disconnected'" />
            <span v-show="appStore.sidebarOpen" class="ws-text">
              {{ wsStore.connected ? '已连接' : '未连接' }}
            </span>
          </div>
          <el-button
            v-show="appStore.sidebarOpen"
            text
            size="small"
            type="primary"
            @click="wsStore.reconnect()"
          >
            重连
          </el-button>
        </div>
      </div>
    </aside>

    <!-- 主内容区 -->
    <div class="main-area">
      <header class="top-header">
        <el-button text @click="appStore.toggleSidebar()">
          <el-icon :size="20"><Fold v-if="appStore.sidebarOpen" /><Expand v-else /></el-icon>
        </el-button>
        <div class="header-right">
          <span class="welcome-text">欢迎</span>
          <el-button type="danger" text @click="authStore.logout()">
            <el-icon class="el-icon--left"><SwitchButton /></el-icon>登出
          </el-button>
        </div>
      </header>
      <main class="page-content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { Monitor, Odometer, Connection, Document, Setting, Fold, Expand, SwitchButton, Notebook } from '@element-plus/icons-vue'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import { useWebSocketStore } from '@/stores/websocket'
import { useTokenRefresh } from '@/composables/useAuth'

const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()
const wsStore = useWebSocketStore()

useTokenRefresh()

onMounted(() => {
  wsStore.connect()
})

onUnmounted(() => {
  wsStore.disconnect()
})

const menuItems = [
  { path: '/dashboard', label: '仪表盘', icon: Odometer },
  { path: '/sources', label: '数据源管理', icon: Connection },
  { path: '/data', label: '数据记录', icon: Document },
  { path: '/settings', label: '系统设置', icon: Setting },
  { path: '/api-docs', label: 'API 文档', icon: Notebook },
]
</script>

<style scoped>
.admin-layout {
  display: flex;
  height: 100vh;
  overflow: hidden;
}

.sidebar {
  width: 240px;
  background: #312e81;
  color: #fff;
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  transition: width 0.3s;
  overflow-y: auto;
  overflow-x: hidden;
}

.sidebar.collapsed {
  width: 64px;
}

.sidebar-header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 20px 16px;
  justify-content: center;
}

.sidebar-title {
  font-size: 18px;
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
}

.sidebar-nav {
  margin-top: 8px;
  display: flex;
  flex-direction: column;
}

.nav-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 20px;
  color: #c7d2fe;
  text-decoration: none;
  transition: background 0.2s;
  white-space: nowrap;
  overflow: hidden;
}

.nav-item:hover {
  background: #3730a3;
}

.nav-item.active {
  background: #1e1b4b;
  border-left: 4px solid #818cf8;
  color: #fff;
}

.collapsed .nav-item {
  justify-content: center;
  padding: 14px 0;
}

.nav-label {
  font-size: 14px;
}

.sidebar-footer {
  margin-top: auto;
  padding: 12px 16px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.ws-status {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 4px;
}

.ws-status.collapsed {
  justify-content: center;
}

.ws-status-main {
  display: flex;
  align-items: center;
  gap: 8px;
}

.ws-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.ws-dot.connected {
  background: #22c55e;
  animation: pulse 2s infinite;
}

.ws-dot.disconnected {
  background: #ef4444;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.ws-text {
  font-size: 13px;
  color: #c7d2fe;
}

.main-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.top-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 20px;
  background: #fff;
  border-bottom: 1px solid #e5e7eb;
  flex-shrink: 0;
}

.header-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.welcome-text {
  color: #6b7280;
  font-size: 14px;
}

.page-content {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  background: #f5f7fa;
}
</style>

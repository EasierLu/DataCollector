<template>
  <div class="login-page">
    <div class="login-card">
      <div class="login-header">
        <el-icon :size="48" color="#fff"><Monitor /></el-icon>
        <h1>DataCollector</h1>
        <p>管理后台登录</p>
      </div>
      <div class="login-body">
        <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" style="margin-bottom: 20px" />
        <el-form :model="form" @keyup.enter="handleLogin" label-position="top">
          <el-form-item label="用户名">
            <el-input v-model="form.username" placeholder="请输入用户名" :prefix-icon="User" size="large" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="form.password" type="password" placeholder="请输入密码" :prefix-icon="Lock" size="large" show-password />
          </el-form-item>
          <el-button type="primary" size="large" :loading="loading" :disabled="!isValid" style="width: 100%; margin-top: 8px" @click="handleLogin">
            {{ loading ? '登录中...' : '登录' }}
          </el-button>
        </el-form>
        <p class="login-footer">DataCollector 数据采集系统</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { User, Lock, Monitor } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'

const router = useRouter()
const authStore = useAuthStore()

const form = ref({ username: '', password: '' })
const loading = ref(false)
const error = ref('')

const isValid = computed(() => form.value.username.trim() !== '' && form.value.password.length >= 6)

async function handleLogin() {
  if (!isValid.value || loading.value) return
  loading.value = true
  error.value = ''
  try {
    await authStore.login(form.value)
    router.push('/dashboard')
  } catch (err: any) {
    error.value = err.message || '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #eef2ff 0%, #dbeafe 100%);
  padding: 16px;
}

.login-card {
  width: 100%;
  max-width: 420px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.login-header {
  background: linear-gradient(135deg, #4f46e5, #2563eb);
  padding: 40px 32px;
  text-align: center;
  color: #fff;
}

.login-header h1 {
  font-size: 24px;
  font-weight: 700;
  margin-top: 12px;
}

.login-header p {
  color: #c7d2fe;
  margin-top: 4px;
  font-size: 14px;
}

.login-body {
  padding: 32px;
}

.login-footer {
  text-align: center;
  color: #9ca3af;
  font-size: 13px;
  margin-top: 24px;
}
</style>

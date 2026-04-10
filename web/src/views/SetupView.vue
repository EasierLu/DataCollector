<template>
  <div class="setup-page">
    <div class="setup-card">
      <div class="setup-header">
        <h1>DataCollector 系统初始化</h1>
        <p>欢迎使用 DataCollector，请完成以下配置以开始使用</p>
      </div>

      <!-- 步骤指示器 -->
      <div class="setup-steps">
        <el-steps :active="step - 1" align-center>
          <el-step title="数据库配置" />
          <el-step title="创建管理员" />
          <el-step title="完成" />
        </el-steps>
      </div>

      <!-- 步骤 1: 数据库配置 -->
      <div v-if="step === 1" class="setup-body">
        <h2>数据库配置</h2>
        <el-form label-position="top">
          <el-form-item label="数据库类型">
            <el-radio-group v-model="dbDriver">
              <el-radio value="sqlite">SQLite</el-radio>
              <el-radio value="postgres">PostgreSQL</el-radio>
            </el-radio-group>
          </el-form-item>

          <template v-if="dbDriver === 'sqlite'">
            <el-form-item label="数据库文件路径">
              <el-input v-model="sqlitePath" placeholder="./data/datacollector.db" />
              <div class="form-tip">SQLite 数据库文件将存储在此路径</div>
            </el-form-item>
          </template>

          <template v-else>
            <el-row :gutter="16">
              <el-col :span="12">
                <el-form-item label="主机地址">
                  <el-input v-model="pgHost" placeholder="localhost" />
                </el-form-item>
              </el-col>
              <el-col :span="12">
                <el-form-item label="端口">
                  <el-input-number v-model="pgPort" :min="1" :max="65535" style="width: 100%" />
                </el-form-item>
              </el-col>
            </el-row>
            <el-form-item label="用户名">
              <el-input v-model="pgUser" placeholder="datacollector" />
            </el-form-item>
            <el-form-item label="密码">
              <el-input v-model="pgPassword" type="password" placeholder="请输入数据库密码" show-password />
            </el-form-item>
            <el-form-item label="数据库名称">
              <el-input v-model="pgDbname" placeholder="datacollector" />
            </el-form-item>
            <el-button :loading="testing" :disabled="!isStep1Valid" style="width: 100%" @click="testConnection">
              {{ testing ? '测试中...' : '测试连接' }}
            </el-button>
            <el-alert v-if="testResult" :title="testResult" :type="testSuccess ? 'success' : 'error'" show-icon :closable="false" style="margin-top: 12px" />
          </template>
        </el-form>
        <div class="step-actions" style="justify-content: flex-end">
          <el-button type="primary" :disabled="!isStep1Valid" @click="step = 2">下一步</el-button>
        </div>
      </div>

      <!-- 步骤 2: 创建管理员 -->
      <div v-if="step === 2" class="setup-body">
        <h2>创建管理员账户</h2>
        <el-form label-position="top">
          <el-form-item label="用户名">
            <el-input v-model="adminUsername" placeholder="admin" />
          </el-form-item>
          <el-form-item label="密码">
            <el-input v-model="adminPassword" type="password" placeholder="请输入密码（至少6位）" show-password />
          </el-form-item>
          <el-form-item label="确认密码" :error="passwordError">
            <el-input v-model="adminPasswordConfirm" type="password" placeholder="请再次输入密码" show-password />
          </el-form-item>
        </el-form>
        <div class="step-actions">
          <el-button @click="step = 1">上一步</el-button>
          <el-button type="primary" :disabled="!isStep2Valid" @click="step = 3">下一步</el-button>
        </div>
      </div>

      <!-- 步骤 3: 确认 -->
      <div v-if="step === 3" class="setup-body">
        <h2>配置摘要</h2>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="数据库类型">{{ dbDriver === 'sqlite' ? 'SQLite' : 'PostgreSQL' }}</el-descriptions-item>
          <el-descriptions-item v-if="dbDriver === 'sqlite'" label="数据库路径">{{ sqlitePath }}</el-descriptions-item>
          <el-descriptions-item v-else label="数据库地址">{{ pgHost }}:{{ pgPort }}</el-descriptions-item>
          <el-descriptions-item label="管理员用户名">{{ adminUsername }}</el-descriptions-item>
        </el-descriptions>

        <el-alert v-if="initSuccess" title="初始化成功！服务正在重启，请稍候..." type="success" show-icon :closable="false" style="margin-top: 16px" />
        <el-alert v-if="error" :title="error" type="error" show-icon :closable="false" style="margin-top: 16px" />

        <div class="step-actions">
          <el-button :disabled="initializing || initSuccess" @click="step = 2">上一步</el-button>
          <el-button type="primary" :loading="initializing" :disabled="initSuccess" @click="handleInitialize">
            {{ initializing ? '初始化中...' : '完成初始化' }}
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { checkSetupStatus, testDatabase, initialize } from '@/api/setup'

const router = useRouter()

const step = ref(1)

// 已初始化则重定向到登录页
onMounted(async () => {
  try {
    const status = await checkSetupStatus()
    if (status.initialized) {
      router.replace('/login')
    }
  } catch {
    // 服务不可用，继续显示 setup 页面
  }
})
const dbDriver = ref('sqlite')
const sqlitePath = ref('./data/datacollector.db')
const pgHost = ref('localhost')
const pgPort = ref(5432)
const pgUser = ref('datacollector')
const pgPassword = ref('')
const pgDbname = ref('datacollector')
const adminUsername = ref('admin')
const adminPassword = ref('')
const adminPasswordConfirm = ref('')
const testing = ref(false)
const testResult = ref('')
const testSuccess = ref(false)
const initializing = ref(false)
const initSuccess = ref(false)
const error = ref('')

const isStep1Valid = computed(() => {
  if (dbDriver.value === 'sqlite') return sqlitePath.value.trim() !== ''
  return pgHost.value.trim() !== '' && pgPort.value > 0 && pgUser.value.trim() !== '' && pgPassword.value !== '' && pgDbname.value.trim() !== ''
})

const isStep2Valid = computed(() => {
  return adminUsername.value.trim() !== '' && adminPassword.value.length >= 6 && adminPassword.value === adminPasswordConfirm.value
})

const passwordError = computed(() => {
  if (adminPassword.value && adminPassword.value.length < 6) return '密码至少需要6位'
  if (adminPasswordConfirm.value && adminPassword.value !== adminPasswordConfirm.value) return '两次输入的密码不一致'
  return ''
})

async function testConnection() {
  testing.value = true
  testResult.value = ''
  testSuccess.value = false
  try {
    await testDatabase({
      driver: dbDriver.value,
      host: pgHost.value,
      port: pgPort.value,
      user: pgUser.value,
      password: pgPassword.value,
      dbname: pgDbname.value,
    })
    testResult.value = '连接成功！'
    testSuccess.value = true
  } catch (err: any) {
    testResult.value = err.message || '连接失败'
  } finally {
    testing.value = false
  }
}

async function handleInitialize() {
  initializing.value = true
  error.value = ''
  try {
    await initialize({
      database: {
        driver: dbDriver.value,
        sqlite: { path: sqlitePath.value },
        postgres: {
          host: pgHost.value,
          port: pgPort.value,
          user: pgUser.value,
          password: pgPassword.value,
          dbname: pgDbname.value,
          sslmode: 'disable',
        },
      },
      server: { port: 8080 },
      admin: {
        username: adminUsername.value,
        password: adminPassword.value,
      },
    })
    initSuccess.value = true
    // 等待服务重启后跳转到登录页
    await waitForServerRestart(true)
    router.push('/login')
  } catch (err: any) {
    error.value = err.message || '初始化失败'
  } finally {
    initializing.value = false
  }
}

// 等待服务重启完成
async function waitForServerRestart(expectInitialized: boolean) {
  // 等待服务开始关闭
  await new Promise(r => setTimeout(r, 1500))
  // 轮询直到服务恢复
  for (let i = 0; i < 30; i++) {
    try {
      const status = await checkSetupStatus()
      if (status.initialized === expectInitialized) return
    } catch {
      // 服务还没恢复
    }
    await new Promise(r => setTimeout(r, 1000))
  }
}
</script>

<style scoped>
.setup-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #eef2ff 0%, #dbeafe 100%);
  padding: 16px;
}

.setup-card {
  width: 100%;
  max-width: 640px;
  background: #fff;
  border-radius: 16px;
  box-shadow: 0 20px 60px rgba(0, 0, 0, 0.1);
  overflow: hidden;
}

.setup-header {
  background: linear-gradient(135deg, #4f46e5, #2563eb);
  padding: 28px 32px;
  color: #fff;
}

.setup-header h1 {
  font-size: 22px;
  font-weight: 700;
}

.setup-header p {
  color: #c7d2fe;
  margin-top: 4px;
  font-size: 14px;
}

.setup-steps {
  padding: 24px 32px;
  border-bottom: 1px solid #f3f4f6;
}

.setup-body {
  padding: 24px 32px 32px;
}

.setup-body h2 {
  font-size: 16px;
  font-weight: 600;
  color: #1f2937;
  margin-bottom: 16px;
}

.step-actions {
  display: flex;
  justify-content: space-between;
  margin-top: 24px;
}

.form-tip {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 4px;
}
</style>

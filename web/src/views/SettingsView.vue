<template>
  <div v-loading="loading">
    <h2 class="page-title">系统设置</h2>

    <template v-if="!loading">
      <!-- 系统信息 -->
      <el-card shadow="hover" style="margin-bottom: 20px">
        <template #header><span class="card-title">系统信息</span></template>
        <el-row :gutter="20">
          <el-col :span="8">
            <div class="info-block">
              <div class="info-label">系统版本</div>
              <div class="info-value">{{ systemInfo.version || '-' }}</div>
            </div>
          </el-col>
          <el-col :span="8">
            <div class="info-block">
              <div class="info-label">运行时间</div>
              <div class="info-value">{{ systemInfo.uptime || '-' }}</div>
            </div>
          </el-col>
          <el-col :span="8">
            <div class="info-block">
              <div class="info-label">系统状态</div>
              <div class="info-value">
                <el-tag :type="systemInfo.status === 'healthy' ? 'success' : 'danger'">
                  {{ systemInfo.status === 'healthy' ? '正常运行' : '异常' }}
                </el-tag>
              </div>
            </div>
          </el-col>
        </el-row>
      </el-card>

      <!-- 数据库配置 -->
      <el-card shadow="hover" style="margin-bottom: 20px">
        <template #header><span class="card-title">数据库配置</span></template>
        <el-descriptions :column="2" border>
          <el-descriptions-item label="连接状态">
            <el-tag :type="systemInfo.database === 'connected' ? 'success' : 'danger'">
              {{ systemInfo.database === 'connected' ? '已连接' : '未连接' }}
            </el-tag>
          </el-descriptions-item>
        </el-descriptions>
        <p class="config-tip">数据库配置为只读展示，如需修改请直接编辑配置文件</p>
      </el-card>

      <!-- 修改密码 -->
      <el-card shadow="hover" style="margin-bottom: 20px">
        <template #header><span class="card-title">修改密码</span></template>
        <el-form ref="pwdFormRef" :model="pwdForm" :rules="pwdRules" label-width="100px" style="max-width: 420px">
          <el-form-item label="旧密码" prop="oldPassword">
            <el-input v-model="pwdForm.oldPassword" type="password" show-password placeholder="请输入旧密码" />
          </el-form-item>
          <el-form-item label="新密码" prop="newPassword">
            <el-input v-model="pwdForm.newPassword" type="password" show-password placeholder="请输入新密码（至少6位）" />
          </el-form-item>
          <el-form-item label="确认密码" prop="confirmPassword">
            <el-input v-model="pwdForm.confirmPassword" type="password" show-password placeholder="请再次输入新密码" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" :loading="changingPwd" @click="handleChangePassword">确认修改</el-button>
          </el-form-item>
        </el-form>
      </el-card>

      <!-- 危险操作 -->
      <el-card shadow="hover" class="danger-card">
        <template #header><span class="card-title danger-title">危险操作</span></template>
        <el-alert title="此操作将清除所有数据并恢复系统到初始状态。所有数据源、Token 和数据记录将被永久删除。" type="error" show-icon :closable="false" style="margin-bottom: 16px" />
        <el-button type="danger" @click="openReinitDialog">重新初始化</el-button>
      </el-card>
    </template>

    <!-- 重新初始化弹窗 -->
    <el-dialog v-model="reinitVisible" title="确认重新初始化" width="460px" destroy-on-close>
      <el-alert type="error" show-icon :closable="false" style="margin-bottom: 16px">
        <template #title>
          <div>
            <p style="margin-bottom: 8px">重新初始化将删除所有数据，包括：</p>
            <ul style="padding-left: 20px; margin: 0">
              <li>所有数据源配置</li>
              <li>所有 Token</li>
              <li>所有数据记录</li>
              <li>管理员账户（需要重新创建）</li>
            </ul>
          </div>
        </template>
      </el-alert>
      <el-form-item label="请输入 &quot;REINITIALIZE&quot; 以确认：">
        <el-input v-model="reinitConfirmText" placeholder="REINITIALIZE" />
      </el-form-item>
      <el-alert v-if="reinitError" :title="reinitError" type="error" show-icon :closable="false" style="margin-top: 8px" />
      <template #footer>
        <el-button @click="reinitVisible = false">取消</el-button>
        <el-button type="danger" :loading="reinitializing" :disabled="reinitConfirmText !== 'REINITIALIZE'" @click="confirmReinit">
          确认重新初始化
        </el-button>
      </template>
    </el-dialog>

    <!-- 重新初始化成功弹窗 -->
    <el-dialog v-model="reinitSuccessVisible" title="重新初始化成功" width="400px" :close-on-click-modal="false" :show-close="false">
      <el-result icon="success" title="系统已重新初始化" sub-title="请重启服务器后访问初始化页面重新配置。" />
      <template #footer>
        <el-button type="primary" style="width: 100%" @click="handleLogout">返回登录页</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { healthCheck } from '@/api/health'
import { reinitialize } from '@/api/setup'
import { changePassword } from '@/api/auth'
import { useAuthStore } from '@/stores/auth'
import type { HealthResponse } from '@/api/health'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'

const authStore = useAuthStore()

const loading = ref(true)
const systemInfo = ref<HealthResponse>({ status: '', version: '', uptime: '', database: '' })

// 重新初始化
const reinitVisible = ref(false)
const reinitConfirmText = ref('')
const reinitError = ref('')
const reinitializing = ref(false)
const reinitSuccessVisible = ref(false)

// 修改密码
const pwdFormRef = ref<FormInstance>()
const changingPwd = ref(false)
const pwdForm = reactive({
  oldPassword: '',
  newPassword: '',
  confirmPassword: '',
})
const pwdRules = reactive<FormRules>({
  oldPassword: [{ required: true, message: '请输入旧密码', trigger: 'blur' }],
  newPassword: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 6, message: '密码长度不能少于6位', trigger: 'blur' },
  ],
  confirmPassword: [
    { required: true, message: '请再次输入新密码', trigger: 'blur' },
    {
      validator: (_rule: any, value: string, callback: any) => {
        if (value !== pwdForm.newPassword) {
          callback(new Error('两次输入的密码不一致'))
        } else {
          callback()
        }
      },
      trigger: 'blur',
    },
  ],
})

async function handleChangePassword() {
  if (!pwdFormRef.value) return
  await pwdFormRef.value.validate()
  changingPwd.value = true
  try {
    await changePassword({ old_password: pwdForm.oldPassword, new_password: pwdForm.newPassword })
    ElMessage.success('密码修改成功')
    pwdForm.oldPassword = ''
    pwdForm.newPassword = ''
    pwdForm.confirmPassword = ''
    pwdFormRef.value.resetFields()
  } catch (err: any) {
    ElMessage.error(err.message || '修改密码失败')
  } finally {
    changingPwd.value = false
  }
}

async function loadHealth() {
  try {
    systemInfo.value = await healthCheck()
  } catch {
    // handled
  } finally {
    loading.value = false
  }
}

function openReinitDialog() {
  reinitConfirmText.value = ''
  reinitError.value = ''
  reinitVisible.value = true
}

async function confirmReinit() {
  if (reinitConfirmText.value !== 'REINITIALIZE') return
  reinitializing.value = true
  reinitError.value = ''
  try {
    await reinitialize(reinitConfirmText.value)
    reinitVisible.value = false
    reinitSuccessVisible.value = true
  } catch (err: any) {
    reinitError.value = err.message || '重新初始化失败'
  } finally {
    reinitializing.value = false
  }
}

function handleLogout() {
  authStore.logout()
}

onMounted(loadHealth)
</script>

<style scoped>
.page-title {
  font-size: 22px;
  font-weight: 700;
  color: #1f2937;
  margin-bottom: 20px;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
}

.danger-title {
  color: #dc2626;
}

.danger-card {
  border-color: #fecaca;
}

.info-block {
  background: #f9fafb;
  border-radius: 8px;
  padding: 16px;
}

.info-label {
  font-size: 13px;
  color: #6b7280;
  margin-bottom: 4px;
}

.info-value {
  font-size: 18px;
  font-weight: 600;
  color: #1f2937;
}

.config-tip {
  font-size: 13px;
  color: #9ca3af;
  margin-top: 12px;
}
</style>

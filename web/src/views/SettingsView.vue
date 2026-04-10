<template>
  <div v-loading="loading">
    <h2 class="page-title">系统设置</h2>

    <template v-if="!loading">
      <el-tabs v-model="activeTab" type="border-card">
        <!-- 基本信息 -->
        <el-tab-pane label="基本信息" name="info">
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

          <el-card shadow="hover">
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
        </el-tab-pane>

        <!-- 限流配置 -->
        <el-tab-pane label="限流配置" name="rateLimit">
          <el-card shadow="hover">
            <el-form :model="rateLimitForm" label-width="180px" style="max-width: 520px" v-loading="loadingRateLimit">
              <el-form-item label="每IP每分钟请求数">
                <el-input-number v-model="rateLimitForm.rate_limit_per_ip" :min="1" :max="100000" style="width: 100%" />
              </el-form-item>
              <el-form-item label="每IP突发量">
                <el-input-number v-model="rateLimitForm.rate_limit_per_ip_burst" :min="1" :max="10000" style="width: 100%" />
              </el-form-item>
              <el-form-item label="每Token每分钟请求数">
                <el-input-number v-model="rateLimitForm.rate_limit_per_token" :min="1" :max="100000" style="width: 100%" />
              </el-form-item>
              <el-form-item label="每Token突发量">
                <el-input-number v-model="rateLimitForm.rate_limit_per_token_burst" :min="1" :max="10000" style="width: 100%" />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" :loading="savingRateLimit" @click="handleSaveRateLimit">保存配置</el-button>
              </el-form-item>
            </el-form>
            <p class="config-tip">限流配置保存后立即生效，无需重启服务。数据源可单独配置限流参数覆盖全局默认值。</p>
          </el-card>
        </el-tab-pane>

        <!-- API Key 管理 -->
        <el-tab-pane label="API Key 管理" name="apiKey">
          <el-card shadow="hover">
            <el-alert type="info" show-icon :closable="false" style="margin-bottom: 16px">
              <template #title>
                API Key 用于数据查询等接口的鉴权，与数据上报 Token 无关。请求时通过 <code>X-API-Key</code> 请求头传递。
              </template>
            </el-alert>

            <div style="margin-bottom: 16px">
              <el-button type="primary" @click="openCreateApiKeyDialog">创建 API Key</el-button>
            </div>

            <div v-loading="loadingApiKeys">
              <el-table :data="apiKeys" size="small" stripe>
                <el-table-column prop="name" label="名称" min-width="150" />
                <el-table-column label="权限" min-width="200">
                  <template #default="{ row }">
                    <el-tag v-for="perm in parsePermissions(row.permissions)" :key="perm" size="small" style="margin-right: 4px">
                      {{ permissionLabel(perm) }}
                    </el-tag>
                    <span v-if="!row.permissions" style="color: #9ca3af">无权限</span>
                  </template>
                </el-table-column>
                <el-table-column label="过期时间" width="170">
                  <template #default="{ row }">
                    {{ row.expires_at ? new Date(row.expires_at).toLocaleString() : '永不过期' }}
                  </template>
                </el-table-column>
                <el-table-column label="最后使用" width="170">
                  <template #default="{ row }">
                    {{ row.last_used_at ? new Date(row.last_used_at).toLocaleString() : '-' }}
                  </template>
                </el-table-column>
                <el-table-column label="创建时间" width="170">
                  <template #default="{ row }">
                    {{ new Date(row.created_at).toLocaleString() }}
                  </template>
                </el-table-column>
                <el-table-column label="操作" width="140" fixed="right">
                  <template #default="{ row }">
                    <el-button link type="primary" size="small" @click="openEditPermissionsDialog(row)">
                      权限
                    </el-button>
                    <el-popconfirm title="确定删除该 API Key？" @confirm="handleDeleteApiKey(row.id)">
                      <template #reference>
                        <el-button link type="danger" size="small">删除</el-button>
                      </template>
                    </el-popconfirm>
                  </template>
                </el-table-column>
              </el-table>
              <el-empty v-if="!loadingApiKeys && apiKeys.length === 0" description="暂无 API Key，请点击上方按钮创建" />
            </div>
          </el-card>
        </el-tab-pane>

        <!-- 账户安全 -->
        <el-tab-pane label="账户安全" name="security">
          <el-card shadow="hover">
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
        </el-tab-pane>

        <!-- 危险操作 -->
        <el-tab-pane label="危险操作" name="danger">
          <el-card shadow="hover" class="danger-card">
            <el-alert title="此操作将清除所有数据并恢复系统到初始状态。所有数据源、Token 和数据记录将被永久删除。" type="error" show-icon :closable="false" style="margin-bottom: 16px" />
            <el-button type="danger" @click="openReinitDialog">重新初始化</el-button>
          </el-card>
        </el-tab-pane>
      </el-tabs>
    </template>

    <!-- 创建 API Key 弹窗 -->
    <el-dialog v-model="createApiKeyVisible" title="创建 API Key" width="480px" destroy-on-close>
      <el-form ref="apiKeyFormRef" :model="apiKeyForm" :rules="apiKeyRules" label-width="100px">
        <el-form-item label="名称" prop="name">
          <el-input v-model="apiKeyForm.name" placeholder="为该 API Key 命名" />
        </el-form-item>
        <el-form-item label="权限" prop="permissions">
          <el-checkbox-group v-model="apiKeyForm.permissions">
            <el-checkbox v-for="perm in availablePermissions" :key="perm" :label="perm" :value="perm">
              {{ permissionLabel(perm) }}
            </el-checkbox>
          </el-checkbox-group>
        </el-form-item>
        <el-form-item label="过期时间">
          <el-date-picker v-model="apiKeyForm.expiresAt" type="datetime" placeholder="不设置则永不过期" style="width: 100%" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createApiKeyVisible = false">取消</el-button>
        <el-button type="primary" :loading="creatingApiKey" @click="handleCreateApiKey">创建</el-button>
      </template>
    </el-dialog>

    <!-- 编辑权限弹窗 -->
    <el-dialog v-model="editPermissionsVisible" title="编辑 API Key 权限" width="400px" destroy-on-close>
      <el-checkbox-group v-model="editPermissionsForm">
        <el-checkbox v-for="perm in availablePermissions" :key="perm" :label="perm" :value="perm">
          {{ permissionLabel(perm) }}
        </el-checkbox>
      </el-checkbox-group>
      <template #footer>
        <el-button @click="editPermissionsVisible = false">取消</el-button>
        <el-button type="primary" :loading="savingPermissions" @click="handleSavePermissions">保存</el-button>
      </template>
    </el-dialog>

    <!-- 创建成功弹窗（显示明文 Key） -->
    <el-dialog v-model="apiKeyResultVisible" title="API Key 创建成功" width="520px" :close-on-click-modal="false">
      <el-alert type="warning" show-icon :closable="false" style="margin-bottom: 16px">
        <template #title>请妥善保存以下 API Key，关闭后将无法再次查看。</template>
      </el-alert>
      <el-input :model-value="apiKeyResult" readonly>
        <template #append>
          <el-button @click="handleCopyApiKey">复制</el-button>
        </template>
      </el-input>
      <template #footer>
        <el-button type="primary" @click="apiKeyResultVisible = false">我已保存</el-button>
      </template>
    </el-dialog>

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
      <el-result icon="success" title="系统已重新初始化" :sub-title="reinitRestarting ? '服务正在重启，请稍候...' : '服务已重启，即将跳转到初始化页面'" />
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { healthCheck } from '@/api/health'
import { reinitialize, checkSetupStatus } from '@/api/setup'
import { changePassword } from '@/api/auth'
import { getRateLimitSettings, updateRateLimitSettings } from '@/api/settings'
import type { RateLimitSettings } from '@/api/settings'
import { listApiKeys, createApiKey, listPermissions, updateApiKeyPermissions, deleteApiKey } from '@/api/apikey'
import type { ApiKey } from '@/api/apikey'
import { copyToClipboard } from '@/utils/clipboard'

import type { HealthResponse } from '@/api/health'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'


const activeTab = ref('info')
const loading = ref(true)
const systemInfo = ref<HealthResponse>({ status: '', version: '', uptime: '', database: '' })

// 限流配置
const loadingRateLimit = ref(false)
const savingRateLimit = ref(false)
const rateLimitForm = reactive<RateLimitSettings>({
  rate_limit_per_ip: 200,
  rate_limit_per_ip_burst: 50,
  rate_limit_per_token: 100,
  rate_limit_per_token_burst: 20,
})

// API Key 管理
const loadingApiKeys = ref(false)
const apiKeys = ref<ApiKey[]>([])
const availablePermissions = ref<string[]>(['query'])

const createApiKeyVisible = ref(false)
const creatingApiKey = ref(false)
const apiKeyFormRef = ref<FormInstance>()
const apiKeyForm = reactive({ name: '', permissions: ['query'] as string[], expiresAt: null as Date | null })
const apiKeyRules = reactive<FormRules>({
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
  permissions: [{ type: 'array', required: true, message: '请至少选择一个权限', trigger: 'change' }],
})
const apiKeyResultVisible = ref(false)
const apiKeyResult = ref('')

// 编辑权限
const editPermissionsVisible = ref(false)
const editPermissionsForm = ref<string[]>([])
const editingApiKeyId = ref<number>(0)
const savingPermissions = ref(false)

// 权限标签映射
const permissionLabelMap: Record<string, string> = {
  query: '数据查询',
}
function permissionLabel(perm: string) {
  return permissionLabelMap[perm] || perm
}
function parsePermissions(perms: string): string[] {
  if (!perms) return []
  return perms.split(',').map(p => p.trim()).filter(Boolean)
}

// 重新初始化
const reinitVisible = ref(false)
const reinitConfirmText = ref('')
const reinitError = ref('')
const reinitializing = ref(false)
const reinitSuccessVisible = ref(false)
const reinitRestarting = ref(false)

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

// ---- 基本信息 ----
async function loadHealth() {
  try {
    systemInfo.value = await healthCheck()
  } catch {
    // handled
  } finally {
    loading.value = false
  }
}

// ---- 限流配置 ----
async function loadRateLimitSettings() {
  loadingRateLimit.value = true
  try {
    const settings = await getRateLimitSettings()
    rateLimitForm.rate_limit_per_ip = settings.rate_limit_per_ip
    rateLimitForm.rate_limit_per_ip_burst = settings.rate_limit_per_ip_burst
    rateLimitForm.rate_limit_per_token = settings.rate_limit_per_token
    rateLimitForm.rate_limit_per_token_burst = settings.rate_limit_per_token_burst
  } catch {
    // handled
  } finally {
    loadingRateLimit.value = false
  }
}

async function handleSaveRateLimit() {
  savingRateLimit.value = true
  try {
    await updateRateLimitSettings({ ...rateLimitForm })
    ElMessage.success('限流配置已保存')
  } catch (err: any) {
    ElMessage.error(err.message || '保存失败')
  } finally {
    savingRateLimit.value = false
  }
}

// ---- API Key 管理 ----
async function loadApiKeys() {
  loadingApiKeys.value = true
  try {
    apiKeys.value = await listApiKeys() || []
  } catch {
    // handled
  } finally {
    loadingApiKeys.value = false
  }
}

async function loadAvailablePermissions() {
  try {
    const perms = await listPermissions()
    if (perms && perms.length > 0) {
      availablePermissions.value = perms
    }
  } catch {
    // 使用默认值
  }
}

function openCreateApiKeyDialog() {
  apiKeyForm.name = ''
  apiKeyForm.permissions = ['query']
  apiKeyForm.expiresAt = null
  createApiKeyVisible.value = true
}

async function handleCreateApiKey() {
  if (!apiKeyFormRef.value) return
  await apiKeyFormRef.value.validate()
  creatingApiKey.value = true
  try {
    const data: any = { name: apiKeyForm.name, permissions: apiKeyForm.permissions }
    if (apiKeyForm.expiresAt) {
      data.expires_at = new Date(apiKeyForm.expiresAt).toISOString()
    }
    const result = await createApiKey(data)
    createApiKeyVisible.value = false
    apiKeyResult.value = result.key
    apiKeyResultVisible.value = true
    loadApiKeys()
  } catch (err: any) {
    ElMessage.error(err.message || '创建失败')
  } finally {
    creatingApiKey.value = false
  }
}

async function handleCopyApiKey() {
  const ok = await copyToClipboard(apiKeyResult.value)
  if (ok) {
    ElMessage.success('已复制到剪贴板')
  } else {
    ElMessage.warning('复制失败，请手动复制')
  }
}

function openEditPermissionsDialog(key: ApiKey) {
  editingApiKeyId.value = key.id
  editPermissionsForm.value = parsePermissions(key.permissions)
  editPermissionsVisible.value = true
}

async function handleSavePermissions() {
  if (editPermissionsForm.value.length === 0) {
    ElMessage.warning('请至少选择一个权限')
    return
  }
  savingPermissions.value = true
  try {
    await updateApiKeyPermissions(editingApiKeyId.value, editPermissionsForm.value)
    ElMessage.success('权限已更新')
    editPermissionsVisible.value = false
    loadApiKeys()
  } catch (err: any) {
    ElMessage.error(err.message || '更新失败')
  } finally {
    savingPermissions.value = false
  }
}

async function handleDeleteApiKey(id: number) {
  try {
    await deleteApiKey(id)
    ElMessage.success('已删除')
    loadApiKeys()
  } catch (err: any) {
    ElMessage.error(err.message || '删除失败')
  }
}

// ---- 修改密码 ----
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

// ---- 危险操作 ----
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
    reinitRestarting.value = true
    // 等待服务重启后跳转到初始化页面
    await waitForServerRestart()
    reinitRestarting.value = false
    await new Promise(r => setTimeout(r, 1000))
    window.location.href = '/setup'
  } catch (err: any) {
    reinitError.value = err.message || '重新初始化失败'
  } finally {
    reinitializing.value = false
  }
}

async function waitForServerRestart() {
  await new Promise(r => setTimeout(r, 1500))
  for (let i = 0; i < 30; i++) {
    try {
      const status = await checkSetupStatus()
      if (!status.initialized) return
    } catch {
      // 服务还没恢复
    }
    await new Promise(r => setTimeout(r, 1000))
  }
}

onMounted(() => {
  loadHealth()
  loadRateLimitSettings()
  loadApiKeys()
  loadAvailablePermissions()
})
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

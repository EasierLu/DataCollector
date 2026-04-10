<template>
  <el-card shadow="hover">
    <el-alert type="info" show-icon :closable="false" style="margin-bottom: 16px">
      <template #title>
        API Key 用于数据查询等接口的鉴权，与数据上报 Token 无关。请求时通过 <code>X-API-Key</code> 请求头传递。
      </template>
    </el-alert>

    <div style="margin-bottom: 16px">
      <el-button type="primary" @click="openCreateDialog">创建 API Key</el-button>
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

  <!-- 创建 API Key 弹窗 -->
  <el-dialog v-model="createDialogVisible" title="创建 API Key" width="480px" destroy-on-close>
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
      <el-button @click="createDialogVisible = false">取消</el-button>
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
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { listApiKeys, createApiKey, listPermissions, updateApiKeyPermissions, deleteApiKey } from '@/api/apikey'
import type { ApiKey } from '@/api/apikey'
import { copyToClipboard } from '@/utils/clipboard'
import type { FormInstance, FormRules } from 'element-plus'
import { ElMessage } from 'element-plus'

const loadingApiKeys = ref(false)
const apiKeys = ref<ApiKey[]>([])
const availablePermissions = ref<string[]>(['query'])

const createDialogVisible = ref(false)
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

function openCreateDialog() {
  apiKeyForm.name = ''
  apiKeyForm.permissions = ['query']
  apiKeyForm.expiresAt = null
  createDialogVisible.value = true
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
    createDialogVisible.value = false
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

onMounted(() => {
  loadApiKeys()
  loadAvailablePermissions()
})
</script>

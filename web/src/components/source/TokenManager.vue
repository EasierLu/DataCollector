<template>
  <el-card shadow="hover">
    <template #header>
      <div class="card-header">
        <span class="card-title">Token 管理</span>
        <el-button type="primary" @click="openCreateToken">
          <el-icon class="el-icon--left"><Plus /></el-icon>生成新 Token
        </el-button>
      </div>
    </template>
    <el-table :data="tokens" stripe style="width: 100%">
      <el-table-column prop="id" label="ID" width="70" />
      <el-table-column prop="name" label="名称" min-width="120" />
      <el-table-column label="状态" width="100">
        <template #default="{ row }">
          <el-switch :model-value="row.status === 1" @change="toggleStatus(row)" active-text="启用" inactive-text="禁用" />
        </template>
      </el-table-column>
      <el-table-column label="过期时间" width="180">
        <template #default="{ row }">{{ row.expires_at ? formatDate(row.expires_at) : '永不过期' }}</template>
      </el-table-column>
      <el-table-column label="最后使用" width="180">
        <template #default="{ row }">{{ row.last_used_at ? formatDate(row.last_used_at) : '从未使用' }}</template>
      </el-table-column>
      <el-table-column label="创建时间" width="180">
        <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" width="80" fixed="right">
        <template #default="{ row }">
          <el-button text type="danger" @click="handleDeleteToken(row)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-if="tokens.length === 0" description="暂无 Token，点击「生成新 Token」按钮创建" />
  </el-card>

  <!-- 创建 Token 弹窗 -->
  <el-dialog v-model="createTokenVisible" title="生成新 Token" width="460px" destroy-on-close>
    <el-form :model="tokenForm" label-position="top">
      <el-form-item label="Token 名称" required>
        <el-input v-model="tokenForm.name" placeholder="例如：生产环境 Token" />
      </el-form-item>
      <el-form-item label="过期时间（可选）">
        <el-date-picker v-model="tokenForm.expires_at" type="datetime" placeholder="留空表示永不过期" value-format="YYYY-MM-DDTHH:mm:ssZ" style="width: 100%" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="createTokenVisible = false">取消</el-button>
      <el-button type="primary" :loading="creatingToken" :disabled="!tokenForm.name.trim()" @click="handleCreateToken">生成</el-button>
    </template>
  </el-dialog>

  <!-- Token 生成结果弹窗 -->
  <el-dialog v-model="tokenResultVisible" title="Token 生成成功" width="520px" :close-on-click-modal="false">
    <el-alert title="警告：此 Token 仅显示一次，关闭后将无法再次查看。请妥善保存！" type="warning" show-icon :closable="false" style="margin-bottom: 16px" />
    <div class="token-display">
      <div class="token-display-header">
        <span>Token</span>
        <el-button text size="small" @click="handleCopyToken">复制</el-button>
      </div>
      <code class="token-value">{{ generatedToken }}</code>
    </div>
    <el-descriptions :column="1" style="margin-top: 16px">
      <el-descriptions-item label="名称">{{ generatedTokenInfo.name }}</el-descriptions-item>
      <el-descriptions-item label="过期时间">{{ generatedTokenInfo.expires_at ? formatDate(generatedTokenInfo.expires_at) : '永不过期' }}</el-descriptions-item>
    </el-descriptions>
    <template #footer>
      <el-button type="primary" style="width: 100%" @click="tokenResultVisible = false">我已保存，关闭</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listTokens, createToken, updateTokenStatus, deleteToken } from '@/api/token'
import { formatDate } from '@/utils/format'
import { copyToClipboard } from '@/utils/clipboard'
import type { DataToken } from '@/types/token'

const props = defineProps<{
  sourceId: number
}>()

const tokens = ref<DataToken[]>([])

// 创建 Token
const createTokenVisible = ref(false)
const creatingToken = ref(false)
const tokenForm = ref({ name: '', expires_at: null as Date | null })

// Token 结果
const tokenResultVisible = ref(false)
const generatedToken = ref('')
const generatedTokenInfo = ref<{ name: string; expires_at: string | null }>({ name: '', expires_at: null })

async function loadTokens() {
  const result = await listTokens(props.sourceId)
  tokens.value = result || []
}

function openCreateToken() {
  tokenForm.value = { name: '', expires_at: null }
  createTokenVisible.value = true
}

async function handleCreateToken() {
  if (!tokenForm.value.name.trim()) return
  creatingToken.value = true
  try {
    const data: any = { name: tokenForm.value.name.trim() }
    if (tokenForm.value.expires_at) {
      data.expires_at = new Date(tokenForm.value.expires_at).toISOString()
    }
    const result = await createToken(props.sourceId, data)
    generatedToken.value = result.token
    generatedTokenInfo.value = { name: result.name, expires_at: result.expires_at }
    createTokenVisible.value = false
    tokenResultVisible.value = true
    await loadTokens()
  } catch (err: any) {
    ElMessage.error(err.message || '生成 Token 失败')
  } finally {
    creatingToken.value = false
  }
}

async function handleCopyToken() {
  const ok = await copyToClipboard(generatedToken.value)
  if (ok) {
    ElMessage.success('Token 已复制到剪贴板')
  } else {
    ElMessage.error('复制失败，请手动复制')
  }
}

async function toggleStatus(token: DataToken) {
  const newStatus = token.status === 1 ? 0 : 1
  try {
    await updateTokenStatus(token.id, newStatus)
    ElMessage.success(newStatus === 1 ? 'Token 已启用' : 'Token 已禁用')
    await loadTokens()
  } catch (err: any) {
    ElMessage.error(err.message || '操作失败')
  }
}

async function handleDeleteToken(token: DataToken) {
  try {
    await ElMessageBox.confirm(`确定要删除 Token "${token.name}" 吗？使用此 Token 的应用将无法正常提交数据。`, '确认删除 Token', {
      type: 'warning',
      confirmButtonText: '确认删除',
      confirmButtonClass: 'el-button--danger',
    })
    await deleteToken(token.id)
    ElMessage.success('Token 已删除')
    await loadTokens()
  } catch (err: any) {
    if (err !== 'cancel' && err?.message) {
      ElMessage.error(err.message)
    }
  }
}

defineExpose({ loadTokens })
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.card-title {
  font-size: 16px;
  font-weight: 600;
}

.token-display {
  background: #111827;
  border-radius: 8px;
  padding: 16px;
}

.token-display-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.token-display-header span {
  color: #9ca3af;
  font-size: 12px;
}

.token-value {
  color: #34d399;
  font-family: monospace;
  font-size: 13px;
  word-break: break-all;
}
</style>

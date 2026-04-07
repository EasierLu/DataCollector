<template>
  <div v-loading="loading">
    <div style="margin-bottom: 16px">
      <el-button text type="primary" @click="$router.push('/sources')">
        <el-icon class="el-icon--left"><ArrowLeft /></el-icon>返回数据源列表
      </el-button>
    </div>

    <template v-if="!loading">
      <!-- 数据源信息 -->
      <el-card shadow="hover" style="margin-bottom: 20px">
        <div class="source-header">
          <div>
            <h2 class="source-name">{{ source.name }}</h2>
            <p class="source-desc">{{ source.description || '暂无描述' }}</p>
          </div>
          <el-tag :type="source.status === 1 ? 'success' : 'danger'">{{ source.status === 1 ? '启用' : '禁用' }}</el-tag>
        </div>
        <el-descriptions :column="3" style="margin-top: 16px">
          <el-descriptions-item label="数据源 ID">{{ source.id }}</el-descriptions-item>
          <el-descriptions-item label="采集标识">
            <code style="background: #f3f4f6; padding: 2px 8px; border-radius: 4px; font-family: monospace">{{ source.collect_id }}</code>
          </el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ formatDate(source.created_at) }}</el-descriptions-item>
          <el-descriptions-item label="更新时间">{{ formatDate(source.updated_at) }}</el-descriptions-item>
          <el-descriptions-item label="限流（每分钟请求数）">
            {{ source.rate_limit ? source.rate_limit : '使用全局默认' }}
          </el-descriptions-item>
          <el-descriptions-item label="限流（突发量）">
            {{ source.rate_limit_burst ? source.rate_limit_burst : '使用全局默认' }}
          </el-descriptions-item>
        </el-descriptions>

        <!-- Schema 配置 -->
        <div v-if="schemaFields.length > 0" style="margin-top: 20px; border-top: 1px solid #f3f4f6; padding-top: 16px">
          <h4 style="font-size: 14px; color: #374151; margin-bottom: 12px">Schema 字段配置</h4>
          <el-table :data="schemaFields" size="small" border>
            <el-table-column prop="name" label="字段名" />
            <el-table-column prop="type" label="类型" width="120">
              <template #default="{ row }">
                <el-tag size="small" type="info">{{ row.type }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="必填" width="80">
              <template #default="{ row }">
                <span :style="{ color: row.required ? '#059669' : '#9ca3af' }">{{ row.required ? '是' : '否' }}</span>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </el-card>

      <!-- 调用示例 -->
      <el-card shadow="hover" style="margin-bottom: 20px">
        <div class="card-header" style="margin-bottom: 12px">
          <span class="card-title">调用示例</span>
          <el-button text size="small" @click="handleCopyCurl">复制</el-button>
        </div>
        <div class="curl-display">
          <pre class="curl-code">{{ curlExample }}</pre>
        </div>
        <p class="curl-tip">请将 <code>dt_&lt;your-token&gt;</code> 替换为实际的 Data Token。</p>
      </el-card>

      <!-- Token 管理 -->
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
    </template>

    <!-- 创建 Token 弹窗 -->
    <el-dialog v-model="createTokenVisible" title="生成新 Token" width="460px" destroy-on-close>
      <el-form :model="tokenForm" label-position="top">
        <el-form-item label="Token 名称" required>
          <el-input v-model="tokenForm.name" placeholder="例如：生产环境 Token" />
        </el-form-item>
        <el-form-item label="过期时间（可选）">
          <el-date-picker v-model="tokenForm.expires_at" type="datetime" placeholder="留空表示永不过期" style="width: 100%" />
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
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowLeft, Plus } from '@element-plus/icons-vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { listSources } from '@/api/source'
import { listTokens, createToken, updateTokenStatus, deleteToken } from '@/api/token'
import { formatDate } from '@/utils/format'
import { copyToClipboard } from '@/utils/clipboard'
import type { DataSource, SchemaField } from '@/types/source'
import type { DataToken } from '@/types/token'

const route = useRoute()
const sourceId = Number(route.params.id)

const loading = ref(true)
const source = ref<Partial<DataSource>>({})
const schemaFields = ref<SchemaField[]>([])
const tokens = ref<DataToken[]>([])

// 创建 Token
const createTokenVisible = ref(false)
const creatingToken = ref(false)
const tokenForm = ref({ name: '', expires_at: null as Date | null })

// Token 结果
const tokenResultVisible = ref(false)
const generatedToken = ref('')
const generatedTokenInfo = ref<{ name: string; expires_at: string | null }>({ name: '', expires_at: null })

const curlExample = computed(() => {
  const origin = window.location.origin
  const fields: Record<string, string> = {}
  for (const f of schemaFields.value) {
    fields[f.name] = getExampleValue(f)
  }
  if (Object.keys(fields).length === 0) {
    fields['key'] = 'value'
  }
  const body = JSON.stringify(fields, null, 2)
  return `curl -X POST ${origin}/api/v1/collect/${source.value.collect_id || sourceId} \\
  -H "Content-Type: application/json" \\
  -H "X-Data-Token: dt_<your-token>" \\
  -d '${body}'`
})

function getExampleValue(field: SchemaField): string {
  const examples: Record<string, string> = {
    string: 'example',
    number: '0',
    integer: '0',
    float: '0.0',
    boolean: 'true',
    email: 'user@example.com',
    url: 'https://example.com',
    date: '2024-01-01',
    datetime: '2024-01-01T00:00:00Z',
    array: '[]',
    object: '{}',
  }
  return examples[field.type] || 'value'
}

async function loadSource() {
  try {
    const result = await listSources(1, 100)
    const found = (result?.list || []).find((s: DataSource) => s.id === sourceId)
    if (found) {
      source.value = found
      if (found.schema_config) {
        try {
          const schema = typeof found.schema_config === 'string' ? JSON.parse(found.schema_config) : found.schema_config
          schemaFields.value = schema.fields || []
        } catch {
          schemaFields.value = []
        }
      }
    }
  } catch {
    // handled
  } finally {
    loading.value = false
  }
}

async function loadTokens() {
  try {
    const result = await listTokens(sourceId)
    tokens.value = result || []
  } catch {
    // handled
  }
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
    const result = await createToken(sourceId, data)
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

async function handleCopyCurl() {
  const ok = await copyToClipboard(curlExample.value)
  if (ok) {
    ElMessage.success('cURL 命令已复制到剪贴板')
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

onMounted(async () => {
  await loadSource()
  await loadTokens()
})
</script>

<style scoped>
.source-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
}

.source-name {
  font-size: 22px;
  font-weight: 700;
  color: #1f2937;
}

.source-desc {
  color: #6b7280;
  margin-top: 4px;
  font-size: 14px;
}

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

.curl-display {
  background: #111827;
  border-radius: 8px;
  padding: 16px;
  overflow-x: auto;
}

.curl-code {
  color: #e5e7eb;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.curl-tip {
  margin-top: 10px;
  font-size: 13px;
  color: #6b7280;
}

.curl-tip code {
  background: #f3f4f6;
  padding: 2px 6px;
  border-radius: 4px;
  color: #dc2626;
  font-size: 12px;
}
</style>

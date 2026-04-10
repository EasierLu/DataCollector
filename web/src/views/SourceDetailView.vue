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

        <!-- Webhook 配置 -->
        <div style="margin-top: 20px; border-top: 1px solid #f3f4f6; padding-top: 16px">
          <h4 style="font-size: 14px; color: #374151; margin-bottom: 12px">Webhook 配置</h4>
          <el-descriptions :column="3">
            <el-descriptions-item label="状态">
              <el-tag :type="source.webhook_enabled ? 'success' : 'info'" size="small">
                {{ source.webhook_enabled ? '已启用' : '未启用' }}
              </el-tag>
            </el-descriptions-item>
            <template v-if="source.webhook_enabled && webhookConfig">
              <el-descriptions-item label="URL">
                <code style="background: #f3f4f6; padding: 2px 8px; border-radius: 4px; font-family: monospace; word-break: break-all">
                  {{ webhookConfig.url }}
                </code>
              </el-descriptions-item>
              <el-descriptions-item label="HTTP 方法">
                <el-tag size="small">{{ webhookConfig.method }}</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="Secret">
                {{ webhookConfig.secret ? '••••••••' : '未设置' }}
              </el-descriptions-item>
              <el-descriptions-item label="超时">
                {{ webhookConfig.timeout || 10 }} 秒
              </el-descriptions-item>
              <el-descriptions-item label="重试策略">
                最多 {{ webhookConfig.retry_count ?? 3 }} 次，间隔 {{ webhookConfig.retry_interval || 5 }} 秒
              </el-descriptions-item>
            </template>
          </el-descriptions>
          <template v-if="source.webhook_enabled && webhookConfig">
            <div v-if="webhookHeadersList.length > 0" style="margin-top: 12px">
              <div style="font-size: 13px; color: #6b7280; margin-bottom: 8px">自定义 Headers</div>
              <el-table :data="webhookHeadersList" size="small" border>
                <el-table-column prop="key" label="名称" />
                <el-table-column prop="value" label="值" />
              </el-table>
            </div>
            <div v-if="webhookConfig.body_template" style="margin-top: 12px">
              <div style="font-size: 13px; color: #6b7280; margin-bottom: 8px">请求体模板</div>
              <div class="curl-display">
                <pre class="curl-code">{{ webhookConfig.body_template }}</pre>
              </div>
            </div>
          </template>
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
      <TokenManager ref="tokenManagerRef" :source-id="sourceId" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ArrowLeft } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { getSourceById } from '@/api/source'
import { formatDate } from '@/utils/format'
import { copyToClipboard } from '@/utils/clipboard'
import type { DataSource, SchemaField, WebhookConfig } from '@/types/source'
import TokenManager from '@/components/source/TokenManager.vue'

const route = useRoute()
const sourceId = Number(route.params.id)

if (isNaN(sourceId) || sourceId <= 0) {
  ElMessage.error('无效的数据源 ID')
}

const loading = ref(true)
const source = ref<Partial<DataSource>>({})
const schemaFields = ref<SchemaField[]>([])
const webhookConfig = ref<WebhookConfig | null>(null)
const webhookHeadersList = ref<{ key: string; value: string }[]>([])
const tokenManagerRef = ref<InstanceType<typeof TokenManager>>()

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

async function handleCopyCurl() {
  const ok = await copyToClipboard(curlExample.value)
  if (ok) {
    ElMessage.success('cURL 命令已复制到剪贴板')
  } else {
    ElMessage.error('复制失败，请手动复制')
  }
}

async function loadSource() {
  const found = await getSourceById(sourceId)
  source.value = found
  if (found.schema_config) {
    try {
      const schema = typeof found.schema_config === 'string' ? JSON.parse(found.schema_config) : found.schema_config
      schemaFields.value = schema.fields || []
    } catch {
      schemaFields.value = []
    }
  }
  // 解析 webhook 配置
  if (found.webhook_config) {
    try {
      const wc = typeof found.webhook_config === 'string' ? JSON.parse(found.webhook_config) : found.webhook_config
      webhookConfig.value = wc
      webhookHeadersList.value = wc.headers
        ? Object.entries(wc.headers).map(([key, value]) => ({ key, value: value as string }))
        : []
    } catch {
      webhookConfig.value = null
      webhookHeadersList.value = []
    }
  } else {
    webhookConfig.value = null
    webhookHeadersList.value = []
  }
}

onMounted(async () => {
  try {
    await loadSource()
    await tokenManagerRef.value?.loadTokens()
  } catch {
    ElMessage.error('加载数据源信息失败')
  } finally {
    loading.value = false
  }
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

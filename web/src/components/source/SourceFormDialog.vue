<template>
  <el-dialog v-model="visible" :title="isEditing ? '编辑数据源' : '创建数据源'" width="620px" destroy-on-close>
    <el-form :model="form" label-position="top">
      <el-form-item label="名称" required>
        <el-input v-model="form.name" placeholder="请输入数据源名称" />
      </el-form-item>
      <el-form-item label="描述">
        <el-input v-model="form.description" type="textarea" :rows="2" placeholder="请输入数据源描述" />
      </el-form-item>
      <el-form-item label="Schema 字段配置">
        <div style="width: 100%">
          <div v-for="(field, index) in form.schema_fields" :key="index" class="schema-field-row">
            <el-input v-model="field.name" placeholder="字段名" style="flex: 1" />
            <el-select v-model="field.type" style="width: 120px">
              <el-option label="字符串" value="string" />
              <el-option label="数字" value="number" />
              <el-option label="布尔" value="boolean" />
              <el-option label="数组" value="array" />
              <el-option label="对象" value="object" />
            </el-select>
            <el-checkbox v-model="field.required">必填</el-checkbox>
            <el-button text type="danger" @click="form.schema_fields.splice(index, 1)">
              <el-icon><Delete /></el-icon>
            </el-button>
          </div>
          <el-button text type="primary" @click="form.schema_fields.push({ name: '', type: 'string', required: false })">
            <el-icon class="el-icon--left"><Plus /></el-icon>添加字段
          </el-button>
          <div v-if="form.schema_fields.length === 0" class="empty-tip">暂无字段配置</div>
        </div>
      </el-form-item>
      <el-form-item label="限流配置（可选）">
        <div style="width: 100%; display: flex; gap: 12px">
          <div style="flex: 1">
            <div style="font-size: 12px; color: #6b7280; margin-bottom: 4px">每分钟请求数</div>
            <el-input-number v-model="form.rate_limit" :min="0" :max="100000" placeholder="0" style="width: 100%" />
          </div>
          <div style="flex: 1">
            <div style="font-size: 12px; color: #6b7280; margin-bottom: 4px">突发量</div>
            <el-input-number v-model="form.rate_limit_burst" :min="0" :max="10000" placeholder="0" style="width: 100%" />
          </div>
        </div>
        <div class="empty-tip">设为 0 表示使用系统设置中的全局默认值</div>
      </el-form-item>

      <!-- Webhook 配置 -->
      <el-divider content-position="left">Webhook 配置</el-divider>
      <el-form-item label="启用 Webhook">
        <el-switch v-model="form.webhook_enabled" active-text="启用" inactive-text="禁用" />
      </el-form-item>
      <template v-if="form.webhook_enabled">
        <el-form-item label="URL" required>
          <el-input v-model="form.webhook_config.url" placeholder="https://example.com/webhook" />
        </el-form-item>
        <el-form-item label="HTTP 方法">
          <el-select v-model="form.webhook_config.method" style="width: 100%">
            <el-option label="POST" value="POST" />
            <el-option label="GET" value="GET" />
            <el-option label="PUT" value="PUT" />
          </el-select>
        </el-form-item>
        <el-form-item label="Secret（HMAC 签名密钥）">
          <el-input v-model="form.webhook_config.secret" type="password" show-password placeholder="留空则不签名" />
        </el-form-item>
        <el-form-item label="超时 / 重试">
          <div style="width: 100%; display: flex; gap: 12px">
            <div style="flex: 1">
              <div style="font-size: 12px; color: #6b7280; margin-bottom: 4px">超时（秒）</div>
              <el-input-number v-model="form.webhook_config.timeout" :min="1" :max="60" style="width: 100%" />
            </div>
            <div style="flex: 1">
              <div style="font-size: 12px; color: #6b7280; margin-bottom: 4px">重试次数</div>
              <el-input-number v-model="form.webhook_config.retry_count" :min="0" :max="10" style="width: 100%" />
            </div>
            <div style="flex: 1">
              <div style="font-size: 12px; color: #6b7280; margin-bottom: 4px">重试间隔（秒）</div>
              <el-input-number v-model="form.webhook_config.retry_interval" :min="1" :max="300" style="width: 100%" />
            </div>
          </div>
        </el-form-item>
        <el-form-item label="自定义 Headers">
          <div style="width: 100%">
            <div v-for="(header, index) in webhookHeaders" :key="index" class="schema-field-row">
              <el-input v-model="header.key" placeholder="Header 名称" style="flex: 1" />
              <el-input v-model="header.value" placeholder="Header 值" style="flex: 1" />
              <el-button text type="danger" @click="webhookHeaders.splice(index, 1)">
                <el-icon><Delete /></el-icon>
              </el-button>
            </div>
            <el-button text type="primary" @click="webhookHeaders.push({ key: '', value: '' })">
              <el-icon class="el-icon--left"><Plus /></el-icon>添加 Header
            </el-button>
          </div>
        </el-form-item>
        <el-form-item label="请求体模板">
          <el-input
            v-model="form.webhook_config.body_template"
            type="textarea"
            :rows="4"
            placeholder="可用变量：.Event, .SourceID, .SourceName, .CollectID, .RecordID, .Data, .Timestamp"
          />
        </el-form-item>
      </template>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">取消</el-button>
      <el-button type="primary" :loading="saving" :disabled="!form.name.trim()" @click="handleSave">保存</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Plus, Delete } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { createSource, updateSource } from '@/api/source'
import { useSourceOptions } from '@/composables/useSourceOptions'
import type { DataSource, SchemaField, WebhookConfig } from '@/types/source'

const { invalidateSourceOptions } = useSourceOptions()

const emit = defineEmits<{
  saved: []
}>()

const visible = ref(false)
const isEditing = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)

interface WebhookHeaderEntry {
  key: string
  value: string
}
const webhookHeaders = ref<WebhookHeaderEntry[]>([])

function getDefaultWebhookConfig(): WebhookConfig {
  return {
    url: '',
    method: 'POST',
    headers: {},
    secret: '',
    timeout: 10,
    retry_count: 3,
    retry_interval: 5,
    body_template: '',
  }
}

const form = ref<{
  name: string
  description: string
  schema_fields: SchemaField[]
  rate_limit: number
  rate_limit_burst: number
  webhook_enabled: boolean
  webhook_config: WebhookConfig
}>({
  name: '',
  description: '',
  schema_fields: [],
  rate_limit: 0,
  rate_limit_burst: 0,
  webhook_enabled: false,
  webhook_config: getDefaultWebhookConfig(),
})

function openCreate() {
  isEditing.value = false
  editingId.value = null
  form.value = { name: '', description: '', schema_fields: [], rate_limit: 0, rate_limit_burst: 0, webhook_enabled: false, webhook_config: getDefaultWebhookConfig() }
  webhookHeaders.value = []
  visible.value = true
}

function openEdit(source: DataSource) {
  isEditing.value = true
  editingId.value = source.id
  let schemaFields: SchemaField[] = []
  if (source.schema_config) {
    try {
      const schema = typeof source.schema_config === 'string' ? JSON.parse(source.schema_config) : source.schema_config
      if (schema.fields && Array.isArray(schema.fields)) {
        schemaFields = schema.fields
      }
    } catch {
      // ignore
    }
  }
  const webhookConfig = source.webhook_config
    ? (typeof source.webhook_config === 'string' ? JSON.parse(source.webhook_config) : source.webhook_config)
    : getDefaultWebhookConfig()
  webhookHeaders.value = webhookConfig.headers
    ? Object.entries(webhookConfig.headers).map(([key, value]) => ({ key, value: value as string }))
    : []

  form.value = {
    name: source.name || '',
    description: source.description || '',
    schema_fields: schemaFields,
    rate_limit: source.rate_limit || 0,
    rate_limit_burst: source.rate_limit_burst || 0,
    webhook_enabled: !!source.webhook_enabled,
    webhook_config: { ...getDefaultWebhookConfig(), ...webhookConfig },
  }
  visible.value = true
}

async function handleSave() {
  if (!form.value.name.trim()) return
  saving.value = true
  const schemaConfig = { fields: form.value.schema_fields.filter((f) => f.name.trim() !== '') }
  const headersObj: Record<string, string> = {}
  for (const h of webhookHeaders.value) {
    if (h.key.trim()) {
      headersObj[h.key.trim()] = h.value
    }
  }
  const webhookConfig = form.value.webhook_enabled
    ? { ...form.value.webhook_config, headers: headersObj }
    : null

  const data = {
    name: form.value.name.trim(),
    description: form.value.description.trim(),
    schema_config: schemaConfig,
    rate_limit: form.value.rate_limit || 0,
    rate_limit_burst: form.value.rate_limit_burst || 0,
    webhook_enabled: form.value.webhook_enabled,
    webhook_config: webhookConfig,
  }
  try {
    if (isEditing.value && editingId.value) {
      await updateSource(editingId.value, data)
      ElMessage.success('更新成功')
    } else {
      await createSource(data)
      ElMessage.success('创建成功')
    }
    visible.value = false
    invalidateSourceOptions()
    emit('saved')
  } catch (err: any) {
    ElMessage.error(err.message || '保存失败')
  } finally {
    saving.value = false
  }
}

defineExpose({ openCreate, openEdit })
</script>

<style scoped>
.schema-field-row {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-bottom: 8px;
  background: #f9fafb;
  padding: 8px 12px;
  border-radius: 8px;
}

.empty-tip {
  color: #9ca3af;
  font-size: 13px;
  text-align: center;
  padding: 12px 0;
}
</style>

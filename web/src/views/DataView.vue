<template>
  <div>
    <!-- 页面标题 & 导出按钮 -->
    <div class="page-header">
      <h2>数据记录</h2>
      <div class="header-actions">
        <el-button type="success" :disabled="exporting || records.length === 0" @click="handleExport('csv')">导出 CSV</el-button>
        <el-button type="primary" :disabled="exporting || records.length === 0" @click="handleExport('json')">导出 JSON</el-button>
      </div>
    </div>

    <!-- 筛选栏 -->
    <el-card shadow="hover" style="margin-bottom: 20px">
      <el-form :inline="true" :model="filter">
        <el-form-item label="数据源">
          <el-select v-model="filter.source_id" clearable placeholder="全部数据源" style="width: 180px">
            <el-option v-for="s in sourceOptions" :key="s.id" :label="s.name" :value="s.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="开始日期">
          <el-date-picker v-model="filter.start_date" type="date" value-format="YYYY-MM-DD" placeholder="开始日期" />
        </el-form-item>
        <el-form-item label="结束日期">
          <el-date-picker v-model="filter.end_date" type="date" value-format="YYYY-MM-DD" placeholder="结束日期" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="applyFilter">搜索</el-button>
          <el-button @click="resetFilter">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <!-- 批量操作 -->
    <div v-if="records.length > 0" class="batch-bar">
      <el-checkbox :model-value="isAllSelected" @change="toggleSelectAll">全选</el-checkbox>
      <span v-if="selectedIds.length > 0" class="selected-count">已选择 {{ selectedIds.length }} 条记录</span>
      <el-button type="danger" size="small" :disabled="selectedIds.length === 0" @click="handleBatchDelete">批量删除</el-button>
    </div>

    <!-- 数据表格 -->
    <el-card shadow="hover" v-loading="loading">
      <el-table :data="records" stripe style="width: 100%" @selection-change="handleSelectionChange" ref="tableRef">
        <el-table-column type="selection" width="50" />
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column label="数据源" width="130">
          <template #default="{ row }">
            <el-tag size="small">{{ getSourceName(row.source_id) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="数据" min-width="250">
          <template #default="{ row }">
            <div class="data-cell" @click="toggleExpand(row.id)">
              <span class="data-summary">{{ getDataSummary(row.data) }}</span>
            </div>
            <div v-if="expandedIds.has(row.id)" class="data-expanded">
              <pre>{{ formatJSON(row.data) }}</pre>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="ip_address" label="IP 地址" width="140" />
        <el-table-column label="User-Agent" width="180">
          <template #default="{ row }">
            <el-tooltip :content="row.user_agent" placement="top" :show-after="500">
              <span class="text-ellipsis">{{ truncate(row.user_agent, 30) }}</span>
            </el-tooltip>
          </template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="100" fixed="right">
          <template #default="{ row }">
            <el-button text type="danger" @click="handleDeleteRecord(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && records.length === 0" description="暂无数据记录" />
      <div v-if="total > 0" class="pagination-bar">
        <el-pagination
          v-model:current-page="filter.page"
          :page-size="filter.size"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="loadRecords"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { queryData, deleteRecord, batchDeleteRecords, exportData } from '@/api/data'
import { listSources } from '@/api/source'
import { formatDate, truncate, formatJSON, getDataSummary } from '@/utils/format'
import { downloadBlob } from '@/utils/clipboard'
import type { DataRecord } from '@/types/record'
import type { DataSource } from '@/types/source'

const loading = ref(true)
const records = ref<DataRecord[]>([])
const total = ref(0)
const sourceOptions = ref<DataSource[]>([])
const selectedIds = ref<number[]>([])
const expandedIds = ref<Set<number>>(new Set())
const exporting = ref(false)

const filter = reactive({
  page: 1,
  size: 20,
  source_id: '' as string | number,
  start_date: '',
  end_date: '',
})

const isAllSelected = computed(() => records.value.length > 0 && selectedIds.value.length === records.value.length)

function handleSelectionChange(rows: DataRecord[]) {
  selectedIds.value = rows.map((r) => r.id)
}

function toggleSelectAll(val: any) {
  const tableRef = document.querySelector('.el-table') as any
  if (val) {
    records.value.forEach((_, index) => {
      tableRef?.__vue__?.toggleRowSelection(records.value[index], true)
    })
  } else {
    selectedIds.value = []
  }
}

function toggleExpand(id: number) {
  if (expandedIds.value.has(id)) {
    expandedIds.value.delete(id)
  } else {
    expandedIds.value.add(id)
  }
  // trigger reactivity
  expandedIds.value = new Set(expandedIds.value)
}

function getSourceName(sourceId: number): string {
  const s = sourceOptions.value.find((src) => src.id === sourceId)
  return s ? s.name : `ID: ${sourceId}`
}

async function loadSources() {
  try {
    const result = await listSources(1, 1000)
    sourceOptions.value = result?.list || []
  } catch {
    // handled
  }
}

async function loadRecords() {
  loading.value = true
  selectedIds.value = []
  expandedIds.value = new Set()
  try {
    const result = await queryData(filter)
    records.value = result?.list || []
    total.value = result?.total || 0
  } catch {
    // handled
  } finally {
    loading.value = false
  }
}

function applyFilter() {
  filter.page = 1
  loadRecords()
}

function resetFilter() {
  filter.page = 1
  filter.source_id = ''
  filter.start_date = ''
  filter.end_date = ''
  loadRecords()
}

async function handleDeleteRecord(record: DataRecord) {
  try {
    await ElMessageBox.confirm('确定要删除这条数据记录吗？此操作不可恢复。', '确认删除', {
      type: 'warning',
      confirmButtonText: '确认删除',
      confirmButtonClass: 'el-button--danger',
    })
    await deleteRecord(record.id)
    ElMessage.success('删除成功')
    await loadRecords()
  } catch (err: any) {
    if (err !== 'cancel' && err?.message) {
      ElMessage.error(err.message)
    }
  }
}

async function handleBatchDelete() {
  if (selectedIds.value.length === 0) return
  try {
    await ElMessageBox.confirm(`确定要删除选中的 ${selectedIds.value.length} 条数据记录吗？此操作不可恢复。`, '确认批量删除', {
      type: 'warning',
      confirmButtonText: '确认删除',
      confirmButtonClass: 'el-button--danger',
    })
    const result = await batchDeleteRecords(selectedIds.value)
    ElMessage.success(`成功删除 ${result?.count || selectedIds.value.length} 条记录`)
    selectedIds.value = []
    await loadRecords()
  } catch (err: any) {
    if (err !== 'cancel' && err?.message) {
      ElMessage.error(err.message)
    }
  }
}

async function handleExport(format: 'csv' | 'json') {
  exporting.value = true
  try {
    const result = await exportData({
      format,
      source_id: filter.source_id || undefined,
      start_date: filter.start_date || undefined,
      end_date: filter.end_date || undefined,
    })
    downloadBlob(result.blob, result.filename)
    ElMessage.success('导出成功')
  } catch (err: any) {
    ElMessage.error(err.message || '导出失败')
  } finally {
    exporting.value = false
  }
}

onMounted(async () => {
  await loadSources()
  await loadRecords()
})
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 20px;
}

.page-header h2 {
  font-size: 22px;
  font-weight: 700;
  color: #1f2937;
}

.header-actions {
  display: flex;
  gap: 8px;
}

.batch-bar {
  display: flex;
  align-items: center;
  gap: 16px;
  background: #f9fafb;
  padding: 10px 16px;
  border-radius: 8px;
  margin-bottom: 12px;
}

.selected-count {
  font-size: 13px;
  color: #6b7280;
}

.pagination-bar {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}

.data-cell {
  cursor: pointer;
  color: #4f46e5;
}

.data-cell:hover {
  text-decoration: underline;
}

.data-summary {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
  max-width: 300px;
}

.data-expanded {
  margin-top: 8px;
  background: #111827;
  border-radius: 8px;
  padding: 12px;
  overflow-x: auto;
}

.data-expanded pre {
  color: #34d399;
  font-size: 12px;
  font-family: monospace;
  margin: 0;
  white-space: pre-wrap;
}

.text-ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
  max-width: 160px;
}
</style>

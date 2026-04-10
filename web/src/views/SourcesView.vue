<template>
  <div>
    <div class="page-header">
      <h2>数据源管理</h2>
      <el-button type="primary" @click="formDialogRef?.openCreate()">
        <el-icon class="el-icon--left"><Plus /></el-icon>创建数据源
      </el-button>
    </div>

    <el-card shadow="hover" v-loading="loading">
      <el-table :data="sources" stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="collect_id" label="采集标识" width="120">
          <template #default="{ row }">
            <code style="font-size: 12px">{{ row.collect_id }}</code>
          </template>
        </el-table-column>
        <el-table-column prop="name" label="名称" min-width="150" />
        <el-table-column label="状态" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'danger'" size="small">{{ row.status === 1 ? '启用' : '禁用' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="token_count" label="Token 数量" width="110" />
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="160" fixed="right">
          <template #default="{ row }">
            <el-button text type="primary" @click="$router.push(`/sources/${row.id}`)">查看详情</el-button>
            <el-dropdown trigger="click">
              <el-button text>
                <el-icon><More /></el-icon>
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="formDialogRef?.openEdit(row)">编辑</el-dropdown-item>
                  <el-dropdown-item divided @click="handleDelete(row)">
                    <span style="color: #ef4444">删除</span>
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && sources.length === 0" description="暂无数据源，点击「创建数据源」按钮添加" />
      <div v-if="total > 0" class="pagination-bar">
        <el-pagination
          v-model:current-page="page"
          :page-size="size"
          :total="total"
          layout="total, prev, pager, next"
          @current-change="loadSources"
        />
      </div>
    </el-card>

    <SourceFormDialog ref="formDialogRef" @saved="loadSources" />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Plus, More } from '@element-plus/icons-vue'
import { ElMessageBox, ElMessage } from 'element-plus'
import { listSources, deleteSource } from '@/api/source'
import { formatDate } from '@/utils/format'
import { useSourceOptions } from '@/composables/useSourceOptions'
import type { DataSource } from '@/types/source'
import SourceFormDialog from '@/components/source/SourceFormDialog.vue'

const { invalidateSourceOptions } = useSourceOptions()

const loading = ref(true)
const sources = ref<DataSource[]>([])
const page = ref(1)
const size = 10
const total = ref(0)

const formDialogRef = ref<InstanceType<typeof SourceFormDialog>>()

async function loadSources() {
  loading.value = true
  try {
    const result = await listSources(page.value, size)
    sources.value = result?.list || []
    total.value = result?.total || 0
  } catch {
    // handled by interceptor
  } finally {
    loading.value = false
  }
}

async function handleDelete(source: DataSource) {
  try {
    await ElMessageBox.confirm(`确定要删除数据源 "${source.name}" 吗？此操作不可恢复，该数据源下的所有 Token 也将被删除。`, '确认删除', {
      type: 'warning',
      confirmButtonText: '确认删除',
      confirmButtonClass: 'el-button--danger',
    })
    await deleteSource(source.id)
    ElMessage.success('删除成功')
    invalidateSourceOptions()
    await loadSources()
  } catch (err: any) {
    if (err !== 'cancel' && err?.message) {
      ElMessage.error(err.message)
    }
  }
}

onMounted(loadSources)
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

.pagination-bar {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
</style>

<template>
  <div v-loading="loading">
    <!-- 统计卡片 -->
    <el-row :gutter="20" style="margin-bottom: 20px">
      <el-col :xs="12" :sm="6" v-for="card in statCards" :key="card.label">
        <el-card shadow="hover" class="stat-card">
          <div class="stat-card-body">
            <div>
              <div class="stat-label">{{ card.label }}</div>
              <div class="stat-value">{{ card.value }}</div>
            </div>
            <el-icon :size="36" :color="card.color"><component :is="card.icon" /></el-icon>
          </div>
          <div class="stat-footer">{{ card.footer }}</div>
        </el-card>
      </el-col>
    </el-row>

    <!-- WebSocket 状态 -->
    <el-card shadow="hover" style="margin-bottom: 20px">
      <div class="ws-status-bar">
        <div class="ws-status-left">
          <span class="ws-dot" :class="wsConnected ? 'connected' : 'disconnected'" />
          <span>实时连接状态: {{ wsConnected ? '已连接' : '未连接' }}</span>
        </div>
        <el-button text type="primary" @click="reconnect">
          <el-icon class="el-icon--left"><Refresh /></el-icon>重连
        </el-button>
      </div>
    </el-card>

    <!-- 数据趋势图表 -->
    <el-card shadow="hover" style="margin-bottom: 20px" v-loading="trendLoading">
      <template #header>
        <div class="card-header">
          <span class="card-title">数据趋势</span>
        </div>
      </template>
      <div class="trend-filter-bar">
        <el-radio-group v-model="timeRange" size="small">
          <el-radio-button value="today">今日</el-radio-button>
          <el-radio-button value="7d">7天</el-radio-button>
          <el-radio-button value="30d">30天</el-radio-button>
          <el-radio-button value="custom">自定义</el-radio-button>
        </el-radio-group>
        <el-date-picker
          v-if="timeRange === 'custom'"
          v-model="customDateRange"
          type="daterange"
          start-placeholder="开始日期"
          end-placeholder="结束日期"
          value-format="YYYY-MM-DD"
          size="small"
          style="width: 260px"
        />
        <el-select
          v-model="trendSourceId"
          placeholder="全部数据源"
          clearable
          size="small"
          style="width: 160px"
        >
          <el-option
            v-for="s in sourceOptions"
            :key="s.id"
            :label="s.name"
            :value="s.id"
          />
        </el-select>
        <el-select
          v-model="trendTokenId"
          placeholder="全部 Token"
          clearable
          size="small"
          style="width: 160px"
          :disabled="!trendSourceId"
        >
          <el-option
            v-for="t in tokenOptions"
            :key="t.id"
            :label="t.name"
            :value="t.id"
          />
        </el-select>
      </div>
      <div v-if="filledTrendData.length > 0" class="trend-chart">
        <v-chart :option="chartOption" autoresize />
      </div>
      <el-empty v-else-if="!trendLoading" description="暂无趋势数据" />
    </el-card>

    <!-- 最近数据记录 -->
    <el-card shadow="hover">
      <template #header>
        <div class="card-header">
          <span class="card-title">最近数据记录</span>
          <router-link to="/data" class="view-all">查看全部 &rarr;</router-link>
        </div>
      </template>
      <el-table :data="recentRecords" stripe style="width: 100%">
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="source_id" label="数据源" width="120">
          <template #default="{ row }">
            <el-tag size="small">ID: {{ row.source_id }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="数据摘要" min-width="200">
          <template #default="{ row }">
            <span class="text-ellipsis">{{ getDataSummary(row.data) }}</span>
          </template>
        </el-table-column>
        <el-table-column prop="ip_address" label="IP 地址" width="140" />
        <el-table-column label="创建时间" width="180">
          <template #default="{ row }">{{ formatDate(row.created_at) }}</template>
        </el-table-column>
      </el-table>
      <el-empty v-if="!loading && recentRecords.length === 0" description="暂无数据记录" />
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import { Calendar, TrendCharts, DataBoard, Connection, Refresh } from '@element-plus/icons-vue'
import { use } from 'echarts/core'
import { LineChart } from 'echarts/charts'
import { GridComponent, TooltipComponent } from 'echarts/components'
import { CanvasRenderer } from 'echarts/renderers'
import VChart from 'vue-echarts'
import { getDashboard, getDashboardTrend } from '@/api/dashboard'
import { listSources } from '@/api/source'
import { listTokens } from '@/api/token'
import { useWebSocket } from '@/composables/useWebSocket'
import { formatDate, getDataSummary } from '@/utils/format'
import type { DataRecord } from '@/types/record'
import type { DataSource } from '@/types/source'
import type { DataToken } from '@/types/token'
import type { TrendPoint } from '@/types/dashboard'

use([LineChart, GridComponent, TooltipComponent, CanvasRenderer])

const loading = ref(true)
const todayCount = ref(0)
const weekCount = ref(0)
const monthCount = ref(0)
const totalSources = ref(0)
const recentRecords = ref<DataRecord[]>([])

const statCards = computed(() => [
  { label: '今日数据量', value: todayCount.value, icon: Calendar, color: '#2563eb', footer: '实时更新' },
  { label: '本周数据量', value: weekCount.value, icon: TrendCharts, color: '#4f46e5', footer: '本周累计' },
  { label: '本月数据量', value: monthCount.value, icon: DataBoard, color: '#7c3aed', footer: '本月累计' },
  { label: '数据源总数', value: totalSources.value, icon: Connection, color: '#059669', footer: '' },
])

// WebSocket
const { connected: wsConnected, connect, disconnect, onMessage } = useWebSocket(() => {
  const protocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = localStorage.getItem('jwt_token')
  return `${protocol}//${location.host}/api/v1/admin/ws/monitor?token=${token}`
})

onMessage((message: any) => {
  if (message.type === 'stats_update') {
    if (message.data.today_count !== undefined) todayCount.value = message.data.today_count
    if (message.data.week_count !== undefined) weekCount.value = message.data.week_count
    if (message.data.month_count !== undefined) monthCount.value = message.data.month_count
  }
})

function reconnect() {
  disconnect()
  connect()
}

// 趋势图表
const trendLoading = ref(false)
const trendData = ref<TrendPoint[]>([])
const timeRange = ref('7d')
const customDateRange = ref<string[]>([])
const trendSourceId = ref<number | undefined>(undefined)
const trendTokenId = ref<number | undefined>(undefined)
const sourceOptions = ref<DataSource[]>([])
const tokenOptions = ref<DataToken[]>([])

function getDateRange(): { start_date: string; end_date: string } | null {
  const today = new Date()
  const fmt = (d: Date) => d.toISOString().slice(0, 10)

  if (timeRange.value === 'today') {
    const s = fmt(today)
    return { start_date: s, end_date: s }
  }
  if (timeRange.value === '7d') {
    const start = new Date(today)
    start.setDate(start.getDate() - 6)
    return { start_date: fmt(start), end_date: fmt(today) }
  }
  if (timeRange.value === '30d') {
    const start = new Date(today)
    start.setDate(start.getDate() - 29)
    return { start_date: fmt(start), end_date: fmt(today) }
  }
  if (timeRange.value === 'custom' && customDateRange.value?.length === 2) {
    return { start_date: customDateRange.value[0], end_date: customDateRange.value[1] }
  }
  return null
}

// 日期填充：确保 x 轴连续
const filledTrendData = computed(() => {
  const range = getDateRange()
  if (!range) return []
  const dataMap = new Map<string, number>()
  for (const p of trendData.value) {
    dataMap.set(p.date, p.count)
  }
  const result: TrendPoint[] = []
  const current = new Date(range.start_date + 'T00:00:00')
  const end = new Date(range.end_date + 'T00:00:00')
  while (current <= end) {
    const dateStr = current.toISOString().slice(0, 10)
    result.push({ date: dateStr, count: dataMap.get(dateStr) || 0 })
    current.setDate(current.getDate() + 1)
  }
  return result
})

const chartOption = computed(() => ({
  tooltip: {
    trigger: 'axis',
    formatter: (params: any) => {
      const p = params[0]
      return `${p.axisValue}<br/>数据量: <b>${p.value}</b>`
    },
  },
  grid: {
    left: 50,
    right: 20,
    top: 20,
    bottom: 30,
  },
  xAxis: {
    type: 'category',
    data: filledTrendData.value.map((p) => p.date),
    axisLabel: {
      fontSize: 11,
      color: '#6b7280',
    },
    axisLine: { lineStyle: { color: '#e5e7eb' } },
  },
  yAxis: {
    type: 'value',
    minInterval: 1,
    axisLabel: {
      fontSize: 11,
      color: '#6b7280',
    },
    splitLine: { lineStyle: { color: '#f3f4f6' } },
  },
  series: [
    {
      type: 'line',
      data: filledTrendData.value.map((p) => p.count),
      smooth: true,
      symbol: 'circle',
      symbolSize: 6,
      lineStyle: { width: 2.5, color: '#4f46e5' },
      itemStyle: { color: '#4f46e5' },
      areaStyle: {
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: 'rgba(79,70,229,0.25)' },
            { offset: 1, color: 'rgba(79,70,229,0.02)' },
          ],
        },
      },
    },
  ],
}))

async function loadTrend() {
  const range = getDateRange()
  if (!range) return
  trendLoading.value = true
  try {
    const params: any = { ...range }
    if (trendSourceId.value) params.source_id = trendSourceId.value
    if (trendTokenId.value) params.token_id = trendTokenId.value
    trendData.value = (await getDashboardTrend(params)) || []
  } catch {
    trendData.value = []
  } finally {
    trendLoading.value = false
  }
}

async function loadSourceOptions() {
  try {
    const result = await listSources(1, 1000)
    sourceOptions.value = result?.list || []
  } catch {
    sourceOptions.value = []
  }
}

async function loadTokenOptions(sourceId: number) {
  try {
    tokenOptions.value = (await listTokens(sourceId)) || []
  } catch {
    tokenOptions.value = []
  }
}

watch(timeRange, () => {
  if (timeRange.value !== 'custom') loadTrend()
})
watch(customDateRange, () => {
  if (timeRange.value === 'custom' && customDateRange.value?.length === 2) loadTrend()
})
watch(trendSourceId, (val) => {
  trendTokenId.value = undefined
  tokenOptions.value = []
  if (val) loadTokenOptions(val)
  loadTrend()
})
watch(trendTokenId, () => {
  loadTrend()
})

onMounted(async () => {
  try {
    const data = await getDashboard()
    todayCount.value = data.today_count || 0
    weekCount.value = data.week_count || 0
    monthCount.value = data.month_count || 0
    totalSources.value = data.total_sources || 0
    recentRecords.value = data.recent_records || []
  } catch {
    // handled by interceptor
  } finally {
    loading.value = false
  }
  connect()
  loadSourceOptions()
  loadTrend()
})

onUnmounted(() => {
  disconnect()
})
</script>

<style scoped>
.stat-card {
  margin-bottom: 12px;
}

.stat-card-body {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.stat-label {
  font-size: 13px;
  color: #6b7280;
}

.stat-value {
  font-size: 28px;
  font-weight: 700;
  color: #1f2937;
  margin-top: 4px;
}

.stat-footer {
  font-size: 12px;
  color: #9ca3af;
  margin-top: 12px;
}

.ws-status-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.ws-status-left {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 14px;
  color: #4b5563;
}

.ws-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  display: inline-block;
}

.ws-dot.connected {
  background: #22c55e;
  animation: pulse 2s infinite;
}

.ws-dot.disconnected {
  background: #ef4444;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
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

.view-all {
  font-size: 13px;
  color: #4f46e5;
  text-decoration: none;
}

.text-ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: block;
  max-width: 300px;
}

.trend-filter-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
  margin-bottom: 16px;
}

.trend-chart {
  height: 350px;
}
</style>

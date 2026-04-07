import type { DataRecord } from './record'

export interface DashboardStats {
  today_count: number
  week_count: number
  month_count: number
  total_sources: number
  recent_records: DataRecord[]
}

export interface TrendPoint {
  date: string
  count: number
}

export interface TrendParams {
  start_date: string
  end_date: string
  source_id?: number
  token_id?: number
}

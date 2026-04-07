import request from './request'
import type { DashboardStats, TrendPoint, TrendParams } from '@/types/dashboard'

export function getDashboard(): Promise<DashboardStats> {
  return request.get('/api/v1/admin/dashboard')
}

export function getDashboardTrend(params: TrendParams): Promise<TrendPoint[]> {
  return request.get('/api/v1/admin/dashboard/trend', { params })
}

import request from './request'
import type { DataSource, CreateSourceRequest, UpdateSourceRequest } from '@/types/source'
import type { PageResult } from '@/types/api'

export function listSources(page: number, size: number): Promise<PageResult<DataSource>> {
  return request
    .get('/api/v1/admin/sources', { params: { page, size } })
    .then((data: any) => ({
      total: data.total,
      list: data.items ?? [],
    }))
}

export function getSourceById(id: number): Promise<DataSource> {
  return request.get(`/api/v1/admin/sources/${id}`)
}

export function createSource(data: CreateSourceRequest): Promise<DataSource> {
  return request.post('/api/v1/admin/sources', data)
}

export function updateSource(id: number, data: UpdateSourceRequest): Promise<DataSource> {
  return request.put(`/api/v1/admin/sources/${id}`, data)
}

export function deleteSource(id: number): Promise<void> {
  return request.delete(`/api/v1/admin/sources/${id}`)
}

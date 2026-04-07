import request from './request'
import type { DataRecord, RecordFilter } from '@/types/record'
import type { PageResult } from '@/types/api'

export function queryData(filter: RecordFilter): Promise<PageResult<DataRecord>> {
  return request.get('/api/v1/admin/data', { params: filter })
}

export function deleteRecord(id: number): Promise<void> {
  return request.delete(`/api/v1/admin/data/${id}`)
}

export function batchDeleteRecords(ids: number[]): Promise<{ message: string; count: number }> {
  return request.post('/api/v1/admin/data/batch-delete', { ids })
}

export function exportData(params: {
  format: 'csv' | 'json'
  source_id?: number | string
  start_date?: string
  end_date?: string
}): Promise<{ blob: Blob; filename: string }> {
  return request
    .get('/api/v1/admin/data/export', {
      params,
      responseType: 'blob',
    })
    .then((response: any) => {
      const disposition = response.headers?.['content-disposition'] || ''
      const match = disposition.match(/filename="(.+)"/)
      const filename = match ? match[1] : `export.${params.format}`
      return { blob: response.data, filename }
    })
}

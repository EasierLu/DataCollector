import request from './request'

export interface ApiKey {
  id: number
  name: string
  permissions: string
  expires_at: string | null
  last_used_at: string | null
  created_by: number
  created_at: string
}

export interface CreateApiKeyRequest {
  name: string
  permissions: string[]
  expires_at?: string
}

export interface CreateApiKeyResponse {
  id: number
  key: string
  name: string
  permissions: string
  expires_at: string | null
  created_at: string
}

export function listApiKeys(): Promise<ApiKey[]> {
  return request.get('/api/v1/admin/settings/api-keys')
}

export function listPermissions(): Promise<string[]> {
  return request.get('/api/v1/admin/settings/api-keys/permissions')
}

export function createApiKey(data: CreateApiKeyRequest): Promise<CreateApiKeyResponse> {
  return request.post('/api/v1/admin/settings/api-keys', data)
}

export function updateApiKeyPermissions(id: number, permissions: string[]): Promise<void> {
  return request.put(`/api/v1/admin/settings/api-keys/${id}/permissions`, { permissions })
}

export function deleteApiKey(id: number): Promise<void> {
  return request.delete(`/api/v1/admin/settings/api-keys/${id}`)
}

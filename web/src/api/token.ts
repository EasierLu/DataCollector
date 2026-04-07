import request from './request'
import type { DataToken, CreateTokenRequest, CreateTokenResponse } from '@/types/token'

export function listTokens(sourceId: number): Promise<DataToken[]> {
  return request.get(`/api/v1/admin/sources/${sourceId}/tokens`)
}

export function createToken(sourceId: number, data: CreateTokenRequest): Promise<CreateTokenResponse> {
  return request.post(`/api/v1/admin/sources/${sourceId}/tokens`, data)
}

export function updateTokenStatus(tokenId: number, status: number): Promise<void> {
  return request.put(`/api/v1/admin/tokens/${tokenId}/status`, { status })
}

export function deleteToken(tokenId: number): Promise<void> {
  return request.delete(`/api/v1/admin/tokens/${tokenId}`)
}

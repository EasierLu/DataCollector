import request from './request'

export interface HealthResponse {
  status: string
  version: string
  uptime: string
  database: string
}

export function healthCheck(): Promise<HealthResponse> {
  return request.get('/api/v1/health')
}

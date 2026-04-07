import request from './request'

export interface RateLimitSettings {
  rate_limit_per_ip: number
  rate_limit_per_ip_burst: number
  rate_limit_per_token: number
  rate_limit_per_token_burst: number
}

export function getRateLimitSettings(): Promise<RateLimitSettings> {
  return request.get('/api/v1/admin/settings/rate-limit')
}

export function updateRateLimitSettings(data: RateLimitSettings): Promise<RateLimitSettings> {
  return request.put('/api/v1/admin/settings/rate-limit', data)
}

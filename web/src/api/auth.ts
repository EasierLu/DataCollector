import request from './request'

export interface LoginRequest {
  username: string
  password: string
}

export interface LoginResponse {
  token: string
  expires_in: number
}

export function login(data: LoginRequest): Promise<LoginResponse> {
  return request.post('/api/v1/admin/login', data)
}

export function refreshToken(): Promise<LoginResponse> {
  return request.post('/api/v1/admin/refresh-token')
}

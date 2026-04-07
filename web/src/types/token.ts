export interface DataToken {
  id: number
  source_id: number
  name: string
  status: number
  expires_at: string | null
  last_used_at: string | null
  created_by: number
  created_at: string
}

export interface CreateTokenRequest {
  name: string
  expires_at?: string
}

export interface CreateTokenResponse {
  id: number
  token: string
  name: string
  status: number
  expires_at: string | null
  created_at: string
}

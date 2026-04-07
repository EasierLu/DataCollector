export interface DataRecord {
  id: number
  source_id: number
  token_id: number
  data: any
  ip_address: string
  user_agent: string
  created_at: string
}

export interface RecordFilter {
  source_id?: number | string
  start_date?: string
  end_date?: string
  page: number
  size: number
}

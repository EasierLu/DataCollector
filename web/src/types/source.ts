export interface SchemaField {
  name: string
  type: string
  required: boolean
  max_length?: number
  min_length?: number
  pattern?: string
}

export interface SchemaConfig {
  fields: SchemaField[]
}

export interface DataSource {
  id: number
  collect_id: string
  name: string
  description: string
  schema_config: SchemaConfig | null
  status: number
  created_by: number
  created_at: string
  updated_at: string
  token_count?: number
  rate_limit: number
  rate_limit_burst: number
}

export interface CreateSourceRequest {
  name: string
  description: string
  schema_config?: SchemaConfig
  rate_limit?: number
  rate_limit_burst?: number
}

export interface UpdateSourceRequest {
  name: string
  description: string
  schema_config?: SchemaConfig
  rate_limit?: number
  rate_limit_burst?: number
}

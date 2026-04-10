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

export interface WebhookConfig {
  url: string
  method: string        // POST | GET | PUT
  headers: Record<string, string>
  secret: string
  timeout: number       // 默认10
  retry_count: number   // 默认3
  retry_interval: number // 默认5
  body_template: string
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
  webhook_enabled: boolean
  webhook_config: WebhookConfig | null
}

export interface CreateSourceRequest {
  name: string
  description: string
  schema_config?: SchemaConfig
  rate_limit?: number
  rate_limit_burst?: number
  webhook_enabled?: boolean
  webhook_config?: WebhookConfig | null
}

export interface UpdateSourceRequest {
  name: string
  description: string
  schema_config?: SchemaConfig
  rate_limit?: number
  rate_limit_burst?: number
  webhook_enabled?: boolean
  webhook_config?: WebhookConfig | null
}

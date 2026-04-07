import request from './request'

export interface SetupStatus {
  initialized: boolean
}

export interface TestDbRequest {
  driver: string
  host?: string
  port?: number
  user?: string
  password?: string
  dbname?: string
}

export interface InitializeRequest {
  database: {
    driver: string
    sqlite: { path: string }
    postgres: {
      host: string
      port: number
      user: string
      password: string
      dbname: string
      sslmode: string
    }
  }
  server: { port: number }
  admin: {
    username: string
    password: string
  }
}

export function checkSetupStatus(): Promise<SetupStatus> {
  return request.get('/api/v1/setup/status')
}

export function testDatabase(data: TestDbRequest): Promise<{ message: string }> {
  return request.post('/api/v1/setup/test-db', data)
}

export function initialize(data: InitializeRequest): Promise<{ message: string }> {
  return request.post('/api/v1/setup/init', data)
}

export function reinitialize(confirm: string): Promise<{ message: string }> {
  return request.post('/api/v1/setup/reinit', { confirm })
}

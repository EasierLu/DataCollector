export interface ApiResponse<T = any> {
  code: number
  message: string
  data?: T
  errors?: any
}

export interface PageResult<T = any> {
  total: number
  list: T[]
}

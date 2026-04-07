import axios from 'axios'
import type { ApiResponse } from '@/types/api'
import router from '@/router'

const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || '',
  timeout: 15000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器：附加 JWT
request.interceptors.request.use((config) => {
  const token = localStorage.getItem('jwt_token')
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

// 响应拦截器：解包响应、处理 401
request.interceptors.response.use(
  (response) => {
    // 导出接口返回 blob，直接返回
    if (response.config.responseType === 'blob') {
      return response
    }
    const res = response.data as ApiResponse
    if (res.code !== 0) {
      // 业务错误
      return Promise.reject(new Error(res.message || '请求失败'))
    }
    return res.data
  },
  (error) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('jwt_token')
      router.push('/login')
    }
    const msg = error.response?.data?.message || error.message || '网络错误'
    return Promise.reject(new Error(msg))
  },
)

export default request

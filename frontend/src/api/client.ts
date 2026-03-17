/**
 * API 客户端封装
 * - 自动注入 JWT token 到请求头
 * - 统一错误处理（401 自动跳转登录）
 * - 统一响应格式解析
 */
import axios from 'axios'
import type { AxiosInstance, InternalAxiosRequestConfig, AxiosResponse } from 'axios'

// 后端统一响应格式
export interface ApiResponse<T = unknown> {
  code: number      // 0=成功，-1=失败
  message: string   // 提示信息
  data?: T          // 响应数据
}

// 创建 axios 实例
const client: AxiosInstance = axios.create({
  baseURL: '/api/v1',           // 所有请求自动加前缀
  timeout: 30000,               // 30秒超时
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器：自动注入 token
client.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('token')
    if (token && config.headers) {
      config.headers.Authorization = `Bearer ${token}`
    }
    return config
  },
  (error) => Promise.reject(error)
)

// 响应拦截器：统一错误处理
client.interceptors.response.use(
  (response: AxiosResponse<ApiResponse>) => {
    // 业务层错误（code !== 0）
    const data = response.data
    if (data.code !== 0) {
      return Promise.reject(new Error(data.message || '请求失败'))
    }
    return response
  },
  (error) => {
    if (error.response) {
      const status = error.response.status
      const message = error.response.data?.message || '请求失败'

      // 401：token 过期或无效，清除登录态并跳转
      if (status === 401) {
        localStorage.removeItem('token')
        localStorage.removeItem('user')
        // 避免在登录页循环跳转
        if (window.location.pathname !== '/login') {
          window.location.href = '/login'
        }
      }

      return Promise.reject(new Error(message))
    }

    // 网络错误
    return Promise.reject(new Error('网络连接失败，请检查网络'))
  }
)

export default client

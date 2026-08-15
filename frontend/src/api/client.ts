/**
 * API 客户端封装
 * - 自动注入 JWT token 到请求头
 * - 统一错误处理（401 自动跳转登录）
 * - 统一响应格式解析
 * - Blob/ArrayBuffer 二进制响应不套业务JSON信封校验
 * - FormData 请求自动移除全局 JSON Content-Type，由浏览器生成 multipart boundary
 *
 * v46修复：全局超时从30秒改为120秒，避免大数据量pages请求超时。
 *
 * Word保真上传修复：
 * axios实例为普通JSON请求保留application/json默认值；
 * 请求体是FormData时，必须在请求拦截器中删除该默认值。
 * 否则Axios会把FormData按JSON序列化，Go后端无法执行ParseMultipartForm。
 */

import axios from 'axios'
import type {
  AxiosInstance,
  AxiosResponse,
  InternalAxiosRequestConfig,
} from 'axios'

// 后端统一响应格式
export interface ApiResponse<T = unknown> {
  code: number
  message: string
  data?: T
}

// 创建 axios 实例
const client: AxiosInstance = axios.create({
  baseURL: '/api/v1',
  timeout: 120000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 请求拦截器：自动注入token，并正确处理multipart请求头。
client.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    const token = localStorage.getItem('token')

    if (token && config.headers) {
      config.headers.Authorization =
        `Bearer ${token}`
    }

    /**
     * 浏览器必须自行生成：
     *
     * Content-Type:
     * multipart/form-data; boundary=...
     *
     * 不能沿用Axios实例的application/json，也不能手写一个没有boundary的
     * multipart/form-data。AxiosHeaders.delete执行大小写不敏感删除。
     */
    const formDataRequest =
      typeof FormData !== 'undefined' &&
      config.data instanceof FormData

    if (
      formDataRequest &&
      config.headers
    ) {
      config.headers.delete('Content-Type')
    }

    return config
  },
  error => Promise.reject(error),
)

function isBinaryResponse(
  response: AxiosResponse,
): boolean {
  if (
    response.config.responseType === 'blob' ||
    response.config.responseType ===
      'arraybuffer'
  ) {
    return true
  }

  if (
    typeof Blob !== 'undefined' &&
    response.data instanceof Blob
  ) {
    return true
  }

  return response.data instanceof ArrayBuffer
}

async function readResponseErrorMessage(
  data: unknown,
): Promise<string> {
  if (
    typeof Blob !== 'undefined' &&
    data instanceof Blob
  ) {
    try {
      const raw = await data.text()
      const parsed = raw
        ? JSON.parse(raw) as {
            message?: unknown
          }
        : null

      if (
        parsed &&
        typeof parsed.message === 'string' &&
        parsed.message.trim()
      ) {
        return parsed.message.trim()
      }
    } catch {
      return '请求失败'
    }
  }

  if (
    data &&
    typeof data === 'object' &&
    'message' in data &&
    typeof (
      data as {
        message?: unknown
      }
    ).message === 'string'
  ) {
    return (
      data as {
        message: string
      }
    ).message || '请求失败'
  }

  return '请求失败'
}

// 响应拦截器：统一错误处理
client.interceptors.response.use(
  (response: AxiosResponse) => {
    if (isBinaryResponse(response)) {
      return response
    }

    const data =
      response.data as ApiResponse

    if (data.code !== 0) {
      return Promise.reject(
        new Error(
          data.message || '请求失败',
        ),
      )
    }

    return response
  },
  async error => {
    if (
      axios.isCancel(error) ||
      error?.code === 'ERR_CANCELED'
    ) {
      return Promise.reject(error)
    }

    if (error.response) {
      const status = error.response.status
      const message =
        await readResponseErrorMessage(
          error.response.data,
        )

      if (status === 401) {
        localStorage.removeItem('token')
        localStorage.removeItem('user')

        if (
          window.location.pathname !==
          '/login'
        ) {
          window.location.href = '/login'
        }
      }

      return Promise.reject(
        new Error(message),
      )
    }

    if (
      error.code === 'ECONNABORTED' ||
      error.code === 'ETIMEDOUT'
    ) {
      return Promise.reject(
        new Error(
          '请求处理超时，请稍后重试',
        ),
      )
    }

    if (
      typeof navigator !== 'undefined' &&
      navigator.onLine === false
    ) {
      return Promise.reject(
        new Error(
          '当前设备网络已断开，请恢复网络后重试',
        ),
      )
    }

    if (error.code === 'ERR_NETWORK') {
      return Promise.reject(
        new Error(
          '暂时无法连接服务，请稍后重试',
        ),
      )
    }

    return Promise.reject(
      new Error(
        '请求未获得服务器响应，请稍后重试',
      ),
    )
  },
)

export default client

/**
 * 认证 Provider 逻辑 Hook
 * - 应用启动时从 localStorage 恢复登录态
 * - 调用 /auth/me 验证 token 是否仍然有效
 * - 提供 login / logout 方法给子组件使用
 */
import { useState, useEffect, useCallback } from 'react'
import { getMe } from '@/api/auth'
import type { UserInfo } from '@/api/auth'
import type { AuthContextType } from '@/store/auth'

export function useAuthProvider(): AuthContextType {
  const [user, setUser] = useState<UserInfo | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  // 登录成功：保存 token 和用户信息到状态 + localStorage
  const login = useCallback((newToken: string, newUser: UserInfo) => {
    setToken(newToken)
    setUser(newUser)
    localStorage.setItem('token', newToken)
    localStorage.setItem('user', JSON.stringify(newUser))
  }, [])

  // 登出：清除所有登录态
  const logout = useCallback(() => {
    setToken(null)
    setUser(null)
    localStorage.removeItem('token')
    localStorage.removeItem('user')
  }, [])

  // 应用启动时：尝试恢复登录态
  useEffect(() => {
    const init = async () => {
      const savedToken = localStorage.getItem('token')
      if (!savedToken) {
        setIsLoading(false)
        return
      }

      try {
        // 用保存的 token 调用 /auth/me 验证有效性
        setToken(savedToken)
        const userInfo = await getMe()
        setUser(userInfo)
      } catch {
        // token 无效或过期，清除登录态
        localStorage.removeItem('token')
        localStorage.removeItem('user')
        setToken(null)
      } finally {
        setIsLoading(false)
      }
    }

    init()
  }, [])

  return { user, token, isLoading, login, logout }
}

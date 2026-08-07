/**
 * useAuthProvider — 认证状态与持久化
 *
 * 启动规则：
 *   - 只要存在token，一律调用/auth/me获取数据库当前真值；
 *   - 不直接使用localStorage.user恢复运行态；
 *   - /auth/me成功后重新写入localStorage.user，清理旧字段和旧教育域信息。
 *
 * 教育域隔离：
 *   - 登录、退出、/auth/me刷新时清理课程目录缓存；
 *   - 避免同一浏览器切换K12、职教或成人账号后复用上一账号课程。
 */

import {
  useCallback,
  useEffect,
  useState,
} from 'react'
import { getMe } from '@/api/auth'
import type { UserInfo } from '@/api/auth'
import type { AuthContextType } from '@/store/auth'
import { resetSubjectCache } from '@/hooks/useSubjects'

export function useAuthProvider(): AuthContextType {
  const [user, setUser] = useState<UserInfo | null>(null)
  const [token, setToken] = useState<string | null>(null)
  const [isLoading, setIsLoading] = useState(true)

  const login = useCallback((
    newToken: string,
    newUser: UserInfo,
  ) => {
    localStorage.setItem('token', newToken)
    localStorage.setItem('user', JSON.stringify(newUser))

    setToken(newToken)
    setUser(newUser)

    resetSubjectCache()

    /**
     * 登录响应只用于立即建立登录态。
     *
     * 随后重新调用/auth/me读取数据库当前真值，确保角色、学校、
     * 教育域、教育域就绪状态和组织信息不是登录响应中的历史快照。
     *
     * 请求返回前若用户已经退出或切换账号，则通过token一致性判断
     * 丢弃迟到响应，防止旧账号信息覆盖新账号运行态。
     */
    void getMe()
      .then(userInfo => {
        if (
          localStorage.getItem('token') !==
          newToken
        ) {
          return
        }

        localStorage.setItem(
          'user',
          JSON.stringify(userInfo),
        )

        setUser(userInfo)
        resetSubjectCache()
      })
      .catch(() => {
        /**
         * 登录请求已经成功时，不因后续资料刷新暂时失败
         * 立即覆盖原登录结果。
         *
         * 401等失效情况仍由全局HTTP拦截器清理登录态。
         */
      })
  }, [])

  const logout = useCallback(() => {
    localStorage.removeItem('token')
    localStorage.removeItem('user')

    setToken(null)
    setUser(null)

    resetSubjectCache()
  }, [])

  useEffect(() => {
    let cancelled = false

    const init = async () => {
      const savedToken = localStorage.getItem('token')

      if (!savedToken) {
        localStorage.removeItem('user')
        resetSubjectCache()

        if (!cancelled) {
          setToken(null)
          setUser(null)
          setIsLoading(false)
        }
        return
      }

      try {
        setToken(savedToken)

        const userInfo = await getMe()
        if (cancelled) return

        setUser(userInfo)

        // 用/auth/me的数据库真值覆盖历史缓存。
        localStorage.setItem(
          'user',
          JSON.stringify(userInfo),
        )

        resetSubjectCache()
      } catch {
        localStorage.removeItem('token')
        localStorage.removeItem('user')
        resetSubjectCache()

        if (!cancelled) {
          setToken(null)
          setUser(null)
        }
      } finally {
        if (!cancelled) {
          setIsLoading(false)
        }
      }
    }

    init()

    return () => {
      cancelled = true
    }
  }, [])

  return {
    user,
    token,
    isLoading,
    login,
    logout,
  }
}

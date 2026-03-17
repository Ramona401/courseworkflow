/**
 * 认证状态管理
 * - 使用 React Context 管理登录态
 * - 提供 login / logout / 初始化 方法
 * - 全局共享用户信息和 token
 */
import { createContext, useContext } from 'react'
import type { UserInfo } from '@/api/auth'

// 认证上下文类型
export interface AuthContextType {
  user: UserInfo | null          // 当前用户（null 表示未登录）
  token: string | null           // JWT token
  isLoading: boolean             // 初始化加载中
  login: (token: string, user: UserInfo) => void   // 登录成功后调用
  logout: () => void             // 登出
}

// 创建上下文（默认值仅用于类型推导）
export const AuthContext = createContext<AuthContextType>({
  user: null,
  token: null,
  isLoading: true,
  login: () => {},
  logout: () => {},
})

// 便捷 Hook：获取认证上下文
export function useAuth(): AuthContextType {
  return useContext(AuthContext)
}

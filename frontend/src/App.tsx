/**
 * App 根组件
 * - AuthProvider 包裹全局，提供认证状态
 * - 路由配置：登录页 / 主布局（含侧边栏）
 * - 路由守卫：未登录跳转登录页
 * - 角色守卫：按角色控制页面访问
 * - P3-1新增：外部数据配置路由
 */
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthContext } from '@/store/auth'
import { useAuth } from '@/store/auth'
import { useAuthProvider } from '@/hooks/useAuthProvider'

// 布局和页面
import MainLayout from '@/components/layout/MainLayout'
import LoginPage from '@/pages/login/LoginPage'
import DashboardPage from '@/pages/dashboard/DashboardPage'
import UsersPage from '@/pages/users/UsersPage'
import AIConfigPage from '@/pages/ai-config/AIConfigPage'
import PromptsPage from '@/pages/prompts/PromptsPage'
import ExternalDataPage from '@/pages/external-data/ExternalDataPage'
import CoursesPage from '@/pages/courses/CoursesPage'

/**
 * 路由守卫组件
 * - 加载中显示 loading
 * - 未登录重定向到 /login
 * - 已登录正常渲染子组件
 */
function AuthGuard({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth()

  // 初始化加载中（正在验证 token）
  if (isLoading) {
    return (
      <div style={{
        height: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        background: '#f5f5f7',
      }}>
        <div style={{ textAlign: 'center' }}>
          <div style={{
            width: '32px',
            height: '32px',
            border: '2px solid #007aff',
            borderTopColor: 'transparent',
            borderRadius: '50%',
            animation: 'spin 0.8s linear infinite',
            margin: '0 auto 12px',
          }} />
          <div style={{ color: '#8e8e93', fontSize: '14px' }}>加载中...</div>
        </div>
      </div>
    )
  }

  // 未登录，跳转登录页
  if (!user) {
    return <Navigate to="/login" replace />
  }

  // 已登录，渲染子组件
  return <>{children}</>
}

/**
 * 角色守卫组件
 * - 检查当前用户角色是否在允许列表中
 * - 不匹配则重定向到首页
 */
function RoleGuard({ children, roles }: { children: React.ReactNode; roles: string[] }) {
  const { user } = useAuth()

  if (!user || !roles.includes(user.role)) {
    return <Navigate to="/" replace />
  }

  return <>{children}</>
}

export default function App() {
  // 初始化认证状态（从 localStorage 恢复 + 验证 token）
  const authValue = useAuthProvider()

  return (
    <AuthContext.Provider value={authValue}>
      <BrowserRouter>
        <Routes>
          {/* 登录页（不需要认证） */}
          <Route path="/login" element={<LoginPage />} />

          {/* 主布局（需要认证） */}
          <Route
            path="/"
            element={
              <AuthGuard>
                <MainLayout />
              </AuthGuard>
            }
          >
            {/* 仪表盘（首页） */}
            <Route index element={<DashboardPage />} />

            {/* P1-4 用户管理（仅admin） */}
            <Route path="users" element={
              <RoleGuard roles={['admin']}>
                <UsersPage />
              </RoleGuard>
            } />

            {/* P2-1 AI配置中心（仅admin） */}
            <Route path="ai-config" element={
              <RoleGuard roles={['admin']}>
                <AIConfigPage />
              </RoleGuard>
            } />

            {/* P2-3 提示词管理（仅admin） */}
            <Route path="prompts" element={
              <RoleGuard roles={['admin']}>
                <PromptsPage />
              </RoleGuard>
            } />

            {/* P3-1 外部数据配置（仅admin） */}
            <Route path="external-data" element={
              <RoleGuard roles={['admin']}>
                <ExternalDataPage />
              </RoleGuard>
            } />

            {/* P3-3 课程管理（admin + operator） */}
            <Route path="courses" element={
              <RoleGuard roles={['admin', 'operator']}>
                <CoursesPage />
              </RoleGuard>
            } />

            {/* TODO: P4 Pipeline */}
            {/* <Route path="pipelines" element={<PipelinesPage />} /> */}

            {/* TODO: P6 审核中心 */}
            {/* <Route path="review" element={<ReviewPage />} /> */}

            {/* 未匹配路由 -> 重定向到首页 */}
            <Route path="*" element={<Navigate to="/" replace />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </AuthContext.Provider>
  )
}

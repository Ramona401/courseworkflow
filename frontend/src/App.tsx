/**
 * App 根组件
 * v5.1：新增统一用户管理中心 /admin（独立路由，admin专属）
 */
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { AuthContext } from '@/store/auth'
import { useAuth } from '@/store/auth'
import { useAuthProvider } from '@/hooks/useAuthProvider'

/* ==================== 课件审核系统 ==================== */
import MainLayout from '@/components/layout/MainLayout'
import LoginPage from '@/pages/login/LoginPage'
import PortalPage from '@/pages/portal/PortalPage'
import DashboardPage from '@/pages/dashboard/DashboardPage'
import UsersPage from '@/pages/users/UsersPage'
import AIConfigPage from '@/pages/ai-config/AIConfigPage'
import PromptsPage from '@/pages/prompts/PromptsPage'
import ExternalDataPage from '@/pages/external-data/ExternalDataPage'
import CoursesPage from '@/pages/courses/CoursesPage'
import PipelinesPage from '@/pages/pipelines/PipelinesPage'
import PipelineDetailPage from '@/pages/pipelines/PipelineDetailPage'
import PipelineReviewPage from '@/pages/pipelines/PipelineReviewPage'
import ReviewCenterPage from '@/pages/review/ReviewCenterPage'
import SettingsPage from '@/pages/settings/SettingsPage'

/* ==================== 教案系统 ==================== */
import LPLayout from '@/components/layout-lp/LPLayout'
import WorkshopPage from '@/pages/lesson-plans/workshop/WorkshopPage'
import MyPlansPage from '@/pages/lesson-plans/my-plans/MyPlansPage'
import LibraryPage from '@/pages/lesson-plans/library/LibraryPage'
import ComponentsPage from '@/pages/lesson-plans/components/ComponentsPage'
import TemplatesPage from '@/pages/lesson-plans/templates/TemplatesPage'
import TemplateEditorPage from '@/pages/lesson-plans/templates/TemplateEditorPage'
import PlanDetailPage from '@/pages/lesson-plans/plan-detail/PlanDetailPage'
import ReviewCenterLPPage from '@/pages/lesson-plans/review/ReviewCenterLPPage'

/* ==================== 通用独立页面 ==================== */
import AccountPage from '@/pages/account/AccountPage'
import AICenterPage from '@/pages/ai-center/AICenterPage'
// v5.1 新增：统一用户管理中心
import AdminPage from '@/pages/admin/AdminPage'

/* ==================== 路由守卫 ==================== */
function AuthGuard({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth()
  if (isLoading) {
    return (
      <div style={{ height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center', background: '#FAFBFC' }}>
        <div style={{ textAlign: 'center' }}>
          <div style={{ width: '32px', height: '32px', border: '2px solid #4F7BE8', borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite', margin: '0 auto 12px' }} />
          <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
          <div style={{ color: '#9CA3AF', fontSize: '14px' }}>加载中...</div>
        </div>
      </div>
    )
  }
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

function RoleGuard({ children, roles }: { children: React.ReactNode; roles: string[] }) {
  const { user } = useAuth()
  if (!user || !roles.includes(user.role)) return <Navigate to="/" replace />
  return <>{children}</>
}

export default function App() {
  const authValue = useAuthProvider()
  return (
    <AuthContext.Provider value={authValue}>
      <BrowserRouter>
        <Routes>
          {/* 登录页 */}
          <Route path="/login" element={<LoginPage />} />

          {/* 入口选择页 */}
          <Route path="/" element={<AuthGuard><PortalPage /></AuthGuard>} />

          {/* ==================== 通用独立页面 ==================== */}
          {/* 个人中心：所有已登录用户 */}
          <Route path="/account" element={<AuthGuard><AccountPage /></AuthGuard>} />

          {/* AI管理中心：仅admin */}
          <Route path="/ai-center" element={
            <AuthGuard><RoleGuard roles={['admin']}><AICenterPage /></RoleGuard></AuthGuard>
          } />

          {/* 用户管理中心：仅admin */}
          <Route path="/admin" element={
            <AuthGuard><RoleGuard roles={['admin']}><AdminPage /></RoleGuard></AuthGuard>
          } />

          {/* ==================== 课件审核系统 ==================== */}
          <Route path="/workflow" element={<AuthGuard><MainLayout /></AuthGuard>}>
            <Route index element={<DashboardPage />} />
            <Route path="users"         element={<RoleGuard roles={['admin']}><UsersPage /></RoleGuard>} />
            <Route path="ai-config"     element={<RoleGuard roles={['admin']}><AIConfigPage /></RoleGuard>} />
            <Route path="prompts"       element={<RoleGuard roles={['admin']}><PromptsPage /></RoleGuard>} />
            <Route path="external-data" element={<RoleGuard roles={['admin']}><ExternalDataPage /></RoleGuard>} />
            <Route path="courses"       element={<RoleGuard roles={['admin','operator','senior_operator']}><CoursesPage /></RoleGuard>} />
            <Route path="pipelines"     element={<RoleGuard roles={['admin','operator','senior_operator']}><PipelinesPage /></RoleGuard>} />
            <Route path="pipelines/:id" element={<RoleGuard roles={['admin','operator','senior_operator']}><PipelineDetailPage /></RoleGuard>} />
            <Route path="pipelines/:id/review" element={<RoleGuard roles={['admin','operator','senior_operator']}><PipelineReviewPage /></RoleGuard>} />
            <Route path="review"        element={<RoleGuard roles={['admin','operator','senior_operator']}><ReviewCenterPage /></RoleGuard>} />
            <Route path="settings"      element={<RoleGuard roles={['admin']}><SettingsPage /></RoleGuard>} />
          </Route>

          {/* ==================== 教案系统 ==================== */}
          <Route path="/lesson-plans" element={<AuthGuard><LPLayout /></AuthGuard>}>
            <Route index element={<WorkshopPage />} />
            <Route path="my-plans"      element={<MyPlansPage />} />
            <Route path="library"       element={<LibraryPage />} />
            <Route path="plans/:id"     element={<PlanDetailPage />} />
            <Route path="review"        element={<ReviewCenterLPPage />} />
            <Route path="components"    element={<ComponentsPage />} />
            <Route path="templates"     element={<TemplatesPage />} />
            <Route path="templates/:id" element={<TemplateEditorPage />} />
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthContext.Provider>
  )
}

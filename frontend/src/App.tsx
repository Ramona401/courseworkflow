/**
 * App 根组件
 * v7.0(迭代7)：新增课本管理页面路由 /lesson-plans/textbooks
 * v6.0(迭代3)：新增前测页面路由 /lesson-plans/assessment
 * v5.3：新增配方编辑器路由
 * v5.2：新增备课配方列表路由
 * v5.1：新增统一用户管理中心 /admin
 * v106新增：/lesson-plans/review/:id 独立全屏评审工作台（脱离LPLayout，全屏布局）
 * v110新增：/school-admin 学校管理员管理中心（senior_operator专属）
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
// v127新增：多级审核工作台
import ReviewV2DashboardPage from '@/pages/lesson-plans/review-v2/ReviewV2DashboardPage'
// v128新增：Token积分管理
import TokenDashboardPage from '@/pages/tokens/TokenDashboardPage'
// v106新增：独立全屏评审工作台
import ReviewWorkbenchPage from '@/pages/lesson-plans/review/ReviewWorkbenchPage'
// Phase 7A新增：备课配方
import RecipesPage from '@/pages/lesson-plans/recipes/RecipesPage'
import RecipeEditorPage from '@/pages/lesson-plans/recipes/RecipeEditorPage'
// v79新增：配方创建向导
import RecipeWizardPage from '@/pages/lesson-plans/recipes/RecipeWizardPage'
import StagesConfigPage from '@/pages/lesson-plans/stages-config/StagesConfigPage'
// 迭代3新增：教学风格前测
import AssessmentPage from '@/pages/lesson-plans/assessment/AssessmentPage'
// 迭代7新增：课本管理
import TextbooksPage from '@/pages/lesson-plans/textbooks/TextbooksPage'

/* ==================== 课件工坊 ==================== */
import CWLayout from '@/components/layout-cw/CWLayout'
import CoursewareListPage from '@/pages/courseware/CoursewareListPage'
import CWComponentsPage from '@/pages/courseware/CWComponentsPage'
import CWTemplatesPage from '@/pages/courseware/CWTemplatesPage'
import CoursewareWorkshopPage from '@/pages/courseware/CoursewareWorkshopPage'

/* ==================== 通用独立页面 ==================== */
import AccountPage from '@/pages/account/AccountPage'
import AICenterPage from '@/pages/ai-center/AICenterPage'
import AITraceDashboardPage from '@/pages/ai-traces/AITraceDashboardPage'
import AdminPage from '@/pages/admin/AdminPage'
// v110新增：学校管理员管理中心
import SchoolAdminPage from '@/pages/account/SchoolAdminPage'

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
          <Route path="/account" element={<AuthGuard><AccountPage /></AuthGuard>} />

          {/* v110新增：学校管理员管理中心
              仅要求登录+senior_operator角色，是否真正绑定学校由页面内部校验 */}
          <Route path="/school-admin" element={
            <AuthGuard>
              <RoleGuard roles={['senior_operator']}>
                <SchoolAdminPage />
              </RoleGuard>
            </AuthGuard>
          } />

          <Route path="/ai-center" element={
            <AuthGuard><RoleGuard roles={['admin']}><AICenterPage /></RoleGuard></AuthGuard>
          } />
          <Route path="/admin" element={
            <AuthGuard><RoleGuard roles={['admin','senior_operator']}><AdminPage /></RoleGuard></AuthGuard>
          } />
          <Route path="/ai-traces" element={
            <AuthGuard><RoleGuard roles={['admin']}><AITraceDashboardPage /></RoleGuard></AuthGuard>
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

          {/* ==================== 课件工坊 ==================== */}
          <Route path="/courseware" element={<AuthGuard><CWLayout /></AuthGuard>}>
            <Route index element={<CoursewareListPage />} />
            <Route path="components" element={<CWComponentsPage />} />
            <Route path="templates" element={<CWTemplatesPage />} />
            <Route path=":id" element={<CoursewareWorkshopPage />} />
          </Route>

          {/* ==================== 教案系统 ==================== */}

          {/* v106新增：独立全屏评审工作台（必须在 /lesson-plans 布局路由之前注册） */}
          <Route path="/lesson-plans/review/:id" element={
            <AuthGuard><ReviewWorkbenchPage /></AuthGuard>
          } />

          <Route path="/lesson-plans" element={<AuthGuard><LPLayout /></AuthGuard>}>
            <Route index element={<WorkshopPage />} />
            <Route path="my-plans"         element={<MyPlansPage />} />
            <Route path="library"          element={<LibraryPage />} />
            <Route path="plans/:id"        element={<PlanDetailPage />} />
            <Route path="review"           element={<ReviewCenterLPPage />} />
            <Route path="review-v2"        element={<ReviewV2DashboardPage />} />
            <Route path="tokens"           element={<TokenDashboardPage />} />
            <Route path="components"       element={<ComponentsPage />} />
            <Route path="templates"        element={<TemplatesPage />} />
            <Route path="templates/:id"    element={<TemplateEditorPage />} />
            <Route path="recipes"          element={<RecipesPage />} />
            <Route path="recipes/wizard"   element={<RecipeWizardPage />} />
            <Route path="recipes/new"      element={<RecipeEditorPage />} />
            <Route path="recipes/:id/edit" element={<RecipeEditorPage />} />
            <Route path="stages-config"    element={<RoleGuard roles={['admin']}><StagesConfigPage /></RoleGuard>} />
            <Route path="assessment"       element={<AssessmentPage />} />
            <Route path="textbooks"        element={<TextbooksPage />} />
          </Route>

          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      </BrowserRouter>
    </AuthContext.Provider>
  )
}

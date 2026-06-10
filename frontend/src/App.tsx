/**
 * App 根组件 — v140 代码分割版
 *
 * 改动：所有页面组件改为 React.lazy 动态导入，按路由懒加载。
 * 效果：首屏只加载当前路由的 chunk，其他路由按需加载。
 *
 * v172新增：ModuleGuard 门户板块守卫
 *   - 给 /workflow（课件审核）加一层组织板块开关守卫
 *   - 即便登录用户直接敲 URL，所属学校未开通该板块也会被重定向回首页
 *   - 与门户卡片显隐配套（光藏入口不够，必须守卫路由）
 *
 * Phase6.2新增（区域管理员）：
 *   - /admin 路由 RoleGuard 角色白名单加入 region_admin，
 *     使区域管理员可进入统一用户管理中心（数据由后端 ResolveDataScope 收窄到辖区）
 *
 * 合并重构改动（本次，废弃 SchoolAdminPage 并轨）：
 *   - /school-admin 路由不再渲染 SchoolAdminPage，改为 <Navigate to="/admin" replace/>
 *     （senior_operator 统一走 /admin 本校视角；保留旧路径重定向，防止存量书签/链接 404）。
 *   - 移除 SchoolAdminPage 的 lazy import（其文件将在后续清理批次删除）。
 *
 * 知识库压缩入库系统（Phase 6 前端）新增：
 *   - 新增隐藏全屏路由 /kb-admin/curriculum（课标压缩入库主页面）。
 *     参照 /lesson-plans/review/:id 的脱离布局全屏模式——只套 AuthGuard，不进任何 Layout，
 *     不出现在 PortalPage 入口卡片（隐藏功能），仅授权人员经直接 URL 访问。
 *     真正的访问拦截靠后端 RequireKBAuthorized 白名单中间件，前端守卫仅做体验优化不是安全边界。
 *
 * 分包策略（Vite 自动按 dynamic import 边界拆分）：
 *   - 主 chunk：路由框架 + 布局组件 + 守卫
 *   - 课件审核板块 chunk
 *   - 教案系统板块 chunk
 *   - 课件工坊板块 chunk
 *   - admin/配置类 chunk
 *   - 各独立页面各自 chunk
 */
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { Suspense, lazy, Component, type ReactNode, type ErrorInfo } from 'react'
import { AuthContext } from '@/store/auth'
import { useAuth } from '@/store/auth'
import { useAuthProvider } from '@/hooks/useAuthProvider'

/* ==================== 布局组件（非懒加载，每个板块必须立即可用） ==================== */
import MainLayout from '@/components/layout/MainLayout'
import LPLayout from '@/components/layout-lp/LPLayout'
import CWLayout from '@/components/layout-cw/CWLayout'

/* ==================== 课件审核系统（懒加载） ==================== */
const LoginPage = lazy(() => import('@/pages/login/LoginPage'))
const PortalPage = lazy(() => import('@/pages/portal/PortalPage'))
const DashboardPage = lazy(() => import('@/pages/dashboard/DashboardPage'))
const UsersPage = lazy(() => import('@/pages/users/UsersPage'))
const AIConfigPage = lazy(() => import('@/pages/ai-config/AIConfigPage'))
const PromptsPage = lazy(() => import('@/pages/prompts/PromptsPage'))
const ExternalDataPage = lazy(() => import('@/pages/external-data/ExternalDataPage'))
const CoursesPage = lazy(() => import('@/pages/courses/CoursesPage'))
const PipelinesPage = lazy(() => import('@/pages/pipelines/PipelinesPage'))
const PipelineDetailPage = lazy(() => import('@/pages/pipelines/PipelineDetailPage'))
const PipelineReviewPage = lazy(() => import('@/pages/pipelines/PipelineReviewPage'))
const ReviewCenterPage = lazy(() => import('@/pages/review/ReviewCenterPage'))
const SettingsPage = lazy(() => import('@/pages/settings/SettingsPage'))

/* ==================== 教案系统（懒加载） ==================== */
const WorkshopPage = lazy(() => import('@/pages/lesson-plans/workshop/WorkshopPage'))
const MyPlansPage = lazy(() => import('@/pages/lesson-plans/my-plans/MyPlansPage'))
const LibraryPage = lazy(() => import('@/pages/lesson-plans/library/LibraryPage'))
const ComponentsPage = lazy(() => import('@/pages/lesson-plans/components/ComponentsPage'))
const TemplatesPage = lazy(() => import('@/pages/lesson-plans/templates/TemplatesPage'))
const TemplateEditorPage = lazy(() => import('@/pages/lesson-plans/templates/TemplateEditorPage'))
const PlanDetailPage = lazy(() => import('@/pages/lesson-plans/plan-detail/PlanDetailPage'))
const ReviewCenterLPPage = lazy(() => import('@/pages/lesson-plans/review/ReviewCenterLPPage'))
const ReviewV2DashboardPage = lazy(() => import('@/pages/lesson-plans/review-v2/ReviewV2DashboardPage'))
const TokenDashboardPage = lazy(() => import('@/pages/tokens/TokenDashboardPage'))
const ReviewWorkbenchPage = lazy(() => import('@/pages/lesson-plans/review/ReviewWorkbenchPage'))
const RecipesPage = lazy(() => import('@/pages/lesson-plans/recipes/RecipesPage'))
const RecipeEditorPage = lazy(() => import('@/pages/lesson-plans/recipes/RecipeEditorPage'))
const RecipeWizardPage = lazy(() => import('@/pages/lesson-plans/recipes/RecipeWizardPage'))
const StagesConfigPage = lazy(() => import('@/pages/lesson-plans/stages-config/StagesConfigPage'))
const AssessmentPage = lazy(() => import('@/pages/lesson-plans/assessment/AssessmentPage'))
const TextbooksPage = lazy(() => import('@/pages/lesson-plans/textbooks/TextbooksPage'))

/* ==================== 课件工坊（懒加载） ==================== */
const CoursewareListPage = lazy(() => import('@/pages/courseware/CoursewareListPage'))
const CWComponentsPage = lazy(() => import('@/pages/courseware/CWComponentsPage'))
const CWTemplatesPage = lazy(() => import('@/pages/courseware/CWTemplatesPage'))
const CoursewareWorkshopPage = lazy(() => import('@/pages/courseware/CoursewareWorkshopPage'))

/* ==================== 知识库压缩入库系统（隐藏全屏，懒加载） ==================== */
const KBCurriculumPage = lazy(() => import('@/pages/kb-admin/KBCurriculumPage'))

/* ==================== 通用独立页面（懒加载） ==================== */
const AccountPage = lazy(() => import('@/pages/account/AccountPage'))
const AICenterPage = lazy(() => import('@/pages/ai-center/AICenterPage'))
const AITraceDashboardPage = lazy(() => import('@/pages/ai-traces/AITraceDashboardPage'))
const AdminPage = lazy(() => import('@/pages/admin/AdminPage'))

/* ==================== 路由加载错误边界 ==================== */
interface EBProps { children: ReactNode }
interface EBState { hasError: boolean }

class RouteErrorBoundary extends Component<EBProps, EBState> {
  constructor(props: EBProps) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(_error: Error): EBState {
    return { hasError: true }
  }

  componentDidCatch(error: Error, info: ErrorInfo) {
    console.error('[RouteErrorBoundary]', error, info)
  }

  render() {
    if (this.state.hasError) {
      return (
        <div style={{
          height: "100vh", display: "flex", alignItems: "center", justifyContent: "center",
          background: "#FAFBFC",
        }}>
          <div style={{ textAlign: "center", maxWidth: "400px", padding: "0 20px" }}>
            <div style={{ fontSize: "48px", marginBottom: "16px" }}>😵</div>
            <div style={{ fontSize: "18px", fontWeight: 700, color: "#1F2937", marginBottom: "8px" }}>
              页面加载失败
            </div>
            <div style={{ fontSize: "13px", color: "#6B7280", marginBottom: "20px", lineHeight: 1.6 }}>
              可能是网络波动导致资源加载失败，请刷新页面重试。
            </div>
            <button
              onClick={() => window.location.reload()}
              style={{
                padding: "10px 28px", borderRadius: "10px", border: "none",
                background: "linear-gradient(135deg, #4F7BE8, #6366F1)",
                color: "#fff", fontSize: "14px", fontWeight: 600, cursor: "pointer",
              }}
            >
              刷新页面
            </button>
          </div>
        </div>
      )
    }
    return this.props.children
  }
}

/* ==================== 全局加载占位符 ==================== */
function PageLoading() {
  return (
    <div style={{
      height: '100vh', display: 'flex', alignItems: 'center', justifyContent: 'center',
      background: '#FAFBFC',
    }}>
      <div style={{ textAlign: 'center' }}>
        <div style={{
          width: '28px', height: '28px',
          border: '2.5px solid #E5E7EB', borderTopColor: '#4F7BE8',
          borderRadius: '50%', animation: 'spin 0.8s linear infinite',
          margin: '0 auto 10px',
        }} />
        <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
        <div style={{ color: '#9CA3AF', fontSize: '13px' }}>页面加载中...</div>
      </div>
    </div>
  )
}

/* ==================== 路由守卫 ==================== */
function AuthGuard({ children }: { children: React.ReactNode }) {
  const { user, isLoading } = useAuth()
  if (isLoading) return <PageLoading />
  if (!user) return <Navigate to="/login" replace />
  return <>{children}</>
}

function RoleGuard({ children, roles }: { children: React.ReactNode; roles: string[] }) {
  const { user } = useAuth()
  if (!user || !roles.includes(user.role)) return <Navigate to="/" replace />
  return <>{children}</>
}

/**
 * ModuleGuard 门户板块守卫（v172）
 * 校验用户所属组织是否开通了指定板块（portal_modules[moduleKey]）。
 * 规则：
 *   - admin 永远放行（后端也会下发全开，这里双保险）
 *   - portal_modules 缺失 / 该 key 缺失 / 值非 false → 放行（缺省可见，不波及存量）
 *   - 仅显式 false → 重定向回首页
 * 注意：必须套在 AuthGuard 内层使用（依赖 user 已存在）。
 */
function ModuleGuard({ children, moduleKey }: { children: React.ReactNode; moduleKey: string }) {
  const { user } = useAuth()
  if (!user) return <Navigate to="/" replace />
  if (user.role === 'admin') return <>{children}</>
  const modules = user.portal_modules
  const enabled = !modules || !(moduleKey in modules) || modules[moduleKey] !== false
  if (!enabled) return <Navigate to="/" replace />
  return <>{children}</>
}

/* ==================== 主路由 ==================== */
export default function App() {
  const authValue = useAuthProvider()
  return (
    <AuthContext.Provider value={authValue}>
      <BrowserRouter>
        <RouteErrorBoundary>
        <Suspense fallback={<PageLoading />}>
          <Routes>
            {/* 登录页 */}
            <Route path="/login" element={<LoginPage />} />

            {/* 入口选择页 */}
            <Route path="/" element={<AuthGuard><PortalPage /></AuthGuard>} />

            {/* ==================== 通用独立页面 ==================== */}
            <Route path="/account" element={<AuthGuard><AccountPage /></AuthGuard>} />

            {/* 合并重构：/school-admin 旧路径重定向到 /admin（senior 走统一用户管理中心本校视角） */}
            <Route path="/school-admin" element={<Navigate to="/admin" replace />} />

            <Route path="/ai-center" element={
              <AuthGuard><RoleGuard roles={['admin']}><AICenterPage /></RoleGuard></AuthGuard>
            } />
            {/* Phase6.2：/admin 加入 region_admin（区域管理员） */}
            <Route path="/admin" element={
              <AuthGuard><RoleGuard roles={['admin','senior_operator','region_admin']}><AdminPage /></RoleGuard></AuthGuard>
            } />
            <Route path="/ai-traces" element={
              <AuthGuard><RoleGuard roles={['admin']}><AITraceDashboardPage /></RoleGuard></AuthGuard>
            } />

            {/* ==================== 知识库压缩入库（隐藏全屏，脱离任何布局；仅授权人员经直接URL访问） ==================== */}
            {/* 参照 /lesson-plans/review/:id 模式：只套 AuthGuard，不进 Layout，不出现在门户入口卡片。
                真正的访问拦截靠后端 RequireKBAuthorized 白名单中间件，前端守卫仅体验优化非安全边界。 */}
            <Route path="/kb-admin/curriculum" element={
              <AuthGuard><KBCurriculumPage /></AuthGuard>
            } />

            {/* ==================== 课件审核系统（v172：叠加 ModuleGuard 板块守卫） ==================== */}
            <Route path="/workflow" element={
              <AuthGuard><ModuleGuard moduleKey="workflow"><MainLayout /></ModuleGuard></AuthGuard>
            }>
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

            {/* 独立全屏评审工作台（必须在 /lesson-plans 布局路由之前注册） */}
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
        </Suspense>
        </RouteErrorBoundary>
      </BrowserRouter>
    </AuthContext.Provider>
  )
}

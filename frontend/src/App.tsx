/**
 * App 根组件 — v140 代码分割版
 *
 * 改动：所有页面组件改为 React.lazy 动态导入，按路由懒加载。
 * 效果：首屏只加载当前路由的 chunk，其他路由按需加载。
 *
 * 【我的备课资料对普通教师开放】新增（v10.1 配套）：
 *   - /lesson-plans/resources 及其学生档案子页 resources/class-profiles/:id/students
 *     两条路由的 RoleGuard 白名单加入 viewer（普通教师）。
 *     备课基础资料（课程大纲/班级学情/单元方案）对全体教学身份开放；
 *     建/改权限由各 Tab 内数据（my-groups 归属）+ 后端 service 层兜底。
 *   - 与 LPSidebar「我的备课资料」菜单 roles 白名单配套（口径必须一致）。
 *
 * 【生产端功能与教研组组长绑定】新增（v10.0 配套）：
 *   - 新增 LeadOrRoleGuard 双通道守卫：「账户身份命中 roles 白名单」或
 *     「当前用户是任一教研组的组长（lead）」二者满足其一即放行。
 *   - /lesson-plans/components 与 recipes 系列共 6 条路由由 RoleGuard 换为
 *     LeadOrRoleGuard（roles 白名单不变，仍为 admin + senior_operator）。
 *   - 组长判定复用共享 Hook useGroupLead（调 /ai-assistants/my-groups，
 *     模块级缓存，与 LPSidebar 菜单显隐共用一次请求；判定中显示 PageLoading
 *     防止误跳首页）。判定口径当前严格绑「组长」，放宽骨干只改 Hook 内一行常量。
 *   - 背景：200+ 教研员账户为骨干教师（operator）身份，被任命为教研组长后
 *     自动解锁备课配方/组件管理两个生产端功能，无需升级为学校管理员身份。
 *   - 后端配方/组件接口本就对所有登录用户放行（管控靠 service 归属校验），
 *     前端放宽无越权风险。
 *
 * 【课件共享课件库独立成栏】新增：
 *   - 新增 SharedCoursewareLibraryPage（懒加载），挂 /courseware/shared 子路由，
 *     位于 /courseware 布局下、:id 详情通配之前（避免 shared 被当作课件ID）。
 *   - 共享课件库从「我的课件」列表页的内嵌 Tab 升级为课件工坊侧边栏独立栏目，
 *     体验对齐教案库；列表页同步去掉共享 Tab。
 *
 * v194+优先级2新增（备课配方权限收敛 · Harness 生产者/消费者分离）：
 *   - /lesson-plans/recipes 及其子路由（wizard / wizard/:id / new / :id/edit）全部叠加
 *     RoleGuard roles=['admin','senior_operator']。
 *     定位：配方创建/管理是生产端，归管理员/教研员；普通老师是消费端，
 *     在备课起步（StartForm）选用现成配方即可，不接触配方管理界面。
 *   - 与 LPSidebar 菜单 roles 白名单配套（光藏入口不够，必须守卫路由防直接敲 URL 越权）。
 *   - 不影响 StartForm 选用现成配方的能力（那是 WorkshopPanels 内的消费端，本次不碰）。
 *
 * 【配方搭建一页化 · 批次4】配方编辑入口从旧单页编辑器迁移到分步向导：
 *   - 新增 recipes/wizard/:id 路由挂 RecipeWizardPage（编辑态，与 /wizard 新建态共用同一组件，
 *     靠 useParams 读 :id 区分；批次1 早已写好编辑态逻辑，本批挂上路由后方可验收）。
 *   - 旧路由改重定向（防存量书签/外链 404）：recipes/new → /wizard（静态）；
 *     recipes/:id/edit → /wizard/:id（动态，经 EditRedirect 用 useParams 取 :id 再 Navigate）。
 *   - 【清backlog】旧单页编辑器 RecipeEditorPage 已无路由指向，其 lazy import 与文件本次一并删除。
 *   - 四条 recipes 路由 RoleGuard 角色限制不变（admin/senior_operator）。
 *
 * v194+优先级2新增（组件管理权限收敛）：
 *   - /lesson-plans/components 路由叠加 RoleGuard roles=['admin','senior_operator']。
 *     与 LPSidebar 菜单项 roles 白名单配套（光藏入口不够，必须守卫路由防直接敲 URL 越权），
 *     范式与 stages-config（菜单 roles + 路由 RoleGuard）一致。
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
 * 合并重构改动（废弃 SchoolAdminPage 并轨）：
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
 * 迭代3.5 Phase A 新增（对话式备课工坊骨架）：
 *   - /lesson-plans index 路由从直接渲染 WorkshopPage 改为渲染 WorkshopModeRouter，
 *     由其按「URL ?mode= > 教案级记忆 > 全局偏好 > 默认值」决定渲染对话模式或专家模式。
 *   - WorkshopPage 自身零改动（专家模式永久保留）；ConversationModePage 为新增对话模式页面。
 *   - WorkshopModeRouter 内部静态引入两个页面（两者同属 index 路由 chunk，按需加载边界不变）。
 *   - 全局回退：workshop/conversation/workshopMode.ts 的 DEFAULT_WORKSHOP_MODE 改 'expert' 一行。
 *
 * 分包策略（Vite 自动按 dynamic import 边界拆分）：
 *   - 主 chunk：路由框架 + 布局组件 + 守卫
 *   - 课件审核板块 chunk
 *   - 教案系统板块 chunk
 *   - 课件工坊板块 chunk
 *   - admin/配置类 chunk
 *   - 各独立页面各自 chunk
 */
import { BrowserRouter, Routes, Route, Navigate, useParams } from 'react-router-dom'
import { Suspense, lazy, Component, type ReactNode, type ErrorInfo } from 'react'
import { AuthContext } from '@/store/auth'
import { useAuth } from '@/store/auth'
import { useAuthProvider } from '@/hooks/useAuthProvider'
import { useGroupLead } from '@/hooks/useGroupLead'

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
/* 迭代3.5 Phase A：index 路由改挂模式路由器（内部含 WorkshopPage 与 ConversationModePage） */
const WorkshopModeRouter = lazy(() => import('@/pages/lesson-plans/workshop/WorkshopModeRouter'))
const MyAssistantsPage = lazy(() => import('@/pages/lesson-plans/my-assistants/MyAssistantsPage'))
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
const RecipeWizardPage = lazy(() => import('@/pages/lesson-plans/recipes/RecipeWizardPage'))
const StagesConfigPage = lazy(() => import('@/pages/lesson-plans/stages-config/StagesConfigPage'))
const AssessmentPage = lazy(() => import('@/pages/lesson-plans/assessment/AssessmentPage'))
const MyTeachingResourcesPage = lazy(() => import('@/pages/lesson-plans/resources/MyTeachingResourcesPage'))
/* 班级学情 批次2a：学生个体档案独立全屏子页 */
const ClassStudentsPage = lazy(() => import('@/pages/lesson-plans/resources/class-profiles/ClassStudentsPage'))

/* ==================== 课件工坊（懒加载） ==================== */
const CoursewareListPage = lazy(() => import('@/pages/courseware/CoursewareListPage'))
const SharedCoursewareLibraryPage = lazy(() => import('@/pages/courseware/SharedCoursewareLibraryPage'))
const CWComponentsPage = lazy(() => import('@/pages/courseware/CWComponentsPage'))
const CWTemplatesPage = lazy(() => import('@/pages/courseware/CWTemplatesPage'))
const CoursewareWorkshopPage = lazy(() => import('@/pages/courseware/CoursewareWorkshopPage'))
const CWReviewDashboardPage = lazy(() => import('@/pages/courseware/review/CWReviewDashboardPage'))
const CWReviewWorkbenchPage = lazy(() => import('@/pages/courseware/review/CWReviewWorkbenchPage'))

/* ==================== 知识库压缩入库系统（隐藏全屏，懒加载） ==================== */
const KBCurriculumPage = lazy(() => import('@/pages/kb-admin/KBCurriculumPage'))

/* ==================== 通用独立页面（懒加载） ==================== */
const AccountPage = lazy(() => import('@/pages/account/AccountPage'))
const AICenterPage = lazy(() => import('@/pages/ai-center/AICenterPage'))
const AITraceDashboardPage = lazy(() => import('@/pages/ai-traces/AITraceDashboardPage'))
const AdminPage = lazy(() => import('@/pages/admin/AdminPage'))
/* 基础数据管理（独立全屏页，与 /admin 同级；门户并列卡片入口） */
const BaseDataPage = lazy(() => import('@/pages/base-data/BaseDataPage'))
/* 回收站（独立全屏页，懒加载） */
const TrashPage = lazy(() => import('@/pages/trash/TrashPage'))

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
 * LeadOrRoleGuard — 「账户身份白名单 或 教研组组长」双通道守卫（v10.0 配套）
 *
 * 放行条件（满足其一即可）：
 *   1) user.role 命中 roles 白名单（与 RoleGuard 行为一致，零请求即时放行）；
 *   2) 当前用户是任一教研组的「组长」（lead）——经共享 Hook useGroupLead 判定，
 *      与 LPSidebar 菜单显隐共用同一次 /ai-assistants/my-groups 请求（模块级缓存）。
 *
 * 判定中（checking）显示 PageLoading，避免异步结果未回时被误判非组长跳回首页。
 * 组长判定失败（网络异常等）fail-closed 跳首页，与改造前行为一致，不产生越权。
 * 注意：本守卫是体验层收口；配方/组件后端接口本就对登录用户放行（service 归属校验兜底）。
 */
function LeadOrRoleGuard({ children, roles }: { children: React.ReactNode; roles: string[] }) {
  const { user } = useAuth()
  // 身份是否直接命中白名单（命中则无需组长判定，Hook 传 enabled=false 不发请求）
  const roleHit = !!user && roles.includes(user.role)
  // React Hooks 规则：必须无条件调用，靠 enabled 参数控制是否真正发请求
  const { isLead, checking } = useGroupLead(!!user && !roleHit)
  if (!user) return <Navigate to="/" replace />
  if (roleHit) return <>{children}</>
  if (checking) return <PageLoading />
  if (isLead) return <>{children}</>
  return <Navigate to="/" replace />
}

/**
 * EditRedirect — 配方旧编辑路由 recipes/:id/edit 的动态重定向（批次4）
 *   React Router v6 的 <Navigate to> 不会自动替换路径参数，故用 useParams 取出当前 :id，
 *   再重定向到分步向导编辑态 /lesson-plans/recipes/wizard/:id（replace 不留历史）。
 *   缺 id 兜底回配方列表，避免拼出畸形 URL。
 */
function EditRedirect() {
  const { id } = useParams<{ id: string }>()
  if (!id) return <Navigate to="/lesson-plans/recipes" replace />
  return <Navigate to={`/lesson-plans/recipes/wizard/${id}`} replace />
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
            {/* 基础数据管理独立全屏页（/base-data，与 /admin 同级）：
                admin 与二线管理员（admin2）均可进（role 都是 admin，RoleGuard 放行）。
                页内含「学科 / 课程大纲」两个并列子 Tab。门户基础数据管理卡片进入。 */}
            <Route path="/base-data" element={
              <AuthGuard><RoleGuard roles={['admin']}><BaseDataPage /></RoleGuard></AuthGuard>
            } />
            <Route path="/ai-traces" element={
              <AuthGuard><RoleGuard roles={['admin']}><AITraceDashboardPage /></RoleGuard></AuthGuard>
            } />

            {/* 提示词管理独立全屏页（治理改造：从 /workflow 子路由挪出，脱离 MainLayout）：
                admin only。页面自带顶栏（← 返回首页 + 标题），入口经首页"提示词管理"卡片进入。
                纳管 prompts 表全部 key，按危险分档展示，高危改动需二次键入确认。 */}
            <Route path="/prompts" element={
              <AuthGuard><RoleGuard roles={['admin']}><PromptsPage /></RoleGuard></AuthGuard>
            } />

            {/* 积分管理独立全屏页（从备课工坊挪出，脱离 LPLayout）：
                只套 AuthGuard 登录即可进，页面内部按角色收窄视图——
                管理者(admin/senior/region)看全套管理Tab，普通老师看"我的账户/消费"两个Tab。
                入口：首页"积分管理"卡片(仅管理角色可见) + 右上角用户菜单"我的积分"(所有人)。 */}
            <Route path="/tokens" element={
              <AuthGuard><TokenDashboardPage /></AuthGuard>
            } />

            {/* ==================== 知识库压缩入库（隐藏全屏，脱离任何布局；仅授权人员经直接URL访问） ==================== */}
            {/* 参照 /lesson-plans/review/:id 模式：只套 AuthGuard，不进 Layout，不出现在 PortalPage 入口卡片。
                真正的访问拦截靠后端 RequireKBAuthorized 白名单中间件，前端守卫仅体验优化非安全边界。 */}

            {/* ==================== 回收站（独立全屏页，登录即可） ==================== */}
            <Route path="/trash" element={
              <AuthGuard><TrashPage /></AuthGuard>
            } />

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
              {/* 共享课件库独立栏目：必须在 :id 详情通配之前，否则 'shared' 会被当作课件ID */}
              <Route path="shared" element={<SharedCoursewareLibraryPage />} />
              {/* 阶段3 课件多级审核中心：固定字面量路径，须在 :id 详情通配之前，否则 review 被当作课件ID。
                  仅套 AuthGuard（CWLayout 已含），具体审核权限由后端按角色分流，菜单对所有登录用户可见。 */}
              <Route path="review" element={<CWReviewDashboardPage />} />
              <Route path="components" element={<CWComponentsPage />} />
              <Route path="templates" element={<CWTemplatesPage />} />
              <Route path=":id" element={<CoursewareWorkshopPage />} />
            </Route>

            {/* ==================== 教案系统 ==================== */}

            {/* 课件审核独立全屏工作台（弹窗改页面）：必须在 /courseware 布局路由之前注册，
                脱离 CWLayout 全屏铺满，左大预览等比缩放课件不截断 + 右批注/历史/决策。
                仅套 AuthGuard，具体审核权限由后端按角色分流。 */}
            <Route path="/courseware/review/:id" element={
              <AuthGuard><CWReviewWorkbenchPage /></AuthGuard>
            } />

            {/* 独立全屏评审工作台（必须在 /lesson-plans 布局路由之前注册） */}
            <Route path="/lesson-plans/review/:id" element={
              <AuthGuard><ReviewWorkbenchPage /></AuthGuard>
            } />

            <Route path="/lesson-plans" element={<AuthGuard><LPLayout /></AuthGuard>}>
              {/* 迭代3.5 Phase A：index 改挂模式路由器（对话模式/专家模式按偏好分发） */}
              <Route index element={<WorkshopModeRouter />} />
              {/* 提示词工坊 阶段A：我的 AI 助手（对话式造助手 + 现成助手复用） */}
              <Route path="my-assistants"   element={<MyAssistantsPage />} />
              <Route path="my-plans"         element={<MyPlansPage />} />
              <Route path="library"          element={<LibraryPage />} />
              <Route path="plans/:id"        element={<PlanDetailPage />} />
              <Route path="review"           element={<ReviewCenterLPPage />} />
              <Route path="review-v2"        element={<ReviewV2DashboardPage />} />
              <Route path="tokens"           element={<Navigate to="/tokens" replace />} />
              {/* 组件管理：身份白名单（admin/senior_operator）或 教研组组长 双通道放行。
                  与 LPSidebar 菜单 leadUnlock 配套（光放菜单不够，路由守卫口径必须一致）。 */}
              <Route path="components"       element={<LeadOrRoleGuard roles={['admin','senior_operator']}><ComponentsPage /></LeadOrRoleGuard>} />
              <Route path="templates"        element={<TemplatesPage />} />
              <Route path="templates/:id"    element={<TemplateEditorPage />} />
              {/* 备课配方：身份白名单（admin/senior_operator）或 教研组组长 双通道放行（生产端）。
                  路由全守卫，防直接敲 URL。消费端（StartForm 选用现成配方）在 WorkshopPanels 内，不受本守卫影响。
                  编辑入口已从旧单页编辑器迁移到分步向导：
                    recipes/wizard       新建（无 id）
                    recipes/wizard/:id   编辑（有 id，与新建共用 RecipeWizardPage，useParams 区分）
                    recipes/new          → 重定向 /wizard（旧路由兜底）
                    recipes/:id/edit     → 经 EditRedirect 动态重定向 /wizard/:id（旧路由兜底）
                  【清backlog】旧单页编辑器 RecipeEditorPage 文件与其 lazy import 已删除。 */}
              <Route path="recipes"            element={<LeadOrRoleGuard roles={['admin','senior_operator']}><RecipesPage /></LeadOrRoleGuard>} />
              <Route path="recipes/wizard"     element={<LeadOrRoleGuard roles={['admin','senior_operator']}><RecipeWizardPage /></LeadOrRoleGuard>} />
              <Route path="recipes/wizard/:id" element={<LeadOrRoleGuard roles={['admin','senior_operator']}><RecipeWizardPage /></LeadOrRoleGuard>} />
              <Route path="recipes/new"        element={<LeadOrRoleGuard roles={['admin','senior_operator']}><Navigate to="/lesson-plans/recipes/wizard" replace /></LeadOrRoleGuard>} />
              <Route path="recipes/:id/edit"   element={<LeadOrRoleGuard roles={['admin','senior_operator']}><EditRedirect /></LeadOrRoleGuard>} />
              <Route path="stages-config"    element={<RoleGuard roles={['admin']}><StagesConfigPage /></RoleGuard>} />
              <Route path="assessment"       element={<AssessmentPage />} />
              {/* v10.1：我的备课资料对普通教师（viewer）开放——备课基础资料不按账户身份区分，
                  建/改权限由各 Tab 内数据 + 后端 service 兜底。与 LPSidebar 菜单白名单口径一致。 */}
              <Route path="resources" element={<RoleGuard roles={['admin','senior_operator','operator','viewer']}><MyTeachingResourcesPage /></RoleGuard>} />
              {/* 班级学情 批次2a：学生个体档案独立全屏子页（角色白名单同 resources） */}
              <Route path="resources/class-profiles/:id/students" element={<RoleGuard roles={['admin','senior_operator','operator','viewer']}><ClassStudentsPage /></RoleGuard>} />
              {/* 旧路径重定向，防存量书签 404 */}
              <Route path="course-outlines" element={<Navigate to="/lesson-plans/resources" replace />} />
            </Route>

            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </Suspense>
        </RouteErrorBoundary>
      </BrowserRouter>
    </AuthContext.Provider>
  )
}

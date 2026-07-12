/**
 * 入口选择页面 — PortalPage
 * 登录后首先进入此页面，根据用户权限显示可用入口卡片。
 *
 * 【教学三板块】（所有 active 用户可见，受组织板块开关过滤）：
 * - 📝 备课工坊 /lesson-plans
 * - 🎨 课件工坊 /courseware
 * - 🖥️ 课件审核 /workflow（admin/senior_operator/operator）
 *
 * 【管理类板块】（按角色白名单显隐，不受组织板块开关限制 moduleGated=false）：
 * - 👥 用户管理 /admin              （admin/senior_operator/region_admin）——管"人与组织"
 * - 📚 基础数据管理 /base-data      （admin，含二线管理员 admin2）——管"业务基础字典：学科/课程大纲"
 * - 💎 积分管理 /tokens             （admin/senior_operator/region_admin）—— superOnly
 * - 🤖 AI 管理中心 /ai-center       （仅 admin）——模型/网关/场景配置 —— superOnly
 * - 📊 AI 调用统计 /ai-traces       （仅 admin）——调用追踪仪表盘 —— superOnly
 * - 📝 提示词管理 /prompts          （仅 admin）——各链路提示词 —— superOnly
 *
 * 【基础数据管理独立并列卡片】（本次）：
 *   基础数据管理与用户管理是两个不同维度的后台（一个管人/组织，一个管业务基础字典），
 *   故在门户作为并列卡片、各自独立页面/路由，而非藏在用户管理内部作二级 Tab。
 *   可见性 roles=['admin']、【不】标 superOnly——admin 与二线管理员(admin2, is_super=false)
 *   均可管理（学科、课程大纲不属超管专属敏感入口）。
 *
 * 【超管收口：superOnly 敏感入口二线管理员隐藏】：
 *   admin 角色被 is_super 细分为"超管(true)"与"二线管理员(false)"。
 *   积分管理 / AI管理中心 / AI调用统计 / 提示词管理 四个入口标记 superOnly=true，
 *   仅超管可见；二线管理员（is_super=false 的 admin）不显示这四张卡片，
 *   只保留用户管理、基础数据管理等常规管理能力。判定：superOnly 卡片需 user.is_super===true。
 *   （is_super 可能 undefined——老缓存/后端未下发，一律按非超管兜底，收紧不误放行。）
 *
 * 【右上角用户菜单】（测试反馈 #12「面板界面没有登出入口」）：
 *   门户首页不套任何系统 Layout，此前停在此页无处登出、也进不了个人中心。
 *   右上角加用户菜单（头像 + 下拉：个人中心 / 我的积分 / 退出登录）。
 *
 * 可见性三层判定：
 *   第1层 角色：entry.roles（'all' 或角色白名单）
 *   第2层 超管标记：entry.superOnly 为 true 时，需 user.is_super===true（仅敏感管理入口）
 *   第3层 组织板块开关：user.portal_modules[entry.key]（仅 moduleGated=true 的教学板块参与）
 *         - 后端按用户所属学校 settings.portal_modules 下发
 *         - 仅当显式为 false 时隐藏；缺省/undefined/true 一律可见
 *         - admin 后端强制全开，不受限
 */
import { useState, useRef, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import { logout as apiLogout } from '@/api/auth'

/* ==================== 入口卡片数据 ==================== */
interface PortalEntry {
  key: string
  icon: string
  title: string
  description: string
  path: string
  roles: string[] | 'all'  // 'all' 表示所有active用户可见
  // 是否参与"组织板块开关"过滤。教学三板块为 true；管理类入口为 false（仅按角色显隐）
  moduleGated: boolean
  // 超管收口：为 true 时该入口仅超级管理员（is_super=true）可见，二线管理员隐藏。
  // 仅积分/AI管理中心/AI调用统计/提示词四个敏感管理入口标记。缺省 undefined=不限超管。
  superOnly?: boolean
  // 卡片分组：'workspace' 教学工作区 / 'manage' 管理类（前端分区展示）
  group: 'workspace' | 'manage'
}

const entries: PortalEntry[] = [
  // ---------- 教学工作区 ----------
  {
    key: 'lesson_plan',
    icon: '📝',
    title: '备课工坊',
    description: 'AI辅助教案开发 · 教案库 · 教研协作',
    path: '/lesson-plans',
    roles: 'all',
    moduleGated: true,
    group: 'workspace',
  },
  {
    key: 'courseware',
    icon: '🎨',
    title: '课件工坊',
    description: 'AI辅助课件生成 · 模板组件 · 多媒体',
    path: '/courseware',
    roles: 'all',
    moduleGated: true,
    group: 'workspace',
  },
  {
    key: 'workflow',
    icon: '🖥️',
    title: '课件审核',
    description: '课件质量评估 · 审核 · 定稿 · 验收',
    path: '/workflow',
    roles: ['admin', 'senior_operator', 'operator'],
    moduleGated: true,
    group: 'workspace',
  },
  // ---------- 管理类 ----------
  {
    // 管理入口：统一用户管理中心。区域管理员的核心工作场所。管"人与组织"。
    // 不标 superOnly——二线管理员也需要管用户/组织架构。
    key: 'admin',
    icon: '👥',
    title: '用户管理',
    description: '用户 · 组织架构 · 教研组 · 权限',
    path: '/admin',
    roles: ['admin', 'senior_operator', 'region_admin'],
    moduleGated: false,
    group: 'manage',
  },
  {
    // 基础数据管理：与用户管理并列的独立入口，管"业务基础字典"（学科、课程大纲）。
    // roles=['admin']、不标 superOnly——admin 与二线管理员(admin2)均可管理。
    key: 'base_data',
    icon: '📚',
    title: '基础数据管理',
    description: '学科字典 · 课程大纲',
    path: '/base-data',
    roles: ['admin'],
    moduleGated: false,
    group: 'manage',
  },
  {
    // 积分管理：从备课工坊侧栏挪到首页（测试反馈 6-22 #2）。
    // admin 看全局 / 区域·学校管理员看辖区（数据由后端 TokenScope 收窄）。
    // 超管收口：superOnly=true，二线管理员（is_super=false）不可见。
    key: 'tokens',
    icon: '💎',
    title: '积分管理',
    description: '账户 · 分配 · 消费 · 策略',
    path: '/tokens',
    roles: ['admin', 'senior_operator', 'region_admin'],
    moduleGated: false,
    superOnly: true,
    group: 'manage',
  },
  {
    // AI 管理中心：模型/网关/场景配置。仅 admin。
    // 超管收口：superOnly=true，二线管理员不可见。
    key: 'ai_center',
    icon: '🤖',
    title: 'AI管理中心',
    description: '模型 · 网关 · 场景配置 · 别名',
    path: '/ai-center',
    roles: ['admin'],
    moduleGated: false,
    superOnly: true,
    group: 'manage',
  },
  {
    // AI 调用统计：调用追踪仪表盘。仅 admin。
    // 超管收口：superOnly=true，二线管理员不可见。
    key: 'ai_traces',
    icon: '📊',
    title: 'AI调用统计',
    description: '调用追踪 · 成本 · 降级 · 多维分析',
    path: '/ai-traces',
    roles: ['admin'],
    moduleGated: false,
    superOnly: true,
    group: 'manage',
  },
  {
    // 提示词管理：治理改造后从课件审核侧栏挪到首页作为独立管理入口。
    // 仅 admin。跳独立全屏页 /prompts，纳管 prompts 表全部 key、按危险分档展示。
    // 超管收口：superOnly=true，二线管理员不可见。
    key: 'prompts',
    icon: '📝',
    title: '提示词管理',
    description: '各业务链路提示词 · 版本 · 回滚',
    path: '/prompts',
    roles: ['admin'],
    moduleGated: false,
    superOnly: true,
    group: 'manage',
  },
]

/**
 * 判断某板块对当前用户是否可见（组织板块开关层）
 * 规则：portal_modules 缺失，或该 key 缺失，或值非 false → 可见；仅显式 false → 隐藏
 */
function isModuleEnabled(modules: Record<string, boolean> | undefined, key: string): boolean {
  if (!modules) return true            // 后端未下发（老缓存等）→ 默认可见
  if (!(key in modules)) return true   // 该板块未配置 → 默认可见
  return modules[key] !== false        // 仅显式 false 才隐藏
}

/* ==================== 样式常量 ==================== */
const styles = {
  /* 页面容器 */
  container: {
    minHeight: '100vh',
    background: '#FAFBFC',
    display: 'flex',
    flexDirection: 'column' as const,
    alignItems: 'center',
    padding: '0 24px 60px',
  },
  /* 内容居中区 */
  content: {
    flex: 1,
    display: 'flex',
    flexDirection: 'column' as const,
    alignItems: 'center',
    justifyContent: 'center',
    width: '100%',
    paddingTop: '40px',
  },
  /* 欢迎语 */
  greeting: {
    fontSize: '24px',
    fontWeight: 600,
    color: '#1F2937',
    marginBottom: '8px',
  },
  subtitle: {
    fontSize: '15px',
    color: '#6B7280',
    marginBottom: '40px',
  },
  /* 分区标题 */
  sectionLabel: {
    alignSelf: 'flex-start' as const,
    maxWidth: '1020px',
    width: '100%',
    margin: '0 auto 14px',
    fontSize: '13px',
    fontWeight: 600,
    color: '#9CA3AF',
    letterSpacing: '0.5px',
  },
  /* 卡片网格 */
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(260px, 1fr))',
    gap: '20px',
    maxWidth: '1020px',
    width: '100%',
    marginBottom: '36px',
  },
  /* 单个入口卡片 */
  card: {
    background: '#FFFFFF',
    borderRadius: '12px',
    padding: '28px 24px',
    cursor: 'pointer',
    transition: 'all 250ms ease',
    boxShadow: '0 1px 3px rgba(0,0,0,0.08)',
    border: '1px solid #F3F4F6',
    minHeight: '148px',
    display: 'flex',
    flexDirection: 'column' as const,
    justifyContent: 'center',
  },
  cardHover: {
    transform: 'translateY(-3px)',
    boxShadow: '0 8px 24px rgba(0,0,0,0.1)',
    borderColor: '#E5E7EB',
  },
  icon: {
    fontSize: '34px',
    marginBottom: '14px',
  },
  cardTitle: {
    fontSize: '19px',
    fontWeight: 600,
    color: '#1F2937',
    marginBottom: '6px',
  },
  cardDesc: {
    fontSize: '13px',
    color: '#6B7280',
    lineHeight: 1.5,
  },
  /* 页脚 */
  footer: {
    marginTop: '20px',
    fontSize: '13px',
    color: '#9CA3AF',
  },
} as const

/* ==================== 组件 ==================== */
export default function PortalPage() {
  const navigate = useNavigate()
  const { user, logout } = useAuth()

  // 如果没有用户信息，不渲染（AuthGuard会处理跳转）
  if (!user) return null

  // 三层过滤：角色 + 超管标记 + 组织板块开关
  //   - 角色：entry.roles 命中当前 role
  //   - 超管：superOnly 入口需 user.is_super===true（二线管理员隐藏敏感入口）
  //   - 板块：管理类入口 moduleGated=false 跳过；教学板块按组织开关
  const visibleEntries = entries.filter(entry => {
    const roleOk = entry.roles === 'all' || entry.roles.includes(user.role)
    if (!roleOk) return false
    // 超管收口：敏感入口仅 is_super===true 放行（undefined/false 一律隐藏）
    if (entry.superOnly && user.is_super !== true) return false
    if (!entry.moduleGated) return true
    return isModuleEnabled(user.portal_modules, entry.key)
  })

  const workspaceEntries = visibleEntries.filter(e => e.group === 'workspace')
  const manageEntries    = visibleEntries.filter(e => e.group === 'manage')

  return (
    <div style={styles.container}>
      {/* 顶栏：右上角用户菜单（个人中心 / 退出登录） */}
      <TopUserBar
        displayName={user.display_name}
        onAccount={() => navigate('/account', { state: { from: '/' } })}
        onTokens={() => navigate('/tokens')}
        onLogout={async () => {
          try { await apiLogout() } catch { /* 登出接口失败也继续本地清理 */ }
          logout()
          navigate('/login', { replace: true })
        }}
      />

      <div style={styles.content}>
        {/* 组织Logo（动态：用户所属组织Logo，无则不显示图片） */}
        {user.org_logo_url && (
          <div style={{ marginBottom: '28px' }}>
            <img src={user.org_logo_url} alt={user.org_name || 'Logo'} style={{ height: '48px', objectFit: 'contain' }} />
            {user.org_name && <div style={{ fontSize: '13px', color: '#6B7280', marginTop: '8px', textAlign: 'center' }}>{user.org_name}</div>}
          </div>
        )}

        {/* 欢迎语 */}
        <div style={styles.greeting}>欢迎回来，{user.display_name}</div>
        <div style={styles.subtitle}>请选择要进入的工作区</div>

        {/* 教学工作区 */}
        {workspaceEntries.length > 0 && (
          <>
            <div style={styles.sectionLabel}>教学工作区</div>
            <div style={styles.grid}>
              {workspaceEntries.map(entry => (
                <CardItem key={entry.key} entry={entry} onClick={() => navigate(entry.path)} />
              ))}
            </div>
          </>
        )}

        {/* 管理类（仅有可见管理卡片时才显示分区） */}
        {manageEntries.length > 0 && (
          <>
            <div style={styles.sectionLabel}>系统管理</div>
            <div style={styles.grid}>
              {manageEntries.map(entry => (
                <CardItem key={entry.key} entry={entry} onClick={() => navigate(entry.path)} />
              ))}
            </div>
          </>
        )}

        {/* 页脚 */}
        <div style={styles.footer}>TE-DNA 2.0{user.org_name ? ` · ${user.org_name}` : ''}</div>
      </div>
    </div>
  )
}

/* ==================== 右上角用户菜单 ==================== */
function TopUserBar({
  displayName,
  onAccount,
  onTokens,
  onLogout,
}: {
  displayName: string
  onAccount: () => void
  onTokens: () => void
  onLogout: () => void
}) {
  const [open, setOpen] = useState(false)
  const wrapRef = useRef<HTMLDivElement>(null)

  // 点击外部关闭下拉
  useEffect(() => {
    if (!open) return
    const handler = (e: MouseEvent) => {
      if (wrapRef.current && !wrapRef.current.contains(e.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', handler)
    return () => document.removeEventListener('mousedown', handler)
  }, [open])

  const initial = displayName?.charAt(0)?.toUpperCase() || 'U'

  return (
    <div style={{ width: '100%', display: 'flex', justifyContent: 'flex-end', padding: '16px 8px 0', position: 'sticky', top: 0, zIndex: 200 }}>
      <div ref={wrapRef} style={{ position: 'relative' }}>
        {/* 触发按钮：头像 + 名字 + 箭头 */}
        <button
          onClick={() => setOpen(o => !o)}
          style={{
            display: 'flex', alignItems: 'center', gap: '8px',
            padding: '6px 12px 6px 6px', borderRadius: '999px',
            border: '1px solid #E5E7EB', background: '#FFFFFF', cursor: 'pointer',
            boxShadow: '0 1px 3px rgba(0,0,0,0.06)',
          }}
        >
          <span style={{
            width: '30px', height: '30px', borderRadius: '50%',
            background: 'linear-gradient(135deg,#4F7BE8,#7C3AED)', color: '#fff',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            fontSize: '14px', fontWeight: 700,
          }}>{initial}</span>
          <span style={{ fontSize: '14px', color: '#1F2937', fontWeight: 500, maxWidth: '120px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{displayName}</span>
          <span style={{ fontSize: '10px', color: '#9CA3AF' }}>▾</span>
        </button>

        {/* 下拉菜单 */}
        {open && (
          <div style={{
            position: 'absolute', right: 0, top: '44px', minWidth: '160px',
            background: '#FFFFFF', borderRadius: '12px', border: '1px solid #E5E7EB',
            boxShadow: '0 8px 24px rgba(0,0,0,0.12)', overflow: 'hidden', padding: '6px',
          }}>
            <button
              onClick={() => { setOpen(false); onAccount() }}
              style={menuItemStyle}
              onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = '#F3F4F6' }}
              onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
            >
              <span>👤</span> 个人中心
            </button>
            <button
              onClick={() => { setOpen(false); onTokens() }}
              style={menuItemStyle}
              onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = '#F3F4F6' }}
              onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
            >
              <span>🪙</span> 我的积分
            </button>
            <button
              onClick={() => { setOpen(false); onLogout() }}
              style={{ ...menuItemStyle, color: '#EF4444' }}
              onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(239,68,68,0.06)' }}
              onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
            >
              <span>🚪</span> 退出登录
            </button>
          </div>
        )}
      </div>
    </div>
  )
}

const menuItemStyle: React.CSSProperties = {
  display: 'flex', alignItems: 'center', gap: '10px',
  width: '100%', padding: '10px 12px', borderRadius: '8px',
  border: 'none', background: 'transparent', cursor: 'pointer',
  fontSize: '14px', color: '#1F2937', textAlign: 'left',
  transition: 'background 150ms ease',
}

/* ==================== 卡片子组件（含hover效果） ==================== */
function CardItem({ entry, onClick }: { entry: PortalEntry; onClick: () => void }) {
  const [hovered, setHovered] = useState(false)

  const cardStyle = {
    ...styles.card,
    ...(hovered ? styles.cardHover : {}),
  }

  return (
    <div
      style={cardStyle}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      onClick={onClick}
    >
      <div style={styles.icon}>{entry.icon}</div>
      <div style={styles.cardTitle}>{entry.title}</div>
      <div style={styles.cardDesc}>{entry.description}</div>
    </div>
  )
}

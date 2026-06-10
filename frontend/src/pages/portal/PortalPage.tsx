/**
 * 入口选择页面 — PortalPage
 * 登录后首先进入此页面，根据用户权限显示可用入口卡片：
 * - 📝 备课工坊：教案系统（所有active用户可见）
 * - 🎨 课件工坊：AI课件生成系统（所有active用户可见）
 * - 🖥️ 课件审核：课件审核系统（admin/senior_operator/operator可见）
 * - 👥 用户管理：统一用户管理中心（admin/senior_operator/region_admin可见）
 *
 * 可见性两层判定（v172）：
 *   第1层 角色：entry.roles（'all' 或角色白名单）
 *   第2层 组织板块开关：user.portal_modules[entry.key]
 *         - 后端按用户所属学校 settings.portal_modules 下发
 *         - 仅当显式为 false 时隐藏；缺省/undefined/true 一律可见（不波及存量）
 *         - admin 后端强制全开，不受限
 *
 * Phase6.2新增（区域管理员）：
 *   - 新增“用户管理”入口卡片，指向 /admin，对 admin/senior_operator/region_admin 可见
 *   - 该卡片不参与组织板块开关过滤（管理入口不属于教学三板块），仅按角色白名单显隐
 */
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'

/* ==================== 入口卡片数据 ==================== */
interface PortalEntry {
  key: string
  icon: string
  title: string
  description: string
  path: string
  roles: string[] | 'all'  // 'all' 表示所有active用户可见
  // 是否参与“组织板块开关”过滤。教学三板块为 true；管理类入口为 false（仅按角色显隐）
  moduleGated: boolean
}

const entries: PortalEntry[] = [
  {
    key: 'lesson_plan',
    icon: '📝',
    title: '备课工坊',
    description: 'AI辅助教案开发 · 教案库 · 教研协作',
    path: '/lesson-plans',
    roles: 'all',
    moduleGated: true,
  },
  {
    key: 'courseware',
    icon: '🎨',
    title: '课件工坊',
    description: 'AI辅助课件生成 · 模板组件 · 多媒体',
    path: '/courseware',
    roles: 'all',
    moduleGated: true,
  },
  {
    key: 'workflow',
    icon: '🖥️',
    title: '课件审核',
    description: '课件质量评估 · 审核 · 定稿 · 验收',
    path: '/workflow',
    roles: ['admin', 'senior_operator', 'operator'],
    moduleGated: true,
  },
  {
    // 管理入口：统一用户管理中心。区域管理员的核心工作场所。
    // admin / senior_operator / region_admin 三类管理角色可见；不受组织板块开关限制。
    key: 'admin',
    icon: '👥',
    title: '用户管理',
    description: '用户 · 组织架构 · 教研组 · 权限',
    path: '/admin',
    roles: ['admin', 'senior_operator', 'region_admin'],
    moduleGated: false,
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
    justifyContent: 'center',
    padding: '40px 24px',
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
    marginBottom: '48px',
  },
  /* 卡片网格 */
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(280px, 1fr))',
    gap: '24px',
    maxWidth: '1020px',
    width: '100%',
  },
  /* 单个入口卡片 */
  card: {
    background: '#FFFFFF',
    borderRadius: '12px',
    padding: '32px 28px',
    cursor: 'pointer',
    transition: 'all 250ms ease',
    boxShadow: '0 1px 3px rgba(0,0,0,0.08)',
    border: '1px solid #F3F4F6',
    minHeight: '160px',
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
    fontSize: '36px',
    marginBottom: '16px',
  },
  cardTitle: {
    fontSize: '20px',
    fontWeight: 600,
    color: '#1F2937',
    marginBottom: '8px',
  },
  cardDesc: {
    fontSize: '14px',
    color: '#6B7280',
    lineHeight: 1.5,
  },
  /* 页脚 */
  footer: {
    marginTop: '60px',
    fontSize: '13px',
    color: '#9CA3AF',
  },
} as const

/* ==================== 组件 ==================== */
export default function PortalPage() {
  const navigate = useNavigate()
  const { user } = useAuth()

  // 如果没有用户信息，不渲染（AuthGuard会处理跳转）
  if (!user) return null

  // 双层过滤：角色 + 组织板块开关（管理类入口 moduleGated=false 跳过第2层）
  const visibleEntries = entries.filter(entry => {
    // 第1层：角色
    const roleOk = entry.roles === 'all' || entry.roles.includes(user.role)
    if (!roleOk) return false
    // 第2层：组织板块开关（仅教学三板块参与）
    if (!entry.moduleGated) return true
    return isModuleEnabled(user.portal_modules, entry.key)
  })

  return (
    <div style={styles.container}>
      {/* 组织Logo（动态：用户所属组织Logo，无则不显示图片） */}
      {user.org_logo_url && (
        <div style={{ marginBottom: '32px' }}>
          <img src={user.org_logo_url} alt={user.org_name || 'Logo'} style={{ height: '48px', objectFit: 'contain' }} />
          {user.org_name && <div style={{ fontSize: '13px', color: '#6B7280', marginTop: '8px', textAlign: 'center' }}>{user.org_name}</div>}
        </div>
      )}

      {/* 欢迎语 */}
      <div style={styles.greeting}>欢迎回来，{user.display_name}</div>
      <div style={styles.subtitle}>请选择要进入的工作区</div>

      {/* 入口卡片网格 */}
      <div style={styles.grid}>
        {visibleEntries.map(entry => (
          <CardItem
            key={entry.key}
            entry={entry}
            onClick={() => navigate(entry.path)}
          />
        ))}
      </div>

      {/* 页脚 */}
      <div style={styles.footer}>TE-DNA 2.0{user.org_name ? ` · ${user.org_name}` : ''}</div>
    </div>
  )
}

/* ==================== 卡片子组件（含hover效果） ==================== */
import { useState } from 'react'

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

/**
 * 教案系统侧边栏 — LPSidebar v11.0
 *
 * v11.0变更（回收站迭代）：
 *   - 底部"返回入口"按钮上方新增「🗑️ 回收站」菜单项，
 *     指向独立全屏页 /trash（脱离 LPLayout，AuthGuard 即可）。
 *   - 不进 menuItems 数组（不受 roles/leadUnlock 过滤，人人可见）。
 */
import { useState } from 'react'
import { useAuth } from '@/store/auth'
import { useLocation, useNavigate } from 'react-router-dom'
import { useGroupLead } from '@/hooks/useGroupLead'

interface LPMenuItem {
  key: string
  label: string
  icon: string
  path: string
  description: string
  roles?: string[]
  leadUnlock?: boolean
}

const menuItems: LPMenuItem[] = [
  { key: 'workshop',      label: '备课工坊',     icon: '✨', path: '/lesson-plans',                description: 'AI辅助对话式备课' },
  { key: 'my-assistants', label: '我的 AI 助手', icon: '🤖', path: '/lesson-plans/my-assistants', description: '和AI聊着造你的备课助手' },
  { key: 'resources',     label: '我的备课资料', icon: '📂', path: '/lesson-plans/resources',     description: '课程大纲·班级学情·单元方案', roles: ['admin', 'senior_operator', 'operator', 'viewer'] },
  { key: 'recipes',       label: '备课配方',     icon: '📦', path: '/lesson-plans/recipes',       description: '可复用的AI备课预设包', roles: ['admin', 'senior_operator'], leadUnlock: true },
  { key: 'my-plans',      label: '我的教案',     icon: '📋', path: '/lesson-plans/my-plans',      description: '个人教案管理' },
  { key: 'library',       label: '教案库',       icon: '📚', path: '/lesson-plans/library',       description: '教研组共享教案' },
  { key: 'review',        label: '评审中心',     icon: '📝', path: '/lesson-plans/review',        description: '人工评审教案' },
  { key: 'review-v2',     label: '多级审核',     icon: '🔍', path: '/lesson-plans/review-v2',     description: '三级审核工作台' },
  { key: 'components',    label: '组件管理',     icon: '🧩', path: '/lesson-plans/components',    description: '教学设计组件库', roles: ['admin', 'senior_operator'], leadUnlock: true },
  { key: 'templates',     label: '提示词模板',   icon: '📐', path: '/lesson-plans/templates',     description: '分层提示词模板配置' },
  { key: 'stages-config', label: '阶段管理',     icon: '⚙️', path: '/lesson-plans/stages-config', description: '备课阶段流程配置', roles: ['admin'] },
]

const COLORS = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  primaryBorder: 'rgba(79,123,232,0.15)',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  bgSidebar: '#FFFFFF',
  bgHover: '#F9FAFB',
  border: '#F3F4F6',
}

export default function LPSidebar() {
  const { user } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()
  const [hoveredKey, setHoveredKey] = useState<string | null>(null)

  const needLeadCheck = !!user && !['admin', 'senior_operator'].includes(user.role)
  const { isLead } = useGroupLead(needLeadCheck)

  const isActive = (path: string) => {
    if (path === '/lesson-plans') return location.pathname === '/lesson-plans'
    if (path === '/lesson-plans/review') {
      return location.pathname === '/lesson-plans/review' || (location.pathname.startsWith('/lesson-plans/review/') && !location.pathname.startsWith('/lesson-plans/review-v2'))
    }
    if (path === '/trash') return location.pathname === '/trash'
    return location.pathname.startsWith(path)
  }

  const visibleItems = menuItems.filter(item => {
    if (!item.roles) return true
    if (user?.role && item.roles.includes(user.role)) return true
    if (item.leadUnlock && isLead) return true
    return false
  })

  // 渲染单个菜单按钮的通用函数（主菜单和回收站共用）
  const renderMenuButton = (key: string, icon: string, label: string, path: string, description: string) => {
    const active = isActive(path)
    const hovered = hoveredKey === key
    return (
      <button
        key={key}
        onClick={() => navigate(path)}
        onMouseEnter={() => setHoveredKey(key)}
        onMouseLeave={() => setHoveredKey(null)}
        style={{
          width: '100%', display: 'flex', alignItems: 'center', gap: '12px',
          padding: '12px 16px', borderRadius: '12px',
          border: active ? `1px solid ${COLORS.primaryBorder}` : '1px solid transparent',
          cursor: 'pointer', fontSize: '15px',
          fontWeight: active ? 600 : 400,
          color: active ? COLORS.primary : COLORS.textSecondary,
          background: active ? COLORS.primaryLight : (hovered ? COLORS.bgHover : 'transparent'),
          transition: 'all 200ms ease',
          textAlign: 'left',
          transform: hovered && !active ? 'translateX(2px)' : 'none',
        }}
      >
        <span style={{ fontSize: '18px', width: '24px', textAlign: 'center', flexShrink: 0 }}>{icon}</span>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div>{label}</div>
          {(active || hovered) && (
            <div style={{ fontSize: '11px', color: active ? COLORS.primary : COLORS.textMuted, marginTop: '2px', opacity: 0.7 }}>{description}</div>
          )}
        </div>
        {active && <div style={{ width: '6px', height: '6px', borderRadius: '50%', background: COLORS.primary, flexShrink: 0 }} />}
      </button>
    )
  }

  return (
    <aside style={{
      width: '260px', height: '100vh',
      display: 'flex', flexDirection: 'column',
      background: COLORS.bgSidebar,
      borderRight: `1px solid ${COLORS.border}`,
      flexShrink: 0,
    }}>
      {/* Logo + 系统名区域 */}
      <div style={{ display: 'flex', flexDirection: 'column', borderBottom: `1px solid ${COLORS.border}` }}>
        <div style={{ padding: '12px 18px 6px', cursor: 'pointer' }} onClick={() => navigate('/')} title="返回首页">
          {user?.org_logo_url ? (
            <img src={user.org_logo_url} alt={user.org_name || 'Logo'} style={{ height: '26px', objectFit: 'contain', display: 'block' }} />
          ) : (
            <div style={{ fontSize: '14px', fontWeight: 700, color: '#4F7BE8', letterSpacing: '1px' }}>TE-DNA 2.0</div>
          )}
        </div>
        <div style={{ padding: '6px 18px 12px', display: 'flex', alignItems: 'center', gap: '10px' }}>
          <div style={{
            width: '32px', height: '32px',
            background: 'linear-gradient(135deg, #4F7BE8, #818CF8)',
            borderRadius: '8px',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            boxShadow: '0 2px 6px rgba(79,123,232,0.2)',
            flexShrink: 0,
          }}>
            <span style={{ fontSize: '14px' }}>📝</span>
          </div>
          <div>
            <div style={{ color: COLORS.textPrimary, fontSize: '14px', fontWeight: 600, letterSpacing: '-0.3px' }}>备课工坊</div>
            <div style={{ color: COLORS.textMuted, fontSize: '10px', marginTop: '1px' }}>AI辅助教案开发</div>
          </div>
        </div>
      </div>

      {/* 菜单列表 */}
      <nav style={{ flex: 1, padding: '16px 12px', overflowY: 'auto' }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
          {visibleItems.map(item => renderMenuButton(item.key, item.icon, item.label, item.path, item.description))}
        </div>
      </nav>

      {/* 底部：回收站 + 返回入口 */}
      <div style={{ padding: '12px', borderTop: `1px solid ${COLORS.border}` }}>
        {/* 回收站入口（人人可见，不受 roles/leadUnlock 过滤） */}
        <div style={{ marginBottom: '4px' }}>
          {renderMenuButton('trash', '🗑️', '回收站', '/trash', '已删除的教案和课件')}
        </div>
        <button
          onClick={() => navigate('/')}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = COLORS.bgHover }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
          style={{
            width: '100%', display: 'flex', alignItems: 'center', gap: '10px',
            padding: '10px 16px', borderRadius: '10px', border: 'none',
            cursor: 'pointer', fontSize: '13px',
            color: COLORS.textMuted, background: 'transparent',
            transition: 'all 200ms ease', textAlign: 'left',
          }}
        >
          <span style={{ fontSize: '14px' }}>←</span>
          <span>返回入口</span>
        </button>
      </div>
    </aside>
  )
}

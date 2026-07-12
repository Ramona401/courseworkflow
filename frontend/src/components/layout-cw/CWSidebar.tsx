/**
 * 课件工坊侧边栏 — CWSidebar v2.0
 *
 * v2.0变更（回收站迭代）：
 *   - 底部"返回入口"按钮上方新增「🗑️ 回收站」入口，
 *     指向独立全屏页 /trash（脱离 CWLayout，AuthGuard 即可）。
 */
import { useState } from 'react'
import { useAuth } from '@/store/auth'
import { useLocation, useNavigate } from 'react-router-dom'

interface CWMenuItem {
  key: string
  label: string
  icon: string
  path: string
  description: string
  adminOnly?: boolean
}

const menuItems: CWMenuItem[] = [
  { key: 'my-coursewares', label: '我的课件',   icon: '📋', path: '/courseware',            description: '课件列表与管理' },
  { key: 'shared',         label: '共享课件库', icon: '🔗', path: '/courseware/shared',     description: '同校同组老师共享的课件' },
  { key: 'review',         label: '审核中心',   icon: '🛡️', path: '/courseware/review',     description: '课件多级审核工作台' },
  { key: 'components',     label: '组件库',     icon: '🧩', path: '/courseware/components', description: '课件交互组件模板' },
  { key: 'templates',      label: '风格模板',   icon: '🎨', path: '/courseware/templates',  description: '预设视觉风格方案' },
]

const COLORS = {
  primary: '#F59E0B',
  primaryLight: 'rgba(245,158,11,0.08)',
  primaryBorder: 'rgba(245,158,11,0.15)',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  bgSidebar: '#FFFFFF',
  bgHover: '#F9FAFB',
  border: '#F3F4F6',
}

export default function CWSidebar() {
  const { user } = useAuth()
  const location = useLocation()
  const navigate = useNavigate()
  const [hoveredKey, setHoveredKey] = useState<string | null>(null)

  const isActive = (path: string) => {
    if (path === '/courseware') return location.pathname === '/courseware'
    if (path === '/courseware/shared') return location.pathname === '/courseware/shared'
    if (path === '/courseware/review') return location.pathname === '/courseware/review'
    if (path === '/trash') return location.pathname === '/trash'
    return location.pathname.startsWith(path)
  }

  // 渲染单个菜单按钮的通用函数
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
          transition: 'all 200ms ease', textAlign: 'left',
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
            <div style={{ fontSize: '14px', fontWeight: 700, color: '#F59E0B', letterSpacing: '1px' }}>TE-DNA 2.0</div>
          )}
        </div>
        <div style={{ padding: '6px 18px 12px', display: 'flex', alignItems: 'center', gap: '10px' }}>
          <div style={{
            width: '32px', height: '32px',
            background: 'linear-gradient(135deg, #F59E0B, #EF4444)',
            borderRadius: '8px',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            boxShadow: '0 2px 6px rgba(245,158,11,0.2)',
            flexShrink: 0,
          }}>
            <span style={{ fontSize: '14px' }}>🎨</span>
          </div>
          <div>
            <div style={{ color: COLORS.textPrimary, fontSize: '14px', fontWeight: 600, letterSpacing: '-0.3px' }}>课件工坊</div>
            <div style={{ color: COLORS.textMuted, fontSize: '10px', marginTop: '1px' }}>AI辅助课件生成</div>
          </div>
        </div>
      </div>

      {/* 菜单列表 */}
      <nav style={{ flex: 1, padding: '16px 12px', overflowY: 'auto' }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
          {menuItems.filter(item => !item.adminOnly || user?.role === 'admin').map(item =>
            renderMenuButton(item.key, item.icon, item.label, item.path, item.description)
          )}
        </div>
      </nav>

      {/* 底部：回收站 + 返回入口 */}
      <div style={{ padding: '12px', borderTop: `1px solid ${COLORS.border}` }}>
        {/* 回收站入口 */}
        <div style={{ marginBottom: '4px' }}>
          {renderMenuButton('trash', '🗑️', '回收站', '/trash', '已删除的课件')}
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

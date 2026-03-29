/**
 * 教案系统侧边栏 — LPSidebar v5.0
 *
 * v5.0变更：
 *   - 移除底部用户信息卡片和登出按钮（已统一到顶部Header下拉菜单）
 *   - 保留"返回入口"按钮
 *   - 其余菜单和样式保持不变
 */
import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'

interface LPMenuItem {
  key: string
  label: string
  icon: string
  path: string
  description: string
}

const menuItems: LPMenuItem[] = [
  { key: 'workshop',   label: '备课工坊',   icon: '✨', path: '/lesson-plans',            description: 'AI辅助对话式备课' },
  { key: 'my-plans',   label: '我的教案',   icon: '📋', path: '/lesson-plans/my-plans',   description: '个人教案管理' },
  { key: 'library',    label: '教案库',     icon: '📚', path: '/lesson-plans/library',    description: '教研组共享教案' },
  { key: 'review',     label: '评审中心',   icon: '📝', path: '/lesson-plans/review',     description: '人工评审教案' },
  { key: 'components', label: '组件管理',   icon: '🧩', path: '/lesson-plans/components', description: '教学设计组件库' },
  { key: 'templates',  label: '提示词模板', icon: '📐', path: '/lesson-plans/templates',  description: '分层提示词模板配置' },
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
  const location = useLocation()
  const navigate = useNavigate()
  const [hoveredKey, setHoveredKey] = useState<string | null>(null)

  const isActive = (path: string) => {
    if (path === '/lesson-plans') return location.pathname === '/lesson-plans'
    return location.pathname.startsWith(path)
  }

  return (
    <aside style={{
      width: '260px', height: '100vh',
      display: 'flex', flexDirection: 'column',
      background: COLORS.bgSidebar,
      borderRight: `1px solid ${COLORS.border}`,
      flexShrink: 0,
    }}>
      {/* Logo 区域 */}
      <div style={{
        height: '64px', display: 'flex', alignItems: 'center',
        padding: '0 22px', borderBottom: `1px solid ${COLORS.border}`,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div style={{
            width: '36px', height: '36px',
            background: 'linear-gradient(135deg, #4F7BE8, #818CF8)',
            borderRadius: '10px',
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            boxShadow: '0 2px 8px rgba(79,123,232,0.25)',
          }}>
            <span style={{ fontSize: '16px' }}>📝</span>
          </div>
          <div>
            <div style={{ color: COLORS.textPrimary, fontSize: '15px', fontWeight: 600, letterSpacing: '-0.3px' }}>
              备课工坊
            </div>
            <div style={{ color: COLORS.textMuted, fontSize: '11px', marginTop: '1px' }}>
              AI辅助教案开发
            </div>
          </div>
        </div>
      </div>

      {/* 菜单列表 */}
      <nav style={{ flex: 1, padding: '16px 12px', overflowY: 'auto' }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
          {menuItems.map((item) => {
            const active = isActive(item.path)
            const hovered = hoveredKey === item.key
            return (
              <button
                key={item.key}
                onClick={() => navigate(item.path)}
                onMouseEnter={() => setHoveredKey(item.key)}
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
                <span style={{ fontSize: '18px', width: '24px', textAlign: 'center', flexShrink: 0 }}>
                  {item.icon}
                </span>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div>{item.label}</div>
                  {(active || hovered) && (
                    <div style={{
                      fontSize: '11px',
                      color: active ? COLORS.primary : COLORS.textMuted,
                      marginTop: '2px', opacity: 0.7,
                    }}>{item.description}</div>
                  )}
                </div>
                {active && (
                  <div style={{
                    width: '6px', height: '6px', borderRadius: '50%',
                    background: COLORS.primary, flexShrink: 0,
                  }} />
                )}
              </button>
            )
          })}
        </div>
      </nav>

      {/* 底部：仅保留返回入口按钮（用户信息已移至顶部Header）*/}
      <div style={{ padding: '12px', borderTop: `1px solid ${COLORS.border}` }}>
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

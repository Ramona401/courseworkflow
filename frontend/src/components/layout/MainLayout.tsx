/**
 * 主布局组件 - 课件审核系统专用 v5.2
 * 下拉菜单新增"用户管理"入口（admin专属）
 */
import { useState, useRef, useEffect } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import Sidebar from './Sidebar'

const pageTitles: Record<string, string> = {
  '/': '仪表盘',
  '/users': '用户管理',
  '/ai-config': 'AI 配置',
  '/prompts': '提示词管理',
  '/external-data': '外部数据配置',
  '/courses': '课程管理',
  '/pipelines': 'Pipeline',
  '/review': '审核中心',
  '/settings': '系统设置',
}

function getPageTitle(pathname: string): string {
  if (pageTitles[pathname]) return pageTitles[pathname]
  if (pathname.startsWith('/pipelines/')) return 'Pipeline 详情'
  return 'TE-DNA 2.0'
}

// ==================== DropdownPortal ====================

function DropdownPortal({
  children,
  triggerRef,
}: {
  children: React.ReactNode
  triggerRef: React.RefObject<HTMLDivElement | null>
}) {
  const [pos, setPos] = useState({ top: 0, right: 0 })

  useEffect(() => {
    if (triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect()
      setPos({ top: rect.bottom + 8, right: window.innerWidth - rect.right })
    }
  }, [triggerRef])

  return (
    <div style={{
      position: 'fixed', top: pos.top, right: pos.right,
      width: '220px', background: '#fff',
      borderRadius: '14px', border: '1px solid rgba(0,0,0,0.08)',
      boxShadow: '0 8px 32px rgba(0,0,0,0.15)',
      overflow: 'hidden', zIndex: 9999,
    }}>
      {children}
    </div>
  )
}

// ==================== 用户下拉菜单 ====================

function UserMenu() {
  const { user, logout } = useAuth()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)
  const menuRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setOpen(false)
    }
    if (open) document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  const go = (path: string, from: string) => {
    setOpen(false)
    navigate(path, { state: { from } })
  }

  const handleLogout = () => {
    setOpen(false)
    logout()
    navigate('/login', { replace: true })
  }

  const roleNames: Record<string, string> = {
    admin: '管理员', senior_operator: '高级操作员',
    operator: '操作员', viewer: '查看者',
  }

  return (
    <div ref={menuRef} style={{ position: 'relative' }}>
      <button
        onClick={() => setOpen(p => !p)}
        style={{
          display: 'flex', alignItems: 'center', gap: '8px',
          padding: '6px 10px 6px 6px', borderRadius: '20px',
          border: '1px solid rgba(0,0,0,0.08)',
          background: open ? 'rgba(0,0,0,0.06)' : 'transparent',
          cursor: 'pointer', transition: 'all 150ms ease',
        }}
        onMouseEnter={e => { if (!open) (e.currentTarget as HTMLElement).style.background = 'rgba(0,0,0,0.04)' }}
        onMouseLeave={e => { if (!open) (e.currentTarget as HTMLElement).style.background = 'transparent' }}
      >
        <div style={{
          width: '30px', height: '30px',
          background: 'linear-gradient(135deg,#5856d6,#007aff)',
          borderRadius: '50%', display: 'flex', alignItems: 'center',
          justifyContent: 'center', flexShrink: 0,
        }}>
          <span style={{ color: '#fff', fontSize: '12px', fontWeight: 700 }}>
            {user?.display_name?.charAt(0)?.toUpperCase() || 'U'}
          </span>
        </div>
        <span style={{ fontSize: '13px', fontWeight: 500, color: '#1d1d1f', maxWidth: '80px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {user?.display_name}
        </span>
        <span style={{ fontSize: '10px', color: '#8e8e93', transform: open ? 'rotate(180deg)' : 'none', transition: 'transform 200ms ease' }}>▼</span>
      </button>

      {open && (
        <DropdownPortal triggerRef={menuRef}>
          {/* 用户信息头部 */}
          <div style={{ padding: '14px 16px 12px', borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
            <div style={{ fontSize: '14px', fontWeight: 600, color: '#1d1d1f' }}>{user?.display_name}</div>
            <div style={{ fontSize: '12px', color: '#8e8e93', marginTop: '2px' }}>
              @{user?.username} · {roleNames[user?.role || ''] || user?.role}
            </div>
          </div>

          {/* 菜单项 */}
          <div style={{ padding: '6px' }}>
            <MenuItem icon="👤" label="个人中心" onClick={() => go('/account', '/workflow')} />
            {/* admin专属功能 */}
            {user?.role === 'admin' && (
              <>
                <MenuItem icon="👥" label="用户管理" onClick={() => go('/admin', '/workflow')} highlight />
                <MenuItem icon="🤖" label="AI 管理中心" onClick={() => go('/ai-center', '/workflow')} />
              </>
            )}
          </div>

          <div style={{ height: '1px', background: 'rgba(0,0,0,0.06)', margin: '0 6px' }} />

          <div style={{ padding: '6px' }}>
            <MenuItem icon="⏻" label="退出登录" onClick={handleLogout} danger />
          </div>
        </DropdownPortal>
      )}
    </div>
  )
}

function MenuItem({ icon, label, onClick, danger, highlight }: {
  icon: string; label: string; onClick: () => void; danger?: boolean; highlight?: boolean
}) {
  const [hovered, setHovered] = useState(false)
  return (
    <button
      onClick={onClick}
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        width: '100%', display: 'flex', alignItems: 'center', gap: '10px',
        padding: '9px 12px', borderRadius: '8px', border: 'none',
        cursor: 'pointer', textAlign: 'left', fontSize: '14px',
        color: danger
          ? (hovered ? '#ff453a' : '#ff3b30')
          : highlight
            ? '#4F7BE8'
            : (hovered ? '#1d1d1f' : '#3c3c43'),
        background: hovered
          ? (danger ? 'rgba(255,59,48,0.06)' : highlight ? 'rgba(79,123,232,0.08)' : 'rgba(0,0,0,0.04)')
          : highlight ? 'rgba(79,123,232,0.04)' : 'transparent',
        transition: 'all 150ms ease',
      }}
    >
      <span style={{ fontSize: '16px', width: '20px', textAlign: 'center' }}>{icon}</span>
      <span style={{ fontWeight: highlight ? 600 : 400 }}>{label}</span>
    </button>
  )
}

// ==================== 主布局 ====================

export default function MainLayout() {
  const location = useLocation()
  const subPath = location.pathname.replace('/workflow', '') || '/'
  const pageTitle = getPageTitle(subPath)

  return (
    <div style={{ display: 'flex', height: '100vh', overflow: 'hidden', background: '#f5f5f7' }}>
      <Sidebar />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <header style={{
          height: '64px', position: 'relative', zIndex: 50,
          background: 'rgba(255,255,255,0.72)',
          backdropFilter: 'blur(20px)', WebkitBackdropFilter: 'blur(20px)',
          borderBottom: '1px solid rgba(0,0,0,0.06)',
          display: 'flex', alignItems: 'center', padding: '0 28px', flexShrink: 0,
        }}>
          <h2 style={{ flex: 1, fontSize: '18px', fontWeight: 600, color: '#1d1d1f', margin: 0, letterSpacing: '-0.3px' }}>
            {pageTitle}
          </h2>
          <UserMenu />
        </header>
        <main style={{ flex: 1, overflowY: 'auto', padding: '28px' }}>
          <Outlet />
        </main>
      </div>
    </div>
  )
}

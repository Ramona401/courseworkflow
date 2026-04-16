/**
 * 主布局组件 — 课件审核系统 v6.0
 *
 * v6.0 改版：与备课工坊 LPLayout 视觉风格统一
 * v110新增：学校管理员下拉菜单入口（senior_operator专属）
 */
import { useState, useRef, useEffect } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import Sidebar from './Sidebar'

const pageTitles: Record<string, string> = {
  '/':               '仪表盘',
  '/users':          '用户管理',
  '/ai-config':      'AI 配置',
  '/prompts':        '提示词管理',
  '/external-data':  '外部数据配置',
  '/courses':        '课程管理',
  '/pipelines':      'Pipeline',
  '/review':         '审核中心',
  '/settings':       '系统设置',
}

function getPageTitle(pathname: string): string {
  if (pageTitles[pathname]) return pageTitles[pathname]
  if (pathname.startsWith('/pipelines/')) return 'Pipeline 详情'
  return 'TE-DNA 2.0'
}

function DropdownPortal({ children, triggerRef }: {
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
      position: 'fixed', top: pos.top, right: pos.right, width: '220px',
      background: '#fff', borderRadius: '16px', border: '1px solid #E5E7EB',
      boxShadow: '0 8px 32px rgba(0,0,0,0.12)', overflow: 'hidden', zIndex: 9999,
    }}>
      {children}
    </div>
  )
}

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
    admin:           '系统管理员',
    senior_operator: '学校管理员',
    operator:        '骨干教师',
    viewer:          '普通教师',
  }

  return (
    <div ref={menuRef} style={{ position: 'relative' }}>
      <button
        onClick={() => setOpen(p => !p)}
        style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '5px 10px 5px 5px', borderRadius: '20px', border: '1px solid #E5E7EB', background: open ? '#F3F4F6' : 'transparent', cursor: 'pointer', transition: 'all 150ms ease' }}
        onMouseEnter={e => { if (!open) (e.currentTarget as HTMLElement).style.background = '#F9FAFB' }}
        onMouseLeave={e => { if (!open) (e.currentTarget as HTMLElement).style.background = 'transparent' }}
      >
        <div style={{ width: '30px', height: '30px', background: 'linear-gradient(135deg, #4F7BE8, #818CF8)', borderRadius: '50%', display: 'flex', alignItems: 'center', justifyContent: 'center', flexShrink: 0 }}>
          <span style={{ color: '#fff', fontSize: '12px', fontWeight: 700 }}>{user?.display_name?.charAt(0)?.toUpperCase() || 'U'}</span>
        </div>
        <span style={{ fontSize: '13px', fontWeight: 500, color: '#1F2937', maxWidth: '80px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{user?.display_name}</span>
        <span style={{ fontSize: '10px', color: '#9CA3AF', transform: open ? 'rotate(180deg)' : 'none', transition: 'transform 200ms ease' }}>▼</span>
      </button>

      {open && (
        <DropdownPortal triggerRef={menuRef}>
          <div style={{ padding: '14px 16px 12px', borderBottom: '1px solid #F3F4F6' }}>
            <div style={{ fontSize: '14px', fontWeight: 600, color: '#1F2937' }}>{user?.display_name}</div>
            <div style={{ fontSize: '12px', color: '#9CA3AF', marginTop: '2px' }}>
              @{user?.username} · {roleNames[user?.role || ''] || user?.role}
            </div>
          </div>

          <div style={{ padding: '6px' }}>
            <DropdownItem icon="👤" label="个人中心" onClick={() => go('/account', '/workflow')} />
            {/* admin 专属功能 */}
            {user?.role === 'admin' && (
              <>
                <DropdownItem icon="👥" label="用户管理" onClick={() => go('/admin', '/workflow')} highlight />
                <DropdownItem icon="🤖" label="AI 管理中心" onClick={() => go('/ai-center', '/workflow')} />
                <DropdownItem icon="📊" label="AI 调用统计" onClick={() => go('/ai-traces', '/workflow')} />
              </>
            )}
            {/* senior_operator 专属：学校管理 */}
            {user?.role === 'senior_operator' && (
              <DropdownItem icon="🏫" label="学校管理" onClick={() => go('/school-admin', '/workflow')} highlight />
            )}
          </div>

          <div style={{ height: '1px', background: '#F3F4F6', margin: '0 6px' }} />

          <div style={{ padding: '6px' }}>
            <DropdownItem icon="⏻" label="退出登录" onClick={handleLogout} danger />
          </div>
        </DropdownPortal>
      )}
    </div>
  )
}

function DropdownItem({ icon, label, onClick, danger, highlight }: {
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
        padding: '9px 12px', borderRadius: '10px', border: 'none',
        cursor: 'pointer', textAlign: 'left', fontSize: '14px',
        color: danger ? (hovered ? '#DC2626' : '#EF4444') : highlight ? '#4F7BE8' : (hovered ? '#1F2937' : '#6B7280'),
        background: hovered ? (danger ? 'rgba(239,68,68,0.06)' : highlight ? 'rgba(79,123,232,0.08)' : '#F9FAFB') : highlight ? 'rgba(79,123,232,0.04)' : 'transparent',
        transition: 'all 150ms ease',
      }}
    >
      <span style={{ fontSize: '16px', width: '20px', textAlign: 'center' }}>{icon}</span>
      <span style={{ fontWeight: highlight ? 600 : 400 }}>{label}</span>
    </button>
  )
}

export default function MainLayout() {
  const location = useLocation()
  const subPath = location.pathname.replace('/workflow', '') || '/'
  const pageTitle = getPageTitle(subPath)

  return (
    <div style={{ display: 'flex', height: '100vh', overflow: 'hidden', background: '#FAFBFC' }}>
      <Sidebar />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', overflow: 'hidden' }}>
        <header style={{ height: '64px', position: 'relative', zIndex: 50, background: 'rgba(255,255,255,0.85)', backdropFilter: 'blur(20px)', WebkitBackdropFilter: 'blur(20px)', borderBottom: '1px solid rgba(0,0,0,0.05)', display: 'flex', alignItems: 'center', padding: '0 32px', flexShrink: 0 }}>
          <h2 style={{ flex: 1, fontSize: '20px', fontWeight: 600, color: '#1F2937', margin: 0, letterSpacing: '-0.3px' }}>{pageTitle}</h2>
          <UserMenu />
        </header>
        <main style={{ flex: 1, overflowY: 'auto', padding: '28px 32px' }}>
          <Outlet />
        </main>
      </div>
    </div>
  )
}

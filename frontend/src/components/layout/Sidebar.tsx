/**
 * 侧边栏组件 - Apple 风格
 * - 深色毛玻璃背景
 * - 根据用户角色显示菜单
 * - 优雅的悬停和激活效果
 * - P3-1新增：外部数据配置菜单项
 */
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  LayoutDashboard,
  Users,
  Bot,
  FileText,
  Database,
  BookOpen,
  Workflow,
  ClipboardCheck,
  Settings,
  LogOut,
} from 'lucide-react'
import type { LucideIcon } from 'lucide-react'

// 菜单项类型
interface MenuItem {
  key: string
  label: string
  icon: LucideIcon
  path: string
  roles: string[]
}

// 菜单配置（P3-1新增外部数据配置）
const menuItems: MenuItem[] = [
  { key: 'dashboard', label: '仪表盘', icon: LayoutDashboard, path: '/', roles: ['admin', 'operator', 'viewer'] },
  { key: 'users', label: '用户管理', icon: Users, path: '/users', roles: ['admin'] },
  { key: 'ai-config', label: 'AI 配置', icon: Bot, path: '/ai-config', roles: ['admin'] },
  { key: 'prompts', label: '提示词管理', icon: FileText, path: '/prompts', roles: ['admin'] },
  { key: 'external-data', label: '外部数据配置', icon: Database, path: '/external-data', roles: ['admin'] },
  { key: 'courses', label: '课程管理', icon: BookOpen, path: '/courses', roles: ['admin', 'operator'] },
  { key: 'pipelines', label: 'Pipeline', icon: Workflow, path: '/pipelines', roles: ['admin', 'operator'] },
  { key: 'review', label: '审核中心', icon: ClipboardCheck, path: '/review', roles: ['admin', 'operator'] },
  { key: 'settings', label: '系统设置', icon: Settings, path: '/settings', roles: ['admin'] },
]

export default function Sidebar() {
  const location = useLocation()
  const navigate = useNavigate()
  const { user, logout } = useAuth()

  const visibleMenus = menuItems.filter(
    (item) => user && item.roles.includes(user.role)
  )

  const isActive = (path: string) => {
    if (path === '/') return location.pathname === '/'
    return location.pathname.startsWith(path)
  }

  const handleLogout = () => {
    logout()
    navigate('/login', { replace: true })
  }

  const getRoleName = (role: string) => {
    const map: Record<string, string> = { admin: '管理员', operator: '操作员', viewer: '查看者' }
    return map[role] || role
  }

  return (
    <aside style={{
      width: '240px',
      height: '100vh',
      display: 'flex',
      flexDirection: 'column',
      background: 'linear-gradient(180deg, #1a1a2e 0%, #16213e 100%)',
      borderRight: '1px solid rgba(255,255,255,0.06)',
      flexShrink: 0,
    }}>
      {/* Logo 区域 */}
      <div style={{
        height: '64px',
        display: 'flex',
        alignItems: 'center',
        padding: '0 20px',
        borderBottom: '1px solid rgba(255,255,255,0.06)',
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div style={{
            width: '34px',
            height: '34px',
            background: 'linear-gradient(135deg, #007aff, #5856d6)',
            borderRadius: '10px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            boxShadow: '0 2px 8px rgba(0,122,255,0.3)',
          }}>
            <span style={{ color: '#fff', fontSize: '14px', fontWeight: 700 }}>TE</span>
          </div>
          <div>
            <div style={{ color: '#fff', fontSize: '14px', fontWeight: 600, letterSpacing: '-0.3px' }}>TE-DNA 2.0</div>
            <div style={{ color: 'rgba(255,255,255,0.35)', fontSize: '11px', marginTop: '1px' }}>课程工作流</div>
          </div>
        </div>
      </div>

      {/* 菜单列表 */}
      <nav style={{ flex: 1, padding: '12px 10px', overflowY: 'auto' }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
          {visibleMenus.map((item) => {
            const Icon = item.icon
            const active = isActive(item.path)
            return (
              <button
                key={item.key}
                onClick={() => navigate(item.path)}
                style={{
                  width: '100%',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '12px',
                  padding: '10px 14px',
                  borderRadius: '10px',
                  border: 'none',
                  cursor: 'pointer',
                  fontSize: '14px',
                  fontWeight: active ? 500 : 400,
                  color: active ? '#fff' : 'rgba(255,255,255,0.55)',
                  background: active ? 'rgba(0,122,255,0.2)' : 'transparent',
                  transition: 'all 0.15s ease',
                  textAlign: 'left',
                }}
                onMouseEnter={(e) => {
                  if (!active) {
                    (e.currentTarget as HTMLElement).style.background = 'rgba(255,255,255,0.06)'
                    ;(e.currentTarget as HTMLElement).style.color = 'rgba(255,255,255,0.8)'
                  }
                }}
                onMouseLeave={(e) => {
                  if (!active) {
                    (e.currentTarget as HTMLElement).style.background = 'transparent'
                    ;(e.currentTarget as HTMLElement).style.color = 'rgba(255,255,255,0.55)'
                  }
                }}
              >
                <Icon size={18} strokeWidth={active ? 2 : 1.5} />
                <span>{item.label}</span>
              </button>
            )
          })}
        </div>
      </nav>

      {/* 底部用户信息 */}
      <div style={{
        padding: '12px',
        borderTop: '1px solid rgba(255,255,255,0.06)',
      }}>
        <div style={{
          display: 'flex',
          alignItems: 'center',
          gap: '10px',
          padding: '8px 10px',
        }}>
          <div style={{
            width: '32px',
            height: '32px',
            background: 'linear-gradient(135deg, #5856d6, #007aff)',
            borderRadius: '50%',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            flexShrink: 0,
          }}>
            <span style={{ color: '#fff', fontSize: '12px', fontWeight: 600 }}>
              {user?.display_name?.charAt(0) || 'U'}
            </span>
          </div>
          <div style={{ flex: 1, minWidth: 0 }}>
            <div style={{
              color: 'rgba(255,255,255,0.85)',
              fontSize: '13px',
              fontWeight: 500,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}>{user?.display_name || '未知用户'}</div>
            <div style={{
              color: 'rgba(255,255,255,0.35)',
              fontSize: '11px',
              marginTop: '1px',
            }}>{getRoleName(user?.role || '')}</div>
          </div>
          <button
            onClick={handleLogout}
            title="退出登录"
            style={{
              background: 'none',
              border: 'none',
              cursor: 'pointer',
              color: 'rgba(255,255,255,0.35)',
              padding: '4px',
              borderRadius: '6px',
              display: 'flex',
              transition: 'all 0.15s ease',
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLElement).style.color = '#ff453a'
              ;(e.currentTarget as HTMLElement).style.background = 'rgba(255,69,58,0.1)'
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLElement).style.color = 'rgba(255,255,255,0.35)'
              ;(e.currentTarget as HTMLElement).style.background = 'none'
            }}
          >
            <LogOut size={16} />
          </button>
        </div>
      </div>
    </aside>
  )
}

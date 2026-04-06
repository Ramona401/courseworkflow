/**
 * 侧边栏组件 — 课件审核系统 v6.1
 *
 * v6.1 变更：
 *   - 移除「用户管理」(/workflow/users) 菜单项
 *     → admin 请通过顶部Header下拉进入新版 /admin 用户管理中心
 *   - 移除「AI 配置」(/workflow/ai-config) 菜单项
 *     → admin 请通过顶部Header下拉进入新版 /ai-center AI管理中心
 *   - 旧路由页面文件保留不动，向后兼容
 *
 * v6.0 改版（延续）：
 *   - 白色背景侧边栏，与备课工坊 LPSidebar 视觉风格统一
 *   - Emoji 图标，激活态蓝色浅底+边框+小圆点
 *   - 悬停右移 2px 动效 + 显示描述文字
 *   - 底部只保留「返回入口」按钮
 */
import { useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'

// ==================== 菜单配置 ====================

interface MenuItem {
  key: string
  label: string
  icon: string
  path: string
  description: string
  roles: string[]
}

/**
 * 课件审核系统侧边栏菜单
 *
 * 说明：
 *   - 「用户管理」和「AI 配置」已移除，统一走顶部Header下拉的新版入口
 *   - 「提示词管理」保留：管理 Pipeline 各步骤专属提示词，课件审核系统专属
 *   - 「外部数据配置」保留：管理 OSS 等外部数据源，课件审核系统专属
 *   - 其余菜单均为课件审核核心流程功能
 */
const menuItems: MenuItem[] = [
  {
    key: 'dashboard',
    label: '仪表盘',
    icon: '📊',
    path: '/workflow',
    description: '系统运行总览',
    roles: ['admin', 'senior_operator', 'operator', 'viewer'],
  },
  {
    key: 'courses',
    label: '课程管理',
    icon: '📖',
    path: '/workflow/courses',
    description: '注册和管理课程',
    roles: ['admin', 'senior_operator', 'operator'],
  },
  {
    key: 'pipelines',
    label: 'Pipeline',
    icon: '⚙️',
    path: '/workflow/pipelines',
    description: 'AI 课件生成流水线',
    roles: ['admin', 'senior_operator', 'operator'],
  },
  {
    key: 'review',
    label: '审核中心',
    icon: '✅',
    path: '/workflow/review',
    description: '课件质量审核与定稿',
    roles: ['admin', 'senior_operator', 'operator'],
  },
  {
    key: 'prompts',
    label: '提示词管理',
    icon: '💬',
    path: '/workflow/prompts',
    description: 'Pipeline 各步骤提示词版本管理',
    roles: ['admin'],
  },
  {
    key: 'external-data',
    label: '外部数据配置',
    icon: '🗄️',
    path: '/workflow/external-data',
    description: 'OSS 等外部数据源配置',
    roles: ['admin'],
  },
  {
    key: 'settings',
    label: '系统设置',
    icon: '🔧',
    path: '/workflow/settings',
    description: '系统全局参数',
    roles: ['admin'],
  },
]

// ==================== 颜色常量（与 LPSidebar 统一）====================

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

// ==================== 主组件 ====================

export default function Sidebar() {
  const location = useLocation()
  const navigate = useNavigate()
  const { user } = useAuth()
  const [hoveredKey, setHoveredKey] = useState<string | null>(null)

  // 根据用户角色过滤菜单
  const visibleMenus = menuItems.filter(
    (item) => user && item.roles.includes(user.role)
  )

  // 判断当前路由是否匹配菜单项
  const isActive = (path: string) => {
    if (path === '/workflow') return location.pathname === '/workflow'
    return location.pathname.startsWith(path)
  }

  return (
    <aside style={{
      width: '260px',
      height: '100vh',
      display: 'flex',
      flexDirection: 'column',
      background: COLORS.bgSidebar,
      borderRight: `1px solid ${COLORS.border}`,
      flexShrink: 0,
    }}>

      {/* ── Logo 区域 ── */}
      <div style={{
        height: '64px',
        display: 'flex',
        alignItems: 'center',
        padding: '0 22px',
        borderBottom: `1px solid ${COLORS.border}`,
      }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <div style={{
            width: '36px',
            height: '36px',
            background: 'linear-gradient(135deg, #4F7BE8, #818CF8)',
            borderRadius: '10px',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            boxShadow: '0 2px 8px rgba(79,123,232,0.25)',
          }}>
            <span style={{ fontSize: '16px' }}>🎬</span>
          </div>
          <div>
            <div style={{
              color: COLORS.textPrimary,
              fontSize: '15px',
              fontWeight: 600,
              letterSpacing: '-0.3px',
            }}>
              课件审核
            </div>
            <div style={{
              color: COLORS.textMuted,
              fontSize: '11px',
              marginTop: '1px',
            }}>
              AI 课件生成与审核
            </div>
          </div>
        </div>
      </div>

      {/* ── 菜单导航 ── */}
      <nav style={{
        flex: 1,
        padding: '16px 12px',
        overflowY: 'auto',
      }}>
        <div style={{ display: 'flex', flexDirection: 'column', gap: '4px' }}>
          {visibleMenus.map((item) => {
            const active = isActive(item.path)
            const hovered = hoveredKey === item.key

            return (
              <button
                key={item.key}
                onClick={() => navigate(item.path)}
                onMouseEnter={() => setHoveredKey(item.key)}
                onMouseLeave={() => setHoveredKey(null)}
                style={{
                  width: '100%',
                  display: 'flex',
                  alignItems: 'center',
                  gap: '12px',
                  padding: '12px 16px',
                  borderRadius: '12px',
                  border: active
                    ? `1px solid ${COLORS.primaryBorder}`
                    : '1px solid transparent',
                  cursor: 'pointer',
                  fontSize: '15px',
                  fontWeight: active ? 600 : 400,
                  color: active ? COLORS.primary : COLORS.textSecondary,
                  background: active
                    ? COLORS.primaryLight
                    : hovered
                      ? COLORS.bgHover
                      : 'transparent',
                  transition: 'all 200ms ease',
                  textAlign: 'left',
                  transform: hovered && !active ? 'translateX(2px)' : 'none',
                }}
              >
                {/* Emoji 图标 */}
                <span style={{
                  fontSize: '18px',
                  width: '24px',
                  textAlign: 'center',
                  flexShrink: 0,
                }}>
                  {item.icon}
                </span>

                {/* 标签 + 悬停/激活时描述 */}
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div>{item.label}</div>
                  {(active || hovered) && (
                    <div style={{
                      fontSize: '11px',
                      color: active ? COLORS.primary : COLORS.textMuted,
                      marginTop: '2px',
                      opacity: 0.7,
                    }}>
                      {item.description}
                    </div>
                  )}
                </div>

                {/* 激活态小圆点指示器 */}
                {active && (
                  <div style={{
                    width: '6px',
                    height: '6px',
                    borderRadius: '50%',
                    background: COLORS.primary,
                    flexShrink: 0,
                  }} />
                )}
              </button>
            )
          })}
        </div>
      </nav>

      {/* ── 底部：返回入口（与 LPSidebar 结构一致）── */}
      <div style={{
        padding: '12px',
        borderTop: `1px solid ${COLORS.border}`,
      }}>
        <button
          onClick={() => navigate('/')}
          onMouseEnter={e => {
            (e.currentTarget as HTMLElement).style.background = COLORS.bgHover
          }}
          onMouseLeave={e => {
            (e.currentTarget as HTMLElement).style.background = 'transparent'
          }}
          style={{
            width: '100%',
            display: 'flex',
            alignItems: 'center',
            gap: '10px',
            padding: '10px 16px',
            borderRadius: '10px',
            border: 'none',
            cursor: 'pointer',
            fontSize: '13px',
            color: COLORS.textMuted,
            background: 'transparent',
            transition: 'all 200ms ease',
            textAlign: 'left',
          }}
        >
          <span style={{ fontSize: '14px' }}>←</span>
          <span>返回入口</span>
        </button>
      </div>
    </aside>
  )
}

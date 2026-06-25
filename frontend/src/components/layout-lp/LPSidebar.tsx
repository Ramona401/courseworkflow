/**
 * 教案系统侧边栏 — LPSidebar v9.0
 *
 * v9.0变更（优先级2 阶段B/C · 备课配方权限收敛 · Harness 生产者/消费者分离）：
 *   - 「备课配方」从「人人可见」收敛为仅 admin + senior_operator 可见。
 *     定位（Harness 单一职责分层）：配方 = 「本节课上下文层」（学情/校本要求/流程设定）的
 *     生产端，由管理员/教研员创建维护；普通老师是消费端，在备课起步（StartForm）一键选用
 *     现成配方即可，无需接触配方的创建/管理界面。
 *   - 路由层 App.tsx 同步给 recipes 四条路由加 RoleGuard 双重保护（光藏菜单不够防直接敲 URL）。
 *   - 注意：本改动只收敛「配方管理入口」，不影响 StartForm 里老师选用现成配方的能力（消费端保留）。
 *
 * v8.0变更（组件管理权限收敛）：
 *   - 菜单项可见性控制字段从布尔 `adminOnly` 升级为通用的 `roles?: string[]` 白名单。
 *     · 未声明 roles 的项 → 人人可见（行为与旧版一致）；
 *     · 声明 roles 的项 → 仅当 user.role 命中白名单才显示。
 *   - 「组件管理」从「人人可见」收敛为仅 admin + senior_operator 可见
 *     （路由层 App.tsx 同步加 RoleGuard 双重保护，光藏菜单不够防直接敲 URL 越权）。
 *   - 「阶段管理」由旧 `adminOnly: true` 平迁为 `roles: ['admin']`，行为完全等价。
 *
 * v7.0变更：新增"课本管理"菜单项（在组件管理之前）
 * v6.0变更：新增"备课配方"菜单项
 */
import { useState } from 'react'
import { useAuth } from '@/store/auth'
import { useLocation, useNavigate } from 'react-router-dom'

interface LPMenuItem {
  key: string
  label: string
  icon: string
  path: string
  description: string
  /**
   * 可见角色白名单。
   * - 不传（undefined）：人人可见；
   * - 传数组：仅当当前用户角色在数组内才渲染该菜单项。
   */
  roles?: string[]
}

const menuItems: LPMenuItem[] = [
  { key: 'workshop',     label: '备课工坊',     icon: '✨', path: '/lesson-plans',                description: 'AI辅助对话式备课' },
  { key: 'my-assistants', label: '我的 AI 助手', icon: '🤖', path: '/lesson-plans/my-assistants', description: '和AI聊着造你的备课助手' },
  // 我的备课资料：统一资料中心（Tab 装课程大纲/班级学情/单元方案）。建/改权限由各 Tab 内数据 + 后端兜底
  { key: 'resources', label: '我的备课资料', icon: '📂', path: '/lesson-plans/resources', description: '课程大纲·班级学情·单元方案', roles: ['admin', 'senior_operator', 'operator'] },
  // 备课配方：收敛为仅 admin + senior_operator 可见（管理员/教研员为生产者；普通老师在备课起步选用现成配方即可）
  { key: 'recipes',    label: '备课配方',   icon: '📦', path: '/lesson-plans/recipes',    description: '可复用的AI备课预设包', roles: ['admin', 'senior_operator'] },
  { key: 'my-plans',   label: '我的教案',   icon: '📋', path: '/lesson-plans/my-plans',   description: '个人教案管理' },
  { key: 'library',    label: '教案库',     icon: '📚', path: '/lesson-plans/library',    description: '教研组共享教案' },
  { key: 'review',     label: '评审中心',   icon: '📝', path: '/lesson-plans/review',     description: '人工评审教案' },
  { key: 'review-v2', label: '多级审核',   icon: '🔍', path: '/lesson-plans/review-v2', description: '三级审核工作台' },
  { key: 'textbooks',  label: '课本管理',   icon: '📷', path: '/lesson-plans/textbooks',  description: '上传课本图片供AI精准备课' },
  // 课程大纲：组长/校管/老师可见入口；建/改权限由页面内 my-groups 数据 + 后端 service 兜底
  // 组件管理：收敛为仅 admin + senior_operator 可见（普通老师无需接触组件库底层）
  { key: 'components', label: '组件管理',   icon: '🧩', path: '/lesson-plans/components', description: '教学设计组件库', roles: ['admin', 'senior_operator'] },
  { key: 'templates',  label: '提示词模板', icon: '📐', path: '/lesson-plans/templates',  description: '分层提示词模板配置' },
  { key: 'tokens', label: '积分管理', icon: '🪙', path: '/lesson-plans/tokens', description: 'Token积分配额管理' },
  // 阶段管理：仅 admin（由旧 adminOnly:true 平迁，行为等价）
  { key: 'stages-config', label: '阶段管理', icon: '⚙️', path: '/lesson-plans/stages-config', description: '备课阶段流程配置', roles: ['admin'] },
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

  const isActive = (path: string) => {
    if (path === '/lesson-plans') return location.pathname === '/lesson-plans'
    // 精确匹配 /review 避免和 /review-v2 互相干扰
    if (path === '/lesson-plans/review') {
      return location.pathname === '/lesson-plans/review' || (location.pathname.startsWith('/lesson-plans/review/') && !location.pathname.startsWith('/lesson-plans/review-v2'))
    }
    return location.pathname.startsWith(path)
  }

  // 角色可见性过滤：未声明 roles 的项人人可见；声明 roles 的项需当前角色命中白名单
  const visibleItems = menuItems.filter(item => !item.roles || (user?.role ? item.roles.includes(user.role) : false))

  return (
    <aside style={{
      width: '260px', height: '100vh',
      display: 'flex', flexDirection: 'column',
      background: COLORS.bgSidebar,
      borderRight: `1px solid ${COLORS.border}`,
      flexShrink: 0,
    }}>
      {/* Logo + 系统名区域 */}
      <div style={{
        display: 'flex', flexDirection: 'column',
        borderBottom: `1px solid ${COLORS.border}`,
      }}>
        {/* 北大实验室 logo */}
        <div
          style={{ padding: '12px 18px 6px', cursor: 'pointer' }}
          onClick={() => navigate('/')}
          title="返回首页"
        >
          {user?.org_logo_url ? (
            <img src={user.org_logo_url} alt={user.org_name || 'Logo'} style={{ height: '26px', objectFit: 'contain', display: 'block' }} />
          ) : (
            <div style={{ fontSize: '14px', fontWeight: 700, color: '#4F7BE8', letterSpacing: '1px' }}>TE-DNA 2.0</div>
          )}
        </div>
        {/* 系统名称 */}
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
          {visibleItems.map((item) => {
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

      {/* 底部：返回入口按钮 */}
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

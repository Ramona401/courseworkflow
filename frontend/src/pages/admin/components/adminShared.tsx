/**
 * adminShared.tsx — Admin页面共用小组件
 *   Toast / RoleBadge / StatusBadge / StatCard
 */
import { useEffect } from 'react'
import { C } from './adminConstants'

// ==================== Toast 通知 ====================
export function Toast({ message, type, onClose }: {
  message: string
  type: 'success' | 'error'
  onClose: () => void
}) {
  // 3.5秒后自动关闭
  useEffect(() => {
    const t = setTimeout(onClose, 3500)
    return () => clearTimeout(t)
  }, [onClose])

  return (
    <div style={{
      position: 'fixed', top: '24px', right: '24px', zIndex: 9999,
      padding: '12px 20px', borderRadius: '12px', color: '#fff',
      fontSize: '14px', fontWeight: 500,
      background: type === 'success'
        ? 'linear-gradient(135deg,#10B981,#059669)'
        : 'linear-gradient(135deg,#EF4444,#DC2626)',
      boxShadow: '0 4px 16px rgba(0,0,0,0.15)',
    }}>
      {type === 'success' ? '✓ ' : '✕ '}{message}
    </div>
  )
}

// ==================== 角色徽章 ====================
export function RoleBadge({ role, roleName }: { role: string; roleName?: string }) {
  const styleMap: Record<string, { bg: string; color: string }> = {
    admin:           { bg: C.dangerLight,  color: C.danger  },
    senior_operator: { bg: C.warningLight, color: C.warning },
    operator:        { bg: C.primaryLight, color: C.primary },
    viewer:          { bg: C.bg,           color: C.textSec },
  }
  const nameMap: Record<string, string> = {
    admin: '管理员', senior_operator: '高级操作员',
    operator: '操作员', viewer: '查看者',
  }
  const s = styleMap[role] || { bg: C.bg, color: C.textSec }
  return (
    <span style={{
      display: 'inline-block', padding: '2px 10px', borderRadius: '12px',
      fontSize: '12px', fontWeight: 600, background: s.bg, color: s.color,
    }}>
      {roleName || nameMap[role] || role}
    </span>
  )
}

// ==================== 状态徽章 ====================
export function StatusBadge({ status }: { status: string }) {
  const active = status === 'active'
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: '4px',
      padding: '2px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 600,
      background: active ? C.successLight : C.dangerLight,
      color:      active ? C.success      : C.danger,
    }}>
      <span style={{
        width: '6px', height: '6px', borderRadius: '50%',
        background: active ? C.success : C.danger,
      }} />
      {active ? '正常' : '已禁用'}
    </span>
  )
}

// ==================== 统计卡片 ====================
export function StatCard({ label, value, sub, color }: {
  label: string
  value: number
  sub?: string
  color?: string
}) {
  return (
    <div style={{
      background: C.white, borderRadius: '14px',
      border: `1px solid ${C.border}`, padding: '20px 24px',
      flex: 1, boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
    }}>
      <div style={{ fontSize: '13px', color: C.textSec, marginBottom: '8px' }}>{label}</div>
      <div style={{ fontSize: '28px', fontWeight: 700, color: color || C.text, lineHeight: 1 }}>{value}</div>
      {sub && <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '6px' }}>{sub}</div>}
    </div>
  )
}

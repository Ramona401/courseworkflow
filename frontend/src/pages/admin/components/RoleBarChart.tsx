/**
 * RoleBarChart.tsx — 角色分布横条图（纯CSS入场动画）
 *
 * 角色名与学校体系对齐：
 *   管理员 / 学校管理员 / 骨干教师 / 普通教师
 */
import { useState, useEffect } from 'react'
import type { AdminStats } from '@/api/admin'
import { C } from './adminConstants'

interface RoleBarChartProps {
  stats: AdminStats
}

export function RoleBarChart({ stats }: RoleBarChartProps) {
  const roles = [
    { label: '系统管理员', count: stats.admin_count,           color: C.danger  },
    { label: '学校管理员', count: stats.senior_operator_count, color: C.warning },
    { label: '骨干教师',   count: stats.operator_count,        color: C.primary },
    { label: '普通教师',   count: stats.viewer_count,          color: C.textSec },
  ]
  const total    = stats.total_users || 1
  const maxCount = Math.max(...roles.map(r => r.count), 1)

  // 入场动画：挂载后50ms触发
  const [animated, setAnimated] = useState(false)
  useEffect(() => {
    const t = setTimeout(() => setAnimated(true), 50)
    return () => clearTimeout(t)
  }, [])

  return (
    <div style={{
      background: C.white, borderRadius: '16px', border: `1px solid ${C.border}`,
      padding: '24px', marginBottom: '16px', boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
    }}>
      {/* 标题行 */}
      <div style={{
        fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '20px',
        display: 'flex', justifyContent: 'space-between',
      }}>
        <span>角色分布</span>
        <span style={{ fontSize: '12px', color: C.textMuted, fontWeight: 400 }}>
          共 {stats.total_users} 人
        </span>
      </div>

      {/* 横条图 */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '14px' }}>
        {roles.map(role => {
          const barPct   = (role.count / maxCount) * 100
          const totalPct = ((role.count / total) * 100).toFixed(1)
          return (
            <div key={role.label} style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              {/* 角色标签 */}
              <div style={{ width: '80px', flexShrink: 0, fontSize: '13px', color: C.textSec, textAlign: 'right', fontWeight: 500 }}>
                {role.label}
              </div>
              {/* 横条 */}
              <div style={{ flex: 1, height: '24px', background: C.bg, borderRadius: '6px', overflow: 'hidden', position: 'relative' }}>
                <div style={{
                  position: 'absolute', left: 0, top: 0, bottom: 0,
                  width: animated ? `${barPct}%` : '0%',
                  background: role.color, borderRadius: '6px', opacity: 0.85,
                  transition: 'width 500ms cubic-bezier(0.4,0,0.2,1)',
                }} />
                {role.count > 0 && barPct > 18 && (
                  <div style={{
                    position: 'absolute', left: '10px', top: 0, bottom: 0,
                    display: 'flex', alignItems: 'center',
                    fontSize: '11px', fontWeight: 600, color: '#fff',
                    opacity: animated ? 1 : 0,
                    transition: 'opacity 400ms ease 200ms',
                  }}>
                    {role.count} 人
                  </div>
                )}
              </div>
              {/* 右侧数字 */}
              <div style={{ width: '76px', flexShrink: 0, display: 'flex', flexDirection: 'column', alignItems: 'flex-end' }}>
                <span style={{ fontSize: '14px', fontWeight: 700, color: role.color }}>{role.count}</span>
                <span style={{ fontSize: '11px', color: C.textMuted }}>{totalPct}%</span>
              </div>
            </div>
          )
        })}
      </div>

      {/* 图例 */}
      <div style={{
        marginTop: '16px', paddingTop: '12px', borderTop: `1px solid ${C.border}`,
        fontSize: '12px', color: C.textMuted,
        display: 'flex', gap: '20px', flexWrap: 'wrap',
      }}>
        {roles.map(r => (
          <span key={r.label} style={{ display: 'flex', alignItems: 'center', gap: '5px' }}>
            <span style={{ display: 'inline-block', width: '8px', height: '8px', borderRadius: '2px', background: r.color }} />
            {r.label}：{r.count} 人
          </span>
        ))}
      </div>
    </div>
  )
}

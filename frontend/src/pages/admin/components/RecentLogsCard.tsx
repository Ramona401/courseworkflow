/**
 * RecentLogsCard.tsx — 概览Tab最近操作日志快览卡
 */
import type { AuditLogItem } from '@/api/admin'
import { C, formatRelativeTime, getActionStyle } from './adminConstants'

interface RecentLogsCardProps {
  logs: AuditLogItem[]
  loading: boolean
  onViewAll: () => void
}

export function RecentLogsCard({ logs, loading, onViewAll }: RecentLogsCardProps) {
  return (
    <div style={{
      background: C.white, borderRadius: '16px', border: `1px solid ${C.border}`,
      padding: '24px', marginBottom: '16px', boxShadow: '0 1px 4px rgba(0,0,0,0.04)',
    }}>
      {/* 标题行 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
        <div style={{ fontSize: '15px', fontWeight: 600, color: C.text }}>最近操作</div>
        <button
          onClick={onViewAll}
          style={{
            padding: '5px 14px', borderRadius: '8px', cursor: 'pointer',
            border: `1px solid ${C.border}`, background: C.bg,
            fontSize: '12px', color: C.primary, fontWeight: 500,
          }}
          onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = C.primaryLight }}
          onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = C.bg }}>
          查看全部 →
        </button>
      </div>

      {/* 骨架屏 */}
      {loading ? (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
          {[1, 2, 3].map(i => (
            <div key={i} style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
              <div style={{ width: '32px', height: '32px', borderRadius: '50%', background: C.border, flexShrink: 0 }} />
              <div style={{ flex: 1 }}>
                <div style={{ height: '12px', borderRadius: '4px', background: C.border, width: `${50 + i * 10}%`, marginBottom: '6px' }} />
                <div style={{ height: '10px', borderRadius: '4px', background: C.bg, width: '40%' }} />
              </div>
            </div>
          ))}
        </div>
      ) : logs.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '24px', color: C.textMuted, fontSize: '13px' }}>
          暂无操作记录
        </div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column' }}>
          {logs.map((log, idx) => {
            const s = getActionStyle(log.action)
            // 解析detail摘要
            let dd = ''
            try {
              const d = JSON.parse(log.detail)
              const entries = Object.entries(d)
              if (entries.length) { const [k, v] = entries[0]; dd = `${k}: ${v}` }
            } catch { dd = log.detail || '' }

            return (
              <div
                key={log.id}
                style={{
                  display: 'flex', alignItems: 'center', gap: '12px',
                  padding: '10px 0',
                  borderBottom: idx < logs.length - 1 ? `1px solid ${C.border}` : 'none',
                }}>
                {/* 头像 */}
                <div style={{
                  width: '32px', height: '32px', borderRadius: '50%', flexShrink: 0,
                  background: `linear-gradient(135deg,${C.primary},#7C3AED)`,
                  display: 'flex', alignItems: 'center', justifyContent: 'center',
                  color: '#fff', fontSize: '12px', fontWeight: 700,
                }}>
                  {(log.display_name || log.username || '?').charAt(0).toUpperCase()}
                </div>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
                    <span style={{ fontSize: '13px', fontWeight: 600, color: C.text }}>
                      {log.display_name || log.username}
                    </span>
                    <span style={{
                      padding: '1px 7px', borderRadius: '5px', fontSize: '11px', fontWeight: 600,
                      background: s.bg, color: s.color,
                    }}>
                      {log.action_name}
                    </span>
                  </div>
                  {dd && (
                    <div style={{
                      fontSize: '11px', color: C.textMuted, marginTop: '2px',
                      fontFamily: 'monospace', overflow: 'hidden',
                      textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                    }}>
                      {dd}
                    </div>
                  )}
                </div>
                {/* 相对时间 */}
                <div style={{ flexShrink: 0, fontSize: '11px', color: C.textMuted, whiteSpace: 'nowrap' }}>
                  {formatRelativeTime(
                    typeof log.created_at === 'string'
                      ? log.created_at
                      : new Date(log.created_at).toISOString()
                  )}
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

/**
 * 共享课件库卡片 — SharedCWCard.tsx
 *
 * 从 CoursewareListPage.tsx 拆出的"共享课件库"列表卡片组件。
 * 展示他人共享给"我"(同校/同组)的课件:
 *   标题 + 代码范围徽章 + 来源标签 + 作者名 + 学校名 + 学科/年级/页数 + 创建日期。
 *
 * 复制到我的(Fork):
 *   - can_copy=true 显「📋 复制到我的」,点击调 forkCourseware,
 *     成功后把新课件 id 交回父组件(父组件弹确认框问是否立即去编辑)。
 *   - can_copy=false 显灰字「🔒 作者未开放复制」。
 * forking 态内聚在卡片内,不外溢。
 */

import { useState } from 'react'
import { CW_CODE_SHARE_SCOPE_CONFIG, forkCourseware } from '@/api/coursewares'
import type { SharedCoursewareListItem } from '@/api/coursewares'
import { SOURCE_CONFIG } from './listConstants'

export default function SharedCWCard({ item, onClick, onForked }: {
  item: SharedCoursewareListItem
  onClick: () => void
  onForked: (newId: string) => void
}) {
  const [hovered, setHovered] = useState(false)
  const [forking, setForking] = useState(false)
  const src = SOURCE_CONFIG[item.source_type] || SOURCE_CONFIG.lesson_plan
  const codeCfg = CW_CODE_SHARE_SCOPE_CONFIG[item.code_share_scope] || CW_CODE_SHARE_SCOPE_CONFIG.none

  // 复制到我的:调 forkCourseware,成功后把新 id 交回父组件(父组件弹确认框问是否去编辑)
  const handleFork = async (e: React.MouseEvent) => {
    e.stopPropagation()
    if (forking) return
    setForking(true)
    try {
      const r = await forkCourseware(item.id)
      onForked(r.id)
    } catch (err) {
      alert('复制失败: ' + (err instanceof Error ? err.message : '未知错误'))
    } finally { setForking(false) }
  }

  return (
    <div onClick={onClick} onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)}
      style={{
        background: '#fff', borderRadius: '12px', padding: '20px',
        border: `1px solid ${hovered ? 'rgba(37,99,235,0.3)' : '#E5E7EB'}`,
        cursor: 'pointer', transition: 'all 200ms',
        transform: hovered ? 'translateY(-2px)' : 'none',
        boxShadow: hovered ? '0 4px 16px rgba(0,0,0,0.08)' : '0 1px 3px rgba(0,0,0,0.04)',
      }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '12px' }}>
        <div style={{ fontSize: '16px', fontWeight: 600, color: '#1F2937', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{item.title}</div>
        <span style={{ padding: '2px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 500, color: codeCfg.color, background: codeCfg.bg, flexShrink: 0, marginLeft: '8px' }}>{codeCfg.short}</span>
      </div>
      {/* 作者 + 学校 */}
      <div style={{ fontSize: '13px', color: '#6B7280', marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
        <span style={{ padding: '1px 8px', borderRadius: '8px', fontSize: '11px', color: src.color, background: src.bg }}>{src.emoji} {src.label}</span>
        {item.author_name && <span>👤 {item.author_name}</span>}
        {item.school_name && <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>🏫 {item.school_name}</span>}
      </div>
      <div style={{ display: 'flex', gap: '16px', fontSize: '12px', color: '#9CA3AF' }}>
        {item.subject && <span>📚 {item.subject}</span>}
        {item.grade && <span>🎓 {item.grade}</span>}
        <span>📄 {item.page_count} 页</span>
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '14px', paddingTop: '12px', borderTop: '1px solid #E5E7EB' }}>
        <span style={{ fontSize: '12px', color: '#9CA3AF' }}>{item.created_at ? new Date(item.created_at).toLocaleDateString('zh-CN') : ''}</span>
        {/* can_copy=true 显「复制到我的」;否则显灰字提示作者未开放 */}
        {item.can_copy ? (
          <button onClick={handleFork} disabled={forking}
            style={{ padding: '4px 14px', borderRadius: '6px', border: 'none', background: forking ? '#93C5FD' : 'linear-gradient(135deg, #2563EB, #3B82F6)', color: '#fff', fontSize: '12px', fontWeight: 600, cursor: forking ? 'default' : 'pointer' }}>
            {forking ? '复制中…' : '📋 复制到我的'}
          </button>
        ) : (
          <span style={{ fontSize: '12px', color: '#9CA3AF' }}>🔒 作者未开放复制</span>
        )}
      </div>
    </div>
  )
}

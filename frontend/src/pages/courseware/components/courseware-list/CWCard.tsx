/**
 * 我的课件卡片 — CWCard.tsx
 *
 * 从 CoursewareListPage.tsx 拆出的"我的课件"列表卡片组件。
 * 展示:标题 + 发布态徽章(private 不显) + 生产状态徽章 + 来源标签 +
 *       代码开放范围标签(非 none 才显) + 学科/年级/页数 + 创建日期 +
 *       操作区(提交审核/发布/分享、下载离线包、删除)。
 *
 * 纯展示 + 回调组件,不持有列表级状态:
 *   - onClick        点卡片进课件工坊
 *   - onPublish      点"发布/分享"打开发布面板(父组件持有 publishTarget)
 *   - onSubmitReview 点"提交审核"由父组件调 submitCoursewareForReview(阶段3新增)
 *   - onDelete       点"删除"由父组件弹二次确认
 * 下载离线包逻辑(downloading 态)内聚在卡片内,不外溢。
 *
 * 阶段3 新增「提交审核」按钮：
 *   - 仅在课件【已生成】(status≥preview, 用 SHAREABLE_STATUSES 判定) 才有意义；
 *   - 且发布态须 ∈ {private, published_personal, revision}(与后端 SubmitForReview 校验一致)——
 *     submitted(审核中)/approved(待发布)/published_shared(已共享) 时不显示，避免重复提交；
 *   - publish_state 缺省视为 private(存量数据 DEFAULT)，故无该字段也按可提交处理。
 */

import { useState } from 'react'
import { CW_STATUS_CONFIG, CW_PUBLISH_STATE_CONFIG, CW_CODE_SHARE_SCOPE_CONFIG, downloadCoursewareBundle } from '@/api/coursewares'
import type { CoursewareListItem } from '@/api/coursewares'
import { SOURCE_CONFIG, SHAREABLE_STATUSES } from './listConstants'

// 允许提交审核的发布态白名单（与后端 SubmitForReview 校验一致）
const SUBMITTABLE_PUBLISH_STATES = ['private', 'published_personal', 'revision']

export default function CWCard({ item, onDelete, onClick, onPublish, onSubmitReview }: {
  item: CoursewareListItem
  onDelete: (id: string, t: string) => void
  onClick: () => void
  onPublish: () => void
  onSubmitReview: (id: string, t: string) => void
}) {
  const [hovered, setHovered] = useState(false)
  const [downloading, setDownloading] = useState(false)

  // 生产状态徽章配置(取不到则灰底兜底)
  const sc = CW_STATUS_CONFIG[item.status] || { label: item.status, color: '#6B7280', bg: '#F3F4F6' }
  // 来源标签配置(取不到回退教案生成)
  const src = SOURCE_CONFIG[item.source_type] || SOURCE_CONFIG.lesson_plan
  // 发布态徽章:private 默认态不显,其余显
  const pub = (item.publish_state && item.publish_state !== 'private')
    ? (CW_PUBLISH_STATE_CONFIG[item.publish_state] || null)
    : null
  // 代码范围标签:非 none 才显
  const codeCfg = (item.code_share_scope && item.code_share_scope !== 'none')
    ? (CW_CODE_SHARE_SCOPE_CONFIG[item.code_share_scope] || null)
    : null
  // 下载离线包仅在课件已生成可预览之后的状态下才有意义
  const canDownload = SHAREABLE_STATUSES.includes(item.status)
  // 发布/分享按钮仅在课件已生成(preview 及以上)才显示——与后端共享规则一致
  const canPublish = SHAREABLE_STATUSES.includes(item.status)
  // 提交审核：①课件已生成(status≥preview) ②发布态在可提交白名单(缺省=private 视为可提交)
  const effPublishState = item.publish_state || 'private'
  const canSubmitReview = SHAREABLE_STATUSES.includes(item.status) && SUBMITTABLE_PUBLISH_STATES.includes(effPublishState)

  return (
    <div onClick={onClick} onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)}
      style={{
        background: '#fff', borderRadius: '12px', padding: '20px',
        border: `1px solid ${hovered ? 'rgba(245,158,11,0.3)' : '#E5E7EB'}`,
        cursor: 'pointer', transition: 'all 200ms',
        transform: hovered ? 'translateY(-2px)' : 'none',
        boxShadow: hovered ? '0 4px 16px rgba(0,0,0,0.08)' : '0 1px 3px rgba(0,0,0,0.04)',
      }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '12px' }}>
        <div style={{ fontSize: '16px', fontWeight: 600, color: '#1F2937', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{item.title}</div>
        {/* 右上角徽章区:生产状态 + 发布态(叠加,发布态在生产态左侧) */}
        <div style={{ display: 'flex', gap: '6px', flexShrink: 0, marginLeft: '8px' }}>
          {pub && <span style={{ padding: '2px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 500, color: pub.color, background: pub.bg }}>{pub.label}</span>}
          <span style={{ padding: '2px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 500, color: sc.color, background: sc.bg }}>{sc.label}</span>
        </div>
      </div>
      <div style={{ fontSize: '13px', color: '#6B7280', marginBottom: '8px', display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap' }}>
        <span style={{ padding: '1px 8px', borderRadius: '8px', fontSize: '11px', color: src.color, background: src.bg }}>{src.emoji} {src.label}</span>
        {/* 代码开放范围标签(非 none 才显) */}
        {codeCfg && <span style={{ padding: '1px 8px', borderRadius: '8px', fontSize: '11px', color: codeCfg.color, background: codeCfg.bg }}>{codeCfg.short}</span>}
        {item.lesson_plan_title && <span style={{ overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>📝 {item.lesson_plan_title}</span>}
      </div>
      <div style={{ display: 'flex', gap: '16px', fontSize: '12px', color: '#9CA3AF' }}>
        {item.subject && <span>📚 {item.subject}</span>}
        {item.grade && <span>🎓 {item.grade}</span>}
        <span>📄 {item.page_count} 页</span>
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '14px', paddingTop: '12px', borderTop: '1px solid #E5E7EB' }}>
        {/* 左侧:创建日期 */}
        <span style={{ fontSize: '12px', color: '#9CA3AF' }}>{item.created_at ? new Date(item.created_at).toLocaleDateString('zh-CN') : ''}</span>
        {/* 右侧:操作按钮区——提交审核 + 发布/分享(preview+)+ 下载离线包(preview+)+ 删除(所有状态) */}
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flexWrap: 'wrap', justifyContent: 'flex-end' }}>
          {canSubmitReview && (
            <button
              onClick={e => { e.stopPropagation(); onSubmitReview(item.id, item.title) }}
              style={{ padding: '2px 10px', borderRadius: '6px', border: '1px solid #FED7AA', background: 'transparent', color: '#D97706', fontSize: '12px', cursor: 'pointer' }}>
              🛡️ 提交审核
            </button>
          )}
          {canPublish && (
            <button
              onClick={e => { e.stopPropagation(); onPublish() }}
              style={{ padding: '2px 10px', borderRadius: '6px', border: '1px solid #FED7AA', background: 'transparent', color: '#EA580C', fontSize: '12px', cursor: 'pointer' }}>
              🔗 发布/分享
            </button>
          )}
          {canDownload && (
            <button
              onClick={async e => {
                e.stopPropagation()
                if (downloading) return
                setDownloading(true)
                try { await downloadCoursewareBundle(item.id, item.title) }
                catch (err) { alert('下载失败: ' + (err instanceof Error ? err.message : '未知错误')) }
                finally { setDownloading(false) }
              }}
              disabled={downloading}
              style={{ padding: '2px 10px', borderRadius: '6px', border: '1px solid #BFDBFE', background: downloading ? '#EFF6FF' : 'transparent', color: '#2563EB', fontSize: '12px', cursor: downloading ? 'default' : 'pointer' }}>
              {downloading ? '⏳ 打包中…' : '⬇ 下载离线包'}
            </button>
          )}
          {/* 删除按钮:对所有状态可见,点击后由父组件 handleDelete 弹出二次确认 */}
          <button onClick={e => { e.stopPropagation(); onDelete(item.id, item.title) }}
            style={{ padding: '2px 10px', borderRadius: '6px', border: '1px solid #FECACA', background: 'transparent', color: '#EF4444', fontSize: '12px', cursor: 'pointer' }}>
            删除
          </button>
        </div>
      </div>
    </div>
  )
}

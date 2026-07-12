/**
 * AnnotationPanel.tsx — 课件页级批注面板（阶段2新建）
 *
 * 挂在 PagePreviewBlock 预览块下方，展示「当前选中页」的批注：
 *   - 列表：当前页全部批注（待处理在前、已处理灰显在后），含批注人/时间/内容/状态
 *   - 新增：底部输入框 + 提交按钮，对当前页发表批注
 *   - 操作：每条批注可标记「已处理 / 重新待处理」、可删除（后端按权限裁决，失败给提示）
 *
 * 数据流：本组件不持有批注全集，由父组件 PagePreviewBlock 透传 annotations(已按当前页过滤前的全集)
 * 与 currentNum，本组件内部按 currentNum 过滤出当前页批注；增删改后调 onChanged() 让父级重拉。
 *
 * 权限：前端不做拦截，统一交后端裁决。操作失败(403等)以 alert 提示，不静默。
 */
import { useState } from 'react'
import { C } from './workshopConstants'
import {
  createCWAnnotation,
  resolveCWAnnotation,
  deleteCWAnnotation,
  type CoursewareAnnotation,
} from '@/api/coursewares'

interface Props {
  /** 课件ID */
  coursewareId: string
  /** 当前选中页号 */
  pageNumber: number
  /** 批注全集（父组件加载，本组件按 pageNumber 过滤当前页） */
  annotations: CoursewareAnnotation[]
  /** 增删改后回调，父组件据此重拉批注全集刷新 */
  onChanged: () => void
}

/** 简短时间格式：YYYY-MM-DD HH:mm */
function fmtTime(iso: string): string {
  if (!iso) return ''
  const d = new Date(iso)
  if (isNaN(d.getTime())) return ''
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`
}

export default function AnnotationPanel({ coursewareId, pageNumber, annotations, onChanged }: Props) {
  const [input, setInput] = useState('')
  const [submitting, setSubmitting] = useState(false)
  const [busyId, setBusyId] = useState('')  // 正在标记/删除的批注ID

  // 过滤出当前页批注：待处理(pending)在前，已处理(resolved)在后，archived 不显示
  const pageItems = annotations
    .filter(a => a.page_number === pageNumber && a.status !== 'archived')
    .sort((a, b) => {
      const wa = a.status === 'pending' ? 0 : 1
      const wb = b.status === 'pending' ? 0 : 1
      if (wa !== wb) return wa - wb
      return a.created_at < b.created_at ? -1 : 1
    })

  const pendingCount = pageItems.filter(a => a.status === 'pending').length

  // 提交新批注
  const handleSubmit = async () => {
    const text = input.trim()
    if (!text || submitting || pageNumber <= 0) return
    setSubmitting(true)
    try {
      await createCWAnnotation(coursewareId, pageNumber, text)
      setInput('')
      onChanged()
    } catch (e: any) {
      alert(e?.response?.data?.message || e?.message || '批注提交失败')
    } finally {
      setSubmitting(false)
    }
  }

  // 标记已处理 / 重新待处理
  const handleToggle = async (a: CoursewareAnnotation) => {
    if (busyId) return
    setBusyId(a.id)
    try {
      const next = a.status === 'resolved' ? 'pending' : 'resolved'
      await resolveCWAnnotation(a.id, next)
      onChanged()
    } catch (e: any) {
      alert(e?.response?.data?.message || e?.message || '操作失败')
    } finally {
      setBusyId('')
    }
  }

  // 删除批注
  const handleDelete = async (a: CoursewareAnnotation) => {
    if (busyId) return
    if (!window.confirm('确定删除这条批注吗？此操作不可恢复。')) return
    setBusyId(a.id)
    try {
      await deleteCWAnnotation(a.id)
      onChanged()
    } catch (e: any) {
      alert(e?.response?.data?.message || e?.message || '删除失败')
    } finally {
      setBusyId('')
    }
  }

  if (pageNumber <= 0) return null

  return (
    <div style={{ marginTop: 16, borderRadius: 12, border: `1px solid ${C.border}`, background: C.white, overflow: 'hidden' }}>
      {/* 标题栏 */}
      <div style={{ padding: '10px 14px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between', background: '#FAFAFA' }}>
        <div style={{ fontSize: 14, fontWeight: 600, color: C.textPrimary }}>
          💬 第 {pageNumber} 页批注
          {pendingCount > 0 && (
            <span style={{ marginLeft: 8, padding: '1px 8px', borderRadius: 10, background: '#FEE2E2', color: '#DC2626', fontSize: 12, fontWeight: 600 }}>{pendingCount} 条待处理</span>
          )}
        </div>
        <span style={{ fontSize: 12, color: C.textSecondary }}>共 {pageItems.length} 条</span>
      </div>

      {/* 批注列表 */}
      <div style={{ maxHeight: 320, overflowY: 'auto', padding: pageItems.length > 0 ? '8px 0' : 0 }}>
        {pageItems.length === 0 ? (
          <div style={{ padding: '20px 14px', textAlign: 'center', color: C.textSecondary, fontSize: 13 }}>这一页还没有批注，在下方写下第一条吧。</div>
        ) : (
          pageItems.map(a => {
            const resolved = a.status === 'resolved'
            return (
              <div key={a.id} style={{ padding: '10px 14px', borderBottom: `1px solid ${C.border}`, opacity: resolved ? 0.6 : 1 }}>
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 4 }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                    <span style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>{a.reviewer_name || '匿名'}</span>
                    <span style={{ fontSize: 11, color: C.textSecondary }}>{fmtTime(a.created_at)}</span>
                    {resolved && <span style={{ fontSize: 11, color: '#059669', fontWeight: 600 }}>✓ 已处理</span>}
                  </div>
                  <div style={{ display: 'flex', gap: 6 }}>
                    <button onClick={() => handleToggle(a)} disabled={busyId === a.id} title={resolved ? '重新标记为待处理' : '标记为已处理'} style={{ padding: '2px 8px', borderRadius: 6, border: `1px solid ${C.border}`, background: 'transparent', color: resolved ? C.textSecondary : '#059669', fontSize: 12, cursor: busyId === a.id ? 'default' : 'pointer' }}>{resolved ? '↩ 重开' : '✓ 已处理'}</button>
                    <button onClick={() => handleDelete(a)} disabled={busyId === a.id} title="删除批注" style={{ padding: '2px 8px', borderRadius: 6, border: `1px solid ${C.border}`, background: 'transparent', color: '#DC2626', fontSize: 12, cursor: busyId === a.id ? 'default' : 'pointer' }}>🗑</button>
                  </div>
                </div>
                <div style={{ fontSize: 13, color: C.textPrimary, lineHeight: 1.6, whiteSpace: 'pre-wrap', wordBreak: 'break-word', textDecoration: resolved ? 'line-through' : 'none' }}>{a.content}</div>
              </div>
            )
          })
        )}
      </div>

      {/* 新增批注输入区 */}
      <div style={{ padding: '10px 14px', borderTop: `1px solid ${C.border}`, background: '#FAFAFA' }}>
        <textarea
          value={input}
          onChange={e => setInput(e.target.value)}
          onKeyDown={e => { if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) { e.preventDefault(); handleSubmit() } }}
          placeholder={`对第 ${pageNumber} 页留下批注…（Ctrl+Enter 提交）`}
          rows={2}
          style={{ width: '100%', boxSizing: 'border-box', padding: '8px 10px', borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 13, resize: 'vertical', fontFamily: 'inherit', outline: 'none' }}
        />
        <div style={{ display: 'flex', justifyContent: 'flex-end', marginTop: 8 }}>
          <button onClick={handleSubmit} disabled={submitting || !input.trim()} style={{ padding: '6px 16px', borderRadius: 8, border: 'none', background: input.trim() && !submitting ? C.primary : C.border, color: C.white, fontSize: 13, fontWeight: 600, cursor: input.trim() && !submitting ? 'pointer' : 'default' }}>{submitting ? '提交中…' : '发表批注'}</button>
        </div>
      </div>
    </div>
  )
}

/**
 * 课件方案编辑器 — IndexEditor.tsx v2.0 (Phase 3.5)
 *
 * 两层架构展示：
 *   - 普通用户：看到翻译后的方案（知识目标、能力目标、互动设计、回收方式等）
 *   - admin：额外可展开查看层1 AOCI技术索引原文
 *
 * 卡片列表展示+编辑+增删+排序
 */
import { useState } from 'react'
import {
  updateCWPageIndex, addCWPage, deleteCWPage, reorderCWPages,
  CW_INTERACTION_TYPES, CW_VISUAL_FORMATS, CW_COGNITIVE_LEVELS,
} from '@/api/coursewares'
import type { CoursewarePage } from '@/api/coursewares'

// ==================== 颜色常量 ====================
const C = {
  primary: '#F59E0B', primaryBg: 'rgba(245,158,11,0.08)', primaryBorder: 'rgba(245,158,11,0.3)',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
  border: '#E5E7EB', danger: '#EF4444', success: '#059669',
  white: '#fff',
}

// ==================== 复杂度颜色映射 ====================
const COMPLEXITY_COLORS: Record<number, { color: string; bg: string; label: string }> = {
  1: { color: '#059669', bg: '#D1FAE5', label: '简单' },
  2: { color: '#0891B2', bg: '#CFFAFE', label: '较简单' },
  3: { color: '#D97706', bg: '#FEF3C7', label: '中等' },
  4: { color: '#DC2626', bg: '#FEE2E2', label: '较复杂' },
  5: { color: '#7C3AED', bg: '#EDE9FE', label: '复杂' },
}

// ==================== Props ====================
interface IndexEditorProps {
  coursewareId: string
  pages: CoursewarePage[]
  onPagesChange: (pages: CoursewarePage[]) => void
  loading?: boolean
  isAdmin?: boolean // admin可见层1索引
  indexOverview?: string // 课件脉络概述
}

export default function IndexEditor({ coursewareId, pages, onPagesChange, loading, isAdmin, indexOverview }: IndexEditorProps) {
  const [editingPage, setEditingPage] = useState<number | null>(null)
  const [editForm, setEditForm] = useState<Record<string, string | number>>({})
  const [saving, setSaving] = useState(false)
  const [addingPage, setAddingPage] = useState(false)
  const [expandedIndex, setExpandedIndex] = useState<Set<number>>(new Set()) // admin展开索引的页码集合

  // ==================== admin展开/折叠层1索引 ====================
  const toggleIndexExpand = (pageNum: number) => {
    setExpandedIndex(prev => {
      const next = new Set(prev)
      if (next.has(pageNum)) { next.delete(pageNum) } else { next.add(pageNum) }
      return next
    })
  }

  // ==================== 开始编辑 ====================
  const startEdit = (page: CoursewarePage) => {
    setEditingPage(page.page_number)
    setEditForm({
      title: page.title,
      purpose: page.purpose,
      content_summary: page.content_summary,
      interaction_type: page.interaction_type,
      visual_format: page.visual_format,
      media_requirements: page.media_requirements,
      estimated_complexity: page.estimated_complexity,
    })
  }

  // ==================== 保存编辑 ====================
  const saveEdit = async () => {
    if (editingPage === null) return
    setSaving(true)
    try {
      await updateCWPageIndex(coursewareId, editingPage, {
        title: String(editForm.title || ''),
        purpose: String(editForm.purpose || ''),
        content_summary: String(editForm.content_summary || ''),
        interaction_type: String(editForm.interaction_type || ''),
        visual_format: String(editForm.visual_format || ''),
        media_requirements: String(editForm.media_requirements || ''),
        estimated_complexity: Number(editForm.estimated_complexity) || 1,
      })
      const updated = pages.map(p => p.page_number === editingPage ? {
        ...p,
        title: String(editForm.title || ''),
        purpose: String(editForm.purpose || ''),
        content_summary: String(editForm.content_summary || ''),
        interaction_type: String(editForm.interaction_type || ''),
        visual_format: String(editForm.visual_format || ''),
        media_requirements: String(editForm.media_requirements || ''),
        estimated_complexity: Number(editForm.estimated_complexity) || 1,
      } : p)
      onPagesChange(updated)
      setEditingPage(null)
    } catch { alert('保存失败') } finally { setSaving(false) }
  }

  // ==================== 删除页面 ====================
  const handleDelete = async (pageNum: number) => {
    if (!window.confirm(`确定删除第 ${pageNum} 页？`)) return
    try {
      await deleteCWPage(coursewareId, pageNum)
      const remaining = pages.filter(p => p.page_number !== pageNum)
      const renumbered = remaining.map((p, i) => ({ ...p, page_number: i + 1 }))
      onPagesChange(renumbered)
    } catch { alert('删除失败') }
  }

  // ==================== 添加页面 ====================
  const handleAdd = async () => {
    setAddingPage(true)
    try {
      const newPage = await addCWPage(coursewareId, {
        title: `第 ${pages.length + 1} 页`,
        purpose: '',
        content_summary: '',
        interaction_type: 'static',
        visual_format: 'text_heavy',
      })
      onPagesChange([...pages, newPage])
    } catch { alert('添加失败') } finally { setAddingPage(false) }
  }

  // ==================== 上移/下移 ====================
  const movePage = async (index: number, direction: 'up' | 'down') => {
    const target = direction === 'up' ? index - 1 : index + 1
    if (target < 0 || target >= pages.length) return
    const newPages = [...pages]
    const temp = newPages[index]
    newPages[index] = newPages[target]
    newPages[target] = temp
    const renumbered = newPages.map((p, i) => ({ ...p, page_number: i + 1 }))
    onPagesChange(renumbered)
    try {
      await reorderCWPages(coursewareId, renumbered.map(p => p.id))
    } catch { /* 静默 */ }
  }

  if (loading) {
    return <div style={{ textAlign: 'center', padding: '60px 0', color: C.textMuted }}>加载中...</div>
  }

  return (
    <div>
      {/* 课件脉络概述 */}
      {indexOverview && (
        <div style={{
          padding: '14px 18px', borderRadius: '10px', marginBottom: '16px',
          background: 'linear-gradient(135deg, rgba(245,158,11,0.06), rgba(239,68,68,0.04))',
          border: '1px solid rgba(245,158,11,0.2)',
        }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: '#D97706', marginBottom: '6px' }}>📋 课件脉络</div>
          <div style={{ fontSize: '13px', color: '#4B5563', lineHeight: '1.6' }}>{indexOverview}</div>
        </div>
      )}

      {/* 页面数量统计 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
        <div style={{ fontSize: '14px', color: C.textSecondary }}>
          共 <strong style={{ color: C.primary }}>{pages.length}</strong> 页
        </div>
        <button onClick={handleAdd} disabled={addingPage} style={{
          padding: '6px 16px', borderRadius: '8px', border: `1px dashed ${C.primary}`,
          background: C.primaryBg, color: C.primary, fontSize: '13px', fontWeight: 600,
          cursor: addingPage ? 'default' : 'pointer', opacity: addingPage ? 0.6 : 1,
        }}>{addingPage ? '添加中...' : '+ 添加页面'}</button>
      </div>

      {/* 卡片列表 */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
        {pages.map((page, idx) => {
          const isEditing = editingPage === page.page_number
          const cc = COMPLEXITY_COLORS[page.estimated_complexity] || COMPLEXITY_COLORS[1]
          const it = CW_INTERACTION_TYPES[page.interaction_type] || { label: page.interaction_type, emoji: '📄' }
          const vf = CW_VISUAL_FORMATS[page.visual_format] || { label: page.visual_format, emoji: '📝' }
          const cg = CW_COGNITIVE_LEVELS[page.idx_cognitive_level] || null
          const isIndexExpanded = expandedIndex.has(page.page_number)

          return (
            <div key={page.id || idx} style={{
              background: C.white, borderRadius: '12px', padding: '16px 20px',
              border: `1px solid ${isEditing ? C.primaryBorder : C.border}`,
              boxShadow: isEditing ? '0 2px 12px rgba(245,158,11,0.15)' : '0 1px 3px rgba(0,0,0,0.04)',
            }}>
              {/* 卡片头部 */}
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '10px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                  <span style={{
                    width: '28px', height: '28px', borderRadius: '50%',
                    background: 'linear-gradient(135deg, #F59E0B, #EF4444)',
                    color: '#fff', fontSize: '13px', fontWeight: 700,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                  }}>{page.page_number}</span>
                  {isEditing ? (
                    <input value={editForm.title || ''} onChange={e => setEditForm({ ...editForm, title: e.target.value })}
                      style={{ fontSize: '15px', fontWeight: 600, border: `1px solid ${C.border}`, borderRadius: '6px', padding: '4px 8px', flex: 1, minWidth: '200px' }} />
                  ) : (
                    <span style={{ fontSize: '15px', fontWeight: 600, color: C.textPrimary }}>{page.title || '(未命名)'}</span>
                  )}
                </div>
                <div style={{ display: 'flex', gap: '4px', alignItems: 'center' }}>
                  <button onClick={() => movePage(idx, 'up')} disabled={idx === 0} title="上移"
                    style={{ background: 'transparent', border: 'none', fontSize: '16px', cursor: idx === 0 ? 'default' : 'pointer', opacity: idx === 0 ? 0.3 : 1, padding: '2px 6px' }}>⬆</button>
                  <button onClick={() => movePage(idx, 'down')} disabled={idx === pages.length - 1} title="下移"
                    style={{ background: 'transparent', border: 'none', fontSize: '16px', cursor: idx === pages.length - 1 ? 'default' : 'pointer', opacity: idx === pages.length - 1 ? 0.3 : 1, padding: '2px 6px' }}>⬇</button>
                  {isEditing ? (
                    <>
                      <button onClick={saveEdit} disabled={saving} style={{
                        padding: '3px 10px', borderRadius: '6px', border: 'none',
                        background: C.primary, color: '#fff', fontSize: '12px', cursor: 'pointer',
                      }}>{saving ? '...' : '保存'}</button>
                      <button onClick={() => setEditingPage(null)} style={{
                        padding: '3px 10px', borderRadius: '6px', border: `1px solid ${C.border}`,
                        background: 'transparent', color: C.textSecondary, fontSize: '12px', cursor: 'pointer',
                      }}>取消</button>
                    </>
                  ) : (
                    <button onClick={() => startEdit(page)} style={{
                      padding: '3px 10px', borderRadius: '6px', border: `1px solid ${C.border}`,
                      background: 'transparent', color: C.textSecondary, fontSize: '12px', cursor: 'pointer',
                    }}>编辑</button>
                  )}
                  <button onClick={() => handleDelete(page.page_number)} style={{
                    padding: '3px 10px', borderRadius: '6px', border: `1px solid ${C.border}`,
                    background: 'transparent', color: C.danger, fontSize: '12px', cursor: 'pointer',
                  }}>删除</button>
                </div>
              </div>

              {/* 卡片内容 */}
              {isEditing ? (
                <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', fontSize: '13px' }}>
                  <label style={{ color: C.textSecondary }}>教学目的
                    <textarea value={String(editForm.purpose || '')} onChange={e => setEditForm({ ...editForm, purpose: e.target.value })} rows={2}
                      style={{ width: '100%', border: `1px solid ${C.border}`, borderRadius: '6px', padding: '6px 8px', resize: 'vertical', marginTop: '4px' }} />
                  </label>
                  <label style={{ color: C.textSecondary }}>内容概要
                    <textarea value={String(editForm.content_summary || '')} onChange={e => setEditForm({ ...editForm, content_summary: e.target.value })} rows={3}
                      style={{ width: '100%', border: `1px solid ${C.border}`, borderRadius: '6px', padding: '6px 8px', resize: 'vertical', marginTop: '4px' }} />
                  </label>
                  <div style={{ display: 'flex', gap: '12px' }}>
                    <label style={{ flex: 1, color: C.textSecondary }}>交互类型
                      <select value={String(editForm.interaction_type || 'static')} onChange={e => setEditForm({ ...editForm, interaction_type: e.target.value })}
                        style={{ width: '100%', border: `1px solid ${C.border}`, borderRadius: '6px', padding: '6px 8px', marginTop: '4px' }}>
                        {Object.entries(CW_INTERACTION_TYPES).map(([k, v]) => (
                          <option key={k} value={k}>{v.emoji} {v.label}</option>
                        ))}
                      </select>
                    </label>
                    <label style={{ flex: 1, color: C.textSecondary }}>视觉形式
                      <select value={String(editForm.visual_format || 'text_heavy')} onChange={e => setEditForm({ ...editForm, visual_format: e.target.value })}
                        style={{ width: '100%', border: `1px solid ${C.border}`, borderRadius: '6px', padding: '6px 8px', marginTop: '4px' }}>
                        {Object.entries(CW_VISUAL_FORMATS).map(([k, v]) => (
                          <option key={k} value={k}>{v.emoji} {v.label}</option>
                        ))}
                      </select>
                    </label>
                    <label style={{ flex: 1, color: C.textSecondary }}>复杂度 (1-5)
                      <input type="number" min={1} max={5} value={Number(editForm.estimated_complexity) || 1}
                        onChange={e => setEditForm({ ...editForm, estimated_complexity: parseInt(e.target.value) || 1 })}
                        style={{ width: '100%', border: `1px solid ${C.border}`, borderRadius: '6px', padding: '6px 8px', marginTop: '4px' }} />
                    </label>
                  </div>
                  <label style={{ color: C.textSecondary }}>多媒体需求
                    <input value={String(editForm.media_requirements || '')} onChange={e => setEditForm({ ...editForm, media_requirements: e.target.value })}
                      style={{ width: '100%', border: `1px solid ${C.border}`, borderRadius: '6px', padding: '6px 8px', marginTop: '4px' }} />
                  </label>
                </div>
              ) : (
                <div>
                  {/* 方案展示（层2用户友好内容） */}
                  {page.purpose && (
                    <div style={{ fontSize: '13px', color: C.textSecondary, marginBottom: '6px' }}>
                      <strong>目的：</strong>{page.purpose}
                    </div>
                  )}
                  {page.content_summary && (
                    <div style={{ fontSize: '13px', color: C.textSecondary, marginBottom: '8px' }}>
                      <strong>概要：</strong>{page.content_summary}
                    </div>
                  )}
                  <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
                    <span style={{ padding: '2px 8px', borderRadius: '10px', fontSize: '12px', background: '#DBEAFE', color: '#2563EB' }}>
                      {it.emoji} {it.label}
                    </span>
                    <span style={{ padding: '2px 8px', borderRadius: '10px', fontSize: '12px', background: '#EDE9FE', color: '#7C3AED' }}>
                      {vf.emoji} {vf.label}
                    </span>
                    <span style={{ padding: '2px 8px', borderRadius: '10px', fontSize: '12px', background: cc.bg, color: cc.color }}>
                      ⚡ {cc.label}
                    </span>
                    {cg && (
                      <span style={{ padding: '2px 8px', borderRadius: '10px', fontSize: '12px', background: cg.bg, color: cg.color }}>
                        🧠 {cg.label}
                      </span>
                    )}
                    {page.media_requirements && (
                      <span style={{ padding: '2px 8px', borderRadius: '10px', fontSize: '12px', background: '#FEF3C7', color: '#D97706' }}>
                        🖼️ {page.media_requirements.length > 20 ? page.media_requirements.slice(0, 20) + '...' : page.media_requirements}
                      </span>
                    )}
                  </div>

                  {/* admin可见：层1 AOCI技术索引 */}
                  {isAdmin && page.page_index && (
                    <div style={{ marginTop: '8px' }}>
                      <button onClick={() => toggleIndexExpand(page.page_number)} style={{
                        background: 'transparent', border: 'none', fontSize: '12px',
                        color: C.textMuted, cursor: 'pointer', padding: '2px 0',
                      }}>
                        {isIndexExpanded ? '▼' : '▶'} AOCI索引
                      </button>
                      {isIndexExpanded && (
                        <pre style={{
                          marginTop: '4px', padding: '8px 12px', borderRadius: '6px',
                          background: '#F9FAFB', border: `1px solid ${C.border}`,
                          fontSize: '11px', color: C.textSecondary, whiteSpace: 'pre-wrap',
                          fontFamily: 'monospace', lineHeight: '1.5', maxHeight: '200px', overflow: 'auto',
                        }}>{page.page_index}</pre>
                      )}
                    </div>
                  )}
                </div>
              )}
            </div>
          )
        })}
      </div>

      {pages.length === 0 && (
        <div style={{ textAlign: 'center', padding: '60px 0' }}>
          <div style={{ fontSize: '40px', marginBottom: '12px' }}>📋</div>
          <div style={{ fontSize: '15px', color: C.textSecondary }}>还没有课件方案</div>
          <div style={{ fontSize: '13px', color: C.textMuted, marginTop: '4px' }}>点击"AI生成方案"，AI将自动分析教案内容</div>
        </div>
      )}
    </div>
  )
}

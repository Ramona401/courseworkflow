/**
 * 课件方案编辑器 — IndexEditor.tsx v3.0
 *
 * 本次改造（内容丰富度 + 降认知负担）：
 *   1. 把不直观的「复杂度 (1-5) 数字框」改为「内容丰富度」三档大按钮：
 *      🌱 精简 / 📖 适中 / 🎯 充实，并配一句引导文案。
 *      底层仍写 estimated_complexity（精简=2 / 适中=3 / 充实=5），后端无感、无需改接口。
 *   2. 把专业字段「交互类型 / 视觉形式」折叠进「⚙ 高级（可不改）」区，默认收起，
 *      老师不展开就用 AI 给的默认值，减少困惑。
 *   3. 内容概要文本框上方加引导："写得越详细，AI 越会逐点展开。"
 *   4. 卡片展示态：突出「内容丰富度」人话标签，其余技术维度弱化为一行小灰字。
 *
 * 两层架构展示：
 *   - 普通用户：看到翻译后的方案（目的、概要、内容丰富度等人话信息）
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

// ==================== 内容丰富度三档（老师可见的人话） ====================
// value 即落库的 estimated_complexity：精简=2 / 适中=3 / 充实=5。
// 这三个值落在后端合法范围(1-5)内，且与后端 appendRichnessGuidance 的归档逻辑对齐
// （>=4=充实页，==3=适中页，<=2=精简页）。
interface RichnessOption {
  value: number
  emoji: string
  label: string
  desc: string
  color: string
  bg: string
}
const RICHNESS_OPTIONS: RichnessOption[] = [
  { value: 2, emoji: '🌱', label: '精简', desc: '要点为主，简洁留白', color: '#059669', bg: '#D1FAE5' },
  { value: 3, emoji: '📖', label: '适中', desc: '标准图文讲解', color: '#0891B2', bg: '#CFFAFE' },
  { value: 5, emoji: '🎯', label: '充实', desc: '详尽展开，多举例', color: '#DC2626', bg: '#FEE2E2' },
]

// 把任意 estimated_complexity(1-5) 归并到最近的三档之一，用于展示与高亮选中态。
// 规则与后端 appendRichnessGuidance 一致：>=4→充实, ==3→适中, <=2→精简。
function richnessOf(complexity: number): RichnessOption {
  if (complexity >= 4) return RICHNESS_OPTIONS[2] // 充实
  if (complexity === 3) return RICHNESS_OPTIONS[1] // 适中
  return RICHNESS_OPTIONS[0] // 精简（含 1、2 及异常值兜底）
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
  const [showAdvanced, setShowAdvanced] = useState(false) // 编辑态：是否展开「高级（交互/视觉）」区

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
    setShowAdvanced(false) // 每次进入编辑默认收起高级区
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
        estimated_complexity: Number(editForm.estimated_complexity) || 3,
      })
      const updated = pages.map(p => p.page_number === editingPage ? {
        ...p,
        title: String(editForm.title || ''),
        purpose: String(editForm.purpose || ''),
        content_summary: String(editForm.content_summary || ''),
        interaction_type: String(editForm.interaction_type || ''),
        visual_format: String(editForm.visual_format || ''),
        media_requirements: String(editForm.media_requirements || ''),
        estimated_complexity: Number(editForm.estimated_complexity) || 3,
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
          const rich = richnessOf(page.estimated_complexity) // 内容丰富度（人话标签）
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
                <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', fontSize: '13px' }}>
                  {/* 教学目的 */}
                  <label style={{ color: C.textSecondary }}>教学目的
                    <textarea value={String(editForm.purpose || '')} onChange={e => setEditForm({ ...editForm, purpose: e.target.value })} rows={2}
                      style={{ width: '100%', border: `1px solid ${C.border}`, borderRadius: '6px', padding: '6px 8px', resize: 'vertical', marginTop: '4px' }} />
                  </label>

                  {/* 内容概要（加引导文案） */}
                  <label style={{ color: C.textSecondary }}>内容概要
                    <span style={{ color: C.textMuted, fontSize: '12px', marginLeft: '6px' }}>（写得越详细，AI 越会逐点展开这一页）</span>
                    <textarea value={String(editForm.content_summary || '')} onChange={e => setEditForm({ ...editForm, content_summary: e.target.value })} rows={4}
                      style={{ width: '100%', border: `1px solid ${C.border}`, borderRadius: '6px', padding: '6px 8px', resize: 'vertical', marginTop: '4px' }} />
                  </label>

                  {/* 内容丰富度（三档大按钮，取代复杂度数字框） */}
                  <div>
                    <div style={{ color: C.textSecondary, marginBottom: '6px' }}>
                      内容丰富度
                      <span style={{ color: C.textMuted, fontSize: '12px', marginLeft: '6px' }}>（想让这一页内容更多、举例更丰富，就选「充实」）</span>
                    </div>
                    <div style={{ display: 'flex', gap: '8px' }}>
                      {RICHNESS_OPTIONS.map(opt => {
                        const selected = richnessOf(Number(editForm.estimated_complexity) || 3).value === opt.value
                        return (
                          <button key={opt.value} type="button"
                            onClick={() => setEditForm({ ...editForm, estimated_complexity: opt.value })}
                            style={{
                              flex: 1, padding: '10px 8px', borderRadius: '10px', cursor: 'pointer',
                              border: `2px solid ${selected ? opt.color : C.border}`,
                              background: selected ? opt.bg : C.white,
                              display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '2px',
                              transition: 'all 0.15s',
                            }}>
                            <span style={{ fontSize: '20px' }}>{opt.emoji}</span>
                            <span style={{ fontSize: '13px', fontWeight: 700, color: selected ? opt.color : C.textPrimary }}>{opt.label}</span>
                            <span style={{ fontSize: '11px', color: selected ? opt.color : C.textMuted, textAlign: 'center', lineHeight: '1.3' }}>{opt.desc}</span>
                          </button>
                        )
                      })}
                    </div>
                  </div>

                  {/* 高级（可不改）：交互类型 / 视觉形式 / 多媒体需求，默认折叠 */}
                  <div style={{ marginTop: '2px' }}>
                    <button type="button" onClick={() => setShowAdvanced(v => !v)}
                      style={{ background: 'transparent', border: 'none', color: C.textMuted, fontSize: '12px', cursor: 'pointer', padding: '4px 0' }}>
                      {showAdvanced ? '▼' : '▶'} ⚙ 高级选项（可不改，不确定就保持默认）
                    </button>
                    {showAdvanced && (
                      <div style={{ display: 'flex', flexDirection: 'column', gap: '10px', marginTop: '6px', padding: '12px', borderRadius: '8px', background: '#F9FAFB', border: `1px solid ${C.border}` }}>
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
                        </div>
                        <label style={{ color: C.textSecondary }}>多媒体需求
                          <input value={String(editForm.media_requirements || '')} onChange={e => setEditForm({ ...editForm, media_requirements: e.target.value })}
                            style={{ width: '100%', border: `1px solid ${C.border}`, borderRadius: '6px', padding: '6px 8px', marginTop: '4px' }} />
                        </label>
                      </div>
                    )}
                  </div>
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

                  {/* 突出展示：内容丰富度（人话标签，老师最该关注的一项） */}
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
                    <span style={{
                      padding: '3px 12px', borderRadius: '10px', fontSize: '12px', fontWeight: 700,
                      background: rich.bg, color: rich.color,
                    }}>
                      {rich.emoji} 内容{rich.label}
                    </span>
                  </div>

                  {/* 弱化展示：技术维度收成一行小灰字（交互/视觉/认知/多媒体） */}
                  <div style={{ fontSize: '12px', color: C.textMuted, display: 'flex', gap: '12px', flexWrap: 'wrap' }}>
                    <span>{it.emoji} {it.label}</span>
                    <span>{vf.emoji} {vf.label}</span>
                    {cg && <span>🧠 {cg.label}</span>}
                    {page.media_requirements && (
                      <span>🖼️ {page.media_requirements.length > 16 ? page.media_requirements.slice(0, 16) + '...' : page.media_requirements}</span>
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

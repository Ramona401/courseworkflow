/**
 * 组件管理页面 — ComponentsPage
 *
 * Phase5完整实现：
 *   - 顶部Tab：组件概览 / 组件列表 / 萃取待审队列
 *   - 组件概览Tab：13类组件统计卡片（可点击查看该类下组件）+ 创建按钮
 *   - 组件列表Tab：全部组件表格（支持筛选+编辑+删除）
 *   - 萃取待审Tab：待审列表 + 确认/拒绝操作
 *
 * 迭代4B-1改造：
 *   - 新增"组件列表"Tab（全部组件的可搜索列表视图）
 *   - 概览Tab增加"创建组件"按钮
 *   - 概览卡片点击进入该类型的组件列表
 *   - 集成ComponentEditModal创建/编辑弹窗
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getComponents, deleteComponent, getExtractions, confirmExtraction,
  type ComponentListItem, type ExtractionListItem, type LibraryType,
} from '../../../api/lesson-plans'
import ComponentEditModal from './ComponentEditModal'

// ==================== 常量定义 ====================

const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981', danger: '#EF4444', accent: '#F59E0B',
  text: '#1F2937', textSec: '#6B7280', textMuted: '#9CA3AF',
  card: '#FFFFFF', border: '#F3F4F6', bg: '#FAFBFC',
}

/** 13类组件元数据 */
const COMPONENT_TYPES = [
  { key: 'curriculum_standard',  icon: '📖', name: '课标与能力框架库', mode: 'silent' },
  { key: 'knowledge_graph',      icon: '🧠', name: '知识图谱库',       mode: 'silent' },
  { key: 'student_profile',      icon: '👤', name: '学情特征库',       mode: 'silent' },
  { key: 'pedagogy',             icon: '🎓', name: '教学法库',         mode: 'recommend' },
  { key: 'assessment_strategy',  icon: '📊', name: '评估策略库',       mode: 'recommend' },
  { key: 'activity_design',      icon: '🎯', name: '活动设计方案库',   mode: 'recommend' },
  { key: 'questioning_strategy', icon: '❓', name: '提问引导策略库',   mode: 'on_demand' },
  { key: 'cross_subject',        icon: '🔗', name: '跨学科连接库',     mode: 'on_demand' },
  { key: 'teaching_tool',        icon: '🛠️', name: '教学工具库',       mode: 'on_demand' },
  { key: 'scenario_material',    icon: '🎬', name: '素材情境库',       mode: 'on_demand' },
  { key: 'quality_rubric',       icon: '✅', name: '质量评估标准库',   mode: 'silent' },
  { key: 'design_defect',        icon: '⚠️', name: '常见设计缺陷库',   mode: 'silent' },
  { key: 'review_rubric',        icon: '📋', name: '教案评审规则库',   mode: 'ai_only' },
] as const

/** 注入模式中文和颜色 */
const MODE_LABEL: Record<string, string> = { silent: '静默注入', recommend: '推荐确认', on_demand: '按需调用', ai_only: 'AI内部' }
const MODE_COLOR: Record<string, { bg: string; fg: string }> = {
  silent:    { bg: 'rgba(79,123,232,0.08)',  fg: '#4F7BE8' },
  recommend: { bg: 'rgba(245,158,11,0.08)',  fg: '#F59E0B' },
  on_demand: { bg: 'rgba(16,185,129,0.08)',  fg: '#10B981' },
  ai_only:   { bg: 'rgba(156,163,175,0.08)', fg: '#9CA3AF' },
}

/** 来源中文映射 */
const SOURCE_LABEL: Record<string, string> = { conversation: '💬 对话萃取', lesson_plan: '📄 评审萃取', manual: '✍️ 手动入库' }

// ==================== 子组件：骨架屏 ====================

function SkeletonCard() {
  return (
    <div style={{ background: '#fff', borderRadius: '12px', padding: '20px', border: '1px solid #F3F4F6' }}>
      <div style={{ display: 'flex', gap: '12px', alignItems: 'center', marginBottom: '10px' }}>
        <div style={{ width: 36, height: 36, borderRadius: '8px', background: '#F3F4F6', animation: 'shimmer 1.5s infinite' }} />
        <div style={{ flex: 1 }}>
          <div style={{ height: 14, borderRadius: 4, background: '#F3F4F6', marginBottom: 6, animation: 'shimmer 1.5s infinite' }} />
          <div style={{ height: 10, borderRadius: 4, background: '#F3F4F6', width: '60%', animation: 'shimmer 1.5s infinite' }} />
        </div>
      </div>
      <div style={{ height: 10, borderRadius: 4, background: '#F3F4F6', width: '40%', animation: 'shimmer 1.5s infinite' }} />
    </div>
  )
}

// ==================== 子组件：Toast ====================

function Toast({ msg, type }: { msg: string; type: 'success' | 'error' }) {
  return (
    <div style={{
      position: 'fixed', bottom: 32, left: '50%', transform: 'translateX(-50%)',
      background: type === 'success' ? '#1F2937' : '#DC2626', color: '#FFF',
      padding: '10px 24px', borderRadius: '8px', fontSize: '14px', fontWeight: 500,
      zIndex: 9999, boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
    }}>{msg}</div>
  )
}

// ==================== 主页面 ====================

export default function ComponentsPage() {
  // Tab状态：overview / list / extractions
  const [activeTab, setActiveTab] = useState<'overview' | 'list' | 'extractions'>('overview')

  // 概览统计
  const [countMap, setCountMap] = useState<Record<string, number>>({})
  const [overviewLoading, setOverviewLoading] = useState(true)

  // 组件列表（迭代4B-1新增Tab）
  const [components, setComponents] = useState<ComponentListItem[]>([])
  const [listLoading, setListLoading] = useState(false)
  const [listFilter, setListFilter] = useState<{ libraryType: string; subject: string }>({ libraryType: '', subject: '' })
  const [listTotal, setListTotal] = useState(0)

  // 萃取待审
  const [extractions, setExtractions] = useState<ExtractionListItem[]>([])
  const [extractionLoading, setExtractionLoading] = useState(false)
  const [confirmingId, setConfirmingId] = useState<string | null>(null)

  // 弹窗状态（迭代4B-1新增）
  const [editModalOpen, setEditModalOpen] = useState(false)
  const [editComponentId, setEditComponentId] = useState<string | null>(null)
  const [editPresetType, setEditPresetType] = useState<LibraryType | undefined>(undefined)

  // 删除确认
  const [deleteConfirmId, setDeleteConfirmId] = useState<string | null>(null)
  const [deleting, setDeleting] = useState(false)

  // Toast
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)
  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type }); setTimeout(() => setToast(null), 3000)
  }

  // ---- 加载概览统计 ----
  const loadOverview = useCallback(async () => {
    setOverviewLoading(true)
    try {
      const resp = await getComponents({ limit: 1000 })
      const map: Record<string, number> = {}
      const list = (resp as unknown as { components: ComponentListItem[] }).components || []
      list.forEach(c => { map[c.library_type] = (map[c.library_type] || 0) + 1 })
      setCountMap(map)
    } catch (e) { console.error('加载组件概览失败', e) }
    finally { setOverviewLoading(false) }
  }, [])

  // ---- 加载组件列表 ----
  const loadList = useCallback(async () => {
    setListLoading(true)
    try {
      const resp = await getComponents({
        library_type: (listFilter.libraryType || undefined) as LibraryType | undefined,
        subject: listFilter.subject || undefined,
        limit: 200,
      })
      const data = resp as unknown as { components: ComponentListItem[]; total: number }
      setComponents(data.components || [])
      setListTotal(data.total || 0)
    } catch (e) { console.error('加载组件列表失败', e) }
    finally { setListLoading(false) }
  }, [listFilter])

  // ---- 加载萃取列表 ----
  const loadExtractions = useCallback(async () => {
    setExtractionLoading(true)
    try {
      const resp = await getExtractions({ limit: 100 })
      setExtractions(resp.extractions || [])
    } catch (e) { console.error('加载萃取列表失败', e) }
    finally { setExtractionLoading(false) }
  }, [])

  // 初始加载
  useEffect(() => { loadOverview() }, [loadOverview])
  useEffect(() => { if (activeTab === 'list') loadList() }, [activeTab, loadList])
  useEffect(() => { if (activeTab === 'extractions') loadExtractions() }, [activeTab, loadExtractions])

  // ---- 打开创建弹窗 ----
  const openCreate = (presetType?: LibraryType) => {
    setEditComponentId(null)
    setEditPresetType(presetType)
    setEditModalOpen(true)
  }

  // ---- 打开编辑弹窗 ----
  const openEdit = (compId: string) => {
    setEditComponentId(compId)
    setEditPresetType(undefined)
    setEditModalOpen(true)
  }

  // ---- 弹窗保存成功后刷新 ----
  const handleSaved = () => {
    loadOverview()
    if (activeTab === 'list') loadList()
  }

  // ---- 删除组件 ----
  const handleDelete = async (id: string) => {
    setDeleting(true)
    try {
      await deleteComponent(id)
      showToast('已删除 ✓')
      setDeleteConfirmId(null)
      loadList()
      loadOverview()
    } catch { showToast('删除失败', 'error') }
    finally { setDeleting(false) }
  }

  // ---- 萃取确认/拒绝 ----
  const handleConfirm = async (id: string, decision: 'confirmed' | 'rejected') => {
    setConfirmingId(id)
    try {
      await confirmExtraction(id, decision)
      showToast(decision === 'confirmed' ? '✅ 已确认入库' : '已拒绝')
      setExtractions(prev => prev.filter(e => e.id !== id))
    } catch { showToast('操作失败', 'error') }
    finally { setConfirmingId(null) }
  }

  // ---- 概览卡片点击：切换到列表Tab并设置类型筛选 ----
  const viewTypeComponents = (typeKey: string) => {
    setListFilter({ libraryType: typeKey, subject: '' })
    setActiveTab('list')
  }

  const pendingCount = extractions.filter(e => e.status === 'pending').length

  // ==================== 渲染 ====================
  return (
    <div>
      <style>{`@keyframes shimmer { 0%{opacity:1} 50%{opacity:0.4} 100%{opacity:1} }`}</style>

      {/* 描述+创建按钮 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', margin: '0 0 20px 0' }}>
        <p style={{ fontSize: '14px', color: C.textSec, margin: 0 }}>
          管理13类教学设计组件，支持匹配引擎智能推荐，好的设计片段自动沉淀进库
        </p>
        <button onClick={() => openCreate()} style={{
          padding: '8px 18px', borderRadius: '8px', border: 'none', fontSize: '13px', fontWeight: 600,
          background: C.primary, color: '#fff', cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '6px',
          boxShadow: '0 2px 8px rgba(79,123,232,0.25)',
        }}>＋ 创建组件</button>
      </div>

      {/* Tab切换 */}
      <div style={{ display: 'flex', gap: '4px', marginBottom: '24px', borderBottom: '1px solid #E5E7EB' }}>
        {([
          { key: 'overview' as const,    label: '🧩 组件概览' },
          { key: 'list' as const,        label: `📋 组件列表${listTotal > 0 ? ` (${listTotal})` : ''}` },
          { key: 'extractions' as const, label: `💡 萃取待审${pendingCount > 0 ? ` (${pendingCount})` : ''}` },
        ]).map(tab => (
          <button key={tab.key} onClick={() => setActiveTab(tab.key)} style={{
            padding: '10px 20px', border: 'none', background: 'transparent', fontSize: '14px',
            fontWeight: activeTab === tab.key ? 600 : 400,
            color: activeTab === tab.key ? C.primary : C.textSec,
            borderBottom: activeTab === tab.key ? `2px solid ${C.primary}` : '2px solid transparent',
            cursor: 'pointer', marginBottom: '-1px', transition: 'all 150ms ease',
          }}>{tab.label}</button>
        ))}
      </div>

      {/* ==================== Tab：组件概览 ==================== */}
      {activeTab === 'overview' && (
        <div>
          <div style={{
            display: 'flex', gap: '12px', marginBottom: '20px', padding: '16px 20px',
            background: '#F0F4FF', borderRadius: '12px', alignItems: 'center',
          }}>
            <span style={{ fontSize: '20px' }}>📊</span>
            <div>
              <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>
                组件库总计：{Object.values(countMap).reduce((a, b) => a + b, 0)} 个组件
              </div>
              <div style={{ fontSize: '12px', color: C.textSec, marginTop: '2px' }}>
                涵盖13个类别 · 点击卡片查看详细组件 · 点击右上角按钮创建新组件
              </div>
            </div>
          </div>

          <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(260px, 1fr))', gap: '16px' }}>
            {overviewLoading
              ? Array.from({ length: 13 }).map((_, i) => <SkeletonCard key={i} />)
              : COMPONENT_TYPES.map(ct => {
                  const mc = MODE_COLOR[ct.mode]
                  const count = countMap[ct.key] || 0
                  return (
                    <div key={ct.key} onClick={() => viewTypeComponents(ct.key)} style={{
                      background: '#fff', borderRadius: '12px', padding: '20px',
                      border: '1px solid #F3F4F6', cursor: 'pointer', transition: 'all 200ms ease',
                    }}
                    onMouseEnter={e => { (e.currentTarget as HTMLElement).style.borderColor = C.primary; (e.currentTarget as HTMLElement).style.transform = 'translateY(-2px)' }}
                    onMouseLeave={e => { (e.currentTarget as HTMLElement).style.borderColor = '#F3F4F6'; (e.currentTarget as HTMLElement).style.transform = 'none' }}>
                      <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px', marginBottom: '12px' }}>
                        <span style={{ fontSize: '24px', lineHeight: 1 }}>{ct.icon}</span>
                        <div style={{ flex: 1 }}>
                          <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>{ct.name}</div>
                          <span style={{ fontSize: '10px', fontWeight: 600, color: mc.fg, background: mc.bg, padding: '2px 8px', borderRadius: '6px' }}>
                            {MODE_LABEL[ct.mode]}
                          </span>
                        </div>
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <span style={{ fontSize: '12px', color: C.textMuted }}>
                          组件数量：<span style={{ fontWeight: 700, color: count > 0 ? C.primary : C.textMuted }}>{count}</span>
                        </span>
                        <button onClick={e => { e.stopPropagation(); openCreate(ct.key as LibraryType) }} style={{
                          padding: '3px 10px', borderRadius: '6px', border: `1px solid ${C.border}`, background: '#fff',
                          fontSize: '11px', color: C.primary, cursor: 'pointer', fontWeight: 600,
                        }}>＋ 添加</button>
                      </div>
                    </div>
                  )
                })
            }
          </div>
        </div>
      )}

      {/* ==================== Tab：组件列表 ==================== */}
      {activeTab === 'list' && (
        <div>
          {/* 筛选栏 */}
          <div style={{ display: 'flex', gap: '12px', marginBottom: '16px', alignItems: 'center' }}>
            <select value={listFilter.libraryType} onChange={e => setListFilter(f => ({ ...f, libraryType: e.target.value }))} style={{
              padding: '7px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', color: C.text, background: '#fff', cursor: 'pointer',
            }}>
              <option value="">全部类型</option>
              {COMPONENT_TYPES.map(ct => <option key={ct.key} value={ct.key}>{ct.icon} {ct.name}</option>)}
            </select>
            <select value={listFilter.subject} onChange={e => setListFilter(f => ({ ...f, subject: e.target.value }))} style={{
              padding: '7px 12px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', color: C.text, background: '#fff', cursor: 'pointer',
            }}>
              <option value="">全部学科</option>
              {['AI','人工智能','语文','数学','英语','物理','化学','生物','历史','地理','政治','信息技术'].map(s =>
                <option key={s} value={s}>{s}</option>
              )}
            </select>
            <div style={{ flex: 1 }} />
            <span style={{ fontSize: '12px', color: C.textMuted }}>{listTotal} 个组件</span>
          </div>

          {/* 列表 */}
          {listLoading ? (
            <div style={{ textAlign: 'center', padding: '40px', color: C.textMuted }}>加载中...</div>
          ) : components.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '60px', background: '#fff', borderRadius: '12px', border: `1px solid ${C.border}` }}>
              <div style={{ fontSize: '40px', marginBottom: '12px' }}>📭</div>
              <div style={{ fontSize: '16px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>暂无组件</div>
              <div style={{ fontSize: '14px', color: C.textSec, marginBottom: '16px' }}>
                {listFilter.libraryType ? '该类型下暂无组件，点击下方按钮创建第一个' : '组件库还是空的，开始建设吧'}
              </div>
              <button onClick={() => openCreate(listFilter.libraryType as LibraryType || undefined)} style={{
                padding: '9px 24px', borderRadius: '8px', border: 'none', background: C.primary,
                color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer',
              }}>＋ 创建组件</button>
            </div>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
              {components.map(comp => {
                const typeInfo = COMPONENT_TYPES.find(t => t.key === comp.library_type)
                return (
                  <div key={comp.id} style={{
                    display: 'flex', alignItems: 'center', gap: '14px', padding: '14px 18px',
                    background: '#fff', borderRadius: '10px', border: `1px solid ${C.border}`,
                    transition: 'border-color 150ms ease', cursor: 'pointer',
                  }}
                  onClick={() => openEdit(comp.id)}
                  onMouseEnter={e => { (e.currentTarget as HTMLElement).style.borderColor = C.primary }}
                  onMouseLeave={e => { (e.currentTarget as HTMLElement).style.borderColor = C.border }}>
                    {/* 类型图标 */}
                    <span style={{ fontSize: '22px', flexShrink: 0 }}>{typeInfo?.icon || '📋'}</span>

                    {/* 主信息 */}
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '3px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {comp.display_label}
                      </div>
                      <div style={{ display: 'flex', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
                        <span style={{ fontSize: '11px', color: C.textMuted }}>{comp.library_name || typeInfo?.name}</span>
                        <span style={{ fontSize: '11px', color: C.textMuted }}>·</span>
                        <span style={{ fontSize: '11px', color: C.textMuted }}>{comp.subject === 'general' ? '通用' : comp.subject}</span>
                        {comp.grade_range && <>
                          <span style={{ fontSize: '11px', color: C.textMuted }}>·</span>
                          <span style={{ fontSize: '11px', color: C.textMuted }}>{comp.grade_range}</span>
                        </>}
                      </div>
                    </div>

                    {/* 质量分 */}
                    <div style={{ textAlign: 'center', flexShrink: 0 }}>
                      <div style={{ fontSize: '16px', fontWeight: 700, color: comp.quality_score > 5 ? C.success : C.textMuted }}>
                        {comp.quality_score.toFixed(1)}
                      </div>
                      <div style={{ fontSize: '10px', color: C.textMuted }}>质量分</div>
                    </div>

                    {/* 使用统计 */}
                    <div style={{ textAlign: 'center', flexShrink: 0, minWidth: '50px' }}>
                      <div style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>{comp.select_count}</div>
                      <div style={{ fontSize: '10px', color: C.textMuted }}>选中</div>
                    </div>

                    {/* 来源标签 */}
                    <span style={{
                      fontSize: '10px', fontWeight: 600, padding: '2px 8px', borderRadius: '6px', flexShrink: 0,
                      background: comp.source === 'ai_extracted' ? 'rgba(245,158,11,0.08)' : 'rgba(79,123,232,0.08)',
                      color: comp.source === 'ai_extracted' ? C.accent : C.primary,
                    }}>
                      {comp.source === 'ai_extracted' ? 'AI萃取' : comp.source === 'manual' ? '手动' : comp.source}
                    </span>

                    {/* 删除按钮 */}
                    <button onClick={e => { e.stopPropagation(); setDeleteConfirmId(comp.id) }} style={{
                      background: 'none', border: 'none', cursor: 'pointer', fontSize: '14px', color: C.textMuted,
                      padding: '4px 6px', borderRadius: '6px', flexShrink: 0,
                    }}
                    onMouseEnter={e => { (e.currentTarget as HTMLElement).style.color = C.danger }}
                    onMouseLeave={e => { (e.currentTarget as HTMLElement).style.color = C.textMuted }}>
                      🗑️
                    </button>
                  </div>
                )
              })}
            </div>
          )}
        </div>
      )}

      {/* ==================== Tab：萃取待审 ==================== */}
      {activeTab === 'extractions' && (
        <div>
          <div style={{
            background: '#FFFBEB', border: '1px solid #FDE68A', borderRadius: '12px',
            padding: '16px 20px', marginBottom: '20px', display: 'flex', gap: '12px', alignItems: 'flex-start',
          }}>
            <span style={{ fontSize: '20px', flexShrink: 0 }}>💡</span>
            <div>
              <div style={{ fontSize: '14px', fontWeight: 600, color: '#92400E', marginBottom: '4px' }}>关于萃取待审</div>
              <div style={{ fontSize: '13px', color: '#78350F', lineHeight: 1.6 }}>
                系统会从优秀对话和评审通过的高分教案中自动识别可复用的教学设计片段。
                教研组长和骨干教师确认后，这些片段将进入组件库，被匹配引擎推荐给其他老师。
              </div>
            </div>
          </div>

          {extractionLoading && <div style={{ textAlign: 'center', padding: '40px', color: C.textMuted }}>加载中...</div>}

          {!extractionLoading && extractions.length === 0 && (
            <div style={{ textAlign: 'center', padding: '60px 20px', background: '#fff', borderRadius: '12px', border: `1px solid ${C.border}` }}>
              <div style={{ fontSize: '40px', marginBottom: '12px' }}>🌱</div>
              <div style={{ fontSize: '16px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>暂无待审萃取记录</div>
              <div style={{ fontSize: '14px', color: C.textSec }}>当老师在备课中产生好的设计片段，或有高分教案评审通过时，系统会自动识别并在这里等待您确认</div>
            </div>
          )}

          {!extractionLoading && extractions.length > 0 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px' }}>
              {extractions.map(item => (
                <ExtractionCard key={item.id} item={item} isConfirming={confirmingId === item.id}
                  onConfirm={decision => handleConfirm(item.id, decision)} />
              ))}
            </div>
          )}
        </div>
      )}

      {/* ==================== 组件编辑弹窗 ==================== */}
      {editModalOpen && (
        <ComponentEditModal
          componentId={editComponentId}
          presetLibraryType={editPresetType}
          onClose={() => setEditModalOpen(false)}
          onSaved={handleSaved}
        />
      )}

      {/* ==================== 删除确认弹窗 ==================== */}
      {deleteConfirmId && (
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          background: 'rgba(0,0,0,0.4)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 10000,
        }} onClick={() => setDeleteConfirmId(null)}>
          <div style={{
            background: '#fff', borderRadius: '12px', padding: '24px', width: '360px',
            boxShadow: '0 20px 60px rgba(0,0,0,0.2)',
          }} onClick={e => e.stopPropagation()}>
            <div style={{ fontSize: '16px', fontWeight: 700, color: C.text, marginBottom: '12px' }}>确认删除</div>
            <div style={{ fontSize: '14px', color: C.textSec, marginBottom: '20px', lineHeight: 1.6 }}>
              删除后该组件将被归档，不再出现在匹配引擎中。此操作可以通过数据库恢复。
            </div>
            <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
              <button onClick={() => setDeleteConfirmId(null)} style={{
                padding: '8px 18px', borderRadius: '8px', border: `1px solid ${C.border}`,
                background: '#fff', color: C.textSec, fontSize: '14px', cursor: 'pointer',
              }}>取消</button>
              <button onClick={() => handleDelete(deleteConfirmId)} disabled={deleting} style={{
                padding: '8px 18px', borderRadius: '8px', border: 'none',
                background: C.danger, color: '#fff', fontSize: '14px', fontWeight: 600,
                cursor: deleting ? 'not-allowed' : 'pointer',
              }}>{deleting ? '删除中...' : '确认删除'}</button>
            </div>
          </div>
        </div>
      )}

      {toast && <Toast msg={toast.msg} type={toast.type} />}
    </div>
  )
}

// ==================== 子组件：萃取卡片 ====================

function ExtractionCard({ item, isConfirming, onConfirm }: {
  item: ExtractionListItem; isConfirming: boolean
  onConfirm: (decision: 'confirmed' | 'rejected') => void
}) {
  const [expanded, setExpanded] = useState(false)
  const statusBadge = item.status === 'pending'
    ? { label: '待审核', bg: '#FEF3C7', fg: '#92400E' }
    : item.status === 'confirmed'
    ? { label: '已确认', bg: '#D1FAE5', fg: '#065F46' }
    : { label: '已拒绝', bg: '#FEE2E2', fg: '#991B1B' }

  return (
    <div style={{ background: '#fff', borderRadius: '12px', border: '1px solid #E5E7EB', overflow: 'hidden', opacity: isConfirming ? 0.7 : 1, transition: 'opacity 200ms' }}>
      <div style={{ padding: '16px 20px' }}>
        <div style={{ display: 'flex', alignItems: 'flex-start', gap: '12px' }}>
          <div style={{ flexShrink: 0, fontSize: '12px', fontWeight: 600, color: '#4F7BE8', background: 'rgba(79,123,232,0.08)', padding: '3px 10px', borderRadius: '6px', marginTop: '2px' }}>
            {SOURCE_LABEL[item.source_type] || item.source_type}
          </div>
          <div style={{ flex: 1 }}>
            <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
              <span style={{ fontSize: '12px', color: '#F59E0B', background: 'rgba(245,158,11,0.08)', padding: '2px 8px', borderRadius: '6px', fontWeight: 600 }}>
                {item.library_name || item.extraction_type}
              </span>
              <span style={{ fontSize: '11px', fontWeight: 600, color: statusBadge.fg, background: statusBadge.bg, padding: '2px 8px', borderRadius: '6px' }}>
                {statusBadge.label}
              </span>
            </div>
            <div style={{
              fontSize: '14px', color: '#1F2937', lineHeight: 1.6,
              display: '-webkit-box', WebkitLineClamp: expanded ? undefined : 2,
              WebkitBoxOrient: 'vertical', overflow: 'hidden',
            }}>{item.source_content}</div>
          </div>
          <button onClick={() => setExpanded(!expanded)} style={{
            flexShrink: 0, fontSize: '12px', color: '#6B7280', background: 'transparent',
            border: 'none', cursor: 'pointer', padding: '4px 8px', borderRadius: '6px',
          }}>{expanded ? '收起 ▲' : '展开 ▼'}</button>
        </div>
        {expanded && (
          <div style={{ marginTop: '12px', paddingTop: '12px', borderTop: '1px solid #F3F4F6' }}>
            <div style={{ fontSize: '12px', color: '#6B7280', marginBottom: '4px', fontWeight: 600 }}>完整内容片段：</div>
            <div style={{ fontSize: '13px', color: '#374151', lineHeight: 1.7, background: '#F9FAFB', borderRadius: '8px', padding: '12px' }}>
              {item.source_content}
            </div>
          </div>
        )}
      </div>
      <div style={{ padding: '12px 20px', background: '#FAFAFA', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: '12px' }}>
        <div style={{ fontSize: '12px', color: '#9CA3AF', display: 'flex', gap: '12px', flexWrap: 'wrap' }}>
          {item.plan_title && <span>📄 {item.plan_title}</span>}
          {item.created_by_name && <span>👤 {item.created_by_name}</span>}
          <span>🕐 {new Date(item.created_at).toLocaleDateString('zh-CN')}</span>
        </div>
        {item.status === 'pending' && (
          <div style={{ display: 'flex', gap: '8px', flexShrink: 0 }}>
            <button disabled={isConfirming} onClick={() => onConfirm('rejected')} style={{
              padding: '6px 16px', borderRadius: '8px', border: '1px solid #E5E7EB', background: '#fff',
              color: '#6B7280', fontSize: '13px', fontWeight: 500, cursor: isConfirming ? 'not-allowed' : 'pointer',
            }}>{isConfirming ? '处理中...' : '不入库'}</button>
            <button disabled={isConfirming} onClick={() => onConfirm('confirmed')} style={{
              padding: '6px 16px', borderRadius: '8px', border: 'none', background: '#4F7BE8',
              color: '#fff', fontSize: '13px', fontWeight: 600, cursor: isConfirming ? 'not-allowed' : 'pointer',
            }}>{isConfirming ? '处理中...' : '✓ 入库'}</button>
          </div>
        )}
      </div>
    </div>
  )
}

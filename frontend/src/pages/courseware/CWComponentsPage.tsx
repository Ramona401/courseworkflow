/**
 * 课件组件库页面 — CWComponentsPage v2.0（Phase 2）
 *
 * 组件浏览 + 类型筛选 + 详情弹窗（含iframe代码预览）+ admin删除 + 种子填充
 */
import { useState, useEffect, useCallback } from 'react'
import { getCWComponents, getCWComponent, deleteCWComponent, seedCoursewareData, CW_COMP_TYPE_CONFIG } from '@/api/coursewares'
import type { CWComponentListItem, CWComponentFull, SeedResult } from '@/api/coursewares'
import { useAuth } from '@/store/auth'

const C = {
  primary: '#F59E0B', textPrimary: '#1F2937', textSecondary: '#6B7280',
  textMuted: '#9CA3AF', border: '#E5E7EB', bgCard: '#FFFFFF', danger: '#EF4444',
}

export default function CWComponentsPage() {
  const { user } = useAuth()
  const isAdmin = user?.role === 'admin'
  const [items, setItems] = useState<CWComponentListItem[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [typeFilter, setTypeFilter] = useState('')
  const [detailComp, setDetailComp] = useState<CWComponentFull | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  const [seeding, setSeeding] = useState(false)

  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const resp = await getCWComponents({ component_type: typeFilter || undefined, limit: 100 })
      setItems(resp.components || [])
      setTotal(resp.total)
    } catch { /* 静默 */ } finally { setLoading(false) }
  }, [typeFilter])

  useEffect(() => { loadData() }, [loadData])

  const openDetail = async (id: string) => {
    setDetailLoading(true); setDetailComp(null)
    try { setDetailComp(await getCWComponent(id)) } catch { alert('加载失败') } finally { setDetailLoading(false) }
  }

  const handleDelete = async (id: string, name: string) => {
    if (!window.confirm(`确定删除组件「${name}」？`)) return
    try { await deleteCWComponent(id); loadData() } catch { alert('删除失败') }
  }

  const handleSeed = async (force: boolean) => {
    if (!window.confirm(force ? '将清空后重建，确定？' : '填充种子数据，确定？')) return
    setSeeding(true)
    try {
      const r: SeedResult = await seedCoursewareData(force)
      alert(`完成！组件:${r.components_created} 模板:${r.templates_created}${r.errors?.length ? '\n错误:' + r.errors.join('\n') : ''}`)
      loadData()
    } catch { alert('填充失败') } finally { setSeeding(false) }
  }

  const typeFilters = [{ value: '', label: '全部' }, ...Object.entries(CW_COMP_TYPE_CONFIG).map(([k, v]) => ({ value: k, label: v.label }))]

  return (
    <div>
      {/* 顶部 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px', flexWrap: 'wrap', gap: '12px' }}>
        <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap' }}>
          {typeFilters.map(f => (
            <button key={f.value} onClick={() => setTypeFilter(f.value)} style={{
              padding: '6px 14px', borderRadius: '20px',
              border: `1px solid ${typeFilter === f.value ? C.primary : C.border}`,
              background: typeFilter === f.value ? 'rgba(245,158,11,0.08)' : 'transparent',
              color: typeFilter === f.value ? C.primary : C.textSecondary,
              fontSize: '13px', fontWeight: typeFilter === f.value ? 600 : 400, cursor: 'pointer',
            }}>{f.label}</button>
          ))}
        </div>
        {isAdmin && (
          <div style={{ display: 'flex', gap: '8px' }}>
            <button onClick={() => handleSeed(false)} disabled={seeding} style={{
              padding: '7px 16px', borderRadius: '8px', border: `1px solid ${C.primary}`,
              background: 'transparent', color: C.primary, fontSize: '13px', fontWeight: 600, cursor: seeding ? 'wait' : 'pointer',
            }}>{seeding ? '填充中...' : '🌱 填充种子数据'}</button>
            <button onClick={() => handleSeed(true)} disabled={seeding} style={{
              padding: '7px 16px', borderRadius: '8px', border: `1px solid ${C.danger}`,
              background: 'transparent', color: C.danger, fontSize: '13px', fontWeight: 600, cursor: seeding ? 'wait' : 'pointer',
            }}>🔄 强制重建</button>
          </div>
        )}
      </div>

      <div style={{ fontSize: '13px', color: C.textMuted, marginBottom: '16px' }}>共 {total} 个组件</div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: '60px 0', color: C.textMuted }}>加载中...</div>
      ) : items.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '80px 0' }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>🧩</div>
          <div style={{ fontSize: '16px', color: C.textSecondary, marginBottom: '8px' }}>组件库暂无内容</div>
          <div style={{ fontSize: '13px', color: C.textMuted }}>{isAdmin ? '点击「🌱 填充种子数据」一键入库' : '等待管理员填充数据'}</div>
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(300px, 1fr))', gap: '16px' }}>
          {items.map(item => {
            const tc = CW_COMP_TYPE_CONFIG[item.component_type] || { label: item.component_type, color: '#6B7280', bg: '#F3F4F6' }
            return (
              <div key={item.id} onClick={() => openDetail(item.id)} style={{
                background: C.bgCard, borderRadius: '12px', padding: '20px', border: `1px solid ${C.border}`, cursor: 'pointer', transition: 'all 200ms ease',
              }} onMouseEnter={e => { (e.currentTarget as HTMLDivElement).style.borderColor = 'rgba(245,158,11,0.3)'; (e.currentTarget as HTMLDivElement).style.transform = 'translateY(-2px)' }}
                 onMouseLeave={e => { (e.currentTarget as HTMLDivElement).style.borderColor = C.border; (e.currentTarget as HTMLDivElement).style.transform = 'none' }}>
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '10px' }}>
                  <div style={{ fontSize: '15px', fontWeight: 600, color: C.textPrimary }}>{item.name}</div>
                  <span style={{ padding: '2px 10px', borderRadius: '12px', fontSize: '11px', fontWeight: 500, color: tc.color, background: tc.bg, flexShrink: 0 }}>{tc.label}</span>
                </div>
                {item.description && <div style={{ fontSize: '13px', color: C.textSecondary, marginBottom: '8px', lineHeight: 1.5 }}>{item.description.length > 80 ? item.description.slice(0, 80) + '...' : item.description}</div>}
                <div style={{ display: 'flex', gap: '12px', fontSize: '12px', color: C.textMuted, justifyContent: 'space-between' }}>
                  <div style={{ display: 'flex', gap: '12px' }}>
                    <span>📚 {item.subject_scope}</span><span>🎓 {item.grade_scope}</span>
                    {item.idx_interaction_level != null && <span>⚡ IL:{item.idx_interaction_level}</span>}
                  </div>
                  {isAdmin && <button onClick={e => { e.stopPropagation(); handleDelete(item.id, item.name) }} style={{ padding: '2px 8px', borderRadius: '4px', border: `1px solid ${C.border}`, background: 'transparent', color: C.danger, fontSize: '11px', cursor: 'pointer' }}>删除</button>}
                </div>
              </div>
            )
          })}
        </div>
      )}

      {/* 详情弹窗 */}
      {(detailComp || detailLoading) && (
        <div style={{ position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.5)', zIndex: 9999, display: 'flex', alignItems: 'center', justifyContent: 'center' }} onClick={() => setDetailComp(null)}>
          <div style={{ background: '#fff', borderRadius: '16px', width: '90%', maxWidth: '1000px', maxHeight: '85vh', overflow: 'auto' }} onClick={e => e.stopPropagation()}>
            {detailLoading ? <div style={{ padding: '60px', textAlign: 'center', color: C.textMuted }}>加载中...</div> : detailComp && (
              <>
                <div style={{ padding: '24px 28px 16px', borderBottom: `1px solid ${C.border}`, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <div>
                    <div style={{ fontSize: '20px', fontWeight: 700, color: C.textPrimary, marginBottom: '6px' }}>{detailComp.name}</div>
                    <div style={{ display: 'flex', gap: '10px', fontSize: '12px', color: C.textMuted, flexWrap: 'wrap' }}>
                      <span style={{ padding: '2px 10px', borderRadius: '12px', fontSize: '11px', fontWeight: 500, color: (CW_COMP_TYPE_CONFIG[detailComp.component_type] || {}).color || '#6B7280', background: (CW_COMP_TYPE_CONFIG[detailComp.component_type] || {}).bg || '#F3F4F6' }}>{(CW_COMP_TYPE_CONFIG[detailComp.component_type] || {}).label || detailComp.component_type}</span>
                      <span>📚 {detailComp.subject_scope}</span><span>🎓 {detailComp.grade_scope}</span>
                      {detailComp.idx_interaction_level != null && <span>⚡ IL:{detailComp.idx_interaction_level}</span>}
                    </div>
                    {detailComp.description && <div style={{ fontSize: '13px', color: C.textSecondary, marginTop: '8px', lineHeight: 1.5 }}>{detailComp.description}</div>}
                  </div>
                  <button onClick={() => setDetailComp(null)} style={{ background: 'none', border: 'none', fontSize: '24px', cursor: 'pointer', color: C.textMuted, padding: '0 4px' }}>✕</button>
                </div>
                <div style={{ padding: '20px 28px' }}>
                  <div style={{ fontSize: '14px', fontWeight: 600, color: C.textPrimary, marginBottom: '12px' }}>📺 实时预览</div>
                  <div style={{ border: `1px solid ${C.border}`, borderRadius: '12px', overflow: 'hidden', background: '#f9f9f9' }}>
                    <iframe srcDoc={detailComp.code_content} style={{ width: '100%', height: '400px', border: 'none' }} sandbox="allow-scripts" title="组件预览" />
                  </div>
                </div>
                <details style={{ padding: '0 28px 24px' }}>
                  <summary style={{ fontSize: '14px', fontWeight: 600, color: C.textPrimary, cursor: 'pointer', marginBottom: '8px' }}>💻 查看源代码（{detailComp.code_content.length} 字符）</summary>
                  <pre style={{ background: '#1E293B', color: '#E2E8F0', padding: '16px', borderRadius: '8px', fontSize: '12px', lineHeight: 1.6, overflow: 'auto', maxHeight: '300px', whiteSpace: 'pre-wrap', wordBreak: 'break-all' }}>{detailComp.code_content}</pre>
                </details>
              </>
            )}
          </div>
        </div>
      )}
    </div>
  )
}

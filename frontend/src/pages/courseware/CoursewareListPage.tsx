/**
 * 课件工坊列表页 — CoursewareListPage v2.1（Phase 2 - Unicode fix）
 */
import { useState, useEffect } from 'react'
import { useNavigate } from 'react-router-dom'
import { getCoursewares, deleteCourseware, createCourseware, CW_STATUS_CONFIG } from '@/api/coursewares'
import type { CoursewareListItem } from '@/api/coursewares'
import apiClient from '@/api/client'

const C = {
  primary: '#F59E0B', primaryBg: 'rgba(245,158,11,0.08)',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
  border: '#E5E7EB',
}

interface LPItem { id: string; title: string; subject: string; grade: string; status: string }

export default function CoursewareListPage() {
  const navigate = useNavigate()
  const [items, setItems] = useState<CoursewareListItem[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)
  const [statusFilter, setStatusFilter] = useState('')
  const [showCreate, setShowCreate] = useState(false)
  const [plans, setPlans] = useState<LPItem[]>([])
  const [plansLoading, setPlansLoading] = useState(false)
  const [selectedPlanId, setSelectedPlanId] = useState('')
  const [creating, setCreating] = useState(false)

  useEffect(() => { loadData() }, [statusFilter])

  const loadData = async () => {
    setLoading(true)
    try {
      const resp = await getCoursewares({ status: statusFilter || undefined, limit: 50 })
      setItems(resp.coursewares || []); setTotal(resp.total)
    } catch { /* */ } finally { setLoading(false) }
  }

  const handleDelete = async (id: string, title: string) => {
    if (!window.confirm('确定删除课件「' + title + '」？')) return
    try { await deleteCourseware(id); loadData() } catch { alert('删除失败') }
  }

  const openCreateModal = async () => {
    setShowCreate(true); setSelectedPlanId(''); setPlansLoading(true)
    try {
      const resp = await apiClient.get('/lesson-plans', { params: { limit: 100 } })
      const data = resp?.data?.data
      const all: LPItem[] = (data?.plans || data?.lesson_plans || []) as LPItem[]
      setPlans(all.filter((p: LPItem) => ['published_personal', 'approved', 'published_shared'].includes(p.status)))
    } catch { setPlans([]) } finally { setPlansLoading(false) }
  }

  const handleCreate = async () => {
    if (!selectedPlanId) { alert('请选择关联的教案'); return }
    setCreating(true)
    try {
      await createCourseware({ lesson_plan_id: selectedPlanId })
      setShowCreate(false); loadData()
    } catch { alert('创建课件失败') } finally { setCreating(false) }
  }

  const statusFilters = [
    { value: '', label: '全部' }, { value: 'draft', label: '草稿' },
    { value: 'generating', label: '生成中' }, { value: 'preview', label: '预览中' },
    { value: 'confirmed', label: '已确认' }, { value: 'in_pipeline', label: '审核中' },
  ]

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <div style={{ display: 'flex', gap: '8px' }}>
          {statusFilters.map(f => (
            <button key={f.value} onClick={() => setStatusFilter(f.value)} style={{
              padding: '6px 16px', borderRadius: '20px',
              border: `1px solid ${statusFilter === f.value ? C.primary : C.border}`,
              background: statusFilter === f.value ? C.primaryBg : 'transparent',
              color: statusFilter === f.value ? C.primary : C.textSecondary,
              fontSize: '13px', fontWeight: statusFilter === f.value ? 600 : 400, cursor: 'pointer',
            }}>{f.label}</button>
          ))}
        </div>
        <button onClick={openCreateModal} style={{
          padding: '8px 20px', borderRadius: '8px', border: 'none',
          background: 'linear-gradient(135deg, #F59E0B, #EF4444)', color: '#fff',
          fontSize: '14px', fontWeight: 600, cursor: 'pointer',
          boxShadow: '0 2px 8px rgba(245,158,11,0.3)',
        }}>+ 新建课件</button>
      </div>

      <div style={{ fontSize: '13px', color: C.textMuted, marginBottom: '16px' }}>共 {total} 套课件</div>

      {loading ? (
        <div style={{ textAlign: 'center', padding: '60px 0', color: C.textMuted }}>加载中...</div>
      ) : items.length === 0 ? (
        <div style={{ textAlign: 'center', padding: '80px 0' }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>🎨</div>
          <div style={{ fontSize: '16px', color: C.textSecondary, marginBottom: '8px' }}>还没有课件</div>
          <div style={{ fontSize: '13px', color: C.textMuted }}>从教案出发，AI帮你自动生成课件</div>
        </div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(320px, 1fr))', gap: '16px' }}>
          {items.map(item => (
            <CWCard key={item.id} item={item} onDelete={handleDelete}
              onClick={() => navigate('/courseware/' + item.id)} />
          ))}
        </div>
      )}

      {showCreate && (
        <div style={{ position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.5)', zIndex: 9999, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
          onClick={() => setShowCreate(false)}>
          <div style={{ background: '#fff', borderRadius: '16px', width: '90%', maxWidth: '560px', maxHeight: '70vh', overflow: 'auto', padding: '28px' }}
            onClick={e => e.stopPropagation()}>
            <div style={{ fontSize: '20px', fontWeight: 700, color: C.textPrimary, marginBottom: '20px' }}>🎨 新建课件</div>
            <div style={{ fontSize: '14px', color: C.textSecondary, marginBottom: '16px' }}>选择一份已完成的教案，AI将基于教案内容自动生成课件</div>
            {plansLoading ? (
              <div style={{ textAlign: 'center', padding: '40px 0', color: C.textMuted }}>加载教案列表...</div>
            ) : plans.length === 0 ? (
              <div style={{ textAlign: 'center', padding: '40px 0' }}>
                <div style={{ fontSize: '14px', color: C.textSecondary }}>没有可用的教案</div>
                <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '6px' }}>请先在备课工坊完成教案开发</div>
              </div>
            ) : (
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '300px', overflow: 'auto', marginBottom: '20px' }}>
                {plans.map(p => (
                  <div key={p.id} onClick={() => setSelectedPlanId(p.id)} style={{
                    padding: '14px 16px', borderRadius: '10px', cursor: 'pointer',
                    border: `2px solid ${selectedPlanId === p.id ? C.primary : C.border}`,
                    background: selectedPlanId === p.id ? 'rgba(245,158,11,0.05)' : '#fff',
                  }}>
                    <div style={{ fontSize: '14px', fontWeight: 600, color: C.textPrimary }}>{p.title}</div>
                    <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px', display: 'flex', gap: '10px' }}>
                      {p.subject && <span>📚 {p.subject}</span>}
                      {p.grade && <span>🎓 {p.grade}</span>}
                    </div>
                  </div>
                ))}
              </div>
            )}
            <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '8px' }}>
              <button onClick={() => setShowCreate(false)} style={{
                padding: '8px 20px', borderRadius: '8px', border: `1px solid ${C.border}`,
                background: 'transparent', color: C.textSecondary, fontSize: '14px', cursor: 'pointer',
              }}>取消</button>
              <button onClick={handleCreate} disabled={!selectedPlanId || creating} style={{
                padding: '8px 24px', borderRadius: '8px', border: 'none',
                background: selectedPlanId ? 'linear-gradient(135deg, #F59E0B, #EF4444)' : '#E5E7EB',
                color: selectedPlanId ? '#fff' : '#9CA3AF',
                fontSize: '14px', fontWeight: 600,
                cursor: selectedPlanId && !creating ? 'pointer' : 'default',
              }}>{creating ? '创建中...' : '确认创建'}</button>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}

function CWCard({ item, onDelete, onClick }: {
  item: CoursewareListItem; onDelete: (id: string, t: string) => void; onClick: () => void
}) {
  const [hovered, setHovered] = useState(false)
  const sc = CW_STATUS_CONFIG[item.status] || { label: item.status, color: '#6B7280', bg: '#F3F4F6' }
  return (
    <div onClick={onClick} onMouseEnter={() => setHovered(true)} onMouseLeave={() => setHovered(false)}
      style={{
        background: '#fff', borderRadius: '12px', padding: '20px',
        border: `1px solid ${hovered ? 'rgba(245,158,11,0.3)' : '#E5E7EB'}`,
        cursor: 'pointer', transition: 'all 200ms ease',
        transform: hovered ? 'translateY(-2px)' : 'none',
        boxShadow: hovered ? '0 4px 16px rgba(0,0,0,0.08)' : '0 1px 3px rgba(0,0,0,0.04)',
      }}>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '12px' }}>
        <div style={{ fontSize: '16px', fontWeight: 600, color: '#1F2937', flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{item.title}</div>
        <span style={{ padding: '2px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 500, color: sc.color, background: sc.bg, flexShrink: 0, marginLeft: '8px' }}>{sc.label}</span>
      </div>
      <div style={{ fontSize: '13px', color: '#6B7280', marginBottom: '8px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
        📝 来源教案：{item.lesson_plan_title || '未知'}
      </div>
      <div style={{ display: 'flex', gap: '16px', fontSize: '12px', color: '#9CA3AF' }}>
        {item.subject && <span>📚 {item.subject}</span>}
        {item.grade && <span>🎓 {item.grade}</span>}
        <span>📄 {item.page_count} 页</span>
      </div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '14px', paddingTop: '12px', borderTop: '1px solid #E5E7EB' }}>
        <span style={{ fontSize: '12px', color: '#9CA3AF' }}>{item.created_at ? new Date(item.created_at).toLocaleDateString('zh-CN') : ''}</span>
        {item.status === 'draft' && (
          <button onClick={e => { e.stopPropagation(); onDelete(item.id, item.title) }}
            style={{ padding: '2px 10px', borderRadius: '6px', border: '1px solid #E5E7EB', background: 'transparent', color: '#EF4444', fontSize: '12px', cursor: 'pointer' }}>
            删除
          </button>
        )}
      </div>
    </div>
  )
}

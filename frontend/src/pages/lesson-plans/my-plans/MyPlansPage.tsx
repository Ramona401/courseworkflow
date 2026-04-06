/**
 * MyPlansPage — 我的教案页面（主文件）
 *
 * 功能：
 *   1. 教案列表展示（卡片式）
 *   2. 状态筛选栏 + 学科/年级二级筛选
 *   3. 卡片状态操作 + 查看详情
 *   4. 空状态引导到备课工坊
 *
 * 子组件均从 ./components/ 引入，本文件只保留：
 *   - 筛选状态管理 + 数据加载
 *   - 操作函数（handleAction）
 *   - 页面级渲染框架（标题/筛选栏/卡片网格/Toast）
 *
 * v56修改：移除页面内重复标题（LPLayout header已有标题）
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getLessonPlans, deleteLessonPlan,
  publishLessonPlanPersonal, submitLessonPlanForReview, startDevelopment,
  type LessonPlan,
} from '@/api/lesson-plans'

// ---- 子组件 ----
import { C, STATUS_FILTERS, SUBJECTS, GRADES } from './components/myPlansConstants'
import { PlanCard, SkeletonCard, EmptyState } from './components/MyPlanCards'

// ==================== 主组件 ====================
export default function MyPlansPage() {
  const { user }  = useAuth()
  const navigate  = useNavigate()

  const [plans, setPlans]     = useState<LessonPlan[]>([])
  const [total, setTotal]     = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError]     = useState<string | null>(null)

  // ---- 筛选状态 ----
  const [statusFilter,  setStatusFilter]  = useState<string>('all')
  const [subjectFilter, setSubjectFilter] = useState('全部')
  const [gradeFilter,   setGradeFilter]   = useState('全部')

  // ---- 操作状态 ----
  const [loadingId, setLoadingId] = useState<string | null>(null)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  // ==================== 数据加载 ====================
  const loadPlans = useCallback(async () => {
    if (!user) return
    setLoading(true); setError(null)
    try {
      const statusGroup = STATUS_FILTERS.find(f => f.key === statusFilter)
      // 多状态筛选在客户端过滤；单状态直接传给后端
      const needsClientFilter = statusGroup?.statuses && statusGroup.statuses.length > 1

      const params: Record<string, string | number> = { author_id: user.id, limit: 100 }
      if (statusGroup?.statuses?.length === 1) params.status = statusGroup.statuses[0]
      if (subjectFilter !== '全部') params.subject = subjectFilter
      if (gradeFilter   !== '全部') params.grade   = gradeFilter

      const resp = await getLessonPlans(params)
      let list = resp.lesson_plans || []

      if (needsClientFilter && statusGroup?.statuses) {
        const allowed = new Set(statusGroup.statuses)
        list = list.filter(p => allowed.has(p.status))
      }

      setPlans(list); setTotal(list.length)
    } catch (e) {
      console.error('加载教案列表失败:', e)
      setError('加载失败，请稍后重试')
    } finally {
      setLoading(false)
    }
  }, [user, statusFilter, subjectFilter, gradeFilter])

  useEffect(() => { loadPlans() }, [loadPlans])

  const handleReset = () => {
    setStatusFilter('all'); setSubjectFilter('全部'); setGradeFilter('全部')
  }

  const isFiltered = statusFilter !== 'all' || subjectFilter !== '全部' || gradeFilter !== '全部'

  // ==================== 操作处理 ====================
  const handleAction = async (planId: string, action: string) => {
    if (loadingId) return
    setLoadingId(planId)
    try {
      switch (action) {
        case 'resume':
          // 继续备课：跳转到备课工坊并恢复对话
          navigate('/lesson-plans', { state: { resumePlanId: planId } })
          setLoadingId(null)
          return
        case 'publish':  await publishLessonPlanPersonal(planId); showToast('教案已个人发布 ✓'); break
        case 'submit':   await submitLessonPlanForReview(planId);  showToast('已提交教研组评审 ✓'); break
        case 'develop':  await startDevelopment(planId);           showToast('已进入课件开发流程 ✓'); break
        case 'delete':   await deleteLessonPlan(planId);           showToast('教案已删除'); break
        default: console.warn('未知操作:', action)
      }
      await loadPlans()
    } catch (e: unknown) {
      console.error(`操作${action}失败:`, e)
      showToast(e instanceof Error ? e.message : '操作失败，请稍后重试', 'error')
    } finally {
      setLoadingId(null)
    }
  }

  // 各状态计数（用于筛选按钮徽章）
  const statusCounts = plans.reduce<Record<string, number>>((acc, p) => {
    acc[p.status] = (acc[p.status] || 0) + 1; return acc
  }, {})

  // ==================== 渲染 ====================
  return (
    <div>
      {/* ---- 摘要行 + 新建按钮（标题已在LPLayout header中显示，此处不再重复） ---- */}
      <div style={{ marginBottom: '20px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <p style={{ fontSize: '14px', color: C.textSec, margin: 0 }}>
          共 {total} 份教案{isFiltered ? `（筛选后 ${plans.length} 份）` : ''}
        </p>
        <button
          onClick={() => navigate('/lesson-plans')}
          style={{ display: 'flex', alignItems: 'center', gap: '6px', padding: '9px 18px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer', transition: 'all 150ms ease', flexShrink: 0 }}
          onMouseEnter={e => { (e.currentTarget as HTMLButtonElement).style.opacity = '0.88' }}
          onMouseLeave={e => { (e.currentTarget as HTMLButtonElement).style.opacity = '1' }}>
          <span>✨</span><span>新建教案</span>
        </button>
      </div>

      {/* ---- 筛选栏 ---- */}
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '16px 20px', marginBottom: '20px' }}>
        {/* 状态筛选按钮行 */}
        <div style={{ marginBottom: '12px' }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec, marginRight: '12px' }}>状态</span>
          <div style={{ display: 'inline-flex', flexWrap: 'wrap', gap: '6px' }}>
            {STATUS_FILTERS.map(f => {
              const isActive = statusFilter === f.key
              const count = f.statuses
                ? f.statuses.reduce((s, st) => s + (statusCounts[st] || 0), 0)
                : total
              return (
                <button
                  key={f.key}
                  onClick={() => setStatusFilter(f.key)}
                  style={{
                    padding: '5px 12px', borderRadius: '20px',
                    border: `1px solid ${isActive ? C.primary : C.border}`,
                    background: isActive ? C.primaryLight : 'transparent',
                    color: isActive ? C.primary : C.textSec,
                    fontSize: '13px', fontWeight: isActive ? 600 : 400,
                    cursor: 'pointer', transition: 'all 150ms ease',
                  }}>
                  {f.label}
                  {count > 0 && (
                    <span style={{ marginLeft: '5px', padding: '0 5px', background: isActive ? C.primary : C.border, color: isActive ? '#fff' : C.textMuted, borderRadius: '10px', fontSize: '11px', fontWeight: 600 }}>
                      {count}
                    </span>
                  )}
                </button>
              )
            })}
          </div>
        </div>

        {/* 学科 + 年级下拉 */}
        <div style={{ display: 'flex', gap: '16px', flexWrap: 'wrap', alignItems: 'center' }}>
          {[
            { label: '学科', value: subjectFilter, options: SUBJECTS, onChange: setSubjectFilter },
            { label: '年级', value: gradeFilter,   options: GRADES,   onChange: setGradeFilter   },
          ].map(({ label, value, options, onChange }) => (
            <div key={label} style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec, flexShrink: 0 }}>{label}</span>
              <select
                value={value} onChange={e => onChange(e.target.value)}
                style={{
                  padding: '5px 10px', borderRadius: '6px',
                  border: `1px solid ${value !== '全部' ? C.primary : C.border}`,
                  background: value !== '全部' ? C.primaryLight : 'transparent',
                  color: value !== '全部' ? C.primary : C.textSec,
                  fontSize: '13px', cursor: 'pointer', outline: 'none',
                }}>
                {options.map(o => <option key={o} value={o}>{o}</option>)}
              </select>
            </div>
          ))}
          {isFiltered && (
            <button onClick={handleReset} style={{ padding: '5px 10px', borderRadius: '6px', border: 'none', background: 'transparent', fontSize: '12px', color: C.textMuted, cursor: 'pointer', textDecoration: 'underline' }}>
              清空筛选
            </button>
          )}
        </div>
      </div>

      {/* ---- 错误提示 ---- */}
      {error && (
        <div style={{ padding: '12px 16px', background: '#FEF2F2', border: '1px solid #FECACA', borderRadius: '8px', marginBottom: '16px', fontSize: '14px', color: C.danger, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span>⚠️ {error}</span>
          <button onClick={loadPlans} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.primary, fontSize: '13px' }}>重试</button>
        </div>
      )}

      {/* ---- 教案卡片网格 ---- */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))', gap: '16px' }}>
        {loading && Array.from({ length: 6 }).map((_, i) => <SkeletonCard key={i} />)}
        {!loading && plans.map(plan => (
          <PlanCard key={plan.id} plan={plan} onAction={handleAction} loadingId={loadingId} />
        ))}
        {!loading && !error && plans.length === 0 && (
          <EmptyState filtered={isFiltered} onReset={handleReset} />
        )}
      </div>

      {/* ---- Toast通知 ---- */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)',
          padding: '12px 24px', borderRadius: '10px',
          background: toast.type === 'error' ? '#FEF2F2' : '#1F2937',
          color: toast.type === 'error' ? C.danger : '#fff',
          fontSize: '14px', fontWeight: 500,
          boxShadow: '0 8px 24px rgba(0,0,0,0.15)', zIndex: 9999, whiteSpace: 'nowrap',
          border: toast.type === 'error' ? '1px solid #FECACA' : 'none',
          animation: 'toast-in 200ms ease',
        }}>
          <style>{`@keyframes toast-in { from{opacity:0;transform:translateX(-50%) translateY(8px)} to{opacity:1;transform:translateX(-50%) translateY(0)} }`}</style>
          {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
        </div>
      )}
    </div>
  )
}

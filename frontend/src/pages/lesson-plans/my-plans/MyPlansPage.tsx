/**
 * MyPlansPage — 我的教案页面（主文件）
 *
 * 功能：
 *   1. 教案列表展示（卡片式）
 *   2. 状态筛选栏 + 学科/年级二级筛选
 *   3. 卡片状态操作 + 查看详情
 *   4. 空状态引导到备课工坊
 *
 * v101修复：提交评审前弹出教研组选择弹窗，修复提交参数错误问题
 *   - 点击"提交评审"先拉取 /my-groups 获取当前用户所在教研组
 *   - 弹窗让用户选择目标教研组后再调用 submitLessonPlanForReview(id, groupId)
 *   - 若用户未加入任何教研组，给出友好提示
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import { createCourseware } from '@/api/coursewares'
import {
  getLessonPlans, deleteLessonPlan,
  publishLessonPlanPersonal, submitLessonPlanForReview, startDevelopment,
  getMyGroups,
  type LessonPlan, type TeachingGroup,
} from '@/api/lesson-plans'

// ---- 子组件 ----
import { C, STATUS_FILTERS, SUBJECTS, GRADES } from './components/myPlansConstants'
import { PlanCard, SkeletonCard, EmptyState } from './components/MyPlanCards'

// ==================== 教研组选择弹窗 ====================

interface SubmitReviewModalProps {
  planTitle: string
  groups: TeachingGroup[]
  loading: boolean
  onConfirm: (groupId: string) => void
  onCancel: () => void
}

function SubmitReviewModal({ planTitle, groups, loading, onConfirm, onCancel }: SubmitReviewModalProps) {
  const [selectedGroupId, setSelectedGroupId] = useState<string>('')

  useEffect(() => {
    if (groups.length === 1) setSelectedGroupId(groups[0].id)
  }, [groups])

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 10000,
      background: 'rgba(0,0,0,0.45)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: '20px',
    }}
    onClick={onCancel}>
      <div
        style={{
          background: '#fff', borderRadius: '14px',
          width: '100%', maxWidth: '440px',
          boxShadow: '0 20px 60px rgba(0,0,0,0.18)',
          overflow: 'hidden',
        }}
        onClick={e => e.stopPropagation()}>

        <div style={{ padding: '20px 24px 16px', borderBottom: `1px solid ${C.border}` }}>
          <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 700, color: C.text }}>
            📤 提交教案评审
          </h3>
          <p style={{ margin: '6px 0 0', fontSize: '13px', color: C.textSec, lineHeight: 1.5 }}>
            将「{planTitle}」提交到教研组进行评审
          </p>
        </div>

        <div style={{ padding: '20px 24px' }}>
          {loading ? (
            <div style={{ textAlign: 'center', padding: '24px 0', color: C.textMuted, fontSize: '14px' }}>
              <div style={{ fontSize: '24px', marginBottom: '8px' }}>⏳</div>
              正在获取您的教研组...
            </div>
          ) : groups.length === 0 ? (
            <div style={{
              textAlign: 'center', padding: '24px 0',
              background: 'rgba(249,115,22,0.05)', borderRadius: '10px',
            }}>
              <div style={{ fontSize: '32px', marginBottom: '10px' }}>🏫</div>
              <p style={{ margin: '0 0 6px', fontSize: '14px', fontWeight: 600, color: C.text }}>
                您还未加入任何教研组
              </p>
              <p style={{ margin: 0, fontSize: '13px', color: C.textSec, lineHeight: 1.6 }}>
                请联系学校管理员将您加入教研组后，<br />再提交教案评审
              </p>
            </div>
          ) : (
            <>
              <p style={{ margin: '0 0 12px', fontSize: '13px', color: C.textSec }}>
                请选择要提交到的教研组：
              </p>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', maxHeight: '240px', overflowY: 'auto' }}>
                {groups.map(g => {
                  const isSelected = selectedGroupId === g.id
                  return (
                    <div
                      key={g.id}
                      onClick={() => setSelectedGroupId(g.id)}
                      style={{
                        padding: '12px 14px', borderRadius: '10px', cursor: 'pointer',
                        border: `2px solid ${isSelected ? C.primary : C.border}`,
                        background: isSelected ? C.primaryLight : '#fff',
                        transition: 'all 150ms ease',
                      }}>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                        <div>
                          <div style={{ fontSize: '14px', fontWeight: isSelected ? 600 : 400, color: isSelected ? C.primary : C.text }}>
                            {g.name}
                          </div>
                          {(g.subject || g.grade_range) && (
                            <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '3px' }}>
                              {[g.subject, g.grade_range].filter(Boolean).join(' · ')}
                              {g.school_name && ` · ${g.school_name}`}
                            </div>
                          )}
                        </div>
                        {isSelected && (
                          <span style={{ fontSize: '18px', color: C.primary, flexShrink: 0 }}>✓</span>
                        )}
                      </div>
                    </div>
                  )
                })}
              </div>
            </>
          )}
        </div>

        <div style={{
          padding: '16px 24px', borderTop: `1px solid ${C.border}`,
          display: 'flex', gap: '10px', justifyContent: 'flex-end',
        }}>
          <button
            onClick={onCancel}
            style={{
              padding: '9px 20px', borderRadius: '8px',
              border: `1px solid ${C.border}`, background: 'transparent',
              fontSize: '14px', color: C.textSec, cursor: 'pointer',
            }}>
            取消
          </button>
          {!loading && groups.length > 0 && (
            <button
              onClick={() => selectedGroupId && onConfirm(selectedGroupId)}
              disabled={!selectedGroupId}
              style={{
                padding: '9px 20px', borderRadius: '8px', border: 'none',
                background: selectedGroupId ? C.primary : C.border,
                color: selectedGroupId ? '#fff' : C.textMuted,
                fontSize: '14px', fontWeight: 600,
                cursor: selectedGroupId ? 'pointer' : 'not-allowed',
                transition: 'all 150ms ease',
              }}>
              确认提交
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

// ==================== 主组件 ====================
export default function MyPlansPage() {
  const { user }  = useAuth()
  const navigate  = useNavigate()

  const [plans, setPlans]     = useState<LessonPlan[]>([])
  const [total, setTotal]     = useState(0)
  const [loading, setLoading] = useState(true)
  const [error, setError]     = useState<string | null>(null)

  const [statusFilter,  setStatusFilter]  = useState<string>('all')
  const [subjectFilter, setSubjectFilter] = useState('全部')
  const [gradeFilter,   setGradeFilter]   = useState('全部')

  const [loadingId, setLoadingId] = useState<string | null>(null)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const [submitModal, setSubmitModal] = useState<{
    planId: string
    planTitle: string
    groups: TeachingGroup[]
    groupsLoading: boolean
  } | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  const loadPlans = useCallback(async () => {
    if (!user) return
    setLoading(true); setError(null)
    try {
      const statusGroup = STATUS_FILTERS.find(f => f.key === statusFilter)
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

  // 打开教研组选择弹窗
  const openSubmitModal = async (planId: string) => {
    const plan = plans.find(p => p.id === planId)
    const planTitle = plan?.title ?? '当前教案'
    setSubmitModal({ planId, planTitle, groups: [], groupsLoading: true })
    try {
      const groups = await getMyGroups()
      setSubmitModal(prev => prev ? { ...prev, groups, groupsLoading: false } : null)
    } catch (e) {
      console.error('获取教研组列表失败:', e)
      setSubmitModal(null)
      showToast('获取教研组失败，请稍后重试', 'error')
    }
  }

  // 弹窗确认后执行提交
  const handleSubmitConfirm = async (groupId: string) => {
    if (!submitModal) return
    const { planId } = submitModal
    setSubmitModal(null)
    setLoadingId(planId)
    try {
      await submitLessonPlanForReview(planId, groupId)
      showToast('已提交教研组评审 ✓')
      await loadPlans()
    } catch (e: unknown) {
      console.error('提交评审失败:', e)
      showToast(e instanceof Error ? e.message : '提交评审失败，请稍后重试', 'error')
    } finally {
      setLoadingId(null)
    }
  }

  const handleAction = async (planId: string, action: string) => {
    if (loadingId) return

    if (action === 'submit') {
      await openSubmitModal(planId)
      return
    }

    setLoadingId(planId)
    try {
      switch (action) {
        case 'resume':
          navigate('/lesson-plans', { state: { resumePlanId: planId } })
          setLoadingId(null)
          return
        case 'publish':  await publishLessonPlanPersonal(planId); showToast('教案已个人发布 ✓'); break
        case 'courseware': {
          // Phase 3: 创建课件并跳转到课件工坊
          try {
            const cw = await createCourseware({ lesson_plan_id: planId })
            showToast('课件创建成功，正在跳转课件工坊... ✓')
            setTimeout(() => navigate('/courseware/' + (cw as { id: string }).id), 500)
          } catch {
            showToast('创建课件失败，请稍后重试', 'error')
          }
          break
        }
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

  const statusCounts = plans.reduce<Record<string, number>>((acc, p) => {
    acc[p.status] = (acc[p.status] || 0) + 1; return acc
  }, {})

  return (
    <div>
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

      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '16px 20px', marginBottom: '20px' }}>
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

      {error && (
        <div style={{ padding: '12px 16px', background: '#FEF2F2', border: '1px solid #FECACA', borderRadius: '8px', marginBottom: '16px', fontSize: '14px', color: C.danger, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span>⚠️ {error}</span>
          <button onClick={loadPlans} style={{ background: 'none', border: 'none', cursor: 'pointer', color: C.primary, fontSize: '13px' }}>重试</button>
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(340px, 1fr))', gap: '16px' }}>
        {loading && Array.from({ length: 6 }).map((_, i) => <SkeletonCard key={i} />)}
        {!loading && plans.map(plan => (
          <PlanCard key={plan.id} plan={plan} onAction={handleAction} loadingId={loadingId} />
        ))}
        {!loading && !error && plans.length === 0 && (
          <EmptyState filtered={isFiltered} onReset={handleReset} />
        )}
      </div>

      {submitModal && (
        <SubmitReviewModal
          planTitle={submitModal.planTitle}
          groups={submitModal.groups}
          loading={submitModal.groupsLoading}
          onConfirm={handleSubmitConfirm}
          onCancel={() => setSubmitModal(null)}
        />
      )}

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

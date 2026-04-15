/**
 * PlanDetailPage — 教案详情页（主文件）
 *
 * 功能：
 *   1. 教案基本信息头部（标题/学科/年级/课时/状态/作者）
 *   2. 四Tab内容区：教案内容 / AI评审 / 使用统计 / 关联课件
 *   3. 操作栏：发布/提交评审/进入课件开发/Fork/删除（按状态动态渲染）
 *   4. 返回按钮（智能返回：从库来→库，从我的来→我的）
 *
 * v101修复：提交评审前弹出教研组选择弹窗，修复提交参数错误
 * v104改动：
 *   - 批注数据提升到 PlanDetailPage 统一加载管理（解决RevisionBanner与ContentTab各自请求问题）
 *   - RevisionBanner 直接接收 annotations prop，数字实时响应批注状态变化
 *   - ContentTab 接收 annotations / onAnnotationsChange props，不再内部独立请求
 *   - 返回备课工坊后显示Toast提示当前为退回修改模式
 */
import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getLessonPlan,
  publishLessonPlanPersonal,
  submitLessonPlanForReview,
  startDevelopment,
  deleteLessonPlan,
  forkLessonPlan,
  getMyGroups,
  type LessonPlan,
  type StartDevelopmentResult,
  type TeachingGroup,
} from '@/api/lesson-plans'
import { getAnnotations, type Annotation } from '@/api/annotations'

// ---- 子组件 ----
import { C, TABS, type TabKey } from './components/planDetailConstants'
import { DetailSkeleton, StatusBadge, MetaTag, ActionBar } from './components/PlanDetailHeader'
import { ContentTab, ReviewTab, StatsTab, CoursewareTab } from './components/PlanDetailTabs'

// ==================== revision状态批注提示条 ====================

/**
 * RevisionBanner — 退回状态提示条
 * v104：直接接收 annotations prop，不再内部独立请求，数字实时更新
 */
function RevisionBanner({
  annotations,
  onSubmit,
  actionLoading,
}: {
  annotations: Annotation[]
  onSubmit: () => void
  actionLoading: string | null
}) {
  // v106修复：先找出最大轮次，再只统计该轮次的 pending 批注（避免历史轮次计入）
  const currentRound = annotations.reduce((max, a) => Math.max(max, a.review_round ?? 1), 1)
  const currentRoundAnnotations = annotations.filter(a => (a.review_round ?? 1) === currentRound)
  const pendingCount = currentRoundAnnotations.filter(a => a.status === 'pending').length

  return (
    <div style={{
      padding: '14px 20px',
      borderRadius: '10px',
      marginBottom: '16px',
      background: '#FFF7ED',
      border: '2px solid #F97316',
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'space-between',
      gap: '16px',
      flexWrap: 'wrap',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
        <span style={{ fontSize: '20px' }}>📋</span>
        <div>
          <div style={{ fontSize: '14px', fontWeight: 700, color: '#C2410C' }}>
            教案已被退回修改
            {currentRound > 1 && (
              <span style={{ marginLeft: '8px', fontSize: '12px', fontWeight: 400, color: '#92400E' }}>
                （第 {currentRound} 轮评审）
              </span>
            )}
          </div>
          <div style={{ fontSize: '13px', color: '#92400E', marginTop: '2px' }}>
            {pendingCount > 0
              ? `本轮共 ${pendingCount} 条批注待处理——在「教案内容」Tab中点击 💬 图标查看并修改`
              : '本轮批注已全部处理完成，可以重新提交评审'
            }
          </div>
        </div>
      </div>
      <button
        onClick={onSubmit}
        disabled={actionLoading === 'submit'}
        style={{
          padding: '9px 20px', borderRadius: '8px', border: 'none',
          background: actionLoading === 'submit' ? '#FED7AA' : '#F97316',
          color: '#fff', fontSize: '13px', fontWeight: 700,
          cursor: actionLoading === 'submit' ? 'not-allowed' : 'pointer',
          flexShrink: 0, whiteSpace: 'nowrap',
        }}
      >
        {actionLoading === 'submit' ? '提交中...' : '📤 重新提交评审'}
      </button>
    </div>
  )
}

// ==================== 教研组选择弹窗 ====================

interface SubmitReviewModalProps {
  planTitle: string
  groups: TeachingGroup[]
  loading: boolean
  onConfirm: (groupId: string) => void
  onCancel: () => void
}

/**
 * SubmitReviewModal — 提交评审时选择目标教研组的弹窗
 */
function SubmitReviewModal({ planTitle, groups, loading, onConfirm, onCancel }: SubmitReviewModalProps) {
  const [selectedGroupId, setSelectedGroupId] = useState<string>('')

  // 只有一个教研组时自动选中
  useEffect(() => {
    if (groups.length === 1) setSelectedGroupId(groups[0].id)
  }, [groups])

  return (
    <div
      style={{
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

        {/* 标题栏 */}
        <div style={{ padding: '20px 24px 16px', borderBottom: `1px solid ${C.border}` }}>
          <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 700, color: C.text }}>
            📤 提交教案评审
          </h3>
          <p style={{ margin: '6px 0 0', fontSize: '13px', color: C.textSec, lineHeight: 1.5 }}>
            将「{planTitle}」提交到教研组进行评审
          </p>
        </div>

        {/* 内容区 */}
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

        {/* 底部按钮 */}
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
export default function PlanDetailPage() {
  const { id }    = useParams<{ id: string }>()
  const navigate  = useNavigate()
  const location  = useLocation()
  const { user }  = useAuth()

  const [plan, setPlan]         = useState<LessonPlan | null>(null)
  const [loading, setLoading]   = useState(true)
  const [error, setError]       = useState<string | null>(null)
  const [activeTab, setActiveTab]   = useState<TabKey>('content')
  const [actionLoading, setActionLoading] = useState<string | null>(null)
  const [toast, setToast]       = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  // v104：批注数据提升到页面层，RevisionBanner与ContentTab共享同一份数据
  const [annotations, setAnnotations]           = useState<Annotation[]>([])
  const [annotationsLoading, setAnnotationsLoading] = useState(false)

  // ---- 教研组选择弹窗状态 ----
  const [submitModal, setSubmitModal] = useState<{
    groups: TeachingGroup[]
    groupsLoading: boolean
  } | null>(null)

  // ---- Toast工具 ----
  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type })
    setTimeout(() => setToast(null), 3000)
  }

  // ---- 加载教案数据 ----
  useEffect(() => {
    if (!id) return
    setLoading(true); setError(null)
    getLessonPlan(id)
      .then(data => { setPlan(data); setLoading(false) })
      .catch(() => { setError('加载失败，教案不存在或无权限查看'); setLoading(false) })
  }, [id])

  // v104：统一加载批注（仅教案属主需要，其他角色不需要）
  // 在教案加载完成后触发，后续由 handleAnnotationsChange 局部更新
  const loadAnnotations = useCallback(async (planId: string) => {
    setAnnotationsLoading(true)
    try {
      const resp = await getAnnotations(planId)
      setAnnotations(resp.annotations || [])
    } catch {
      setAnnotations([])
    } finally {
      setAnnotationsLoading(false)
    }
  }, [])

  useEffect(() => {
    if (plan?.id) {
      loadAnnotations(plan.id)
    }
  }, [plan?.id, loadAnnotations])

  // ---- 批注变化回调（ContentTab调用，更新页面层批注状态） ----
  // ContentTab 通过此回调通知页面层，RevisionBanner会自动响应
  const handleAnnotationsChange = useCallback((updated: Annotation[]) => {
    setAnnotations(updated)
  }, [])

  // ---- 智能返回 ----
  const handleBack = () => {
    const from = (location.state as { from?: string })?.from
    if (from) { navigate(from); return }
    navigate('/lesson-plans/my-plans')
  }

  // ---- 打开提交评审弹窗 ----
  const openSubmitModal = async () => {
    setSubmitModal({ groups: [], groupsLoading: true })
    try {
      const groups = await getMyGroups()
      setSubmitModal(prev => prev ? { ...prev, groups, groupsLoading: false } : null)
    } catch {
      setSubmitModal(null)
      showToast('获取教研组失败，请稍后重试', 'error')
    }
  }

  // ---- 确认提交评审 ----
  const handleSubmitConfirm = async (groupId: string) => {
    if (!plan) return
    setSubmitModal(null)
    setActionLoading('submit')
    try {
      await submitLessonPlanForReview(plan.id, groupId)
      showToast('已提交教研组评审 ✓')
      const refreshed = await getLessonPlan(plan.id)
      setPlan(refreshed)
      // 提交后后端已归档pending批注，重新拉取最新批注列表
      await loadAnnotations(plan.id)
    } catch (e) {
      console.error('提交评审失败:', e)
      showToast('提交失败，请稍后重试', 'error')
    } finally {
      setActionLoading(null)
    }
  }

  // ---- 操作处理 ----
  const handleAction = async (action: string) => {
    if (!plan || actionLoading) return

    // 提交评审走弹窗流程
    if (action === 'submit') {
      await openSubmitModal()
      return
    }

    setActionLoading(action)
    try {
      switch (action) {
        case 'publish':
          await publishLessonPlanPersonal(plan.id)
          showToast('教案已个人发布 ✓')
          break

        case 'develop': {
          const result = await startDevelopment(plan.id) as StartDevelopmentResult
          showToast('已创建课件开发任务 ✓')
          const refreshed = await getLessonPlan(plan.id)
          setPlan(refreshed)
          setActiveTab('courseware')
          setTimeout(() => {
            if (window.confirm(`课件开发任务已创建（Pipeline ID: ${result.pipeline_id}）\n是否立即前往课件审核系统查看？`)) {
              navigate(`/workflow/pipelines/${result.pipeline_id}`)
            }
          }, 500)
          return
        }

        case 'fork': {
          const forked = await forkLessonPlan(plan.id)
          showToast(`已Fork到我的草稿：${forked.title} ✓`)
          break
        }

        case 'delete':
          await deleteLessonPlan(plan.id)
          showToast('教案已删除')
          setTimeout(() => navigate('/lesson-plans/my-plans'), 1200)
          return
      }
      // 刷新教案数据
      const refreshed = await getLessonPlan(plan.id)
      setPlan(refreshed)
    } catch (e) {
      console.error(`操作${action}失败:`, e)
      showToast('操作失败，请稍后重试', 'error')
    } finally {
      setActionLoading(null)
    }
  }

  // ==================== 渲染 ====================

  if (loading) return <DetailSkeleton />

  if (error || !plan) {
    return (
      <div style={{ textAlign: 'center', padding: '80px 40px', background: C.card, borderRadius: '12px', border: `1px solid ${C.border}` }}>
        <div style={{ fontSize: '48px', marginBottom: '16px' }}>😕</div>
        <div style={{ fontSize: '16px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
          {error || '教案不存在'}
        </div>
        <button onClick={handleBack} style={{ marginTop: '16px', padding: '9px 20px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}>
          返回列表
        </button>
      </div>
    )
  }

  const isOwner = plan.author_id === user?.id

  return (
    <div>
      {/* 返回按钮 */}
      <button onClick={handleBack} style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '16px', padding: '6px 0', background: 'none', border: 'none', fontSize: '13px', color: C.textSec, cursor: 'pointer' }}>
        ← 返回
      </button>

      {/* ---- 头部信息卡 ---- */}
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '28px', marginBottom: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.04)' }}>
        {/* 标题行 */}
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px', marginBottom: '20px' }}>
          <h1 style={{ fontSize: '22px', fontWeight: 700, color: C.text, margin: 0, lineHeight: 1.4, flex: 1 }}>
            {plan.title}
          </h1>
          <StatusBadge status={plan.status} />
        </div>

        {/* 元信息标签组 */}
        <div style={{ display: 'flex', gap: '32px', flexWrap: 'wrap', paddingBottom: '20px', marginBottom: '20px', borderBottom: `1px solid ${C.border}` }}>
          <MetaTag icon="📚" label="学科" value={plan.subject} />
          <MetaTag icon="🎓" label="年级" value={plan.grade} />
          <MetaTag icon="⏱"  label="课时" value={`${plan.duration_minutes} 分钟`} />
          <MetaTag icon="📌" label="课题" value={plan.topic} />
          {plan.author_name && <MetaTag icon="👤" label="作者" value={plan.author_name} />}
          {plan.recipe_name && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
              <span style={{ fontSize: '11px', color: '#9CA3AF', fontWeight: 500 }}>配方</span>
              <button
                onClick={() => plan.recipe_id && navigate(`/lesson-plans/recipes/${plan.recipe_id}/edit`, { state: { from: `/lesson-plans/plans/${plan.id}` } })}
                style={{ fontSize: '14px', color: '#4F7BE8', display: 'flex', alignItems: 'center', gap: '4px', background: 'none', border: 'none', cursor: 'pointer', padding: 0 }}>
                <span>📦</span><span style={{ textDecoration: 'underline', textDecorationColor: 'rgba(79,123,232,0.3)' }}>{plan.recipe_name}</span>
              </button>
            </div>
          )}
          {plan.ai_review_score != null && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '2px' }}>
              <span style={{ fontSize: '11px', color: C.textMuted, fontWeight: 500 }}>AI评分</span>
              <span style={{ fontSize: '14px', fontWeight: 700, color: plan.ai_review_score >= 8.5 ? C.success : C.accent }}>
                🤖 {plan.ai_review_score.toFixed(1)}
              </span>
            </div>
          )}
        </div>

        {/* 操作按钮组 */}
        <ActionBar plan={plan} isOwner={isOwner} actionLoading={actionLoading} onAction={handleAction} />
      </div>

      {/* ---- revision状态：批注待处理提示条（v104：使用共享批注数据，数字实时响应） ---- */}
      {plan.status === 'revision' && isOwner && (
        <RevisionBanner
          annotations={annotations}
          onSubmit={openSubmitModal}
          actionLoading={actionLoading}
        />
      )}

      {/* ---- Tab内容卡 ---- */}
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, boxShadow: '0 1px 3px rgba(0,0,0,0.04)', overflow: 'hidden' }}>
        {/* Tab导航 */}
        <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}`, padding: '0 4px' }}>
          {TABS.map(tab => {
            const isActive = activeTab === tab.key
            let label = tab.label
            if (tab.key === 'review' && plan.ai_review_score != null) {
              label = `🤖 AI评审 ${plan.ai_review_score.toFixed(1)}`
            }
            if (tab.key === 'courseware' && plan.linked_pipeline_id) {
              label = '🔗 关联课件 ●'
            }
            return (
              <button key={tab.key} onClick={() => setActiveTab(tab.key)} style={{
                padding: '14px 20px', border: 'none', background: 'transparent',
                fontSize: '13px', fontWeight: isActive ? 600 : 400,
                color: isActive ? C.primary : C.textSec,
                cursor: 'pointer',
                borderBottom: isActive ? `2px solid ${C.primary}` : '2px solid transparent',
                marginBottom: '-1px', transition: 'all 150ms ease', whiteSpace: 'nowrap',
              }}>
                {label}
              </button>
            )
          })}
        </div>

        {/* Tab内容切换 */}
        {activeTab === 'content' && (
          <ContentTab
            plan={plan}
            isOwner={isOwner}
            annotations={annotations}
            annotationsLoading={annotationsLoading}
            onAnnotationsChange={handleAnnotationsChange}
          />
        )}
        {activeTab === 'review'     && <ReviewTab  plan={plan} />}
        {activeTab === 'stats'      && <StatsTab   plan={plan} />}
        {activeTab === 'courseware' && (
          <CoursewareTab
            plan={plan}
            onNavigatePipeline={pipelineId => {
              if (pipelineId) navigate(`/workflow/pipelines/${pipelineId}`)
              else navigate('/workflow/pipelines')
            }}
          />
        )}
      </div>

      {/* ---- 教研组选择弹窗 ---- */}
      {submitModal && (
        <SubmitReviewModal
          planTitle={plan.title}
          groups={submitModal.groups}
          loading={submitModal.groupsLoading}
          onConfirm={handleSubmitConfirm}
          onCancel={() => setSubmitModal(null)}
        />
      )}

      {/* Toast通知 */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)',
          padding: '12px 24px', borderRadius: '10px',
          background: toast.type === 'error' ? '#FEF2F2' : '#1F2937',
          color: toast.type === 'error' ? C.danger : '#fff',
          fontSize: '14px', fontWeight: 500, boxShadow: '0 8px 24px rgba(0,0,0,0.15)',
          zIndex: 9999, whiteSpace: 'nowrap',
          border: toast.type === 'error' ? '1px solid #FECACA' : 'none',
        }}>
          {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
        </div>
      )}
    </div>
  )
}

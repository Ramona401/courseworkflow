/**
 * PlanDetailPage — 教案详情页（主文件）
 *
 * 功能：
 *   1. 教案基本信息头部（标题/学科/年级/课时/状态/作者）
 *   2. 四Tab内容区：教案内容 / AI评审 / 使用统计 / 关联课件
 *   3. 操作栏：发布/提交评审/进入课件开发/Fork/删除（按状态动态渲染）
 *   4. 返回按钮（智能返回：从库来→库，从我的来→我的）
 *
 * 子组件均从 ./components/ 引入，本文件只保留：
 *   - 路由参数解析 + 数据加载
 *   - 操作函数（handleAction）
 *   - 页面级渲染框架（头部卡片 + Tab导航 + Tab内容切换）
 */
import { useState, useEffect } from 'react'
import { useParams, useNavigate, useLocation } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getLessonPlan,
  publishLessonPlanPersonal,
  submitLessonPlanForReview,
  startDevelopment,
  deleteLessonPlan,
  forkLessonPlan,
  type LessonPlan,
  type StartDevelopmentResult,
} from '@/api/lesson-plans'

// ---- 子组件 ----
import { C, TABS, type TabKey } from './components/planDetailConstants'
import { DetailSkeleton, StatusBadge, MetaTag, ActionBar } from './components/PlanDetailHeader'
import { ContentTab, ReviewTab, StatsTab, CoursewareTab } from './components/PlanDetailTabs'

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

  // ---- 智能返回 ----
  const handleBack = () => {
    const from = (location.state as { from?: string })?.from
    if (from) { navigate(from); return }
    navigate('/lesson-plans/my-plans')
  }

  // ---- 操作处理 ----
  const handleAction = async (action: string) => {
    if (!plan || actionLoading) return
    setActionLoading(action)
    try {
      switch (action) {
        case 'publish':
          await publishLessonPlanPersonal(plan.id)
          showToast('教案已个人发布 ✓')
          break

        case 'submit':
          await submitLessonPlanForReview(plan.id)
          showToast('已提交教研组评审 ✓')
          break

        case 'develop': {
          // Phase6：返回 pipeline_id，刷新教案并切到关联课件Tab
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

      {/* ---- Tab内容卡 ---- */}
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, boxShadow: '0 1px 3px rgba(0,0,0,0.04)', overflow: 'hidden' }}>

        {/* Tab导航 */}
        <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}`, padding: '0 4px' }}>
          {TABS.map(tab => {
            const isActive = activeTab === tab.key
            // Tab标签动态增强
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
        {activeTab === 'content'    && <ContentTab plan={plan} />}
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

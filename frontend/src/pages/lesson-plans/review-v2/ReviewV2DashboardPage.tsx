/**
 * 多级审核工作台 — ReviewV2DashboardPage
 *
 * v127.3 优化：
 *   - 统计数据和列表分离加载：切换子视图只请求列表，不重复请求统计
 *   - admin 可审核任何教案（后端已放开权限）
 *   - 统计卡片可点击切换子视图
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getPendingReviews,
  getReviewStats,
  getReviewedRecords,
  reviewL1,
  reviewL2,
  type PendingReviewItem,
  type ReviewStatsResponse,
  type ReviewDecisionRequest,
  type ReviewedListItem,
  REVIEW_LEVEL_COLORS,
} from '@/api/review-v2'
import {
  getInspections,
  getInspectionStats,
  type InspectionListItem,
  type InspectionStatsResponse,
  INSPECTION_STATUS_CONFIG,
} from '@/api/inspection'

// ==================== 样式常量 ====================
const C = {
  primary:      '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  success:      '#10B981',
  warning:      '#F59E0B',
  danger:       '#EF4444',
  purple:       '#8B5CF6',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  bg:           '#FAFBFC',
  card:         '#FFFFFF',
  border:       '#F3F4F6',
}

type ReviewTab = 'l1' | 'l2' | 'l3'
type SubView = 'pending' | 'reviewed' | 'approved' | 'revision'

function formatDate(iso: string): string {
  try {
    const d = new Date(iso)
    return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`
  } catch { return iso }
}

const DECISION_LABELS: Record<string, { label: string; color: string; icon: string }> = {
  approved: { label: '通过', color: '#10B981', icon: '✅' },
  revision: { label: '退回', color: '#F59E0B', icon: '↩️' },
  revoked:  { label: '撤回', color: '#EF4444', icon: '🚫' },
}

// ==================== 审核决策弹窗 ====================
function ReviewDecisionModal({ planId, planTitle, level, onClose, onSubmit }: {
  planId: string; planTitle: string; level: number
  onClose: () => void
  onSubmit: (planId: string, level: number, req: ReviewDecisionRequest) => Promise<void>
}) {
  const [decision, setDecision] = useState<'approved' | 'revision'>('approved')
  const [comment, setComment] = useState('')
  const [score, setScore] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const handleSubmit = async () => {
    if (!comment.trim()) { alert('请填写审核意见'); return }
    setSubmitting(true)
    try {
      await onSubmit(planId, level, {
        decision,
        comment: comment.trim(),
        score: score ? parseFloat(score) : undefined,
      })
      onClose()
    } catch (e: unknown) {
      const msg = e instanceof Error ? e.message : '审核失败'
      alert(msg)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div style={{ position: 'fixed', inset: 0, zIndex: 9999, display: 'flex', alignItems: 'center', justifyContent: 'center', background: 'rgba(0,0,0,0.4)' }} onClick={onClose}>
      <div style={{ background: '#fff', borderRadius: '16px', width: '480px', maxHeight: '80vh', overflow: 'auto', boxShadow: '0 20px 60px rgba(0,0,0,0.2)' }} onClick={e => e.stopPropagation()}>
        <div style={{ padding: '24px 24px 16px', borderBottom: `1px solid ${C.border}` }}>
          <h3 style={{ margin: 0, fontSize: '18px', color: C.text }}>
            {level === 1 ? '📋 L1 教研组审核' : '🏫 L2 学校审核'}
          </h3>
          <p style={{ margin: '6px 0 0', fontSize: '13px', color: C.textSec, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
            {planTitle}
          </p>
        </div>
        <div style={{ padding: '20px 24px' }}>
          <div style={{ marginBottom: '16px' }}>
            <label style={{ fontSize: '13px', fontWeight: 600, color: C.text, display: 'block', marginBottom: '8px' }}>审核决策</label>
            <div style={{ display: 'flex', gap: '10px' }}>
              {[
                { value: 'approved' as const, label: '✅ 通过', color: C.success },
                { value: 'revision' as const, label: '↩️ 退回修改', color: C.warning },
              ].map(opt => (
                <button key={opt.value} onClick={() => setDecision(opt.value)}
                  style={{ flex: 1, padding: '10px', borderRadius: '10px', border: decision === opt.value ? `2px solid ${opt.color}` : `1px solid ${C.border}`, background: decision === opt.value ? opt.color + '10' : '#fff', cursor: 'pointer', fontSize: '14px', fontWeight: decision === opt.value ? 600 : 400, color: decision === opt.value ? opt.color : C.textSec, transition: 'all 150ms' }}>
                  {opt.label}
                </button>
              ))}
            </div>
          </div>
          <div style={{ marginBottom: '16px' }}>
            <label style={{ fontSize: '13px', fontWeight: 600, color: C.text, display: 'block', marginBottom: '6px' }}>评分（可选，1-10）</label>
            <input type="number" min="1" max="10" step="0.5" value={score} onChange={e => setScore(e.target.value)}
              placeholder="可选评分"
              style={{ width: '100%', padding: '10px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', boxSizing: 'border-box' }} />
          </div>
          <div style={{ marginBottom: '20px' }}>
            <label style={{ fontSize: '13px', fontWeight: 600, color: C.text, display: 'block', marginBottom: '6px' }}>审核意见 *</label>
            <textarea value={comment} onChange={e => setComment(e.target.value)}
              placeholder={decision === 'approved' ? '教案整体质量良好...' : '请说明需要修改的地方...'}
              rows={4}
              style={{ width: '100%', padding: '12px 14px', borderRadius: '10px', border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none', resize: 'vertical', boxSizing: 'border-box', fontFamily: 'inherit' }} />
          </div>
        </div>
        <div style={{ padding: '16px 24px', borderTop: `1px solid ${C.border}`, display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
          <button onClick={onClose} style={{ padding: '9px 20px', borderRadius: '8px', border: `1px solid ${C.border}`, background: '#fff', cursor: 'pointer', fontSize: '14px', color: C.textSec }}>取消</button>
          <button onClick={handleSubmit} disabled={submitting}
            style={{ padding: '9px 24px', borderRadius: '8px', border: 'none', background: decision === 'approved' ? C.success : C.warning, color: '#fff', cursor: submitting ? 'not-allowed' : 'pointer', fontSize: '14px', fontWeight: 600, opacity: submitting ? 0.7 : 1 }}>
            {submitting ? '提交中...' : (decision === 'approved' ? '✅ 确认通过' : '↩️ 确认退回')}
          </button>
        </div>
      </div>
    </div>
  )
}

// ==================== 主组件 ====================
export default function ReviewV2DashboardPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<ReviewTab>('l1')
  const [subView, setSubView] = useState<SubView>('pending')
  const [pendingItems, setPendingItems] = useState<PendingReviewItem[]>([])
  const [reviewedItems, setReviewedItems] = useState<ReviewedListItem[]>([])
  const [reviewStats, setReviewStats] = useState<ReviewStatsResponse | null>(null)
  const [inspectionItems, setInspectionItems] = useState<InspectionListItem[]>([])
  const [inspectionStats, setInspectionStats] = useState<InspectionStatsResponse | null>(null)
  const [loadingStats, setLoadingStats] = useState(false)
  const [loadingList, setLoadingList] = useState(false)
  const [reviewModal, setReviewModal] = useState<{ planId: string; planTitle: string; level: number } | null>(null)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  // 缓存：同一Tab下切换子视图不重新加载统计
  const statsTabRef = useRef<string>('')

  // 可用Tab
  const availableTabs: { key: ReviewTab; label: string; icon: string }[] = []
  if (user) {
    const r = user.role
    if (['admin', 'operator', 'viewer', 'senior_operator'].includes(r)) {
      availableTabs.push({ key: 'l1', label: 'L1 教研组审核', icon: '📋' })
    }
    if (['admin', 'senior_operator'].includes(r)) {
      availableTabs.push({ key: 'l2', label: 'L2 学校审核', icon: '🏫' })
    }
    if (['admin', 'district_inspector'].includes(r)) {
      availableTabs.push({ key: 'l3', label: 'L3 区域抽查', icon: '🔍' })
    }
  }

  useEffect(() => {
    if (availableTabs.length > 0 && !availableTabs.find(t => t.key === activeTab)) {
      setActiveTab(availableTabs[0].key)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.role])

  // 切换Tab时重置子视图
  useEffect(() => {
    setSubView('pending')
    statsTabRef.current = '' // 清除统计缓存，强制重新加载
  }, [activeTab])

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type }); setTimeout(() => setToast(null), 3000)
  }

  // 加载统计数据（仅Tab切换或审核操作后才调用）
  const loadStats = useCallback(async () => {
    const tabKey = activeTab
    if (tabKey === 'l1' || tabKey === 'l2') {
      const level = tabKey === 'l1' ? 1 : 2
      // 如果同Tab已有缓存，跳过
      if (statsTabRef.current === tabKey && reviewStats) return
      setLoadingStats(true)
      try {
        const stats = await getReviewStats(level)
        setReviewStats(stats)
        statsTabRef.current = tabKey
      } catch (e) {
        console.error('加载统计失败:', e)
      } finally {
        setLoadingStats(false)
      }
    } else if (tabKey === 'l3') {
      if (statsTabRef.current === tabKey && inspectionStats) return
      setLoadingStats(true)
      try {
        const stats = await getInspectionStats()
        setInspectionStats(stats)
        statsTabRef.current = tabKey
      } catch (e) {
        console.error('加载抽查统计失败:', e)
      } finally {
        setLoadingStats(false)
      }
    }
  }, [activeTab, reviewStats, inspectionStats])

  // 加载列表数据（Tab或子视图切换时调用）
  const loadList = useCallback(async () => {
    setLoadingList(true)
    try {
      if (activeTab === 'l1' || activeTab === 'l2') {
        const level = activeTab === 'l1' ? 1 : 2
        if (subView === 'pending') {
          const pending = await getPendingReviews({ limit: 100 })
          setPendingItems((pending?.items || []).filter(i => i.review_level === level))
          setReviewedItems([])
        } else {
          const decision = subView === 'approved' ? 'approved' : subView === 'revision' ? 'revision' : ''
          const reviewed = await getReviewedRecords({ level, decision, limit: 100 })
          setReviewedItems(reviewed?.items || [])
          setPendingItems([])
        }
      } else if (activeTab === 'l3') {
        const inspList = await getInspections({ limit: 100 })
        setInspectionItems(inspList?.items || [])
      }
    } catch (e) {
      console.error('加载列表失败:', e)
    } finally {
      setLoadingList(false)
    }
  }, [activeTab, subView])

  // Tab切换时：加载统计 + 列表
  useEffect(() => { loadStats() }, [loadStats])
  useEffect(() => { loadList() }, [loadList])

  // 审核操作后刷新统计+列表
  const handleReviewSubmit = async (planId: string, level: number, req: ReviewDecisionRequest) => {
    if (level === 1) {
      await reviewL1(planId, req)
    } else if (level === 2) {
      await reviewL2(planId, req)
    }
    showToast(req.decision === 'approved' ? '审核通过' : '已退回修改')
    // 审核后统计会变化，强制刷新
    statsTabRef.current = ''
    loadStats()
    loadList()
  }

  const loading = loadingStats || loadingList

  // 统计卡片配置
  const statsCards = reviewStats ? [
    { key: 'pending' as SubView, label: '待审核', value: reviewStats.total_pending, color: C.warning, icon: '📋' },
    { key: 'reviewed' as SubView, label: '已审核', value: reviewStats.total_reviewed, color: C.primary, icon: '📊' },
    { key: 'approved' as SubView, label: '已通过', value: reviewStats.total_approved, color: C.success, icon: '✅' },
    { key: 'revision' as SubView, label: '已退回', value: reviewStats.total_revision, color: C.danger, icon: '↩️' },
  ] : []

  // ==================== 渲染 ====================
  return (
    <div>
      <p style={{ fontSize: '14px', color: C.textSec, margin: '0 0 20px' }}>
        多级审核工作台 — 审核教师提交的教案，按级别分步审核
      </p>

      {/* 统计卡片（L1/L2可点击切换子视图） */}
      {(activeTab === 'l1' || activeTab === 'l2') && statsCards.length > 0 && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '12px', marginBottom: '20px' }}>
          {statsCards.map(s => {
            const isActive = subView === s.key
            return (
              <div key={s.key}
                onClick={() => setSubView(s.key)}
                style={{
                  padding: '16px 20px', borderRadius: '12px', cursor: 'pointer', transition: 'all 150ms',
                  background: isActive ? s.color + '18' : s.color + '08',
                  border: isActive ? `2px solid ${s.color}` : `1px solid ${s.color}20`,
                  transform: isActive ? 'scale(1.02)' : 'scale(1)',
                }}>
                <div style={{ fontSize: '16px', marginBottom: '4px' }}>{s.icon}</div>
                <div style={{ fontSize: '24px', fontWeight: 700, color: s.color }}>{loadingStats ? '—' : s.value}</div>
                <div style={{ fontSize: '12px', color: isActive ? s.color : C.textSec, marginTop: '4px', fontWeight: isActive ? 600 : 400 }}>{s.label}</div>
              </div>
            )
          })}
        </div>
      )}

      {activeTab === 'l3' && inspectionStats && (
        <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '12px', marginBottom: '20px' }}>
          {[
            { label: '总抽查', value: inspectionStats.total_sampled, color: C.purple, icon: '🔍' },
            { label: '待审查', value: inspectionStats.total_pending, color: C.warning, icon: '⏳' },
            { label: '已通过', value: inspectionStats.total_passed, color: C.success, icon: '✅' },
            { label: '已撤回', value: inspectionStats.total_revoked, color: C.danger, icon: '🚫' },
          ].map(s => (
            <div key={s.label} style={{ padding: '16px 20px', background: s.color + '10', borderRadius: '12px', border: `1px solid ${s.color}20` }}>
              <div style={{ fontSize: '16px', marginBottom: '4px' }}>{s.icon}</div>
              <div style={{ fontSize: '24px', fontWeight: 700, color: s.color }}>{loadingStats ? '—' : s.value}</div>
              <div style={{ fontSize: '12px', color: C.textSec, marginTop: '4px' }}>{s.label}</div>
            </div>
          ))}
        </div>
      )}

      {/* Tab栏 + 列表 */}
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
        <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}`, padding: '0 4px' }}>
          {availableTabs.map(tab => {
            const isActive = activeTab === tab.key
            const color = REVIEW_LEVEL_COLORS[tab.key === 'l1' ? 1 : tab.key === 'l2' ? 2 : 3]
            return (
              <button key={tab.key} onClick={() => setActiveTab(tab.key)}
                style={{ padding: '14px 20px', border: 'none', background: 'transparent', fontSize: '14px', fontWeight: isActive ? 600 : 400, color: isActive ? color : C.textSec, cursor: 'pointer', borderBottom: isActive ? `2px solid ${color}` : '2px solid transparent', marginBottom: '-1px', transition: 'all 150ms ease' }}>
                {tab.icon} {tab.label}
              </button>
            )
          })}
        </div>

        {/* L1/L2 — 待审核列表 */}
        {(activeTab === 'l1' || activeTab === 'l2') && subView === 'pending' && (
          <div>
            {loadingList && <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div>}
            {!loadingList && pendingItems.length === 0 && (
              <div style={{ padding: '60px 40px', textAlign: 'center', color: C.textMuted }}>
                <div style={{ fontSize: '40px', marginBottom: '12px' }}>🎉</div>
                <div style={{ fontSize: '15px', fontWeight: 600, color: C.textSec }}>暂无待审核教案</div>
              </div>
            )}
            {!loadingList && pendingItems.map(item => (
              <div key={item.lesson_plan_id} style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}`, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px' }}>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '6px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                    {item.title}
                  </div>
                  <div style={{ display: 'flex', gap: '14px', flexWrap: 'wrap', fontSize: '13px', color: C.textSec }}>
                    <span>📚 {item.subject}</span>
                    <span>🎓 {item.grade}</span>
                    <span>✍️ {item.author_name}</span>
                    {item.group_name && <span>👥 {item.group_name}</span>}
                    {item.school_name && <span>🏫 {item.school_name}</span>}
                    {item.ai_review_score != null && (
                      <span style={{ color: item.ai_review_score >= 8.5 ? C.success : C.warning, fontWeight: 600 }}>
                        🤖 {item.ai_review_score.toFixed(1)}
                      </span>
                    )}
                    <span style={{ color: C.textMuted }}>提交于 {formatDate(item.submitted_at)}</span>
                  </div>
                </div>
                <div style={{ display: 'flex', gap: '8px', flexShrink: 0 }}>
                  <button onClick={() => navigate(`/lesson-plans/plans/${item.lesson_plan_id}`)}
                    style={{ padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: '#fff', cursor: 'pointer', fontSize: '13px', color: C.textSec }}>
                    查看
                  </button>
                  <button onClick={() => setReviewModal({ planId: item.lesson_plan_id, planTitle: item.title, level: activeTab === 'l1' ? 1 : 2 })}
                    style={{ padding: '8px 20px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', cursor: 'pointer', fontSize: '13px', fontWeight: 600 }}>
                    审核 →
                  </button>
                </div>
              </div>
            ))}
          </div>
        )}

        {/* L1/L2 — 已审核记录列表 */}
        {(activeTab === 'l1' || activeTab === 'l2') && subView !== 'pending' && (
          <div>
            {loadingList && <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div>}
            {!loadingList && reviewedItems.length === 0 && (
              <div style={{ padding: '60px 40px', textAlign: 'center', color: C.textMuted }}>
                <div style={{ fontSize: '40px', marginBottom: '12px' }}>📋</div>
                <div style={{ fontSize: '15px', fontWeight: 600, color: C.textSec }}>
                  暂无{subView === 'approved' ? '已通过' : subView === 'revision' ? '已退回' : '已审核'}记录
                </div>
              </div>
            )}
            {!loadingList && reviewedItems.map(item => {
              const dCfg = DECISION_LABELS[item.decision] || DECISION_LABELS.approved
              return (
                <div key={item.id} style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}`, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px' }}>
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '6px' }}>
                      <span style={{ fontSize: '15px', fontWeight: 600, color: C.text, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {item.plan_title}
                      </span>
                      <span style={{ padding: '2px 8px', borderRadius: '10px', background: dCfg.color + '15', color: dCfg.color, fontSize: '11px', fontWeight: 600, flexShrink: 0 }}>
                        {dCfg.icon} {dCfg.label}
                      </span>
                    </div>
                    <div style={{ display: 'flex', gap: '14px', flexWrap: 'wrap', fontSize: '13px', color: C.textSec }}>
                      <span>📚 {item.plan_subject}</span>
                      <span>🎓 {item.plan_grade}</span>
                      <span>✍️ {item.author_name}</span>
                      <span>🔍 {item.reviewer_name}</span>
                      {item.score != null && (
                        <span style={{ color: C.primary, fontWeight: 600 }}>⭐ {item.score.toFixed(1)}</span>
                      )}
                      <span style={{ color: C.textMuted }}>{formatDate(item.created_at)}</span>
                    </div>
                    {item.comment && (
                      <div style={{ marginTop: '6px', fontSize: '13px', color: C.textSec, lineHeight: '1.5', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '600px' }}>
                        💬 {item.comment}
                      </div>
                    )}
                  </div>
                  <button onClick={() => navigate(`/lesson-plans/plans/${item.lesson_plan_id}`)}
                    style={{ padding: '8px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: '#fff', cursor: 'pointer', fontSize: '13px', color: C.textSec, flexShrink: 0 }}>
                    查看教案
                  </button>
                </div>
              )
            })}
          </div>
        )}

        {/* L3 抽查列表 */}
        {activeTab === 'l3' && (
          <div>
            {loadingList && <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div>}
            {!loadingList && inspectionItems.length === 0 && (
              <div style={{ padding: '60px 40px', textAlign: 'center', color: C.textMuted }}>
                <div style={{ fontSize: '40px', marginBottom: '12px' }}>🔍</div>
                <div style={{ fontSize: '15px', fontWeight: 600, color: C.textSec }}>暂无抽查记录</div>
                <p style={{ fontSize: '13px', marginTop: '8px' }}>管理员可在后台触发抽样</p>
              </div>
            )}
            {!loadingList && inspectionItems.map(item => {
              const statusCfg = INSPECTION_STATUS_CONFIG[item.status] || INSPECTION_STATUS_CONFIG.pending
              return (
                <div key={item.id} style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}` }}>
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px' }}>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>{item.plan_title}</div>
                      <div style={{ display: 'flex', gap: '14px', flexWrap: 'wrap', fontSize: '13px', color: C.textSec }}>
                        <span>📚 {item.plan_subject}</span>
                        <span>🎓 {item.plan_grade}</span>
                        <span>✍️ {item.author_name}</span>
                        <span>🏫 {item.school_name}</span>
                        {item.inspector_name && <span>🔍 {item.inspector_name}</span>}
                        <span style={{ color: C.textMuted }}>{formatDate(item.created_at)}</span>
                      </div>
                    </div>
                    <span style={{ padding: '4px 12px', borderRadius: '20px', background: statusCfg.bg, color: statusCfg.color, fontSize: '12px', fontWeight: 600, flexShrink: 0 }}>
                      {statusCfg.icon} {statusCfg.label}
                    </span>
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>

      {/* 审核决策弹窗 */}
      {reviewModal && (
        <ReviewDecisionModal
          planId={reviewModal.planId}
          planTitle={reviewModal.planTitle}
          level={reviewModal.level}
          onClose={() => setReviewModal(null)}
          onSubmit={handleReviewSubmit}
        />
      )}

      {/* Toast */}
      {toast && (
        <div style={{ position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)', padding: '12px 24px', borderRadius: '10px', background: toast.type === 'error' ? '#FEF2F2' : '#1F2937', color: toast.type === 'error' ? C.danger : '#fff', fontSize: '14px', fontWeight: 500, boxShadow: '0 8px 24px rgba(0,0,0,0.15)', zIndex: 9999 }}>
          {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
        </div>
      )}
    </div>
  )
}

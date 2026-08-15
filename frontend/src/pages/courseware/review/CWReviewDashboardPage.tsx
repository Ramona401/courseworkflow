/**
 * 课件审核中心 — CWReviewDashboardPage.tsx（阶段3）
 *
 * 让已上线的 8 个课件审核 API 真正可用，是阶段3 真正交付的最后一公里。
 * 镜像教案审核工作台，但有三处差异：
 *   1. 审核主键是 courseware_id（教案是 plan_id）。
 *   2. 课件【无 L3 区域抽查】，只有 L1/L2 两 Tab。
 *   3. 审核台改为【独立全屏页面】CWReviewWorkbenchPage（左大预览 + 右批注/历史 + 决策），
 *      不再用弹窗——弹窗里 iframe 未等比缩放课件被截断看不全无法审核。
 *      点「审核 →」navigate 到 /courseware/review/{id}?level=N，与教案审核跳详情页范式一致。
 *
 * R-03：
 *   - 待审核记录仍进入当前审核工作台；
 *   - 已审核记录使用review_id进入独立只读历史详情；
 *   - 已审核记录绝不重新进入作者工坊或当前审核工作台。
 *
 * 角色可见 Tab（与后端 GetPendingReviews 角色分流一致）：
 *   - operator/viewer            → L1
 *   - senior_operator            → L1 + L2
 *   - admin                      → L1 + L2（后端 L1All 全量）
 *
 * 颜色用课件工坊暖色系（橙→红），与教案审核蓝紫系区分。
 * 挂载在 CWLayout 下的 /courseware/review 路由。本文件 < 600 行。
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getCWPendingReviews,
  getCWReviewStats,
  getCWReviewedRecords,
  CW_REVIEW_LEVEL_COLORS,
  type CWPendingReviewItem,
  type CWReviewStatsResponse,
  type CWReviewedListItem,
} from '@/api/coursewares'

// ==================== 样式常量（暖色系） ====================
const C = {
  primary:   '#F59E0B',
  danger:    '#EF4444',
  success:   '#10B981',
  warning:   '#F59E0B',
  blue:      '#2563EB',
  text:      '#1F2937',
  textSec:   '#6B7280',
  textMuted: '#9CA3AF',
  border:    '#F3F4F6',
  borderMid: '#E5E7EB',
  card:      '#FFFFFF',
}

type ReviewTab = 'l1' | 'l2'
type SubView = 'pending' | 'reviewed' | 'approved' | 'revision'

const DECISION_LABELS: Record<string, { label: string; color: string; icon: string }> = {
  approved: { label: '通过', color: '#10B981', icon: '✅' },
  revision: { label: '退回', color: '#F59E0B', icon: '↩️' },
  revoked:  { label: '撤回', color: '#EF4444', icon: '🚫' },
}

function formatDate(iso: string): string {
  try {
    const d = new Date(iso)
    const p = (n: number) => String(n).padStart(2, '0')
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
  } catch { return iso }
}

// ==================== 主组件 ====================
export default function CWReviewDashboardPage() {
  const { user } = useAuth()
  const navigate = useNavigate()
  const [activeTab, setActiveTab] = useState<ReviewTab>('l1')
  const [subView, setSubView] = useState<SubView>('pending')
  const [pendingItems, setPendingItems] = useState<CWPendingReviewItem[]>([])
  const [reviewedItems, setReviewedItems] = useState<CWReviewedListItem[]>([])
  const [reviewStats, setReviewStats] = useState<CWReviewStatsResponse | null>(null)
  const [loadingStats, setLoadingStats] = useState(false)
  const [loadingList, setLoadingList] = useState(false)

  // 同一 Tab 下切换子视图不重复请求统计的缓存键
  const statsTabRef = useRef<string>('')

  // —— 角色可见 Tab ——
  const availableTabs: { key: ReviewTab; label: string; icon: string }[] = []
  if (user) {
    const r = user.role
    if (['admin', 'operator', 'viewer', 'senior_operator'].includes(r)) {
      availableTabs.push({ key: 'l1', label: 'L1 教研组审核', icon: '📋' })
    }
    if (['admin', 'senior_operator'].includes(r)) {
      availableTabs.push({ key: 'l2', label: 'L2 学校审核', icon: '🏫' })
    }
  }

  // 当前 activeTab 不在可见 Tab 里时纠正到第一个
  useEffect(() => {
    if (availableTabs.length > 0 && !availableTabs.find(t => t.key === activeTab)) {
      setActiveTab(availableTabs[0].key)
    }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [user?.role])

  // 切换 Tab 时重置子视图 + 清统计缓存
  useEffect(() => {
    setSubView('pending')
    statsTabRef.current = ''
  }, [activeTab])

  // —— 加载统计（仅 Tab 切换时调用） ——
  const loadStats = useCallback(async () => {
    const level = activeTab === 'l1' ? 1 : 2
    if (statsTabRef.current === activeTab && reviewStats) return
    setLoadingStats(true)
    try {
      const stats = await getCWReviewStats(level)
      setReviewStats(stats)
      statsTabRef.current = activeTab
    } catch (e) {
      console.error('加载课件审核统计失败:', e)
    } finally {
      setLoadingStats(false)
    }
  }, [activeTab, reviewStats])

  // —— 加载列表（Tab 或子视图切换时调用） ——
  const loadList = useCallback(async () => {
    setLoadingList(true)
    try {
      const level = activeTab === 'l1' ? 1 : 2
      if (subView === 'pending') {
        const pending = await getCWPendingReviews({ limit: 100 })
        setPendingItems((pending?.items || []).filter(i => i.review_level === level))
        setReviewedItems([])
      } else {
        const decision = subView === 'approved' ? 'approved' : subView === 'revision' ? 'revision' : ''
        const reviewed = await getCWReviewedRecords({ level, decision, limit: 100 })
        setReviewedItems(reviewed?.items || [])
        setPendingItems([])
      }
    } catch (e) {
      console.error('加载课件审核列表失败:', e)
    } finally {
      setLoadingList(false)
    }
  }, [activeTab, subView])

  useEffect(() => { loadStats() }, [loadStats])
  useEffect(() => { loadList() }, [loadList])

  // —— 跳转到当前待审核工作台 ——
  const gotoWorkbench = (coursewareId: string) => {
    const level = activeTab === 'l1' ? 1 : 2
    navigate(`/courseware/review/${coursewareId}?level=${level}`)
  }

  // —— R-03：已审核记录只能进入review_id只读历史详情 ——
  const gotoHistory = (reviewId: string) => {
    navigate(`/courseware/review-history/${reviewId}`)
  }

  const statsCards = reviewStats ? [
    { key: 'pending' as SubView, label: '待审核', value: reviewStats.total_pending, color: C.warning, icon: '📋' },
    { key: 'reviewed' as SubView, label: '已审核', value: reviewStats.total_reviewed, color: C.blue, icon: '📊' },
    { key: 'approved' as SubView, label: '已通过', value: reviewStats.total_approved, color: C.success, icon: '✅' },
    { key: 'revision' as SubView, label: '已退回', value: reviewStats.total_revision, color: C.danger, icon: '↩️' },
  ] : []

  return (
    <div style={{ padding: '24px 28px', maxWidth: '1200px', margin: '0 auto' }}>
      <div style={{ marginBottom: '20px' }}>
        <h2 style={{ margin: 0, fontSize: '22px', fontWeight: 700, color: C.text }}>🛡️ 课件审核中心</h2>
        <p style={{ fontSize: '14px', color: C.textSec, margin: '6px 0 0' }}>
          审核教师提交的课件 — 按级别分步审核，边看课件与批注边决策
        </p>
      </div>

      {availableTabs.length === 0 && (
        <div style={{ padding: '60px 40px', textAlign: 'center', color: C.textMuted, background: C.card, borderRadius: '12px', border: `1px solid ${C.border}` }}>
          <div style={{ fontSize: '40px', marginBottom: '12px' }}>🔒</div>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.textSec }}>当前角色暂无课件审核权限</div>
        </div>
      )}

      {availableTabs.length > 0 && (
        <>
          {statsCards.length > 0 && (
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

          <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, overflow: 'hidden' }}>
            <div style={{ display: 'flex', borderBottom: `1px solid ${C.border}`, padding: '0 4px' }}>
              {availableTabs.map(tab => {
                const isActive = activeTab === tab.key
                const color = CW_REVIEW_LEVEL_COLORS[tab.key === 'l1' ? 1 : 2]
                return (
                  <button key={tab.key} onClick={() => setActiveTab(tab.key)}
                    style={{ padding: '14px 20px', border: 'none', background: 'transparent', fontSize: '14px', fontWeight: isActive ? 600 : 400, color: isActive ? color : C.textSec, cursor: 'pointer', borderBottom: isActive ? `2px solid ${color}` : '2px solid transparent', marginBottom: '-1px', transition: 'all 150ms ease' }}>
                    {tab.icon} {tab.label}
                  </button>
                )
              })}
            </div>

            {subView === 'pending' && (
              <div>
                {loadingList && <div style={{ padding: '40px', textAlign: 'center', color: C.textMuted }}>加载中...</div>}
                {!loadingList && pendingItems.length === 0 && (
                  <div style={{ padding: '60px 40px', textAlign: 'center', color: C.textMuted }}>
                    <div style={{ fontSize: '40px', marginBottom: '12px' }}>🎉</div>
                    <div style={{ fontSize: '15px', fontWeight: 600, color: C.textSec }}>暂无待审核课件</div>
                  </div>
                )}
                {!loadingList && pendingItems.map(item => (
                  <div key={item.courseware_id} style={{ padding: '18px 24px', borderBottom: `1px solid ${C.border}`, display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', gap: '16px' }}>
                    <div style={{ flex: 1, minWidth: 0 }}>
                      <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '6px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {item.title}
                      </div>
                      <div style={{ display: 'flex', gap: '14px', flexWrap: 'wrap', fontSize: '13px', color: C.textSec }}>
                        {item.subject && <span>📚 {item.subject}</span>}
                        {item.grade && <span>🎓 {item.grade}</span>}
                        <span>📄 {item.page_count} 页</span>
                        <span>✍️ {item.author_name}</span>
                        {item.school_name && <span>🏫 {item.school_name}</span>}
                        <span style={{ color: C.textMuted }}>提交于 {formatDate(item.submitted_at)}</span>
                      </div>
                    </div>
                    <div style={{ display: 'flex', gap: '8px', flexShrink: 0 }}>
                      <button onClick={() => gotoWorkbench(item.courseware_id)}
                        style={{ padding: '8px 20px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', cursor: 'pointer', fontSize: '13px', fontWeight: 600 }}>
                        审核 →
                      </button>
                    </div>
                  </div>
                ))}
              </div>
            )}

            {subView !== 'pending' && (
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
                            {item.courseware_title}
                          </span>
                          <span style={{ padding: '2px 8px', borderRadius: '10px', background: dCfg.color + '15', color: dCfg.color, fontSize: '11px', fontWeight: 600, flexShrink: 0 }}>
                            {dCfg.icon} {dCfg.label}
                          </span>
                        </div>
                        <div style={{ display: 'flex', gap: '14px', flexWrap: 'wrap', fontSize: '13px', color: C.textSec }}>
                          {item.subject && <span>📚 {item.subject}</span>}
                          {item.grade && <span>🎓 {item.grade}</span>}
                          <span>✍️ {item.author_name}</span>
                          <span>🔍 {item.reviewer_name}</span>
                          {item.score != null && <span style={{ color: C.primary, fontWeight: 600 }}>⭐ {item.score.toFixed(1)}</span>}
                          <span style={{ color: C.textMuted }}>{formatDate(item.created_at)}</span>
                        </div>
                        {item.comment && (
                          <div style={{ marginTop: '6px', fontSize: '13px', color: C.textSec, lineHeight: '1.5', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', maxWidth: '600px' }}>
                            💬 {item.comment}
                          </div>
                        )}
                      </div>
                      <button
                        type="button"
                        onClick={() => gotoHistory(item.id)}
                        style={{
                          padding: '8px 16px',
                          borderRadius: '8px',
                          border: `1px solid ${C.blue}45`,
                          background: `${C.blue}0D`,
                          cursor: 'pointer',
                          fontSize: '13px',
                          color: C.blue,
                          fontWeight: 600,
                          flexShrink: 0,
                        }}
                      >
                        查看审核记录 →
                      </button>
                    </div>
                  )
                })}
              </div>
            )}
          </div>
        </>
      )}
    </div>
  )
}

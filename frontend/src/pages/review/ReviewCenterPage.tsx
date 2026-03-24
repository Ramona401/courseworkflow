/**
 * 审核中心页面（Phase 6 - P6-1）
 *
 * 核心功能：
 * 1. 集中展示所有待审核Pipeline（review_queue + needs_human状态）
 * 2. 按优先级排序：needs_human优先 → 2审优先 → 创建时间正序
 * 3. 审核统计卡片：待审核/今日已审核/总已定稿/验收通过
 * 4. 快捷操作：一键进入审核、快捷通过
 * 5. 已完成审核记录（finalized/verified，最近20条）
 * 6. 审核轮次标识（1审/2审徽章）
 *
 * Apple风格内联CSS，无Tailwind运行时依赖
 */
import { useState, useEffect, useCallback } from 'react'
import { useNavigate } from 'react-router-dom'
import { useAuth } from '@/store/auth'
import {
  getPipelines, markPassed,
  type PipelineListItem,
} from '@/api/pipelines'
import {
  ClipboardCheck, RefreshCw, Zap, AlertTriangle,
  CheckCircle, ArrowRight, Loader, ShieldCheck,
  FileText, XCircle,
} from 'lucide-react'

// ==================== Toast组件 ====================
function Toast({ message, type, onClose }: { message: string; type: 'ok' | 'err' | 'info'; onClose: () => void }) {
  useEffect(() => { const t = setTimeout(onClose, 5000); return () => clearTimeout(t) }, [onClose])
  const bg = type === 'ok' ? '#34c759' : type === 'err' ? '#ff3b30' : '#007aff'
  return (
    <div style={{
      position: 'fixed', bottom: 24, right: 24, background: bg, color: '#fff',
      padding: '12px 22px', borderRadius: 12, fontSize: 13, fontWeight: 500,
      zIndex: 9999, boxShadow: '0 4px 24px rgba(0,0,0,0.18)', maxWidth: 500,
    }}>
      {message}
    </div>
  )
}

// ==================== 统计卡片组件 ====================
function StatCard({ label, value, icon, color, sub }: {
  label: string; value: string | number; icon: React.ReactNode; color: string; sub?: string
}) {
  return (
    <div style={{
      background: '#fff', borderRadius: 16, padding: '20px 22px',
      border: '1px solid rgba(0,0,0,0.04)', boxShadow: '0 1px 3px rgba(0,0,0,0.04)',
      flex: 1, minWidth: 140, transition: 'all 0.2s ease', cursor: 'default',
    }}
    onMouseEnter={e => {
      (e.currentTarget as HTMLElement).style.transform = 'translateY(-2px)';
      (e.currentTarget as HTMLElement).style.boxShadow = '0 8px 25px rgba(0,0,0,0.08)'
    }}
    onMouseLeave={e => {
      (e.currentTarget as HTMLElement).style.transform = 'translateY(0)';
      (e.currentTarget as HTMLElement).style.boxShadow = '0 1px 3px rgba(0,0,0,0.04)'
    }}>
      <div style={{
        width: 36, height: 36, background: color, borderRadius: 10,
        display: 'flex', alignItems: 'center', justifyContent: 'center',
        marginBottom: 12,
      }}>{icon}</div>
      <div style={{ fontSize: 11, color: '#86868b', fontWeight: 600, marginBottom: 2 }}>{label}</div>
      <div style={{ fontSize: 28, fontWeight: 700, color: '#1d1d1f', letterSpacing: '-0.5px' }}>{value}</div>
      {sub && <div style={{ fontSize: 11, color: '#86868b', marginTop: 2 }}>{sub}</div>}
    </div>
  )
}

// ==================== 状态徽章 ====================
function StatusBadge({ status }: { status: string }) {
  const map: Record<string, { bg: string; fg: string; label: string }> = {
    review_queue:  { bg: 'rgba(255,149,0,0.12)', fg: '#ff9500', label: '待审核' },
    needs_human:   { bg: 'rgba(255,204,0,0.15)', fg: '#cc9900', label: '需人工' },
    finalized:     { bg: 'rgba(52,199,89,0.12)', fg: '#34c759', label: '已定稿' },
    verified:      { bg: 'rgba(52,199,89,0.15)', fg: '#248a3d', label: '验收通过' },
    verify_failed: { bg: 'rgba(255,59,48,0.15)', fg: '#d70015', label: '验收未通过' },
  }
  const c = map[status] || { bg: 'rgba(142,142,147,0.1)', fg: '#8e8e93', label: status }
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: 4,
      padding: '3px 10px', borderRadius: 20, fontSize: 11, fontWeight: 600,
      background: c.bg, color: c.fg, whiteSpace: 'nowrap',
    }}>{c.label}</span>
  )
}

// ==================== 审核轮次徽章 ====================
function RoundBadge({ round }: { round: number }) {
  if (round <= 1) return null
  return (
    <span style={{
      display: 'inline-flex', alignItems: 'center', gap: 3,
      padding: '2px 8px', borderRadius: 10, fontSize: 10, fontWeight: 700,
      background: 'rgba(255,59,48,0.1)', color: '#ff3b30',
      letterSpacing: '0.02em',
    }}>
      {round}审
    </span>
  )
}

// ==================== 分数显示 ====================
function ScoreDisplay({ value, label }: { value: number | null; label: string }) {
  if (value === null || value === undefined) {
    return <span style={{ fontSize: 11, color: '#c7c7cc' }}>{label}: -</span>
  }
  let color = '#ff3b30'
  if (value >= 9.0) color = '#34c759'
  else if (value >= 7.0) color = '#ff9500'
  return (
    <span style={{ fontSize: 11, color: '#8e8e93' }}>
      {label}: <span style={{ color, fontWeight: 600, fontVariantNumeric: 'tabular-nums' }}>{value.toFixed(1)}</span>
    </span>
  )
}

// ==================== 判断是否可快捷通过 ====================
function canMarkPassed(p: PipelineListItem): boolean {
  const allowedStatuses = ['review_queue', 'needs_human', 'failed']
  return allowedStatuses.includes(p.status) && p.meta_score !== null && p.meta_score >= 9.0
}

// ==================== 格式化时间 ====================
function formatTime(t: string | null): string {
  if (!t) return '-'
  const d = new Date(t)
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  if (diff < 60000) return '刚刚'
  if (diff < 3600000) return Math.floor(diff / 60000) + '分钟前'
  if (diff < 86400000) return Math.floor(diff / 3600000) + '小时前'
  return d.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

// ==================== 判断是否为今天 ====================
function isToday(t: string | null): boolean {
  if (!t) return false
  const d = new Date(t)
  const now = new Date()
  return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth() && d.getDate() === now.getDate()
}

// ==================== 主页面组件 ====================

export default function ReviewCenterPage() {
  const navigate = useNavigate()
  const { user } = useAuth()
  const canOperate = user?.role === 'admin' || user?.role === 'operator'

  const [pipelines, setPipelines] = useState<PipelineListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [toast, setToast] = useState<{ message: string; type: 'ok' | 'err' | 'info' } | null>(null)
  // 已完成区域的折叠状态
  const [showCompleted, setShowCompleted] = useState(true)

  /** 加载所有Pipeline数据 */
  const loadData = useCallback(async () => {
    setLoading(true)
    try {
      const data = await getPipelines()
      setPipelines(data.pipelines || [])
    } catch (e: any) {
      setToast({ message: '加载失败: ' + (e.message || ''), type: 'err' })
    }
    setLoading(false)
  }, [])

  useEffect(() => { loadData() }, [loadData])

  // ===== 数据分组与排序 =====

  // 待审核队列：review_queue + needs_human
  // 排序：needs_human优先 → 2审优先 → 创建时间正序（先创建先审核）
  const pendingReview = pipelines
    .filter(p => p.status === 'review_queue' || p.status === 'needs_human')
    .sort((a, b) => {
      // needs_human优先
      if (a.status === 'needs_human' && b.status !== 'needs_human') return -1
      if (a.status !== 'needs_human' && b.status === 'needs_human') return 1
      // 2审优先
      if ((a.review_round || 1) > (b.review_round || 1)) return -1
      if ((a.review_round || 1) < (b.review_round || 1)) return 1
      // 创建时间正序（先创建先审核）
      const ta = a.created_at ? new Date(a.created_at).getTime() : 0
      const tb = b.created_at ? new Date(b.created_at).getTime() : 0
      return ta - tb
    })

  // 已完成审核：finalized + verified + verify_failed（按完成时间倒序，最近20条）
  const completedReview = pipelines
    .filter(p => p.status === 'finalized' || p.status === 'verified' || p.status === 'verify_failed')
    .sort((a, b) => {
      const ta = a.completed_at ? new Date(a.completed_at).getTime() : (a.created_at ? new Date(a.created_at).getTime() : 0)
      const tb = b.completed_at ? new Date(b.completed_at).getTime() : (b.created_at ? new Date(b.created_at).getTime() : 0)
      return tb - ta
    })
    .slice(0, 20)

  // ===== 统计数据 =====
  const totalPending = pendingReview.length
  const needsHumanCount = pendingReview.filter(p => p.status === 'needs_human').length
  const todayFinalized = pipelines.filter(p =>
    (p.status === 'finalized' || p.status === 'verified') && isToday(p.completed_at)
  ).length
  const totalFinalized = pipelines.filter(p => p.status === 'finalized' || p.status === 'verified').length
  const verifiedCount = pipelines.filter(p => p.status === 'verified').length

  /** 快捷通过 */
  const handleMarkPassed = async (p: PipelineListItem) => {
    if (!confirm('确认快捷通过 ' + p.course_code + '？\n将跳过审核流程直接标记为已定稿。')) return
    try {
      await markPassed(p.id)
      setToast({ message: p.course_code + ' 已快捷通过并归档', type: 'ok' })
      loadData()
    } catch (e: any) {
      setToast({ message: '快捷通过失败: ' + (e.message || ''), type: 'err' })
    }
  }

  // ===== 样式常量 =====
  const btn: React.CSSProperties = {
    padding: '7px 14px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer',
    display: 'inline-flex', alignItems: 'center', gap: 5, transition: 'all 0.15s ease',
  }

  return (
    <div>
      {/* ===== 页面标题区 ===== */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 24 }}>
        <div>
          <h1 style={{ fontSize: 24, fontWeight: 700, color: '#1d1d1f', margin: 0, letterSpacing: '-0.3px' }}>
            审核中心
          </h1>
          <p style={{ fontSize: 14, color: '#86868b', margin: '4px 0 0 0' }}>
            集中管理所有待审核的Pipeline课程
          </p>
        </div>
        <button style={btn} onClick={loadData}>
          <RefreshCw size={14} /> 刷新
        </button>
      </div>

      {/* ===== 统计卡片 ===== */}
      <div style={{ display: 'flex', gap: 12, marginBottom: 24, flexWrap: 'wrap' }}>
        <StatCard
          label="待审核" value={totalPending}
          icon={<ClipboardCheck size={18} color="#fff" />}
          color="linear-gradient(135deg, #ff9500, #ffcc00)"
          sub={needsHumanCount > 0 ? needsHumanCount + ' 个需人工干预' : undefined}
        />
        <StatCard
          label="今日已审核" value={todayFinalized}
          icon={<CheckCircle size={18} color="#fff" />}
          color="linear-gradient(135deg, #34c759, #30d158)"
        />
        <StatCard
          label="总已定稿" value={totalFinalized}
          icon={<FileText size={18} color="#fff" />}
          color="linear-gradient(135deg, #007aff, #5ac8fa)"
        />
        <StatCard
          label="验收通过" value={verifiedCount}
          icon={<ShieldCheck size={18} color="#fff" />}
          color="linear-gradient(135deg, #32ade6, #5856d6)"
        />
      </div>

      {/* ===== 加载状态 ===== */}
      {loading && (
        <div style={{ textAlign: 'center', padding: 40, color: '#8e8e93' }}>
          <Loader size={24} style={{ animation: 'spin 1s linear infinite', marginBottom: 8 }} />
          <div style={{ fontSize: 14 }}>加载中...</div>
        </div>
      )}

      {/* ===== 待审核队列 ===== */}
      {!loading && (
        <div style={{
          background: '#fff', borderRadius: 16, border: '1px solid rgba(0,0,0,0.04)',
          boxShadow: '0 1px 3px rgba(0,0,0,0.04)', marginBottom: 24, overflow: 'hidden',
        }}>
          {/* 区块标题 */}
          <div style={{
            display: 'flex', alignItems: 'center', gap: 10,
            padding: '16px 20px', borderBottom: '1px solid rgba(0,0,0,0.04)',
            background: 'rgba(255,149,0,0.03)',
          }}>
            <ClipboardCheck size={18} color="#ff9500" />
            <span style={{ fontSize: 15, fontWeight: 700, color: '#1c1c1e' }}>待审核队列</span>
            <span style={{
              fontSize: 12, fontWeight: 600, color: '#ff9500',
              background: 'rgba(255,149,0,0.12)', padding: '2px 10px', borderRadius: 10,
            }}>{totalPending}</span>
            {needsHumanCount > 0 && (
              <span style={{
                fontSize: 11, fontWeight: 600, color: '#cc9900',
                background: 'rgba(255,204,0,0.15)', padding: '2px 10px', borderRadius: 10,
                display: 'inline-flex', alignItems: 'center', gap: 4,
              }}>
                <AlertTriangle size={11} /> {needsHumanCount} 需人工
              </span>
            )}
          </div>

          {/* 待审核列表 */}
          {pendingReview.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '40px 20px', color: '#aeaeb2' }}>
              <CheckCircle size={36} style={{ marginBottom: 8, opacity: 0.4 }} />
              <div style={{ fontSize: 14, fontWeight: 500 }}>所有课程审核已完成</div>
              <div style={{ fontSize: 12, marginTop: 4 }}>当前没有待审核的Pipeline</div>
            </div>
          ) : (
            <div>
              {pendingReview.map((p, idx) => (
                <div
                  key={p.id}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 14,
                    padding: '14px 20px',
                    borderBottom: idx < pendingReview.length - 1 ? '1px solid rgba(0,0,0,0.03)' : 'none',
                    transition: 'background 0.15s ease', cursor: 'pointer',
                  }}
                  onClick={() => navigate('/pipelines/' + p.id + '/review')}
                  onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.03)' }}
                  onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
                >
                  {/* 优先级指示器 */}
                  <div style={{
                    width: 4, height: 40, borderRadius: 2, flexShrink: 0,
                    background: p.status === 'needs_human' ? '#cc9900'
                      : (p.review_round || 1) > 1 ? '#ff3b30' : '#ff9500',
                  }} />

                  {/* 课程信息 */}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8, marginBottom: 4 }}>
                      <span style={{ fontSize: 14, fontWeight: 700, color: '#1c1c1e' }}>{p.course_code}</span>
                      <StatusBadge status={p.status} />
                      <RoundBadge round={p.review_round || 1} />
                    </div>
                    <div style={{
                      fontSize: 12, color: '#86868b',
                      overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                    }}>
                      {p.course_name || p.course_code}
                    </div>
                  </div>

                  {/* 分数区 */}
                  <div style={{ display: 'flex', gap: 14, flexShrink: 0 }}>
                    <ScoreDisplay value={p.eval_avg_score} label="评估" />
                    <ScoreDisplay value={p.meta_score} label="Meta" />
                    <ScoreDisplay value={p.translator_score} label="翻译" />
                  </div>

                  {/* 创建时间 */}
                  <div style={{ fontSize: 11, color: '#aeaeb2', flexShrink: 0, minWidth: 60, textAlign: 'right' }}>
                    {formatTime(p.created_at)}
                  </div>

                  {/* 操作按钮区 */}
                  <div style={{ display: 'flex', gap: 6, flexShrink: 0 }} onClick={e => e.stopPropagation()}>
                    {canMarkPassed(p) && canOperate && (
                      <button
                        title="快捷通过（Meta>=9.0）"
                        onClick={() => handleMarkPassed(p)}
                        style={{
                          width: 30, height: 30, borderRadius: 8,
                          border: '1px solid rgba(52,199,89,0.3)', background: 'rgba(52,199,89,0.08)',
                          cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
                          color: '#34c759', padding: 0, transition: 'all 0.15s ease',
                        }}
                        onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(52,199,89,0.15)' }}
                        onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(52,199,89,0.08)' }}
                      >
                        <Zap size={14} />
                      </button>
                    )}
                    <button
                      title="进入审核"
                      onClick={() => navigate('/pipelines/' + p.id + '/review')}
                      style={{
                        width: 30, height: 30, borderRadius: 8,
                        border: '1px solid rgba(0,122,255,0.3)', background: 'rgba(0,122,255,0.08)',
                        cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center',
                        color: '#007aff', padding: 0, transition: 'all 0.15s ease',
                      }}
                      onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.15)' }}
                      onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.08)' }}
                    >
                      <ArrowRight size={14} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* ===== 已完成审核记录 ===== */}
      {!loading && completedReview.length > 0 && (
        <div style={{
          background: '#fff', borderRadius: 16, border: '1px solid rgba(0,0,0,0.04)',
          boxShadow: '0 1px 3px rgba(0,0,0,0.04)', overflow: 'hidden',
        }}>
          {/* 区块标题（可折叠） */}
          <div
            style={{
              display: 'flex', alignItems: 'center', gap: 10,
              padding: '14px 20px', borderBottom: showCompleted ? '1px solid rgba(0,0,0,0.04)' : 'none',
              cursor: 'pointer', transition: 'background 0.15s ease',
            }}
            onClick={() => setShowCompleted(!showCompleted)}
            onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,0,0,0.015)' }}
            onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent' }}
          >
            <CheckCircle size={18} color="#34c759" />
            <span style={{ fontSize: 15, fontWeight: 700, color: '#1c1c1e', flex: 1 }}>已完成审核</span>
            <span style={{
              fontSize: 12, color: '#8e8e93', background: 'rgba(0,0,0,0.04)',
              padding: '2px 10px', borderRadius: 10, fontWeight: 500,
            }}>最近 {completedReview.length} 条</span>
            <span style={{
              fontSize: 12, color: '#c7c7cc', transition: 'transform 0.2s',
              transform: showCompleted ? 'rotate(90deg)' : 'none',
            }}>&#9654;</span>
          </div>

          {/* 已完成列表 */}
          {showCompleted && (
            <div>
              {completedReview.map((p, idx) => (
                <div
                  key={p.id}
                  style={{
                    display: 'flex', alignItems: 'center', gap: 14,
                    padding: '12px 20px',
                    borderBottom: idx < completedReview.length - 1 ? '1px solid rgba(0,0,0,0.03)' : 'none',
                    transition: 'background 0.15s ease, opacity 0.15s ease', cursor: 'pointer',
                    opacity: 0.85,
                  }}
                  onClick={() => navigate('/pipelines/' + p.id)}
                  onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(0,122,255,0.03)'; (e.currentTarget as HTMLElement).style.opacity = '1' }}
                  onMouseLeave={e => { (e.currentTarget as HTMLElement).style.background = 'transparent'; (e.currentTarget as HTMLElement).style.opacity = '0.85' }}
                >
                  {/* 状态图标 */}
                  <div style={{ flexShrink: 0 }}>
                    {p.status === 'verified' ? (
                      <ShieldCheck size={18} color="#248a3d" />
                    ) : p.status === 'verify_failed' ? (
                      <XCircle size={18} color="#d70015" />
                    ) : (
                      <CheckCircle size={18} color="#34c759" />
                    )}
                  </div>

                  {/* 课程信息 */}
                  <div style={{ flex: 1, minWidth: 0 }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
                      <span style={{ fontSize: 13, fontWeight: 600, color: '#1c1c1e' }}>{p.course_code}</span>
                      <StatusBadge status={p.status} />
                      <RoundBadge round={p.review_round || 1} />
                    </div>
                    <div style={{
                      fontSize: 11, color: '#aeaeb2', marginTop: 2,
                      overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                    }}>
                      {p.course_name || p.course_code}
                    </div>
                  </div>

                  {/* 分数 */}
                  <div style={{ display: 'flex', gap: 12, flexShrink: 0 }}>
                    <ScoreDisplay value={p.meta_score} label="Meta" />
                    <ScoreDisplay value={p.translator_score} label="翻译" />
                  </div>

                  {/* 完成时间 */}
                  <div style={{ fontSize: 11, color: '#c7c7cc', flexShrink: 0, minWidth: 60, textAlign: 'right' }}>
                    {formatTime(p.completed_at || p.created_at)}
                  </div>

                  {/* 箭头 */}
                  <div style={{ color: '#c7c7cc', fontSize: 14, flexShrink: 0 }}>&#8250;</div>
                </div>
              ))}
            </div>
          )}
        </div>
      )}

      {/* 旋转动画CSS */}
      <style>{`@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }`}</style>

      {/* Toast */}
      {toast && <Toast message={toast.message} type={toast.type} onClose={() => setToast(null)} />}
    </div>
  )
}

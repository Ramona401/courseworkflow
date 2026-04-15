/**
 * PipelineDetailPage — Pipeline详情页（主文件）
 *
 * v37改进: 断点续跑按钮：admin/senior_operator在任何非running状态可见；
 *          operator仅在failed/cancelled可见
 * v41修复: 导航路径加 /workflow 前缀
 * v46修复: showReviewBtn 增加 pending_finalize 状态
 * v69新增: 编号12 — 全流程时间线组件（1审→验收→2审→验收）
 * v69新增: 编号13 — 验收后自动追踪2审进度（SSE扩展+验收后轮询+2审横幅）
 *
 * 子组件均从 ./components/ 引入，本文件只保留：
 *   - SSE实时推送 + 轮询回退逻辑
 *   - 验收/断点续跑操作函数
 *   - 页面级渲染框架（顶部导航 + 各横幅 + 8步进度 + 步骤卡片列表）
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  getPipelineDetail, verifyPipeline, restartFromStep,
  type PipelineDetailResponse,
} from '@/api/pipelines'
import { ArrowLeft, RefreshCw, ClipboardCheck, ShieldCheck } from 'lucide-react'
import { useAuth } from '@/store/auth'

// ---- 子组件 ----
import {
  RunningBanner, FailedBanner, MessageBanner,
  EarlyStopBanner, VerifiedBanner, VerifyFailedBanner, PublishedBanner,
  RetrialRunningBanner,
} from './components/PipelineDetailBanners'
import { StepCard } from './components/PipelineStepCard'
import { HistoryRoundPanel, PipelineFlowTimeline } from './components/HistoryRoundPanel'
import { PublishButton } from './components/PublishButton'

// ==================== 常量 ====================

/** 轮询间隔（SSE回退模式）*/
const POLL_INTERVAL_MS = 5000
/** SSE基础URL */
const SSE_BASE_URL = '/api/v1/sse/pipelines'

// ==================== 主组件 ====================

export default function PipelineDetailPage() {
  const { id }    = useParams<{ id: string }>()
  const navigate  = useNavigate()
  const { token, user } = useAuth()

  const [detail, setDetail]   = useState<PipelineDetailResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError]     = useState('')

  // 高级角色判断
  const isHighRole = user?.role === 'admin' || user?.role === 'senior_operator'

  // 验收操作状态
  const [verifying, setVerifying]   = useState(false)
  const [verifyMsg, setVerifyMsg]   = useState('')

  // 断点续跑操作状态
  const [restarting, setRestarting] = useState(false)
  const [restartMsg, setRestartMsg] = useState('')

  // SSE连接状态
  const [sseConnected, setSseConnected] = useState(false)

  // 定时器 / EventSource引用
  const pollTimerRef    = useRef<ReturnType<typeof setInterval> | null>(null)
  const eventSourceRef  = useRef<EventSource | null>(null)

  // ---- 数据加载 ----
  const loadDetail = useCallback(async () => {
    if (!id) return
    setLoading(true); setError('')
    try {
      const data = await getPipelineDetail(id)
      setDetail(data)
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : '加载失败')
    }
    setLoading(false)
  }, [id])

  const silentRefresh = useCallback(async () => {
    if (!id) return
    try { setDetail(await getPipelineDetail(id)) } catch { /* 静默忽略 */ }
  }, [id])

  useEffect(() => { loadDetail() }, [loadDetail])

  // ==================== SSE实时推送 + 轮询回退 ====================
  useEffect(() => {
    const cleanup = () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close()
        eventSourceRef.current = null
        setSseConnected(false)
      }
      if (pollTimerRef.current) {
        clearInterval(pollTimerRef.current)
        pollTimerRef.current = null
      }
    }

    // v69（编号13）：SSE连接条件扩展——verify_failed状态下可能正在自动触发2审
    // 需要SSE/轮询来追踪2审的执行进度
    const needsTracking = detail && id && (
      detail.status === 'running' || detail.status === 'verify_failed'
    )
    if (!needsTracking) {
      cleanup(); return cleanup
    }

    const startPolling = () => {
      if (pollTimerRef.current) return
      pollTimerRef.current = setInterval(silentRefresh, POLL_INTERVAL_MS)
    }

    const connectSSE = () => {
      try {
        const url = `${SSE_BASE_URL}/${id}/stream?token=${encodeURIComponent(token || '')}`
        const es  = new EventSource(url)
        eventSourceRef.current = es

        es.addEventListener('connected',      () => { setSseConnected(true); if (pollTimerRef.current) { clearInterval(pollTimerRef.current); pollTimerRef.current = null } })
        es.addEventListener('step_update',    () => { silentRefresh() })
        es.addEventListener('pipeline_done',  () => { silentRefresh(); cleanup() })
        es.addEventListener('pipeline_error', () => { silentRefresh(); cleanup() })
        es.onerror = () => { es.close(); eventSourceRef.current = null; setSseConnected(false); startPolling() }
      } catch { startPolling() }
    }

    connectSSE()
    return cleanup
  // eslint-disable-next-line
  }, [detail?.status, id, silentRefresh, token])

  // ==================== 验收 ====================
  const handleVerify = async () => {
    if (!id || verifying) return
    if (!window.confirm('启动验收将执行2次AI调用（索引压缩+评估），预计耗时5-15分钟。确认启动？')) return
    setVerifying(true)
    setVerifyMsg('验收进行中，请耐心等待（预计5-15分钟）...')
    try {
      await verifyPipeline(id)
      setVerifyMsg('验收完成！正在刷新...')
      await loadDetail()
      // v69（编号13）：验收完成后启动短轮询追踪2审进度
      // 如果验收未通过触发了2审，需要持续刷新直到2审执行完成
      startPostVerifyPolling()
      setVerifyMsg('')
    } catch (e: unknown) {
      setVerifyMsg('验收失败: ' + (e instanceof Error ? e.message : '未知错误'))
    }
    setVerifying(false)
  }

  // ==================== v69新增（编号13）：验收后轮询追踪 ====================
  // 验收完成后，如果触发了2审，启动短轮询（3秒间隔）追踪2审执行进度
  // 持续刷新直到状态稳定（review_queue/needs_human/verified/failed等非running状态）
  const postVerifyTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)

  const startPostVerifyPolling = useCallback(() => {
    // 清除可能存在的旧轮询
    if (postVerifyTimerRef.current) {
      clearInterval(postVerifyTimerRef.current)
      postVerifyTimerRef.current = null
    }
    let pollCount = 0
    const maxPolls = 120 // 最多轮询120次（6分钟），防止无限轮询
    postVerifyTimerRef.current = setInterval(async () => {
      pollCount++
      if (pollCount >= maxPolls) {
        if (postVerifyTimerRef.current) clearInterval(postVerifyTimerRef.current)
        postVerifyTimerRef.current = null
        return
      }
      try {
        const freshData = await getPipelineDetail(id!)
        setDetail(freshData)
        // 状态稳定后停止轮询
        const stableStatuses = ['review_queue', 'needs_human', 'verified', 'published', 'pending', 'cancelled', 'pending_finalize', 'finalized']
        if (stableStatuses.includes(freshData.status) || freshData.status === 'failed') {
          if (postVerifyTimerRef.current) clearInterval(postVerifyTimerRef.current)
          postVerifyTimerRef.current = null
        }
      } catch { /* 静默忽略 */ }
    }, 3000)
  }, [id])

  // 组件卸载时清除验收后轮询
  useEffect(() => {
    return () => {
      if (postVerifyTimerRef.current) {
        clearInterval(postVerifyTimerRef.current)
        postVerifyTimerRef.current = null
      }
    }
  }, [])

  // ==================== 断点续跑 ====================
  const handleRestartFromStep = async (stepName: string, stepNameCN: string) => {
    if (!id || restarting) return

    const isCompleted = detail && !['failed', 'cancelled', 'pending', 'running'].includes(detail.status)
    const confirmMsg = [
      `确认从「${stepNameCN}」步骤重新执行？`,
      '',
      '此操作将：',
      `• 重置「${stepNameCN}」及后续所有步骤的执行数据`,
      '• 如果包含页面生成步骤，旧的生成页面将被清空',
      '• 审核和验收步骤的旧数据也将被重置',
      '• Pipeline立即进入running状态开始执行',
      ...(isCompleted ? ['', '⚠️ 注意：该Pipeline当前处于已完成状态，重跑后需要重新走审核流程。'] : []),
      '',
      '已完成的前序步骤（如数据检查、评估等）不受影响。',
    ].join('\n')

    if (!window.confirm(confirmMsg)) return

    setRestarting(true)
    setRestartMsg(`正在从「${stepNameCN}」步骤重新执行，请稍候...`)
    try {
      const updated = await restartFromStep(id, stepName)
      setDetail(updated)
      setRestartMsg('')
    } catch (e: unknown) {
      setRestartMsg('重跑失败: ' + (e instanceof Error ? e.message : '未知错误'))
    }
    setRestarting(false)
  }

  // ==================== 渲染 ====================

  const btn: React.CSSProperties = {
    padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer',
    display: 'inline-flex', alignItems: 'center', gap: 6,
  }

  /**
   * v46修复：审核按钮显示条件
   * 增加 pending_finalize：骨干教师提交定稿后，学校管理员需进入审核确认
   */
  const showReviewBtn = detail && (
    detail.status === 'review_queue' || detail.status === 'needs_human' ||
    detail.status === 'pending_finalize' ||
    detail.status === 'finalized'  ||
    detail.status === 'verified'   || detail.status === 'verify_failed'
  )

  const showVerifyBtn = detail && detail.status === 'finalized' && !verifying

  if (loading) return <div style={{ textAlign: 'center', padding: 60, color: '#8e8e93' }}>加载中...</div>

  if (error) {
    return (
      <div style={{ textAlign: 'center', padding: 60 }}>
        <div style={{ color: '#ff3b30', fontSize: 14, marginBottom: 12 }}>{error}</div>
        <button style={btn} onClick={() => navigate('/workflow/pipelines')}>
          <ArrowLeft size={14} /> 返回列表
        </button>
      </div>
    )
  }

  if (!detail) return null

  return (
    <div>
      {/* ---- 顶部导航栏 ---- */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
        <button style={btn} onClick={() => navigate('/workflow/pipelines')}>
          <ArrowLeft size={14} /> 返回
        </button>

        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 18, fontWeight: 700, color: '#1c1c1e' }}>
            {detail.course_code} — {detail.course_name}
          </div>
          <div style={{ fontSize: 13, color: '#8e8e93', marginTop: 2 }}>
            状态: {detail.status_name}
            {detail.review_round > 1 ? ` (第${detail.review_round}审)` : ''}
            {' · '}当前步骤: {detail.current_step_name}
            {detail.config && (
              <span style={{ marginLeft: 12, color: '#aeaeb2' }}>
                阈值: {detail.config.threshold} · 评估轮: {detail.config.eval_rounds} · TR循环: {detail.config.max_tr_loop}
              </span>
            )}
          </div>
        </div>

        {/* 启动验收按钮 */}
        {showVerifyBtn && (
          <button style={{ ...btn, background: '#5856d6', color: '#fff', border: 'none' }} onClick={handleVerify}>
            <ShieldCheck size={14} /> 启动验收
          </button>
        )}
        {verifying && (
          <button style={{ ...btn, background: '#5856d6', color: '#fff', border: 'none', opacity: 0.7, cursor: 'not-allowed' }} disabled>
            <RefreshCw size={14} style={{ animation: 'spin 1s linear infinite' }} /> 验收中...
          </button>
        )}

        {/* 进入审核按钮 */}
        {showReviewBtn && (
          <button
            style={{
              ...btn, color: '#fff', border: 'none',
              background:
                detail.status === 'finalized' || detail.status === 'verified' ? '#34c759'
                : detail.status === 'verify_failed' ? '#ff9500'
                : detail.status === 'pending_finalize' ? '#cc6600'
                : '#007aff',
            }}
            onClick={() => navigate(`/workflow/pipelines/${id}/review`)}>
            <ClipboardCheck size={14} />
            {detail.status === 'finalized' || detail.status === 'verified' || detail.status === 'verify_failed'
              ? '查看审核结果'
              : detail.status === 'pending_finalize'
              ? '查看并确认定稿'
              : '进入审核'
            }
          </button>
        )}

        <button style={btn} onClick={loadDetail}>
          <RefreshCw size={14} /> 刷新
        </button>
      </div>

      {/* ---- 各状态横幅 ---- */}
      {detail.status === 'running' && (
        <RunningBanner sseConnected={sseConnected} currentStepName={detail.current_step_name} />
      )}
      {(detail.status === 'failed' || detail.status === 'cancelled') && (
        <FailedBanner status={detail.status} errorMessage={detail.error_message} />
      )}
      {verifyMsg  && <MessageBanner message={verifyMsg}  type="verify"  />}
      {/* v69新增（编号13）：2审进行中横幅 */}
      <RetrialRunningBanner detail={detail} sseConnected={sseConnected} />
      {restartMsg && <MessageBanner message={restartMsg} type="restart" />}
      <EarlyStopBanner detail={detail} />
      {/* verified 状态：显示待发布提示区块 */}
      {detail.status === 'verified' && (
        <div style={{ marginBottom: 16 }}>
          <PublishButton pipelineId={detail.id} onPublished={loadDetail} />
        </div>
      )}
      {/* published 状态：显示已发布横幅 */}
      {detail.status === 'published' && <PublishedBanner />}
      {detail.status === 'verified'      && <VerifiedBanner />}
      {detail.status === 'verify_failed' && <VerifyFailedBanner reviewRound={detail.review_round} />}

      {/* ---- v69新增（编号12）：全流程时间线（1审→验收→2审→验收）---- */}
      <PipelineFlowTimeline detail={detail} />

      {/* ---- 8步进度可视化 ---- */}
      <div style={{
        background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
        border: '1px solid rgba(0,0,0,0.06)', borderRadius: 16, padding: 24, marginBottom: 20,
      }}>
        <div style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e', marginBottom: 16 }}>执行步骤</div>
        <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
          {detail.steps.map(step => {
            const colorMap: Record<string, string> = {
              done: '#34c759', running: '#007aff', failed: '#ff3b30',
              skipped: '#aeaeb2', pending: '#e5e5ea',
            }
            const color = colorMap[step.status] || '#e5e5ea'
            return (
              <div key={step.step_name} style={{ flex: 1, textAlign: 'center' }}>
                <div style={{ height: 6, borderRadius: 3, background: color, marginBottom: 8, transition: 'background 0.3s ease' }} />
                <div style={{ fontSize: 11, fontWeight: 600, color: step.status === 'done' ? '#34c759' : step.status === 'running' ? '#007aff' : step.status === 'failed' ? '#ff3b30' : '#aeaeb2' }}>
                  {step.step_name_cn}
                </div>
                <div style={{ fontSize: 10, color: '#aeaeb2', marginTop: 2 }}>{step.status_name}</div>
                {step.duration_ms > 0 && (
                  <div style={{ fontSize: 10, color: '#c7c7cc', marginTop: 1 }}>
                    {step.duration_ms < 1000 ? step.duration_ms + 'ms' : (step.duration_ms / 1000).toFixed(1) + 's'}
                  </div>
                )}
              </div>
            )
          })}
        </div>
      </div>

      {/* ---- 步骤详情卡片列表 ---- */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {detail.steps
          .filter(s => s.has_data || s.status === 'done' || s.status === 'failed')
          .map(step => (
            <StepCard
              key={step.step_name}
              pipelineId={detail.id}
              step={step}
              pipelineStatus={detail.status}
              onRestartFromStep={handleRestartFromStep}
              restarting={restarting}
              isHighRole={isHighRole}
            />
          ))
        }
      </div>

      {/* ---- 历史轮次数据（review_round >= 2 时显示）---- */}
      {detail.review_round >= 2 && (
        <HistoryRoundPanel
          pipelineId={detail.id}
          currentRound={detail.review_round}
        />
      )}

      <style>{`
        @keyframes spin {
          from { transform: rotate(0deg); }
          to   { transform: rotate(360deg); }
        }
      `}</style>
    </div>
  )
}

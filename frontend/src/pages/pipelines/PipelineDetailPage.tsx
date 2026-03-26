/**
 * Pipeline详情页面
 * v37改进: 断点续跑功能增强——admin/senior_operator在任何非running状态都能看到重跑按钮
 *          operator仅在failed/cancelled状态看到重跑按钮（与v36行为一致）
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import {
  getPipelineDetail,
  getStepDetail,
  verifyPipeline,
  restartFromStep,
  type PipelineDetailResponse,
  type StepListItem,
} from '@/api/pipelines'
import { ArrowLeft, RefreshCw, ClipboardCheck, ShieldCheck, Radio, RotateCcw } from 'lucide-react'
import { useAuth } from '@/store/auth'

// 导入各步骤面板组件
import DbCheckPanel from './steps/DbCheckPanel'
import ScannerPanel from './steps/ScannerPanel'
import EvaluatorPanel from './steps/EvaluatorPanel'
import MetaPanel from './steps/MetaPanel'
import TranslatorPanel from './steps/TranslatorPanel'
import GeneratorPanel from './steps/GeneratorPanel'
import VerifyPanel from './steps/VerifyPanel'

// ==================== 常量 ====================

/** 轮询间隔（SSE回退模式使用） */
const POLL_INTERVAL_MS = 5000
/** SSE基础URL */
const SSE_BASE_URL = '/api/v1/sse/pipelines'

/**
 * 支持断点续跑的步骤集合
 * review 是人工审核步骤不支持重跑；verify 有独立入口
 */
const RESTARTABLE_STEPS = new Set([
  'dbCheck', 'scanner', 'evaluator', 'meta', 'translator', 'generator',
])

/** 步骤中文名称映射（用于确认弹窗） */
const STEP_CN_MAP: Record<string, string> = {
  dbCheck: '数据检查', scanner: '扫描定位', evaluator: '多轮评估',
  meta: 'Meta综合', translator: '优化翻译', generator: '页面生成',
}

// ==================== 主页面组件 ====================

export default function PipelineDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { token, user } = useAuth()
  const [detail, setDetail] = useState<PipelineDetailResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  // v37新增：判断当前用户是否为高级角色（admin/senior_operator）
  const isHighRole = user?.role === 'admin' || user?.role === 'senior_operator'

  // 验收操作状态
  const [verifying, setVerifying] = useState(false)
  const [verifyMsg, setVerifyMsg] = useState('')

  // 断点续跑操作状态
  const [restarting, setRestarting] = useState(false)
  const [restartMsg, setRestartMsg] = useState('')

  // SSE连接状态
  const [sseConnected, setSseConnected] = useState(false)

  // 轮询定时器引用（SSE回退模式）
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const eventSourceRef = useRef<EventSource | null>(null)

  /** 加载Pipeline详情 */
  const loadDetail = useCallback(async () => {
    if (!id) return
    setLoading(true)
    setError('')
    try {
      const data = await getPipelineDetail(id)
      setDetail(data)
    } catch (e: any) {
      setError(e.message || '加载失败')
    }
    setLoading(false)
  }, [id])

  /** 静默刷新 */
  const silentRefresh = useCallback(async () => {
    if (!id) return
    try {
      const data = await getPipelineDetail(id)
      setDetail(data)
    } catch { /* 静默忽略 */ }
  }, [id])

  useEffect(() => { loadDetail() }, [loadDetail])

  // ==================== SSE实时推送 + 回退轮询 ====================
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

    if (!detail || detail.status !== 'running' || !id) {
      cleanup()
      return cleanup
    }

    const connectSSE = () => {
      try {
        const url = SSE_BASE_URL + '/' + id + '/stream?token=' + encodeURIComponent(token || '')
        const es = new EventSource(url)
        eventSourceRef.current = es

        es.addEventListener('connected', () => {
          setSseConnected(true)
          if (pollTimerRef.current) { clearInterval(pollTimerRef.current); pollTimerRef.current = null }
        })
        es.addEventListener('step_update', () => { silentRefresh() })
        es.addEventListener('pipeline_done', () => { silentRefresh(); cleanup() })
        es.addEventListener('pipeline_error', () => { silentRefresh(); cleanup() })
        es.onerror = () => { es.close(); eventSourceRef.current = null; setSseConnected(false); startPolling() }
      } catch { startPolling() }
    }

    const startPolling = () => {
      if (pollTimerRef.current) return
      pollTimerRef.current = setInterval(silentRefresh, POLL_INTERVAL_MS)
    }

    connectSSE()
    return cleanup
  }, [detail?.status, id, silentRefresh])

  // ==================== 启动验收 ====================
  const handleVerify = async () => {
    if (!id || verifying) return
    if (!window.confirm('启动验收将执行2次AI调用（索引压缩+评估），预计耗时5-15分钟。确认启动？')) return
    setVerifying(true)
    setVerifyMsg('验收进行中，请耐心等待（预计5-15分钟）...')
    try {
      await verifyPipeline(id)
      setVerifyMsg('验收完成！正在刷新...')
      await loadDetail()
      setVerifyMsg('')
    } catch (e: any) {
      setVerifyMsg('验收失败: ' + (e.message || '未知错误'))
    }
    setVerifying(false)
  }

  // ==================== 断点续跑 ====================
  const handleRestartFromStep = async (stepName: string, stepNameCN: string) => {
    if (!id || restarting) return

    // v37改进：对已完成Pipeline重跑给予更明确的警告
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
    } catch (e: any) {
      setRestartMsg('重跑失败: ' + (e.message || '未知错误'))
    }
    setRestarting(false)
  }

  // ==================== 样式常量 ====================
  const btn: React.CSSProperties = {
    padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer',
    display: 'inline-flex', alignItems: 'center', gap: 6,
  }

  const showReviewBtn = detail && (
    detail.status === 'review_queue' || detail.status === 'needs_human' ||
    detail.status === 'finalized' || detail.status === 'verified' ||
    detail.status === 'verify_failed'
  )

  const showVerifyBtn = detail && detail.status === 'finalized' && !verifying

  // ==================== 渲染 ====================

  if (loading) {
    return <div style={{ textAlign: 'center', padding: 60, color: '#8e8e93' }}>加载中...</div>
  }

  if (error) {
    return (
      <div style={{ textAlign: 'center', padding: 60 }}>
        <div style={{ color: '#ff3b30', fontSize: 14, marginBottom: 12 }}>{error}</div>
        <button style={btn} onClick={() => navigate('/pipelines')}>
          <ArrowLeft size={14} /> 返回列表
        </button>
      </div>
    )
  }

  if (!detail) return null

  return (
    <div>
      {/* 顶部导航栏 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
        <button style={btn} onClick={() => navigate('/pipelines')}>
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
        {showReviewBtn && (
          <button
            style={{
              ...btn,
              background: (detail.status === 'finalized' || detail.status === 'verified') ? '#34c759'
                : detail.status === 'verify_failed' ? '#ff9500' : '#007aff',
              color: '#fff', border: 'none',
            }}
            onClick={() => navigate('/pipelines/' + id + '/review')}
          >
            <ClipboardCheck size={14} />
            {detail.status === 'finalized' || detail.status === 'verified' || detail.status === 'verify_failed'
              ? '查看审核结果' : '进入审核'}
          </button>
        )}

        <button style={btn} onClick={loadDetail}>
          <RefreshCw size={14} /> 刷新
        </button>
      </div>

      {/* running状态实时推送横幅 */}
      {detail.status === 'running' && (
        <div style={{
          display: 'flex', alignItems: 'center', gap: 10,
          padding: '10px 16px', borderRadius: 10, marginBottom: 16,
          background: sseConnected ? 'rgba(52,199,89,0.06)' : 'rgba(0,122,255,0.06)',
          border: `1px solid ${sseConnected ? 'rgba(52,199,89,0.15)' : 'rgba(0,122,255,0.15)'}`,
        }}>
          {sseConnected
            ? <Radio size={16} color="#34c759" />
            : <RefreshCw size={16} color="#007aff" style={{ animation: 'spin 2s linear infinite' }} />
          }
          <div style={{ fontSize: 13, color: sseConnected ? '#34c759' : '#007aff', fontWeight: 500 }}>
            {sseConnected ? 'Pipeline正在执行中，实时推送已连接...' : 'Pipeline正在执行中，页面每5秒自动刷新...'}
          </div>
          <div style={{ fontSize: 12, color: '#8e8e93' }}>当前步骤: {detail.current_step_name}</div>
          {sseConnected && (
            <span style={{
              marginLeft: 'auto', fontSize: 10, padding: '2px 8px', borderRadius: 10,
              background: 'rgba(52,199,89,0.12)', color: '#34c759', fontWeight: 600,
            }}>SSE</span>
          )}
        </div>
      )}

      {/* failed/cancelled状态提示横幅 */}
      {(detail.status === 'failed' || detail.status === 'cancelled') && (
        <div style={{
          display: 'flex', alignItems: 'flex-start', gap: 12,
          padding: '12px 18px', borderRadius: 12, marginBottom: 16,
          background: detail.status === 'failed'
            ? 'linear-gradient(135deg, rgba(255,59,48,0.08), rgba(255,59,48,0.03))'
            : 'linear-gradient(135deg, rgba(142,142,147,0.08), rgba(142,142,147,0.03))',
          border: `1px solid ${detail.status === 'failed' ? 'rgba(255,59,48,0.2)' : 'rgba(142,142,147,0.2)'}`,
        }}>
          <RotateCcw size={18} color={detail.status === 'failed' ? '#ff3b30' : '#8e8e93'} style={{ marginTop: 1, flexShrink: 0 }} />
          <div>
            <div style={{ fontSize: 14, fontWeight: 700, color: detail.status === 'failed' ? '#ff3b30' : '#8e8e93' }}>
              {detail.status === 'failed' ? 'Pipeline执行失败' : 'Pipeline已取消'}
            </div>
            <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 3 }}>
              {detail.error_message ? `错误信息: ${detail.error_message}` : ''}
            </div>
            <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 2 }}>
              展开下方步骤卡片，点击「从此步重跑」可从任意步骤重新执行，无需重建Pipeline。
            </div>
          </div>
        </div>
      )}

      {/* 验收消息提示 */}
      {verifyMsg && (
        <div style={{
          padding: '10px 16px', borderRadius: 10, marginBottom: 16, fontSize: 13, fontWeight: 500,
          background: verifyMsg.includes('失败') ? 'rgba(255,59,48,0.08)' : 'rgba(88,86,214,0.08)',
          color: verifyMsg.includes('失败') ? '#ff3b30' : '#5856d6',
          border: `1px solid ${verifyMsg.includes('失败') ? 'rgba(255,59,48,0.2)' : 'rgba(88,86,214,0.2)'}`,
        }}>
          {verifyMsg}
        </div>
      )}

      {/* 断点续跑消息提示 */}
      {restartMsg && (
        <div style={{
          padding: '10px 16px', borderRadius: 10, marginBottom: 16, fontSize: 13, fontWeight: 500,
          display: 'flex', alignItems: 'center', gap: 8,
          background: restartMsg.includes('失败') ? 'rgba(255,59,48,0.08)' : 'rgba(255,149,0,0.08)',
          color: restartMsg.includes('失败') ? '#ff3b30' : '#ff9500',
          border: `1px solid ${restartMsg.includes('失败') ? 'rgba(255,59,48,0.2)' : 'rgba(255,149,0,0.2)'}`,
        }}>
          {!restartMsg.includes('失败') && (
            <RefreshCw size={13} style={{ animation: 'spin 1s linear infinite', flexShrink: 0 }} />
          )}
          {restartMsg}
        </div>
      )}

      {/* 高分早停横幅 */}
      {detail.status === 'review_queue' &&
        detail.steps.some(s => s.step_name === 'review' && s.has_data) &&
        (() => {
          const metaStep = detail.steps.find(s => s.step_name === 'meta')
          const transStep = detail.steps.find(s => s.step_name === 'translator')
          const genStep = detail.steps.find(s => s.step_name === 'generator')
          const isEarlyStop = metaStep?.status === 'pending' && transStep?.status === 'pending' && genStep?.status === 'pending'
          if (!isEarlyStop) return null
          return (
            <div style={{
              display: 'flex', alignItems: 'center', gap: 10,
              padding: '12px 18px', borderRadius: 12, marginBottom: 16,
              background: 'linear-gradient(135deg, rgba(52,199,89,0.1), rgba(52,199,89,0.03))',
              border: '1px solid rgba(52,199,89,0.25)',
            }}>
              <span style={{ fontSize: 20 }}>⚡</span>
              <div>
                <div style={{ fontSize: 14, fontWeight: 700, color: '#34c759' }}>高分早停 — 直接进入审核</div>
                <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 2 }}>
                  Evaluator多轮评估均分已达到阈值（{detail.config?.threshold ?? 9.0}分），
                  课程质量良好，已跳过Meta优化/翻译/页面生成步骤，直接进入审核队列。
                  <span style={{ marginLeft: 6, color: '#34c759', fontWeight: 600 }}>
                    可点击「进入审核」查看原始页面。
                  </span>
                </div>
              </div>
            </div>
          )
        })()
      }

      {/* verified状态验收通过横幅 */}
      {detail.status === 'verified' && (
        <div style={{
          display: 'flex', alignItems: 'center', gap: 10,
          padding: '12px 18px', borderRadius: 12, marginBottom: 16,
          background: 'linear-gradient(135deg, rgba(52,199,89,0.1), rgba(52,199,89,0.03))',
          border: '1px solid rgba(52,199,89,0.25)',
        }}>
          <ShieldCheck size={20} color="#34c759" />
          <div>
            <div style={{ fontSize: 14, fontWeight: 700, color: '#34c759' }}>验收通过</div>
            <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 1 }}>该Pipeline已通过验收评估，质量达标（≥9.0分）</div>
          </div>
        </div>
      )}

      {/* verify_failed状态验收未通过横幅 */}
      {detail.status === 'verify_failed' && (
        <div style={{
          display: 'flex', alignItems: 'center', gap: 10,
          padding: '12px 18px', borderRadius: 12, marginBottom: 16,
          background: 'linear-gradient(135deg, rgba(255,149,0,0.1), rgba(255,149,0,0.03))',
          border: '1px solid rgba(255,149,0,0.25)',
        }}>
          <ShieldCheck size={20} color="#ff9500" />
          <div>
            <div style={{ fontSize: 14, fontWeight: 700, color: '#ff9500' }}>验收未通过</div>
            <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 1 }}>
              {detail.review_round >= 2
                ? '2审验收仍未达标，需要人工介入处理'
                : '验收评分未达标（<9.0分），系统将自动启动2审流程'
              }
            </div>
          </div>
        </div>
      )}

      {/* 8步进度可视化 */}
      <div style={{
        background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
        border: '1px solid rgba(0,0,0,0.06)', borderRadius: 16, padding: 24, marginBottom: 20,
      }}>
        <div style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e', marginBottom: 16 }}>执行步骤</div>
        <div style={{ display: 'flex', gap: 4, alignItems: 'center' }}>
          {detail.steps.map((step) => {
            const colorMap: Record<string, string> = {
              done: '#34c759', running: '#007aff', failed: '#ff3b30',
              skipped: '#aeaeb2', pending: '#e5e5ea',
            }
            const color = colorMap[step.status] || '#e5e5ea'
            return (
              <div key={step.step_name} style={{ flex: 1, textAlign: 'center' }}>
                <div style={{
                  height: 6, borderRadius: 3, background: color,
                  marginBottom: 8, transition: 'background 0.3s ease',
                }} />
                <div style={{
                  fontSize: 11, fontWeight: 600,
                  color: step.status === 'done' ? '#34c759'
                    : step.status === 'running' ? '#007aff'
                    : step.status === 'failed' ? '#ff3b30' : '#aeaeb2',
                }}>
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

      {/* 步骤详情卡片列表 */}
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

      <style>{`
        @keyframes spin {
          from { transform: rotate(0deg); }
          to { transform: rotate(360deg); }
        }
      `}</style>
    </div>
  )
}

// ==================== 步骤详情卡片 ====================

interface StepCardProps {
  pipelineId: string
  step: StepListItem
  pipelineStatus: string
  onRestartFromStep: (stepName: string, stepNameCN: string) => void
  restarting: boolean
  /** v37新增：是否为高级角色（admin/senior_operator），决定重跑按钮在哪些状态下显示 */
  isHighRole: boolean
}

function StepCard({ pipelineId, step, pipelineStatus, onRestartFromStep, restarting, isHighRole }: StepCardProps) {
  const [expanded, setExpanded] = useState(false)
  const [stepData, setStepData] = useState<any>(null)
  const [loadingData, setLoadingData] = useState(false)

  const toggleExpand = async () => {
    if (expanded) { setExpanded(false); return }
    setExpanded(true)
    if (!stepData && step.has_data) {
      setLoadingData(true)
      try {
        const data = await getStepDetail(pipelineId, step.step_name)
        setStepData(data.step_data)
      } catch { /* 静默处理 */ }
      setLoadingData(false)
    }
  }

  // v37改进：重跑按钮显示条件
  // - 该步骤必须在 RESTARTABLE_STEPS 集合中
  // - 当前没有正在进行的重跑操作
  // - Pipeline不在running状态
  // - 权限判断：
  //   * admin/senior_operator：任何非running状态都能看到重跑按钮
  //   * operator：仅 failed/cancelled 状态
  const showRestartBtn = (() => {
    if (!RESTARTABLE_STEPS.has(step.step_name)) return false
    if (restarting) return false
    if (pipelineStatus === 'running' || pipelineStatus === 'pending') return false
    if (isHighRole) return true
    // 普通操作员仅在 failed/cancelled 时显示
    return pipelineStatus === 'failed' || pipelineStatus === 'cancelled'
  })()

  const statusColors: Record<string, string> = {
    done: '#34c759', running: '#007aff', failed: '#ff3b30',
    skipped: '#aeaeb2', pending: '#c7c7cc',
  }

  return (
    <div style={{
      background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
      border: `1px solid ${step.status === 'failed' ? 'rgba(255,59,48,0.15)' : 'rgba(0,0,0,0.06)'}`,
      borderRadius: 14, overflow: 'hidden',
    }}>
      <div onClick={toggleExpand} style={{
        display: 'flex', alignItems: 'center', gap: 12, padding: '14px 18px', cursor: 'pointer',
        borderBottom: expanded ? '1px solid rgba(0,0,0,0.04)' : 'none',
      }}>
        <div style={{
          width: 8, height: 8, borderRadius: 4,
          background: statusColors[step.status] || '#c7c7cc', flexShrink: 0,
        }} />
        <div style={{ flex: 1 }}>
          <span style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>
            {step.step_order}. {step.step_name_cn}
          </span>
          <span style={{ fontSize: 12, color: '#8e8e93', marginLeft: 10 }}>{step.status_name}</span>
        </div>
        {step.duration_ms > 0 && (
          <span style={{ fontSize: 12, color: '#aeaeb2' }}>
            {step.duration_ms < 1000 ? step.duration_ms + 'ms' : (step.duration_ms / 1000).toFixed(1) + 's'}
          </span>
        )}
        {step.tokens_used > 0 && (
          <span style={{ fontSize: 12, color: '#aeaeb2' }}>
            {step.tokens_used.toLocaleString()} tokens
          </span>
        )}

        {/* 断点续跑按钮 */}
        {showRestartBtn && (
          <button
            onClick={(e) => {
              e.stopPropagation()
              onRestartFromStep(step.step_name, STEP_CN_MAP[step.step_name] || step.step_name_cn)
            }}
            style={{
              padding: '5px 12px', borderRadius: 8, border: 'none',
              background: 'rgba(255,149,0,0.12)', color: '#ff9500',
              fontSize: 12, fontWeight: 600, cursor: 'pointer',
              display: 'inline-flex', alignItems: 'center', gap: 4, flexShrink: 0,
            }}
            title={`从「${STEP_CN_MAP[step.step_name] || step.step_name_cn}」步骤重新执行`}
          >
            <RotateCcw size={12} />
            从此步重跑
          </button>
        )}

        <span style={{
          fontSize: 12, color: '#c7c7cc',
          transition: 'transform 0.2s', transform: expanded ? 'rotate(90deg)' : 'none',
        }}>▶</span>
      </div>

      {expanded && (
        <div style={{ padding: '14px 18px' }}>
          {step.status === 'failed' && step.error_message && (
            <div style={{
              padding: '10px 14px', borderRadius: 8, marginBottom: 12,
              background: 'rgba(255,59,48,0.06)', border: '1px solid rgba(255,59,48,0.15)',
              color: '#ff3b30', fontSize: 13, whiteSpace: 'pre-wrap', lineHeight: 1.5,
            }}>
              <div style={{ fontWeight: 600, marginBottom: 4 }}>错误信息：</div>
              {step.error_message}
            </div>
          )}
          {loadingData ? (
            <div style={{ color: '#8e8e93', fontSize: 13 }}>加载步骤数据...</div>
          ) : stepData ? (
            <StepPanelRouter stepName={step.step_name} data={stepData} />
          ) : !step.error_message ? (
            <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>
          ) : null}
        </div>
      )}
    </div>
  )
}

// ==================== 步骤面板路由 ====================

function StepPanelRouter({ stepName, data }: { stepName: string; data: any }) {
  if (!data) return <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>

  switch (stepName) {
    case 'dbCheck':    return <DbCheckPanel data={data} />
    case 'scanner':    return <ScannerPanel data={data} />
    case 'evaluator':  return <EvaluatorPanel data={data} />
    case 'meta':       return <MetaPanel data={data} />
    case 'translator': return <TranslatorPanel data={data} />
    case 'generator':  return <GeneratorPanel data={data} />
    case 'verify':     return <VerifyPanel data={data} />
    default:
      return (
        <pre style={{
          fontSize: 12, color: '#3c3c43', overflow: 'auto', maxHeight: 400,
          margin: 0, whiteSpace: 'pre-wrap',
        }}>
          {JSON.stringify(data, null, 2)}
        </pre>
      )
  }
}

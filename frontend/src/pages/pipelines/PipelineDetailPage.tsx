/**
 * Pipeline详情页面
 * P4.5-B: 重构为主页面 + 6个独立步骤面板组件
 * P4.5-C: 新增审核入口按钮（review_queue/needs_human状态时显示）
 * P4.6-1: 新增verified/verify_failed状态适配+审核轮次显示
 * P4.6-5: 新增VerifyPanel专用面板+启动验收按钮（finalized状态）
 * P5-1: 新增running状态自动轮询（每5秒刷新）
 * P5-4: 改用SSE实时推送替代轮询，SSE失败时回退轮询
 * 8步进度可视化 + StepCard懒加载展开 + 各步骤调试面板
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { getPipelineDetail, getStepDetail, verifyPipeline, type PipelineDetailResponse, type StepListItem } from '@/api/pipelines'
import { ArrowLeft, RefreshCw, ClipboardCheck, ShieldCheck, Radio } from 'lucide-react'

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
const SSE_BASE_URL = '/api/v1/pipelines'

// ==================== 主页面组件 ====================

export default function PipelineDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<PipelineDetailResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  // P4.6-5：验收操作状态
  const [verifying, setVerifying] = useState(false)
  const [verifyMsg, setVerifyMsg] = useState('')
  // P5-4：SSE连接状态
  const [sseConnected, setSseConnected] = useState(false)
  // P5-1/P5-4：轮询定时器引用（SSE回退模式使用）
  const pollTimerRef = useRef<ReturnType<typeof setInterval> | null>(null)
  // P5-4：EventSource引用
  const eventSourceRef = useRef<EventSource | null>(null)
  // P5-4：SSE是否已尝试过（用于判断回退）
  const sseTriedRef = useRef(false)

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

  /** 静默刷新（不显示loading状态） */
  const silentRefresh = useCallback(async () => {
    if (!id) return
    try {
      const data = await getPipelineDetail(id)
      setDetail(data)
    } catch {
      // 静默忽略
    }
  }, [id])

  useEffect(() => { loadDetail() }, [loadDetail])

  // ==================== P5-4：SSE实时推送 + 回退轮询 ====================
  useEffect(() => {
    // 清理函数：关闭SSE和轮询
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

    // 仅running状态需要实时更新
    if (!detail || detail.status !== 'running' || !id) {
      cleanup()
      return cleanup
    }

    // 尝试建立SSE连接
    const connectSSE = () => {
      try {
        const url = SSE_BASE_URL + '/' + id + '/stream'
        const es = new EventSource(url)
        eventSourceRef.current = es

        es.addEventListener('connected', () => {
          setSseConnected(true)
          sseTriedRef.current = true
          // SSE连接成功，停止轮询（如果有的话）
          if (pollTimerRef.current) {
            clearInterval(pollTimerRef.current)
            pollTimerRef.current = null
          }
        })

        es.addEventListener('step_update', () => {
          // 收到步骤更新事件，刷新详情
          silentRefresh()
        })

        es.addEventListener('pipeline_done', () => {
          // Pipeline执行完成，刷新并关闭SSE
          silentRefresh()
          cleanup()
        })

        es.addEventListener('pipeline_error', () => {
          // Pipeline执行失败，刷新并关闭SSE
          silentRefresh()
          cleanup()
        })

        es.onerror = () => {
          // SSE连接错误，回退到轮询
          es.close()
          eventSourceRef.current = null
          setSseConnected(false)
          sseTriedRef.current = true
          startPolling()
        }
      } catch {
        // SSE不可用，回退到轮询
        sseTriedRef.current = true
        startPolling()
      }
    }

    // 回退轮询模式
    const startPolling = () => {
      if (pollTimerRef.current) return // 已有轮询
      pollTimerRef.current = setInterval(silentRefresh, POLL_INTERVAL_MS)
    }

    // 优先尝试SSE
    connectSSE()

    return cleanup
  }, [detail?.status, id, silentRefresh])

  /** P4.6-5：启动验收 */
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

  /** 通用按钮样式 */
  const btn: React.CSSProperties = {
    padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer',
    display: 'inline-flex', alignItems: 'center', gap: 6,
  }

  /** 是否显示审核入口按钮 */
  const showReviewBtn = detail && (
    detail.status === 'review_queue' || detail.status === 'needs_human' || detail.status === 'finalized'
    || detail.status === 'verified' || detail.status === 'verify_failed'
  )

  /** 是否显示"启动验收"按钮 */
  const showVerifyBtn = detail && detail.status === 'finalized' && !verifying

  // 加载中
  if (loading) {
    return <div style={{ textAlign: 'center', padding: 60, color: '#8e8e93' }}>加载中...</div>
  }

  // 错误状态
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
      {/* 顶部导航 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 12, marginBottom: 20 }}>
        <button style={btn} onClick={() => navigate('/pipelines')}>
          <ArrowLeft size={14} /> 返回
        </button>
        <div style={{ flex: 1 }}>
          <div style={{ fontSize: 18, fontWeight: 700, color: '#1c1c1e' }}>
            {detail.course_code} — {detail.course_name}
          </div>
          <div style={{ fontSize: 13, color: '#8e8e93', marginTop: 2 }}>
            状态: {detail.status_name}{detail.review_round > 1 ? ` (第${detail.review_round}审)` : ''} · 当前步骤: {detail.current_step_name}
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
        {/* 验收进行中 */}
        {verifying && (
          <button style={{ ...btn, background: '#5856d6', color: '#fff', border: 'none', opacity: 0.7, cursor: 'not-allowed' }} disabled>
            <RefreshCw size={14} style={{ animation: 'spin 1s linear infinite' }} /> 验收中...
          </button>
        )}
        {/* 审核入口按钮 */}
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
            {detail.status === 'finalized' || detail.status === 'verified' || detail.status === 'verify_failed' ? '查看审核结果' : '进入审核'}
          </button>
        )}
        <button style={btn} onClick={loadDetail}>
          <RefreshCw size={14} /> 刷新
        </button>
      </div>

      {/* P5-4：running状态实时推送横幅 */}
      {detail.status === 'running' && (
        <div style={{
          display: 'flex', alignItems: 'center', gap: 10,
          padding: '10px 16px', borderRadius: 10, marginBottom: 16,
          background: sseConnected ? 'rgba(52,199,89,0.06)' : 'rgba(0,122,255,0.06)',
          border: `1px solid ${sseConnected ? 'rgba(52,199,89,0.15)' : 'rgba(0,122,255,0.15)'}`,
        }}>
          {sseConnected ? (
            <Radio size={16} color="#34c759" />
          ) : (
            <RefreshCw size={16} color="#007aff" style={{ animation: 'spin 2s linear infinite' }} />
          )}
          <div style={{ fontSize: 13, color: sseConnected ? '#34c759' : '#007aff', fontWeight: 500 }}>
            {sseConnected
              ? 'Pipeline正在执行中，实时推送已连接...'
              : 'Pipeline正在执行中，页面每5秒自动刷新...'
            }
          </div>
          <div style={{ fontSize: 12, color: '#8e8e93' }}>
            当前步骤: {detail.current_step_name}
          </div>
          {sseConnected && (
            <span style={{
              marginLeft: 'auto', fontSize: 10, padding: '2px 8px', borderRadius: 10,
              background: 'rgba(52,199,89,0.12)', color: '#34c759', fontWeight: 600,
            }}>SSE</span>
          )}
        </div>
      )}

      {/* 验收操作消息提示 */}
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
            <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 1 }}>
              该Pipeline已通过验收评估，质量达标（≥9.0分）
            </div>
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
        {detail.steps.filter(s => s.has_data || s.status === 'done' || s.status === 'failed').map(step => (
          <StepCard key={step.step_name} pipelineId={detail.id} step={step} />
        ))}
      </div>

      {/* 旋转动画CSS */}
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

function StepCard({ pipelineId, step }: { pipelineId: string; step: StepListItem }) {
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

  const statusColors: Record<string, string> = {
    done: '#34c759', running: '#007aff', failed: '#ff3b30',
    skipped: '#aeaeb2', pending: '#c7c7cc',
  }

  return (
    <div style={{
      background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
      border: '1px solid rgba(0,0,0,0.06)', borderRadius: 14, overflow: 'hidden',
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
        <span style={{
          fontSize: 12, color: '#c7c7cc',
          transition: 'transform 0.2s', transform: expanded ? 'rotate(90deg)' : 'none',
        }}>▶</span>
      </div>

      {expanded && (
        <div style={{ padding: '14px 18px' }}>
          {loadingData ? (
            <div style={{ color: '#8e8e93', fontSize: 13 }}>加载步骤数据...</div>
          ) : stepData ? (
            <StepPanelRouter stepName={step.step_name} data={stepData} />
          ) : step.error_message ? (
            <div style={{ color: '#ff3b30', fontSize: 13, whiteSpace: 'pre-wrap' }}>
              {step.error_message}
            </div>
          ) : (
            <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>
          )}
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

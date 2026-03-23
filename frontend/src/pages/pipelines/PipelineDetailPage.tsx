/**
 * Pipeline详情页面
 * P4.5-B: 重构为主页面 + 6个独立步骤面板组件
 * P4.5-C: 新增审核入口按钮（review_queue/needs_human状态时显示）
 * 7步进度可视化 + StepCard懒加载展开 + 各步骤调试面板
 */
import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { getPipelineDetail, getStepDetail, type PipelineDetailResponse, type StepListItem } from '@/api/pipelines'
import { ArrowLeft, RefreshCw, ClipboardCheck } from 'lucide-react'

// 导入各步骤面板组件
import DbCheckPanel from './steps/DbCheckPanel'
import ScannerPanel from './steps/ScannerPanel'
import EvaluatorPanel from './steps/EvaluatorPanel'
import MetaPanel from './steps/MetaPanel'
import TranslatorPanel from './steps/TranslatorPanel'
import GeneratorPanel from './steps/GeneratorPanel'

// ==================== 主页面组件 ====================

export default function PipelineDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<PipelineDetailResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

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

  useEffect(() => { loadDetail() }, [loadDetail])

  /** 通用按钮样式 */
  const btn: React.CSSProperties = {
    padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer',
    display: 'inline-flex', alignItems: 'center', gap: 6,
  }

  /** 是否显示审核入口按钮 */
  /** 是否显示审核入口按钮（P4.6更新：verified/verify_failed状态也显示） */
  const showReviewBtn = detail && (
    detail.status === 'review_queue' || detail.status === 'needs_human' || detail.status === 'finalized'
    || detail.status === 'verified' || detail.status === 'verify_failed'
  )

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
        {/* P4.5-C: 审核入口按钮 */}
        {showReviewBtn && (
          <button
            style={{
              ...btn,
              background: (detail.status === 'finalized' || detail.status === 'verified') ? '#34c759'
                : detail.status === 'verify_failed' ? '#ff9500' : '#007aff',
              color: '#fff',
              border: 'none',
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

      {/* 7步进度可视化 */}
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
                {/* 进度条段 */}
                <div style={{
                  height: 6, borderRadius: 3, background: color,
                  marginBottom: 8, transition: 'background 0.3s ease',
                }} />
                {/* 步骤信息 */}
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
    </div>
  )
}

// ==================== 步骤详情卡片 ====================

/** 步骤卡片 — 支持懒加载展开，展开后渲染对应的步骤面板组件 */
function StepCard({ pipelineId, step }: { pipelineId: string; step: StepListItem }) {
  const [expanded, setExpanded] = useState(false)
  const [stepData, setStepData] = useState<any>(null)
  const [loadingData, setLoadingData] = useState(false)

  /** 点击展开/折叠 */
  const toggleExpand = async () => {
    if (expanded) { setExpanded(false); return }
    setExpanded(true)
    // 懒加载：展开时才请求步骤详情
    if (!stepData && step.has_data) {
      setLoadingData(true)
      try {
        const data = await getStepDetail(pipelineId, step.step_name)
        setStepData(data.step_data)
      } catch { /* 静默处理 */ }
      setLoadingData(false)
    }
  }

  /** 状态颜色 */
  const statusColors: Record<string, string> = {
    done: '#34c759', running: '#007aff', failed: '#ff3b30',
    skipped: '#aeaeb2', pending: '#c7c7cc',
  }

  return (
    <div style={{
      background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
      border: '1px solid rgba(0,0,0,0.06)', borderRadius: 14, overflow: 'hidden',
    }}>
      {/* 卡片头部（可点击展开） */}
      <div onClick={toggleExpand} style={{
        display: 'flex', alignItems: 'center', gap: 12, padding: '14px 18px', cursor: 'pointer',
        borderBottom: expanded ? '1px solid rgba(0,0,0,0.04)' : 'none',
      }}>
        {/* 状态圆点 */}
        <div style={{
          width: 8, height: 8, borderRadius: 4,
          background: statusColors[step.status] || '#c7c7cc', flexShrink: 0,
        }} />
        {/* 步骤名称和状态 */}
        <div style={{ flex: 1 }}>
          <span style={{ fontSize: 14, fontWeight: 600, color: '#1c1c1e' }}>
            {step.step_order}. {step.step_name_cn}
          </span>
          <span style={{ fontSize: 12, color: '#8e8e93', marginLeft: 10 }}>{step.status_name}</span>
        </div>
        {/* 耗时 */}
        {step.duration_ms > 0 && (
          <span style={{ fontSize: 12, color: '#aeaeb2' }}>
            {step.duration_ms < 1000 ? step.duration_ms + 'ms' : (step.duration_ms / 1000).toFixed(1) + 's'}
          </span>
        )}
        {/* Token */}
        {step.tokens_used > 0 && (
          <span style={{ fontSize: 12, color: '#aeaeb2' }}>
            {step.tokens_used.toLocaleString()} tokens
          </span>
        )}
        {/* 展开箭头 */}
        <span style={{
          fontSize: 12, color: '#c7c7cc',
          transition: 'transform 0.2s', transform: expanded ? 'rotate(90deg)' : 'none',
        }}>▶</span>
      </div>

      {/* 展开的详情内容 */}
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

/** 根据步骤名称分发到对应的面板组件 */
function StepPanelRouter({ stepName, data }: { stepName: string; data: any }) {
  if (!data) return <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>

  switch (stepName) {
    case 'dbCheck':    return <DbCheckPanel data={data} />
    case 'scanner':    return <ScannerPanel data={data} />
    case 'evaluator':  return <EvaluatorPanel data={data} />
    case 'meta':       return <MetaPanel data={data} />
    case 'translator': return <TranslatorPanel data={data} />
    case 'generator':  return <GeneratorPanel data={data} />
    case 'verify':
      return (
        <pre style={{
          fontSize: 12, color: '#3c3c43', overflow: 'auto', maxHeight: 400,
          margin: 0, whiteSpace: 'pre-wrap',
        }}>
          {JSON.stringify(data, null, 2)}
        </pre>
      )
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

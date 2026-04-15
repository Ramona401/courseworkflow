/**
 * HistoryRoundPanel — 历史轮次查看面板 + 全流程时间线
 *
 * v69新增（编号12）：PipelineFlowTimeline 组件
 *   - 在Pipeline详情页顶部展示完整的1审→验收→2审→验收流程时间线
 *   - 每个节点显示状态（通过/未通过/进行中/待执行）和关键分数
 *   - 只在有验收记录或review_round>=2时显示
 *
 * 原有功能：
 *   - 显示可用历史轮次选项卡
 *   - 展示该轮次的步骤执行情况（状态/耗时/token）
 *   - 展示该轮次的评估分数
 *   - 只读，不可操作
 */
import { useState, useEffect } from 'react'
import { getPipelineAvailableRounds, getPipelineHistory } from '@/api/pipelines'
import type { PipelineDetailResponse, HistoryStepItem, HistoryEvalRound } from '@/api/pipelines'
import { ChevronDown, ChevronUp, CheckCircle, XCircle, Clock, ArrowRight, Loader, Minus } from 'lucide-react'

// ==================== 全流程时间线组件（v69新增，编号12）====================

/**
 * PipelineFlowTimeline 全流程时间线
 *
 * 展示完整的1审→验收1→2审→验收2流程概览
 * 每个阶段显示：状态图标 + 阶段名 + 关键分数 + 耗时
 *
 * 显示条件：
 *   - review_round >= 2（已进入或完成2审）
 *   - 或当前状态为 verified/verify_failed/published（有验收结果）
 *   - 或当前状态为 finalized 且 review_round >= 1（可以触发验收）
 */
export function PipelineFlowTimeline({ detail }: { detail: PipelineDetailResponse }) {
  // 判断是否需要显示时间线
  const hasVerifyHistory = detail.status === 'verified' || detail.status === 'verify_failed' || detail.status === 'published'
  const isMultiRound = detail.review_round >= 2
  const showTimeline = isMultiRound || hasVerifyHistory || detail.status === 'finalized'

  if (!showTimeline) return null

  // 构建时间线节点
  const nodes: TimelineNode[] = []

  // === 节点1：1审 ===
  const round1Status = getRound1Status(detail)
  const evalScore = getEvalScore(detail)
  const metaScore = getMetaScore(detail)
  nodes.push({
    label: '1审',
    sublabel: '评估+优化+生成',
    status: round1Status,
    score: metaScore || evalScore,
    scoreLabel: metaScore ? 'Meta' : 'Eval',
  })

  // === 节点2：验收1 ===
  const verify1Status = getVerify1Status(detail)
  const verify1Score = getVerifyScore(detail, 1)
  nodes.push({
    label: '验收',
    sublabel: detail.review_round >= 2 ? '1审验收' : '质量验收',
    status: verify1Status,
    score: verify1Score,
    scoreLabel: '验收分',
  })

  // === 节点3+4：2审 + 验收2（仅review_round>=2时显示）===
  if (isMultiRound) {
    const round2Status = getRound2Status(detail)
    nodes.push({
      label: '2审',
      sublabel: '增量优化+生成',
      status: round2Status,
    })

    const verify2Status = getVerify2Status(detail)
    const verify2Score = getVerifyScore(detail, 2)
    nodes.push({
      label: '验收',
      sublabel: '2审验收',
      status: verify2Status,
      score: verify2Score,
      scoreLabel: '验收分',
    })
  }

  return (
    <div style={{
      background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
      border: '1px solid rgba(88,86,214,0.15)', borderRadius: 16,
      padding: '20px 24px', marginBottom: 20,
    }}>
      <div style={{ fontSize: 14, fontWeight: 600, color: '#5856d6', marginBottom: 16, display: 'flex', alignItems: 'center', gap: 8 }}>
        <Clock size={16} /> 全流程概览
        {detail.review_round >= 2 && (
          <span style={{ fontSize: 11, padding: '2px 8px', borderRadius: 10, background: 'rgba(255,149,0,0.1)', color: '#ff9500', fontWeight: 600 }}>
            当前第{detail.review_round}审
          </span>
        )}
      </div>

      {/* 时间线横向布局 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 0 }}>
        {nodes.map((node, i) => (
          <div key={i} style={{ display: 'flex', alignItems: 'center', flex: 1 }}>
            {/* 节点 */}
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', minWidth: 100, flex: 1 }}>
              {/* 状态图标 */}
              <div style={{
                width: 40, height: 40, borderRadius: 20,
                display: 'flex', alignItems: 'center', justifyContent: 'center',
                background: getNodeBackground(node.status),
                border: `2px solid ${getNodeBorderColor(node.status)}`,
                marginBottom: 8,
              }}>
                {getNodeIcon(node.status)}
              </div>
              {/* 阶段名 */}
              <div style={{ fontSize: 13, fontWeight: 600, color: getNodeTextColor(node.status), textAlign: 'center' }}>
                {node.label}
              </div>
              <div style={{ fontSize: 10, color: '#aeaeb2', marginTop: 2, textAlign: 'center' }}>
                {node.sublabel}
              </div>
              {/* 分数（如有） */}
              {node.score !== undefined && node.score > 0 && (
                <div style={{
                  marginTop: 4, fontSize: 12, fontWeight: 600,
                  color: node.score >= 9.0 ? '#34c759' : node.score >= 7.0 ? '#ff9500' : '#ff3b30',
                }}>
                  {node.scoreLabel}: {node.score.toFixed(1)}
                </div>
              )}
            </div>
            {/* 连接箭头（最后一个节点后不显示） */}
            {i < nodes.length - 1 && (
              <div style={{ display: 'flex', alignItems: 'center', padding: '0 4px', marginBottom: 20 }}>
                <div style={{ width: 24, height: 2, background: i < nodes.length - 1 && nodes[i].status === 'done' ? '#34c759' : '#e5e5ea' }} />
                <ArrowRight size={14} color={nodes[i].status === 'done' ? '#34c759' : '#e5e5ea'} />
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

// ==================== 时间线辅助类型和函数 ====================

interface TimelineNode {
  label: string
  sublabel: string
  status: 'done' | 'failed' | 'running' | 'pending' | 'skipped'
  score?: number
  scoreLabel?: string
}

/** 获取1审状态 */
function getRound1Status(detail: PipelineDetailResponse): TimelineNode['status'] {
  if (detail.review_round >= 2) return 'done' // 已进入2审说明1审已完成
  // 1审中，根据当前pipeline状态判断
  if (detail.status === 'running') return 'running'
  if (detail.status === 'failed' || detail.status === 'cancelled') return 'failed'
  if (detail.status === 'review_queue' || detail.status === 'pending_finalize' || detail.status === 'needs_human') return 'done'
  if (detail.status === 'finalized' || detail.status === 'verified' || detail.status === 'verify_failed' || detail.status === 'published') return 'done'
  return 'pending'
}

/** 获取验收1状态 */
function getVerify1Status(detail: PipelineDetailResponse): TimelineNode['status'] {
  if (detail.review_round >= 2) {
    // 已进入2审说明验收1未通过
    return 'failed'
  }
  if (detail.status === 'verified' || detail.status === 'published') return 'done'
  if (detail.status === 'verify_failed') return 'failed'
  if (detail.status === 'finalized') return 'pending'
  return 'pending'
}

/** 获取2审状态 */
function getRound2Status(detail: PipelineDetailResponse): TimelineNode['status'] {
  if (detail.review_round < 2) return 'pending'
  if (detail.status === 'running') return 'running'
  if (detail.status === 'failed') return 'failed'
  if (detail.status === 'review_queue' || detail.status === 'pending_finalize' || detail.status === 'needs_human') return 'done'
  if (detail.status === 'finalized' || detail.status === 'verified' || detail.status === 'verify_failed' || detail.status === 'published') return 'done'
  return 'running'
}

/** 获取验收2状态 */
function getVerify2Status(detail: PipelineDetailResponse): TimelineNode['status'] {
  if (detail.review_round < 2) return 'pending'
  if (detail.status === 'verified' || detail.status === 'published') return 'done'
  if (detail.status === 'verify_failed') return 'failed'
  if (detail.status === 'finalized') return 'pending'
  if (detail.status === 'needs_human') return 'failed'
  return 'pending'
}

/** 从detail步骤中提取Evaluator均分 */
function getEvalScore(detail: PipelineDetailResponse): number | undefined {
  const evalStep = detail.steps.find(s => s.step_name === 'evaluator')
  if (!evalStep || !evalStep.has_data) return undefined
  // 分数在step_data中，但详情页列表只有has_data标记没有实际数据
  // 这里只能返回undefined，时间线不显示分数
  return undefined
}

/** 从detail步骤中提取Meta分数 */
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function getMetaScore(detail: PipelineDetailResponse): number | undefined {
  // 同理，列表级别没有分数数据
  return undefined
}

/** 获取验收分数（需要从verify步骤获取，但列表级别无数据） */
// eslint-disable-next-line @typescript-eslint/no-unused-vars
function getVerifyScore(_detail: PipelineDetailResponse, _round: number): number | undefined {
  return undefined
}

/** 节点背景色 */
function getNodeBackground(status: TimelineNode['status']): string {
  switch (status) {
    case 'done': return 'rgba(52,199,89,0.1)'
    case 'failed': return 'rgba(255,59,48,0.1)'
    case 'running': return 'rgba(0,122,255,0.1)'
    default: return 'rgba(0,0,0,0.03)'
  }
}

/** 节点边框色 */
function getNodeBorderColor(status: TimelineNode['status']): string {
  switch (status) {
    case 'done': return '#34c759'
    case 'failed': return '#ff3b30'
    case 'running': return '#007aff'
    default: return '#e5e5ea'
  }
}

/** 节点文字色 */
function getNodeTextColor(status: TimelineNode['status']): string {
  switch (status) {
    case 'done': return '#34c759'
    case 'failed': return '#ff3b30'
    case 'running': return '#007aff'
    default: return '#aeaeb2'
  }
}

/** 节点图标 */
function getNodeIcon(status: TimelineNode['status']) {
  switch (status) {
    case 'done': return <CheckCircle size={20} color="#34c759" />
    case 'failed': return <XCircle size={20} color="#ff3b30" />
    case 'running': return <Loader size={20} color="#007aff" style={{ animation: 'spin 2s linear infinite' }} />
    default: return <Minus size={20} color="#e5e5ea" />
  }
}

// ==================== 以下为原有 HistoryRoundPanel 代码 ====================

// ==================== 步骤状态颜色 ====================
const STATUS_COLORS: Record<string, string> = {
  done: '#34c759', running: '#007aff', failed: '#ff3b30',
  skipped: '#aeaeb2', pending: '#c7c7cc',
}

// ==================== 评估分数显示 ====================
function ScoreCell({ value, label }: { value: number | null; label: string }) {
  if (value === null || value === undefined) return (
    <span style={{ fontSize: 12, color: '#c7c7cc' }}>{label}: -</span>
  )
  const color = value >= 9.0 ? '#34c759' : value >= 7.0 ? '#ff9500' : '#ff3b30'
  return (
    <span style={{ fontSize: 12, color: '#8e8e93' }}>
      {label}: <span style={{ color, fontWeight: 600 }}>{value.toFixed(2)}</span>
    </span>
  )
}

// ==================== 单步骤行 ====================
function HistoryStepRow({ step }: { step: HistoryStepItem }) {
  const [expanded, setExpanded] = useState(false)
  const color = STATUS_COLORS[step.status] || '#c7c7cc'

  return (
    <div style={{ borderBottom: '1px solid rgba(0,0,0,0.04)' }}>
      <div
        onClick={() => step.has_data && setExpanded(p => !p)}
        style={{
          display: 'flex', alignItems: 'center', gap: 10,
          padding: '10px 16px',
          cursor: step.has_data ? 'pointer' : 'default',
          background: expanded ? 'rgba(0,0,0,0.02)' : 'transparent',
        }}
        onMouseEnter={e => { if (step.has_data) (e.currentTarget as HTMLElement).style.background = 'rgba(0,0,0,0.02)' }}
        onMouseLeave={e => { if (!expanded) (e.currentTarget as HTMLElement).style.background = 'transparent' }}
      >
        {/* 状态圆点 */}
        <div style={{ width: 8, height: 8, borderRadius: 4, background: color, flexShrink: 0 }} />

        {/* 步骤名 */}
        <div style={{ flex: 1, fontSize: 13, fontWeight: 500, color: '#1c1c1e' }}>
          {step.step_order}. {step.step_name_cn}
          <span style={{ marginLeft: 8, fontSize: 11, color: '#8e8e93', fontWeight: 400 }}>
            {step.status_name}
          </span>
        </div>

        {/* 耗时 */}
        {step.duration_ms > 0 && (
          <span style={{ fontSize: 11, color: '#aeaeb2' }}>
            {step.duration_ms < 1000 ? step.duration_ms + 'ms' : (step.duration_ms / 1000).toFixed(1) + 's'}
          </span>
        )}

        {/* Token */}
        {step.tokens_used > 0 && (
          <span style={{ fontSize: 11, color: '#aeaeb2' }}>
            {step.tokens_used.toLocaleString()} tokens
          </span>
        )}

        {/* 展开箭头 */}
        {step.has_data && (
          <span style={{ color: '#c7c7cc', flexShrink: 0 }}>
            {expanded ? <ChevronUp size={14} /> : <ChevronDown size={14} />}
          </span>
        )}
      </div>

      {/* 错误信息 */}
      {step.error_message && (
        <div style={{
          margin: '0 16px 8px', padding: '8px 12px', borderRadius: 8,
          background: 'rgba(255,59,48,0.06)', border: '1px solid rgba(255,59,48,0.15)',
          fontSize: 12, color: '#ff3b30', whiteSpace: 'pre-wrap',
        }}>
          {step.error_message}
        </div>
      )}

      {/* 展开的步骤数据 */}
      {expanded && step.step_data && (
        <div style={{
          margin: '0 16px 10px', padding: '10px 14px', borderRadius: 8,
          background: '#f9f9f9', border: '1px solid rgba(0,0,0,0.06)',
        }}>
          <pre style={{
            fontSize: 11, color: '#3c3c43', margin: 0,
            overflow: 'auto', maxHeight: 300,
            whiteSpace: 'pre-wrap', wordBreak: 'break-word',
            fontFamily: '"Fira Code", Consolas, monospace',
          }}>
            {typeof step.step_data === 'string'
              ? (() => {
                  try {
                    return JSON.stringify(JSON.parse(step.step_data), null, 2)
                  } catch {
                    return step.step_data
                  }
                })()
              : JSON.stringify(step.step_data, null, 2)
            }
          </pre>
        </div>
      )}
    </div>
  )
}

// ==================== 评估轮次摘要 ====================
function EvalRoundSummary({ rounds }: { rounds: HistoryEvalRound[] }) {
  if (rounds.length === 0) return null
  const avg = rounds.reduce((s, r) => s + (r.score_total || 0), 0) / rounds.length

  return (
    <div style={{
      margin: '12px 16px', padding: '12px 16px', borderRadius: 10,
      background: 'rgba(0,122,255,0.04)', border: '1px solid rgba(0,122,255,0.12)',
    }}>
      <div style={{ fontSize: 12, fontWeight: 600, color: '#007aff', marginBottom: 8 }}>
        评估结果（{rounds.length} 轮）· 均分 {avg.toFixed(2)}
      </div>
      <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
        {rounds.map(r => (
          <div key={r.id} style={{ display: 'flex', alignItems: 'center', gap: 16, fontSize: 12 }}>
            <span style={{ color: '#8e8e93', minWidth: 40 }}>第{r.round_number}轮</span>
            <ScoreCell value={r.score_total} label="总分" />
            <ScoreCell value={r.score_e1} label="E1" />
            <ScoreCell value={r.score_e2} label="E2" />
            <ScoreCell value={r.score_e3} label="E3" />
            <ScoreCell value={r.score_e4} label="E4" />
            <span style={{ color: '#aeaeb2', fontSize: 11 }}>
              {r.tokens_used.toLocaleString()} tokens
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

// ==================== 历史轮次面板主组件 ====================
export function HistoryRoundPanel({ pipelineId, currentRound }: { pipelineId: string; currentRound: number }) {
  const [expanded, setExpanded]           = useState(false)
  const [availableRounds, setAvailableRounds] = useState<number[]>([])
  const [selectedRound, setSelectedRound] = useState<number | null>(null)
  const [loading, setLoading]             = useState(false)
  const [steps, setSteps]                 = useState<HistoryStepItem[]>([])
  const [evalRounds, setEvalRounds]       = useState<HistoryEvalRound[]>([])

  const historyRounds = availableRounds.filter(r => r < currentRound)

  // 加载可用轮次列表
  useEffect(() => {
    if (currentRound < 2) return
    getPipelineAvailableRounds(pipelineId)
      .then(rounds => {
        setAvailableRounds(rounds)
        const hist = rounds.filter(r => r < currentRound)
        if (hist.length > 0) setSelectedRound(hist[hist.length - 1])
      })
      .catch(() => {})
  }, [pipelineId, currentRound])

  // 加载选中轮次的数据
  useEffect(() => {
    if (!selectedRound || !expanded) return
     
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setLoading(true)
    getPipelineHistory(pipelineId, selectedRound)
      .then(data => {
        setSteps(data.steps || [])
        setEvalRounds(data.eval_rounds || [])
      })
      .catch(() => {})
      .finally(() => setLoading(false))
  }, [pipelineId, selectedRound, expanded])

  if (currentRound < 2 || historyRounds.length === 0) return null

  return (
    <div style={{
      background: 'rgba(255,255,255,0.72)', backdropFilter: 'blur(20px)',
      border: '1px solid rgba(88,86,214,0.15)',
      borderRadius: 14, overflow: 'hidden', marginTop: 12,
    }}>
      {/* 标题栏 */}
      <div
        onClick={() => setExpanded(p => !p)}
        style={{
          display: 'flex', alignItems: 'center', gap: 10,
          padding: '14px 18px', cursor: 'pointer',
          background: expanded ? 'rgba(88,86,214,0.04)' : 'transparent',
          borderBottom: expanded ? '1px solid rgba(88,86,214,0.1)' : 'none',
          transition: 'background 0.15s ease',
        }}
        onMouseEnter={e => { (e.currentTarget as HTMLElement).style.background = 'rgba(88,86,214,0.04)' }}
        onMouseLeave={e => { if (!expanded) (e.currentTarget as HTMLElement).style.background = 'transparent' }}
      >
        <span style={{ fontSize: 16 }}>🕐</span>
        <span style={{ fontSize: 14, fontWeight: 600, color: '#5856d6', flex: 1 }}>
          历史轮次详细数据
        </span>
        <span style={{
          fontSize: 11, padding: '2px 8px', borderRadius: 10, fontWeight: 600,
          background: 'rgba(88,86,214,0.1)', color: '#5856d6',
        }}>
          {historyRounds.length} 轮可查
        </span>
        <span style={{ color: '#c7c7cc' }}>
          {expanded ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
        </span>
      </div>

      {/* 展开内容 */}
      {expanded && (
        <div>
          {/* 轮次选择 Tab */}
          {historyRounds.length > 1 && (
            <div style={{
              display: 'flex', gap: 4, padding: '10px 16px',
              borderBottom: '1px solid rgba(0,0,0,0.04)',
            }}>
              {historyRounds.map(r => (
                <button
                  key={r}
                  onClick={() => setSelectedRound(r)}
                  style={{
                    padding: '5px 14px', borderRadius: 8, border: 'none',
                    fontSize: 12, fontWeight: 500, cursor: 'pointer',
                    background: selectedRound === r ? '#5856d6' : 'rgba(0,0,0,0.05)',
                    color: selectedRound === r ? '#fff' : '#8e8e93',
                    transition: 'all 150ms ease',
                  }}
                >
                  第{r}审
                </button>
              ))}
            </div>
          )}

          {/* 数据内容 */}
          {loading ? (
            <div style={{ padding: '24px', textAlign: 'center', color: '#8e8e93', fontSize: 13 }}>
              加载历史数据...
            </div>
          ) : (
            <div>
              <EvalRoundSummary rounds={evalRounds} />
              <div style={{ padding: '8px 0' }}>
                {steps
                  .filter(s => s.status !== 'pending')
                  .map(step => (
                    <HistoryStepRow key={step.step_name} step={step} />
                  ))
                }
                {steps.filter(s => s.status !== 'pending').length === 0 && (
                  <div style={{ padding: '16px', textAlign: 'center', color: '#aeaeb2', fontSize: 13 }}>
                    该轮次暂无执行数据
                  </div>
                )}
              </div>
              <div style={{
                padding: '10px 16px', fontSize: 11, color: '#aeaeb2',
                borderTop: '1px solid rgba(0,0,0,0.04)',
              }}>
                📋 以上为第{selectedRound}审的历史数据，仅供参考对照，不可操作
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  )
}

/**
 * Evaluator步骤调试面板
 * P4.5-B: 展示多轮评估分数、方差、各维度均值 + 各轮评估详情（含AI原始输出）
 */
import { useState, useEffect } from 'react'
import { useParams } from 'react-router-dom'
import { getEvalRounds, type EvalRoundDetail } from '@/api/pipelines'
import {
  kvRow, kvLabel, kvValue, passStyle, failStyle,
  ScoreCardRow, TokenDuration, PromptInfo, SectionTitle, scoreColor,
  AIOutputViewer,
} from './StepCommon'

/** Evaluator面板属性 */
interface EvaluatorPanelProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  data: any
}

/** Evaluator步骤调试面板 */
export default function EvaluatorPanel({ data }: EvaluatorPanelProps) {
  if (!data) return <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>

  return (
    <div>
      {/* 分数总览卡片 */}
      <ScoreCardRow items={[
        { label: '综合均分', value: data.avg_total?.toFixed(2), color: scoreColor(data.avg_total) },
        { label: 'E1 难度适配', value: data.avg_e1?.toFixed(2) },
        { label: 'E2 时间节奏', value: data.avg_e2?.toFixed(2) },
        { label: 'E3 互动评估', value: data.avg_e3?.toFixed(2) },
        { label: 'E4 课程设计', value: data.avg_e4?.toFixed(2) },
      ]} minWidth={80} />

      {/* 轮次信息 */}
      <div style={kvRow}>
        <span style={kvLabel}>轮次统计</span>
        <span style={kvValue}>
          {data.done_rounds} 完成 / {data.total_rounds} 总轮
          {data.failed_rounds > 0 && <span style={failStyle}> ({data.failed_rounds} 失败)</span>}
        </span>
      </div>

      {/* 各轮分数badge */}
      <div style={kvRow}>
        <span style={kvLabel}>各轮分数</span>
        <span style={kvValue}>
          {(data.round_scores || []).map((s: number, i: number) => (
            <span key={i} style={{
              display: 'inline-block', padding: '2px 8px', borderRadius: 10,
              background: scoreColor(s) === '#34c759' ? 'rgba(52,199,89,0.1)' : scoreColor(s) === '#ff9500' ? 'rgba(255,149,0,0.1)' : 'rgba(255,59,48,0.1)',
              color: scoreColor(s), fontSize: 12, fontWeight: 600, marginRight: 6, marginBottom: 4,
            }}>
              R{i + 1}: {s.toFixed(2)}
            </span>
          ))}
        </span>
      </div>

      {/* 方差 */}
      <div style={kvRow}>
        <span style={kvLabel}>方差</span>
        <span style={kvValue}>
          {data.variance?.toFixed(4) ?? '-'}
          {data.variance_warn && <span style={{ ...failStyle, marginLeft: 8 }}>⚠️ 超过阈值</span>}
        </span>
      </div>

      {/* 模型和Token */}
      <div style={kvRow}>
        <span style={kvLabel}>模型</span>
        <span style={kvValue}>{data.model_used || '-'}</span>
      </div>
      <TokenDuration tokens={data.total_tokens} durationMs={data.total_latency_ms} />
      <PromptInfo data={data} />

      {/* 各轮评估详情（从eval_rounds表加载） */}
      <EvalRoundsSection />
    </div>
  )
}

// ==================== 各轮评估详情区块 ====================

/** 各轮评估详情区块 — 从API加载eval_rounds数据 */
function EvalRoundsSection() {
  const { id: pipelineId } = useParams<{ id: string }>()
  const [rounds, setRounds] = useState<EvalRoundDetail[]>([])
  const [loading, setLoading] = useState(false)
  const [loaded, setLoaded] = useState(false)
  const [error, setError] = useState('')

  /** 加载评估轮次数据 */
  const loadRounds = async () => {
    if (!pipelineId || loaded) return
    setLoading(true)
    setError('')
    try {
      const data = await getEvalRounds(pipelineId)
      setRounds(data || [])
      setLoaded(true)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    } catch (e: any) {
      setError(e.message || '加载失败')
    }
    setLoading(false)
  }

  // 自动加载
  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { loadRounds() }, [pipelineId])

  if (loading) {
    return (
      <div style={{ marginTop: 16 }}>
        <SectionTitle title="各轮评估报告" />
        <div style={{ color: '#8e8e93', fontSize: 13 }}>加载评估轮次数据...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div style={{ marginTop: 16 }}>
        <SectionTitle title="各轮评估报告" />
        <div style={{ color: '#ff3b30', fontSize: 13 }}>{error}</div>
      </div>
    )
  }

  if (rounds.length === 0) return null

  return (
    <div style={{ marginTop: 16 }}>
      <SectionTitle title={`各轮评估报告 (${rounds.length}轮)`} />
      {rounds.map((round) => (
        <EvalRoundCard key={round.id} round={round} />
      ))}
    </div>
  )
}

// ==================== 单轮评估卡片 ====================

/** 单轮评估详情卡片 */
function EvalRoundCard({ round }: { round: EvalRoundDetail }) {
  const [expanded, setExpanded] = useState(false)

  const isDone = round.status === 'done'
  const isFailed = round.status === 'failed'

  return (
    <div style={{
      border: '1px solid rgba(0,0,0,0.04)', borderRadius: 10,
      marginBottom: 8, overflow: 'hidden',
    }}>
      {/* 卡片头部 */}
      <div
        onClick={() => setExpanded(!expanded)}
        style={{
          display: 'flex', alignItems: 'center', gap: 10, padding: '10px 14px',
          cursor: 'pointer', background: expanded ? 'rgba(0,0,0,0.01)' : 'transparent',
        }}
      >
        {/* 展开箭头 */}
        <span style={{
          fontSize: 10, color: '#c7c7cc', transition: 'transform 0.2s',
          transform: expanded ? 'rotate(90deg)' : 'none',
        }}>▶</span>

        {/* 轮次编号 */}
        <span style={{ fontSize: 13, fontWeight: 600, color: '#1c1c1e', minWidth: 50 }}>
          第{round.round_number}轮
        </span>

        {/* 状态 */}
        <span style={{
          fontSize: 11, fontWeight: 500,
          color: isDone ? '#34c759' : isFailed ? '#ff3b30' : '#8e8e93',
        }}>
          {isDone ? '完成' : isFailed ? '失败' : round.status}
        </span>

        {/* 分数 */}
        {round.score_total !== null && round.score_total !== undefined && (
          <span style={{
            fontSize: 13, fontWeight: 700,
            color: scoreColor(round.score_total),
            marginLeft: 8,
          }}>
            {round.score_total.toFixed(2)}
          </span>
        )}

        {/* 维度分数摘要 */}
        {isDone && (
          <div style={{ display: 'flex', gap: 8, fontSize: 11, color: '#8e8e93', marginLeft: 'auto' }}>
            {round.score_e1 !== null && <span>E1:{round.score_e1.toFixed(1)}</span>}
            {round.score_e2 !== null && <span>E2:{round.score_e2.toFixed(1)}</span>}
            {round.score_e3 !== null && <span>E3:{round.score_e3.toFixed(1)}</span>}
            {round.score_e4 !== null && <span>E4:{round.score_e4.toFixed(1)}</span>}
            {round.hard_constraint && (
              <span style={round.hard_constraint === 'PASS' ? passStyle : failStyle}>
                {round.hard_constraint}
              </span>
            )}
            {round.grade && <span>评级:{round.grade}</span>}
          </div>
        )}

        {/* Token */}
        {round.tokens_used > 0 && (
          <span style={{ fontSize: 11, color: '#aeaeb2', marginLeft: 8 }}>
            {round.tokens_used.toLocaleString()} tokens
          </span>
        )}
      </div>

      {/* 展开的详情 */}
      {expanded && (
        <div style={{ padding: '10px 14px', borderTop: '1px solid rgba(0,0,0,0.04)' }}>
          {/* 详细分数 */}
          {isDone && (
            <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginBottom: 12 }}>
              {[
                { label: '综合', value: round.score_total, color: scoreColor(round.score_total ?? 0) },
                { label: 'E1', value: round.score_e1 },
                { label: 'E2', value: round.score_e2 },
                { label: 'E3', value: round.score_e3 },
                { label: 'E4', value: round.score_e4 },
              ].map(item => (
                <div key={item.label} style={{ textAlign: 'center', minWidth: 50 }}>
                  <div style={{ fontSize: 10, color: '#8e8e93' }}>{item.label}</div>
                  <div style={{ fontSize: 16, fontWeight: 700, color: item.color || '#1c1c1e' }}>
                    {item.value !== null && item.value !== undefined ? item.value.toFixed(2) : '-'}
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* 硬性约束和评级 */}
          {(round.hard_constraint || round.grade) && (
            <div style={{ display: 'flex', gap: 16, marginBottom: 8, fontSize: 12 }}>
              {round.hard_constraint && (
                <span>硬性约束: <span style={round.hard_constraint === 'PASS' ? passStyle : failStyle}>{round.hard_constraint}</span></span>
              )}
              {round.grade && <span>评级: {round.grade}</span>}
            </div>
          )}

          {/* 模型 */}
          {round.model_used && (
            <div style={{ fontSize: 12, color: '#8e8e93', marginBottom: 8 }}>
              模型: {round.model_used}
            </div>
          )}

          {/* AI原始评估报告 */}
          <AIOutputViewer
            output={round.output}
            label={`第${round.round_number}轮评估报告（Prompt B 完整输出）`}
            maxPreviewLen={500}
          />
        </div>
      )}
    </div>
  )
}

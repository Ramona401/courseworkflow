/**
 * Translator步骤调试面板
 * P4.5-B: 展示Translator+Reviewer循环详情、最终评分、各轮评分、AI输出
 */
import { useState } from 'react'
import {
  kvRow, kvLabel, kvValue, passStyle, failStyle,
  TokenDuration, AIOutputViewer, PromptInfo, SectionTitle, scoreColor,
} from './StepCommon'

/** Translator面板属性 */
interface TranslatorPanelProps {
  data: any
}

/** Translator步骤调试面板 */
export default function TranslatorPanel({ data }: TranslatorPanelProps) {
  if (!data) return <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>

  return (
    <div>
      {/* 最终结果总览 */}
      <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginBottom: 12, alignItems: 'center' }}>
        <div style={{ textAlign: 'center', minWidth: 90 }}>
          <div style={{ fontSize: 11, color: '#8e8e93', marginBottom: 2 }}>最终评分</div>
          <div style={{ fontSize: 24, fontWeight: 700, color: data.passed ? '#34c759' : '#ff3b30' }}>
            {data.final_score?.toFixed(2) ?? '-'}
          </div>
        </div>
        <div>
          <div style={{ fontSize: 12 }}>
            QUALITY_GATE:{' '}
            <span style={data.final_quality_gate === 'PASS' ? passStyle : failStyle}>
              {data.final_quality_gate || '-'}
            </span>
          </div>
          <div style={{ fontSize: 12, marginTop: 2 }}>
            评级: {data.final_grade || '-'} · 第{data.final_round ?? '-'}轮结束
          </div>
        </div>
        <div style={{ marginLeft: 'auto', textAlign: 'right' }}>
          <div style={{ fontSize: 12, color: '#8e8e93' }}>
            通过: {data.passed ? <span style={passStyle}>是</span> : <span style={failStyle}>否</span>}
          </div>
          <div style={{ fontSize: 12, color: '#aeaeb2', marginTop: 2 }}>
            阈值: {data.threshold ?? 9.0} · 最大轮次: {data.max_loops ?? '-'}
          </div>
        </div>
      </div>

      {/* 模型/Token信息 */}
      <div style={kvRow}>
        <span style={kvLabel}>模型</span>
        <span style={kvValue}>{data.model_used || '-'}</span>
      </div>
      <TokenDuration tokens={data.total_tokens} durationMs={data.total_latency_ms} />
      <PromptInfo data={data} />

      {/* 各轮循环详情 */}
      {(data.rounds || []).length > 0 && (
        <>
          <SectionTitle title="Translator-Reviewer 循环详情" />
          {(data.rounds || []).map((r: any) => (
            <RoundDetail key={r.round} round={r} />
          ))}
        </>
      )}

      {/* 最终Translator输出查看器 */}
      <AIOutputViewer output={data.final_trans_output} label="最终Translator输出（Prompt C）" />

      {/* 最终Reviewer输出查看器 */}
      <AIOutputViewer output={data.final_review_output} label="最终Reviewer输出（Prompt D）" />
    </div>
  )
}

// ==================== 单轮详情组件 ====================

/** 单轮Translator-Reviewer详情 */
function RoundDetail({ round }: { round: any }) {
  const [expanded, setExpanded] = useState(false)

  return (
    <div style={{
      padding: '8px 0', borderTop: '1px solid rgba(0,0,0,0.04)',
    }}>
      {/* 轮次头部（可点击展开AI输出） */}
      <div
        onClick={() => setExpanded(!expanded)}
        style={{ cursor: 'pointer', display: 'flex', alignItems: 'center', gap: 8 }}
      >
        <span style={{
          transition: 'transform 0.2s', transform: expanded ? 'rotate(90deg)' : 'none',
          fontSize: 10, color: '#c7c7cc',
        }}>▶</span>
        <div style={{ flex: 1 }}>
          <span style={{ fontSize: 13, fontWeight: 600, color: '#1c1c1e' }}>
            第{round.round}轮
          </span>
          <span style={{ marginLeft: 8, fontSize: 12 }}>
            {round.passed ? <span style={passStyle}>通过</span> : <span style={failStyle}>未通过</span>}
          </span>
        </div>
        {/* 分数摘要 */}
        <div style={{ display: 'flex', gap: 10, fontSize: 12, color: '#8e8e93' }}>
          <span>总分: <span style={{ color: scoreColor(round.score), fontWeight: 600 }}>{round.score?.toFixed(2)}</span></span>
          <span>E1: {round.e1?.toFixed(1)}</span>
          <span>E2: {round.e2?.toFixed(1)}</span>
          <span>E3: {round.e3?.toFixed(1)}</span>
          <span>E4: {round.e4?.toFixed(1)}</span>
          <span>GATE: <span style={round.quality_gate === 'PASS' ? passStyle : round.quality_gate === 'FAIL' ? failStyle : { color: '#8e8e93' }}>{round.quality_gate || '-'}</span></span>
        </div>
      </div>

      {/* 展开的详情：Token + AI输出 */}
      {expanded && (
        <div style={{ paddingLeft: 18, marginTop: 8 }}>
          <div style={{ fontSize: 12, color: '#8e8e93', marginBottom: 6 }}>
            Translator: {round.trans_tokens?.toLocaleString() ?? '-'} tokens · {round.trans_latency_ms ? (round.trans_latency_ms / 1000).toFixed(1) + 's' : '-'}
            {round.trans_error && <span style={failStyle}> · 错误: {round.trans_error}</span>}
          </div>
          <div style={{ fontSize: 12, color: '#8e8e93', marginBottom: 8 }}>
            Reviewer: {round.review_tokens?.toLocaleString() ?? '-'} tokens · {round.review_latency_ms ? (round.review_latency_ms / 1000).toFixed(1) + 's' : '-'}
            {round.review_error && <span style={failStyle}> · 错误: {round.review_error}</span>}
          </div>

          {/* Translator输出 */}
          <AIOutputViewer output={round.trans_output} label={`Translator输出（第${round.round}轮）`} maxPreviewLen={300} />

          {/* Reviewer输出 */}
          <AIOutputViewer output={round.review_output} label={`Reviewer输出（第${round.round}轮）`} maxPreviewLen={300} />
        </div>
      )}
    </div>
  )
}

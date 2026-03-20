/**
 * Evaluator步骤调试面板
 * P4.5-B: 展示多轮评估分数、方差、各维度均值、轮次详情
 */
import {
  kvRow, kvLabel, kvValue, passStyle, failStyle,
  ScoreCardRow, TokenDuration, PromptInfo, SectionTitle, scoreColor,
} from './StepCommon'

/** Evaluator面板属性 */
interface EvaluatorPanelProps {
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

      {/* 各轮分数 */}
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
    </div>
  )
}

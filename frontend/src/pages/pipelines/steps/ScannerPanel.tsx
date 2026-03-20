/**
 * Scanner步骤调试面板
 * P4.5-B: 展示课程定位JSON + AI原始输出 + 模型/Token信息
 */
import {
  kvRow, kvLabel, kvValue, passStyle, failStyle,
  TokenDuration, AIOutputViewer, PromptInfo, SectionTitle, JsonViewer,
} from './StepCommon'

/** Scanner面板属性 */
interface ScannerPanelProps {
  data: any
}

/** Scanner步骤调试面板 */
export default function ScannerPanel({ data }: ScannerPanelProps) {
  if (!data) return <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>

  const parsed = data.parsed

  return (
    <div>
      {/* 基本信息 */}
      <div style={kvRow}>
        <span style={kvLabel}>解析状态</span>
        <span style={kvValue}>
          {data.is_valid
            ? <span style={passStyle}>有效</span>
            : <span style={failStyle}>无效</span>}
        </span>
      </div>
      <div style={kvRow}>
        <span style={kvLabel}>模型</span>
        <span style={kvValue}>{data.model_used || '-'}</span>
      </div>
      <TokenDuration tokens={data.tokens_used} durationMs={data.latency_ms} />
      <PromptInfo data={data} />

      {/* 课程定位信息（parsed JSON） */}
      {parsed && (
        <>
          <SectionTitle title="课程定位信息" />
          <div style={kvRow}>
            <span style={kvLabel}>核心目标</span>
            <span style={kvValue}>{parsed.target || '-'}</span>
          </div>
          <div style={kvRow}>
            <span style={kvLabel}>能力目标</span>
            <span style={kvValue}>{(parsed.ability_targets || []).join('；') || '-'}</span>
          </div>
          <div style={kvRow}>
            <span style={kvLabel}>年级标准</span>
            <span style={kvValue}>{parsed.grade_standard || '-'}</span>
          </div>
          <div style={kvRow}>
            <span style={kvLabel}>课程标准</span>
            <span style={kvValue}>{parsed.course_standard || '-'}</span>
          </div>

          {/* 完整定位JSON查看器 */}
          <JsonViewer data={parsed} label="完整定位JSON" />
        </>
      )}

      {/* AI原始输出 */}
      <AIOutputViewer output={data.raw_output} label="AI原始输出（Prompt A）" />
    </div>
  )
}

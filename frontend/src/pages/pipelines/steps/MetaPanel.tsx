/**
 * Meta步骤调试面板
 * P4.5-B: 展示仲裁分数、各维度终裁、硬性约束、评级、各轮E1-E4分数、AI原始输出
 */
import {
  kvRow, kvLabel, kvValue, passStyle, failStyle,
  ScoreCardRow, TokenDuration, AIOutputViewer, PromptInfo, SectionTitle, scoreColor,
} from './StepCommon'

/** Meta面板属性 */
interface MetaPanelProps {
  data: any
}

/** Meta步骤调试面板 */
export default function MetaPanel({ data }: MetaPanelProps) {
  if (!data) return <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>

  return (
    <div>
      {/* 仲裁总分卡片 */}
      <ScoreCardRow items={[
        { label: '仲裁总分', value: data.total_final?.toFixed(2), color: data.passed ? '#34c759' : '#ff3b30' },
        { label: 'E1 终裁', value: data.e1_final?.toFixed(2) },
        { label: 'E2 终裁', value: data.e2_final?.toFixed(2) },
        { label: 'E3 终裁', value: data.e3_final?.toFixed(2) },
        { label: 'E4 终裁', value: data.e4_final?.toFixed(2) },
      ]} />

      {/* 硬性约束和评级 */}
      <div style={kvRow}>
        <span style={kvLabel}>硬性约束</span>
        <span style={kvValue}>
          <span style={data.hard_constraint === 'PASS' ? passStyle : failStyle}>
            {data.hard_constraint || '-'}
          </span>
        </span>
      </div>
      <div style={kvRow}>
        <span style={kvLabel}>评级</span>
        <span style={kvValue}>{data.grade || '-'}</span>
      </div>
      <div style={kvRow}>
        <span style={kvLabel}>通过状态</span>
        <span style={kvValue}>
          {data.passed
            ? <span style={passStyle}>通过 (≥{9.0})</span>
            : <span style={failStyle}>不通过</span>}
        </span>
      </div>
      <div style={kvRow}>
        <span style={kvLabel}>尝试次数</span>
        <span style={kvValue}>{data.attempt ?? '-'} / {data.total_retries ?? '-'}</span>
      </div>

      {/* 各轮E1-E4分数表格（如果有多轮数据） */}
      {data.e1_rounds && data.e1_rounds.length > 0 && (
        <>
          <SectionTitle title="各轮维度分数" />
          <div style={{ overflowX: 'auto' }}>
            <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
              <thead>
                <tr style={{ borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
                  <th style={{ textAlign: 'left', padding: '6px', color: '#8e8e93', fontWeight: 600 }}>轮次</th>
                  <th style={{ textAlign: 'center', padding: '6px', color: '#8e8e93', fontWeight: 600 }}>E1</th>
                  <th style={{ textAlign: 'center', padding: '6px', color: '#8e8e93', fontWeight: 600 }}>E2</th>
                  <th style={{ textAlign: 'center', padding: '6px', color: '#8e8e93', fontWeight: 600 }}>E3</th>
                  <th style={{ textAlign: 'center', padding: '6px', color: '#8e8e93', fontWeight: 600 }}>E4</th>
                </tr>
              </thead>
              <tbody>
                {data.e1_rounds.map((_: number, i: number) => (
                  <tr key={i} style={{ borderBottom: '1px solid rgba(0,0,0,0.03)' }}>
                    <td style={{ padding: '6px', fontWeight: 600, color: '#1c1c1e' }}>R{i + 1}</td>
                    <td style={{ padding: '6px', textAlign: 'center', color: scoreColor(data.e1_rounds?.[i]) }}>
                      {data.e1_rounds?.[i]?.toFixed(2) ?? '-'}
                    </td>
                    <td style={{ padding: '6px', textAlign: 'center', color: scoreColor(data.e2_rounds?.[i]) }}>
                      {data.e2_rounds?.[i]?.toFixed(2) ?? '-'}
                    </td>
                    <td style={{ padding: '6px', textAlign: 'center', color: scoreColor(data.e3_rounds?.[i]) }}>
                      {data.e3_rounds?.[i]?.toFixed(2) ?? '-'}
                    </td>
                    <td style={{ padding: '6px', textAlign: 'center', color: scoreColor(data.e4_rounds?.[i]) }}>
                      {data.e4_rounds?.[i]?.toFixed(2) ?? '-'}
                    </td>
                  </tr>
                ))}
                {/* 终裁行 */}
                <tr style={{ borderTop: '2px solid rgba(0,0,0,0.08)', fontWeight: 700 }}>
                  <td style={{ padding: '6px', color: '#1c1c1e' }}>终裁</td>
                  <td style={{ padding: '6px', textAlign: 'center', color: scoreColor(data.e1_final) }}>{data.e1_final?.toFixed(2)}</td>
                  <td style={{ padding: '6px', textAlign: 'center', color: scoreColor(data.e2_final) }}>{data.e2_final?.toFixed(2)}</td>
                  <td style={{ padding: '6px', textAlign: 'center', color: scoreColor(data.e3_final) }}>{data.e3_final?.toFixed(2)}</td>
                  <td style={{ padding: '6px', textAlign: 'center', color: scoreColor(data.e4_final) }}>{data.e4_final?.toFixed(2)}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </>
      )}

      {/* 模型/Token/耗时 */}
      <div style={kvRow}>
        <span style={kvLabel}>模型</span>
        <span style={kvValue}>{data.model_used || '-'}</span>
      </div>
      <TokenDuration tokens={data.tokens_used} durationMs={data.latency_ms} />
      <PromptInfo data={data} />

      {/* AI原始输出查看器 */}
      <AIOutputViewer output={data.raw_output} label="AI原始输出（Prompt E）" />
    </div>
  )
}

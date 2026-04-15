/**
 * Generator步骤调试面板
 * P4.5-B: 展示逐页生成统计、操作分布、页面详情表格
 */
import {
  kvRow, kvLabel, kvValue,
  ScoreCardRow, TokenDuration, PromptInfo,
} from './StepCommon'

/** Generator面板属性 */
interface GeneratorPanelProps {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  data: any
}

/** 操作类型颜色映射 */
const opColorMap: Record<string, { bg: string; fg: string }> = {
  keep:   { bg: 'rgba(142,142,147,0.1)', fg: '#8e8e93' },
  modify: { bg: 'rgba(0,122,255,0.1)', fg: '#007aff' },
  create: { bg: 'rgba(52,199,89,0.1)', fg: '#34c759' },
  merge:  { bg: 'rgba(175,82,222,0.1)', fg: '#af52de' },
  delete: { bg: 'rgba(255,59,48,0.1)', fg: '#ff3b30' },
}

/** Generator步骤调试面板 */
export default function GeneratorPanel({ data }: GeneratorPanelProps) {
  if (!data) return <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>

  return (
    <div>
      {/* 操作统计卡片 */}
      <ScoreCardRow items={[
        { label: '总页面', value: data.total_pages, color: '#1c1c1e' },
        { label: '保留', value: data.kept_pages, color: '#8e8e93' },
        { label: '修改', value: data.modified_pages, color: '#007aff' },
        { label: '新建', value: data.created_pages, color: '#34c759' },
        { label: '合并', value: data.merged_pages, color: '#af52de' },
        { label: '删除', value: data.deleted_pages, color: '#ff3b30' },
        { label: '失败', value: data.failed_pages, color: data.failed_pages > 0 ? '#ff3b30' : '#aeaeb2' },
      ]} minWidth={60} />

      {/* 模型/Token/耗时 */}
      <div style={kvRow}>
        <span style={kvLabel}>模型</span>
        <span style={kvValue}>{data.model_used || '-'}</span>
      </div>
      <TokenDuration tokens={data.total_tokens} durationMs={data.total_latency_ms} />
      <PromptInfo data={data} />

      {/* 页面详情表格 */}
      {(data.pages || []).length > 0 && (
        <div style={{ marginTop: 12, overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
            <thead>
              <tr style={{ borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
                {['页码', '标题', '操作', '原始HTML', '生成HTML', 'Token', '耗时', '状态'].map(h => (
                  <th key={h} style={{
                    textAlign: 'left', padding: '8px 6px',
                    color: '#8e8e93', fontWeight: 600, fontSize: 11,
                  }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              {(data.pages || []).map((p: any) => { // eslint-disable-line @typescript-eslint/no-explicit-any
                const opColor = opColorMap[p.operation] || opColorMap.keep
                return (
                  <tr key={p.page_number} style={{ borderBottom: '1px solid rgba(0,0,0,0.03)' }}>
                    <td style={{ padding: '6px', fontFamily: 'monospace', fontWeight: 600 }}>
                      P{String(p.page_number).padStart(2, '0')}
                    </td>
                    <td style={{
                      padding: '6px', maxWidth: 200, overflow: 'hidden',
                      textOverflow: 'ellipsis', whiteSpace: 'nowrap',
                    }}>{p.page_title}</td>
                    <td style={{ padding: '6px' }}>
                      <span style={{
                        padding: '2px 8px', borderRadius: 10, fontSize: 11,
                        fontWeight: 500, background: opColor.bg, color: opColor.fg,
                      }}>{p.operation}</span>
                    </td>
                    <td style={{ padding: '6px', fontFamily: 'monospace', color: '#8e8e93' }}>
                      {p.has_orig_html
                        ? (p.orig_html_len > 1000 ? (p.orig_html_len / 1000).toFixed(1) + 'K' : p.orig_html_len)
                        : '-'}
                    </td>
                    <td style={{ padding: '6px', fontFamily: 'monospace', color: '#1c1c1e' }}>
                      {p.gen_html_len > 0
                        ? (p.gen_html_len > 1000 ? (p.gen_html_len / 1000).toFixed(1) + 'K' : p.gen_html_len)
                        : '-'}
                    </td>
                    <td style={{ padding: '6px', fontFamily: 'monospace', color: '#8e8e93' }}>
                      {p.tokens_used > 0 ? p.tokens_used.toLocaleString() : '-'}
                    </td>
                    <td style={{ padding: '6px', fontFamily: 'monospace', color: '#8e8e93' }}>
                      {p.latency_ms > 0 ? (p.latency_ms / 1000).toFixed(1) + 's' : '-'}
                    </td>
                    <td style={{ padding: '6px' }}>
                      {p.status === 'done'
                        ? <span style={{ color: '#34c759' }}>✓</span>
                        : p.status === 'failed'
                          ? <span style={{ color: '#ff3b30' }}>✗ {p.error}</span>
                          : <span style={{ color: '#aeaeb2' }}>{p.status}</span>}
                    </td>
                  </tr>
                )
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

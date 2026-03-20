/**
 * Pipeline详情页面（占位）
 * P4-7: 下一步完善 - 7步进度可视化 + 各步骤详情面板
 */
import { useState, useEffect, useCallback } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { getPipelineDetail, getStepDetail, type PipelineDetailResponse, type StepDetailResponse } from '@/api/pipelines'
import { ArrowLeft, RefreshCw } from 'lucide-react'

export default function PipelineDetailPage() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [detail, setDetail] = useState<PipelineDetailResponse | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

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

  const btn: React.CSSProperties = {
    padding: '8px 16px', borderRadius: 10, border: '1px solid rgba(0,0,0,0.08)',
    background: '#fff', fontSize: 13, fontWeight: 500, cursor: 'pointer',
    display: 'inline-flex', alignItems: 'center', gap: 6,
  }

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
            状态: {detail.status_name} · 当前步骤: {detail.current_step_name}
          </div>
        </div>
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
          {detail.steps.map((step, i) => {
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
                <div style={{ fontSize: 11, fontWeight: 600, color: step.status === 'done' ? '#34c759' : step.status === 'running' ? '#007aff' : step.status === 'failed' ? '#ff3b30' : '#aeaeb2' }}>
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

      {/* 步骤详情卡片（每个步骤一张卡片） */}
      <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
        {detail.steps.filter(s => s.has_data || s.status === 'done' || s.status === 'failed').map(step => (
          <StepCard key={step.step_name} pipelineId={detail.id} step={step} />
        ))}
      </div>
    </div>
  )
}

// ==================== 步骤详情卡片 ====================
function StepCard({ pipelineId, step }: { pipelineId: string; step: import('@/api/pipelines').StepListItem }) {
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
    done: '#34c759', running: '#007aff', failed: '#ff3b30', skipped: '#aeaeb2', pending: '#c7c7cc',
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
          <span style={{ fontSize: 12, color: '#aeaeb2' }}>{step.tokens_used.toLocaleString()} tokens</span>
        )}
        <span style={{ fontSize: 12, color: '#c7c7cc', transition: 'transform 0.2s', transform: expanded ? 'rotate(90deg)' : 'none' }}>▶</span>
      </div>

      {/* 展开的详情内容 */}
      {expanded && (
        <div style={{ padding: '14px 18px' }}>
          {loadingData ? (
            <div style={{ color: '#8e8e93', fontSize: 13 }}>加载步骤数据...</div>
          ) : stepData ? (
            <StepDataView stepName={step.step_name} data={stepData} />
          ) : step.error_message ? (
            <div style={{ color: '#ff3b30', fontSize: 13 }}>{step.error_message}</div>
          ) : (
            <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>
          )}
        </div>
      )}
    </div>
  )
}

// ==================== 步骤数据视图（按步骤类型分发渲染） ====================
function StepDataView({ stepName, data }: { stepName: string; data: any }) {
  if (!data) return <div style={{ color: '#aeaeb2', fontSize: 13 }}>暂无数据</div>

  switch (stepName) {
    case 'dbCheck': return <DbCheckView data={data} />
    case 'scanner': return <ScannerView data={data} />
    case 'evaluator': return <EvaluatorView data={data} />
    case 'meta': return <MetaView data={data} />
    case 'translator': return <TranslatorView data={data} />
    case 'generator': return <GeneratorView data={data} />
    default: return <pre style={{ fontSize: 12, color: '#3c3c43', overflow: 'auto', maxHeight: 400, margin: 0, whiteSpace: 'pre-wrap' }}>{JSON.stringify(data, null, 2)}</pre>
  }
}

// ==================== 各步骤UI组件 ====================

const kvRow: React.CSSProperties = { display: 'flex', gap: 12, padding: '6px 0', borderBottom: '1px solid rgba(0,0,0,0.03)', fontSize: 13 }
const kvLabel: React.CSSProperties = { width: 120, color: '#8e8e93', fontWeight: 500, flexShrink: 0 }
const kvValue: React.CSSProperties = { color: '#1c1c1e', flex: 1, wordBreak: 'break-all' }
const passStyle: React.CSSProperties = { color: '#34c759', fontWeight: 600 }
const failStyle: React.CSSProperties = { color: '#ff3b30', fontWeight: 600 }

/** dbCheck步骤视图 */
function DbCheckView({ data }: { data: any }) {
  return (
    <div>
      <div style={kvRow}><span style={kvLabel}>课程编号</span><span style={kvValue}>{data.course_code}</span></div>
      <div style={kvRow}><span style={kvLabel}>模块ID</span><span style={kvValue}>{data.module_id}</span></div>
      <div style={kvRow}><span style={kvLabel}>索引状态</span><span style={kvValue}>{data.has_index ? <span style={passStyle}>已有索引</span> : <span style={failStyle}>无索引</span>}</span></div>
      <div style={kvRow}><span style={kvLabel}>页面数</span><span style={kvValue}>{data.page_count}</span></div>
      <div style={kvRow}><span style={kvLabel}>总长度</span><span style={kvValue}>{data.total_length?.toLocaleString()} 字符</span></div>
      <div style={kvRow}><span style={kvLabel}>验证结果</span><span style={kvValue}>{data.is_valid ? <span style={passStyle}>通过</span> : <span style={failStyle}>不通过 — {data.error_detail}</span>}</span></div>
    </div>
  )
}

/** Scanner步骤视图 */
function ScannerView({ data }: { data: any }) {
  const parsed = data.parsed
  return (
    <div>
      <div style={kvRow}><span style={kvLabel}>解析状态</span><span style={kvValue}>{data.is_valid ? <span style={passStyle}>有效</span> : <span style={failStyle}>无效</span>}</span></div>
      <div style={kvRow}><span style={kvLabel}>模型</span><span style={kvValue}>{data.model_used}</span></div>
      <div style={kvRow}><span style={kvLabel}>Token</span><span style={kvValue}>{data.tokens_used?.toLocaleString()}</span></div>
      {parsed && (
        <>
          <div style={{ fontSize: 13, fontWeight: 600, color: '#1c1c1e', marginTop: 12, marginBottom: 6 }}>课程定位信息</div>
          <div style={kvRow}><span style={kvLabel}>核心目标</span><span style={kvValue}>{parsed.target}</span></div>
          <div style={kvRow}><span style={kvLabel}>能力目标</span><span style={kvValue}>{(parsed.ability_targets || []).join('；')}</span></div>
          <div style={kvRow}><span style={kvLabel}>年级标准</span><span style={kvValue}>{parsed.grade_standard}</span></div>
          <div style={kvRow}><span style={kvLabel}>课程标准</span><span style={kvValue}>{parsed.course_standard}</span></div>
        </>
      )}
    </div>
  )
}

/** Evaluator步骤视图 */
function EvaluatorView({ data }: { data: any }) {
  return (
    <div>
      <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginBottom: 12 }}>
        {[{ label: '综合均分', value: data.avg_total?.toFixed(2), color: data.avg_total >= 9 ? '#34c759' : '#ff9500' },
          { label: 'E1难度', value: data.avg_e1?.toFixed(2) },
          { label: 'E2节奏', value: data.avg_e2?.toFixed(2) },
          { label: 'E3互动', value: data.avg_e3?.toFixed(2) },
          { label: 'E4设计', value: data.avg_e4?.toFixed(2) },
        ].map(s => (
          <div key={s.label} style={{ textAlign: 'center', minWidth: 80 }}>
            <div style={{ fontSize: 11, color: '#8e8e93', marginBottom: 2 }}>{s.label}</div>
            <div style={{ fontSize: 20, fontWeight: 700, color: s.color || '#1c1c1e' }}>{s.value}</div>
          </div>
        ))}
      </div>
      <div style={kvRow}><span style={kvLabel}>轮次</span><span style={kvValue}>{data.done_rounds} 完成 / {data.total_rounds} 总轮 {data.failed_rounds > 0 && <span style={failStyle}>({data.failed_rounds}失败)</span>}</span></div>
      <div style={kvRow}><span style={kvLabel}>各轮分数</span><span style={kvValue}>{(data.round_scores || []).map((s: number) => s.toFixed(2)).join(' , ')}</span></div>
      <div style={kvRow}><span style={kvLabel}>方差</span><span style={kvValue}>{data.variance?.toFixed(4)} {data.variance_warn && <span style={failStyle}>⚠️ 超过阈值</span>}</span></div>
      <div style={kvRow}><span style={kvLabel}>Token / 耗时</span><span style={kvValue}>{data.total_tokens?.toLocaleString()} / {data.total_latency_ms > 60000 ? (data.total_latency_ms / 60000).toFixed(1) + '分钟' : (data.total_latency_ms / 1000).toFixed(1) + '秒'}</span></div>
    </div>
  )
}

/** Meta步骤视图 */
function MetaView({ data }: { data: any }) {
  return (
    <div>
      <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginBottom: 12 }}>
        {[{ label: '仲裁总分', value: data.total_final?.toFixed(2), color: data.passed ? '#34c759' : '#ff3b30' },
          { label: 'E1', value: data.e1_final?.toFixed(2) },
          { label: 'E2', value: data.e2_final?.toFixed(2) },
          { label: 'E3', value: data.e3_final?.toFixed(2) },
          { label: 'E4', value: data.e4_final?.toFixed(2) },
        ].map(s => (
          <div key={s.label} style={{ textAlign: 'center', minWidth: 70 }}>
            <div style={{ fontSize: 11, color: '#8e8e93', marginBottom: 2 }}>{s.label}</div>
            <div style={{ fontSize: 20, fontWeight: 700, color: s.color || '#1c1c1e' }}>{s.value}</div>
          </div>
        ))}
      </div>
      <div style={kvRow}><span style={kvLabel}>硬性约束</span><span style={kvValue}><span style={data.hard_constraint === 'PASS' ? passStyle : failStyle}>{data.hard_constraint}</span></span></div>
      <div style={kvRow}><span style={kvLabel}>评级</span><span style={kvValue}>{data.grade}</span></div>
      <div style={kvRow}><span style={kvLabel}>通过状态</span><span style={kvValue}>{data.passed ? <span style={passStyle}>通过</span> : <span style={failStyle}>不通过</span>}</span></div>
      <div style={kvRow}><span style={kvLabel}>尝试次数</span><span style={kvValue}>{data.attempt} / {data.total_retries}</span></div>
      <div style={kvRow}><span style={kvLabel}>Token / 耗时</span><span style={kvValue}>{data.tokens_used?.toLocaleString()} / {data.latency_ms > 60000 ? (data.latency_ms / 60000).toFixed(1) + '分钟' : (data.latency_ms / 1000).toFixed(1) + '秒'}</span></div>
    </div>
  )
}

/** Translator步骤视图 */
function TranslatorView({ data }: { data: any }) {
  return (
    <div>
      <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginBottom: 12, alignItems: 'center' }}>
        <div style={{ textAlign: 'center', minWidth: 90 }}>
          <div style={{ fontSize: 11, color: '#8e8e93', marginBottom: 2 }}>最终评分</div>
          <div style={{ fontSize: 24, fontWeight: 700, color: data.passed ? '#34c759' : '#ff3b30' }}>{data.final_score?.toFixed(2)}</div>
        </div>
        <div>
          <div style={{ fontSize: 12 }}>QUALITY_GATE: <span style={data.final_quality_gate === 'PASS' ? passStyle : failStyle}>{data.final_quality_gate || '-'}</span></div>
          <div style={{ fontSize: 12, marginTop: 2 }}>评级: {data.final_grade || '-'} · 第{data.final_round}轮结束</div>
        </div>
        <div style={{ marginLeft: 'auto', textAlign: 'right' }}>
          <div style={{ fontSize: 12, color: '#8e8e93' }}>通过: {data.passed ? <span style={passStyle}>是</span> : <span style={failStyle}>否</span>}</div>
          <div style={{ fontSize: 12, color: '#aeaeb2', marginTop: 2 }}>{data.total_tokens?.toLocaleString()} tokens · {data.total_latency_ms > 60000 ? (data.total_latency_ms / 60000).toFixed(1) + '分钟' : (data.total_latency_ms / 1000).toFixed(1) + '秒'}</div>
        </div>
      </div>
      {/* 各轮循环详情 */}
      {(data.rounds || []).map((r: any) => (
        <div key={r.round} style={{ padding: '8px 0', borderTop: '1px solid rgba(0,0,0,0.04)', fontSize: 12 }}>
          <div style={{ fontWeight: 600, color: '#1c1c1e', marginBottom: 4 }}>
            第{r.round}轮 — {r.passed ? <span style={passStyle}>通过</span> : <span style={failStyle}>未通过</span>}
          </div>
          <div style={{ color: '#8e8e93', display: 'flex', gap: 12, flexWrap: 'wrap' }}>
            <span>总分: {r.score?.toFixed(2)}</span>
            <span>E1: {r.e1?.toFixed(2)}</span>
            <span>E2: {r.e2?.toFixed(2)}</span>
            <span>E3: {r.e3?.toFixed(2)}</span>
            <span>E4: {r.e4?.toFixed(2)}</span>
            <span>GATE: {r.quality_gate || '-'}</span>
            <span>评级: {r.grade || '-'}</span>
          </div>
        </div>
      ))}
    </div>
  )
}

/** Generator步骤视图 */
function GeneratorView({ data }: { data: any }) {
  const opColorMap: Record<string, { bg: string; fg: string }> = {
    keep:   { bg: 'rgba(142,142,147,0.1)', fg: '#8e8e93' },
    modify: { bg: 'rgba(0,122,255,0.1)', fg: '#007aff' },
    create: { bg: 'rgba(52,199,89,0.1)', fg: '#34c759' },
    merge:  { bg: 'rgba(175,82,222,0.1)', fg: '#af52de' },
    delete: { bg: 'rgba(255,59,48,0.1)', fg: '#ff3b30' },
  }

  return (
    <div>
      {/* 操作统计 */}
      <div style={{ display: 'flex', gap: 12, flexWrap: 'wrap', marginBottom: 16 }}>
        {[
          { label: '总页面', value: data.total_pages, color: '#1c1c1e' },
          { label: '保留', value: data.kept_pages, color: '#8e8e93' },
          { label: '修改', value: data.modified_pages, color: '#007aff' },
          { label: '新建', value: data.created_pages, color: '#34c759' },
          { label: '合并', value: data.merged_pages, color: '#af52de' },
          { label: '删除', value: data.deleted_pages, color: '#ff3b30' },
          { label: '失败', value: data.failed_pages, color: data.failed_pages > 0 ? '#ff3b30' : '#aeaeb2' },
        ].map(s => (
          <div key={s.label} style={{ textAlign: 'center', minWidth: 60 }}>
            <div style={{ fontSize: 11, color: '#8e8e93', marginBottom: 2 }}>{s.label}</div>
            <div style={{ fontSize: 18, fontWeight: 700, color: s.color }}>{s.value}</div>
          </div>
        ))}
      </div>

      <div style={kvRow}>
        <span style={kvLabel}>Token / 耗时</span>
        <span style={kvValue}>{data.total_tokens?.toLocaleString()} / {data.total_latency_ms > 60000 ? (data.total_latency_ms / 60000).toFixed(1) + '分钟' : (data.total_latency_ms / 1000).toFixed(1) + '秒'}</span>
      </div>

      {/* 页面表格 */}
      {(data.pages || []).length > 0 && (
        <div style={{ marginTop: 12, overflowX: 'auto' }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 12 }}>
            <thead>
              <tr style={{ borderBottom: '1px solid rgba(0,0,0,0.06)' }}>
                {['页码', '标题', '操作', '原始HTML', '生成HTML', 'Token', '耗时', '状态'].map(h => (
                  <th key={h} style={{ textAlign: 'left', padding: '8px 6px', color: '#8e8e93', fontWeight: 600, fontSize: 11 }}>{h}</th>
                ))}
              </tr>
            </thead>
            <tbody>
              {(data.pages || []).map((p: any) => {
                const opColor = opColorMap[p.operation] || opColorMap.keep
                return (
                  <tr key={p.page_number} style={{ borderBottom: '1px solid rgba(0,0,0,0.03)' }}>
                    <td style={{ padding: '6px', fontFamily: 'monospace', fontWeight: 600 }}>P{String(p.page_number).padStart(2, '0')}</td>
                    <td style={{ padding: '6px', maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>{p.page_title}</td>
                    <td style={{ padding: '6px' }}>
                      <span style={{ padding: '2px 8px', borderRadius: 10, fontSize: 11, fontWeight: 500, background: opColor.bg, color: opColor.fg }}>{p.operation}</span>
                    </td>
                    <td style={{ padding: '6px', fontFamily: 'monospace', color: '#8e8e93' }}>{p.has_orig_html ? (p.orig_html_len > 1000 ? (p.orig_html_len / 1000).toFixed(1) + 'K' : p.orig_html_len) : '-'}</td>
                    <td style={{ padding: '6px', fontFamily: 'monospace', color: '#1c1c1e' }}>{p.gen_html_len > 0 ? (p.gen_html_len > 1000 ? (p.gen_html_len / 1000).toFixed(1) + 'K' : p.gen_html_len) : '-'}</td>
                    <td style={{ padding: '6px', fontFamily: 'monospace', color: '#8e8e93' }}>{p.tokens_used > 0 ? p.tokens_used.toLocaleString() : '-'}</td>
                    <td style={{ padding: '6px', fontFamily: 'monospace', color: '#8e8e93' }}>{p.latency_ms > 0 ? (p.latency_ms / 1000).toFixed(1) + 's' : '-'}</td>
                    <td style={{ padding: '6px' }}>
                      {p.status === 'done' ? <span style={{ color: '#34c759' }}>✓</span> : p.status === 'failed' ? <span style={{ color: '#ff3b30' }}>✗ {p.error}</span> : <span style={{ color: '#aeaeb2' }}>{p.status}</span>}
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

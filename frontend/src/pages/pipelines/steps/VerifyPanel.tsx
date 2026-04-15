/**
 * Verify验收步骤面板组件
 * P4.6-5新建：展示验收评估结果（压缩索引+评估报告+四维评分+通过/失败状态）
 * 数据来源：pipeline_steps.step_data（verify步骤，由verify_service.go的executeVerify填充）
 * step_data结构：{generated_index, eval_score, eval_output, eval_e1~e4, passed, review_round, model_used, tokens_used, latency_ms}
 *
 * v100 修复：删除未使用的 useEffect 导入
 */
import { useState } from 'react'
import type { VerifyStepData, GeneratedPageFull } from '@/api/pipelines'
import { getPagesMeta } from '@/api/pipelines'

// ==================== 类型定义 ====================

interface VerifyPanelProps {
  data: VerifyStepData
  pipelineId?: string  // v99优化2：用于加载页面标题列表
}

// ==================== 辅助函数 ====================

/** 格式化耗时（毫秒→可读格式） */
function formatDuration(ms: number): string {
  if (!ms || ms <= 0) return '-'
  if (ms < 1000) return ms + 'ms'
  if (ms < 60000) return (ms / 1000).toFixed(1) + 's'
  const min = Math.floor(ms / 60000)
  const sec = Math.round((ms % 60000) / 1000)
  return min + '分' + sec + '秒'
}

/** 评分颜色（≥9.0绿色达标，≥7.0橙色，<7.0红色不达标） */
function scoreColor(score: number): string {
  if (score >= 9.0) return '#34c759'
  if (score >= 7.0) return '#ff9500'
  return '#ff3b30'
}

// ==================== 主组件 ====================

export default function VerifyPanel({ data, pipelineId }: VerifyPanelProps) {
  // 索引内容默认折叠（通常很长）
  const [showIndex, setShowIndex] = useState(false)
  // 评估报告默认折叠
  const [showReport, setShowReport] = useState(false)
  // v99优化2：页面标题列表（验收通过后显示每页编号和标题）
  const [showPages, setShowPages] = useState(false)
  const [pagesList, setPagesList] = useState<GeneratedPageFull[]>([])
  const [pagesLoading, setPagesLoading] = useState(false)

  // 点击展开时按需加载页面列表
  const loadPages = async () => {
    if (!pipelineId || pagesList.length > 0) { setShowPages(!showPages); return }
    setPagesLoading(true)
    try {
      const pages = await getPagesMeta(pipelineId)
      setPagesList(pages || [])
    } catch { /* ignore */ }
    setPagesLoading(false)
    setShowPages(true)
  }

  // 四维评分数据
  const dimensions = [
    { label: 'E1 难度适配', score: data.eval_e1 },
    { label: 'E2 时间节奏', score: data.eval_e2 },
    { label: 'E3 互动评估', score: data.eval_e3 },
    { label: 'E4 课程设计', score: data.eval_e4 },
  ]

  // ==================== 样式定义 ====================

  /** 信息行样式 */
  const rowStyle: React.CSSProperties = {
    display: 'flex', alignItems: 'center', gap: 12,
    padding: '8px 0', borderBottom: '1px solid rgba(0,0,0,0.04)',
    fontSize: 13,
  }

  /** 标签样式 */
  const labelStyle: React.CSSProperties = {
    width: 100, flexShrink: 0, color: '#8e8e93', fontWeight: 500,
  }

  /** 值样式 */
  const valueStyle: React.CSSProperties = {
    flex: 1, color: '#1c1c1e', fontWeight: 500,
  }

  /** 折叠按钮样式 */
  const toggleBtnStyle: React.CSSProperties = {
    padding: '6px 14px', borderRadius: 8, border: '1px solid rgba(0,0,0,0.08)',
    background: '#f5f5f7', fontSize: 12, fontWeight: 500, color: '#007aff',
    cursor: 'pointer', display: 'inline-flex', alignItems: 'center', gap: 4,
  }

  /** 内容块样式 */
  const contentBlockStyle: React.CSSProperties = {
    marginTop: 8, padding: 14, borderRadius: 10,
    background: '#f9f9fb', border: '1px solid rgba(0,0,0,0.04)',
    fontSize: 12, lineHeight: 1.6, color: '#3c3c43',
    maxHeight: 400, overflow: 'auto', whiteSpace: 'pre-wrap',
    fontFamily: 'ui-monospace, "SF Mono", Monaco, "Cascadia Code", monospace',
  }

  return (
    <div>
      {/* ===== 验收结果概览 ===== */}
      <div style={{
        display: 'flex', alignItems: 'center', gap: 14, marginBottom: 16,
        padding: '14px 18px', borderRadius: 12,
        background: data.passed
          ? 'linear-gradient(135deg, rgba(52,199,89,0.08), rgba(52,199,89,0.02))'
          : 'linear-gradient(135deg, rgba(255,59,48,0.08), rgba(255,59,48,0.02))',
        border: `1px solid ${data.passed ? 'rgba(52,199,89,0.2)' : 'rgba(255,59,48,0.2)'}`,
      }}>
        {/* 通过/失败图标 */}
        <div style={{
          width: 40, height: 40, borderRadius: 20,
          background: data.passed ? '#34c759' : '#ff3b30',
          display: 'flex', alignItems: 'center', justifyContent: 'center',
          color: '#fff', fontSize: 18, fontWeight: 700, flexShrink: 0,
        }}>
          {data.passed ? '✓' : '✗'}
        </div>
        {/* 状态文字 */}
        <div style={{ flex: 1 }}>
          <div style={{
            fontSize: 16, fontWeight: 700,
            color: data.passed ? '#34c759' : '#ff3b30',
          }}>
            {data.passed ? '验收通过' : '验收未通过'}
          </div>
          <div style={{ fontSize: 12, color: '#8e8e93', marginTop: 2 }}>
            {data.review_round > 1 ? `第${data.review_round}审验收` : '初审验收'}
            {' · '}评分: {data.eval_score?.toFixed(2) ?? '-'}/10
            {' · '}阈值: ≥9.0
          </div>
        </div>
        {/* 综合评分大字 */}
        <div style={{
          fontSize: 32, fontWeight: 800, color: scoreColor(data.eval_score ?? 0),
          lineHeight: 1,
        }}>
          {data.eval_score?.toFixed(1) ?? '-'}
        </div>
      </div>

      {/* ===== 四维评分卡片 ===== */}
      <div style={{
        display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: 10,
        marginBottom: 16,
      }}>
        {dimensions.map(dim => (
          <div key={dim.label} style={{
            padding: '12px 14px', borderRadius: 10,
            background: 'rgba(255,255,255,0.8)', border: '1px solid rgba(0,0,0,0.06)',
            textAlign: 'center',
          }}>
            <div style={{ fontSize: 11, color: '#8e8e93', fontWeight: 500, marginBottom: 4 }}>
              {dim.label}
            </div>
            <div style={{
              fontSize: 20, fontWeight: 700, color: scoreColor(dim.score ?? 0),
            }}>
              {dim.score?.toFixed(1) ?? '-'}
            </div>
          </div>
        ))}
      </div>

      {/* ===== 基本信息 ===== */}
      <div style={{ marginBottom: 16 }}>
        <div style={rowStyle}>
          <div style={labelStyle}>使用模型</div>
          <div style={valueStyle}>{data.model_used || '-'}</div>
        </div>
        <div style={rowStyle}>
          <div style={labelStyle}>Token消耗</div>
          <div style={valueStyle}>{data.tokens_used?.toLocaleString() ?? '-'}</div>
        </div>
        <div style={rowStyle}>
          <div style={labelStyle}>执行耗时</div>
          <div style={valueStyle}>{formatDuration(data.latency_ms)}</div>
        </div>
        <div style={{ ...rowStyle, borderBottom: 'none' }}>
          <div style={labelStyle}>审核轮次</div>
          <div style={valueStyle}>
            {data.review_round === 1 ? '初审' : `第${data.review_round}审`}
          </div>
        </div>
      </div>

      {/* ===== 压缩索引（可折叠） ===== */}
      <div style={{ marginBottom: 12 }}>
        <button style={toggleBtnStyle} onClick={() => setShowIndex(!showIndex)}>
          <span style={{
            display: 'inline-block', transition: 'transform 0.2s',
            transform: showIndex ? 'rotate(90deg)' : 'none',
          }}>▶</span>
          查看压缩索引
          {data.generated_index && (
            <span style={{ color: '#aeaeb2', marginLeft: 4 }}>
              ({data.generated_index.length.toLocaleString()} 字符)
            </span>
          )}
        </button>
        {showIndex && (
          <div style={contentBlockStyle}>
            {data.generated_index || '(无索引数据)'}
          </div>
        )}
      </div>

      {/* ===== 页面清单（v99优化2新增，可折叠） ===== */}
      {pipelineId && (
        <div style={{ marginBottom: 12 }}>
          <button style={toggleBtnStyle} onClick={loadPages}>
            <span style={{
              display: 'inline-block', transition: 'transform 0.2s',
              transform: showPages ? 'rotate(90deg)' : 'none',
            }}>▶</span>
            {pagesLoading ? '加载中...' : '查看页面清单'}
          </button>
          {showPages && pagesList.length > 0 && (
            <div style={{ marginTop: 8, padding: 14, borderRadius: 10, background: '#f9f9fb', border: '1px solid rgba(0,0,0,0.04)', maxHeight: 300, overflow: 'auto' }}>
              {pagesList.sort((a, b) => a.page_number - b.page_number).map(p => (
                <div key={p.page_number} style={{ display: 'flex', alignItems: 'center', gap: 8, padding: '4px 0', fontSize: 12, borderBottom: '1px solid rgba(0,0,0,0.03)' }}>
                  <span style={{ fontWeight: 600, color: '#1c1c1e', minWidth: 36 }}>P{String(p.page_number).padStart(2, '0')}</span>
                  <span style={{ color: '#3c3c43', flex: 1 }}>{p.page_title || '无标题'}</span>
                  <span style={{ fontSize: 10, fontWeight: 600, color: '#fff', padding: '1px 6px', borderRadius: 3, background: p.operation === 'keep' ? '#34c759' : p.operation === 'modify' ? '#007aff' : p.operation === 'create' ? '#af52de' : p.operation === 'merge' ? '#ff9500' : p.operation === 'delete' ? '#ff3b30' : '#aeaeb2' }}>
                    {p.operation === 'keep' ? '保留' : p.operation === 'modify' ? '修改' : p.operation === 'create' ? '新建' : p.operation === 'merge' ? '合并' : p.operation === 'delete' ? '删除' : p.operation}
                  </span>
                  <span style={{ fontSize: 10, color: p.decision === 'approve' ? '#34c759' : p.decision === 'reject' ? '#ff3b30' : p.decision === 'edit' ? '#ff9500' : '#c7c7cc', fontWeight: 500 }}>
                    {p.decision === 'approve' ? '已采用' : p.decision === 'reject' ? '已拒绝' : p.decision === 'edit' ? '已编辑' : '待决策'}
                  </span>
                </div>
              ))}
            </div>
          )}
          {showPages && pagesList.length === 0 && !pagesLoading && (
            <div style={{ marginTop: 8, fontSize: 12, color: '#aeaeb2' }}>暂无页面数据</div>
          )}
        </div>
      )}

      {/* ===== 评估报告（可折叠） ===== */}
      <div style={{ marginBottom: 8 }}>
        <button style={toggleBtnStyle} onClick={() => setShowReport(!showReport)}>
          <span style={{
            display: 'inline-block', transition: 'transform 0.2s',
            transform: showReport ? 'rotate(90deg)' : 'none',
          }}>▶</span>
          查看评估报告
          {data.eval_output && (
            <span style={{ color: '#aeaeb2', marginLeft: 4 }}>
              ({data.eval_output.length.toLocaleString()} 字符)
            </span>
          )}
        </button>
        {showReport && (
          <div style={contentBlockStyle}>
            {data.eval_output || '(无评估报告)'}
          </div>
        )}
      </div>
    </div>
  )
}

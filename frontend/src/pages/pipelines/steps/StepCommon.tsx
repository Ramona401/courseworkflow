/* eslint-disable react-refresh/only-export-components */
/**
 * 步骤面板通用组件
 * P4.5-B: 提供KV行、分数卡片、AI输出查看器、提示词信息等通用UI组件
 */
import { useState } from 'react'

// ==================== 样式常量 ====================

/** KV行样式 */
export const kvRow: React.CSSProperties = {
  display: 'flex', gap: 12, padding: '6px 0',
  borderBottom: '1px solid rgba(0,0,0,0.03)', fontSize: 13,
}

/** KV标签样式 */
export const kvLabel: React.CSSProperties = {
  width: 120, color: '#8e8e93', fontWeight: 500, flexShrink: 0,
}

/** KV值样式 */
export const kvValue: React.CSSProperties = {
  color: '#1c1c1e', flex: 1, wordBreak: 'break-all',
}

/** 通过状态文字样式 */
export const passStyle: React.CSSProperties = {
  color: '#34c759', fontWeight: 600,
}

/** 失败状态文字样式 */
export const failStyle: React.CSSProperties = {
  color: '#ff3b30', fontWeight: 600,
}

// ==================== 通用分数卡片 ====================

/** 分数卡片项定义 */
interface ScoreCardItem {
  label: string
  value: string | number | undefined
  color?: string
}

/** 分数卡片组（水平排列的分数块） */
export function ScoreCardRow({ items, minWidth = 70 }: { items: ScoreCardItem[]; minWidth?: number }) {
  return (
    <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap', marginBottom: 12 }}>
      {items.map((item) => (
        <div key={item.label} style={{ textAlign: 'center', minWidth }}>
          <div style={{ fontSize: 11, color: '#8e8e93', marginBottom: 2 }}>{item.label}</div>
          <div style={{ fontSize: 20, fontWeight: 700, color: item.color || '#1c1c1e' }}>
            {item.value ?? '-'}
          </div>
        </div>
      ))}
    </div>
  )
}

// ==================== Token/耗时 格式化 ====================

/** 格式化耗时（毫秒 → 秒/分钟） */
export function formatDuration(ms: number | undefined): string {
  if (!ms || ms <= 0) return '-'
  if (ms < 1000) return ms + 'ms'
  if (ms < 60000) return (ms / 1000).toFixed(1) + '秒'
  return (ms / 60000).toFixed(1) + '分钟'
}

/** 格式化Token数 */
export function formatTokens(tokens: number | undefined): string {
  if (!tokens || tokens <= 0) return '-'
  return tokens.toLocaleString()
}

/** Token和耗时组合显示 */
export function TokenDuration({ tokens, durationMs }: { tokens?: number; durationMs?: number }) {
  return (
    <div style={kvRow}>
      <span style={kvLabel}>Token / 耗时</span>
      <span style={kvValue}>{formatTokens(tokens)} / {formatDuration(durationMs)}</span>
    </div>
  )
}

// ==================== AI原始输出查看器 ====================

/** AI原始输出折叠查看器 */
export function AIOutputViewer({ output, label = 'AI原始输出', maxPreviewLen = 500 }: {
  output: string | undefined
  label?: string
  maxPreviewLen?: number
}) {
  const [expanded, setExpanded] = useState(false)

  if (!output) return null

  const isLong = output.length > maxPreviewLen
  const displayText = expanded ? output : output.slice(0, maxPreviewLen) + (isLong ? '...' : '')

  return (
    <div style={{ marginTop: 12 }}>
      <div
        onClick={() => setExpanded(!expanded)}
        style={{
          display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer',
          fontSize: 13, fontWeight: 600, color: '#007aff', marginBottom: 8,
        }}
      >
        <span style={{
          transition: 'transform 0.2s',
          transform: expanded ? 'rotate(90deg)' : 'none',
          fontSize: 10,
        }}>▶</span>
        {label}
        {isLong && <span style={{ fontSize: 11, color: '#8e8e93', fontWeight: 400 }}>
          ({(output.length / 1000).toFixed(1)}K 字符)
        </span>}
      </div>
      {(expanded || !isLong) && (
        <pre style={{
          fontSize: 12, color: '#3c3c43', background: 'rgba(0,0,0,0.02)',
          borderRadius: 8, padding: 12, overflow: 'auto', maxHeight: 500,
          margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all',
          border: '1px solid rgba(0,0,0,0.04)',
        }}>
          {displayText}
        </pre>
      )}
      {!expanded && isLong && (
        <div style={{
          fontSize: 12, color: '#007aff', cursor: 'pointer', marginTop: 4,
        }} onClick={() => setExpanded(true)}>
          点击展开完整内容...
        </div>
      )}
    </div>
  )
}

// ==================== 提示词信息行 ====================

 
/** 提示词版本信息（如果step_data中包含prompt信息则显示） */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function PromptInfo({ data }: { data: any }) {
  if (!data?.prompt_key) return null
  return (
    <div style={kvRow}>
      <span style={kvLabel}>提示词</span>
      <span style={kvValue}>
        {data.prompt_key}
        {data.prompt_version && <span style={{ color: '#8e8e93', marginLeft: 6 }}>v{data.prompt_version}</span>}
      </span>
    </div>
  )
}

// ==================== 分数颜色工具 ====================

/** 根据分数返回颜色（≥9.0绿、≥7.0橙、<7.0红） */
export function scoreColor(score: number | undefined, threshold = 9.0): string {
  if (score === undefined || score === null) return '#aeaeb2'
  if (score >= threshold) return '#34c759'
  if (score >= 7.0) return '#ff9500'
  return '#ff3b30'
}

// ==================== 节区标题 ====================

/** 面板内部的小节标题 */
export function SectionTitle({ title }: { title: string }) {
  return (
    <div style={{
      fontSize: 13, fontWeight: 600, color: '#1c1c1e',
      marginTop: 16, marginBottom: 8,
      paddingBottom: 4, borderBottom: '1px solid rgba(0,0,0,0.06)',
    }}>
      {title}
    </div>
  )
}

// ==================== JSON 查看器 ====================

 
/** JSON数据查看器（折叠式） */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function JsonViewer({ data, label = '原始JSON' }: { data: any; label?: string }) {
  const [expanded, setExpanded] = useState(false)

  if (!data) return null

  const jsonStr = typeof data === 'string' ? data : JSON.stringify(data, null, 2)

  return (
    <div style={{ marginTop: 12 }}>
      <div
        onClick={() => setExpanded(!expanded)}
        style={{
          display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer',
          fontSize: 13, fontWeight: 600, color: '#8e8e93', marginBottom: 8,
        }}
      >
        <span style={{
          transition: 'transform 0.2s',
          transform: expanded ? 'rotate(90deg)' : 'none',
          fontSize: 10,
        }}>▶</span>
        {label}
      </div>
      {expanded && (
        <pre style={{
          fontSize: 11, color: '#3c3c43', background: 'rgba(0,0,0,0.02)',
          borderRadius: 8, padding: 12, overflow: 'auto', maxHeight: 400,
          margin: 0, whiteSpace: 'pre-wrap', wordBreak: 'break-all',
          border: '1px solid rgba(0,0,0,0.04)',
        }}>
          {jsonStr}
        </pre>
      )}
    </div>
  )
}

/**
 * AIFixPanel.tsx — AI辅助修改面板（从PlanDetailTabs.tsx拆分）
 *
 * v104新增：
 *   - 错误状态新增「重新生成」按钮
 *   - 接收 planContext 传给后端，让AI了解教案整体再改段落
 * v105新增（P2-10批量AI修改支持）：
 *   - 支持 batchMode prop（批量模式，标题显示批次编号）
 *   - onAdopt 回调统一：(suggestion: string) => void
 */
import { useState, useEffect, useRef } from 'react'
import {
  aiFixAnnotation,
  type Annotation,
  type AIFixSSEConnection,
} from '@/api/annotations'

interface AIFixPanelProps {
  annotation: Annotation
  paragraphContent: string
  planContext: string         // 教案全貌上下文（学科/年级/课题+完整正文）
  onAdopt: (suggestion: string) => void
  onClose: () => void
  batchMode?: boolean         // v105：批量模式，隐藏「忽略」按钮，标题显示「批量」字样
  batchIndex?: number         // v105：批量模式下当前批次编号（1-based）
  batchTotal?: number         // v105：批量总数
}

export function AIFixPanel({
  annotation,
  paragraphContent,
  planContext,
  onAdopt,
  onClose,
  batchMode = false,
  batchIndex,
  batchTotal,
}: AIFixPanelProps) {
  const [status, setStatus] = useState<'loading' | 'streaming' | 'done' | 'error'>('loading')
  const [streamText, setStreamText] = useState('')
  const [errorMsg, setErrorMsg] = useState('')
  const [suggestion, setSuggestion] = useState('')
  const connRef = useRef<AIFixSSEConnection | null>(null)

  /**
   * 启动AI调用（可重复调用，实现「重新生成」功能）
   */
  const startAIFix = () => {
    setStatus('loading')
    setStreamText('')
    setErrorMsg('')
    setSuggestion('')

    connRef.current?.close()

    const conn = aiFixAnnotation(
      annotation.lesson_plan_id,
      annotation.id,
      paragraphContent,
      annotation.content,
      planContext,
      {
        onConnected: () => setStatus('streaming'),
        onChunk: (chunk: string) => {
          setStreamText(prev => prev + chunk)
          setStatus('streaming')
        },
        onDone: (fullContent: string) => {
          setStreamText(fullContent)
          setSuggestion(extractSuggestion(fullContent))
          setStatus('done')
        },
        onError: (err: string) => {
          setErrorMsg(err)
          setStatus('error')
        },
      }
    )
    connRef.current = conn
  }

  useEffect(() => {
    startAIFix()
    return () => { connRef.current?.close() }
  // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  /**
   * 从AI输出中提取【修改建议】部分
   * 格式：【问题分析】...【修改建议】...
   */
  function extractSuggestion(text: string): string {
    const marker = '【修改建议】'
    const idx = text.indexOf(marker)
    if (idx >= 0) return text.slice(idx + marker.length).trim()
    return text.trim()
  }

  // 批量模式标题
  const titleText = batchMode
    ? `🤖 批量AI修改 ${batchIndex != null && batchTotal != null ? `(${batchIndex}/${batchTotal})` : ''}`
    : '🤖 AI辅助修改建议'

  return (
    <div style={{
      margin: '8px 0 12px 15px',
      padding: '16px',
      background: 'linear-gradient(135deg, #EFF6FF 0%, #F0FDF4 100%)',
      borderRadius: '10px',
      border: '1px solid #BFDBFE',
    }}>
      {/* 标题行 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '12px' }}>
        <div style={{ fontSize: '13px', fontWeight: 700, color: '#1D4ED8', display: 'flex', alignItems: 'center', gap: '6px' }}>
          {titleText}
          {status === 'streaming' && (
            <span style={{ fontSize: '11px', fontWeight: 400, color: '#60A5FA' }}>生成中...</span>
          )}
          {status === 'done' && (
            <span style={{ fontSize: '11px', fontWeight: 400, color: '#10B981' }}>✓ 生成完成</span>
          )}
        </div>
        {/* 批量模式下不显示关闭按钮（由父组件统一控制） */}
        {!batchMode && (
          <button
            onClick={() => { connRef.current?.close(); onClose() }}
            style={{ background: 'none', border: 'none', cursor: 'pointer', color: '#9CA3AF', fontSize: '16px', padding: '0 4px' }}
          >×</button>
        )}
      </div>

      {/* 批注内容预览（批量模式时显示，帮助用户了解当前处理哪条批注） */}
      {batchMode && (
        <div style={{
          padding: '8px 10px',
          background: '#FFF7ED',
          borderRadius: '6px',
          border: '1px solid #FED7AA',
          fontSize: '12px',
          color: '#92400E',
          marginBottom: '10px',
          lineHeight: 1.6,
        }}>
          <strong>批注：</strong>{annotation.content}
        </div>
      )}

      {/* 加载中提示 */}
      {status === 'loading' && (
        <div style={{ textAlign: 'center', padding: '16px 0', color: '#60A5FA', fontSize: '13px' }}>
          ⏳ 正在连接AI...
        </div>
      )}

      {/* 错误提示 + 重新生成按钮 */}
      {status === 'error' && (
        <div>
          <div style={{ padding: '12px', background: '#FEF2F2', borderRadius: '8px', color: '#DC2626', fontSize: '13px', marginBottom: '10px' }}>
            ⚠️ {errorMsg}
          </div>
          <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
            {!batchMode && (
              <button
                onClick={onClose}
                style={{ padding: '7px 16px', borderRadius: '7px', border: '1px solid #E5E7EB', background: '#fff', color: '#6B7280', fontSize: '13px', cursor: 'pointer' }}
              >关闭</button>
            )}
            <button
              onClick={startAIFix}
              style={{ padding: '7px 16px', borderRadius: '7px', border: 'none', background: '#3B82F6', color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}
            >🔄 重新生成</button>
          </div>
        </div>
      )}

      {/* AI流式输出内容 */}
      {(status === 'streaming' || status === 'done') && streamText && (
        <div style={{
          padding: '12px',
          background: '#fff',
          borderRadius: '8px',
          border: '1px solid #E5E7EB',
          fontSize: '13px',
          lineHeight: 1.8,
          color: '#374151',
          whiteSpace: 'pre-wrap',
          maxHeight: '280px',
          overflowY: 'auto',
          marginBottom: '12px',
        }}>
          {streamText}
          {status === 'streaming' && (
            <span style={{ display: 'inline-block', width: '2px', height: '14px', background: '#3B82F6', marginLeft: '2px', animation: 'blink 0.8s step-end infinite', verticalAlign: 'middle' }} />
          )}
        </div>
      )}

      {/* 操作按钮（生成完成后显示） */}
      {status === 'done' && (
        <div style={{ display: 'flex', gap: '10px', justifyContent: 'flex-end' }}>
          {/* 批量模式下不显示「忽略」（跳过由父组件BatchAIFixPanel控制） */}
          {!batchMode && (
            <button
              onClick={onClose}
              style={{ padding: '7px 16px', borderRadius: '7px', border: '1px solid #E5E7EB', background: '#fff', color: '#6B7280', fontSize: '13px', cursor: 'pointer' }}
            >忽略</button>
          )}
          <button
            onClick={() => onAdopt(suggestion)}
            style={{ padding: '7px 16px', borderRadius: '7px', border: 'none', background: '#3B82F6', color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}
          >{batchMode ? '✓ 采用并继续' : '✓ 采用建议'}</button>
        </div>
      )}

      <style>{`
        @keyframes blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }
      `}</style>
    </div>
  )
}

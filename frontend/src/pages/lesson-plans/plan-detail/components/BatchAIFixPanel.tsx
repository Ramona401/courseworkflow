/**
 * BatchAIFixPanel.tsx — 批量AI修改面板（P2-10）
 *
 * 功能：
 *   1. 收集所有 pending 批注，逐条调用 AI 生成修改建议
 *   2. 用户可「采用并继续」跳到下一条，或「跳过」本条
 *   3. 全部处理完毕后显示汇总统计，关闭面板
 */
import { useState, useCallback } from 'react'
import { type Annotation } from '@/api/annotations'
import { AIFixPanel } from './AIFixPanel'

interface BatchItem {
  annotation: Annotation
  paragraphContent: string
  paragraphIndex: number
}

interface BatchAIFixPanelProps {
  items: BatchItem[]
  planContext: string
  onAdopt: (paragraphIndex: number, newContent: string, annotationId: string) => void
  onSkip: (annotationId: string) => void
  onClose: () => void
}

export function BatchAIFixPanel({
  items,
  planContext,
  onAdopt,
  onSkip,
  onClose,
}: BatchAIFixPanelProps) {
  const [currentIdx, setCurrentIdx] = useState(0)
  const [adopted, setAdopted] = useState(0)
  const [skipped, setSkipped] = useState(0)
  const isDone = currentIdx >= items.length

  const current = items[currentIdx]

  const handleAdopt = useCallback((suggestion: string) => {
    if (!current) return
    onAdopt(current.paragraphIndex, suggestion, current.annotation.id)
    setAdopted(prev => prev + 1)
    setCurrentIdx(prev => prev + 1)
  }, [current, onAdopt])

  const handleSkip = useCallback(() => {
    if (!current) return
    onSkip(current.annotation.id)
    setSkipped(prev => prev + 1)
    setCurrentIdx(prev => prev + 1)
  }, [current, onSkip])

  return (
    <div style={{
      position: 'fixed', inset: 0, zIndex: 10000,
      background: 'rgba(0,0,0,0.45)',
      display: 'flex', alignItems: 'center', justifyContent: 'center',
      padding: '20px',
    }}
    onClick={e => { if (e.target === e.currentTarget && isDone) onClose() }}
    >
      <div style={{
        background: '#fff',
        borderRadius: '14px',
        width: '100%',
        maxWidth: '600px',
        maxHeight: '80vh',
        display: 'flex',
        flexDirection: 'column',
        boxShadow: '0 20px 60px rgba(0,0,0,0.18)',
        overflow: 'hidden',
      }}
      onClick={e => e.stopPropagation()}
      >
        {/* 头部 */}
        <div style={{
          padding: '16px 20px',
          borderBottom: '1px solid #F3F4F6',
          background: 'linear-gradient(135deg, rgba(79,123,232,0.08), rgba(129,140,248,0.06))',
          flexShrink: 0,
        }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
            <div style={{ fontSize: '15px', fontWeight: 700, color: '#1D4ED8' }}>
              🤖 批量AI修改
            </div>
            <button
              onClick={onClose}
              style={{ background: 'none', border: 'none', cursor: 'pointer', color: '#9CA3AF', fontSize: '18px' }}
            >×</button>
          </div>
          {/* 进度条 */}
          <div style={{ marginTop: '10px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '12px', color: '#6B7280', marginBottom: '6px' }}>
              <span>处理进度</span>
              <span>{Math.min(currentIdx, items.length)} / {items.length} 条</span>
            </div>
            <div style={{ height: '4px', background: '#E5E7EB', borderRadius: '2px', overflow: 'hidden' }}>
              <div style={{
                height: '100%',
                borderRadius: '2px',
                width: `${items.length > 0 ? (Math.min(currentIdx, items.length) / items.length) * 100 : 0}%`,
                background: isDone ? '#10B981' : '#4F7BE8',
                transition: 'width 400ms ease',
              }} />
            </div>
          </div>
        </div>

        {/* 内容区 */}
        <div style={{ flex: 1, overflow: 'auto', padding: '16px 20px' }}>
          {isDone ? (
            <div style={{ textAlign: 'center', padding: '32px 20px' }}>
              <div style={{ fontSize: '48px', marginBottom: '16px' }}>🎉</div>
              <div style={{ fontSize: '18px', fontWeight: 700, color: '#1F2937', marginBottom: '8px' }}>
                批量修改完成！
              </div>
              <div style={{ fontSize: '14px', color: '#6B7280', lineHeight: 1.7 }}>
                共处理 {items.length} 条批注<br />
                ✅ 采用 <strong>{adopted}</strong> 条 &nbsp;·&nbsp; ⏭ 跳过 <strong>{skipped}</strong> 条
              </div>
              <button
                onClick={onClose}
                style={{
                  marginTop: '24px',
                  padding: '10px 32px',
                  borderRadius: '10px',
                  border: 'none',
                  background: '#4F7BE8',
                  color: '#fff',
                  fontSize: '14px',
                  fontWeight: 600,
                  cursor: 'pointer',
                }}
              >完成</button>
            </div>
          ) : (
            <div>
              {/* 段落预览 */}
              <div style={{
                padding: '10px 14px',
                background: '#F9FAFB',
                borderRadius: '8px',
                border: '1px solid #E5E7EB',
                fontSize: '13px',
                color: '#374151',
                lineHeight: 1.7,
                marginBottom: '12px',
                maxHeight: '100px',
                overflowY: 'auto',
              }}>
                <div style={{ fontSize: '11px', color: '#9CA3AF', marginBottom: '4px', fontWeight: 600 }}>📝 原段落内容</div>
                {current.paragraphContent}
              </div>

              {/* AI修改面板（批量模式） */}
              <AIFixPanel
                annotation={current.annotation}
                paragraphContent={current.paragraphContent}
                planContext={planContext}
                onAdopt={handleAdopt}
                onClose={() => {}}
                batchMode={true}
                batchIndex={currentIdx + 1}
                batchTotal={items.length}
              />

              {/* 跳过按钮 */}
              <div style={{ display: 'flex', justifyContent: 'flex-start', marginTop: '4px' }}>
                <button
                  onClick={handleSkip}
                  style={{
                    padding: '6px 16px',
                    borderRadius: '8px',
                    border: '1px solid #E5E7EB',
                    background: 'transparent',
                    fontSize: '12px',
                    color: '#9CA3AF',
                    cursor: 'pointer',
                  }}
                >⏭ 跳过此条</button>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

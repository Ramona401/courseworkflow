/**
 * ManualEditBox.tsx — 段落手动编辑框（从PlanDetailTabs.tsx拆分）
 *
 * v107改动（P1.5）：
 *   - 新增实时 Markdown 预览切换（编辑模式 ↔ 预览模式）
 *   - 新增字数统计（汉字/字母/数字均计入）
 *   - 预览使用与正文一致的 renderMarkdown 渲染器
 */
import { useState } from 'react'
import { renderMarkdown } from './planDetailConstants'

interface ManualEditBoxProps {
  initialContent: string
  onSave: (newContent: string) => void
  onClose: () => void
}

export function ManualEditBox({ initialContent, onSave, onClose }: ManualEditBoxProps) {
  const [text, setText]       = useState(initialContent)
  const [saving, setSaving]   = useState(false)
  const [preview, setPreview] = useState(false)  // 是否处于预览模式

  // 字数统计：去除Markdown符号后按Unicode字符计数
  const charCount = text.replace(/[#*`>_~\[\]()!-]/g, '').replace(/\s+/g, '').length

  const handleSave = () => {
    if (!text.trim()) return
    setSaving(true)
    onSave(text.trim())
    setSaving(false)
  }

  return (
    <div style={{
      margin: '8px 0 12px 15px', padding: '14px',
      background: '#FAFAFA', borderRadius: '10px', border: '1px solid #E5E7EB',
    }}>
      {/* 标题栏：左侧标题，右侧切换预览按钮 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '8px' }}>
        <div style={{ fontSize: '13px', fontWeight: 600, color: '#374151' }}>
          ✏️ 手动编辑段落
        </div>
        <button
          onClick={() => setPreview(prev => !prev)}
          style={{
            padding: '3px 10px', borderRadius: '6px', fontSize: '12px', fontWeight: 500,
            border: `1px solid ${preview ? '#3B82F6' : '#D1D5DB'}`,
            background: preview ? 'rgba(59,130,246,0.08)' : '#fff',
            color: preview ? '#2563EB' : '#6B7280',
            cursor: 'pointer', transition: 'all 150ms ease',
          }}
        >
          {preview ? '📝 编辑模式' : '👁 预览模式'}
        </button>
      </div>

      {/* 编辑区 / 预览区 */}
      {preview ? (
        <div style={{
          minHeight: '120px', padding: '10px 12px', borderRadius: '8px',
          border: '1px solid #E5E7EB', background: '#fff',
          fontSize: '13px', lineHeight: 1.9, color: '#374151',
          overflowY: 'auto',
        }}>
          {text.trim() ? renderMarkdown(text) : (
            <span style={{ color: '#9CA3AF', fontStyle: 'italic' }}>（暂无内容）</span>
          )}
        </div>
      ) : (
        <textarea
          value={text}
          onChange={e => setText(e.target.value)}
          rows={6}
          style={{
            width: '100%', boxSizing: 'border-box',
            padding: '10px 12px', borderRadius: '8px',
            border: '1px solid #D1D5DB', fontSize: '13px',
            lineHeight: 1.7, color: '#374151',
            resize: 'vertical', fontFamily: 'inherit', outline: 'none',
            transition: 'border-color 150ms ease',
          }}
          onFocus={e => { e.target.style.borderColor = '#3B82F6' }}
          onBlur={e  => { e.target.style.borderColor = '#D1D5DB' }}
        />
      )}

      {/* 底部：字数统计 + 操作按钮 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: '10px' }}>
        <div style={{ fontSize: '11px', color: '#9CA3AF' }}>
          约 <span style={{ fontWeight: 600, color: charCount > 0 ? '#6B7280' : '#9CA3AF' }}>{charCount}</span> 字
          {!preview && (
            <span style={{ marginLeft: '6px', opacity: 0.7 }}>（支持 Markdown 格式）</span>
          )}
        </div>
        <div style={{ display: 'flex', gap: '10px' }}>
          <button
            onClick={onClose}
            style={{
              padding: '7px 16px', borderRadius: '7px',
              border: '1px solid #E5E7EB', background: '#fff',
              color: '#6B7280', fontSize: '13px', cursor: 'pointer',
            }}
          >取消</button>
          <button
            onClick={handleSave}
            disabled={saving || !text.trim()}
            style={{
              padding: '7px 16px', borderRadius: '7px', border: 'none',
              background: text.trim() ? '#10B981' : '#E5E7EB',
              color: text.trim() ? '#fff' : '#9CA3AF',
              fontSize: '13px', fontWeight: 600,
              cursor: text.trim() ? 'pointer' : 'not-allowed',
              transition: 'background 150ms ease',
            }}
          >{saving ? '保存中...' : '✓ 保存'}</button>
        </div>
      </div>
    </div>
  )
}

/**
 * ManualEditBox.tsx — 段落手动编辑框
 *
 * v107 (P1.5):
 *   - 新增实时 Markdown 预览切换(编辑模式 ↔ 预览模式)
 *   - 新增字数统计(汉字/字母/数字均计入)
 *   - 预览使用与正文一致的 renderMarkdown 渲染器
 *
 * v123 (教案插入图片):
 *   - 工具栏新增"📷 插入图片"按钮(支持点击选择文件)
 *   - 支持拖拽图片到编辑区(drop 事件)
 *   - 支持粘贴图片(paste 事件,从剪贴板读取)
 *   - 上传成功自动在光标位置插入 ![alt](url) markdown
 *   - 上传中显示进度提示,禁用编辑防止操作冲突
 *   - 上传失败显示友好错误提示,不阻断编辑
 *
 * 集成路径:Props 新增 planID 字段,父组件 PlanDetailTabs.tsx 已有 plan 对象,
 *          只需在调用处传 plan.id 即可,改动极小。
 */
import { useState, useRef } from 'react'
import { renderMarkdown } from './planDetailConstants'
import { uploadAsset, validateImageFile } from '@/api/lesson-plan-assets'

interface ManualEditBoxProps {
  initialContent: string
  onSave: (newContent: string) => void
  onClose: () => void
  /** v123 新增:教案 ID(用于上传图片关联) */
  planID?: string
}

export function ManualEditBox({ initialContent, onSave, onClose, planID }: ManualEditBoxProps) {
  const [text, setText]       = useState(initialContent)
  const [saving, setSaving]   = useState(false)
  const [preview, setPreview] = useState(false)

  // v123: 图片上传状态
  const [uploading, setUploading]   = useState(false)
  const [uploadError, setUploadError] = useState<string | null>(null)

  // 编辑区 textarea 引用,用于在光标位置插入 markdown
  const textareaRef = useRef<HTMLTextAreaElement>(null)
  // 隐藏的文件 input 引用,点击图片按钮时触发
  const fileInputRef = useRef<HTMLInputElement>(null)

  // 字数统计:去除 Markdown 符号后按 Unicode 字符计数
  const charCount = text.replace(/[#*`>_~[\]()!-]/g, '').replace(/\s+/g, '').length

  /**
   * 在 textarea 当前光标位置插入文本
   * 如果 textarea 不可用,则追加到末尾
   */
  const insertAtCursor = (insertText: string) => {
    const ta = textareaRef.current
    if (!ta) {
      setText(prev => prev + '\n' + insertText + '\n')
      return
    }
    const start = ta.selectionStart
    const end   = ta.selectionEnd
    const before = text.slice(0, start)
    const after  = text.slice(end)
    // 智能加换行:如果前面不是换行,补一个;后面同理
    const needPrefixNL = before.length > 0 && !before.endsWith('\n')
    const needSuffixNL = after.length > 0 && !after.startsWith('\n')
    const wrapped = (needPrefixNL ? '\n' : '') + insertText + (needSuffixNL ? '\n' : '')
    const newText = before + wrapped + after
    setText(newText)
    // 光标定位到插入内容之后(异步,等 React 重渲染完)
    setTimeout(() => {
      const pos = start + wrapped.length
      ta.focus()
      ta.setSelectionRange(pos, pos)
    }, 0)
  }

  /**
   * v123 上传图片核心函数
   * 适用于点击按钮、拖拽、粘贴三种触发场景
   */
  const handleUploadImage = async (file: File) => {
    setUploadError(null)
    if (!planID) {
      setUploadError('教案 ID 缺失,无法上传')
      setTimeout(() => setUploadError(null), 4000)
      return
    }
    // 前端预校验
    const validateErr = validateImageFile(file)
    if (validateErr) {
      setUploadError(validateErr)
      setTimeout(() => setUploadError(null), 4000)
      return
    }
    setUploading(true)
    try {
      // 默认 alt 用文件名(去扩展名)
      const altGuess = file.name.replace(/\.[^.]+$/, '')
      const resp = await uploadAsset(planID, file, altGuess)
      // 后端已返回拼好的 markdown,直接插入
      insertAtCursor(resp.markdown)
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : '上传失败'
      setUploadError('图片上传失败: ' + msg)
      setTimeout(() => setUploadError(null), 5000)
    } finally {
      setUploading(false)
    }
  }

  // 点击"📷 插入图片"按钮 → 触发隐藏 file input
  const handleClickImageBtn = () => {
    fileInputRef.current?.click()
  }

  // file input 选择文件后触发
  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      handleUploadImage(file)
    }
    // 清空 input value 以便选同一文件还能再次触发
    e.target.value = ''
  }

  // textarea 拖拽放下 → 上传图片
  const handleDrop = (e: React.DragEvent<HTMLTextAreaElement>) => {
    e.preventDefault()
    const file = e.dataTransfer.files?.[0]
    if (file && file.type.startsWith('image/')) {
      handleUploadImage(file)
    }
  }

  // textarea 粘贴 → 检测剪贴板里有没有图片(截图常见)
  const handlePaste = (e: React.ClipboardEvent<HTMLTextAreaElement>) => {
    const items = e.clipboardData?.items
    if (!items) return
    for (let i = 0; i < items.length; i++) {
      const item = items[i]
      if (item.type.startsWith('image/')) {
        const file = item.getAsFile()
        if (file) {
          e.preventDefault() // 阻止默认粘贴(防止把二进制粘成乱码)
          handleUploadImage(file)
          return
        }
      }
    }
    // 没有图片时按默认行为继续(粘贴文本)
  }

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
      {/* 标题栏:左侧标题+图片按钮,右侧切换预览按钮 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: '8px' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <div style={{ fontSize: '13px', fontWeight: 600, color: '#374151' }}>
            ✏️ 手动编辑段落
          </div>
          {/* v123: 图片插入按钮(预览模式下隐藏) */}
          {!preview && planID && (
            <button
              onClick={handleClickImageBtn}
              disabled={uploading}
              title="支持点击选择 / 拖拽 / 粘贴 上传图片"
              style={{
                padding: '3px 10px', borderRadius: '6px', fontSize: '12px',
                border: '1px solid #D1D5DB',
                background: uploading ? '#F3F4F6' : '#fff',
                color: uploading ? '#9CA3AF' : '#374151',
                cursor: uploading ? 'wait' : 'pointer', transition: 'all 150ms ease',
                display: 'inline-flex', alignItems: 'center', gap: '4px',
              }}
            >
              {uploading ? '⏳ 上传中…' : '📷 插入图片'}
            </button>
          )}
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

      {/* v123: 上传错误提示条(瞬时,3-5 秒后自动消失) */}
      {uploadError && (
        <div style={{
          marginBottom: '8px', padding: '6px 10px', borderRadius: '6px',
          background: 'rgba(239,68,68,0.08)', border: '1px solid rgba(239,68,68,0.25)',
          fontSize: '12px', color: '#DC2626',
        }}>
          ⚠️ {uploadError}
        </div>
      )}

      {/* 编辑区 / 预览区 */}
      {preview ? (
        <div style={{
          minHeight: '120px', padding: '10px 12px', borderRadius: '8px',
          border: '1px solid #E5E7EB', background: '#fff',
          fontSize: '13px', lineHeight: 1.9, color: '#374151',
          overflowY: 'auto',
        }}>
          {text.trim() ? renderMarkdown(text) : (
            <span style={{ color: '#9CA3AF', fontStyle: 'italic' }}>(暂无内容)</span>
          )}
        </div>
      ) : (
        <textarea
          ref={textareaRef}
          value={text}
          onChange={e => setText(e.target.value)}
          onDrop={handleDrop}
          onDragOver={e => e.preventDefault()}
          onPaste={handlePaste}
          rows={6}
          disabled={uploading}
          style={{
            width: '100%', boxSizing: 'border-box',
            padding: '10px 12px', borderRadius: '8px',
            border: '1px solid #D1D5DB', fontSize: '13px',
            lineHeight: 1.7, color: '#374151',
            resize: 'vertical', fontFamily: 'inherit', outline: 'none',
            transition: 'border-color 150ms ease',
            background: uploading ? '#F9FAFB' : '#fff',
          }}
          onFocus={e => { e.target.style.borderColor = '#3B82F6' }}
          onBlur={e  => { e.target.style.borderColor = '#D1D5DB' }}
        />
      )}

      {/* v123: 隐藏 file input,只接受图片 */}
      <input
        ref={fileInputRef}
        type="file"
        accept="image/jpeg,image/png,image/webp,image/gif"
        style={{ display: 'none' }}
        onChange={handleFileChange}
      />

      {/* 底部:字数统计 + 操作按钮 */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: '10px' }}>
        <div style={{ fontSize: '11px', color: '#9CA3AF' }}>
          约 <span style={{ fontWeight: 600, color: charCount > 0 ? '#6B7280' : '#9CA3AF' }}>{charCount}</span> 字
          {!preview && (
            <span style={{ marginLeft: '6px', opacity: 0.7 }}>(支持 Markdown · 拖拽/粘贴图片可上传)</span>
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
            disabled={saving || uploading || !text.trim()}
            style={{
              padding: '7px 16px', borderRadius: '7px', border: 'none',
              background: text.trim() && !uploading ? '#10B981' : '#E5E7EB',
              color: text.trim() && !uploading ? '#fff' : '#9CA3AF',
              fontSize: '13px', fontWeight: 600,
              cursor: text.trim() && !uploading ? 'pointer' : 'not-allowed',
              transition: 'background 150ms ease',
            }}
          >{saving ? '保存中...' : '✓ 保存'}</button>
        </div>
      </div>
    </div>
  )
}

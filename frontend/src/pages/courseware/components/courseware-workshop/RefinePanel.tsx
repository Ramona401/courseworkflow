/**
 * RefinePanel.tsx — 页面微调面板（批次W2从主页面抽出，工作台默认Tab）
 *
 * 内容：单页AI微调（支持附截图/Ctrl+V粘贴截图走多模态）+ 单页从零重生。
 * 选中页跟随上方大预览框（批次4b口径，pageNum=父级buildPreviewNum）。
 * W2改进：自带消息条——原Step5没有buildMessage的展示位，微调成功/失败提示此前不可见。
 */
import { useState } from 'react'
import { refinePage, regenerateCWPage } from '@/api/coursewares'
import { C } from './workshopConstants'

interface Props {
  coursewareId: string
  /** 当前选中页（父级 buildPreviewNum） */
  pageNum: number
  /** 微调/重生成功后回写该页HTML（父级更新 generatedPages 刷新预览） */
  onPageUpdated: (pageNum: number, html: string) => void
}

export default function RefinePanel({ coursewareId, pageNum, onPageUpdated }: Props) {
  const [refineInput, setRefineInput] = useState('')
  const [refineRunning, setRefineRunning] = useState(false)
  const [refineImage, setRefineImage] = useState('')   // 截图dataURI(走多模态)
  const [regenRunning, setRegenRunning] = useState(false)
  const [message, setMessage] = useState('')

  // 单页AI微调(批次4a: 支持随附截图走多模态; 微调=保留页内已插入图片)
  const handleRefinePage = async () => {
    if (!coursewareId || pageNum <= 0 || !refineInput.trim()) return
    setRefineRunning(true)
    try {
      const result = await refinePage(coursewareId, pageNum, refineInput.trim(), refineImage || undefined)
      if (result.html_content) onPageUpdated(pageNum, result.html_content)
      setRefineInput(''); setRefineImage('')
      setMessage('✅ ' + result.message)
    } catch (e) { setMessage('❌ 微调失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setRefineRunning(false) }
  }

  // 单页从零重生(重生=不保留页内已插入图片; 后端无并发锁故运行态禁用按钮)
  const handleRegeneratePage = async () => {
    if (!coursewareId || pageNum <= 0 || regenRunning || refineRunning) return
    if (!confirm('⚠️ 重生第 ' + pageNum + ' 页将按方案从零重画整页，会清空本页已插入的图片（图片资产仍在多媒体库，可重新插入）。确定重生？')) return
    setRegenRunning(true); setMessage('🔄 正在重生第 ' + pageNum + ' 页，请稍候...')
    try {
      const result = await regenerateCWPage(coursewareId, pageNum)
      if (result.html_content) onPageUpdated(pageNum, result.html_content)
      setMessage('✅ ' + result.message)
    } catch (e) { setMessage('❌ 重生失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setRegenRunning(false) }
  }

  // 共用——将图片文件读为 dataURI 存入 refineImage(8MB上限)
  const loadRefineImageFile = (f: File, fromPaste = false) => {
    if (f.size > 8 * 1024 * 1024) { setMessage('❌ 截图不能超过8MB'); return }
    const reader = new FileReader()
    reader.onload = () => {
      setRefineImage(typeof reader.result === 'string' ? reader.result : '')
      if (fromPaste) setMessage('✅ 已从剪贴板粘贴截图，微调将参考该图')
    }
    reader.onerror = () => setMessage('❌ 截图读取失败')
    reader.readAsDataURL(f)
  }

  // 微调输入框 Ctrl+V 粘贴剪贴板图片(仅含图片时拦截; 纯文本粘贴走默认行为)
  const handleRefinePaste = (e: React.ClipboardEvent<HTMLInputElement>) => {
    const items = e.clipboardData?.items
    if (!items) return
    for (let i = 0; i < items.length; i++) {
      const it = items[i]
      if (it.type && it.type.startsWith('image/')) {
        const f = it.getAsFile()
        if (!f) continue
        e.preventDefault()
        loadRefineImageFile(f, true)
        return
      }
    }
  }

  return (
    <div style={{ marginTop: 16, padding: '16px', borderRadius: 10, border: `1px solid ${C.border}`, background: '#FAFAFA' }}>
      <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>🎨 对某页不满意？在上方预览区选中该页，输入修改意见</div>
      <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', alignItems: 'center' }}>
        <span style={{ padding: '8px 12px', borderRadius: 8, background: C.primaryBg, color: C.primary, fontSize: 13, fontWeight: 600, whiteSpace: 'nowrap' }}>
          当前：第 {pageNum || '—'} 页
        </span>
        <input value={refineInput} onChange={e => setRefineInput(e.target.value)}
          placeholder="例如：标题字号再大一些、增加图片占位...（可 Ctrl+V 粘贴截图，先在上方选要改的页）"
          onKeyDown={e => { if (e.key === 'Enter' && !refineRunning && pageNum > 0) handleRefinePage() }}
          onPaste={handleRefinePaste}
          style={{ flex: 1, padding: '10px 14px', borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 14, outline: 'none', minWidth: 200 }}
          disabled={refineRunning} />
        <button onClick={handleRefinePage} disabled={refineRunning || pageNum <= 0 || !refineInput.trim()}
          style={{ padding: '10px 20px', borderRadius: 8, border: 'none', background: pageNum > 0 && refineInput.trim() && !refineRunning ? '#7C3AED' : '#E5E7EB', color: pageNum > 0 && refineInput.trim() && !refineRunning ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: pageNum > 0 && refineInput.trim() && !refineRunning ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
          {refineRunning ? '⏳ 微调中...' : '🎨 AI微调'}
        </button>
      </div>
      {/* 截图粘贴 + 重生本页 */}
      <div style={{ display: 'flex', gap: 10, marginTop: 10, flexWrap: 'wrap', alignItems: 'center' }}>
        {refineImage ? (
          <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
            <img src={refineImage} alt="参考截图" style={{ width: 40, height: 40, objectFit: 'cover', borderRadius: 6, border: '2px solid #7C3AED' }} />
            <span style={{ fontSize: 11, color: '#7C3AED' }}>已附截图(微调将参考)</span>
            <button onClick={() => setRefineImage('')} disabled={refineRunning || regenRunning} style={{ padding: '2px 8px', borderRadius: 4, border: '1px solid #EF4444', background: 'transparent', color: '#EF4444', fontSize: 11, cursor: (refineRunning || regenRunning) ? 'default' : 'pointer' }}>移除</button>
          </div>
        ) : (
          <button onClick={() => {
            const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/*'
            inp.onchange = (ev) => {
              const f = (ev.target as HTMLInputElement).files?.[0]
              if (!f) return
              loadRefineImageFile(f)
            }; inp.click()
          }} disabled={refineRunning || regenRunning} style={{ padding: '8px 14px', borderRadius: 8, border: '1px dashed #7C3AED', background: 'rgba(124,58,237,0.04)', color: '#7C3AED', fontSize: 13, cursor: (refineRunning || regenRunning) ? 'default' : 'pointer' }}>📷 附截图微调（或在输入框 Ctrl+V 粘贴）</button>
        )}
        <div style={{ flex: 1 }} />
        <button onClick={handleRegeneratePage} disabled={pageNum <= 0 || regenRunning || refineRunning}
          title={pageNum <= 0 ? '请先在上方预览区选中页' : '按方案从零重画本页(会清空本页已插入的图片)'}
          style={{ padding: '8px 18px', borderRadius: 8, border: 'none', background: (pageNum > 0 && !regenRunning && !refineRunning) ? 'linear-gradient(135deg, #F59E0B, #EF4444)' : '#E5E7EB', color: (pageNum > 0 && !regenRunning && !refineRunning) ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (pageNum > 0 && !regenRunning && !refineRunning) ? 'pointer' : 'default', whiteSpace: 'nowrap' }}>
          {regenRunning ? '⏳ 重生中...' : '🔄 重生本页'}
        </button>
      </div>
      <div style={{ marginTop: 6, fontSize: 11, color: '#9CA3AF' }}>💡 微调=在现有页面上增量修改、保留已插图片；重生=按方案从零重画、不保留已插图片。页面变形/损坏时用重生补救。截图除「附截图微调」选文件外，也可在微调输入框直接 Ctrl+V 粘贴。</div>
      {/* W2: 自带消息条(原Step5无展示位, 微调结果此前不可见) */}
      {message && (
        <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, fontSize: 13, background: message.startsWith('❌') ? '#FEE2E2' : message.startsWith('✅') ? '#D1FAE5' : '#EFF6FF', color: message.startsWith('❌') ? '#DC2626' : message.startsWith('✅') ? '#059669' : '#2563EB' }}>{message}</div>
      )}
    </div>
  )
}

/**
 * InlineTextEditor.tsx — 课件页就地编辑器。
 *
 * 支持文字、图片、视频和绝对定位模块四种模式：
 * 文字保留行内格式；图片支持替换、AI生成与原位转视频；
 * 视频支持上传替换和AI重新生成；模块支持拖拽、缩放及精确定位。
 * iframe内部识别与DOM写回协议位于 inlineEditorInject.ts。
 */

import { useState, useEffect, useRef, useCallback } from 'react'
import { getCoursewarePages, savePageHtml, uploadCWImage, generateCWImage, uploadCWVideo, generateCWVideo, queryVideoStatus } from '@/api/coursewares'
import { C, CW_WIDTH, CW_HEIGHT } from './workshopConstants'
import { injectEditor } from './inlineEditorInject'

// ==================== 类型定义 ====================

interface Props {
  coursewareId: string
  pageNum: number
  onPageUpdated: (pageNum: number, html: string) => void
  onClose: () => void
}

interface TextSelectedInfo {
  mode: 'text'
  path: string
  text: string
  fontSizePx: number
  color: string
  hasSelection: boolean
}

interface ImageSelectedInfo {
  mode: 'image'
  path: string
  src: string
  width: number
  height: number
  naturalWidth: number
  naturalHeight: number
  isVideoPlaceholder: boolean
}

interface VideoSelectedInfo {
  mode: 'video'
  path: string
  src: string
  poster: string
  width: number
  height: number
}

interface BlockSelectedInfo {
  mode: 'block'
  path: string
  tagName: string
  left: number
  top: number
  width: number
  height: number
  hasText: boolean
  /** v5.6 新增：元素的 HTML id 属性（可空） */
  elementId?: string
  /** v5.6 新增：元素的 CSS class 属性（可空） */
  elementClass?: string
}

type SelectedInfo = TextSelectedInfo | ImageSelectedInfo | VideoSelectedInfo | BlockSelectedInfo

// ==================== 工具函数 ====================

/** rgb(a) → #rrggbb */
function rgbToHex(rgb: string): string {
  if (!rgb) return '#000000'
  if (rgb.startsWith('#')) return rgb
  const m = rgb.match(/rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i)
  if (!m) return '#000000'
  const toHex = (n: string) => {
    const h = Math.max(0, Math.min(255, parseInt(n, 10))).toString(16)
    return h.length === 1 ? '0' + h : h
  }
  return '#' + toHex(m[1]) + toHex(m[2]) + toHex(m[3])
}

/** 常用字体选项 */
const FONT_OPTIONS = [
  { label: '默认（跟随模板）', value: '' },
  { label: '黑体', value: "'Microsoft YaHei', 'PingFang SC', sans-serif" },
  { label: '宋体', value: "'SimSun', 'Songti SC', serif" },
  { label: '楷体', value: "'KaiTi', 'STKaiti', serif" },
  { label: 'Arial', value: 'Arial, Helvetica, sans-serif' },
  { label: 'Georgia', value: 'Georgia, serif' },
  { label: '等宽', value: "'Courier New', Consolas, monospace" },
]

/** 快捷色板 */
const QUICK_COLORS = ['#1F2937', '#EF4444', '#F59E0B', '#059669', '#2563EB', '#7C3AED', '#FFFFFF', '#0EA5E9', '#EC4899', '#8B5CF6']

/**
 * v5.6 新增：从 elementId 或 elementClass 提取人类可读的元素标识
 * 优先用 id（如 "hotspot-eyepiece" → "eyepiece"），
 * 其次用 class 中最有意义的类名（如 "hotspot matched" → "hotspot"）
 */
function getElementLabel(info: BlockSelectedInfo): string {
  if (info.elementId) {
    // 去掉常见前缀让标识更简洁
    return '#' + info.elementId
  }
  if (info.elementClass) {
    // 取第一个非通用的类名
    const classes = info.elementClass.trim().split(/\s+/)
    const meaningful = classes.find(c =>
      c !== 'matched' && c !== 'active' && c !== 'disabled' && c !== 'show' &&
      c !== 'completed' && c !== 'hidden' && !c.startsWith('tedna-')
    )
    if (meaningful) return '.' + meaningful
  }
  return info.tagName
}

// ==================== 主组件 ====================

export default function InlineTextEditor({ coursewareId, pageNum, onPageUpdated, onClose }: Props) {
  /* 基础状态 */
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [saving, setSaving] = useState(false)
  const [dirty, setDirty] = useState(false)
  const [selected, setSelected] = useState<SelectedInfo | null>(null)
  /* 文字选区 */
  const [hasTextSelection, setHasTextSelection] = useState(false)
  const [selectionColor, setSelectionColor] = useState('#EF4444')
  /* 图片 AI */
  const [imageUploading, setImageUploading] = useState(false)
  const [imageGenerating, setImageGenerating] = useState(false)
  const [imagePrompt, setImagePrompt] = useState('')
  const [imageMode, setImageMode] = useState<'none' | 'ref' | 'new'>('none')
  /* 任意图片→原位真视频 */
  const [videoGenStatus, setVideoGenStatus] = useState<'' | 'submitting' | 'polling' | 'done' | 'failed'>('')
  const [videoGenMessage, setVideoGenMessage] = useState('')
  const [videoPromptForGen, setVideoPromptForGen] = useState('')
  const [videoGenMode, setVideoGenMode] = useState<'none' | 'ref' | 'new'>('none')
  /* 真 <video> 上传 */
  const [videoUploading, setVideoUploading] = useState(false)
  const [videoUploadProgress, setVideoUploadProgress] = useState(0)

  const iframeRef = useRef<HTMLIFrameElement | null>(null)
  const tokenRef = useRef<string>(Math.random().toString(36).slice(2) + Date.now().toString(36))
  const isFullDocRef = useRef<boolean>(true)
  const srcDocRef = useRef<string>('')
  const exportResolveRef = useRef<((html: string) => void) | null>(null)
  const scaleWrapRef = useRef<HTMLDivElement | null>(null)

  // ==================== 初始化 ====================

  useEffect(() => {
    let alive = true
    ;(async () => {
      setLoading(true); setError('')
      try {
        const pages = await getCoursewarePages(coursewareId)
        if (!alive) return
        const page = (pages || []).find(p => p.page_number === pageNum)
        const html = page?.html_content || ''
        if (!html.trim()) { setError('该页暂无内容，无法编辑'); setLoading(false); return }
        isFullDocRef.current = /<html[\s>]/i.test(html)
        srcDocRef.current = injectEditor(html, tokenRef.current, isFullDocRef.current)
        setLoading(false)
      } catch (e) {
        if (!alive) return
        setError('加载页面失败: ' + (e instanceof Error ? e.message : '未知错误'))
        setLoading(false)
      }
    })()
    return () => { alive = false }
  }, [coursewareId, pageNum])

  // ==================== iframe 消息 ====================

  useEffect(() => {
    const onMsg = (e: MessageEvent) => {
      const d = e.data
      if (!d || d.__tedna !== tokenRef.current) return
      if (d.type === 'select') {
        setHasTextSelection(false)
        setImageMode('none'); setVideoGenMode('none'); setVideoUploadProgress(0)
        const m = d.payload.mode
        if (m === 'image') {
          setSelected({ mode: 'image', path: d.payload.path, src: d.payload.src || '', width: d.payload.width || 0, height: d.payload.height || 0, naturalWidth: d.payload.naturalWidth || 0, naturalHeight: d.payload.naturalHeight || 0, isVideoPlaceholder: !!d.payload.isVideoPlaceholder })
        } else if (m === 'video') {
          setSelected({
            mode: 'video',
            path: d.payload.path,
            src: d.payload.src || '',
            poster: d.payload.poster || '',
            width: d.payload.width || 0,
            height: d.payload.height || 0,
          })
        } else if (m === 'block') {
          setSelected({
            mode: 'block', path: d.payload.path, tagName: d.payload.tagName || 'DIV',
            left: d.payload.left || 0, top: d.payload.top || 0,
            width: d.payload.width || 0, height: d.payload.height || 0,
            hasText: !!d.payload.hasText,
            /* v5.6: 接收元素标识信息 */
            elementId: d.payload.elementId || '',
            elementClass: d.payload.elementClass || '',
          })
        } else {
          setSelected({ mode: 'text', path: d.payload.path, text: d.payload.text || '', fontSizePx: d.payload.fontSizePx || 16, color: d.payload.color || 'rgb(0,0,0)', hasSelection: false })
        }
      } else if (d.type === 'deselect') {
        /* v5.7: iframe 内点空白处或按 ESC 取消选中 → 面板同步回到未选中提示态 */
        setSelected(null)
        setHasTextSelection(false)
        setImageMode('none'); setVideoGenMode('none'); setVideoUploadProgress(0)
      } else if (d.type === 'block_update') {
        setSelected(prev => {
          if (!prev || prev.mode !== 'block') return prev
          return { ...prev, left: d.payload.left, top: d.payload.top, width: d.payload.width, height: d.payload.height }
        })
        setDirty(true)
      } else if (d.type === 'selection_change') {
        setHasTextSelection(!!d.payload.hasSelection)
      } else if (d.type === 'exported') {
        const cb = exportResolveRef.current
        exportResolveRef.current = null
        if (cb) cb(d.payload.html || '')
      }
    }
    window.addEventListener('message', onMsg)
    return () => window.removeEventListener('message', onMsg)
  }, [])

  // ==================== 缩放 ====================

  const applyScale = useCallback(() => {
    const el = scaleWrapRef.current
    if (!el) return
    const p = el.parentElement
    if (!p) return
    const s = Math.min(p.clientWidth / CW_WIDTH, p.clientHeight / CW_HEIGHT)
    el.style.setProperty('--edit-scale', String(s > 0 ? s : 0.1))
  }, [])

  useEffect(() => {
    if (loading) return
    applyScale()
    window.addEventListener('resize', applyScale)
    return () => window.removeEventListener('resize', applyScale)
  }, [loading, applyScale])

  // ==================== 通用 iframe 通信 ====================

  const postToIframe = (type: string, payload: Record<string, unknown>) => {
    const win = iframeRef.current?.contentWindow
    if (!win) return
    win.postMessage({ __tedna: tokenRef.current, type, payload }, '*')
  }

  // ==================== 文字操作 ====================

  const sendApply = (patch: Record<string, unknown>) => {
    if (!selected || selected.mode !== 'text') return
    postToIframe('apply', { path: selected.path, ...patch })
    setDirty(true)
  }
  const onChangeText = (text: string) => {
    if (!selected || selected.mode !== 'text') return
    setSelected({ ...selected, text }); sendApply({ text })
  }
  const onChangeFontSize = (px: number) => {
    if (!selected || selected.mode !== 'text') return
    const v = Math.max(8, Math.min(200, Math.round(px || 0)))
    setSelected({ ...selected, fontSizePx: v }); sendApply({ fontSizePx: v })
  }
  const onChangeColor = (hex: string) => {
    if (!selected || selected.mode !== 'text') return
    setSelected({ ...selected, color: hex }); sendApply({ color: hex })
  }
  const sendFormat = (action: string, value?: string) => {
    if (!selected || selected.mode !== 'text') return
    postToIframe('format', { action, value })
    setDirty(true); setHasTextSelection(false)
  }

  // ==================== 图片操作 ====================

  const replaceImgInIframe = (newUrl: string) => {
    if (!newUrl || !selected || selected.mode !== 'image') return
    postToIframe('replace_image', { path: selected.path, src: newUrl })
    setSelected({ ...selected, src: newUrl }); setDirty(true)
  }

  const handleReplaceImage = () => {
    if (!selected || selected.mode !== 'image') return
    const inp = document.createElement('input')
    inp.type = 'file'; inp.accept = 'image/jpeg,image/png,image/webp,image/gif,image/svg+xml'
    inp.onchange = async (ev) => {
      const f = (ev.target as HTMLInputElement).files?.[0]
      if (!f) return
      if (f.size > 5 * 1024 * 1024) { setError('图片不能超过5MB'); return }
      setImageUploading(true); setError('')
      try {
        const result = await uploadCWImage(coursewareId, pageNum, f)
        replaceImgInIframe(result.url)
      } catch (e) { setError('上传图片失败: ' + (e instanceof Error ? e.message : '未知错误'))
      } finally { setImageUploading(false) }
    }
    inp.click()
  }

  const handleAIImage = async () => {
    if (!selected || selected.mode !== 'image' || !imagePrompt.trim() || imageGenerating) return
    setImageGenerating(true); setError('')
    try {
      const refUrl = imageMode === 'ref' ? selected.src : undefined
      const result = await generateCWImage(coursewareId, pageNum, imagePrompt.trim(), undefined, '1920x1920', refUrl)
      const newUrl = result.url || ''
      if (!newUrl) throw new Error('AI 生成成功但未返回图片地址')
      replaceImgInIframe(newUrl); setImagePrompt(''); setImageMode('none')
    } catch (e) { setError('AI 生成图片失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally { setImageGenerating(false) }
  }

  // ==================== 图片/已有视频就地生成视频 ====================

  const applyVideoToTarget = (
    target: ImageSelectedInfo | VideoSelectedInfo,
    videoUrl: string,
  ) => {
    if (!videoUrl) return

    if (target.mode === 'image') {
      postToIframe(
        'replace_img_with_video',
        { path: target.path, src: videoUrl },
      )
      setSelected(prev => (
        prev?.mode === 'image' && prev.path === target.path
          ? null
          : prev
      ))
    } else {
      postToIframe(
        'replace_video',
        { path: target.path, src: videoUrl },
      )
      setSelected(prev => (
        prev?.mode === 'video' && prev.path === target.path
          ? { ...prev, src: videoUrl }
          : prev
      ))
    }

    setDirty(true)
  }

  const handleUploadVideoForImage = () => {
    if (!selected || selected.mode !== 'image') return
    const target = selected
    const inp = document.createElement('input')
    inp.type = 'file'; inp.accept = 'video/mp4,video/webm,video/quicktime'
    inp.onchange = async (ev) => {
      const f = (ev.target as HTMLInputElement).files?.[0]
      if (!f) return
      if (f.size > 50 * 1024 * 1024) { setError('视频不能超过50MB'); return }
      setVideoGenStatus('submitting'); setVideoGenMessage('⏳ 正在上传视频...'); setError('')
      try {
        const result = await uploadCWVideo(coursewareId, pageNum, f, (pct) => setVideoGenMessage('⏳ 上传中 ' + pct + '%...'))
        applyVideoToTarget(target, result.url)
        setVideoGenStatus('done'); setVideoGenMessage('✅ 视频已上传并原位替换当前图片'); setVideoGenMode('none')
      } catch (e) {
        setVideoGenStatus('failed'); setVideoGenMessage('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误'))
      }
    }
    inp.click()
  }

  const handleAIVideoGen = async () => {
    if (
      !selected ||
      (selected.mode !== 'image' && selected.mode !== 'video') ||
      !videoPromptForGen.trim()
    ) return

    const target = selected
    const refUrl = videoGenMode === 'ref'
      ? (
          target.mode === 'image'
            ? target.src
            : target.poster
        )
      : undefined

    if (videoGenMode === 'ref' && !refUrl) {
      setVideoGenStatus('failed')
      setVideoGenMessage('❌ 当前媒体没有可用封面图，请改用“AI全新生成视频”')
      return
    }

    setVideoGenStatus('submitting'); setVideoGenMessage('⏳ 正在提交视频生成任务...'); setError('')
    try {
      const result = await generateCWVideo(coursewareId, pageNum, videoPromptForGen.trim(), refUrl)
      if (!result.asset_id) throw new Error('未返回任务ID')
      setVideoGenStatus('polling'); setVideoGenMessage('⏳ AI 正在生成视频（约30-120秒）...')
      let attempts = 0
      const poll = async () => {
        attempts++
        try {
          const status = await queryVideoStatus(coursewareId, result.asset_id)
          if (status.status === 'uploaded' && status.video_url) {
            applyVideoToTarget(target, status.video_url)
            setVideoGenStatus('done')
            setVideoGenMessage(
              target.mode === 'image'
                ? '✅ 视频已生成并原位替换当前图片'
                : '✅ 新视频已生成并替换当前视频',
            )
            setVideoPromptForGen(''); setVideoGenMode('none'); return
          } else if (status.status === 'failed') {
            setVideoGenStatus('failed'); setVideoGenMessage('❌ 视频生成失败: ' + (status.error_msg || '未知错误')); return
          }
          if (attempts >= 60) { setVideoGenStatus('failed'); setVideoGenMessage('❌ 超时，请到媒体管理Tab查看'); return }
          setVideoGenMessage('⏳ AI 生成中（已等 ' + (attempts * 5) + ' 秒）...')
          setTimeout(poll, 5000)
        } catch (e) { setVideoGenStatus('failed'); setVideoGenMessage('❌ 查询失败: ' + (e instanceof Error ? e.message : '')) }
      }
      setTimeout(poll, 5000)
    } catch (e) { setVideoGenStatus('failed'); setVideoGenMessage('❌ 提交失败: ' + (e instanceof Error ? e.message : '')) }
  }

  // ==================== 真 <video> 上传替换 ====================

  const handleReplaceVideo = () => {
    if (!selected || selected.mode !== 'video') return
    const inp = document.createElement('input')
    inp.type = 'file'; inp.accept = 'video/mp4,video/webm,video/quicktime,video/x-msvideo'
    inp.onchange = async (ev) => {
      const f = (ev.target as HTMLInputElement).files?.[0]
      if (!f) return
      if (f.size > 50 * 1024 * 1024) { setError('视频不能超过50MB'); return }
      setVideoUploading(true); setVideoUploadProgress(0); setError('')
      try {
        const result = await uploadCWVideo(coursewareId, pageNum, f, (pct) => setVideoUploadProgress(pct))
        postToIframe('replace_video', { path: selected.path, src: result.url })
        setSelected({ ...selected, src: result.url }); setDirty(true)
      } catch (e) { setError('上传视频失败: ' + (e instanceof Error ? e.message : ''))
      } finally { setVideoUploading(false); setVideoUploadProgress(0) }
    }
    inp.click()
  }

  // ==================== 图片尺寸调整 ====================

  const onResizeImage = (w: number, h: number) => {
    if (!selected || selected.mode !== 'image') return
    postToIframe('resize_image', { path: selected.path, width: w, height: h })
    setSelected({ ...selected, width: w, height: h }); setDirty(true)
  }

  // ==================== 块元素数字调整 ====================

  const sendBlockUpdate = (patch: Partial<{ left: number; top: number; width: number; height: number }>) => {
    if (!selected || selected.mode !== 'block') return
    postToIframe('update_block', { path: selected.path, ...patch })
    setSelected({ ...selected, ...patch } as BlockSelectedInfo); setDirty(true)
  }

  // ==================== 保存 / 关闭 ====================

  const requestExport = (): Promise<string> => {
    return new Promise((resolve, reject) => {
      const win = iframeRef.current?.contentWindow
      if (!win) { reject(new Error('编辑器未就绪')); return }
      const timer = setTimeout(() => { exportResolveRef.current = null; reject(new Error('导出超时')) }, 5000)
      exportResolveRef.current = (html: string) => { clearTimeout(timer); resolve(html) }
      win.postMessage({ __tedna: tokenRef.current, type: 'export', payload: {} }, '*')
    })
  }

  const handleSave = async () => {
    if (saving) return
    if (!dirty) { onClose(); return }
    setSaving(true); setError('')
    try {
      const html = await requestExport()
      if (!html || !html.trim()) throw new Error('导出内容为空')
      const result = await savePageHtml(coursewareId, pageNum, html)
      if (result.html_content) onPageUpdated(pageNum, result.html_content)
      onClose()
    } catch (e) { setError('保存失败: ' + (e instanceof Error ? e.message : ''))
    } finally { setSaving(false) }
  }

  const handleClose = () => {
    if (saving) return
    if (dirty) { if (!confirm('有未保存的修改，确定放弃并关闭吗？')) return }
    onClose()
  }

  // ==================== 渲染：文字工具 ====================

  const renderTextTools = () => {
    if (!selected || selected.mode !== 'text') return null
    const s = selected
    return (
      <>
        <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary }}>📝 已选中文字元素</div>
        <div>
          <label style={labelStyle}>整段文字内容</label>
          <textarea value={s.text} onChange={e => onChangeText(e.target.value)} rows={3}
            style={{ width: '100%', padding: '8px 10px', borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 13, outline: 'none', resize: 'vertical', fontFamily: 'inherit', lineHeight: 1.6, boxSizing: 'border-box' }} />
          <div style={{ fontSize: 11, color: C.textMuted, marginTop: 2 }}>修改这里只替换文字内容，保留原有加粗、颜色、字体等行内格式；新增文字沿用相邻格式</div>
        </div>
        <div>
          <label style={labelStyle}>整段字号（px）</label>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <button onClick={() => onChangeFontSize(s.fontSizePx - 2)} style={stepBtnStyle}>−</button>
            <input type="number" value={s.fontSizePx} onChange={e => onChangeFontSize(parseInt(e.target.value, 10))}
              style={{ width: 60, padding: '6px', borderRadius: 8, border: `1px solid ${C.border}`, fontSize: 13, textAlign: 'center', outline: 'none' }} />
            <button onClick={() => onChangeFontSize(s.fontSizePx + 2)} style={stepBtnStyle}>+</button>
          </div>
        </div>
        <div>
          <label style={labelStyle}>整段颜色</label>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <input type="color" value={rgbToHex(s.color)} onChange={e => onChangeColor(e.target.value)}
              style={{ width: 40, height: 30, borderRadius: 6, border: `1px solid ${C.border}`, cursor: 'pointer', padding: 2 }} />
            <span style={{ fontSize: 12, color: C.textSecondary, fontFamily: 'monospace' }}>{rgbToHex(s.color)}</span>
          </div>
          <div style={{ display: 'flex', gap: 5, marginTop: 6, flexWrap: 'wrap' }}>
            {QUICK_COLORS.map(hex => (
              <button key={hex} onClick={() => onChangeColor(hex)} title={hex}
                style={{ width: 22, height: 22, borderRadius: 5, border: hex === '#FFFFFF' ? `1px solid ${C.border}` : '1px solid transparent', background: hex, cursor: 'pointer' }} />
            ))}
          </div>
        </div>
        <div style={{ borderTop: `1px solid ${C.border}`, margin: '4px 0' }} />
        {/* 选区格式化工具——选即生效 */}
        <div>
          <label style={labelStyle}>
            ✂️ 拖选部分文字 → 直接格式化
            <span style={{ fontWeight: 400, color: hasTextSelection ? '#059669' : C.textMuted, marginLeft: 6 }}>
              {hasTextSelection ? '✅ 已选中，操作即生效' : '在左侧拖选文字'}
            </span>
          </label>
          <button onClick={() => sendFormat('bold')} disabled={!hasTextSelection}
            style={{ ...formatBtnStyle, opacity: hasTextSelection ? 1 : 0.4, cursor: hasTextSelection ? 'pointer' : 'default', fontWeight: 700, marginBottom: 10 }}>
            <b>B</b> 加粗 / 取消加粗
          </button>
          <div style={{ marginBottom: 10 }}>
            <div style={{ fontSize: 11, color: C.textSecondary, marginBottom: 4 }}>🎨 选中文字改色（点色块即生效）</div>
            <div style={{ display: 'flex', gap: 5, flexWrap: 'wrap', alignItems: 'center' }}>
              {QUICK_COLORS.map(hex => (
                <button key={'sel_' + hex} onClick={() => { if (hasTextSelection) sendFormat('color', hex) }} disabled={!hasTextSelection} title={hex}
                  style={{ width: 26, height: 26, borderRadius: 6, border: hex === '#FFFFFF' ? '1px solid ' + C.border : '1px solid transparent', background: hex, cursor: hasTextSelection ? 'pointer' : 'default', opacity: hasTextSelection ? 1 : 0.4 }} />
              ))}
              <input type="color" value={selectionColor} disabled={!hasTextSelection}
                onChange={e => { setSelectionColor(e.target.value); if (hasTextSelection) sendFormat('color', e.target.value) }}
                title="自定义颜色" style={{ width: 26, height: 26, borderRadius: 6, border: '1px solid ' + C.border, cursor: hasTextSelection ? 'pointer' : 'default', padding: 1, opacity: hasTextSelection ? 1 : 0.4 }} />
            </div>
          </div>
          <div>
            <div style={{ fontSize: 11, color: C.textSecondary, marginBottom: 4 }}>Aa 选中文字改字体（选即生效）</div>
            <select value="" disabled={!hasTextSelection}
              onChange={e => { if (e.target.value && hasTextSelection) { sendFormat('font', e.target.value); e.target.value = '' } }}
              style={{ width: '100%', padding: '7px 8px', borderRadius: 6, border: '1px solid ' + C.border, fontSize: 12, outline: 'none', cursor: hasTextSelection ? 'pointer' : 'default', opacity: hasTextSelection ? 1 : 0.5, color: '#6B7280' }}>
              <option value="">选择字体...</option>
              {FONT_OPTIONS.filter(f => f.value).map(f => (<option key={f.value} value={f.value}>{f.label}</option>))}
            </select>
          </div>
        </div>
      </>
    )
  }

  // ==================== 渲染：图片工具 ====================

  const renderImageTools = () => {
    if (!selected || selected.mode !== 'image') return null
    const s = selected
    const busy = imageUploading || imageGenerating
    return (
      <>
        <div style={{ fontSize: 13, fontWeight: 600, color: '#0284C7' }}>🖼️ 已选中图片</div>
        <div style={{ borderRadius: 8, overflow: 'hidden', border: `1px solid ${C.border}`, background: '#F9FAFB' }}>
          {s.src ? (<img src={s.src} alt="当前图片" style={{ width: '100%', maxHeight: 160, objectFit: 'contain', display: 'block' }} />) : (<div style={{ padding: 16, textAlign: 'center', color: C.textMuted, fontSize: 13 }}>无图片源</div>)}
        </div>
        {/* 任意图片原位转视频操作区 */}
        <div style={{ padding: '10px 12px', borderRadius: 8, background: '#FFFBEB', border: '1px solid #F59E0B' }}>
          <div style={{ fontSize: 12, fontWeight: 700, color: '#D97706', marginBottom: 5 }}>
            {s.isVideoPlaceholder
              ? '🎬 检测到视频占位图'
              : '🎬 将当前图片原位转换为视频'}
          </div>
          <div style={{ marginBottom: 8, color: '#92400E', fontSize: 11, lineHeight: 1.5 }}>
            生成或上传成功后只替换当前图片节点，不会清空同一容器中的标题、按钮或装饰；点击“保存修改”后才写回课件。
          </div>
          <div style={{ display: 'flex', flexDirection: 'column', gap: 6 }}>
            <button onClick={handleUploadVideoForImage} disabled={videoGenStatus === 'submitting' || videoGenStatus === 'polling'}
              style={{ padding: '8px 12px', borderRadius: 6, border: 'none', background: '#F59E0B', color: '#fff', fontSize: 12, fontWeight: 600, cursor: (videoGenStatus === 'submitting' || videoGenStatus === 'polling') ? 'default' : 'pointer', width: '100%' }}>
              📤 上传视频原位替换当前图片
            </button>
            <button onClick={() => setVideoGenMode(videoGenMode === 'ref' ? 'none' : 'ref')} disabled={videoGenStatus === 'submitting' || videoGenStatus === 'polling'}
              style={{ padding: '8px 12px', borderRadius: 6, border: videoGenMode === 'ref' ? '2px solid #7C3AED' : '1px solid #E5E7EB', background: videoGenMode === 'ref' ? '#F5F3FF' : '#fff', color: videoGenMode === 'ref' ? '#7C3AED' : '#374151', fontSize: 12, fontWeight: 600, cursor: (videoGenStatus === 'submitting' || videoGenStatus === 'polling') ? 'default' : 'pointer', width: '100%' }}>
              🎨 以此图为首帧 AI 生成视频
            </button>
            <button onClick={() => setVideoGenMode(videoGenMode === 'new' ? 'none' : 'new')} disabled={videoGenStatus === 'submitting' || videoGenStatus === 'polling'}
              style={{ padding: '8px 12px', borderRadius: 6, border: videoGenMode === 'new' ? '2px solid #F59E0B' : '1px solid #E5E7EB', background: videoGenMode === 'new' ? '#FFFBEB' : '#fff', color: videoGenMode === 'new' ? '#D97706' : '#374151', fontSize: 12, fontWeight: 600, cursor: (videoGenStatus === 'submitting' || videoGenStatus === 'polling') ? 'default' : 'pointer', width: '100%' }}>
              ✨ AI 全新生成视频
            </button>
          </div>
          {(videoGenMode === 'ref' || videoGenMode === 'new') && (
            <div style={{ marginTop: 8 }}>
              <textarea value={videoPromptForGen} onChange={e => setVideoPromptForGen(e.target.value)}
                placeholder={videoGenMode === 'ref' ? '描述要生成什么样的视频动效（AI会以当前图为首帧）...' : '描述要生成什么样的视频...'}
                rows={2} disabled={videoGenStatus === 'submitting' || videoGenStatus === 'polling'}
                style={{ width: '100%', padding: '6px 8px', borderRadius: 6, border: '1px solid #E5E7EB', fontSize: 12, outline: 'none', resize: 'vertical', fontFamily: 'inherit', lineHeight: 1.5, boxSizing: 'border-box' }} />
              <button onClick={handleAIVideoGen} disabled={!videoPromptForGen.trim() || videoGenStatus === 'submitting' || videoGenStatus === 'polling'}
                style={{ marginTop: 6, padding: '8px 12px', borderRadius: 6, border: 'none', background: (videoPromptForGen.trim() && videoGenStatus !== 'submitting' && videoGenStatus !== 'polling') ? (videoGenMode === 'ref' ? '#7C3AED' : '#F59E0B') : '#E5E7EB', color: (videoPromptForGen.trim() && videoGenStatus !== 'submitting' && videoGenStatus !== 'polling') ? '#fff' : '#9CA3AF', fontSize: 12, fontWeight: 600, cursor: (videoPromptForGen.trim() && videoGenStatus !== 'submitting' && videoGenStatus !== 'polling') ? 'pointer' : 'default', width: '100%' }}>
                {videoGenStatus === 'polling' ? '⏳ AI 生成中...' : videoGenStatus === 'submitting' ? '⏳ 提交中...' : (videoGenMode === 'ref' ? '🎨 开始生成' : '✨ 开始生成')}
              </button>
            </div>
          )}
          {videoGenMessage && (
            <div style={{ marginTop: 6, padding: '6px 8px', borderRadius: 6, fontSize: 11, lineHeight: 1.5, background: videoGenMessage.startsWith('❌') ? '#FEE2E2' : videoGenMessage.startsWith('✅') ? '#D1FAE5' : '#EFF6FF', color: videoGenMessage.startsWith('❌') ? '#DC2626' : videoGenMessage.startsWith('✅') ? '#059669' : '#2563EB' }}>
              {videoGenMessage}
            </div>
          )}
        </div>
        
        {/* 图片三操作 */}
        <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
          <button onClick={handleReplaceImage} disabled={busy} style={{ padding: '9px 14px', borderRadius: 8, border: 'none', background: busy ? '#E5E7EB' : '#0EA5E9', color: busy ? '#9CA3AF' : '#fff', fontSize: 13, fontWeight: 600, cursor: busy ? 'default' : 'pointer', width: '100%' }}>
            {imageUploading ? '⏳ 上传中...' : '📤 上传本地图片替换'}
          </button>
          <button onClick={() => setImageMode(imageMode === 'ref' ? 'none' : 'ref')} disabled={busy} style={{ padding: '9px 14px', borderRadius: 8, border: imageMode === 'ref' ? '2px solid #7C3AED' : '1px solid ' + C.border, background: imageMode === 'ref' ? '#F5F3FF' : '#fff', color: imageMode === 'ref' ? '#7C3AED' : '#374151', fontSize: 13, fontWeight: 600, cursor: busy ? 'default' : 'pointer', width: '100%' }}>
            🎨 以此图为参考，AI 修改
          </button>
          <button onClick={() => setImageMode(imageMode === 'new' ? 'none' : 'new')} disabled={busy} style={{ padding: '9px 14px', borderRadius: 8, border: imageMode === 'new' ? '2px solid #F59E0B' : '1px solid ' + C.border, background: imageMode === 'new' ? '#FFFBEB' : '#fff', color: imageMode === 'new' ? '#D97706' : '#374151', fontSize: 13, fontWeight: 600, cursor: busy ? 'default' : 'pointer', width: '100%' }}>
            ✨ AI 全新生成图片
          </button>
        </div>
        {imageMode !== 'none' && (
          <div style={{ padding: '12px', borderRadius: 8, border: '1px solid ' + (imageMode === 'ref' ? '#7C3AED' : '#F59E0B'), background: imageMode === 'ref' ? '#FAFAFE' : '#FFFEF5' }}>
            <div style={{ fontSize: 12, fontWeight: 600, color: imageMode === 'ref' ? '#7C3AED' : '#D97706', marginBottom: 6 }}>
              {imageMode === 'ref' ? '🎨 描述要怎么修改这张图' : '✨ 描述你想要的新图片'}
            </div>
            <textarea value={imagePrompt} onChange={e => setImagePrompt(e.target.value)}
              placeholder={imageMode === 'ref' ? '例如：背景换成蓝色天空、人物改成微笑...' : '例如：一个小朋友举手回答问题，皮克斯3D风格...'}
              rows={3} disabled={imageGenerating}
              style={{ width: '100%', padding: '8px 10px', borderRadius: 6, border: '1px solid ' + C.border, fontSize: 13, outline: 'none', resize: 'vertical', fontFamily: 'inherit', lineHeight: 1.5, boxSizing: 'border-box' }} />
            <button onClick={handleAIImage} disabled={imageGenerating || !imagePrompt.trim()}
              style={{ marginTop: 8, padding: '9px 14px', borderRadius: 8, border: 'none', background: (!imageGenerating && imagePrompt.trim()) ? (imageMode === 'ref' ? '#7C3AED' : '#F59E0B') : '#E5E7EB', color: (!imageGenerating && imagePrompt.trim()) ? '#fff' : '#9CA3AF', fontSize: 13, fontWeight: 600, cursor: (!imageGenerating && imagePrompt.trim()) ? 'pointer' : 'default', width: '100%' }}>
              {imageGenerating ? '⏳ AI 生成中...' : (imageMode === 'ref' ? '🎨 生成修改版' : '✨ 生成新图片')}
            </button>
          </div>
        )}
        <div style={{ borderTop: '1px solid ' + C.border, margin: '2px 0' }} />
        <div>
          <label style={labelStyle}>📐 图片尺寸（px）</label>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <div>
              <span style={{ fontSize: 11, color: C.textMuted }}>宽</span>
              <input type="number" value={s.width} min={20} max={1920} onChange={e => { const w = parseInt(e.target.value, 10) || s.width; const r = s.naturalHeight && s.naturalWidth ? s.naturalHeight / s.naturalWidth : s.height / s.width; onResizeImage(w, Math.round(w * r)) }}
                style={{ width: 70, padding: '6px', borderRadius: 6, border: '1px solid ' + C.border, fontSize: 13, textAlign: 'center', outline: 'none', display: 'block', marginTop: 2 }} />
            </div>
            <span style={{ color: C.textMuted, fontSize: 16, marginTop: 14 }}>×</span>
            <div>
              <span style={{ fontSize: 11, color: C.textMuted }}>高</span>
              <input type="number" value={s.height} min={20} max={1080} onChange={e => { const h = parseInt(e.target.value, 10) || s.height; const r = s.naturalWidth && s.naturalHeight ? s.naturalWidth / s.naturalHeight : s.width / s.height; onResizeImage(Math.round(h * r), h) }}
                style={{ width: 70, padding: '6px', borderRadius: 6, border: '1px solid ' + C.border, fontSize: 13, textAlign: 'center', outline: 'none', display: 'block', marginTop: 2 }} />
            </div>
          </div>
          {s.naturalWidth > 0 && (<div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>原始尺寸：{s.naturalWidth} × {s.naturalHeight}</div>)}
        </div>
      </>
    )
  }

  // ==================== 渲染：视频工具 ====================

  const renderVideoTools = () => {
    if (!selected || selected.mode !== 'video') return null
    const s = selected
    const videoBusy = videoGenStatus === 'submitting' || videoGenStatus === 'polling'
    return (
      <>
        <div style={{ fontSize: 13, fontWeight: 600, color: '#D97706' }}>🎬 已选中视频</div>
        <div style={{ borderRadius: 8, overflow: 'hidden', border: '1px solid ' + C.border, background: '#1F2937' }}>
          {s.src ? (<video src={s.src} poster={s.poster || undefined} style={{ width: '100%', maxHeight: 160, display: 'block' }} controls muted />) : (<div style={{ padding: 16, textAlign: 'center', color: '#9CA3AF', fontSize: 13 }}>无视频源</div>)}
        </div>
        <button onClick={handleReplaceVideo} disabled={videoUploading}
          style={{ padding: '10px 14px', borderRadius: 8, border: 'none', background: videoUploading ? '#E5E7EB' : '#F59E0B', color: videoUploading ? '#9CA3AF' : '#fff', fontSize: 13, fontWeight: 600, cursor: videoUploading ? 'default' : 'pointer', width: '100%' }}>
          {videoUploading ? '⏳ 上传中...' : '📤 上传新视频替换'}
        </button>
        {videoUploading && videoUploadProgress > 0 && <div style={{ width: '100%', height: 6, borderRadius: 3, background: '#E5E7EB', overflow: 'hidden' }}><div style={{ width: videoUploadProgress + '%', height: '100%', background: '#F59E0B' }} /></div>}
        <div style={{ padding: 10, borderRadius: 8, border: '1px solid #F59E0B', background: '#FFFBEB' }}>
          <div style={{ fontSize: 12, fontWeight: 700, color: '#D97706', marginBottom: 6 }}>🤖 AI 重新生成当前视频</div>
          <div style={{ display: 'flex', gap: 6 }}>
            <button onClick={() => setVideoGenMode(videoGenMode === 'ref' ? 'none' : 'ref')} disabled={!s.poster || videoBusy} style={{ ...quickBtnStyle, flex: 1, opacity: s.poster ? 1 : 0.45 }}>🎨 参考封面</button>
            <button onClick={() => setVideoGenMode(videoGenMode === 'new' ? 'none' : 'new')} disabled={videoBusy} style={{ ...quickBtnStyle, flex: 1 }}>✨ 全新生成</button>
          </div>
          {videoGenMode !== 'none' && <><textarea value={videoPromptForGen} onChange={e => setVideoPromptForGen(e.target.value)} rows={2} placeholder="描述希望生成的视频内容和动效..." disabled={videoBusy} style={{ width: '100%', marginTop: 7, padding: 7, borderRadius: 6, border: '1px solid #E5E7EB', boxSizing: 'border-box', resize: 'vertical' }} /><button onClick={handleAIVideoGen} disabled={!videoPromptForGen.trim() || videoBusy} style={{ ...quickBtnStyle, width: '100%', marginTop: 6, background: '#F59E0B', color: '#fff' }}>{videoBusy ? '⏳ 生成中...' : '开始生成并替换'}</button></>}
          {videoGenMessage && <div style={{ marginTop: 6, fontSize: 11, color: videoGenMessage.startsWith('❌') ? '#DC2626' : '#2563EB' }}>{videoGenMessage}</div>}
          <div style={{ marginTop: 6, fontSize: 11, color: '#92400E' }}>生成完成后仍需点击“保存修改”才写回课件。</div>
        </div>
        <div style={{ fontSize: 11, color: C.textMuted, lineHeight: 1.6 }}>点击自定义视频封面、播放按钮或封面文字，也会选中其下方真实视频。</div>
      </>
    )
  }

  // ==================== 渲染：块元素工具（v5.6 增强） ====================

  const renderBlockTools = () => {
    if (!selected || selected.mode !== 'block') return null
    const s = selected
    const elLabel = getElementLabel(s)
    return (
      <>
        <div style={{ fontSize: 13, fontWeight: 600, color: '#10B981' }}>
          📦 已选中模块
          <span style={{ fontWeight: 400, color: '#9CA3AF', marginLeft: 4 }}>({s.tagName})</span>
        </div>
        {/* v5.6: 元素标识信息——显示 id 或主 class，帮助区分多个同类热点 */}
        {elLabel !== s.tagName && (
          <div style={{ fontSize: 12, color: '#0EA5E9', padding: '5px 8px', borderRadius: 6, background: '#EFF6FF', border: '1px solid #BFDBFE', fontFamily: 'monospace', wordBreak: 'break-all' }}>
            🏷️ {elLabel}
          </div>
        )}
        {s.hasText && (
          <div style={{ fontSize: 11, color: '#6B7280', padding: '6px 8px', borderRadius: 6, background: '#F0FDF4', border: '1px solid #BBF7D0' }}>
            💡 含文字内容。拖拽/缩放只调位置大小；模块内的文字可直接点选修改，若此模块本身就是文字，<b>再点它一次</b>即可切换为文字编辑。
          </div>
        )}
        <div style={{ fontSize: 11, color: '#059669', padding: '6px 8px', borderRadius: 6, background: '#ECFDF5' }}>
          🖱️ 可直接在左侧画布拖拽移动/缩放，也可输入精确数值。点空白处或按 ESC 取消选中。
        </div>
        {/* 位置控制区：数字输入 + 微调步进按钮 */}
        <div>
          <label style={labelStyle}>📍 位置（px）</label>
          <div style={{ display: 'flex', gap: 12 }}>
            <div>
              <span style={{ fontSize: 11, color: '#9CA3AF' }}>左 (left)</span>
              <div style={{ display: 'flex', alignItems: 'center', gap: 4, marginTop: 2 }}>
                <input type="number" value={s.left} onChange={e => sendBlockUpdate({ left: parseInt(e.target.value, 10) || 0 })}
                  style={blockNumInputStyle} />
                {/* v5.6: 微调步进按钮 */}
                <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  <button onClick={() => sendBlockUpdate({ left: s.left - 1 })} style={microStepBtnStyle} title="左移1px">◀</button>
                  <button onClick={() => sendBlockUpdate({ left: s.left + 1 })} style={microStepBtnStyle} title="右移1px">▶</button>
                </div>
              </div>
            </div>
            <div>
              <span style={{ fontSize: 11, color: '#9CA3AF' }}>上 (top)</span>
              <div style={{ display: 'flex', alignItems: 'center', gap: 4, marginTop: 2 }}>
                <input type="number" value={s.top} onChange={e => sendBlockUpdate({ top: parseInt(e.target.value, 10) || 0 })}
                  style={blockNumInputStyle} />
                <div style={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
                  <button onClick={() => sendBlockUpdate({ top: s.top - 1 })} style={microStepBtnStyle} title="上移1px">▲</button>
                  <button onClick={() => sendBlockUpdate({ top: s.top + 1 })} style={microStepBtnStyle} title="下移1px">▼</button>
                </div>
              </div>
            </div>
          </div>
          {/* v5.6: 批量步进按钮行——±5px 和 ±10px 快速微调 */}
          <div style={{ display: 'flex', gap: 4, marginTop: 6, flexWrap: 'wrap' }}>
            <button onClick={() => sendBlockUpdate({ left: s.left - 10 })} style={stepBatchBtnStyle}>← 10</button>
            <button onClick={() => sendBlockUpdate({ left: s.left - 5 })} style={stepBatchBtnStyle}>← 5</button>
            <button onClick={() => sendBlockUpdate({ left: s.left + 5 })} style={stepBatchBtnStyle}>5 →</button>
            <button onClick={() => sendBlockUpdate({ left: s.left + 10 })} style={stepBatchBtnStyle}>10 →</button>
            <button onClick={() => sendBlockUpdate({ top: s.top - 10 })} style={stepBatchBtnStyle}>↑ 10</button>
            <button onClick={() => sendBlockUpdate({ top: s.top - 5 })} style={stepBatchBtnStyle}>↑ 5</button>
            <button onClick={() => sendBlockUpdate({ top: s.top + 5 })} style={stepBatchBtnStyle}>5 ↓</button>
            <button onClick={() => sendBlockUpdate({ top: s.top + 10 })} style={stepBatchBtnStyle}>10 ↓</button>
          </div>
        </div>
        <div>
          <label style={labelStyle}>📐 尺寸（px）</label>
          <div style={{ display: 'flex', gap: 12 }}>
            <div>
              <span style={{ fontSize: 11, color: '#9CA3AF' }}>宽</span>
              <input type="number" value={s.width} min={30} max={1920} onChange={e => sendBlockUpdate({ width: Math.max(30, parseInt(e.target.value, 10) || 30) })}
                style={{ ...blockNumInputStyle, marginTop: 2 }} />
            </div>
            <div>
              <span style={{ fontSize: 11, color: '#9CA3AF' }}>高</span>
              <input type="number" value={s.height} min={30} max={1080} onChange={e => sendBlockUpdate({ height: Math.max(30, parseInt(e.target.value, 10) || 30) })}
                style={{ ...blockNumInputStyle, marginTop: 2 }} />
            </div>
          </div>
        </div>
        <div>
          <label style={labelStyle}>⚡ 快捷操作</label>
          <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
            <button onClick={() => sendBlockUpdate({ width: Math.round(s.width * 1.1), height: Math.round(s.height * 1.1) })} style={quickBtnStyle}>放大 10%</button>
            <button onClick={() => sendBlockUpdate({ width: Math.max(30, Math.round(s.width * 0.9)), height: Math.max(30, Math.round(s.height * 0.9)) })} style={quickBtnStyle}>缩小 10%</button>
            <button onClick={() => sendBlockUpdate({ left: Math.round((1920 - s.width) / 2) })} style={quickBtnStyle}>水平居中</button>
            <button onClick={() => sendBlockUpdate({ top: Math.round((1080 - s.height) / 2) })} style={quickBtnStyle}>垂直居中</button>
          </div>
        </div>
        <div style={{ fontSize: 11, color: '#9CA3AF', lineHeight: 1.6 }}>
          绿色框 = 可拖拽/缩放模块。拖中心区域移动，拖八个控制点缩放。仅限绝对定位的卡片/区块/装饰/热点。
        </div>
      </>
    )
  }

  // ==================== 主渲染 ====================

  return (
    <div onClick={handleClose}
      style={{ position: 'fixed', inset: 0, zIndex: 99991, background: 'rgba(15,23,42,0.72)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 24 }}>
      <div onClick={e => e.stopPropagation()}
        style={{ width: '96vw', height: '92vh', background: '#fff', borderRadius: 14, display: 'flex', flexDirection: 'column', overflow: 'hidden', boxShadow: '0 20px 60px rgba(0,0,0,0.4)' }}>
        {/* 头部 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 14, padding: '14px 20px', borderBottom: `1px solid ${C.border}`, flexWrap: 'wrap' }}>
          <div style={{ fontSize: 15, fontWeight: 700, color: C.textPrimary }}>
            ✏️ 第 {pageNum} 页 · 就地编辑
            <span style={{ marginLeft: 10, fontSize: 12, fontWeight: 400, color: C.textMuted }}>文字/图片/视频/模块 全可编辑，绿框模块可鼠标拖拽</span>
          </div>
          <div style={{ flex: 1 }} />
          <button onClick={handleSave} disabled={saving || loading}
            style={{ padding: '7px 20px', borderRadius: 8, border: 'none', background: (saving || loading) ? '#E5E7EB' : (dirty ? '#7C3AED' : '#10B981'), color: (saving || loading) ? '#9CA3AF' : '#fff', fontSize: 14, fontWeight: 600, cursor: (saving || loading) ? 'default' : 'pointer', whiteSpace: 'nowrap' }}>
            {saving ? '⏳ 保存中...' : (dirty ? '💾 保存修改' : '✓ 完成')}
          </button>
          <button onClick={handleClose} disabled={saving}
            style={{ padding: '7px 16px', borderRadius: 8, border: `1px solid ${C.border}`, background: '#fff', color: C.textSecondary, fontSize: 14, fontWeight: 600, cursor: saving ? 'default' : 'pointer' }}>
            ✕ 关闭
          </button>
        </div>
        {/* 主体 */}
        <div style={{ flex: 1, display: 'flex', minHeight: 0 }}>
          <div style={{ flex: 1, position: 'relative', overflow: 'hidden', background: '#F3F4F6', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            {loading ? (<div style={{ color: C.textSecondary, fontSize: 15 }}>⏳ 正在加载页面...</div>)
              : error && !srcDocRef.current ? (<div style={{ color: C.danger, fontSize: 14, padding: 24, textAlign: 'center' }}>❌ {error}</div>)
              : (
                <div style={{ position: 'absolute', top: '50%', left: '50%', width: CW_WIDTH, height: CW_HEIGHT, transform: 'translate(-50%, -50%) scale(var(--edit-scale))', transformOrigin: 'center center' }} ref={scaleWrapRef}>
                  <iframe ref={iframeRef} title="就地编辑" srcDoc={srcDocRef.current} sandbox="allow-scripts allow-same-origin"
                    style={{ width: CW_WIDTH, height: CW_HEIGHT, border: 'none', display: 'block', background: '#fff' }} />
                </div>
              )}
          </div>
          <div style={{ width: 300, borderLeft: `1px solid ${C.border}`, padding: '14px 16px', display: 'flex', flexDirection: 'column', gap: 12, overflowY: 'auto', background: '#FAFAFA' }}>
            {!selected ? (
              <div style={{ color: C.textMuted, fontSize: 13, lineHeight: 1.7, marginTop: 8 }}>
                👈 在左侧页面上<b>点选元素</b>。<br /><br />
                <span style={{ color: '#7C3AED' }}>📝 文字</span>：紫色虚线框，改内容/字号/颜色/加粗/字体。<br /><br />
                <span style={{ color: '#0EA5E9' }}>🖼️ 图片</span>：蓝色虚线框，上传替换/AI修改/AI新生成/调尺寸/原位转视频。<br /><br />
                <span style={{ color: '#D97706' }}>🎬 视频</span>：橙色虚线框，上传替换或AI重新生成；自定义封面也可直接点选。<br /><br />
                <span style={{ color: '#10B981' }}>📦 模块</span>：绿色虚线框，可鼠标拖拽移动/缩放或输入精确数值。适用于热点圆点、装饰卡片等绝对定位元素。<br /><br />
                <span style={{ color: C.textSecondary, fontSize: 12 }}>提示：四种类型可随时互相切换——模块内的文字可直接点选；若整个模块就是一段文字，先点选为模块（可拖拽），再点一次即切换为文字编辑；点空白处或按 ESC 取消选中。</span>
              </div>
            ) : selected.mode === 'text' ? renderTextTools() : selected.mode === 'image' ? renderImageTools() : selected.mode === 'video' ? renderVideoTools() : renderBlockTools()}
            {error && srcDocRef.current && (<div style={{ marginTop: 'auto', padding: '8px 10px', borderRadius: 8, background: '#FEE2E2', color: '#DC2626', fontSize: 12 }}>❌ {error}</div>)}
          </div>
        </div>
        <div style={{ padding: '10px 20px', borderTop: `1px solid ${C.border}`, fontSize: 12, color: C.textMuted, textAlign: 'center' }}>
          就地编辑：文字（保留原格式）+ 图片（上传/AI生成/尺寸/原位转视频）+ 视频（上传替换/AI重新生成）+ 模块（拖拽/缩放/位置/热点微调）。保存前系统自动存旧版，可在「📜 历史版本」回退。
        </div>
      </div>
    </div>
  )
}

// ==================== 共享样式常量 ====================

const labelStyle: React.CSSProperties = { fontSize: 12, color: '#6B7280', display: 'block', marginBottom: 4, fontWeight: 600 }
const stepBtnStyle: React.CSSProperties = { width: 30, height: 30, borderRadius: 6, border: '1px solid #E5E7EB', background: '#fff', color: '#1F2937', fontSize: 16, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }
const formatBtnStyle: React.CSSProperties = { padding: '7px 12px', borderRadius: 6, border: '1px solid #E5E7EB', background: '#fff', color: '#1F2937', fontSize: 12, fontWeight: 600, display: 'flex', alignItems: 'center', gap: 6, width: '100%', justifyContent: 'center' }
const quickBtnStyle: React.CSSProperties = { padding: '5px 10px', borderRadius: 6, border: '1px solid #E5E7EB', background: '#fff', fontSize: 11, cursor: 'pointer' }
/** v5.6: block 模式位置输入框统一样式 */
const blockNumInputStyle: React.CSSProperties = { width: 80, padding: '6px', borderRadius: 6, border: '1px solid #E5E7EB', fontSize: 13, textAlign: 'center', outline: 'none', display: 'block' }
/** v5.6: ±1px 微调步进小按钮 */
const microStepBtnStyle: React.CSSProperties = { width: 20, height: 14, borderRadius: 3, border: '1px solid #D1D5DB', background: '#F9FAFB', color: '#6B7280', fontSize: 8, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: 0, lineHeight: 1 }
/** v5.6: ±5px / ±10px 批量步进按钮 */
const stepBatchBtnStyle: React.CSSProperties = { padding: '3px 8px', borderRadius: 4, border: '1px solid #E5E7EB', background: '#F9FAFB', fontSize: 10, cursor: 'pointer', color: '#374151', fontWeight: 500 }

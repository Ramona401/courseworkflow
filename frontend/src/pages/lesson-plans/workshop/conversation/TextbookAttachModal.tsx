/**
 * TextbookAttachModal.tsx — 课本中途挂载弹窗（迭代3.5 A2-2 新增，A2-4 体验修正）
 *
 * A2-4 修正（一线反馈）：
 *   1. 删除「查看全部课本」逃生口——只显示与本教案【同学科+同年级】的课本页，
 *      杜绝挂上其他学科/年级的课本污染备课上下文；
 *   2. 弹窗内置上传入口——库里没有匹配课本时直接引导上传（学科年级锁定为本教案的，
 *      不可修改），支持一次选多张图，上传成功自动触发OCR并自动选中；
 *      列表非空时上传入口也常驻头部，方便补页。
 *
 * 既有交互（保持不变）：
 *   - 勾选未识别的课本页时自动触发 OCR，识别中拦截确认按钮（保证挂载的都有OCR文本）
 *   - 确认时调 PUT /plans/{id}/textbooks 整体替换关联列表（传空数组=解除全部）
 *   - currentPageIds 是本会话内已挂载记录（页面状态），跨会话恢复时
 *     教案详情接口不返回 textbook_page_ids，故弹窗按空选中初始化——确认即整体替换
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { uploadTextbook, getTextbooks, triggerTextbookOCR, type TextbookListItem } from '@/api/textbooks'
import { updatePlanTextbooks } from '@/api/lesson-plan-textbooks'
import { C } from '../components/workshopConstants'

/** 单张课本图片上限（与后端 textbook_service 校验口径一致：JPG/PNG/WEBP 最大10MB） */
const MAX_FILE_SIZE = 10 * 1024 * 1024
const ALLOWED_MIME = ['image/jpeg', 'image/png', 'image/webp']
/** 一份教案最多关联的课本页数（与后端 PUT /plans/{id}/textbooks 的20张拦截一致） */
const MAX_ATTACH = 20

interface TextbookAttachModalProps {
  planId: string
  subject: string
  grade: string
  /** 本会话内已挂载的课本页ID（用于预选中） */
  currentPageIds: string[]
  /** 挂载成功回调（回传最终关联的ID列表） */
  onSuccess: (pageIds: string[]) => void
  onCancel: () => void
}

export default function TextbookAttachModal({
  planId, subject, grade, currentPageIds, onSuccess, onCancel,
}: TextbookAttachModalProps) {
  const [pages, setPages] = useState<TextbookListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set(currentPageIds))
  /** 正在OCR识别中的课本页ID（识别中拦截确认） */
  const [ocrPending, setOcrPending] = useState<Set<string>>(new Set())
  const [saving, setSaving] = useState(false)

  // ===== A2-4：弹窗内上传状态 =====
  /** 上传面板展开态（列表为空时自动展开引导上传） */
  const [showUpload, setShowUpload] = useState(false)
  /** 教材名称（必填；多批上传间保留，省得反复输入） */
  const [upName, setUpName] = useState('')
  /** 章节（可选） */
  const [upChapter, setUpChapter] = useState('')
  /** 起始页码（可选，>0 时多张图按序递增；0=不标页码） */
  const [upStartPage, setUpStartPage] = useState(0)
  /** 已选择待上传的图片文件 */
  const [upFiles, setUpFiles] = useState<File[]>([])
  const [uploading, setUploading] = useState(false)
  /** 上传进度文案（第N/共M张） */
  const [upProgress, setUpProgress] = useState<string | null>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // 拉取课本列表 —— 严格限定本教案学科+年级（A2-4：不再提供放宽筛选）
  const fetchPages = useCallback(async () => {
    setLoading(true)
    setError(null)
    try {
      const resp = await getTextbooks({ limit: 100, subject, grade_range: grade })
      const list = resp.pages || []
      setPages(list)
      // 库里没有匹配课本 → 自动展开上传面板引导老师上传
      if (list.length === 0) setShowUpload(true)
    } catch (err) {
      console.error('获取课本列表失败:', err)
      setError('加载课本列表失败')
      setPages([])
    } finally {
      setLoading(false)
    }
  }, [subject, grade])

  useEffect(() => { fetchPages() }, [fetchPages])

  /** 触发单页OCR识别（勾选未识别项/上传成功后自动调用） */
  const startOCR = async (item: TextbookListItem) => {
    setOcrPending(prev => new Set(prev).add(item.id))
    try {
      await triggerTextbookOCR(item.id)
      // 识别成功：本地标记 has_ocr=true
      setPages(prev => prev.map(p => p.id === item.id ? { ...p, has_ocr: true } : p))
    } catch (err) {
      console.error('课本OCR识别失败:', err)
      // 识别失败：自动取消勾选，避免挂载无OCR文本的课本页
      setSelectedIds(prev => { const next = new Set(prev); next.delete(item.id); return next })
      alert(`「${item.textbook_name} 第${item.page_number}页」AI识别失败，已取消选择，请稍后重试`)
    } finally {
      setOcrPending(prev => { const next = new Set(prev); next.delete(item.id); return next })
    }
  }

  /** 切换选中（勾选未识别项自动触发OCR） */
  const handleToggle = (item: TextbookListItem) => {
    if (selectedIds.has(item.id)) {
      setSelectedIds(prev => { const next = new Set(prev); next.delete(item.id); return next })
      return
    }
    if (selectedIds.size >= MAX_ATTACH) {
      alert(`一份教案最多关联${MAX_ATTACH}张课本页`)
      return
    }
    setSelectedIds(prev => new Set(prev).add(item.id))
    if (!item.has_ocr && !ocrPending.has(item.id)) {
      startOCR(item)
    }
  }

  /** A2-4：选择待上传图片（前端预校验类型与大小，无效文件剔除并提示） */
  const handlePickFiles = (files: FileList | null) => {
    if (!files || files.length === 0) return
    const valid: File[] = []
    const rejected: string[] = []
    Array.from(files).forEach(f => {
      if (!ALLOWED_MIME.includes(f.type)) { rejected.push(`${f.name}（仅支持JPG/PNG/WEBP）`); return }
      if (f.size > MAX_FILE_SIZE) { rejected.push(`${f.name}（超过10MB）`); return }
      valid.push(f)
    })
    if (rejected.length > 0) alert(`以下文件已跳过：\n${rejected.join('\n')}`)
    setUpFiles(valid)
  }

  /**
   * A2-4：执行上传 —— 学科年级锁定取自本教案；多张图逐张上传，
   * 成功后构造本地列表项插入列表头部 + 自动选中（不超上限）+ 自动触发OCR
   */
  const handleUpload = async () => {
    if (uploading) return
    if (upFiles.length === 0) { alert('请先选择课本图片'); return }
    if (!upName.trim()) { alert('请填写教材名称（如：人教版三年级上册语文），方便以后查找'); return }
    setUploading(true)
    const okItems: TextbookListItem[] = []
    const failNames: string[] = []
    for (let i = 0; i < upFiles.length; i++) {
      const f = upFiles[i]
      setUpProgress(`正在上传 ${i + 1}/${upFiles.length}：${f.name}`)
      try {
        const fd = new FormData()
        fd.append('file', f)
        fd.append('subject', subject)            // 锁定为本教案学科，不可修改
        fd.append('grade_range', grade)          // 锁定为本教案年级，不可修改
        fd.append('textbook_name', upName.trim())
        fd.append('chapter', upChapter.trim())
        fd.append('page_number', String(upStartPage > 0 ? upStartPage + i : 0))
        fd.append('description', '')
        fd.append('scope', 'public')             // 与课本管理页默认口径一致：所有人可见
        const resp = await uploadTextbook(fd)
        // 构造本地列表项（接口只返回少量字段，其余按已知上下文补齐）
        okItems.push({
          id: resp.id,
          subject, grade_range: grade,
          textbook_name: upName.trim(), chapter: upChapter.trim(),
          page_number: upStartPage > 0 ? upStartPage + i : 0,
          file_name: resp.file_name, file_size: resp.file_size, mime_type: f.type,
          has_ocr: false, description: '', scope: 'public', scope_name: '所有人',
          uploaded_by: '', uploader_name: '', usage_count: 0,
          image_url: resp.image_url, created_at: new Date().toISOString(),
        })
      } catch (err) {
        console.error('课本上传失败:', f.name, err)
        failNames.push(f.name)
      }
    }
    setUpProgress(null)
    setUploading(false)
    if (okItems.length > 0) {
      // 新上传的页插到列表头部
      setPages(prev => [...okItems, ...prev])
      // 自动选中（不超上限）+ 自动触发OCR
      setSelectedIds(prev => {
        const next = new Set(prev)
        okItems.forEach(item => { if (next.size < MAX_ATTACH) next.add(item.id) })
        return next
      })
      okItems.forEach(item => startOCR(item))
      // 收起面板、清空文件；教材名/章节保留，起始页码顺延，方便继续补传
      setShowUpload(false)
      setUpFiles([])
      if (fileInputRef.current) fileInputRef.current.value = ''
      setUpStartPage(prev => (prev > 0 ? prev + okItems.length : 0))
    }
    if (failNames.length > 0) {
      alert(`以下图片上传失败，请稍后重试：\n${failNames.join('\n')}`)
    }
  }

  /** 确认挂载：整体替换教案的课本关联列表 */
  const handleConfirm = async () => {
    if (saving) return
    setSaving(true)
    try {
      const ids = Array.from(selectedIds)
      await updatePlanTextbooks(planId, ids)
      onSuccess(ids)
    } catch (err) {
      console.error('课本关联失败:', err)
      const errMsg = err instanceof Error ? err.message : ''
      alert(errMsg && errMsg !== '请求失败' ? `课本关联失败：${errMsg}` : '课本关联失败，请稍后重试')
      setSaving(false)
    }
  }

  /** 选中项中是否还有OCR识别中的（识别中拦截确认） */
  const hasOcrInProgress = Array.from(selectedIds).some(id => ocrPending.has(id))
  /** 确认按钮可用性：0选中且原本也无关联 → 没有意义，禁用；上传中也禁用 */
  const confirmDisabled = saving || uploading || hasOcrInProgress || (selectedIds.size === 0 && currentPageIds.length === 0)

  const inputSt: React.CSSProperties = {
    padding: '8px 12px', borderRadius: '8px', border: `1px solid ${C.border}`,
    fontSize: '13px', color: C.text, outline: 'none', boxSizing: 'border-box', fontFamily: 'inherit', width: '100%',
  }

  return (
    <div style={{ position: 'fixed', top: 0, left: 0, right: 0, bottom: 0, background: 'rgba(0,0,0,0.45)', zIndex: 1000, display: 'flex', alignItems: 'center', justifyContent: 'center' }}
      onClick={e => { if (e.target === e.currentTarget && !saving && !uploading) onCancel() }}>
      <div style={{ background: '#fff', borderRadius: '16px', width: '620px', maxHeight: '85vh', display: 'flex', flexDirection: 'column', boxShadow: '0 20px 60px rgba(0,0,0,0.15)', overflow: 'hidden' }}>

        {/* 头部 */}
        <div style={{ padding: '24px 28px 14px', borderBottom: `1px solid ${C.border}` }}>
          <div style={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between' }}>
            <div>
              <h3 style={{ margin: 0, fontSize: '17px', fontWeight: 700, color: C.text }}>📷 关联课本图片</h3>
              <p style={{ margin: '6px 0 0', fontSize: '13px', color: C.textMuted, lineHeight: 1.5 }}>
                选中后AI会贴着课文原文来设计，下一轮对话起生效。未识别的页面会自动进行AI识别。
              </p>
            </div>
            <button onClick={() => !saving && !uploading && onCancel()} style={{ background: 'none', border: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted, padding: '4px' }}>✕</button>
          </div>
          {/* A2-4：固定学科年级展示（与本教案一致，不可放宽）+ 上传入口 */}
          <div style={{ marginTop: '10px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', fontSize: '12px', color: C.textSec }}>
            <span>📚 仅显示：<b style={{ color: C.text }}>{subject} · {grade}</b>（与本教案一致）</span>
            <button onClick={() => setShowUpload(v => !v)} disabled={uploading}
              style={{ background: 'none', border: `1px solid ${showUpload ? C.primary : C.border}`, borderRadius: '8px', cursor: uploading ? 'not-allowed' : 'pointer', fontSize: '12px', color: C.primary, padding: '4px 10px', fontWeight: 600 }}>
              {showUpload ? '收起上传' : '📤 上传课本页'}
            </button>
          </div>
        </div>

        {/* 内容区 */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '14px 28px' }}>

          {/* A2-4：弹窗内上传面板（学科年级锁定，只填教材名/章节/页码+选图） */}
          {showUpload && (
            <div style={{ border: `1.5px dashed ${C.primary}`, borderRadius: '12px', padding: '14px 16px', marginBottom: '14px', background: 'rgba(79,123,232,0.03)' }}>
              <div style={{ fontSize: '13px', fontWeight: 700, color: C.text, marginBottom: '10px' }}>
                📤 上传课本页（{subject} · {grade}，自动归入本学科年级）
              </div>
              <div style={{ display: 'grid', gridTemplateColumns: '2fr 1.4fr 76px', gap: '10px', marginBottom: '10px' }}>
                <div>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '4px' }}>教材名称 *</label>
                  <input value={upName} onChange={e => setUpName(e.target.value)} disabled={uploading}
                    placeholder="如：人教版三年级上册语文" style={inputSt} />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '4px' }}>章节（可选）</label>
                  <input value={upChapter} onChange={e => setUpChapter(e.target.value)} disabled={uploading}
                    placeholder="如：第1课 观潮" style={inputSt} />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.textSec, marginBottom: '4px' }}>起始页码</label>
                  <input type="number" min={0} value={upStartPage} disabled={uploading}
                    onChange={e => setUpStartPage(parseInt(e.target.value) || 0)} style={inputSt} />
                </div>
              </div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <div style={{ flex: 1, minWidth: 0 }}>
                  <input ref={fileInputRef} type="file" multiple accept="image/jpeg,image/png,image/webp"
                    disabled={uploading}
                    onChange={e => handlePickFiles(e.target.files)}
                    style={{ fontSize: '12px', color: C.textSec, maxWidth: '100%' }} />
                  <div style={{ fontSize: '11px', color: C.textMuted, marginTop: '4px' }}>
                    可一次选多张 · JPG/PNG/WEBP · 单张≤10MB{upFiles.length > 0 ? ` · 已选 ${upFiles.length} 张` : ''}
                    {upStartPage > 0 && upFiles.length > 1 ? `（页码将按 ${upStartPage}~${upStartPage + upFiles.length - 1} 顺序标注）` : ''}
                  </div>
                </div>
                <button onClick={handleUpload} disabled={uploading || upFiles.length === 0 || !upName.trim()}
                  style={{ padding: '8px 18px', borderRadius: '8px', border: 'none', fontSize: '13px', fontWeight: 600, flexShrink: 0,
                    cursor: (uploading || upFiles.length === 0 || !upName.trim()) ? 'not-allowed' : 'pointer',
                    background: (uploading || upFiles.length === 0 || !upName.trim()) ? '#E5E7EB' : '#10B981',
                    color: (uploading || upFiles.length === 0 || !upName.trim()) ? C.textMuted : '#fff' }}>
                  {uploading ? '上传中…' : '确认上传'}
                </button>
              </div>
              {upProgress && (
                <div style={{ marginTop: '8px', fontSize: '12px', color: '#D97706' }}>🔄 {upProgress}</div>
              )}
            </div>
          )}

          {loading && (
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', padding: '48px 0', gap: '12px' }}>
              <div style={{ width: '32px', height: '32px', border: `3px solid ${C.primary}`, borderTopColor: 'transparent', borderRadius: '50%', animation: 'spin 0.8s linear infinite' }} />
              <span style={{ fontSize: '14px', color: C.textMuted }}>正在加载课本列表...</span>
              <style>{`@keyframes spin { to { transform: rotate(360deg); } }`}</style>
            </div>
          )}

          {!loading && error && (
            <div style={{ textAlign: 'center', padding: '40px 0', color: C.textMuted, fontSize: '14px' }}>
              <div style={{ fontSize: '28px', marginBottom: '8px' }}>⚠️</div>
              {error}
              <div style={{ marginTop: '12px' }}>
                <button onClick={fetchPages} style={{ padding: '6px 16px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '13px', color: C.primary, cursor: 'pointer' }}>重试</button>
              </div>
            </div>
          )}

          {/* A2-4：空态 → 引导上传（不再提供"查看全部课本"逃生口） */}
          {!loading && !error && pages.length === 0 && (
            <div style={{ textAlign: 'center', padding: '32px 0', color: C.textMuted, fontSize: '14px' }}>
              <div style={{ fontSize: '28px', marginBottom: '8px' }}>📭</div>
              课本库里还没有 <b style={{ color: C.text }}>{subject} · {grade}</b> 的课本页
              <p style={{ fontSize: '12px', marginTop: '8px', color: C.textMuted, lineHeight: 1.6 }}>
                用手机拍下课本页面，在上方上传区直接上传即可——<br />上传后会自动进行AI识别并选中，确认关联后AI就能贴着课文设计了。
              </p>
              {!showUpload && (
                <button onClick={() => setShowUpload(true)}
                  style={{ marginTop: '10px', padding: '8px 20px', borderRadius: '10px', border: 'none', background: `linear-gradient(135deg, ${C.primary}, #818CF8)`, color: '#fff', fontSize: '13px', fontWeight: 600, cursor: 'pointer' }}>
                  📤 上传课本图片
                </button>
              )}
            </div>
          )}

          {!loading && !error && pages.length > 0 && (
            <>
              <div style={{ fontSize: '13px', color: C.textSec, marginBottom: '10px' }}>
                共 {pages.length} 张课本页，已选 {selectedIds.size} 张（上限{MAX_ATTACH}张）
              </div>
              <div style={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
                {pages.map(item => {
                  const isSelected = selectedIds.has(item.id)
                  const isOcrIng = ocrPending.has(item.id)
                  return (
                    <div key={item.id} onClick={() => handleToggle(item)}
                      style={{ display: 'flex', alignItems: 'center', gap: '12px', padding: '10px 12px', borderRadius: '10px', border: `1.5px solid ${isSelected ? C.primary : C.border}`, background: isSelected ? 'rgba(79,123,232,0.04)' : '#fff', cursor: 'pointer', transition: 'all 150ms ease' }}>
                      {/* 复选框 */}
                      <div style={{ width: '20px', height: '20px', borderRadius: '5px', flexShrink: 0, border: `2px solid ${isSelected ? C.primary : '#D1D5DB'}`, background: isSelected ? C.primary : '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                        {isSelected && <span style={{ color: '#fff', fontSize: '12px', fontWeight: 700 }}>✓</span>}
                      </div>
                      {/* 缩略图 */}
                      {item.image_url && (
                        <img src={item.image_url} alt={item.file_name} loading="lazy"
                          style={{ width: '52px', height: '52px', objectFit: 'cover', borderRadius: '6px', flexShrink: 0, border: `1px solid ${C.border}` }} />
                      )}
                      {/* 信息 */}
                      <div style={{ flex: 1, minWidth: 0 }}>
                        <div style={{ fontSize: '14px', fontWeight: 500, color: C.text, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {item.textbook_name || item.file_name}{item.page_number > 0 ? ` · 第${item.page_number}页` : ''}
                        </div>
                        <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '2px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                          {item.subject} · {item.grade_range}{item.chapter ? ` · ${item.chapter}` : ''}
                        </div>
                      </div>
                      {/* OCR状态徽标 */}
                      <span style={{ fontSize: '11px', padding: '2px 8px', borderRadius: '8px', flexShrink: 0, whiteSpace: 'nowrap',
                        background: isOcrIng ? 'rgba(245,158,11,0.1)' : item.has_ocr ? 'rgba(16,185,129,0.1)' : 'rgba(0,0,0,0.05)',
                        color: isOcrIng ? '#D97706' : item.has_ocr ? '#059669' : C.textMuted }}>
                        {isOcrIng ? '🔄 AI识别中…' : item.has_ocr ? '✅ 已识别' : '选中后自动识别'}
                      </span>
                    </div>
                  )
                })}
              </div>
            </>
          )}
        </div>

        {/* 底部操作栏 */}
        <div style={{ padding: '16px 28px', borderTop: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'flex-end', background: '#FAFBFC', gap: '12px' }}>
          <button onClick={() => !saving && !uploading && onCancel()}
            style={{ padding: '9px 20px', borderRadius: '10px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '13px', color: C.textSec, cursor: 'pointer' }}>取消</button>
          <button onClick={handleConfirm} disabled={confirmDisabled}
            style={{ padding: '9px 24px', borderRadius: '10px', border: 'none', background: confirmDisabled ? '#E5E7EB' : `linear-gradient(135deg, ${C.primary}, #818CF8)`, color: confirmDisabled ? C.textMuted : '#fff', fontSize: '14px', fontWeight: 600, cursor: confirmDisabled ? 'not-allowed' : 'pointer', boxShadow: confirmDisabled ? 'none' : '0 3px 12px rgba(79,123,232,0.3)', transition: 'all 200ms ease' }}>
            {saving ? '正在关联…'
              : uploading ? '上传中，请稍候…'
              : hasOcrInProgress ? 'AI识别中，请稍候…'
              : selectedIds.size > 0 ? `📷 关联 ${selectedIds.size} 张课本页`
              : currentPageIds.length > 0 ? '解除全部课本关联'
              : '请至少选择1张课本页'}
          </button>
        </div>
      </div>
    </div>
  )
}

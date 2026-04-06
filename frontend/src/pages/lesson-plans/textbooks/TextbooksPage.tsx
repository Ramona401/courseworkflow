/**
 * TextbooksPage — 课本管理页面
 *
 * 迭代7新增：上传/查看/OCR识别/删除课本图片
 * 功能：
 *   1. 上传区（拖拽或点击选择图片+填写元数据）
 *   2. 课本列表（按教材分组，缩略图+元信息）
 *   3. 图片预览弹窗（大图+OCR文字）
 *   4. 筛选：学科+教材名搜索
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { useAuth } from '@/store/auth'
import {
  uploadTextbook, getTextbooks, deleteTextbook, triggerTextbookOCR,
  type TextbookListItem,
} from '@/api/textbooks'

/* ==================== 颜色常量 ==================== */
const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981', danger: '#EF4444', accent: '#F59E0B',
  text: '#1F2937', textSec: '#6B7280', textMuted: '#9CA3AF',
  card: '#FFFFFF', border: '#F3F4F6', bg: '#FAFBFC',
}

const SUBJECTS = ['全部','AI','人工智能','语文','数学','英语','物理','化学','生物','历史','地理','政治','信息技术']
const GRADES = ['七年级','八年级','九年级','高一','高二','高三','小学低段','小学中段','小学高段']

/* ==================== 文件大小格式化 ==================== */
function formatSize(bytes: number): string {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / (1024 * 1024)).toFixed(1) + ' MB'
}

/* ==================== 主组件 ==================== */
export default function TextbooksPage() {
  const { user } = useAuth()

  // 列表数据
  const [pages, setPages] = useState<TextbookListItem[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(true)

  // 筛选
  const [subjectFilter, setSubjectFilter] = useState('全部')
  const [searchText, setSearchText] = useState('')

  // 上传表单
  const [showUpload, setShowUpload] = useState(false)
  const [uploadSubject, setUploadSubject] = useState('AI')
  const [uploadGrade, setUploadGrade] = useState('七年级')
  const [uploadTextbookName, setUploadTextbookName] = useState('')
  const [uploadChapter, setUploadChapter] = useState('')
  const [uploadPageNum, setUploadPageNum] = useState(0)
  const [uploadDesc, setUploadDesc] = useState('')
  const [uploadFile, setUploadFile] = useState<File | null>(null)
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // 预览弹窗
  const [previewItem, setPreviewItem] = useState<TextbookListItem | null>(null)
  const [ocrText, setOcrText] = useState('')
  const [ocrLoading, setOcrLoading] = useState(false)

  // 操作状态
  const [loadingId, setLoadingId] = useState<string | null>(null)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)

  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type }); setTimeout(() => setToast(null), 3000)
  }

  // ==================== 加载列表 ====================
  const loadPages = useCallback(async () => {
    setLoading(true)
    try {
      const params: Record<string, string | number> = { limit: 100 }
      if (subjectFilter !== '全部') params.subject = subjectFilter
      if (searchText.trim()) params.textbook_name = searchText.trim()
      const resp = await getTextbooks(params)
      setPages(resp.pages || []); setTotal(resp.total || 0)
    } catch { showToast('加载失败', 'error') }
    finally { setLoading(false) }
  }, [subjectFilter, searchText])

  useEffect(() => { loadPages() }, [loadPages])

  // ==================== 上传 ====================
  const handleUpload = async () => {
    if (!uploadFile || !uploadTextbookName.trim()) {
      showToast('请选择图片并填写教材名称', 'error'); return
    }
    setUploading(true)
    try {
      const fd = new FormData()
      fd.append('file', uploadFile)
      fd.append('subject', uploadSubject)
      fd.append('grade_range', uploadGrade)
      fd.append('textbook_name', uploadTextbookName.trim())
      fd.append('chapter', uploadChapter.trim())
      fd.append('page_number', String(uploadPageNum))
      fd.append('description', uploadDesc.trim())
      fd.append('scope', 'public') // 默认所有人可见
      await uploadTextbook(fd)
      showToast('上传成功 ✓')
      setUploadFile(null); setUploadChapter(''); setUploadPageNum(0); setUploadDesc('')
      if (fileInputRef.current) fileInputRef.current.value = ''
      await loadPages()
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : '上传失败', 'error')
    } finally { setUploading(false) }
  }

  // ==================== 删除 ====================
  const handleDelete = async (id: string, name: string) => {
    if (!confirm(`确定删除「${name}」？`)) return
    setLoadingId(id)
    try {
      await deleteTextbook(id); showToast('已删除'); await loadPages()
    } catch (e: unknown) { showToast(e instanceof Error ? e.message : '删除失败', 'error') }
    finally { setLoadingId(null) }
  }

  // ==================== OCR识别 ====================
  const handleOCR = async (id: string) => {
    setOcrLoading(true); setOcrText('')
    try {
      const resp = await triggerTextbookOCR(id)
      setOcrText(resp.ocr_text)
      showToast('识别完成 ✓')
      await loadPages() // 刷新has_ocr状态
    } catch (e: unknown) {
      showToast(e instanceof Error ? e.message : 'OCR识别失败', 'error')
    } finally { setOcrLoading(false) }
  }

  // ==================== 样式 ====================
  const inputSt: React.CSSProperties = {
    padding: '8px 12px', borderRadius: '6px', border: `1px solid ${C.border}`,
    fontSize: '13px', color: C.text, outline: 'none', boxSizing: 'border-box', fontFamily: 'inherit',
  }
  const selBtn = (active: boolean): React.CSSProperties => ({
    padding: '5px 12px', borderRadius: '20px', border: `1px solid ${active ? C.primary : C.border}`,
    background: active ? C.primaryLight : 'transparent', color: active ? C.primary : C.textSec,
    fontSize: '13px', fontWeight: active ? 600 : 400, cursor: 'pointer',
  })

  // ==================== 按教材名分组 ====================
  const grouped = pages.reduce<Record<string, TextbookListItem[]>>((acc, p) => {
    const key = p.textbook_name || '未分类'
    if (!acc[key]) acc[key] = []
    acc[key].push(p)
    return acc
  }, {})

  return (
    <div>
      {/* 顶部操作栏 */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '20px' }}>
        <p style={{ fontSize: '14px', color: C.textSec, margin: 0 }}>共 {total} 张课本图片</p>
        <button onClick={() => setShowUpload(!showUpload)} style={{
          display: 'flex', alignItems: 'center', gap: '6px', padding: '9px 18px', borderRadius: '8px',
          border: 'none', background: C.primary, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer',
        }}><span>📷</span><span>{showUpload ? '收起上传' : '上传课本图片'}</span></button>
      </div>

      {/* 上传区 */}
      {showUpload && (
        <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '24px', marginBottom: '20px' }}>
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '16px' }}>📷 上传课本图片</div>
          <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px', marginBottom: '16px' }}>
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>学科 *</label>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
                {SUBJECTS.filter(s => s !== '全部').map(s => (
                  <button key={s} onClick={() => setUploadSubject(s)} style={selBtn(uploadSubject === s)}>{s}</button>
                ))}
              </div>
            </div>
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>年级 *</label>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
                {GRADES.map(g => (
                  <button key={g} onClick={() => setUploadGrade(g)} style={selBtn(uploadGrade === g)}>{g}</button>
                ))}
              </div>
            </div>
          </div>
          <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 80px', gap: '12px', marginBottom: '16px' }}>
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>教材名称 *</label>
              <input value={uploadTextbookName} onChange={e => setUploadTextbookName(e.target.value)}
                placeholder="如：人教版七年级上册数学" style={{ ...inputSt, width: '100%' }} />
            </div>
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>章节</label>
              <input value={uploadChapter} onChange={e => setUploadChapter(e.target.value)}
                placeholder="如：第三章 一元一次方程" style={{ ...inputSt, width: '100%' }} />
            </div>
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>页码</label>
              <input type="number" value={uploadPageNum} onChange={e => setUploadPageNum(parseInt(e.target.value) || 0)}
                min={0} style={{ ...inputSt, width: '100%' }} />
            </div>
          </div>
          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>描述（可选）</label>
            <input value={uploadDesc} onChange={e => setUploadDesc(e.target.value)}
              placeholder="对这张图片的补充说明" style={{ ...inputSt, width: '100%' }} />
          </div>
          <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
            <div style={{ flex: 1 }}>
              <input ref={fileInputRef} type="file" accept="image/jpeg,image/png,image/webp"
                onChange={e => setUploadFile(e.target.files?.[0] || null)}
                style={{ fontSize: '13px', color: C.textSec }} />
              {uploadFile && <span style={{ fontSize: '12px', color: C.textMuted, marginLeft: '8px' }}>{formatSize(uploadFile.size)}</span>}
            </div>
            <button onClick={handleUpload} disabled={uploading || !uploadFile || !uploadTextbookName.trim()} style={{
              padding: '9px 24px', borderRadius: '8px', border: 'none', fontSize: '14px', fontWeight: 600, cursor: 'pointer',
              background: uploading || !uploadFile || !uploadTextbookName.trim() ? C.border : C.success,
              color: uploading || !uploadFile || !uploadTextbookName.trim() ? C.textMuted : '#fff',
            }}>{uploading ? '上传中...' : '确认上传'}</button>
          </div>
        </div>
      )}

      {/* 筛选栏 */}
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '16px 20px', marginBottom: '20px', display: 'flex', gap: '16px', alignItems: 'center', flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec }}>学科</span>
          <select value={subjectFilter} onChange={e => setSubjectFilter(e.target.value)} style={{
            ...inputSt, cursor: 'pointer', background: subjectFilter !== '全部' ? C.primaryLight : 'transparent',
            color: subjectFilter !== '全部' ? C.primary : C.textSec, borderColor: subjectFilter !== '全部' ? C.primary : C.border,
          }}>{SUBJECTS.map(s => <option key={s} value={s}>{s}</option>)}</select>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flex: 1 }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec }}>搜索</span>
          <input value={searchText} onChange={e => setSearchText(e.target.value)}
            placeholder="搜索教材名称..." style={{ ...inputSt, flex: 1 }} />
        </div>
      </div>

      {/* 课本列表（按教材分组） */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: '60px', color: C.textMuted }}>加载中...</div>
      ) : Object.keys(grouped).length === 0 ? (
        <div style={{ textAlign: 'center', padding: '80px 40px', background: C.card, borderRadius: '12px', border: `1px solid ${C.border}` }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>📷</div>
          <div style={{ fontSize: '16px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>还没有课本图片</div>
          <div style={{ fontSize: '14px', color: C.textMuted }}>上传课本页面的真实图片，AI可以精准识别课本内容辅助备课</div>
        </div>
      ) : (
        Object.entries(grouped).map(([textbookName, items]) => (
          <div key={textbookName} style={{ marginBottom: '20px' }}>
            <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '10px', display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span>📚</span> {textbookName}
              <span style={{ fontSize: '12px', fontWeight: 400, color: C.textMuted }}>（{items.length}页）</span>
            </div>
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(200px, 1fr))', gap: '12px' }}>
              {items.map(item => (
                <div key={item.id} style={{
                  background: C.card, borderRadius: '10px', border: `1px solid ${C.border}`,
                  overflow: 'hidden', cursor: 'pointer', transition: 'all 200ms ease',
                }} onClick={() => { setPreviewItem(item); setOcrText('') }}>
                  {/* 缩略图 */}
                  <div style={{ height: '140px', background: C.bg, display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden' }}>
                    <img src={item.image_url} alt={item.file_name} style={{ maxWidth: '100%', maxHeight: '100%', objectFit: 'contain' }}
                      onError={e => { (e.target as HTMLImageElement).style.display = 'none' }} />
                  </div>
                  {/* 信息 */}
                  <div style={{ padding: '10px 12px' }}>
                    <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '4px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {item.chapter || `第${item.page_number}页`}
                    </div>
                    <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                      <span style={{ fontSize: '11px', color: C.textMuted }}>{formatSize(item.file_size)}</span>
                      <div style={{ display: 'flex', gap: '4px' }}>
                        {item.has_ocr && <span style={{ fontSize: '10px', padding: '1px 6px', borderRadius: '4px', background: 'rgba(16,185,129,0.08)', color: C.success }}>已识别</span>}
                        <span style={{ fontSize: '10px', padding: '1px 6px', borderRadius: '4px', background: C.primaryLight, color: C.primary }}>{item.scope_name}</span>
                      </div>
                    </div>
                    {/* 删除按钮（仅上传者可见） */}
                    {user?.id === item.uploaded_by && (
                      <button onClick={e => { e.stopPropagation(); handleDelete(item.id, item.file_name) }}
                        disabled={loadingId === item.id}
                        style={{ marginTop: '6px', width: '100%', padding: '4px', borderRadius: '4px', border: `1px solid rgba(239,68,68,0.2)`, background: 'transparent', color: C.danger, fontSize: '11px', cursor: 'pointer' }}>
                        {loadingId === item.id ? '删除中...' : '删除'}
                      </button>
                    )}
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))
      )}

      {/* 预览弹窗 */}
      {previewItem && (
        <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 10000 }}
          onClick={e => { if (e.target === e.currentTarget) setPreviewItem(null) }}>
          <div style={{ background: C.card, borderRadius: '16px', width: '800px', maxHeight: '90vh', overflow: 'auto', boxShadow: '0 20px 60px rgba(0,0,0,0.3)' }}>
            {/* 标题栏 */}
            <div style={{ padding: '16px 24px', borderBottom: `1px solid ${C.border}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <div>
                <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>{previewItem.textbook_name}</div>
                <div style={{ fontSize: '13px', color: C.textMuted, marginTop: '4px' }}>
                  {previewItem.chapter || `第${previewItem.page_number}页`} · {previewItem.subject} · {previewItem.grade_range} · by {previewItem.uploader_name}
                </div>
              </div>
              <button onClick={() => setPreviewItem(null)} style={{ border: 'none', background: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>✕</button>
            </div>
            {/* 图片+OCR */}
            <div style={{ display: 'flex', gap: '0' }}>
              {/* 左：大图 */}
              <div style={{ flex: 1, padding: '16px', display: 'flex', alignItems: 'center', justifyContent: 'center', background: C.bg, minHeight: '400px' }}>
                <img src={previewItem.image_url} alt={previewItem.file_name} style={{ maxWidth: '100%', maxHeight: '70vh', objectFit: 'contain', borderRadius: '4px' }} />
              </div>
              {/* 右：OCR区 */}
              <div style={{ width: '300px', borderLeft: `1px solid ${C.border}`, display: 'flex', flexDirection: 'column' }}>
                <div style={{ padding: '12px 16px', borderBottom: `1px solid ${C.border}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                  <span style={{ fontSize: '14px', fontWeight: 600, color: C.text }}>📝 文字识别</span>
                  <button onClick={() => handleOCR(previewItem.id)} disabled={ocrLoading} style={{
                    padding: '5px 12px', borderRadius: '6px', border: 'none', fontSize: '12px', fontWeight: 600, cursor: ocrLoading ? 'not-allowed' : 'pointer',
                    background: ocrLoading ? C.border : C.primary, color: ocrLoading ? C.textMuted : '#fff',
                  }}>{ocrLoading ? 'AI识别中...' : previewItem.has_ocr ? '重新识别' : 'AI识别'}</button>
                </div>
                <div style={{ flex: 1, padding: '12px 16px', overflowY: 'auto', fontSize: '13px', color: C.text, lineHeight: 1.7, whiteSpace: 'pre-wrap' }}>
                  {ocrLoading ? (
                    <div style={{ textAlign: 'center', padding: '40px 0', color: C.textMuted }}>
                      <div style={{ marginBottom: '8px' }}>🤖</div>AI正在识别图片文字...
                    </div>
                  ) : ocrText ? (
                    ocrText
                  ) : previewItem.has_ocr ? (
                    <div style={{ color: C.textMuted, textAlign: 'center', padding: '20px 0' }}>已有识别结果，点击"重新识别"更新</div>
                  ) : (
                    <div style={{ color: C.textMuted, textAlign: 'center', padding: '40px 0' }}>
                      <div style={{ fontSize: '24px', marginBottom: '8px' }}>🔍</div>
                      点击「AI识别」让AI读取图片中的文字内容
                    </div>
                  )}
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Toast */}
      {toast && (
        <div style={{
          position: 'fixed', bottom: '32px', left: '50%', transform: 'translateX(-50%)',
          padding: '12px 24px', borderRadius: '10px',
          background: toast.type === 'error' ? '#FEF2F2' : '#1F2937',
          color: toast.type === 'error' ? C.danger : '#fff',
          fontSize: '14px', fontWeight: 500, boxShadow: '0 8px 24px rgba(0,0,0,0.15)',
          zIndex: 9999, whiteSpace: 'nowrap', border: toast.type === 'error' ? '1px solid #FECACA' : 'none',
        }}>
          {toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}
        </div>
      )}
    </div>
  )
}

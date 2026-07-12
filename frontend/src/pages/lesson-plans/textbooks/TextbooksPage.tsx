/**
 * TextbooksPage — 课本管理页面
 *
 * 迭代7：上传/查看/OCR识别/删除课本图片
 * v231升级：
 *   1. 多选批量上传（一次选多张图，页码从起始页码自动递增，逐张上传显示进度）
 *   2. 新增"学期""单元"两个归档维度（表单填写 + 列表筛选）
 *   3. 筛选升级为 学科 + 年级 + 学期 + 单元 + 教材名搜索
 *   4. 列表按"教材名 · 学期"分组，卡片显示单元/章节标签，便于按单元查找管理
 *   5. 上传默认 scope=public（所有人可见），同校/同组老师都能查到
 */
import { useState, useEffect, useCallback, useRef } from 'react'
import { DEFAULT_SUBJECTS } from '@/constants/subjects'
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

const SUBJECTS = ['全部', ...DEFAULT_SUBJECTS]  // 单一真相源（方案甲，v231）
const GRADES = ['七年级', '八年级', '九年级', '高一', '高二', '高三', '小学低段', '小学中段', '小学高段']
// v231：学期选项（含"全册"表示不分学期整本）
const SEMESTERS = ['上册', '下册', '第一学期', '第二学期', '全册']

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

  // 筛选（v231：新增 semester/unit 两维）
  const [subjectFilter, setSubjectFilter] = useState('全部')
  const [gradeFilter, setGradeFilter] = useState('全部')
  const [semesterFilter, setSemesterFilter] = useState('全部')
  const [unitFilter, setUnitFilter] = useState('')
  const [searchText, setSearchText] = useState('')

  // 上传表单（v231：新增 uploadSemester/uploadUnit，文件改为多选数组）
  const [showUpload, setShowUpload] = useState(false)
  const [uploadSubject, setUploadSubject] = useState('AI')
  const [uploadGrade, setUploadGrade] = useState('七年级')
  const [uploadSemester, setUploadSemester] = useState('上册')
  const [uploadUnit, setUploadUnit] = useState('')
  const [uploadTextbookName, setUploadTextbookName] = useState('')
  const [uploadChapter, setUploadChapter] = useState('')
  const [uploadStartPage, setUploadStartPage] = useState(1) // 起始页码，多图自动递增
  const [uploadDesc, setUploadDesc] = useState('')
  const [uploadFiles, setUploadFiles] = useState<File[]>([]) // v231：多选
  const [uploading, setUploading] = useState(false)
  const [uploadProgress, setUploadProgress] = useState('') // 上传进度文案
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
      const params: Record<string, string | number> = { limit: 200 }
      if (subjectFilter !== '全部') params.subject = subjectFilter
      if (gradeFilter !== '全部') params.grade_range = gradeFilter
      if (semesterFilter !== '全部') params.semester = semesterFilter
      if (unitFilter.trim()) params.unit = unitFilter.trim()
      if (searchText.trim()) params.textbook_name = searchText.trim()
      const resp = await getTextbooks(params)
      setPages(resp.pages || []); setTotal(resp.total || 0)
    } catch { showToast('加载失败', 'error') }
    finally { setLoading(false) }
  }, [subjectFilter, gradeFilter, semesterFilter, unitFilter, searchText])

  useEffect(() => { loadPages() }, [loadPages])

  // ==================== 批量上传（v231核心）====================
  // 逐张上传，页码从起始页码自动递增，实时显示 第N/M张 进度
  const handleUpload = async () => {
    if (uploadFiles.length === 0 || !uploadTextbookName.trim()) {
      showToast('请选择图片并填写教材名称', 'error'); return
    }
    setUploading(true)
    let successCount = 0
    const failedNames: string[] = []
    try {
      for (let i = 0; i < uploadFiles.length; i++) {
        const file = uploadFiles[i]
        setUploadProgress(`上传中 ${i + 1}/${uploadFiles.length}：${file.name}`)
        const fd = new FormData()
        fd.append('file', file)
        fd.append('subject', uploadSubject)
        fd.append('grade_range', uploadGrade)
        fd.append('semester', uploadSemester)
        fd.append('unit', uploadUnit.trim())
        fd.append('textbook_name', uploadTextbookName.trim())
        fd.append('chapter', uploadChapter.trim())
        // 页码：起始页码 + 当前序号，多图自动连续递增
        fd.append('page_number', String(uploadStartPage + i))
        fd.append('description', uploadDesc.trim())
        fd.append('scope', 'public') // 默认所有人可见
        try {
          await uploadTextbook(fd)
          successCount++
        } catch {
          failedNames.push(file.name)
        }
      }
      // 汇总结果
      if (failedNames.length === 0) {
        showToast(`成功上传 ${successCount} 张 ✓`)
      } else {
        showToast(`成功 ${successCount} 张，失败 ${failedNames.length} 张（${failedNames.join('、')}）`, 'error')
      }
      // 重置文件选择（保留学科/年级/学期/单元/教材名等，方便连续上传下一批）
      setUploadFiles([])
      setUploadStartPage(uploadStartPage + successCount) // 起始页码自动推进，方便接着传下一单元
      if (fileInputRef.current) fileInputRef.current.value = ''
      await loadPages()
    } finally {
      setUploading(false)
      setUploadProgress('')
    }
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
  const filterSel: (active: boolean) => React.CSSProperties = (active) => ({
    ...inputSt, cursor: 'pointer',
    background: active ? C.primaryLight : 'transparent',
    color: active ? C.primary : C.textSec,
    borderColor: active ? C.primary : C.border,
  })

  // ==================== 分组：按 教材名 · 学期 归档 ====================
  const grouped = pages.reduce<Record<string, TextbookListItem[]>>((acc, p) => {
    const semLabel = p.semester ? ` · ${p.semester}` : ''
    const key = `${p.textbook_name || '未分类'}${semLabel}`
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
          <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '16px' }}>📷 上传课本图片（可一次选多张）</div>

          {/* 学科 + 年级 */}
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

          {/* 学期（v231新增） */}
          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>学期</label>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
              {SEMESTERS.map(s => (
                <button key={s} onClick={() => setUploadSemester(s)} style={selBtn(uploadSemester === s)}>{s}</button>
              ))}
            </div>
          </div>

          {/* 教材名 + 单元 + 章节 + 起始页码 */}
          <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 1fr 90px', gap: '12px', marginBottom: '16px' }}>
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>教材名称 *</label>
              <input value={uploadTextbookName} onChange={e => setUploadTextbookName(e.target.value)}
                placeholder="如：人教版七年级上册数学" style={{ ...inputSt, width: '100%' }} />
            </div>
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>单元</label>
              <input value={uploadUnit} onChange={e => setUploadUnit(e.target.value)}
                placeholder="如：第三单元" style={{ ...inputSt, width: '100%' }} />
            </div>
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>章节</label>
              <input value={uploadChapter} onChange={e => setUploadChapter(e.target.value)}
                placeholder="如：一元一次方程" style={{ ...inputSt, width: '100%' }} />
            </div>
            <div>
              <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>起始页码</label>
              <input type="number" value={uploadStartPage} onChange={e => setUploadStartPage(parseInt(e.target.value) || 1)}
                min={1} style={{ ...inputSt, width: '100%' }} />
            </div>
          </div>

          {/* 描述 */}
          <div style={{ marginBottom: '16px' }}>
            <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>描述（可选）</label>
            <input value={uploadDesc} onChange={e => setUploadDesc(e.target.value)}
              placeholder="对这批图片的补充说明" style={{ ...inputSt, width: '100%' }} />
          </div>

          {/* 文件选择（多选）+ 上传按钮 */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '16px' }}>
            <div style={{ flex: 1 }}>
              <input ref={fileInputRef} type="file" multiple accept="image/jpeg,image/png,image/webp"
                onChange={e => setUploadFiles(e.target.files ? Array.from(e.target.files) : [])}
                style={{ fontSize: '13px', color: C.textSec }} />
              {uploadFiles.length > 0 && (
                <span style={{ fontSize: '12px', color: C.primary, marginLeft: '8px', fontWeight: 600 }}>
                  已选 {uploadFiles.length} 张（页码 {uploadStartPage} ~ {uploadStartPage + uploadFiles.length - 1}）
                </span>
              )}
            </div>
            <button onClick={handleUpload} disabled={uploading || uploadFiles.length === 0 || !uploadTextbookName.trim()} style={{
              padding: '9px 24px', borderRadius: '8px', border: 'none', fontSize: '14px', fontWeight: 600, cursor: 'pointer',
              background: uploading || uploadFiles.length === 0 || !uploadTextbookName.trim() ? C.border : C.success,
              color: uploading || uploadFiles.length === 0 || !uploadTextbookName.trim() ? C.textMuted : '#fff',
              whiteSpace: 'nowrap',
            }}>{uploading ? '上传中...' : `确认上传${uploadFiles.length > 0 ? ` (${uploadFiles.length}张)` : ''}`}</button>
          </div>
          {/* 上传进度 */}
          {uploading && uploadProgress && (
            <div style={{ marginTop: '12px', fontSize: '13px', color: C.primary }}>⏳ {uploadProgress}</div>
          )}
          <div style={{ marginTop: '10px', fontSize: '12px', color: C.textMuted }}>
            提示：一次可选多张图片，系统会按选择顺序从「起始页码」开始自动编号；上传后所有老师都能查询到这些教材。
          </div>
        </div>
      )}

      {/* 筛选栏（v231：学科 + 年级 + 学期 + 单元 + 搜索） */}
      <div style={{ background: C.card, borderRadius: '12px', border: `1px solid ${C.border}`, padding: '16px 20px', marginBottom: '20px', display: 'flex', gap: '16px', alignItems: 'center', flexWrap: 'wrap' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec }}>学科</span>
          <select value={subjectFilter} onChange={e => setSubjectFilter(e.target.value)} style={filterSel(subjectFilter !== '全部')}>
            {SUBJECTS.map(s => <option key={s} value={s}>{s}</option>)}
          </select>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec }}>年级</span>
          <select value={gradeFilter} onChange={e => setGradeFilter(e.target.value)} style={filterSel(gradeFilter !== '全部')}>
            <option value="全部">全部</option>
            {GRADES.map(g => <option key={g} value={g}>{g}</option>)}
          </select>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec }}>学期</span>
          <select value={semesterFilter} onChange={e => setSemesterFilter(e.target.value)} style={filterSel(semesterFilter !== '全部')}>
            <option value="全部">全部</option>
            {SEMESTERS.map(s => <option key={s} value={s}>{s}</option>)}
          </select>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec }}>单元</span>
          <input value={unitFilter} onChange={e => setUnitFilter(e.target.value)}
            placeholder="如：第三单元" style={{ ...inputSt, width: '120px' }} />
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px', flex: 1, minWidth: '180px' }}>
          <span style={{ fontSize: '13px', fontWeight: 500, color: C.textSec }}>搜索</span>
          <input value={searchText} onChange={e => setSearchText(e.target.value)}
            placeholder="搜索教材名称..." style={{ ...inputSt, flex: 1 }} />
        </div>
      </div>

      {/* 课本列表（按 教材名·学期 分组） */}
      {loading ? (
        <div style={{ textAlign: 'center', padding: '60px', color: C.textMuted }}>加载中...</div>
      ) : Object.keys(grouped).length === 0 ? (
        <div style={{ textAlign: 'center', padding: '80px 40px', background: C.card, borderRadius: '12px', border: `1px solid ${C.border}` }}>
          <div style={{ fontSize: '48px', marginBottom: '16px' }}>📷</div>
          <div style={{ fontSize: '16px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>没有匹配的课本图片</div>
          <div style={{ fontSize: '14px', color: C.textMuted }}>上传课本页面的真实图片，AI可以精准识别课本内容辅助备课</div>
        </div>
      ) : (
        Object.entries(grouped).map(([groupName, items]) => (
          <div key={groupName} style={{ marginBottom: '20px' }}>
            <div style={{ fontSize: '15px', fontWeight: 600, color: C.text, marginBottom: '10px', display: 'flex', alignItems: 'center', gap: '8px' }}>
              <span>📚</span> {groupName}
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
                      {item.unit || item.chapter || `第${item.page_number}页`}
                    </div>
                    {/* 单元/章节/页码次级信息 */}
                    <div style={{ fontSize: '11px', color: C.textMuted, marginBottom: '6px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {[item.chapter, `第${item.page_number}页`].filter(Boolean).join(' · ')}
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
                  {[previewItem.semester, previewItem.unit, previewItem.chapter, `第${previewItem.page_number}页`].filter(Boolean).join(' · ')} · {previewItem.subject} · {previewItem.grade_range} · by {previewItem.uploader_name}
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

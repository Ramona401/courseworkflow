/**
 * TextbookAttachModal — 对话模式课本关联弹窗
 *
 * 老师在备课对话中关联课本图片，AI贴着课本原文设计教案。
 * 只显示与当前教案「学科+年级」匹配的、以及自己上传的课本图片。
 *
 * v231升级（教材去中心化 + 归档 + 自助删除，接口保持不变）：
 *   1. 集中式「课本管理页」已下线，上传/选择/删除全部收进本弹窗，用完即走。
 *   2. 上传补充「学期 / 单元」两个归档字段。
 *   3. 老师可删除自己上传的图片（后端 service 层做归属校验，只能删本人上传）。
 *   4. 上传区顶部新增版权与使用声明。
 *   ⚠️ props 接口（planId/subject/grade/currentPageIds/onSuccess/onCancel）保持原样不变。
 */
import { useState, useEffect, useCallback } from 'react'
import {
  getTextbooks, uploadTextbook, deleteTextbook, triggerTextbookOCR,
  type TextbookListItem,
} from '@/api/textbooks'
import { useAuth } from '@/store/auth'
import { useEducationProfile } from '@/hooks/useEducationProfile'
import ProtectedTextbookImage from '@/components/textbooks/ProtectedTextbookImage'

/* ==================== 颜色常量 ==================== */
const C = {
  primary: '#4F7BE8', primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981', danger: '#EF4444', warning: '#B45309',
  warningBg: 'rgba(245,158,11,0.06)', warningBorder: 'rgba(245,158,11,0.25)',
  text: '#1F2937', textSec: '#6B7280', textMuted: '#9CA3AF',
  card: '#FFFFFF', border: '#F3F4F6', bg: '#FAFBFC',
}

// v231：学期选项（与教材归档口径一致）
const SEMESTERS = ['上册', '下册', '第一学期', '第二学期', '全册']

/* ==================== Props（保持原始接口不变）==================== */
interface Props {
  planId: string                          // 当前教案ID
  subject: string                         // 当前教案学科（锁定）
  grade: string                           // 当前教案年级（锁定）
  currentPageIds: string[]                // 已关联的课本图片ID
  onSuccess: (pageIds: string[]) => void  // 确认关联回调（只传ID数组）
  onCancel: () => void                    // 取消/关闭回调
}

export default function TextbookAttachModal({ planId, subject, grade, currentPageIds, onSuccess, onCancel }: Props) {
  const { user } = useAuth()
  const { isK12 } = useEducationProfile()

  // 列表 + 选择
  const [pages, setPages] = useState<TextbookListItem[]>([])
  const [loading, setLoading] = useState(true)
  const [selectedIds, setSelectedIds] = useState<string[]>(currentPageIds || [])

  // 上传表单（v231：新增 upSemester/upUnit）
  const [showUpload, setShowUpload] = useState(false)
  const [upSemester, setUpSemester] = useState('上册')
  const [upUnit, setUpUnit] = useState('')
  const [upName, setUpName] = useState('')
  const [upChapter, setUpChapter] = useState('')
  const [upStartPage, setUpStartPage] = useState(1)
  const [upFiles, setUpFiles] = useState<File[]>([])
  const [uploading, setUploading] = useState(false)
  const [upProgress, setUpProgress] = useState('')

  // 操作状态
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [ocrRunningId, setOcrRunningId] = useState<string | null>(null)
  const [toast, setToast] = useState<{ msg: string; type: 'success' | 'error' } | null>(null)
  const showToast = (msg: string, type: 'success' | 'error' = 'success') => {
    setToast({ msg, type }); setTimeout(() => setToast(null), 2600)
  }

  // ==================== 加载课本列表（按当前教案学科+年级过滤）====================
  const loadPages = useCallback(async () => {
    // 当前教育域不为K12时不发送课本列表请求。
    // 后端仍保留独立硬闸，前端判断只用于避免无意义请求。
    if (!isK12) {
      setPages([])
      setLoading(false)
      return
    }

    setLoading(true)
    try {
      const params: Record<string, string | number> = { limit: 200 }
      if (subject) params.subject = subject
      if (grade) params.grade_range = grade
      const resp = await getTextbooks(params)
      setPages(resp.pages || [])
    } catch { showToast('加载课本列表失败', 'error') }
    finally { setLoading(false) }
  }, [isK12, subject, grade])

  useEffect(() => { loadPages() }, [loadPages])

  // ==================== 切换选择 ====================
  const toggleSelect = (id: string) => {
    setSelectedIds(prev => prev.includes(id) ? prev.filter(x => x !== id) : [...prev, id])
  }

  // ==================== 批量上传（页码自动递增，上传后自动选中+自动OCR）====================
  const handleUpload = async () => {
    // 非K12不能通过缓存弹窗触发课本上传。
    if (!isK12) {
      showToast('当前教育域暂无课本能力', 'error')
      return
    }

    if (upFiles.length === 0 || !upName.trim()) {
      showToast('请选择图片并填写教材名称', 'error'); return
    }
    setUploading(true)
    const newIds: string[] = []
    const failed: string[] = []
    try {
      for (let i = 0; i < upFiles.length; i++) {
        const file = upFiles[i]
        setUpProgress(`上传中 ${i + 1}/${upFiles.length}：${file.name}`)
        const fd = new FormData()
        fd.append('file', file)
        fd.append('subject', subject)
        fd.append('grade_range', grade)
        fd.append('semester', upSemester)
        fd.append('unit', upUnit.trim())
        fd.append('textbook_name', upName.trim())
        fd.append('chapter', upChapter.trim())
        fd.append('page_number', String(upStartPage + i))
        fd.append('description', '')
        fd.append('scope', 'public') // 默认所有人可见
        try {
          const resp = await uploadTextbook(fd)
          newIds.push(resp.id)
        } catch { failed.push(file.name) }
      }
      if (failed.length === 0) showToast(`成功上传 ${newIds.length} 张，正在自动识别文字…`)
      else showToast(`成功 ${newIds.length} 张，失败 ${failed.length} 张`, 'error')

      // 上传成功的自动选中
      if (newIds.length > 0) {
        setSelectedIds(prev => Array.from(new Set([...prev, ...newIds])))
      }
      setUpFiles([])
      setUpStartPage(upStartPage + newIds.length) // 起始页码自动推进
      await loadPages()

      // 新上传的图片自动触发OCR识别（后台逐张，不阻塞）
      for (const id of newIds) {
        try { await triggerTextbookOCR(id) } catch { /* 单张失败忽略，老师可在详情里手动重试 */ }
      }
      await loadPages() // OCR完刷新has_ocr状态
    } finally {
      setUploading(false)
      setUpProgress('')
    }
  }

  // ==================== 删除自己上传的图 ====================
  const handleDelete = async (e: React.MouseEvent, item: TextbookListItem) => {
    e.stopPropagation()
    // 非K12不能通过缓存弹窗触发课本删除。
    if (!isK12) {
      showToast('当前教育域暂无课本能力', 'error')
      return
    }

    if (!confirm(`确定删除自己上传的「${item.file_name}」？删除后无法恢复。`)) return
    setDeletingId(item.id)
    try {
      await deleteTextbook(item.id)
      setSelectedIds(prev => prev.filter(x => x !== item.id))
      showToast('已删除')
      await loadPages()
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : '删除失败（仅能删自己上传的图）', 'error')
    } finally { setDeletingId(null) }
  }

  // ==================== 单张手动OCR ====================
  const handleOCR = async (e: React.MouseEvent, id: string) => {
    e.stopPropagation()
    // 非K12不能通过缓存弹窗触发OCR。
    if (!isK12) {
      showToast('当前教育域暂无课本能力', 'error')
      return
    }

    setOcrRunningId(id)
    try {
      await triggerTextbookOCR(id)
      showToast('识别完成 ✓')
      await loadPages()
    } catch (err: unknown) {
      showToast(err instanceof Error ? err.message : 'AI识别失败', 'error')
    } finally { setOcrRunningId(null) }
  }

  // ==================== 确认关联（保持原回调签名：只传ID数组）====================
  const handleConfirm = () => {
    if (!isK12) {
      showToast('当前教育域暂无课本能力', 'error')
      return
    }

    onSuccess(selectedIds)
  }

  // ==================== 样式 ====================
  const inputSt: React.CSSProperties = {
    padding: '7px 10px', borderRadius: '6px', border: `1px solid ${C.border}`,
    fontSize: '13px', color: C.text, outline: 'none', boxSizing: 'border-box', fontFamily: 'inherit',
  }
  const semBtn = (active: boolean): React.CSSProperties => ({
    padding: '4px 10px', borderRadius: '16px', border: `1px solid ${active ? C.primary : C.border}`,
    background: active ? C.primaryLight : 'transparent', color: active ? C.primary : C.textSec,
    fontSize: '12px', fontWeight: active ? 600 : 400, cursor: 'pointer',
  })

  if (!isK12) {
    return (
      <div
        style={{
          position: 'fixed',
          inset: 0,
          background: 'rgba(0,0,0,0.6)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          zIndex: 10000,
        }}
        onClick={event => {
          if (event.target === event.currentTarget) {
            onCancel()
          }
        }}
      >
        <div
          style={{
            width: '520px',
            maxWidth: 'calc(100vw - 32px)',
            background: C.card,
            borderRadius: '16px',
            boxShadow: '0 20px 60px rgba(0,0,0,0.3)',
            overflow: 'hidden',
          }}
        >
          <div
            style={{
              padding: '16px 24px',
              borderBottom: `1px solid ${C.border}`,
              display: 'flex',
              justifyContent: 'space-between',
              alignItems: 'center',
            }}
          >
            <div
              style={{
                fontSize: '16px',
                fontWeight: 700,
                color: C.text,
              }}
            >
              📷 关联课本图片
            </div>

            <button
              onClick={onCancel}
              style={{
                border: 'none',
                background: 'none',
                cursor: 'pointer',
                fontSize: '20px',
                color: C.textMuted,
              }}
            >
              ✕
            </button>
          </div>

          <div
            style={{
              padding: '64px 24px',
              textAlign: 'center',
            }}
          >
            <div
              style={{
                fontSize: '42px',
                marginBottom: '14px',
              }}
            >
              📚
            </div>

            <div
              style={{
                fontSize: '16px',
                fontWeight: 700,
                color: C.text,
              }}
            >
              当前教育域暂无课本能力
            </div>
          </div>

          <div
            style={{
              padding: '14px 24px',
              borderTop: `1px solid ${C.border}`,
              display: 'flex',
              justifyContent: 'flex-end',
            }}
          >
            <button
              onClick={onCancel}
              style={{
                padding: '8px 20px',
                borderRadius: '8px',
                border: 'none',
                background: C.primary,
                color: '#fff',
                fontSize: '13px',
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              关闭
            </button>
          </div>
        </div>
      </div>
    )
  }

  return (
    <div style={{ position: 'fixed', inset: 0, background: 'rgba(0,0,0,0.6)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 10000 }}
      onClick={e => { if (e.target === e.currentTarget) onCancel() }}>
      <div style={{ background: C.card, borderRadius: '16px', width: '760px', maxHeight: '88vh', display: 'flex', flexDirection: 'column', boxShadow: '0 20px 60px rgba(0,0,0,0.3)' }}>

        {/* 标题栏 */}
        <div style={{ padding: '16px 24px', borderBottom: `1px solid ${C.border}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div>
            <div style={{ fontSize: '16px', fontWeight: 700, color: C.text }}>📷 关联课本图片</div>
            <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '3px' }}>
              选中后AI会贴着课文原文来设计，下一轮对话起生效。未识别的页面会自动进行AI识别。
            </div>
          </div>
          <button onClick={onCancel} style={{ border: 'none', background: 'none', cursor: 'pointer', fontSize: '20px', color: C.textMuted }}>✕</button>
        </div>

        {/* 内容区（可滚动）*/}
        <div style={{ flex: 1, overflowY: 'auto', padding: '16px 24px' }}>

          {/* 顶部：学科年级锁定提示 + 上传切换 */}
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '14px' }}>
            <div style={{ fontSize: '13px', color: C.textSec }}>
              📗 仅显示：<strong style={{ color: C.text }}>{subject} · {grade}</strong>（与本教案一致）
            </div>
            <button onClick={() => setShowUpload(!showUpload)} style={{
              padding: '6px 14px', borderRadius: '8px',
              border: `1px solid ${C.primary}`, background: showUpload ? C.primaryLight : 'transparent',
              color: C.primary, fontSize: '13px', fontWeight: 600, cursor: 'pointer',
            }}>{showUpload ? '收起上传' : '＋ 上传课本页'}</button>
          </div>

          {/* 上传区 */}
          {showUpload && (
            <div style={{ background: C.bg, borderRadius: '10px', border: `1px dashed ${C.primary}`, padding: '16px', marginBottom: '16px' }}>
              <div style={{ fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '12px' }}>
                📷 上传课本页（{subject} · {grade}，自动归入本学科年级）
              </div>

              {/* 版权与使用声明 */}
              <div style={{
                background: C.warningBg, border: `1px solid ${C.warningBorder}`, borderRadius: '8px',
                padding: '10px 12px', marginBottom: '14px', fontSize: '12px', color: C.warning, lineHeight: 1.6,
              }}>
                ⚠️ <strong>版权与使用声明</strong>：本平台不出售、不传播任何教材内容。上传的教材图片仅供本人及本校教师在平台内备课参考，著作权归原教材出版方所有。请确保用于个人教学研究等合理使用范围，不得对外分发、商用或公开传播；因上传或使用引发的版权责任由上传者自行承担。
              </div>

              {/* 学期 */}
              <div style={{ marginBottom: '12px' }}>
                <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>学期</label>
                <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px' }}>
                  {SEMESTERS.map(s => (
                    <button key={s} onClick={() => setUpSemester(s)} style={semBtn(upSemester === s)}>{s}</button>
                  ))}
                </div>
              </div>

              {/* 教材名 + 单元 + 章节 + 起始页码 */}
              <div style={{ display: 'grid', gridTemplateColumns: '2fr 1fr 1fr 80px', gap: '10px', marginBottom: '12px' }}>
                <div>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.text, marginBottom: '5px' }}>教材名称 *</label>
                  <input value={upName} onChange={e => setUpName(e.target.value)} placeholder="如：人教版一年级上册语文" style={{ ...inputSt, width: '100%' }} />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.text, marginBottom: '5px' }}>单元</label>
                  <input value={upUnit} onChange={e => setUpUnit(e.target.value)} placeholder="第三单元" style={{ ...inputSt, width: '100%' }} />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.text, marginBottom: '5px' }}>章节（可选）</label>
                  <input value={upChapter} onChange={e => setUpChapter(e.target.value)} placeholder="如：观潮" style={{ ...inputSt, width: '100%' }} />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: C.text, marginBottom: '5px' }}>起始页</label>
                  <input type="number" value={upStartPage} onChange={e => setUpStartPage(parseInt(e.target.value) || 1)} min={1} style={{ ...inputSt, width: '100%' }} />
                </div>
              </div>

              {/* 文件选择（多选）+ 上传 */}
              <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                <div style={{ flex: 1 }}>
                  <input type="file" multiple accept="image/jpeg,image/png,image/webp"
                    onChange={e => setUpFiles(e.target.files ? Array.from(e.target.files) : [])}
                    style={{ fontSize: '12px', color: C.textSec }} />
                  {upFiles.length > 0 && (
                    <span style={{ fontSize: '11px', color: C.primary, marginLeft: '6px', fontWeight: 600 }}>
                      已选 {upFiles.length} 张（页 {upStartPage}~{upStartPage + upFiles.length - 1}）
                    </span>
                  )}
                </div>
                <button onClick={handleUpload} disabled={uploading || upFiles.length === 0 || !upName.trim()} style={{
                  padding: '7px 18px', borderRadius: '8px', border: 'none', fontSize: '13px', fontWeight: 600,
                  cursor: uploading || upFiles.length === 0 || !upName.trim() ? 'not-allowed' : 'pointer',
                  background: uploading || upFiles.length === 0 || !upName.trim() ? C.border : C.success,
                  color: uploading || upFiles.length === 0 || !upName.trim() ? C.textMuted : '#fff', whiteSpace: 'nowrap',
                }}>{uploading ? '上传中...' : `确认上传${upFiles.length > 0 ? ` (${upFiles.length})` : ''}`}</button>
              </div>
              {uploading && upProgress && (
                <div style={{ marginTop: '10px', fontSize: '12px', color: C.primary }}>⏳ {upProgress}</div>
              )}
              <div style={{ marginTop: '8px', fontSize: '11px', color: C.textMuted }}>
                可一次选多张 · JPG/PNG/WEBP · 单张≤10MB；上传后自动选中并识别文字，所有老师都能查询到这些教材。
              </div>
            </div>
          )}

          {/* 课本图片网格 */}
          {loading ? (
            <div style={{ textAlign: 'center', padding: '50px', color: C.textMuted }}>加载中...</div>
          ) : pages.length === 0 ? (
            <div style={{ textAlign: 'center', padding: '50px 20px', color: C.textMuted }}>
              <div style={{ fontSize: '40px', marginBottom: '12px' }}>📮</div>
              <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '6px' }}>课本库里还没有 {subject} · {grade} 的课本页</div>
              <div style={{ fontSize: '12px', lineHeight: 1.7 }}>
                用手机拍下课本页面，在上方上传区直接上传即可——<br />
                上传后会自动进行AI识别并选中，确认关联后AI就能贴着课文设计了。
              </div>
            </div>
          ) : (
            <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fill, minmax(150px, 1fr))', gap: '10px' }}>
              {pages.map(item => {
                const selected = selectedIds.includes(item.id)
                const isMine = user?.id === item.uploaded_by
                return (
                  <div key={item.id} onClick={() => toggleSelect(item.id)} style={{
                    position: 'relative', background: C.card, borderRadius: '10px',
                    border: `2px solid ${selected ? C.primary : C.border}`, overflow: 'hidden', cursor: 'pointer',
                    boxShadow: selected ? '0 4px 12px rgba(79,123,232,0.15)' : 'none', transition: 'all 150ms ease',
                  }}>
                    {/* 选中勾 */}
                    {selected && (
                      <div style={{ position: 'absolute', top: '6px', right: '6px', width: '22px', height: '22px', borderRadius: '50%', background: C.primary, color: '#fff', display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '13px', zIndex: 2 }}>✓</div>
                    )}
                    {/* 缩略图 */}
                    <div style={{ height: '110px', background: C.bg, display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden' }}>
                      <ProtectedTextbookImage
                        textbookId={item.id}
                        alt={item.file_name}
                        style={{
                          maxWidth: '100%',
                          maxHeight: '100%',
                          objectFit: 'contain',
                        }}
                        fallback="📷"
                      />
                    </div>
                    {/* 信息 */}
                    <div style={{ padding: '8px 10px' }}>
                      <div style={{ fontSize: '12px', fontWeight: 600, color: C.text, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {item.unit || item.chapter || `第${item.page_number}页`}
                      </div>
                      <div style={{ fontSize: '10px', color: C.textMuted, marginTop: '2px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                        {[item.semester, `第${item.page_number}页`].filter(Boolean).join(' · ')}
                      </div>
                      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '4px' }}>
                        {/* OCR状态/按钮 */}
                        {ocrRunningId === item.id ? (
                          <span style={{ fontSize: '9px', color: C.primary }}>识别中…</span>
                        ) : item.has_ocr ? (
                          <span style={{ fontSize: '9px', padding: '1px 5px', borderRadius: '4px', background: 'rgba(16,185,129,0.08)', color: C.success }}>已识别</span>
                        ) : (
                          <button onClick={e => handleOCR(e, item.id)}
                            style={{ border: 'none', background: 'none', cursor: 'pointer', fontSize: '9px', color: C.primary, padding: 0, textDecoration: 'underline' }}
                            title="AI识别文字">识别文字</button>
                        )}
                        {/* 删除按钮：仅自己上传的可见 */}
                        {isMine && (
                          <button onClick={e => handleDelete(e, item)} disabled={deletingId === item.id}
                            style={{ border: 'none', background: 'none', cursor: 'pointer', fontSize: '11px', color: C.danger, padding: '0 2px' }}
                            title="删除自己上传的图片">
                            {deletingId === item.id ? '…' : '🗑'}
                          </button>
                        )}
                      </div>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {/* 底部操作栏 */}
        <div style={{ padding: '14px 24px', borderTop: `1px solid ${C.border}`, display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <span style={{ fontSize: '13px', color: C.textSec }}>已选 {selectedIds.length} 张课本页</span>
          <div style={{ display: 'flex', gap: '10px' }}>
            <button onClick={onCancel} style={{ padding: '8px 18px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', color: C.textSec, fontSize: '13px', cursor: 'pointer' }}>取消</button>
            <button onClick={handleConfirm} style={{
              padding: '8px 22px', borderRadius: '8px', border: 'none',
              background: C.primary,
              color: '#fff',
              fontSize: '13px', fontWeight: 600, cursor: 'pointer',
            }}>
              {selectedIds.length === 0
                ? currentPageIds.length > 0
                  ? '解除全部关联'
                  : '确认不关联课本'
                : `确认关联 (${selectedIds.length})`}
            </button>
          </div>
        </div>

        {/* Toast */}
        {toast && (
          <div style={{
            position: 'absolute', bottom: '80px', left: '50%', transform: 'translateX(-50%)',
            padding: '10px 20px', borderRadius: '10px',
            background: toast.type === 'error' ? '#FEF2F2' : '#1F2937',
            color: toast.type === 'error' ? C.danger : '#fff',
            fontSize: '13px', fontWeight: 500, boxShadow: '0 8px 24px rgba(0,0,0,0.15)',
            zIndex: 10, whiteSpace: 'nowrap', border: toast.type === 'error' ? '1px solid #FECACA' : 'none',
          }}>{toast.type === 'success' ? '✓ ' : '⚠️ '}{toast.msg}</div>
        )}
      </div>
    </div>
  )
}

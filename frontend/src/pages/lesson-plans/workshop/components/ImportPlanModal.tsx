/**
 * ImportPlanModal.tsx — 已有教案导入弹窗
 *
 * v108新增：支持老师将现有教案导入系统，跳过备课阶段直接进入AI评审
 *
 * 功能：
 *   - 两步流程：第一步填写信息+上传内容，第二步关联课本（可选）
 *   - 支持三种内容来源：粘贴文本 / Word(.docx) / PDF
 *   - Word解析：原生FileReader + JSZip(CDN) + XML解析，零npm依赖
 *   - PDF解析：CDN加载pdf.js，仅支持文字型PDF
 *   - 课本关联：复用 StartForm 的课本选择逻辑
 *
 * 设计原则：不引入任何新npm包，所有文件解析通过CDN动态加载实现
 */
import { useState, useEffect, useRef } from 'react'
import { useNavigate } from 'react-router-dom'
import { getTextbooks, type TextbookListItem } from '@/api/textbooks'
import { importExistingPlan, type ImportExistingPlanRequest, type ConversationMessage } from '@/api/lesson-plans'
import { C, SUBJECTS, GRADES } from './workshopConstants'

// ==================== 类型定义 ====================

interface ImportPlanModalProps {
  onSuccess: (planId: string, openingMessage: ConversationMessage) => void
  onCancel: () => void
}

type SourceType = 'paste' | 'docx' | 'pdf'
type Step = 1 | 2

// ==================== 样式工具 ====================

const selBtn = (active: boolean): React.CSSProperties => ({
  padding: '6px 14px', borderRadius: '20px',
  border: `1px solid ${active ? C.primary : C.border}`,
  background: active ? C.primaryLight : 'transparent',
  color: active ? C.primary : C.textSec,
  fontSize: '13px', fontWeight: active ? 600 : 400,
  cursor: 'pointer', transition: 'all 150ms ease',
})

// ==================== CDN加载工具 ====================

/**
 * loadScript — 运行时动态加载CDN脚本（幂等：已加载则直接resolve）
 * 用于按需加载JSZip（Word解析）和pdf.js（PDF解析），不增加初始bundle体积
 */
function loadScript(src: string, globalKey: string): Promise<void> {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  if ((window as any)[globalKey]) return Promise.resolve()
  return new Promise((resolve, reject) => {
    const existing = document.querySelector(`script[src="${src}"]`)
    if (existing) {
      // 脚本标签已存在但可能还在加载，等待load事件
      existing.addEventListener('load', () => resolve())
      existing.addEventListener('error', () => reject(new Error(`加载失败: ${src}`)))
      return
    }
    const script = document.createElement('script')
    script.src = src
    script.onload = () => resolve()
    script.onerror = () => reject(new Error(`CDN加载失败: ${src}`))
    document.head.appendChild(script)
  })
}

// ==================== Word文档解析 ====================

/**
 * parseDocxFile — 解析 .docx 文件，提取纯文本
 *
 * .docx 本质是ZIP压缩包，内含 word/document.xml
 * 通过CDN加载JSZip解压后，用DOMParser解析XML提取文字节点
 * 完全不依赖npm包
 */
async function parseDocxFile(file: File): Promise<string> {
  // 加载JSZip（CDN）
  await loadScript(
    'https://cdnjs.cloudflare.com/ajax/libs/jszip/3.10.1/jszip.min.js',
    'JSZip'
  )
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const JSZip = (window as any).JSZip
  if (!JSZip) throw new Error('JSZip加载失败')

  const arrayBuffer = await file.arrayBuffer()
  const zip = await JSZip.loadAsync(arrayBuffer)

  // 读取主文档XML
  const docXmlFile = zip.file('word/document.xml')
  if (!docXmlFile) throw new Error('不是有效的docx文件')

  const xmlStr = await docXmlFile.async('string')
  const parser = new DOMParser()
  const xmlDoc = parser.parseFromString(xmlStr, 'application/xml')

  // 提取所有 <w:t> 文字节点，按段落分组
  const paragraphs = xmlDoc.querySelectorAll('w\\:p, p')
  const lines: string[] = []

  paragraphs.forEach(para => {
    const texts = para.querySelectorAll('w\\:t, t')
    const line = Array.from(texts)
      .map(t => t.textContent || '')
      .join('')
      .trim()
    if (line) lines.push(line)
  })

  return lines.join('\n')
}

// ==================== PDF文档解析 ====================

/**
 * parsePdfFile — 解析文字型PDF，提取纯文本
 * 通过CDN加载pdf.js，逐页提取文字内容
 * 扫描型PDF无法提取（返回空字符串）
 */
async function parsePdfFile(file: File): Promise<string> {
  // 加载pdf.js（CDN）
  await loadScript(
    'https://cdnjs.cloudflare.com/ajax/libs/pdf.js/3.11.174/pdf.min.js',
    'pdfjsLib'
  )
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const pdfjsLib = (window as any).pdfjsLib
  if (!pdfjsLib) throw new Error('pdf.js加载失败')

  pdfjsLib.GlobalWorkerOptions.workerSrc =
    'https://cdnjs.cloudflare.com/ajax/libs/pdf.js/3.11.174/pdf.worker.min.js'

  const arrayBuffer = await file.arrayBuffer()
  const pdf = await pdfjsLib.getDocument({ data: arrayBuffer }).promise
  const textParts: string[] = []

  for (let i = 1; i <= pdf.numPages; i++) {
    const page = await pdf.getPage(i)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const textContent = await page.getTextContent()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const pageText = textContent.items.map((item: any) => item.str).join(' ').trim()
    if (pageText) textParts.push(pageText)
  }

  return textParts.join('\n\n')
}

// ==================== 主组件 ====================

export default function ImportPlanModal({ onSuccess, onCancel }: ImportPlanModalProps) {
  const navigate = useNavigate()

  // ---- 步骤控制 ----
  const [step, setStep] = useState<Step>(1)

  // ---- 第一步：基本信息 ----
  const [subject, setSubject]       = useState('语文')
  const [grade, setGrade]           = useState('七年级')
  const [topic, setTopic]           = useState('')
  const [duration, setDuration]     = useState(45)
  const [sourceType, setSourceType] = useState<SourceType>('paste')

  // ---- 内容相关 ----
  const [pasteContent, setPasteContent]   = useState('')
  const [parsedContent, setParsedContent] = useState('')  // 文件解析后的文本
  const [fileName, setFileName]           = useState('')
  const [parseError, setParseError]       = useState('')
  const [parsing, setParsing]             = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  // ---- 第二步：课本关联 ----
  const [textbooks, setTextbooks]               = useState<TextbookListItem[]>([])
  const [textbooksLoading, setTextbooksLoading] = useState(false)
  const [selectedTextbookIds, setSelectedTBIds] = useState<Set<string>>(new Set())

  // ---- 提交状态 ----
  const [submitting, setSubmitting]   = useState(false)
  const [submitError, setSubmitError] = useState('')

  // 当前有效内容
  const effectiveContent = sourceType === 'paste' ? pasteContent : parsedContent

  // ---- 进入第二步时加载课本列表 ----
  useEffect(() => {
    if (step !== 2) return
    setTextbooksLoading(true)
    getTextbooks({ subject, grade_range: grade, limit: 50 })
      .then(resp => setTextbooks(resp.pages || []))
      .catch(() => setTextbooks([]))
      .finally(() => setTextbooksLoading(false))
  }, [step, subject, grade])

  // ---- 文件选择处理 ----
  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setFileName(file.name)
    setParseError('')
    setParsedContent('')
    setParsing(true)
    try {
      let text = ''
      if (sourceType === 'docx') {
        text = await parseDocxFile(file)
      } else if (sourceType === 'pdf') {
        text = await parsePdfFile(file)
      }
      if (!text.trim()) {
        if (sourceType === 'pdf') {
          setParseError('该PDF为扫描件或无可提取文字，请复制PDF内容后改用粘贴方式导入')
        } else {
          setParseError('文档内容为空或无法提取文字，请改用粘贴文本方式')
        }
      } else {
        setParsedContent(text.trim())
      }
    } catch (err) {
      const msg = err instanceof Error ? err.message : '未知错误'
      if (msg.includes('CDN') || msg.includes('加载失败')) {
        setParseError(`解析库加载失败（请检查网络），或改用粘贴文本方式`)
      } else if (sourceType === 'docx') {
        setParseError('Word文档解析失败，请检查文件格式，或改用粘贴文本方式')
      } else {
        setParseError('PDF解析失败，请改用粘贴文本方式')
      }
    } finally {
      setParsing(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  // ---- 切换来源类型时重置文件状态 ----
  const handleSourceTypeChange = (newType: SourceType) => {
    setSourceType(newType)
    setParsedContent('')
    setFileName('')
    setParseError('')
  }

  // ---- 课本图片选择 ----
  const toggleTextbook = (id: string) => {
    setSelectedTBIds(prev => {
      const n = new Set(prev)
      n.has(id) ? n.delete(id) : n.add(id)
      return n
    })
  }

  // ---- 第一步验证 ----
  const step1Valid = topic.trim().length > 0 && effectiveContent.trim().length >= 50

  // ---- 提交导入 ----
  const handleSubmit = async () => {
    if (submitting) return
    setSubmitting(true)
    setSubmitError('')
    try {
      const req: ImportExistingPlanRequest = {
        subject,
        grade,
        topic: topic.trim(),
        duration_minutes: duration,
        content_markdown: effectiveContent.trim(),
        source_type: sourceType,
      }
      if (selectedTextbookIds.size > 0) {
        req.textbook_page_ids = Array.from(selectedTextbookIds)
      }
      const resp = await importExistingPlan(req)
      onSuccess(resp.plan.id, resp.opening_message)
    } catch {
      setSubmitError('导入失败，请检查内容后重试')
      setSubmitting(false)
    }
  }

  // ==================== 渲染 ====================

  return (
    <div
      style={{ position: 'fixed', inset: 0, zIndex: 10000, background: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '20px' }}
      onClick={onCancel}
    >
      <div
        onClick={e => e.stopPropagation()}
        style={{ background: '#fff', borderRadius: '16px', width: '100%', maxWidth: '680px', maxHeight: '90vh', boxShadow: '0 24px 64px rgba(0,0,0,0.18)', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
      >
        {/* ---- 标题栏 ---- */}
        <div style={{ padding: '20px 28px 16px', borderBottom: `1px solid ${C.border}`, display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
          <div>
            <h2 style={{ margin: 0, fontSize: '18px', fontWeight: 700, color: C.text }}>📂 导入已有教案</h2>
            <p style={{ margin: '4px 0 0', fontSize: '13px', color: C.textSec }}>将您现有的教案导入系统，AI将自动评审并提供改进建议</p>
          </div>
          {/* 步骤指示器 */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px' }}>
            {([1, 2] as Step[]).map(s => (
              <div key={s} style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                <div style={{ width: '24px', height: '24px', borderRadius: '50%', background: step >= s ? C.primary : '#E5E7EB', color: step >= s ? '#fff' : C.textMuted, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: '12px', fontWeight: 700 }}>{s}</div>
                <span style={{ color: step >= s ? C.primary : C.textMuted, fontWeight: step === s ? 600 : 400 }}>{s === 1 ? '内容信息' : '关联课本'}</span>
                {s < 2 && <span style={{ color: C.border }}>›</span>}
              </div>
            ))}
          </div>
        </div>

        {/* ---- 内容区 ---- */}
        <div style={{ flex: 1, overflowY: 'auto', padding: '24px 28px' }}>

          {/* ===== 第一步：内容信息 ===== */}
          {step === 1 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>

              {/* 学科 + 年级 */}
              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                <div>
                  <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>学科</label>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                    {SUBJECTS.map(s => <button key={s} onClick={() => setSubject(s)} style={selBtn(subject === s)}>{s}</button>)}
                  </div>
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>年级</label>
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
                    {GRADES.map(g => <button key={g} onClick={() => setGrade(g)} style={selBtn(grade === g)}>{g}</button>)}
                  </div>
                </div>
              </div>

              {/* 课题 + 课时 */}
              <div style={{ display: 'grid', gridTemplateColumns: '1fr auto', gap: '16px', alignItems: 'end' }}>
                <div>
                  <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
                    课题 <span style={{ color: C.danger }}>*</span>
                  </label>
                  <input
                    type="text" value={topic} onChange={e => setTopic(e.target.value)}
                    placeholder="请输入本节课课题名称"
                    style={{ width: '100%', padding: '10px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '14px', color: C.text, outline: 'none', boxSizing: 'border-box' }}
                    onFocus={e => { e.target.style.borderColor = C.primary }}
                    onBlur={e  => { e.target.style.borderColor = C.border }}
                  />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>课时</label>
                  <div style={{ display: 'flex', gap: '6px' }}>
                    {[40, 45, 50, 60].map(d => <button key={d} onClick={() => setDuration(d)} style={selBtn(duration === d)}>{d}分</button>)}
                  </div>
                </div>
              </div>

              {/* 来源类型选择 */}
              <div>
                <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>教案来源</label>
                <div style={{ display: 'flex', gap: '10px' }}>
                  {([
                    { key: 'paste' as SourceType, icon: '📋', label: '粘贴文本' },
                    { key: 'docx'  as SourceType, icon: '📝', label: 'Word文档' },
                    { key: 'pdf'   as SourceType, icon: '📄', label: 'PDF文件'  },
                  ]).map(opt => (
                    <button
                      key={opt.key}
                      onClick={() => handleSourceTypeChange(opt.key)}
                      style={{ flex: 1, padding: '12px', borderRadius: '10px', border: `2px solid ${sourceType === opt.key ? C.primary : C.border}`, background: sourceType === opt.key ? C.primaryLight : '#fff', cursor: 'pointer', transition: 'all 150ms ease', display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '6px' }}
                    >
                      <span style={{ fontSize: '22px' }}>{opt.icon}</span>
                      <span style={{ fontSize: '13px', fontWeight: sourceType === opt.key ? 600 : 400, color: sourceType === opt.key ? C.primary : C.text }}>{opt.label}</span>
                    </button>
                  ))}
                </div>
              </div>

              {/* 粘贴文本输入区 */}
              {sourceType === 'paste' && (
                <div>
                  <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
                    粘贴教案内容 <span style={{ color: C.danger }}>*</span>
                    <span style={{ fontWeight: 400, color: C.textMuted, marginLeft: '8px' }}>（支持Markdown格式，至少50字）</span>
                  </label>
                  <textarea
                    value={pasteContent}
                    onChange={e => setPasteContent(e.target.value)}
                    rows={12}
                    placeholder={'将您的教案内容粘贴到这里...\n\n支持Markdown格式，也可以直接粘贴纯文本。'}
                    style={{ width: '100%', boxSizing: 'border-box', padding: '12px 14px', borderRadius: '8px', border: `1px solid ${C.border}`, fontSize: '13px', lineHeight: 1.8, color: C.text, resize: 'vertical', fontFamily: 'inherit', outline: 'none' }}
                    onFocus={e => { e.target.style.borderColor = C.primary }}
                    onBlur={e  => { e.target.style.borderColor = C.border }}
                  />
                  <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '4px', textAlign: 'right' }}>
                    已输入 {pasteContent.replace(/\s/g, '').length} 字
                    {pasteContent.trim().length >= 50
                      ? <span style={{ color: C.success, marginLeft: '8px' }}>✓ 内容充足</span>
                      : <span style={{ color: C.danger, marginLeft: '8px' }}>请至少输入50字</span>
                    }
                  </div>
                </div>
              )}

              {/* Word / PDF 文件上传区 */}
              {(sourceType === 'docx' || sourceType === 'pdf') && (
                <div>
                  <label style={{ display: 'block', fontSize: '13px', fontWeight: 600, color: C.text, marginBottom: '8px' }}>
                    上传{sourceType === 'docx' ? 'Word文档' : 'PDF文件'}
                    <span style={{ fontWeight: 400, color: C.textMuted, marginLeft: '8px' }}>
                      （{sourceType === 'docx' ? '.docx格式' : '.pdf格式'}，系统自动提取文字内容）
                    </span>
                  </label>

                  {/* 点击上传区域 */}
                  <div
                    onClick={() => fileInputRef.current?.click()}
                    style={{ border: `2px dashed ${parsedContent ? C.success : parseError ? C.danger : C.border}`, borderRadius: '10px', padding: '28px', textAlign: 'center', cursor: 'pointer', background: parsedContent ? 'rgba(16,185,129,0.04)' : '#FAFAFA', transition: 'all 150ms ease' }}
                  >
                    {parsing ? (
                      <div style={{ color: C.primary, fontSize: '14px' }}>
                        <div style={{ fontSize: '28px', marginBottom: '8px' }}>⏳</div>
                        正在解析文档内容...
                      </div>
                    ) : parsedContent ? (
                      <div style={{ color: C.success }}>
                        <div style={{ fontSize: '28px', marginBottom: '8px' }}>✅</div>
                        <div style={{ fontSize: '14px', fontWeight: 600 }}>{fileName}</div>
                        <div style={{ fontSize: '12px', marginTop: '4px' }}>
                          已提取约 {parsedContent.replace(/\s/g, '').length} 字 · 点击重新上传
                        </div>
                      </div>
                    ) : (
                      <div style={{ color: C.textMuted }}>
                        <div style={{ fontSize: '36px', marginBottom: '10px' }}>
                          {sourceType === 'docx' ? '📝' : '📄'}
                        </div>
                        <div style={{ fontSize: '14px', fontWeight: 500, color: C.text, marginBottom: '4px' }}>
                          点击选择{sourceType === 'docx' ? 'Word文档' : 'PDF文件'}
                        </div>
                        <div style={{ fontSize: '12px', lineHeight: 1.6 }}>
                          {sourceType === 'pdf'
                            ? '仅支持文字型PDF，扫描件请改用粘贴文本方式'
                            : '支持 .docx 格式（Word 2007及以上版本）'
                          }
                        </div>
                      </div>
                    )}
                  </div>

                  {/* 隐藏的文件input */}
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept={sourceType === 'docx' ? '.docx' : '.pdf'}
                    onChange={handleFileChange}
                    style={{ display: 'none' }}
                  />

                  {/* 解析错误提示 */}
                  {parseError && (
                    <div style={{ marginTop: '10px', padding: '12px 14px', borderRadius: '8px', background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.2)', fontSize: '13px', color: C.danger, lineHeight: 1.6 }}>
                      ⚠️ {parseError}
                    </div>
                  )}

                  {/* 解析成功预览 */}
                  {parsedContent && (
                    <div style={{ marginTop: '10px', padding: '12px 14px', borderRadius: '8px', background: '#F9FAFB', border: `1px solid ${C.border}`, fontSize: '12px', color: C.textSec, maxHeight: '120px', overflowY: 'auto', lineHeight: 1.7, whiteSpace: 'pre-wrap' }}>
                      {parsedContent.slice(0, 400)}{parsedContent.length > 400 ? '...' : ''}
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* ===== 第二步：关联课本 ===== */}
          {step === 2 && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '16px' }}>

              {/* 内容就绪提示 */}
              <div style={{ padding: '14px 18px', borderRadius: '10px', background: 'rgba(16,185,129,0.06)', border: '1px solid rgba(16,185,129,0.2)', fontSize: '13px', color: '#166534', lineHeight: 1.6 }}>
                ✅ 教案内容已就绪（约 {effectiveContent.replace(/\s/g, '').length} 字）<br />
                <span style={{ color: C.textSec }}>关联课本图片后AI评审更精准。此步骤可选，直接点击「开始导入」也可。</span>
              </div>

              {/* 课本图片选择 */}
              <div>
                <div style={{ fontSize: '14px', fontWeight: 600, color: C.text, marginBottom: '12px' }}>
                  📷 关联课本图片
                  <span style={{ fontSize: '12px', fontWeight: 400, color: C.textMuted, marginLeft: '8px' }}>
                    已选 {selectedTextbookIds.size} 张
                  </span>
                </div>

                {textbooksLoading ? (
                  <div style={{ textAlign: 'center', padding: '24px', color: C.textMuted, fontSize: '13px' }}>
                    加载课本图片中...
                  </div>
                ) : textbooks.length === 0 ? (
                  <div style={{ padding: '24px', borderRadius: '10px', textAlign: 'center', background: 'rgba(79,123,232,0.04)', border: '1px dashed rgba(79,123,232,0.25)' }}>
                    <div style={{ fontSize: '28px', marginBottom: '8px' }}>📷</div>
                    <div style={{ fontSize: '13px', color: C.text, fontWeight: 500, marginBottom: '4px' }}>暂无可用的课本图片</div>
                    <div style={{ fontSize: '12px', color: C.textMuted, lineHeight: 1.6, marginBottom: '12px' }}>
                      可先导入教案，之后在课本管理页上传图片
                    </div>
                    <button
                      onClick={() => navigate('/lesson-plans/textbooks')}
                      style={{ padding: '6px 16px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '12px', fontWeight: 600, cursor: 'pointer' }}
                    >
                      去上传课本图片
                    </button>
                  </div>
                ) : (
                  <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', maxHeight: '240px', overflowY: 'auto' }}>
                    {textbooks.map(tb => {
                      const checked = selectedTextbookIds.has(tb.id)
                      return (
                        <label
                          key={tb.id}
                          onClick={() => toggleTextbook(tb.id)}
                          style={{ display: 'flex', alignItems: 'center', gap: '8px', padding: '8px 12px', borderRadius: '8px', cursor: 'pointer', background: checked ? 'rgba(79,123,232,0.08)' : '#fff', border: checked ? '1px solid #4F7BE8' : '1px solid #E5E7EB', fontSize: '12px', color: '#1F2937', transition: 'all 150ms ease' }}
                        >
                          <input type="checkbox" checked={checked} readOnly style={{ accentColor: '#4F7BE8', pointerEvents: 'none' }} />
                          <img
                            src={tb.image_url} alt=""
                            style={{ width: '32px', height: '32px', objectFit: 'cover', borderRadius: '4px' }}
                            onError={e => { (e.target as HTMLImageElement).style.display = 'none' }}
                          />
                          <div>
                            <div style={{ maxWidth: '120px', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontWeight: checked ? 600 : 400 }}>
                              {tb.chapter || tb.textbook_name}
                            </div>
                            {tb.has_ocr && (
                              <div style={{ fontSize: '10px', color: C.success, marginTop: '2px' }}>✓ 已识别文字</div>
                            )}
                          </div>
                        </label>
                      )
                    })}
                  </div>
                )}
              </div>

              {/* 提交错误提示 */}
              {submitError && (
                <div style={{ padding: '12px 14px', borderRadius: '8px', background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.2)', fontSize: '13px', color: C.danger }}>
                  ⚠️ {submitError}
                </div>
              )}
            </div>
          )}
        </div>

        {/* ---- 底部按钮 ---- */}
        <div style={{ padding: '16px 28px', borderTop: `1px solid ${C.border}`, display: 'flex', gap: '10px', justifyContent: 'space-between', alignItems: 'center' }}>
          <button
            onClick={step === 1 ? onCancel : () => setStep(1)}
            style={{ padding: '10px 20px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '14px', color: C.textSec, cursor: 'pointer' }}
          >
            {step === 1 ? '取消' : '← 上一步'}
          </button>

          <div style={{ display: 'flex', gap: '10px' }}>
            {/* 第一步：下一步 */}
            {step === 1 && (
              <button
                onClick={() => setStep(2)}
                disabled={!step1Valid}
                style={{ padding: '10px 24px', borderRadius: '8px', border: 'none', background: step1Valid ? C.primary : '#E5E7EB', color: step1Valid ? '#fff' : C.textMuted, fontSize: '14px', fontWeight: 600, cursor: step1Valid ? 'pointer' : 'not-allowed' }}
              >
                下一步：关联课本 →
              </button>
            )}

            {/* 第二步：跳过 + 导入 */}
            {step === 2 && (
              <>
                <button
                  onClick={handleSubmit}
                  disabled={submitting}
                  style={{ padding: '10px 20px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '14px', color: C.textSec, cursor: submitting ? 'not-allowed' : 'pointer' }}
                >
                  跳过，直接导入
                </button>
                <button
                  onClick={handleSubmit}
                  disabled={submitting}
                  style={{ padding: '10px 24px', borderRadius: '8px', border: 'none', background: submitting ? '#E5E7EB' : 'linear-gradient(135deg, #4F7BE8, #818CF8)', color: submitting ? C.textMuted : '#fff', fontSize: '14px', fontWeight: 600, cursor: submitting ? 'not-allowed' : 'pointer', boxShadow: submitting ? 'none' : '0 4px 12px rgba(79,123,232,0.35)' }}
                >
                  {submitting ? '导入中...' : '🚀 开始导入并AI评审'}
                </button>
              </>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

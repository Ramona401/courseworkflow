/**
 * KBUploadForm.tsx — 课标压缩任务创建表单（上传区）
 *
 * PRD 6.2：图片多模态 + 文本粘贴两种输入，不做 PDF。
 * 受控组件：自身只管表单态与本地校验，组装好 KBCreateJobRequest 后经 onCreate 交父页面，
 *   由父页面统一编排「建任务 → 立即订阅 SSE」的时序（后端 800ms 延迟才跑压缩，需前端先连 SSE）。
 *
 * 输入校验（创建前本地拦截）：
 *   - batch_tag 必填（蓝绿切换以此为单位，空则无法入库切换）
 *   - 文本 text_content 与 图片 image_data_uris 至少一项非空（否则无可压缩内容）
 *   - rounds 介于 1-5，默认 3
 *
 * 图片：浏览器内 FileReader 转 data URI（filesToDataURIs），不经后端中转；
 *   仅 JPG/PNG/WEBP，单张 ≤ 8MB（与多模态请求体规模相称），超限本地拒绝并提示。
 */
import { useState, useRef } from 'react'
import { C, Spinner, filesToDataURIs } from './kbConstants'
import { KB_DEFAULT_ROUNDS, type KBCreateJobRequest } from '@/api/kb'

// 课标常见学科（与平台口径一致，课标先行阶段主要是数学/英语，已录入；其余备选）
const SUBJECT_OPTIONS = ['数学', '英语', '语文', '信息科技', '物理', '化学', '生物', '历史', '地理', '道德与法治', '科学']

// 图片本地校验
const IMG_ALLOWED = ['image/jpeg', 'image/png', 'image/webp']
const IMG_MAX_BYTES = 8 * 1024 * 1024 // 8MB

interface KBUploadFormProps {
  /** 创建中（父页面建任务+订阅SSE期间置 true，禁用按钮防重复提交） */
  creating: boolean
  /** 组装好的请求交父页面执行 */
  onCreate: (req: KBCreateJobRequest) => void
  /** 本地校验失败/读图失败时向父页面冒泡提示 */
  onError: (msg: string) => void
}

export function KBUploadForm({ creating, onCreate, onError }: KBUploadFormProps) {
  const [batchTag, setBatchTag] = useState('')
  const [subject, setSubject] = useState('数学')
  const [gradeNum, setGradeNum] = useState<number>(3)
  const [rounds, setRounds] = useState<number>(KB_DEFAULT_ROUNDS)
  const [textContent, setTextContent] = useState('')

  // 图片：保留 File 用于预览展示，提交前才转 data URI
  const [images, setImages] = useState<{ file: File; previewUrl: string }[]>([])
  const [readingImages, setReadingImages] = useState(false)
  const fileInputRef = useRef<HTMLInputElement | null>(null)

  // ---- 选择图片 ----
  const handlePickImages = (e: React.ChangeEvent<HTMLInputElement>) => {
    const picked = Array.from(e.target.files || [])
    e.target.value = '' // 允许重复选择同一文件
    if (picked.length === 0) return

    const accepted: { file: File; previewUrl: string }[] = []
    for (const f of picked) {
      if (!IMG_ALLOWED.includes(f.type)) {
        onError(`不支持的图片格式：${f.name}（仅 JPG/PNG/WEBP）`)
        continue
      }
      if (f.size > IMG_MAX_BYTES) {
        onError(`图片过大：${f.name}（单张上限 8MB）`)
        continue
      }
      accepted.push({ file: f, previewUrl: URL.createObjectURL(f) })
    }
    if (accepted.length > 0) setImages(prev => [...prev, ...accepted])
  }

  // ---- 删除某张图片 ----
  const removeImage = (idx: number) => {
    setImages(prev => {
      const target = prev[idx]
      if (target) URL.revokeObjectURL(target.previewUrl) // 释放预览 URL
      return prev.filter((_, i) => i !== idx)
    })
  }

  // ---- 提交创建 ----
  const handleSubmit = async () => {
    const tag = batchTag.trim()
    if (!tag) { onError('请填写批次标识（batch_tag），蓝绿切换以此为单位'); return }

    const text = textContent.trim()
    if (!text && images.length === 0) {
      onError('请至少粘贴课标文本或上传课标图片（PDF 不支持）')
      return
    }
    if (rounds < 1 || rounds > 5) { onError('压缩轮数需在 1-5 之间'); return }

    // 提交前才把图片转 data URI（耗时操作，给 Spinner 反馈）
    let imageDataURIs: string[] = []
    if (images.length > 0) {
      try {
        setReadingImages(true)
        imageDataURIs = await filesToDataURIs(images.map(i => i.file))
      } catch (err: unknown) {
        setReadingImages(false)
        onError(err instanceof Error ? err.message : '读取图片失败')
        return
      }
      setReadingImages(false)
    }

    const req: KBCreateJobRequest = {
      kind: 'curriculum',
      batch_tag: tag,
      rounds,
      subject,
      grade_num: gradeNum,
      text_content: text || undefined,
      image_data_uris: imageDataURIs.length > 0 ? imageDataURIs : undefined,
    }
    onCreate(req)
  }

  const labelStyle: React.CSSProperties = { fontSize: '13px', fontWeight: 600, color: C.textSec, marginBottom: '6px', display: 'block' }
  const inputStyle: React.CSSProperties = {
    width: '100%', padding: '9px 12px', borderRadius: '8px',
    border: `1px solid ${C.border}`, fontSize: '14px', outline: 'none',
    background: C.white, color: C.text, boxSizing: 'border-box',
  }

  const busy = creating || readingImages

  return (
    <div style={{ background: C.white, borderRadius: '14px', border: `1px solid ${C.border}`, padding: '20px 22px' }}>
      <div style={{ fontSize: '15px', fontWeight: 700, color: C.text, marginBottom: '16px' }}>
        📥 新建课标压缩任务
      </div>

      {/* 第一行：批次 / 学科 / 年级 / 轮数 */}
      <div style={{ display: 'grid', gridTemplateColumns: '1.6fr 1fr 1fr 1fr', gap: '14px', marginBottom: '16px' }}>
        <div>
          <label style={labelStyle}>批次标识 <span style={{ color: C.danger }}>*</span></label>
          <input
            value={batchTag}
            onChange={e => setBatchTag(e.target.value)}
            placeholder="如 math-2022-v1"
            style={inputStyle}
            disabled={busy}
          />
        </div>
        <div>
          <label style={labelStyle}>学科</label>
          <select value={subject} onChange={e => setSubject(e.target.value)} style={inputStyle} disabled={busy}>
            {SUBJECT_OPTIONS.map(s => <option key={s} value={s}>{s}</option>)}
          </select>
        </div>
        <div>
          <label style={labelStyle}>年级</label>
          <select value={gradeNum} onChange={e => setGradeNum(Number(e.target.value))} style={inputStyle} disabled={busy}>
            <option value={0}>学段级（不分年级）</option>
            {Array.from({ length: 12 }, (_, i) => i + 1).map(g => (
              <option key={g} value={g}>{g} 年级</option>
            ))}
          </select>
        </div>
        <div>
          <label style={labelStyle}>压缩轮数</label>
          <select value={rounds} onChange={e => setRounds(Number(e.target.value))} style={inputStyle} disabled={busy}>
            {[1, 2, 3, 4, 5].map(n => (
              <option key={n} value={n}>{n} 轮{n === KB_DEFAULT_ROUNDS ? '（默认）' : ''}</option>
            ))}
          </select>
        </div>
      </div>

      {/* 文本粘贴 */}
      <div style={{ marginBottom: '16px' }}>
        <label style={labelStyle}>课标原文（文本粘贴）</label>
        <textarea
          value={textContent}
          onChange={e => setTextContent(e.target.value)}
          placeholder="粘贴课标原文文本。AI 将通读后识别其中的多个知识点，逐个多轮压缩。"
          style={{ ...inputStyle, minHeight: '120px', resize: 'vertical', lineHeight: 1.6, fontFamily: 'inherit' }}
          disabled={busy}
        />
      </div>

      {/* 图片上传 */}
      <div style={{ marginBottom: '18px' }}>
        <label style={labelStyle}>
          课标原文（图片多模态）
          <span style={{ fontSize: '11px', fontWeight: 400, color: C.textMuted, marginLeft: '8px' }}>
            JPG/PNG/WEBP，单张 ≤ 8MB，图片走多模态天然对齐到页（不支持 PDF）
          </span>
        </label>
        <div style={{ display: 'flex', flexWrap: 'wrap', gap: '10px', alignItems: 'flex-start' }}>
          {images.map((img, idx) => (
            <div key={idx} style={{ position: 'relative', width: '88px', height: '88px', borderRadius: '8px', overflow: 'hidden', border: `1px solid ${C.border}` }}>
              <img src={img.previewUrl} alt={`课标图${idx + 1}`} style={{ width: '100%', height: '100%', objectFit: 'cover' }} />
              {!busy && (
                <button
                  onClick={() => removeImage(idx)}
                  style={{
                    position: 'absolute', top: '2px', right: '2px',
                    width: '20px', height: '20px', borderRadius: '50%', border: 'none',
                    background: 'rgba(0,0,0,0.6)', color: '#fff', fontSize: '12px',
                    cursor: 'pointer', lineHeight: 1, display: 'flex', alignItems: 'center', justifyContent: 'center',
                  }}
                  title="移除"
                >×</button>
              )}
            </div>
          ))}
          {!busy && (
            <button
              onClick={() => fileInputRef.current?.click()}
              style={{
                width: '88px', height: '88px', borderRadius: '8px',
                border: `1.5px dashed ${C.border}`, background: C.bg,
                color: C.textMuted, fontSize: '12px', cursor: 'pointer',
                display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center', gap: '4px',
              }}
            >
              <span style={{ fontSize: '22px' }}>＋</span>
              添加图片
            </button>
          )}
          <input
            ref={fileInputRef}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            multiple
            onChange={handlePickImages}
            style={{ display: 'none' }}
          />
        </div>
      </div>

      {/* 创建按钮 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
        <button
          onClick={handleSubmit}
          disabled={busy}
          style={{
            padding: '10px 26px', borderRadius: '9px', border: 'none',
            background: busy ? C.textMuted : `linear-gradient(135deg,${C.primary},${C.purple})`,
            color: '#fff', fontSize: '14px', fontWeight: 600,
            cursor: busy ? 'not-allowed' : 'pointer',
            display: 'flex', alignItems: 'center', gap: '8px',
          }}
        >
          {busy && <Spinner size={16} />}
          {readingImages ? '读取图片中...' : creating ? '创建中...' : '🚀 创建并开始压缩'}
        </button>
        <span style={{ fontSize: '12px', color: C.textMuted }}>
          全程默认 opus 模型，逻辑严谨优先；创建后实时显示压缩进度。
        </span>
      </div>
    </div>
  )
}

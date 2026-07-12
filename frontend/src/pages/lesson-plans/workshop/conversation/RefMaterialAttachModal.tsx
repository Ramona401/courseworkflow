/**
 * RefMaterialAttachModal.tsx — 备课参考资料附件弹窗（对话模式）
 *
 * 老师上传 PDF/Word 作为本次备课的参考资料。全流程浏览器端提取、不落库：
 *   选文件(.docx/文字版.pdf，≤10MB) → extractDocFile 提取文字
 *     → 短文档(<3000字)：原文直接作为注入文本
 *     → 长文档(≥3000字)：调后端 compressRefMaterial 压成结构化要点作为注入文本
 *   → onAttached({ text, fileName }) 回传给页面，页面持有并每轮 chat 携带 ref_material。
 *
 * 会话级、用完即走：附件不落库，老师移除或退出备课即失（与设计"不落库"一致）。
 *
 * 状态机：idle（待选） → parsing（提取中） → compressing（压缩中） → done/error。
 */
import { useState, useRef } from 'react'
import { C } from '../components/workshopConstants'
import { extractDocFile, REF_COMPRESS_THRESHOLD } from '../utils/docExtract'
import { compressRefMaterial } from '@/api/lesson-plans-ref'

interface RefMaterialAttachModalProps {
  /** 学科（传给压缩端点聚焦，可空） */
  subject?: string
  /** 年级（传给压缩端点聚焦，可空） */
  grade?: string
  /** 附件就绪回调：text=最终注入文本，fileName=文件名（用于提示条显示） */
  onAttached: (payload: { text: string; fileName: string }) => void
  /** 关闭弹窗 */
  onCancel: () => void
}

type Stage = 'idle' | 'parsing' | 'compressing' | 'error'

export default function RefMaterialAttachModal({ subject, grade, onAttached, onCancel }: RefMaterialAttachModalProps) {
  const [stage, setStage] = useState<Stage>('idle')
  const [errorMsg, setErrorMsg] = useState('')
  const [fileName, setFileName] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const busy = stage === 'parsing' || stage === 'compressing'

  const handleFileChange = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    setFileName(file.name)
    setErrorMsg('')
    setStage('parsing')
    try {
      // 1) 浏览器端提取文字
      const { text, charCount } = await extractDocFile(file)

      // 2) 短文档直接注入；长文档调后端压缩
      let injectText = text
      if (charCount >= REF_COMPRESS_THRESHOLD) {
        setStage('compressing')
        const resp = await compressRefMaterial({
          content: text,
          file_name: file.name,
          subject: subject || undefined,
          grade: grade || undefined,
        })
        injectText = (resp.compressed || '').trim() || text // 压缩空结果兜底用原文
      }

      // 3) 回传给页面
      onAttached({ text: injectText, fileName: file.name })
    } catch (err) {
      const msg = err instanceof Error ? err.message : '解析失败，请重试或改用粘贴'
      setErrorMsg(msg)
      setStage('error')
    } finally {
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  return (
    <div
      style={{ position: 'fixed', inset: 0, zIndex: 10000, background: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', padding: '20px' }}
      onClick={busy ? undefined : onCancel}
    >
      <div
        onClick={e => e.stopPropagation()}
        style={{ background: '#fff', borderRadius: '16px', width: '100%', maxWidth: '520px', boxShadow: '0 24px 64px rgba(0,0,0,0.18)', display: 'flex', flexDirection: 'column', overflow: 'hidden' }}
      >
        {/* 标题栏 */}
        <div style={{ padding: '20px 28px 16px', borderBottom: `1px solid ${C.border}` }}>
          <h2 style={{ margin: 0, fontSize: '18px', fontWeight: 700, color: C.text }}>📎 上传参考资料</h2>
          <p style={{ margin: '4px 0 0', fontSize: '13px', color: C.textSec, lineHeight: 1.6 }}>
            上传 PDF 或 Word，AI 会在本次备课时参考其中的知识点、教学要求与重点。
            <br />资料仅在本次对话内有效，不会保存到系统。
          </p>
        </div>

        {/* 内容区 */}
        <div style={{ padding: '24px 28px' }}>
          {/* 上传区（提取/压缩中禁用点击） */}
          <div
            onClick={() => { if (!busy) fileInputRef.current?.click() }}
            style={{ border: `2px dashed ${stage === 'error' ? C.danger : C.border}`, borderRadius: '10px', padding: '32px', textAlign: 'center', cursor: busy ? 'default' : 'pointer', background: '#FAFAFA', transition: 'all 150ms ease' }}
          >
            {stage === 'parsing' ? (
              <div style={{ color: C.primary, fontSize: '14px' }}>
                <div style={{ fontSize: '28px', marginBottom: '8px' }}>⏳</div>
                正在提取「{fileName}」的文字…
              </div>
            ) : stage === 'compressing' ? (
              <div style={{ color: C.primary, fontSize: '14px' }}>
                <div style={{ fontSize: '28px', marginBottom: '8px' }}>🧠</div>
                内容较长，AI 正在提炼要点…
                <div style={{ fontSize: '12px', color: C.textMuted, marginTop: '6px' }}>约需十几秒，请稍候</div>
              </div>
            ) : (
              <div style={{ color: C.textMuted }}>
                <div style={{ fontSize: '36px', marginBottom: '10px' }}>📄</div>
                <div style={{ fontSize: '14px', fontWeight: 500, color: C.text, marginBottom: '4px' }}>点击选择文件</div>
                <div style={{ fontSize: '12px', lineHeight: 1.6 }}>
                  支持 .docx 和文字版 .pdf，单个文件不超过 10MB<br />
                  扫描件无法提取文字，请改用课本图片或粘贴
                </div>
              </div>
            )}
          </div>

          <input
            ref={fileInputRef}
            type="file"
            accept=".docx,.pdf"
            onChange={handleFileChange}
            style={{ display: 'none' }}
          />

          {/* 错误提示 */}
          {stage === 'error' && errorMsg && (
            <div style={{ marginTop: '12px', padding: '12px 14px', borderRadius: '8px', background: 'rgba(239,68,68,0.06)', border: '1px solid rgba(239,68,68,0.2)', fontSize: '13px', color: C.danger, lineHeight: 1.6 }}>
              ⚠️ {errorMsg}
            </div>
          )}
        </div>

        {/* 底部按钮 */}
        <div style={{ padding: '14px 28px', borderTop: `1px solid ${C.border}`, display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
          <button
            onClick={onCancel}
            disabled={busy}
            style={{ padding: '9px 20px', borderRadius: '8px', border: `1px solid ${C.border}`, background: 'transparent', fontSize: '14px', color: C.textSec, cursor: busy ? 'not-allowed' : 'pointer', opacity: busy ? 0.5 : 1 }}
          >
            {busy ? '处理中…' : '取消'}
          </button>
          {stage === 'error' && (
            <button
              onClick={() => { setStage('idle'); setErrorMsg('') }}
              style={{ padding: '9px 20px', borderRadius: '8px', border: 'none', background: C.primary, color: '#fff', fontSize: '14px', fontWeight: 600, cursor: 'pointer' }}
            >
              重新选择
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

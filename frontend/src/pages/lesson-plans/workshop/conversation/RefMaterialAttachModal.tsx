/**
 * RefMaterialAttachModal.tsx — 备课参考资料附件弹窗
 *
 * 支持 DOCX、文字型 PDF、扫描型和混合型 PDF。
 * 扫描页逐页渲染，以最多 2 页并发调用后端多模态转录；
 * 原文件与页面图不永久保存、不落库。
 */

import { useRef, useState } from 'react'
import { C } from '../components/workshopConstants'
import {
  extractDocFile,
  REF_COMPRESS_THRESHOLD,
  type ExtractedDocumentPage,
} from '../utils/docExtract'
import {
  compressRefMaterial,
  transcribeRefMaterialPage,
} from '@/api/lesson-plans-ref'

interface RefMaterialAttachModalProps {
  subject?: string
  grade?: string
  onAttached: (payload: { text: string; fileName: string }) => void
  onCancel: () => void
}

type Stage = 'idle' | 'parsing' | 'transcribing' | 'compressing' | 'error'
const VISION_CONCURRENCY = 2

export default function RefMaterialAttachModal({
  subject,
  grade,
  onAttached,
  onCancel,
}: RefMaterialAttachModalProps) {
  const [stage, setStage] = useState<Stage>('idle')
  const [errorMsg, setErrorMsg] = useState('')
  const [fileName, setFileName] = useState('')
  const [progressText, setProgressText] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  const busy =
    stage === 'parsing' ||
    stage === 'transcribing' ||
    stage === 'compressing'

  const transcribeScanningPages = async (
    pages: ExtractedDocumentPage[],
    currentFileName: string,
    totalPages: number,
  ): Promise<string> => {
    const pageText = new Map<number, string>()
    const scanPages = pages.filter(
      page => !page.text.trim() && Boolean(page.imageDataUri),
    )

    pages.forEach(page => {
      if (page.text.trim()) pageText.set(page.pageNumber, page.text.trim())
    })

    if (scanPages.length > 0) {
      setStage('transcribing')
      let cursor = 0
      let completed = 0

      const worker = async () => {
        while (true) {
          const index = cursor++
          if (index >= scanPages.length) return

          const page = scanPages[index]
          if (!page.imageDataUri) {
            throw new Error(`第 ${page.pageNumber} 页没有可识别的页面图`)
          }

          setProgressText(
            `正在识别扫描页 ${completed + 1}/${scanPages.length}（PDF 第 ${page.pageNumber} 页）…`,
          )

          try {
            const response = await transcribeRefMaterialPage({
              image_data_uri: page.imageDataUri,
              file_name: currentFileName,
              page_number: page.pageNumber,
              total_pages: totalPages,
              subject: subject || undefined,
              grade: grade || undefined,
            })
            const text = (response.text || '').trim()
            if (!text) throw new Error('识别结果为空')

            pageText.set(page.pageNumber, text)
            page.imageDataUri = undefined
            completed++
            setProgressText(
              `扫描页已识别 ${completed}/${scanPages.length}，正在继续…`,
            )
          } catch (error) {
            const detail =
              error instanceof Error ? error.message : '未知错误'
            throw new Error(`PDF 第 ${page.pageNumber} 页识别失败：${detail}`)
          }
        }
      }

      await Promise.all(
        Array.from(
          { length: Math.min(VISION_CONCURRENCY, scanPages.length) },
          () => worker(),
        ),
      )
    }

    return pages
      .map(page => {
        const text = pageText.get(page.pageNumber)
        return text ? `【第${page.pageNumber}页】\n${text}` : ''
      })
      .filter(Boolean)
      .join('\n\n')
  }

  const handleFileChange = async (
    event: React.ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0]
    if (!file) return

    setFileName(file.name)
    setErrorMsg('')
    setProgressText('正在读取文件…')
    setStage('parsing')

    try {
      const extracted = await extractDocFile(file, progress => {
        setProgressText(progress.message)
      })

      let injectText = extracted.text.trim()
      if (extracted.pages?.length) {
        injectText = await transcribeScanningPages(
          extracted.pages,
          file.name,
          extracted.totalPages || extracted.pages.length,
        )
      }

      injectText = injectText.trim()
      if (!injectText) {
        throw new Error('没有获得可用文字，请检查文件内容后重试')
      }

      const charCount = injectText.replace(/\s/g, '').length
      if (charCount >= REF_COMPRESS_THRESHOLD) {
        setStage('compressing')
        setProgressText('内容较长，正在提炼并保留关键事实与页码…')
        const response = await compressRefMaterial({
          content: injectText,
          file_name: file.name,
          subject: subject || undefined,
          grade: grade || undefined,
        })
        injectText = (response.compressed || '').trim() || injectText
      }

      onAttached({ text: injectText, fileName: file.name })
    } catch (error) {
      setErrorMsg(
        error instanceof Error ? error.message : '解析失败，请重试或改用粘贴',
      )
      setProgressText('')
      setStage('error')
    } finally {
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 10000,
        background: 'rgba(0,0,0,0.5)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: '20px',
      }}
      onClick={busy ? undefined : onCancel}
    >
      <div
        onClick={event => event.stopPropagation()}
        style={{
          background: '#fff',
          borderRadius: '16px',
          width: '100%',
          maxWidth: '540px',
          boxShadow: '0 24px 64px rgba(0,0,0,0.18)',
          display: 'flex',
          flexDirection: 'column',
          overflow: 'hidden',
        }}
      >
        <div
          style={{
            padding: '20px 28px 16px',
            borderBottom: `1px solid ${C.border}`,
          }}
        >
          <h2
            style={{
              margin: 0,
              fontSize: '18px',
              fontWeight: 700,
              color: C.text,
            }}
          >
            📎 上传参考资料
          </h2>
          <p
            style={{
              margin: '4px 0 0',
              fontSize: '13px',
              color: C.textSec,
              lineHeight: 1.6,
            }}
          >
            支持 Word、文字型 PDF 和扫描型 PDF。扫描页会逐页识别，
            不再需要转成长图片。
            <br />
            资料只在本次对话内有效，不会永久保存。
          </p>
        </div>

        <div style={{ padding: '24px 28px' }}>
          <div
            onClick={() => {
              if (!busy) fileInputRef.current?.click()
            }}
            style={{
              border: `2px dashed ${
                stage === 'error' ? C.danger : C.border
              }`,
              borderRadius: '10px',
              padding: '32px',
              textAlign: 'center',
              cursor: busy ? 'default' : 'pointer',
              background: '#FAFAFA',
              transition: 'all 150ms ease',
            }}
          >
            {stage === 'parsing' ? (
              <Status
                icon="⏳"
                text={progressText || `正在读取「${fileName}」…`}
              />
            ) : stage === 'transcribing' ? (
              <Status
                icon="👁️"
                text={progressText || '正在逐页识别扫描 PDF…'}
                subText="每页独立识别，避免多页长图造成内容串页"
              />
            ) : stage === 'compressing' ? (
              <Status
                icon="🧠"
                text={progressText || '内容较长，正在提炼要点…'}
              />
            ) : (
              <div style={{ color: C.textMuted }}>
                <div style={{ fontSize: '36px', marginBottom: '10px' }}>
                  📄
                </div>
                <div
                  style={{
                    fontSize: '14px',
                    fontWeight: 500,
                    color: C.text,
                    marginBottom: '4px',
                  }}
                >
                  点击选择文件
                </div>
                <div style={{ fontSize: '12px', lineHeight: 1.6 }}>
                  支持 .docx、文字型/扫描型 .pdf
                  <br />
                  单文件≤10MB · PDF最多12页
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

          {stage === 'error' && errorMsg && (
            <div
              style={{
                marginTop: '12px',
                padding: '12px 14px',
                borderRadius: '8px',
                background: 'rgba(239,68,68,0.06)',
                border: '1px solid rgba(239,68,68,0.2)',
                fontSize: '13px',
                color: C.danger,
                lineHeight: 1.6,
              }}
            >
              ⚠️ {errorMsg}
            </div>
          )}
        </div>

        <div
          style={{
            padding: '14px 28px',
            borderTop: `1px solid ${C.border}`,
            display: 'flex',
            justifyContent: 'flex-end',
            gap: '10px',
          }}
        >
          <button
            onClick={onCancel}
            disabled={busy}
            style={{
              padding: '9px 20px',
              borderRadius: '8px',
              border: `1px solid ${C.border}`,
              background: 'transparent',
              fontSize: '14px',
              color: C.textSec,
              cursor: busy ? 'not-allowed' : 'pointer',
              opacity: busy ? 0.5 : 1,
            }}
          >
            {busy ? '处理中…' : '取消'}
          </button>

          {stage === 'error' && (
            <button
              onClick={() => {
                setStage('idle')
                setErrorMsg('')
                setProgressText('')
              }}
              style={{
                padding: '9px 20px',
                borderRadius: '8px',
                border: 'none',
                background: C.primary,
                color: '#fff',
                fontSize: '14px',
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              重新选择
            </button>
          )}
        </div>
      </div>
    </div>
  )
}

function Status({
  icon,
  text,
  subText,
}: {
  icon: string
  text: string
  subText?: string
}) {
  return (
    <div style={{ color: C.primary, fontSize: '14px' }}>
      <div style={{ fontSize: '28px', marginBottom: '8px' }}>{icon}</div>
      {text}
      {subText && (
        <div
          style={{
            fontSize: '12px',
            color: C.textMuted,
            marginTop: '6px',
          }}
        >
          {subText}
        </div>
      )}
    </div>
  )
}

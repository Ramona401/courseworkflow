/**
 * LessonPlanImportSourceSection.tsx
 *
 * 已有教案导入弹窗中的“内容来源”独立区域。
 *
 * 职责：
 *   - 粘贴文本；
 *   - 普通Word浏览器纯文本提取；
 *   - PDF浏览器文字层提取；
 *   - 原格式Word后端安全预解析；
 *   - 保留文件名、语义正文、Word会话ID和结构化预览。
 *
 * 状态由父弹窗持有，因此K12进入“关联课本”步骤再返回时，
 * 已选择的来源和已解析的Word会话不会丢失。
 */

import {
  useRef,
  useState,
  type ChangeEvent,
} from 'react'
import {
  MAX_REF_FILE_SIZE,
  parseDocxFile,
  parsePdfFile,
} from '../utils/docExtract'
import {
  MAX_WORD_FIDELITY_FILE_SIZE,
  previewLessonPlanWordImport,
  type LessonPlanImportSourceType,
  type LessonPlanWordImportPreview,
} from '@/api/lesson-plan-word-import'
import LessonPlanWordImportPreviewCard from './LessonPlanWordImportPreviewCard'
import { C } from './workshopConstants'

export interface LessonPlanImportSourceSelection {
  sourceType: LessonPlanImportSourceType
  content: string
  fileName: string
  ready: boolean
  wordImportSessionID?: string
  wordPreview?: LessonPlanWordImportPreview | null
}

interface Props {
  lessonPlanLabel: string
  value: LessonPlanImportSourceSelection
  onChange: (
    next: LessonPlanImportSourceSelection,
  ) => void
}

const SOURCE_OPTIONS: Array<{
  key: LessonPlanImportSourceType
  label: string
}> = [
  {
    key: 'paste',
    label: '📋 粘贴文本',
  },
  {
    key: 'docx',
    label: '📝 Word文档',
  },
  {
    key: 'docx_fidelity',
    label: '🧬 保留Word格式',
  },
  {
    key: 'pdf',
    label: '📄 PDF文件',
  },
]

export default function LessonPlanImportSourceSection({
  lessonPlanLabel,
  value,
  onChange,
}: Props) {
  const [parsing, setParsing] =
    useState(false)

  const [parseError, setParseError] =
    useState('')

  const fileInputRef =
    useRef<HTMLInputElement>(null)

  /**
   * 防止较早的异步解析结果覆盖老师后来重新选择的文件。
   */
  const requestIDRef = useRef(0)

  const changeSourceType = (
    sourceType: LessonPlanImportSourceType,
  ) => {
    requestIDRef.current += 1
    setParsing(false)
    setParseError('')

    onChange({
      sourceType,
      content: '',
      fileName: '',
      ready: sourceType === 'paste',
      wordImportSessionID: undefined,
      wordPreview: null,
    })
  }

  const handlePasteChange = (
    content: string,
  ) => {
    onChange({
      ...value,
      content,
      ready: true,
      wordImportSessionID: undefined,
      wordPreview: null,
    })
  }

  const handleFileChange = async (
    event: ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0]

    if (!file) {
      return
    }

    const requestID =
      requestIDRef.current + 1

    requestIDRef.current = requestID

    setParsing(true)
    setParseError('')

    onChange({
      sourceType: value.sourceType,
      content: '',
      fileName: file.name,
      ready: false,
      wordImportSessionID: undefined,
      wordPreview: null,
    })

    try {
      if (
        value.sourceType ===
        'docx_fidelity'
      ) {
        if (
          file.size >
          MAX_WORD_FIDELITY_FILE_SIZE
        ) {
          throw new Error(
            'Word文档超过30MB，请压缩图片或拆分后重新上传',
          )
        }

        const preview =
          await previewLessonPlanWordImport(
            file,
          )

        if (
          requestIDRef.current !==
          requestID
        ) {
          return
        }

        const semanticMarkdown =
          preview.semantic_markdown.trim()

        onChange({
          sourceType: 'docx_fidelity',
          content: semanticMarkdown,
          fileName: file.name,
          ready: Boolean(
            preview.can_confirm &&
            preview.session_id &&
            semanticMarkdown,
          ),
          wordImportSessionID:
            preview.session_id,
          wordPreview: preview,
        })

        if (!preview.can_confirm) {
          setParseError(
            '文档已完成预览，但当前会话暂不能确认，请重新上传',
          )
        }

        return
      }

      if (file.size > MAX_REF_FILE_SIZE) {
        throw new Error(
          '普通文档解析最大支持10MB；较大的Word文档请使用“保留Word格式”',
        )
      }

      const text =
        value.sourceType === 'docx'
          ? await parseDocxFile(file)
          : await parsePdfFile(file)

      if (
        requestIDRef.current !==
        requestID
      ) {
        return
      }

      const normalizedText = text.trim()

      if (!normalizedText) {
        throw new Error(
          value.sourceType === 'pdf'
            ? '该PDF为扫描件或没有可提取文字，请改用粘贴方式'
            : 'Word内容为空或无法提取，请改用粘贴方式',
        )
      }

      onChange({
        sourceType: value.sourceType,
        content: normalizedText,
        fileName: file.name,
        ready: true,
        wordImportSessionID: undefined,
        wordPreview: null,
      })
    } catch (error) {
      if (
        requestIDRef.current !==
        requestID
      ) {
        return
      }

      const message =
        error instanceof Error
          ? error.message.trim()
          : ''

      onChange({
        sourceType: value.sourceType,
        content: '',
        fileName: file.name,
        ready: false,
        wordImportSessionID: undefined,
        wordPreview: null,
      })

      setParseError(
        message ||
          '文件解析失败，请检查文件格式后重试',
      )
    } finally {
      if (
        requestIDRef.current ===
        requestID
      ) {
        setParsing(false)
      }

      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  const selectedFileTypeLabel =
    value.sourceType === 'pdf'
      ? 'PDF文件'
      : value.sourceType ===
          'docx_fidelity'
        ? '需要保留原格式的Word文档'
        : 'Word文档'

  const parsingLabel =
    value.sourceType ===
    'docx_fidelity'
      ? '⏳ 正在由服务器安全解析Word结构...'
      : '⏳ 正在解析文档...'

  const parsedFileSummary =
    value.sourceType ===
      'docx_fidelity' &&
    value.wordPreview
      ? `✅ ${value.fileName} · 已识别${value.wordPreview.metrics?.table_count || 0}个表格、${value.wordPreview.metrics?.formula_count || 0}个公式`
      : `✅ ${value.fileName} · 已提取${value.content.replace(/\s/g, '').length}字`

  return (
    <>
      <div>
        <label style={{
          display: 'block',
          fontSize: '13px',
          fontWeight: 600,
          marginBottom: '8px',
        }}>
          {lessonPlanLabel}来源
        </label>

        <div style={{
          display: 'flex',
          flexWrap: 'wrap',
          gap: '8px',
        }}>
          {SOURCE_OPTIONS.map(item => (
            <button
              key={item.key}
              type="button"
              onClick={() =>
                changeSourceType(item.key)
              }
              style={{
                flex: '1 1 140px',
                minWidth: 0,
                padding: '10px',
                borderRadius: '9px',
                border: `1.5px solid ${
                  value.sourceType === item.key
                    ? C.primary
                    : C.border
                }`,
                background:
                  value.sourceType === item.key
                    ? C.primaryLight
                    : '#FFFFFF',
                color: C.text,
                cursor: 'pointer',
                fontSize: '13px',
              }}
            >
              {item.label}
            </button>
          ))}
        </div>

        {value.sourceType ===
          'docx_fidelity' && (
          <div style={{
            marginTop: '8px',
            padding: '9px 11px',
            borderRadius: '8px',
            background: '#EEF4FF',
            color: '#365FB8',
            fontSize: '11px',
            lineHeight: 1.65,
          }}>
            后端会私有保存原DOCX，并保留表格、合并单元格、
            图片关系、上下标和公式对象。AI评审和课件生成仍读取同步的语义正文。
          </div>
        )}
      </div>

      {value.sourceType === 'paste' ? (
        <div>
          <textarea
            value={value.content}
            onChange={event =>
              handlePasteChange(
                event.target.value,
              )
            }
            rows={12}
            placeholder={
              `将已有${lessonPlanLabel}内容粘贴到这里，至少50字`
            }
            style={{
              width: '100%',
              boxSizing: 'border-box',
              padding: '12px 14px',
              borderRadius: '8px',
              border: `1px solid ${C.border}`,
              fontSize: '13px',
              lineHeight: 1.8,
              resize: 'vertical',
            }}
          />

          <div style={{
            marginTop: '4px',
            textAlign: 'right',
            color: C.textMuted,
            fontSize: '11px',
          }}>
            已输入
            {' '}
            {
              value.content
                .replace(/\s/g, '')
                .length
            }
            {' '}
            字
          </div>
        </div>
      ) : (
        <div>
          <div
            onClick={() =>
              fileInputRef.current?.click()
            }
            style={{
              border: `2px dashed ${
                value.content
                  ? C.success
                  : parseError
                    ? C.danger
                    : C.border
              }`,
              borderRadius: '10px',
              padding: '28px',
              textAlign: 'center',
              cursor: parsing
                ? 'wait'
                : 'pointer',
              background: '#FAFAFA',
              color: C.text,
              fontSize: '13px',
            }}
          >
            {parsing
              ? parsingLabel
              : value.content
                ? parsedFileSummary
                : `点击选择${selectedFileTypeLabel}`}
          </div>

          <input
            ref={fileInputRef}
            type="file"
            accept={
              value.sourceType === 'pdf'
                ? '.pdf'
                : '.docx'
            }
            onChange={handleFileChange}
            style={{ display: 'none' }}
          />

          {parseError && (
            <div style={{
              marginTop: '8px',
              color: C.danger,
              fontSize: '12px',
              lineHeight: 1.6,
            }}>
              ⚠️ {parseError}
            </div>
          )}

          {value.sourceType ===
            'docx_fidelity' &&
          value.wordPreview ? (
            <LessonPlanWordImportPreviewCard
              preview={value.wordPreview}
            />
          ) : value.content ? (
            <div style={{
              marginTop: '8px',
              padding: '10px 12px',
              borderRadius: '8px',
              background: '#F9FAFB',
              maxHeight: '110px',
              overflowY: 'auto',
              fontSize: '12px',
              lineHeight: 1.7,
              whiteSpace: 'pre-wrap',
            }}>
              {value.content.slice(0, 400)}
            </div>
          ) : null}
        </div>
      )}
    </>
  )
}

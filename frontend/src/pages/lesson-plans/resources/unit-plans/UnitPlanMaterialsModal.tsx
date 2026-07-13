/**
 * UnitPlanMaterialsModal.tsx — 大单元方案参考资料管理弹窗
 *
 * 能力：
 * 1. 查看当前单元方案已经保存的资料；
 * 2. 创建者上传docx或文字型PDF；
 * 3. 浏览器端提取文字，不上传原始文件；
 * 4. 短文直接保存，长文先调用现有压缩服务；
 * 5. 创建者可以软删除资料；
 * 6. 非创建者只能查看资料名称、类型和处理信息。
 */
import {
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'
import type {
  ChangeEvent,
  CSSProperties,
  MouseEvent,
} from 'react'

import {
  createUnitPlanMaterial,
  deleteUnitPlanMaterial,
  getUnitPlanMaterials,
} from '@/api/unit-plan-materials'
import type {
  UnitPlanMaterialListItem,
  UnitPlanMaterialType,
} from '@/api/unit-plan-materials'
import { compressRefMaterial } from '@/api/lesson-plans-ref'
import {
  extractDocFile,
  REF_COMPRESS_THRESHOLD,
} from '@/pages/lesson-plans/workshop/utils/docExtract'

const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  border: '#E5E7EB',
  text: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  danger: '#DC2626',
  success: '#059669',
  white: '#FFFFFF',
}

const MAX_SAVED_CONTENT_LENGTH = 120000

const MATERIAL_TYPES: Array<{
  value: UnitPlanMaterialType
  label: string
  icon: string
}> = [
  {
    value: 'textbook',
    label: '教材或课本',
    icon: '📘',
  },
  {
    value: 'teacher_guide',
    label: '教师用书',
    icon: '📗',
  },
  {
    value: 'previous_unit_plan',
    label: '既有大单元方案',
    icon: '🗂️',
  },
  {
    value: 'teaching_requirement',
    label: '学校或区域教研要求',
    icon: '📋',
  },
  {
    value: 'excellent_case',
    label: '优秀课例',
    icon: '🌟',
  },
  {
    value: 'other',
    label: '其他资料',
    icon: '📄',
  },
]

type ProcessingStage =
  | 'idle'
  | 'parsing'
  | 'compressing'
  | 'saving'

interface UnitPlanMaterialsModalProps {
  unitPlanId: string
  subject: string
  grade: string
  onCancel: () => void
}

const buttonPrimary: CSSProperties = {
  border: 'none',
  borderRadius: 8,
  background: C.primary,
  color: C.white,
  fontSize: 13,
  fontWeight: 600,
  padding: '8px 14px',
  cursor: 'pointer',
}

const buttonGhost: CSSProperties = {
  border: '1px solid ' + C.border,
  borderRadius: 8,
  background: C.white,
  color: C.textSecondary,
  fontSize: 13,
  padding: '8px 14px',
  cursor: 'pointer',
}

function materialTypeMeta(type: UnitPlanMaterialType) {
  return (
    MATERIAL_TYPES.find((item) => item.value === type) ||
    MATERIAL_TYPES[MATERIAL_TYPES.length - 1]
  )
}

function formatLength(length: number) {
  if (!Number.isFinite(length) || length <= 0) {
    return '0字'
  }

  if (length >= 10000) {
    return `${(length / 10000).toFixed(1)}万字`
  }

  return `${length}字`
}

function formatDate(value: string) {
  if (!value) return ''

  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''

  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  })
}

export default function UnitPlanMaterialsModal({
  unitPlanId,
  subject,
  grade,
  onCancel,
}: UnitPlanMaterialsModalProps) {
  const [materials, setMaterials] = useState<
    UnitPlanMaterialListItem[]
  >([])
  const [canManage, setCanManage] = useState(false)
  const [loading, setLoading] = useState(true)

  const [materialType, setMaterialType] =
    useState<UnitPlanMaterialType>('teacher_guide')

  const [processingStage, setProcessingStage] =
    useState<ProcessingStage>('idle')

  const [processingFileName, setProcessingFileName] =
    useState('')

  const [errorMessage, setErrorMessage] = useState('')
  const [successMessage, setSuccessMessage] = useState('')

  const fileInputRef = useRef<HTMLInputElement | null>(null)

  const busy = processingStage !== 'idle'

  const loadMaterials = useCallback(async () => {
    setLoading(true)
    setErrorMessage('')

    try {
      const response = await getUnitPlanMaterials(unitPlanId)
      setMaterials(response.materials || [])
      setCanManage(response.can_manage)
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : '加载参考资料失败',
      )
    } finally {
      setLoading(false)
    }
  }, [unitPlanId])

  useEffect(() => {
    loadMaterials()
  }, [loadMaterials])

  const resetMessages = () => {
    setErrorMessage('')
    setSuccessMessage('')
  }

  const handleFileChange = async (
    event: ChangeEvent<HTMLInputElement>,
  ) => {
    const file = event.target.files?.[0]

    if (!file) return

    resetMessages()
    setProcessingFileName(file.name)
    setProcessingStage('parsing')

    try {
      const extracted = await extractDocFile(file)

      if (extracted.charCount > MAX_SAVED_CONTENT_LENGTH) {
        throw new Error(
          '提取文字超过12万字，请按章节或单元拆分后上传',
        )
      }

      let summaryText = ''

      if (extracted.charCount >= REF_COMPRESS_THRESHOLD) {
        setProcessingStage('compressing')

        const compressed = await compressRefMaterial({
          content: extracted.text,
          file_name: file.name,
          subject: subject || undefined,
          grade: grade || undefined,
        })

        summaryText = (compressed.compressed || '').trim()

        if (!summaryText) {
          throw new Error('资料提炼结果为空，请重新上传')
        }
      }

      setProcessingStage('saving')

      await createUnitPlanMaterial(unitPlanId, {
        material_type: materialType,
        file_name: file.name,
        content_text: extracted.text,
        summary_text: summaryText,
      })

      setSuccessMessage(
        summaryText
          ? `「${file.name}」已保存，并完成长文要点提炼`
          : `「${file.name}」已保存`,
      )

      await loadMaterials()
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : '参考资料处理失败',
      )
    } finally {
      setProcessingStage('idle')
      setProcessingFileName('')

      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  const handleDelete = async (
    material: UnitPlanMaterialListItem,
    event: MouseEvent<HTMLButtonElement>,
  ) => {
    event.stopPropagation()

    const confirmed = window.confirm(
      `确认删除参考资料「${material.file_name}」？`,
    )

    if (!confirmed) return

    resetMessages()

    try {
      await deleteUnitPlanMaterial(
        unitPlanId,
        material.id,
      )

      setSuccessMessage('参考资料已删除')
      await loadMaterials()
    } catch (error) {
      setErrorMessage(
        error instanceof Error
          ? error.message
          : '删除参考资料失败',
      )
    }
  }

  const openFilePicker = () => {
    if (!busy && canManage) {
      fileInputRef.current?.click()
    }
  }

  const processingText = (() => {
    switch (processingStage) {
      case 'parsing':
        return `正在提取「${processingFileName}」的文字…`
      case 'compressing':
        return `正在提炼「${processingFileName}」的结构化要点…`
      case 'saving':
        return `正在保存「${processingFileName}」…`
      default:
        return ''
    }
  })()

  return (
    <div
      onClick={busy ? undefined : onCancel}
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 10000,
        background: 'rgba(17,24,39,0.48)',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        padding: 20,
      }}
    >
      <div
        onClick={(event) => event.stopPropagation()}
        style={{
          width: 760,
          maxWidth: '100%',
          maxHeight: '88vh',
          overflow: 'hidden',
          background: C.white,
          borderRadius: 16,
          boxShadow: '0 24px 70px rgba(0,0,0,0.20)',
          display: 'flex',
          flexDirection: 'column',
        }}
      >
        <div
          style={{
            padding: '18px 22px',
            borderBottom: '1px solid ' + C.border,
            display: 'flex',
            alignItems: 'flex-start',
            gap: 14,
          }}
        >
          <div style={{ flex: 1 }}>
            <div
              style={{
                fontSize: 17,
                fontWeight: 700,
                color: C.text,
              }}
            >
              📚 本单元参考资料
            </div>

            <div
              style={{
                marginTop: 5,
                fontSize: 12.5,
                lineHeight: 1.6,
                color: C.textSecondary,
              }}
            >
              Word和文字版PDF在浏览器中提取文字，
              原始文件不会上传到服务器。
              长资料会先提炼为结构化要点，供AI按需参考。
            </div>
          </div>

          <button
            onClick={onCancel}
            disabled={busy}
            style={{
              ...buttonGhost,
              padding: '6px 10px',
              opacity: busy ? 0.5 : 1,
              cursor: busy ? 'not-allowed' : 'pointer',
            }}
          >
            关闭
          </button>
        </div>

        <div
          style={{
            padding: 22,
            overflowY: 'auto',
          }}
        >
          {canManage && (
            <div
              style={{
                border: '1px solid ' + C.border,
                borderRadius: 12,
                padding: 16,
                marginBottom: 18,
                background: '#FAFBFF',
              }}
            >
              <div
                style={{
                  display: 'grid',
                  gridTemplateColumns: '220px 1fr',
                  gap: 12,
                  alignItems: 'end',
                }}
              >
                <div>
                  <label
                    style={{
                      display: 'block',
                      marginBottom: 5,
                      fontSize: 12,
                      fontWeight: 600,
                      color: C.textSecondary,
                    }}
                  >
                    资料类型
                  </label>

                  <select
                    value={materialType}
                    disabled={busy}
                    onChange={(event) =>
                      setMaterialType(
                        event.target.value as UnitPlanMaterialType,
                      )
                    }
                    style={{
                      width: '100%',
                      border: '1px solid ' + C.border,
                      borderRadius: 8,
                      background: C.white,
                      padding: '9px 10px',
                      fontSize: 13,
                      color: C.text,
                    }}
                  >
                    {MATERIAL_TYPES.map((item) => (
                      <option
                        key={item.value}
                        value={item.value}
                      >
                        {item.icon} {item.label}
                      </option>
                    ))}
                  </select>
                </div>

                <div>
                  <button
                    onClick={openFilePicker}
                    disabled={busy}
                    style={{
                      ...buttonPrimary,
                      width: '100%',
                      minHeight: 38,
                      opacity: busy ? 0.6 : 1,
                      cursor: busy
                        ? 'not-allowed'
                        : 'pointer',
                    }}
                  >
                    {busy
                      ? processingText
                      : '＋ 选择Word或PDF资料'}
                  </button>

                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".docx,.pdf"
                    onChange={handleFileChange}
                    style={{ display: 'none' }}
                  />
                </div>
              </div>

              <div
                style={{
                  marginTop: 9,
                  fontSize: 11.5,
                  lineHeight: 1.6,
                  color: C.textMuted,
                }}
              >
                支持.docx和文字版.pdf，单文件不超过10MB，
                提取文字不超过12万字。扫描型PDF请改用课本图片或文字版文件。
              </div>
            </div>
          )}

          {!loading && !canManage && (
            <div
              style={{
                padding: '10px 12px',
                marginBottom: 16,
                borderRadius: 8,
                background: '#F9FAFB',
                color: C.textMuted,
                fontSize: 12,
              }}
            >
              当前为只读查看。只有该单元方案的创建者可以新增或删除参考资料。
            </div>
          )}

          {errorMessage && (
            <div
              style={{
                padding: '10px 12px',
                marginBottom: 14,
                borderRadius: 8,
                border: '1px solid #FECACA',
                background: '#FEF2F2',
                color: C.danger,
                fontSize: 12.5,
                lineHeight: 1.6,
              }}
            >
              ⚠️ {errorMessage}
            </div>
          )}

          {successMessage && (
            <div
              style={{
                padding: '10px 12px',
                marginBottom: 14,
                borderRadius: 8,
                border: '1px solid #A7F3D0',
                background: '#ECFDF5',
                color: C.success,
                fontSize: 12.5,
                lineHeight: 1.6,
              }}
            >
              ✅ {successMessage}
            </div>
          )}

          <div
            style={{
              display: 'flex',
              alignItems: 'center',
              marginBottom: 10,
            }}
          >
            <div
              style={{
                flex: 1,
                fontSize: 13,
                fontWeight: 700,
                color: C.text,
              }}
            >
              已保存资料
            </div>

            <div
              style={{
                fontSize: 12,
                color: C.textMuted,
              }}
            >
              共 {materials.length} 份
            </div>
          </div>

          {loading ? (
            <div
              style={{
                padding: 36,
                textAlign: 'center',
                color: C.textMuted,
                fontSize: 13,
              }}
            >
              正在加载资料…
            </div>
          ) : materials.length === 0 ? (
            <div
              style={{
                padding: 38,
                border: '1px dashed ' + C.border,
                borderRadius: 10,
                textAlign: 'center',
                color: C.textMuted,
                fontSize: 13,
              }}
            >
              还没有保存参考资料
            </div>
          ) : (
            <div
              style={{
                display: 'grid',
                gap: 10,
              }}
            >
              {materials.map((material) => {
                const meta = materialTypeMeta(
                  material.material_type,
                )

                return (
                  <div
                    key={material.id}
                    style={{
                      border: '1px solid ' + C.border,
                      borderRadius: 10,
                      padding: '12px 14px',
                      display: 'flex',
                      alignItems: 'center',
                      gap: 12,
                    }}
                  >
                    <div
                      style={{
                        width: 38,
                        height: 38,
                        borderRadius: 9,
                        background: C.primaryLight,
                        display: 'flex',
                        alignItems: 'center',
                        justifyContent: 'center',
                        fontSize: 20,
                        flexShrink: 0,
                      }}
                    >
                      {meta.icon}
                    </div>

                    <div
                      style={{
                        flex: 1,
                        minWidth: 0,
                      }}
                    >
                      <div
                        style={{
                          fontSize: 13.5,
                          fontWeight: 600,
                          color: C.text,
                          whiteSpace: 'nowrap',
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                        }}
                      >
                        {material.file_name}
                      </div>

                      <div
                        style={{
                          marginTop: 4,
                          fontSize: 11.5,
                          color: C.textMuted,
                          lineHeight: 1.5,
                        }}
                      >
                        {meta.label}
                        {' · 原文'}
                        {formatLength(
                          material.original_length,
                        )}
                        {material.has_summary
                          ? ` · 已提炼为${formatLength(
                              material.summary_length,
                            )}`
                          : ' · 短资料直接使用原文'}
                        {material.created_at
                          ? ` · ${formatDate(
                              material.created_at,
                            )}`
                          : ''}
                      </div>
                    </div>

                    {canManage && (
                      <button
                        onClick={(event) =>
                          handleDelete(material, event)
                        }
                        disabled={busy}
                        style={{
                          border: 'none',
                          background: 'transparent',
                          color: C.textMuted,
                          fontSize: 13,
                          cursor: busy
                            ? 'not-allowed'
                            : 'pointer',
                          padding: '6px 8px',
                        }}
                      >
                        删除
                      </button>
                    )}
                  </div>
                )
              })}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

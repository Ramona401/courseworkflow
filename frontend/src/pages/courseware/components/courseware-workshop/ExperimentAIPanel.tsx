/**
 * ExperimentAIPanel.tsx — 物理/化学/生命科学/地理「AI定制」通用面板
 *
 * 对齐 MathGraphAIPanel 的交互范式：
 *   - 模板改编：基于当前组件HTML底稿做最小修改；
 *   - 从零生成：自然语言生成新互动组件；
 *   - 拍图生成：上传题目、装置、地图、图表或教材截图；
 *   - 追改：已有AI HTML后，把当前HTML作为新底稿继续adapt。
 */

import { useRef, useState } from 'react'
import { useAuth } from '@/store/auth'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import { useVoiceDraftInput } from '@/hooks/useVoiceDraftInput'
import VoiceDraftControls from '@/components/voice/VoiceDraftControls'
import { C } from './workshopConstants'
import { generateSubjectExperimentCode } from '@/api/subjectExperiment'
import type { SubjectExperimentTarget } from '@/api/subjectExperiment'

interface Props {
  target: SubjectExperimentTarget
  mode: 'adapt' | 'create'
  templateName?: string
  baseCode: string
  code: string
  onCode: (code: string) => void
  onExit: () => void
  busyExternal?: boolean
}

interface ExperimentTheme {
  name: string
  icon: string
  main: string
  light: string
  soft: string
  border: string
  gradient: string
  placeholder: string
  followupPlaceholder: string
  imageText: string
  imageHint: string
  generateText: string
  readImageText: string
}

const IMG_MAX_DIM = 1600
const IMG_JPEG_QUALITY = 0.85
const IMG_MAX_FILE_BYTES = 15 * 1024 * 1024

function compressImageToDataURI(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    if (file.size > IMG_MAX_FILE_BYTES) {
      reject(new Error('图片超过15MB，请换一张或先压缩'))
      return
    }

    const reader = new FileReader()

    reader.onerror = () => reject(new Error('图片读取失败'))

    reader.onload = () => {
      const img = new Image()

      img.onerror = () => reject(new Error('图片解码失败，请确认是有效图片'))

      img.onload = () => {
        try {
          const scale = Math.min(
            1,
            IMG_MAX_DIM / Math.max(img.width, img.height),
          )
          const width = Math.round(img.width * scale)
          const height = Math.round(img.height * scale)
          const canvas = document.createElement('canvas')

          canvas.width = width
          canvas.height = height

          const ctx = canvas.getContext('2d')

          if (!ctx) {
            reject(new Error('浏览器不支持图片处理'))
            return
          }

          ctx.fillStyle = '#FFFFFF'
          ctx.fillRect(0, 0, width, height)
          ctx.drawImage(img, 0, 0, width, height)

          resolve(canvas.toDataURL('image/jpeg', IMG_JPEG_QUALITY))
        } catch (error) {
          reject(
            error instanceof Error
              ? error
              : new Error('图片压缩失败'),
          )
        }
      }

      img.src = String(reader.result)
    }

    reader.readAsDataURL(file)
  })
}

function theme(target: SubjectExperimentTarget): ExperimentTheme {
  if (target === 'biology_lab') {
    return {
      name: '生命科学组件',
      icon: '🧬',
      main: '#059669',
      light: '#D1FAE5',
      soft: '#F0FDF4',
      border: '#A7F3D0',
      gradient: 'linear-gradient(135deg,#34D399,#059669)',
      placeholder:
        '描述一个生命科学互动组件，如：设计植物细胞结构观察，切换细胞壁、细胞膜、细胞核、液泡和叶绿体，并显示结构功能……',
      followupPlaceholder:
        '继续追改：如控制条更短、多加一个读数、增加结构标注、加入对照组……',
      imageText: '上传教材、显微图或结构图',
      imageHint: 'AI会参考图片识别结构与观察对象，可在下方补充说明。',
      generateText: '生成生命科学组件',
      readImageText: '读图生成生命科学组件',
    }
  }

  if (target === 'geography_lab') {
    return {
      name: '地理互动探究',
      icon: '🌍',
      main: '#0F766E',
      light: '#CCFBF1',
      soft: '#F0FDFA',
      border: '#99F6E4',
      gradient: 'linear-gradient(135deg,#2DD4BF,#0F766E)',
      placeholder:
        '描述一个地理互动探究，如：设计经纬网定位组件，滑杆控制经纬度，动态显示半球、方向、纬线长度和太阳直射位置……',
      followupPlaceholder:
        '继续追改：如增加图层开关、显示等值线、补充方向标、加入剖面或切换季节……',
      imageText: '上传地图、图表、题目或教材图片',
      imageHint: 'AI会参考图例、方向、变量与空间关系，可在下方补充说明。',
      generateText: '生成地理互动',
      readImageText: '读图生成地理互动',
    }
  }

  if (target === 'physics_lab') {
    return {
      name: '物理实验',
      icon: '🔭',
      main: '#0284C7',
      light: '#E0F2FE',
      soft: '#F0F9FF',
      border: '#BAE6FD',
      gradient: 'linear-gradient(135deg,#38BDF8,#0284C7)',
      placeholder:
        '描述一个物理实验，如：设计电磁感应实验，滑杆控制磁体速度和线圈匝数，显示感应电流方向……',
      followupPlaceholder:
        '继续追改：如控制条更短、多加一个读数、调整装置位置、增加变量控制……',
      imageText: '上传题目或实验装置图片',
      imageHint: 'AI会参考题目和装置结构生成互动实验，可在下方补充说明。',
      generateText: '生成物理实验',
      readImageText: '读图生成物理实验',
    }
  }

  return {
    name: '化学实验',
    icon: '🧪',
    main: '#059669',
    light: '#D1FAE5',
    soft: '#ECFDF5',
    border: '#BBECD8',
    gradient: 'linear-gradient(135deg,#34D399,#059669)',
    placeholder:
      '描述一个化学实验，如：设计酸碱中和实验，滑杆控制NaOH滴加量，显示pH和颜色变化……',
    followupPlaceholder:
      '继续追改：如控制条更短、多加一个读数、把烧杯移到右侧、加入对照组……',
    imageText: '上传题目或实验装置图片',
    imageHint: 'AI会参考图片生成实验装置与现象，可在下方补充说明。',
    generateText: '生成化学实验',
    readImageText: '读图生成化学实验',
  }
}


/**
 * 为实验模板底稿生成短标识，避免把完整HTML写入sessionStorage键。
 */
function hashExperimentDraftIdentity(
  value: string,
): string {
  let hash = 2166136261

  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }

  return (hash >>> 0).toString(36)
}

export default function ExperimentAIPanel({
  target,
  mode,
  templateName,
  baseCode,
  code,
  onCode,
  onExit,
  busyExternal,
}: Props) {
  const { user } = useAuth()
  const currentTheme = theme(target)

  /**
   * 物理、化学、生命科学和地理共用同一保护逻辑，
   * 通过target、模式、模板名和底稿标识彼此隔离。
   */
  const descDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'subject-experiment-ai',
    resourceId: [
      target,
      mode,
      templateName || 'no-template',
      hashExperimentDraftIdentity(baseCode),
    ].join('|'),
    field: 'description',
    initialValue: '',
    maxHistory: 40,
  })
  const desc = descDraft.value
  const setDesc = descDraft.setValue
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [rounds, setRounds] = useState(0)
  const [image, setImage] = useState('')
  const [imgBusy, setImgBusy] = useState(false)

  const fileRef = useRef<HTMLInputElement | null>(null)
  const descriptionRef = useRef<HTMLTextAreaElement | null>(null)

  const hasCode = code.trim().length > 0
  const disabled = loading || Boolean(busyExternal)

  /**
   * 语音只写入当前自然语言描述草稿。
   *
   * 图片、HTML底稿和生成结果均不进入语音识别链路；
   * final结果不会自动调用生成接口。
   */
  const voiceInput = useVoiceDraftInput({
    value: desc,
    setValue: setDesc,
    disabled: disabled || imgBusy,
    maxDurationSeconds: 120,
    onFinalFocus: (finalValue) => {
      const element = descriptionRef.current

      if (!element) {
        return
      }

      element.focus()
      element.setSelectionRange(
        finalValue.length,
        finalValue.length,
      )
    },
    onError: setError,
  })

  /**
   * 录音期间锁定图片、模板切换和生成动作，
   * 但语音按钮自身仍可停止或取消当前录音。
   */
  const interactionDisabled =
    disabled ||
    imgBusy ||
    voiceInput.isActive

  const canAttachImage = !hasCode
  const canSubmit =
    !interactionDisabled &&
    (hasCode
      ? Boolean(desc.trim())
      : Boolean(desc.trim()) || Boolean(image))

  const handlePickImage = async (file: File | null) => {
    if (!file || interactionDisabled) return

    setImgBusy(true)
    setError('')

    try {
      const uri = await compressImageToDataURI(file)
      setImage(uri)
    } catch (caughtError) {
      setError(
        caughtError instanceof Error
          ? caughtError.message
          : '图片处理失败',
      )
    } finally {
      setImgBusy(false)

      if (fileRef.current) {
        fileRef.current.value = ''
      }
    }
  }

  const runGenerate = async (description: string) => {
    setLoading(true)
    setError('')

    try {
      const effectiveMode = hasCode ? 'adapt' : mode
      const effectiveBase = hasCode ? code : baseCode

      const result = await generateSubjectExperimentCode({
        target,
        mode: effectiveMode,
        description,
        base_code:
          effectiveMode === 'adapt'
            ? effectiveBase
            : undefined,
        template_name: templateName,
        image:
          !hasCode && image
            ? image
            : undefined,
      })

      onCode(result.code)
      setRounds(current => current + 1)
      // AI成功返回后才提交文字草稿；失败时保持原文。
      descDraft.commit()
    } catch (caughtError) {
      setError(
        caughtError instanceof Error
          ? caughtError.message
          : '生成失败，请重试',
      )
    } finally {
      setLoading(false)
    }
  }

  const handleGenerate = () => {
    if (!canSubmit || voiceInput.isActive) return
    void runGenerate(desc.trim())
  }

  const handleReset = () => {
    if (interactionDisabled) return

    onCode('')
    setImage('')
    setRounds(0)
    setError('')
  }

  const placeholder = hasCode
    ? currentTheme.followupPlaceholder
    : image
      ? '补充说明（可选）：如只生成第2小问、简化图形、增加变量控制……'
      : currentTheme.placeholder

  return (
    <div>
      <div
        style={{
          padding: '12px 13px',
          borderRadius: 14,
          background:
            'linear-gradient(135deg,' +
            currentTheme.light +
            ',' +
            currentTheme.soft +
            ')',
          border: '1px solid ' + currentTheme.border,
          marginBottom: 13,
        }}
      >
        <div
          style={{
            display: 'flex',
            alignItems: 'center',
            gap: 8,
          }}
        >
          <span
            style={{
              fontSize: 13.5,
              fontWeight: 850,
              color: currentTheme.main,
            }}
          >
            {mode === 'adapt'
              ? '🔧 AI改编模板'
              : '✨ AI新建' + currentTheme.name}
          </span>

          <button
            onClick={() => {
              if (!interactionDisabled) onExit()
            }}
            style={{
              marginLeft: 'auto',
              padding: '4px 10px',
              borderRadius: 999,
              border: '1px solid ' + currentTheme.border,
              background: '#fff',
              color: currentTheme.main,
              fontSize: 11.5,
              fontWeight: 750,
              cursor: interactionDisabled ? 'not-allowed' : 'pointer',
              whiteSpace: 'nowrap',
            }}
          >
            ↩ 返回模板
          </button>
        </div>

        <div
          style={{
            fontSize: 11.6,
            color: '#64748B',
            lineHeight: 1.65,
            marginTop: 6,
          }}
        >
          {mode === 'adapt'
            ? '基于「' +
              (templateName || '当前模板') +
              '」做变种，AI会尽量保留现有结构、底部课堂控制条和离线运行能力。'
            : '用自然语言或图片生成新的' +
              currentTheme.name +
              '，满意后可直接融入课件。'}
        </div>
      </div>

      {canAttachImage && (
        <div style={{ marginBottom: 10 }}>
          <input
            ref={fileRef}
            type="file"
            accept="image/*"
            style={{ display: 'none' }}
            onChange={event => {
              void handlePickImage(
                event.target.files?.[0] || null,
              )
            }}
          />

          {!image ? (
            <button
              onClick={() => {
                if (!interactionDisabled) {
                  fileRef.current?.click()
                }
              }}
              disabled={interactionDisabled}
              style={{
                width: '100%',
                padding: '9px 0',
                borderRadius: 11,
                cursor:
                  interactionDisabled
                    ? 'not-allowed'
                    : 'pointer',
                border:
                  '1.5px dashed ' +
                  currentTheme.border,
                background: '#fff',
                color: currentTheme.main,
                fontSize: 12.5,
                fontWeight: 750,
              }}
            >
              {imgBusy
                ? '⏳ 图片处理中……'
                : '📷 ' +
                  currentTheme.imageText +
                  '（可选）'}
            </button>
          ) : (
            <div
              style={{
                display: 'flex',
                gap: 10,
                alignItems: 'center',
                padding: 8,
                borderRadius: 11,
                border:
                  '1.5px solid ' +
                  currentTheme.border,
                background: currentTheme.soft,
              }}
            >
              <img
                src={image}
                alt={currentTheme.name + '参考图'}
                style={{
                  width: 64,
                  height: 64,
                  objectFit: 'cover',
                  borderRadius: 9,
                  border:
                    '1px solid ' +
                    currentTheme.border,
                  flexShrink: 0,
                }}
              />

              <div
                style={{
                  minWidth: 0,
                  flex: 1,
                }}
              >
                <div
                  style={{
                    fontSize: 12,
                    fontWeight: 800,
                    color: currentTheme.main,
                  }}
                >
                  📷 已附参考图片
                </div>

                <div
                  style={{
                    fontSize: 11,
                    color: '#64748B',
                    marginTop: 3,
                    lineHeight: 1.5,
                  }}
                >
                  {currentTheme.imageHint}
                </div>
              </div>

              <button
                onClick={() => {
                  if (!interactionDisabled) setImage('')
                }}
                title="移除图片"
                style={{
                  border: 'none',
                  background: '#fff',
                  width: 26,
                  height: 26,
                  borderRadius: 8,
                  fontSize: 13,
                  cursor: interactionDisabled
                    ? 'not-allowed'
                    : 'pointer',
                  color: currentTheme.main,
                  flexShrink: 0,
                }}
              >
                ✕
              </button>
            </div>
          )}
        </div>
      )}

      <textarea
        ref={descriptionRef}
        value={desc}
        onChange={event => setDesc(event.target.value)}
        onKeyDown={event => {
          descDraft.handleKeyDown(event)
        }}
        placeholder={placeholder}
        maxLength={2400}
        rows={5}
        disabled={interactionDisabled}
        style={{
          width: '100%',
          boxSizing: 'border-box',
          padding: '10px 11px',
          borderRadius: 11,
          border:
            '1.5px solid ' +
            currentTheme.border,
          fontSize: 12.5,
          lineHeight: 1.6,
          outline: 'none',
          resize: 'vertical',
          fontFamily: 'inherit',
          background: interactionDisabled
            ? '#F9FAFB'
            : '#fff',
        }}
      />

      <VoiceDraftControls
        voice={voiceInput}
        disabled={disabled || imgBusy}
        accentColor={currentTheme.main}
        idleText="点击麦克风可语音描述；识别完成后仍需手动生成"
      />

      <div style={{ marginTop: 6, fontSize: 11, color: C.textMuted, lineHeight: 1.5 }}>
        描述已自动保存 · AI生成失败不会清除 · Ctrl/Command+Z恢复误删
      </div>

      <button
        onClick={handleGenerate}
        disabled={!canSubmit}
        style={{
          width: '100%',
          marginTop: 10,
          padding: '10px 0',
          borderRadius: 12,
          border: 'none',
          fontSize: 13.5,
          fontWeight: 850,
          background: !canSubmit
            ? currentTheme.border
            : currentTheme.gradient,
          boxShadow: !canSubmit
            ? 'none'
            : '0 6px 16px rgba(15,23,42,0.18)',
          color: '#fff',
          cursor: !canSubmit
            ? 'not-allowed'
            : 'pointer',
        }}
      >
        {loading
          ? '⏳ AI生成中（可能需要半分钟以上）……'
          : hasCode
            ? '🔁 按新要求追改'
            : image
              ? '📷 ' + currentTheme.readImageText
              : '✨ ' + currentTheme.generateText}
      </button>

      {error && (
        <div
          style={{
            marginTop: 10,
            padding: '9px 11px',
            borderRadius: 10,
            background: '#FEE2E2',
            color: '#DC2626',
            fontSize: 12,
            lineHeight: 1.5,
          }}
        >
          ❌ {error}
        </div>
      )}

      {hasCode && !loading && (
        <div
          style={{
            marginTop: 10,
            padding: '9px 11px',
            borderRadius: 10,
            background: '#D1FAE5',
            color: '#047857',
            fontSize: 12,
            lineHeight: 1.6,
          }}
        >
          ✅ 已生成第{rounds}轮。请在中间预览区测试滑杆和按钮；不满意可继续追改。

          <span
            onClick={handleReset}
            style={{
              marginLeft: 6,
              color: currentTheme.main,
              fontWeight: 800,
              cursor: interactionDisabled
                ? 'not-allowed'
                : 'pointer',
              textDecoration: 'underline',
            }}
          >
            重置
          </span>
        </div>
      )}

      {!hasCode &&
        !loading &&
        !error &&
        mode === 'adapt' && (
          <div
            style={{
              marginTop: 10,
              fontSize: 11.5,
              color: C.textMuted,
              lineHeight: 1.6,
            }}
          >
            中间当前显示模板底稿。生成后会替换成AI改编结果。
          </div>
        )}
    </div>
  )
}

/**
 * PhysicsSceneAIPanel.tsx — 力学场景 AI 定制面板
 */

import { useRef, useState } from 'react'
import { useAuth } from '@/store/auth'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import { useVoiceDraftInput } from '@/hooks/useVoiceDraftInput'
import VoiceDraftControls from '@/components/voice/VoiceDraftControls'
import { C } from './workshopConstants'
import { generatePhysicsSceneCode } from '@/api/physicsSceneAI'

interface Props {
  mode: 'adapt' | 'create'
  templateName?: string
  baseCode: string
  code: string
  onCode: (code: string) => void
  onExit: () => void
  busyExternal?: boolean
  previewError?: string
}

const IMG_MAX_DIM = 1600
const IMG_JPEG_QUALITY = 0.85
const IMG_MAX_FILE_BYTES = 15 * 1024 * 1024

function compressImageToDataURI(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    if (file.size > IMG_MAX_FILE_BYTES) {
      reject(new Error('图片超过 15MB，请换一张或先压缩'))
      return
    }
    const reader = new FileReader()
    reader.onerror = () => reject(new Error('图片读取失败'))
    reader.onload = () => {
      const img = new Image()
      img.onerror = () => reject(new Error('图片解码失败，请确认是有效图片'))
      img.onload = () => {
        try {
          const scale = Math.min(1, IMG_MAX_DIM / Math.max(img.width, img.height))
          const w = Math.round(img.width * scale)
          const h = Math.round(img.height * scale)
          const canvas = document.createElement('canvas')
          canvas.width = w
          canvas.height = h
          const ctx = canvas.getContext('2d')
          if (!ctx) {
            reject(new Error('浏览器不支持图片处理'))
            return
          }
          ctx.fillStyle = '#FFFFFF'
          ctx.fillRect(0, 0, w, h)
          ctx.drawImage(img, 0, 0, w, h)
          resolve(canvas.toDataURL('image/jpeg', IMG_JPEG_QUALITY))
        } catch (e) {
          reject(e instanceof Error ? e : new Error('图片压缩失败'))
        }
      }
      img.src = String(reader.result)
    }
    reader.readAsDataURL(file)
  })
}


/**
 * 为力学场景模板底稿生成短标识。
 */
function hashPhysicsSceneDraftIdentity(
  value: string,
): string {
  let hash = 2166136261

  for (let index = 0; index < value.length; index += 1) {
    hash ^= value.charCodeAt(index)
    hash = Math.imul(hash, 16777619)
  }

  return (hash >>> 0).toString(36)
}

export default function PhysicsSceneAIPanel({
  mode, templateName, baseCode, code, onCode, onExit, busyExternal, previewError,
}: Props) {
  const { user } = useAuth()

  const descDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'physics-scene-ai',
    resourceId: [
      mode,
      templateName || 'no-template',
      hashPhysicsSceneDraftIdentity(baseCode),
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
  const disabled = loading || !!busyExternal

  /**
   * 力学场景语音只写入自然语言描述。
   *
   * Matter.js底稿、执行报错和生成代码不会进入语音识别；
   * final结果只回填，不自动触发生成或自动修复。
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

  const interactionDisabled =
    disabled ||
    imgBusy ||
    voiceInput.isActive

  const canAttachImage = !hasCode
  const canAutoFix = hasCode && !!previewError?.trim()
  const canSubmit =
    !interactionDisabled &&
    (hasCode
      ? !!desc.trim()
      : !!desc.trim() || !!image)

  const handlePickImage = async (file: File | null) => {
    if (!file || interactionDisabled) return
    setImgBusy(true)
    setError('')
    try {
      const uri = await compressImageToDataURI(file)
      setImage(uri)
    } catch (e) {
      setError(e instanceof Error ? e.message : '图片处理失败')
    } finally {
      setImgBusy(false)
      if (fileRef.current) fileRef.current.value = ''
    }
  }

  const runGenerate = async (
    description: string,
    consumeManualDraft = true,
  ) => {
    setLoading(true)
    setError('')
    try {
      const effectiveMode = hasCode ? 'adapt' : mode
      const effectiveBase = hasCode ? code : baseCode
      const result = await generatePhysicsSceneCode({
        target: 'physics_scene',
        mode: effectiveMode,
        description,
        base_code: effectiveMode === 'adapt' ? effectiveBase : undefined,
        template_name: templateName,
        image: !hasCode && image ? image : undefined,
      })
      onCode(result.code)
      setRounds(r => r + 1)
      // 自动修复不清除老师正在编辑的人工追改草稿。
      if (consumeManualDraft) {
        descDraft.commit()
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : '生成失败，请重试')
    } finally {
      setLoading(false)
    }
  }

  const handleGenerate = () => {
    if (!canSubmit || voiceInput.isActive) return
    void runGenerate(desc.trim())
  }

  const handleAutoFix = () => {
    if (interactionDisabled || !canAutoFix) return
    const fixDesc = '这段 Matter.js setup 代码在预览中执行时报错了，报错信息是：' + (previewError || '').trim()
      + '。请只修复导致报错的问题，保持场景主体和教学设计不变，输出修复后的完整 setup 代码。'
    void runGenerate(fixDesc, false)
  }

  const handleReset = () => {
    if (interactionDisabled) return
    onCode('')
    setImage('')
    setRounds(0)
    setError('')
  }

  const placeholder = hasCode
    ? '继续追改：如 增加斜面摩擦对比 / 加一个弹簧 / 让小球从更高处释放 / 颜色更清晰…'
    : (image ? '补充说明（可选）：如 只做第2小问 / 简化为斜面滑块 / 增加对照组…' : '描述一个力学场景，如：生成一个小车撞弹簧的能量转化场景，带墙体、弹簧、小车和地面；或生成双球碰撞动量守恒演示…')

  return (
    <div>
      <div style={{ padding: '12px 13px', borderRadius: 14, background: 'linear-gradient(135deg,#FEF2F2,#FEE2E2)', border: '1px solid #FBD5D5', marginBottom: 13 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
          <span style={{ fontSize: 13.5, fontWeight: 850, color: '#B91C1C' }}>
            {mode === 'adapt' ? '🔧 AI 改编力学场景' : '✨ AI 新建力学场景'}
          </span>
          <button
            onClick={() => { if (!interactionDisabled) onExit() }}
            style={{ marginLeft: 'auto', padding: '4px 10px', borderRadius: 999, border: '1px solid #FBD5D5', background: '#fff', color: '#B91C1C', fontSize: 11.5, fontWeight: 750, cursor: interactionDisabled ? 'not-allowed' : 'pointer', whiteSpace: 'nowrap' }}
          >↩ 返回模板</button>
        </div>
        <div style={{ fontSize: 11.6, color: '#B07070', lineHeight: 1.65, marginTop: 6 }}>
          {mode === 'adapt'
            ? '基于「' + (templateName || '当前模板') + '」做最小必要改写，生成 Matter.js setup 代码。'
            : '用自然语言或题目图片生成新的 Matter.js 力学仿真场景。'}
        </div>
      </div>

      {canAutoFix && !loading && (
        <div style={{ marginBottom: 10, padding: '10px 12px', borderRadius: 11, background: '#FFF7ED', border: '1.5px solid #FDBA74' }}>
          <div style={{ fontSize: 12, fontWeight: 750, color: '#9A3412', lineHeight: 1.5 }}>⚠️ 预览执行报错，可让 AI 自动修复。</div>
          <button
            onClick={handleAutoFix}
            disabled={interactionDisabled}
            style={{
              width: '100%', marginTop: 8, padding: '8px 0', borderRadius: 9, border: 'none', fontSize: 12.5, fontWeight: 850,
              background: interactionDisabled ? '#FDBA74' : 'linear-gradient(135deg,#F59E0B,#EA580C)',
              color: '#fff', cursor: interactionDisabled ? 'not-allowed' : 'pointer',
            }}
          >🔧 让 AI 修复此错误</button>
        </div>
      )}

      {canAttachImage && (
        <div style={{ marginBottom: 10 }}>
          <input
            ref={fileRef}
            type="file"
            accept="image/*"
            style={{ display: 'none' }}
            onChange={e => handlePickImage(e.target.files?.[0] || null)}
          />
          {!image ? (
            <button
              onClick={() => { if (!interactionDisabled) fileRef.current?.click() }}
              disabled={interactionDisabled}
              style={{
                width: '100%',
                padding: '9px 0',
                borderRadius: 11,
                cursor: interactionDisabled ? 'not-allowed' : 'pointer',
                border: '1.5px dashed #FBD5D5',
                background: '#fff',
                color: '#B91C1C',
                fontSize: 12.5,
                fontWeight: 750,
              }}
            >{imgBusy ? '⏳ 图片处理中…' : '📷 上传题目/物理示意图（可选）'}</button>
          ) : (
            <div style={{ display: 'flex', gap: 10, alignItems: 'center', padding: 8, borderRadius: 11, border: '1.5px solid #FBD5D5', background: '#FEF2F2' }}>
              <img src={image} alt="题目图片" style={{ width: 64, height: 64, objectFit: 'cover', borderRadius: 9, border: '1px solid #FBD5D5', flexShrink: 0 }} />
              <div style={{ minWidth: 0, flex: 1 }}>
                <div style={{ fontSize: 12, fontWeight: 800, color: '#B91C1C' }}>📷 已附参考图片</div>
                <div style={{ fontSize: 11, color: '#B07070', marginTop: 3, lineHeight: 1.5 }}>AI 会参考图片生成力学场景，可在下方补充说明。</div>
              </div>
              <button
                onClick={() => { if (!interactionDisabled) setImage('') }}
                title="移除图片"
                style={{ border: 'none', background: '#fff', width: 26, height: 26, borderRadius: 8, fontSize: 13, cursor: interactionDisabled ? 'not-allowed' : 'pointer', color: '#B91C1C', flexShrink: 0 }}
              >✕</button>
            </div>
          )}
        </div>
      )}

      <textarea
        ref={descriptionRef}
        value={desc}
        onChange={e => setDesc(e.target.value)}
        onKeyDown={e => {
          descDraft.handleKeyDown(e)
        }}
        placeholder={placeholder}
        maxLength={2400}
        rows={5}
        disabled={interactionDisabled}
        style={{ width: '100%', boxSizing: 'border-box', padding: '10px 11px', borderRadius: 11, border: '1.5px solid #FBD5D5', fontSize: 12.5, lineHeight: 1.6, outline: 'none', resize: 'vertical', fontFamily: 'inherit', background: interactionDisabled ? '#F9FAFB' : '#fff' }}
      />

      <VoiceDraftControls
        voice={voiceInput}
        disabled={disabled || imgBusy}
        accentColor="#B91C1C"
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
          background: !canSubmit ? '#FCA5A5' : 'linear-gradient(135deg,#F87171,#B91C1C)',
          boxShadow: !canSubmit ? 'none' : '0 6px 16px rgba(185,28,28,0.24)',
          color: '#fff',
          cursor: !canSubmit ? 'not-allowed' : 'pointer',
        }}
      >
        {loading ? '⏳ AI 生成中（可能需要半分钟以上）…' : hasCode ? '🔁 按新要求追改' : (image ? '📷 读图生成场景' : '✨ 生成力学场景')}
      </button>

      {error && (
        <div style={{ marginTop: 10, padding: '9px 11px', borderRadius: 10, background: '#FEE2E2', color: '#DC2626', fontSize: 12, lineHeight: 1.5 }}>
          ❌ {error}
        </div>
      )}

      {hasCode && !loading && (
        <div style={{ marginTop: 10, padding: '9px 11px', borderRadius: 10, background: '#D1FAE5', color: '#047857', fontSize: 12, lineHeight: 1.6 }}>
          ✅ 已生成第 {rounds} 轮。请在中间预览区按播放测试；不满意可继续追改。
          <span onClick={handleReset} style={{ marginLeft: 6, color: '#B91C1C', fontWeight: 800, cursor: interactionDisabled ? 'not-allowed' : 'pointer', textDecoration: 'underline' }}>重置</span>
        </div>
      )}

      {!hasCode && !loading && !error && mode === 'adapt' && (
        <div style={{ marginTop: 10, fontSize: 11.5, color: C.textMuted, lineHeight: 1.6 }}>
          中间当前显示模板底稿。生成后会替换成 AI 改编场景。
        </div>
      )}
    </div>
  )
}

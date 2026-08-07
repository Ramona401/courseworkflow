/**
 * MediaManagerPanel.tsx — 多媒体管理面板·编排壳（批次5b模块化拆分后）
 *
 * 历史：批次W1从 CoursewareWorkshopPage 整体抽出；批次5a加入红色删除确认弹窗
 * (DangerConfirmModal) 与 makeAsset 工厂；批次5b按600行红线拆分为四个模块：
 *   - makeAsset.ts               : 本地资产对象工厂（共享）
 *   - MediaImageSuggestPanel.tsx : AI配图建议整列（建议状态/缓存/批量生成自含）
 *   - MediaAssetCards.tsx        : ImageAssetList/VideoAssetList 资产卡片列表
 *                                  （云盘上传/复制链接逻辑去重为 CloudActions）
 *   - 本文件                      : 编排壳——共享状态(mediaAssets/消息条/参考图/
 *                                  弹窗/编辑器)+单张生成+手动上传+S-V2导出链
 *
 * 批次5c（按钮风格优先修复，2026-07-03）：风格快选从"末尾追加描述短语"改为
 * "置顶硬约束前缀块"。
 *
 * 批次5d（视频手动上传入口，2026-07-07）：在视频 Tab 增加与图片 Tab 对称的
 * "📤 手动上传视频"区块。
 *
 * 批次1a（学科工具收编，2026-07-08）：原批次5e加入的 formula/music 两 Tab 及更早的
 * stroke Tab 整体迁出到 SubjectToolsPanel.tsx（Step5「🧪学科工具」聚合宫格），
 * 本面板 mediaTab 收窄回 image|video|audio 三值——媒体面板只管"资产"
 * （有列表有云盘），学科面板只管"组件"（编辑器→AI融入页面）。
 *
 * S-V2：onExport 三步串行链（advancedConcat → burnIn可选 → mixNarration可选），
 * 每步失败自动降级保留上一步可用成片，与三选项导出弹窗配套。
 */
import { useState, useEffect } from 'react'
import { useAuth } from '@/store/auth'
import { useProtectedDraft } from '@/hooks/useProtectedDraft'
import {
  generateCWImage, uploadCWImage, listPageAssets, deleteCWAsset,
  advancedConcatCWVideos, uploadCWVideo, burnInSubtitle,
} from '@/api/coursewares'
import type {
  CoursewareAsset,
  CoursewareDetail,
  SetStyleAnchorResult,
} from '@/api/coursewares'
import { mixNarrationCWVideo, uploadCWAudio } from '@/api/coursewares.media'
import { C, CW_IMG_STYLES } from './workshopConstants'
import VideoStoryboardPanel from '../VideoStoryboardPanel'
import VideoEditorModal from '../VideoEditorModal'
import DangerConfirmModal from './DangerConfirmModal'
import MediaImageSuggestPanel from './MediaImageSuggestPanel'
import { ImageAssetList, VideoAssetList, AudioAssetList } from './MediaAssetCards'
import { makeAsset } from './makeAsset'
import AudioEditorModal from '../audio-editor/AudioEditorModal'
import StyleStudioModal from '../style-studio/StyleStudioModal'

// ==================== 批次5c: 按钮风格优先——纯函数工具 ====================

const buildStylePrefix = (desc: string): string =>
  '【整体画风·最高优先级】' + desc + '。全图必须统一采用本段画风；下文若出现任何其他画风描述（绘本、水彩、扁平、手绘、写实等），一律忽略，禁止混合画风。\n'

const stripAIStyleTail = (text: string): { base: string; tail: string } => {
  const m = text.match(/(整体画风[：:][^\n]*)\s*$/)
  if (m && typeof m.index === 'number') {
    return { base: text.slice(0, m.index).trimEnd(), tail: m[1].trim() }
  }
  return { base: text, tail: '' }
}

// ==================== 批次5d: 视频上传辅助常量 ====================

/** 视频允许的 MIME 类型白名单（与后端 cwVideoAllowedMimeTypes 对齐） */
const VIDEO_ACCEPT = 'video/mp4,video/webm,video/quicktime,video/x-msvideo,.mp4,.webm,.mov,.avi'
/** 视频最大文件大小 50MB（与后端 CWVideoMaxSize 对齐） */
const VIDEO_MAX_SIZE = 50 * 1024 * 1024

interface Props {
  coursewareId: string
  pageNum: number
  courseware: CoursewareDetail
  anchorSetting: string
  anchorClearing: boolean
  onSetAnchor: (assetId: string, notify: (msg: string) => void) => void
  onClearAnchor: (notify: (msg: string) => void) => void
  /** 自定义美术风格确认后，通知父级乐观更新课程锚点。 */
  onAnchorChanged?: (result: SetStyleAnchorResult) => void
  /** 批次1a: stroke/formula/music 已迁出到 SubjectToolsPanel，收窄回三值 */
  mediaTab: 'image' | 'video' | 'audio'
  onPageUpdated?: (pageNum: number, html: string) => void
}


const MEDIA_IMAGE_SIZE_OPTIONS = [
  '1920x1920',
  '2560x1440',
  '3072x1280',
  '1440x2560',
]

interface MediaImageDraftForm {
  prompt: string
  size: string
  styleKey: string
  stylePrefixText: string
  strippedStyleTail: string
}

function createMediaImageInitialForm():
  MediaImageDraftForm {
  return {
    prompt: '',
    size: '1920x1920',
    styleKey: '',
    stylePrefixText: '',
    strippedStyleTail: '',
  }
}

function parseMediaImageDraftForm(
  raw: string,
  fallback: MediaImageDraftForm,
): MediaImageDraftForm {
  if (!raw.trim()) {
    return {
      ...fallback,
    }
  }

  try {
    const parsed = JSON.parse(
      raw,
    ) as Partial<MediaImageDraftForm>

    const size =
      typeof parsed.size === 'string'
      && MEDIA_IMAGE_SIZE_OPTIONS.includes(
        parsed.size,
      )
        ? parsed.size
        : fallback.size

    const styleKey =
      typeof parsed.styleKey === 'string'
      && CW_IMG_STYLES.some(
        style =>
          style.key === parsed.styleKey,
      )
        ? parsed.styleKey
        : ''

    return {
      prompt:
        typeof parsed.prompt === 'string'
          ? parsed.prompt
          : '',
      size,
      styleKey,
      stylePrefixText:
        typeof parsed.stylePrefixText ===
          'string'
          ? parsed.stylePrefixText
          : '',
      strippedStyleTail:
        typeof parsed.strippedStyleTail ===
          'string'
          ? parsed.strippedStyleTail
          : '',
    }
  } catch {
    return {
      ...fallback,
    }
  }
}

export default function MediaManagerPanel({ coursewareId, pageNum, courseware, anchorSetting, anchorClearing, onSetAnchor, onClearAnchor, onAnchorChanged, mediaTab }: Props) {
  // ==================== 共享状态 ====================
  const { user } = useAuth()

  const [mediaAssets, setMediaAssets] =
    useState<CoursewareAsset[]>([])

  /**
   * 图片生成文字表单按用户、课件和页码隔离。
   *
   * 这里只保存纯文字和枚举状态；
   * 参考图URL、上传文件、资产ID和生成结果均不进入草稿。
   */
  const initialMediaImageForm =
    createMediaImageInitialForm()

  const mediaImageDraft = useProtectedDraft({
    userId: user?.id,
    scope: 'courseware-media-image',
    resourceId: [
      coursewareId,
      `page-${pageNum}`,
    ].join('|'),
    field: 'generation-form',
    initialValue:
      JSON.stringify(
        initialMediaImageForm,
      ),
    maxHistory: 30,
  })

  const mediaImageForm =
    parseMediaImageDraftForm(
      mediaImageDraft.value,
      initialMediaImageForm,
    )

  const updateMediaImageForm = (
    patch: Partial<MediaImageDraftForm>,
  ) => {
    mediaImageDraft.setValue(
      previousText =>
        JSON.stringify({
          ...parseMediaImageDraftForm(
            previousText,
            initialMediaImageForm,
          ),
          ...patch,
        }),
    )
  }

  const mediaGenPrompt =
    mediaImageForm.prompt
  const mediaSize =
    mediaImageForm.size
  const mediaStyleKey =
    mediaImageForm.styleKey
  const stylePrefixText =
    mediaImageForm.stylePrefixText
  const strippedStyleTail =
    mediaImageForm.strippedStyleTail

  const setMediaGenPrompt = (
    value: string,
  ) => updateMediaImageForm({
    prompt: value,
  })

  const setMediaSize = (
    value: string,
  ) => updateMediaImageForm({
    size: value,
  })

  const setMediaStyleKey = (
    value: string,
  ) => updateMediaImageForm({
    styleKey: value,
  })

  const setStylePrefixText = (
    value: string,
  ) => updateMediaImageForm({
    stylePrefixText: value,
  })

  const setStrippedStyleTail = (
    value: string,
  ) => updateMediaImageForm({
    strippedStyleTail: value,
  })

  const [mediaGenerating, setMediaGenerating] = useState(false)
  const [mediaMessage, setMediaMessage] = useState('')
  const [mediaPreviewUrl, setMediaPreviewUrl] = useState('')
  const [mediaRefUrl, setMediaRefUrl] = useState('')
  const [styleStudioOpen, setStyleStudioOpen] = useState(false)
  const [editorOpen, setEditorOpen] = useState(false)
  const [editorExporting, setEditorExporting] = useState(false)
  const [deleteTarget, setDeleteTarget] = useState<CoursewareAsset | null>(null)
  const [deleting, setDeleting] = useState(false)
  // 批次5d: 视频手动上传进度
  const [videoUploadProgress, setVideoUploadProgress] = useState(0)
  const [videoUploading, setVideoUploading] = useState(false)
  // 音频剪辑器状态
  const [audioEditorAsset, setAudioEditorAsset] = useState<CoursewareAsset | null>(null)

  // ==================== 共享effect ====================
  useEffect(() => {
    if (!coursewareId || pageNum <= 0) { setMediaAssets([]); return }
    let cancelled = false
    setMediaAssets([])
    listPageAssets(coursewareId, pageNum)
      .then(res => { if (!cancelled) setMediaAssets(res.assets || []) })
      .catch(() => { if (!cancelled) setMediaAssets([]) })
    return () => { cancelled = true }
  }, [coursewareId, pageNum])

  useEffect(() => {
    /**
     * 参考图与当前页面运行态强相关，不进入文字草稿。
     * 图片生成文字表单由统一Hook按照页码自动切换和恢复。
     */
    setMediaRefUrl('')
  }, [pageNum, mediaTab])

  // ==================== 共享处理函数 ====================
  const toggleImgStyle = (key: string) => {
    const next = CW_IMG_STYLES.find(s => s.key === key)
    const canceling = key === mediaStyleKey
    let base = mediaGenPrompt
    if (stylePrefixText) {
      if (base.startsWith(stylePrefixText)) base = base.slice(stylePrefixText.length)
      else if (base.includes(stylePrefixText)) base = base.replace(stylePrefixText, '')
    }
    base = base.replace(/^\s+/, '')
    if (canceling) {
      if (strippedStyleTail) {
        base = base ? base.trimEnd() + '\n' + strippedStyleTail : strippedStyleTail
      }
      setStylePrefixText('')
      setStrippedStyleTail('')
      setMediaGenPrompt(base)
      setMediaStyleKey('')
      return
    }
    if (!strippedStyleTail) {
      const stripped = stripAIStyleTail(base)
      if (stripped.tail) {
        base = stripped.base
        setStrippedStyleTail(stripped.tail)
      }
    }
    const prefix = next ? buildStylePrefix(next.desc) : ''
    setStylePrefixText(prefix)
    setMediaGenPrompt(prefix + base)
    setMediaStyleKey(key)
  }

  /**
   * 课程自定义画风确认后，清除当前表单中的快捷风格前缀。
   *
   * 统一画风已经由课程锚点负责，
   * 不再让单张图片提示词中的快捷前缀与锚点互相冲突。
   */
  const clearManualStyleSelection = () => {
    let base = mediaGenPrompt

    if (stylePrefixText) {
      if (base.startsWith(stylePrefixText)) {
        base = base.slice(stylePrefixText.length)
      } else if (base.includes(stylePrefixText)) {
        base = base.replace(stylePrefixText, '')
      }
    }

    base = base.replace(/^\s+/, '')

    if (strippedStyleTail) {
      base = base
        ? base.trimEnd() + '\n' + strippedStyleTail
        : strippedStyleTail
    }

    setStylePrefixText('')
    setStrippedStyleTail('')
    setMediaStyleKey('')
    setMediaGenPrompt(base)
  }

  const handleConfirmDelete = async () => {
    if (!coursewareId || !deleteTarget || deleting) return
    setDeleting(true)
    try {
      await deleteCWAsset(coursewareId, deleteTarget.id)
      setMediaAssets(prev => prev.filter(a => a.id !== deleteTarget.id))
      setMediaMessage('✅ 已删除')
    } catch (e) {
      setMediaMessage('❌ 删除失败: ' + (e instanceof Error ? e.message : '未知错误'))
    } finally {
      setDeleting(false)
      setDeleteTarget(null)
    }
  }

  const buildDeleteWarning = (asset: CoursewareAsset): string => {
    const noun = asset.asset_type === 'video' ? '这个视频' : asset.asset_type === 'audio' ? '这个音频' : '这张图片'
    const effect = asset.asset_type === 'video' ? '视频将无法播放' : asset.asset_type === 'audio' ? '音频将无法播放' : '图片将无法显示'
    if (asset.public_oss_url) {
      return '⚠️ ' + noun + '已上传云盘。\n删除将同时移除云盘副本——若课件页面中已使用该云盘链接，' + effect + '。\n此操作不可恢复，确定删除？'
    }
    return '确定删除' + noun + '？删除后不可恢复。'
  }

  // 批次5d: 视频手动上传处理函数
  const handleUploadVideo = () => {
    if (videoUploading || !coursewareId || pageNum <= 0) return
    const inp = document.createElement('input')
    inp.type = 'file'
    inp.accept = VIDEO_ACCEPT
    inp.onchange = async (ev) => {
      const f = (ev.target as HTMLInputElement).files?.[0]
      if (!f) return
      if (f.size > VIDEO_MAX_SIZE) {
        setMediaMessage('❌ 视频文件不能超过50MB，当前文件 ' + (f.size / (1024 * 1024)).toFixed(1) + 'MB')
        return
      }
      setVideoUploading(true)
      setVideoUploadProgress(0)
      setMediaMessage('⏳ 正在上传视频...')
      try {
        const res = await uploadCWVideo(coursewareId, pageNum, f, (pct) => {
          setVideoUploadProgress(pct)
        })
        setMediaMessage('✅ 视频上传成功！')
        setMediaAssets(prev => [makeAsset(coursewareId, {
          id: res.asset_id, oss_url: res.url, asset_type: 'video',
          generation_prompt: f.name, file_size: res.file_size, mime_type: res.mime_type,
        }), ...prev])
      } catch (e) {
        setMediaMessage('❌ 视频上传失败: ' + (e instanceof Error ? e.message : '未知错误'))
      } finally {
        setVideoUploading(false)
        setVideoUploadProgress(0)
      }
    }
    inp.click()
  }

  // ==================== JSX ====================
  return (
    <div style={{ marginTop: 16, padding: 20, borderRadius: 12, border: '1px solid ' + C.border, background: '#FAFAFA' }}>
      <div style={{ fontSize: 15, fontWeight: 600, color: C.textPrimary, marginBottom: 12 }}>🖼️ 多媒体管理</div>

      {/* 媒体管理跟随上方大预览框选中页 */}
      <div style={{ display: 'flex', gap: 10, flexWrap: 'wrap', marginBottom: 16, alignItems: 'center' }}>
        <span style={{ padding: '10px 14px', borderRadius: 8, background: C.primaryBg, color: C.primary, fontSize: 14, fontWeight: 600, whiteSpace: 'nowrap' }}>
          正在管理：第 {pageNum || '—'} 页的{mediaTab === 'video' ? '视频' : mediaTab === 'audio' ? '音频' : '图片'}
        </span>
        <span style={{ fontSize: 12, color: C.textMuted }}>（切换上方预览页即可管理对应页的媒体）</span>
      </div>

      {/* ==================== 图片Tab ==================== */}
      {pageNum > 0 && mediaTab === 'image' && (
        <>
        {/* 锚点缩略图条 */}
        <div style={{ display: 'flex', alignItems: 'center', gap: 10, marginBottom: 12, padding: '8px 12px', borderRadius: 10, background: courseware.style_anchor_asset_id ? 'rgba(245,158,11,0.07)' : '#FAFAFA', border: '1px solid ' + (courseware.style_anchor_asset_id ? 'rgba(245,158,11,0.3)' : C.border) }}>
          {courseware.style_anchor_asset_id ? (
            <>
              {courseware.style_anchor_url && (
                <img src={courseware.style_anchor_url} alt="风格锚点" onClick={() => setMediaPreviewUrl(courseware.style_anchor_url)} style={{ width: 44, height: 44, objectFit: 'cover', borderRadius: 8, border: '2px solid #F59E0B', cursor: 'pointer', flexShrink: 0 }} title="点击查看锚点大图" />
              )}
              <div style={{ flex: 1, minWidth: 0 }}>
                <div style={{ fontSize: 12, fontWeight: 600, color: '#B45309' }}>⭐ 当前风格锚点</div>
                <div style={{ fontSize: 11, color: C.textMuted, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>本页及全课件配图将自动参考此图，保持风格与人物一致</div>
              </div>
              <button onClick={() => onClearAnchor(setMediaMessage)} disabled={anchorClearing}
                style={{ padding: '4px 12px', borderRadius: 6, border: '1px solid ' + C.danger, background: 'transparent', color: C.danger, fontSize: 12, cursor: anchorClearing ? 'default' : 'pointer', whiteSpace: 'nowrap', flexShrink: 0 }}>
                {anchorClearing ? '清除中...' : '✕ 清除锚点'}
              </button>
            </>
          ) : (
            <div style={{ fontSize: 12, color: C.textMuted, lineHeight: 1.5 }}>
              💡 还未设风格锚点。在下方任意图片上点「⭐设为锚点」，之后全课件配图都会自动参考它，保持<b>画风与人物形象一致</b>。
            </div>
          )}
        </div>

        {/* 上半: 左右两栏 —— 左=AI配图建议, 右=生成图片 */}
        <div style={{ display: 'flex', gap: 16, flexWrap: 'wrap' }}>
          <MediaImageSuggestPanel
            coursewareId={coursewareId}
            pageNum={pageNum}
            active={mediaTab === 'image' && pageNum > 0}
            defaultSize={mediaSize}
            busyExternal={mediaGenerating}
            onFillPrompt={(prompt, size) => {
              if (stylePrefixText) {
                const stripped = stripAIStyleTail(prompt)
                setStrippedStyleTail(stripped.tail)
                setMediaGenPrompt(stylePrefixText + stripped.base)
              } else {
                setStrippedStyleTail('')
                setMediaGenPrompt(prompt)
              }
              setMediaSize(size)
            }}
            onAssetCreated={(asset) => setMediaAssets(prev => [asset, ...prev])}
            notify={setMediaMessage}
          />

          {/* 右栏: 生成图片 */}
          <div style={{ flex: '1 1 320px', padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
            <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>🤖 生成图片</div>
            <div style={{ marginBottom: 8 }}>
              {courseware.style_anchor_asset_id ? (
                <div style={{ padding: '8px 12px', borderRadius: 8, background: 'rgba(245,158,11,0.08)', border: '1px solid rgba(245,158,11,0.3)', display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 10, flexWrap: 'wrap' }}>
                  <div style={{ flex: 1, minWidth: 220, fontSize: 11, color: '#B45309', lineHeight: 1.6 }}>
                    ⭐ 已设统一画风，生成图片时会自动保持全课件视觉一致。
                  </div>
                  <button
                    type="button"
                    disabled={mediaGenerating}
                    onClick={() => setStyleStudioOpen(true)}
                    title="调整课程图片画风"
                    style={{ padding: '5px 11px', borderRadius: 7, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, fontWeight: 700, cursor: mediaGenerating ? 'default' : 'pointer', whiteSpace: 'nowrap' }}
                  >
                    ✨ 自定义
                  </button>
                </div>
              ) : (
                <>
                  <div style={{ fontSize: 11, color: '#6B7280', marginBottom: 6 }}>🎨 画面风格（点选后置顶为最高优先级画风，覆盖提示词里的其他画风描述）:</div>
                  <div style={{ display: 'flex', gap: 6, flexWrap: 'wrap' }}>
                    {CW_IMG_STYLES.map(s => {
                      const on = mediaStyleKey === s.key
                      return (
                        <button key={s.key} onClick={() => toggleImgStyle(s.key)} disabled={mediaGenerating}
                          style={{ padding: '5px 10px', borderRadius: 14, border: '1px solid ' + (on ? '#7C3AED' : C.border), background: on ? 'rgba(124,58,237,0.1)' : '#fff', color: on ? '#7C3AED' : C.textSecondary, fontSize: 11, fontWeight: on ? 600 : 400, cursor: mediaGenerating ? 'default' : 'pointer' }}>
                          {s.label}
                        </button>
                      )
                    })}
                    <button
                      type="button"
                      onClick={() => setStyleStudioOpen(true)}
                      disabled={mediaGenerating}
                      title="通过对话或参考图自定义课程图片画风"
                      style={{ padding: '5px 10px', borderRadius: 14, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, fontWeight: 700, cursor: mediaGenerating ? 'default' : 'pointer' }}
                    >
                      ✨ 自定义
                    </button>
                  </div>
                </>
              )}
            </div>
            <textarea
              value={mediaGenPrompt}
              onChange={e =>
                setMediaGenPrompt(
                  e.target.value,
                )
              }
              onKeyDown={e => {
                mediaImageDraft.handleKeyDown(
                  e,
                )
              }}
              placeholder="从左侧建议点「→ 填入右侧」，或在此手动描述要生成的图片；可点上方风格按钮美化"
              rows={5} style={{ width: '100%', padding: '10px 12px', borderRadius: 8, border: '1px solid ' + C.border, fontSize: 13, resize: 'vertical', outline: 'none', boxSizing: 'border-box' }} disabled={mediaGenerating} />

            <div
              style={{
                marginTop: 5,
                fontSize: 10,
                color: C.textMuted,
                lineHeight: 1.5,
              }}
            >
              当前页图片描述与画风设置已自动保存 ·
              生成失败不会清除 ·
              Ctrl/Command+Z恢复误删
            </div>

            <div style={{ marginTop: 8 }}>
              <span style={{ fontSize: 12, color: '#6B7280', marginRight: 8 }}>图片比例:</span>
              <select value={mediaSize} onChange={e => setMediaSize(e.target.value)}
                style={{ padding: '6px 10px', borderRadius: 6, border: '1px solid #E5E7EB', fontSize: 12 }} disabled={mediaGenerating}>
                <option value="1920x1920">1:1 正方形</option>
                <option value="2560x1440">16:9 宽屏</option>
                <option value="3072x1280">2.4:1 超宽</option>
                <option value="1440x2560">9:16 竖屏</option>
              </select>
            </div>
            <div style={{ marginTop: 8, display: 'flex', alignItems: 'center', gap: 8, flexWrap: 'wrap' }}>
              <span style={{ fontSize: 12, color: '#6B7280' }}>参考图:</span>
              {mediaRefUrl ? (
                <div style={{ display: 'flex', alignItems: 'center', gap: 6 }}>
                  <img src={mediaRefUrl} alt="参考图" style={{ width: 40, height: 40, objectFit: 'cover', borderRadius: 6, border: '2px solid #7C3AED' }} />
                  <span style={{ fontSize: 11, color: '#7C3AED' }}>已选择参考图</span>
                  <button onClick={() => setMediaRefUrl('')} style={{ padding: '2px 8px', borderRadius: 4, border: '1px solid #EF4444', background: 'transparent', color: '#EF4444', fontSize: 11, cursor: 'pointer' }}>取消</button>
                </div>
              ) : (
                <>
                <span style={{ fontSize: 11, color: '#9CA3AF' }}>无</span>
                <button onClick={() => {
                  const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/*'
                  inp.onchange = async (ev) => {
                    const f = (ev.target as HTMLInputElement).files?.[0]
                    if (!f || !coursewareId || pageNum <= 0) return
                    if (f.size > 5 * 1024 * 1024) { setMediaMessage('❌ 参考图不能超过5MB'); return }
                    setMediaMessage('⏳ 上传参考图中...')
                    try {
                      const res = await uploadCWImage(coursewareId, pageNum, f)
                      setMediaRefUrl(res.url)
                      setMediaAssets(prev => [makeAsset(coursewareId, { id: res.asset_id, oss_url: res.url, file_size: res.file_size, mime_type: res.mime_type }), ...prev])
                      setMediaMessage('✅ 参考图上传成功，已自动选为参考图')
                    } catch (e) { setMediaMessage('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                  }; inp.click()
                }} style={{ padding: '3px 10px', borderRadius: 5, border: '1px solid #7C3AED', background: 'rgba(124,58,237,0.06)', color: '#7C3AED', fontSize: 11, cursor: 'pointer' }}>📤 上传参考图</button>
                </>
              )}
            </div>
            <button
              onClick={async () => {
                if (!coursewareId || pageNum <= 0 || !mediaGenPrompt.trim() || mediaGenerating) return
                setMediaGenerating(true); setMediaMessage('')
                try {
                  const res = await generateCWImage(coursewareId, pageNum, mediaGenPrompt.trim(), undefined, mediaSize, mediaRefUrl || undefined)
                  setMediaMessage('✅ 图片生成成功！')
                  setMediaAssets(prev => [makeAsset(coursewareId, { id: res.asset_id, oss_url: res.url, generation_prompt: mediaGenPrompt }), ...prev])

                  /**
                   * 图片已成功生成并返回正式资产后，才提交清空表单。
                   * commit保留撤销快照，Ctrl+Z仍可恢复刚刚使用的描述。
                   */
                  mediaImageDraft.commit()
                } catch (e) { setMediaMessage('❌ 生成失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                finally { setMediaGenerating(false) }
              }}
              disabled={mediaGenerating || !mediaGenPrompt.trim()}
              style={{ marginTop: 8, padding: '10px 20px', borderRadius: 8, border: 'none', background: mediaGenPrompt.trim() && !mediaGenerating ? 'linear-gradient(135deg, #7C3AED, #6D28D9)' : '#E5E7EB', color: mediaGenPrompt.trim() && !mediaGenerating ? '#fff' : '#9CA3AF', fontSize: 14, fontWeight: 600, cursor: mediaGenPrompt.trim() && !mediaGenerating ? 'pointer' : 'default', width: '100%' }}
            >
              {mediaGenerating ? '⏳ AI生成中（约10-30秒）...' : '🤖 生成图片'}
            </button>
          </div>
        </div>

        {/* 下半: 手动上传图片(通栏) */}
        <div style={{ marginTop: 16, padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: C.textPrimary, marginBottom: 8 }}>📤 手动上传图片</div>
          <div style={{ padding: '20px 16px', borderRadius: 8, border: '2px dashed ' + C.border, textAlign: 'center', cursor: 'pointer', background: '#FAFAFA' }}
            onClick={() => { const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'image/*'; inp.onchange = async (ev) => { const f = (ev.target as HTMLInputElement).files?.[0]; if (!f || !coursewareId) return; if (f.size > 5 * 1024 * 1024) { setMediaMessage('❌ 图片不能超过5MB'); return } setMediaGenerating(true); setMediaMessage(''); try { const res = await uploadCWImage(coursewareId, pageNum, f); setMediaMessage('✅ 上传成功！'); setMediaAssets(prev => [makeAsset(coursewareId, { id: res.asset_id, oss_url: res.url, file_size: res.file_size, mime_type: res.mime_type }), ...prev]) } catch (e) { setMediaMessage('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误')) } finally { setMediaGenerating(false) } }; inp.click() }}
          >
            <div style={{ fontSize: 28, marginBottom: 6 }}>📷</div>
            <div style={{ fontSize: 13, color: C.textSecondary }}>点击选择图片</div>
            <div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>支持 JPG/PNG/WEBP/GIF/SVG，最大5MB</div>
          </div>
        </div>
        </>
      )}

      {/* 图片提示消息 */}
      {mediaTab === 'image' && mediaMessage && <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, background: mediaMessage.startsWith('❌') ? '#FEE2E2' : '#D1FAE5', color: mediaMessage.startsWith('❌') ? '#DC2626' : '#059669', fontSize: 13 }}>{mediaMessage}</div>}

      {/* 图片列表 */}
      {(mediaAssets.filter(a => a.asset_type === 'image').length > 0 || courseware.style_anchor_asset_id) && mediaTab === 'image' && (
        <ImageAssetList
          coursewareId={coursewareId} pageNum={pageNum} assets={mediaAssets} courseware={courseware}
          anchorSetting={anchorSetting} anchorClearing={anchorClearing}
          onSetAnchor={(assetId) => onSetAnchor(assetId, setMediaMessage)}
          onClearAnchor={() => onClearAnchor(setMediaMessage)}
          onPreview={setMediaPreviewUrl} onPickRef={setMediaRefUrl}
          onAssetUpdated={(updated) => setMediaAssets(prev => prev.map(a => a.id === updated.id ? updated : a))}
          onDeleteRequest={setDeleteTarget} notify={setMediaMessage}
        />
      )}

      {/* ==================== 视频Tab：AI分镜 → 视频列表 → 手动上传 ==================== */}

      {/* 视频生成区: VideoStoryboardPanel(AI分镜两步法) */}
      {mediaTab === 'video' && pageNum > 0 && (
        <VideoStoryboardPanel
          coursewareId={coursewareId} pageNum={pageNum}
          styleAnchorAssetId={courseware.style_anchor_asset_id}
          onAssetCreated={(asset) => setMediaAssets(prev => prev.some(a => a.id === asset.id) ? prev : [asset, ...prev])}
          onPreviewImage={(url) => setMediaPreviewUrl(url)}
        />
      )}

      {/* 视频列表 + 编辑器入口 */}
      {mediaTab === 'video' && pageNum > 0 && mediaAssets.filter(a => a.asset_type === 'video').length > 0 && (
        <VideoAssetList
          coursewareId={coursewareId} pageNum={pageNum} assets={mediaAssets}
          onOpenEditor={() => setEditorOpen(true)}
          onAssetUpdated={(updated) => setMediaAssets(prev => prev.map(a => a.id === updated.id ? updated : a))}
          onDeleteRequest={setDeleteTarget} notify={setMediaMessage}
        />
      )}

      {/* 批次5d: 视频手动上传区块 */}
      {mediaTab === 'video' && pageNum > 0 && (
        <div style={{ marginTop: 16, padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
          <div style={{ fontSize: 13, fontWeight: 600, color: '#7C3AED', marginBottom: 8 }}>📤 手动上传视频</div>
          <div
            style={{
              padding: '20px 16px', borderRadius: 8,
              border: '2px dashed ' + (videoUploading ? '#7C3AED' : C.border),
              textAlign: 'center',
              cursor: videoUploading ? 'default' : 'pointer',
              background: videoUploading ? 'rgba(124,58,237,0.04)' : '#FAFAFA',
            }}
            onClick={handleUploadVideo}
          >
            {videoUploading ? (
              <>
                <div style={{ fontSize: 14, marginBottom: 8, color: '#7C3AED', fontWeight: 600 }}>
                  ⏳ 上传中 {videoUploadProgress}%
                </div>
                <div style={{ width: '100%', height: 8, borderRadius: 4, background: '#E5E7EB', overflow: 'hidden' }}>
                  <div style={{
                    width: videoUploadProgress + '%', height: '100%', borderRadius: 4,
                    background: 'linear-gradient(90deg, #7C3AED, #6D28D9)',
                    transition: 'width 0.3s ease',
                  }} />
                </div>
                <div style={{ fontSize: 11, color: C.textMuted, marginTop: 6 }}>请稍候，大文件上传可能需要一些时间...</div>
              </>
            ) : (
              <>
                <div style={{ fontSize: 28, marginBottom: 6 }}>🎬</div>
                <div style={{ fontSize: 13, color: C.textSecondary }}>点击选择视频文件</div>
                <div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>支持 MP4/WebM/MOV/AVI，最大50MB</div>
              </>
            )}
          </div>
          <div style={{ marginTop: 10, padding: '8px 12px', borderRadius: 8, background: 'rgba(124,58,237,0.04)', border: '1px solid rgba(124,58,237,0.15)', fontSize: 12, color: '#6D28D9', lineHeight: 1.6 }}>
            💡 上传的视频会出现在上方列表中，可直接上云获取链接，也可以点「🎬 打开视频编辑器」进行裁剪、拼接、加字幕等操作。
          </div>
        </div>
      )}

      {/* 视频提示消息 */}
      {mediaTab === 'video' && mediaMessage && <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, background: mediaMessage.startsWith('❌') ? '#FEE2E2' : '#D1FAE5', color: mediaMessage.startsWith('❌') ? '#DC2626' : '#059669', fontSize: 13 }}>{mediaMessage}</div>}

      {/* ==================== 音频Tab：上传 + 列表 ==================== */}
      {mediaTab === 'audio' && pageNum > 0 && (
        <>
          <div style={{ padding: 16, borderRadius: 10, border: '1px solid ' + C.border, background: '#fff' }}>
            <div style={{ fontSize: 13, fontWeight: 600, color: '#0891B2', marginBottom: 8 }}>🎵 上传音频</div>
            <div style={{ padding: '20px 16px', borderRadius: 8, border: '2px dashed ' + C.border, textAlign: 'center', cursor: mediaGenerating ? 'default' : 'pointer', background: '#F0FDFA' }}
              onClick={() => {
                if (mediaGenerating) return
                const inp = document.createElement('input'); inp.type = 'file'; inp.accept = 'audio/*'
                inp.onchange = async (ev) => {
                  const f = (ev.target as HTMLInputElement).files?.[0]
                  if (!f || !coursewareId) return
                  if (f.size > 20 * 1024 * 1024) { setMediaMessage('❌ 音频文件不能超过20MB'); return }
                  setMediaGenerating(true); setMediaMessage('⏳ 上传音频中...')
                  try {
                    const res = await uploadCWAudio(coursewareId, pageNum, f)
                    setMediaMessage('✅ 音频上传成功！上云后可复制链接在微调中使用')
                    setMediaAssets(prev => [makeAsset(coursewareId, { id: res.asset_id, oss_url: res.url, asset_type: 'audio', file_size: res.file_size, mime_type: res.mime_type }), ...prev])
                  } catch (e) { setMediaMessage('❌ 上传失败: ' + (e instanceof Error ? e.message : '未知错误')) }
                  finally { setMediaGenerating(false) }
                }; inp.click()
              }}
            >
              <div style={{ fontSize: 28, marginBottom: 6 }}>🎵</div>
              <div style={{ fontSize: 13, color: '#0891B2' }}>点击选择音频文件</div>
              <div style={{ fontSize: 11, color: C.textMuted, marginTop: 4 }}>支持 MP3/WAV/OGG/AAC/FLAC/M4A，最大20MB</div>
            </div>
            <div style={{ marginTop: 10, padding: '10px 14px', borderRadius: 8, background: '#ECFDF5', border: '1px solid #A7F3D0', fontSize: 12, color: '#047857', lineHeight: 1.6 }}>
              💡 <b>使用方法：</b>上传音频 → 点「☁️上云」获取公网链接 → 复制链接 → 在微调指令中告诉AI把音频嵌入课件
              <br />示例微调指令：「在第3页添加一个音频播放器，音频地址是 [粘贴链接]」
            </div>
          </div>
          <AudioAssetList
            coursewareId={coursewareId} pageNum={pageNum} assets={mediaAssets}
            onAssetUpdated={(updated) => setMediaAssets(prev => prev.map(a => a.id === updated.id ? updated : a))}
            onDeleteRequest={setDeleteTarget} notify={setMediaMessage}
            onEditAudio={setAudioEditorAsset}
          />
        </>
      )}

      {/* 音频剪辑器弹窗 */}
      {audioEditorAsset && (
        <AudioEditorModal
          audio={audioEditorAsset}
          coursewareId={coursewareId}
          pageNum={pageNum}
          onClose={() => setAudioEditorAsset(null)}
          onExported={(newAsset) => {
            setMediaAssets(prev => [newAsset, ...prev])
            setMediaMessage('✅ 音频裁剪完成，新音频已添加到列表')
            setAudioEditorAsset(null)
          }}
        />
      )}

      {/* 音频提示消息 */}
      {mediaTab === 'audio' && mediaMessage && <div style={{ marginTop: 12, padding: '10px 14px', borderRadius: 8, background: mediaMessage.startsWith('❌') ? '#FEE2E2' : '#D1FAE5', color: mediaMessage.startsWith('❌') ? '#DC2626' : '#059669', fontSize: 13 }}>{mediaMessage}</div>}

      {/* 手动生成图片时按需打开自定义美术风格。 */}
      {styleStudioOpen && (
        <StyleStudioModal
          open
          coursewareId={coursewareId}
          coursewareTitle={courseware.title}
          onClose={() => setStyleStudioOpen(false)}
          onConfirmed={(result) => {
            clearManualStyleSelection()
            setStyleStudioOpen(false)
            onAnchorChanged?.(result)
            setMediaMessage('✅ 自定义画风已更新，后续生成图片将自动保持统一')
          }}
        />
      )}

      {/* 图片大图预览弹窗 */}
      {mediaPreviewUrl && (
        <div onClick={() => setMediaPreviewUrl('')} style={{ position: 'fixed', inset: 0, zIndex: 99990, background: 'rgba(0,0,0,0.75)', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: 'zoom-out' }}>
          <img src={mediaPreviewUrl} alt="大图预览" style={{ maxWidth: '90vw', maxHeight: '90vh', borderRadius: 12, boxShadow: '0 8px 40px rgba(0,0,0,0.5)' }} />
          <button onClick={(e) => { e.stopPropagation(); setMediaPreviewUrl('') }} style={{ position: 'absolute', top: 24, right: 24, width: 40, height: 40, borderRadius: '50%', border: 'none', background: 'rgba(255,255,255,0.2)', color: '#fff', fontSize: 20, cursor: 'pointer', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>✕</button>
        </div>
      )}

      {/* 危险删除红色确认弹窗 */}
      {deleteTarget && (
        <DangerConfirmModal
          title={deleteTarget.asset_type === 'video' ? '🗑 删除视频' : deleteTarget.asset_type === 'audio' ? '🗑 删除音频' : '🗑 删除图片'}
          message={buildDeleteWarning(deleteTarget)}
          confirmText={deleting ? '删除中...' : '确认删除'}
          busy={deleting}
          previewUrl={deleteTarget.asset_type === 'image' ? deleteTarget.oss_url : ''}
          onConfirm={handleConfirmDelete}
          onCancel={() => { if (!deleting) setDeleteTarget(null) }}
        />
      )}

      {/* 视频编辑器弹窗 */}
      {editorOpen && (
        <VideoEditorModal
          coursewareId={coursewareId}
          videos={mediaAssets.filter(a => a.asset_type === 'video' && a.oss_url).map(a => ({
            id: a.id, url: a.oss_url,
            label: a.generation_prompt || a.oss_url.split('/').pop()?.slice(0, 30) || '视频',
          }))}
          exporting={editorExporting}
          onClose={() => setEditorOpen(false)}
          onUploadVideo={async (file, onProgress) => {
            if (!coursewareId || pageNum <= 0) return null
            const res = await uploadCWVideo(coursewareId, pageNum, file, onProgress)
            const newAsset = makeAsset(coursewareId, {
              id: res.asset_id, oss_url: res.url, asset_type: 'video',
              generation_prompt: file.name, file_size: res.file_size, mime_type: res.mime_type,
            })
            setMediaAssets(prev => [newAsset, ...prev])
            return { id: res.asset_id, url: res.url, label: file.name }
          }}
          onExport={async (clips, options) => {
            if (!coursewareId || editorExporting) return
            setEditorExporting(true); setMediaMessage('')
            try {
              const res = await advancedConcatCWVideos(coursewareId, clips)
              let finalAssetId = res.asset_id
              let finalUrl = res.url
              let withSubtitle = false
              let withNarration = false
              const failNotes: string[] = []

              if (options?.burnSubtitle && options.subtitleId) {
                setMediaMessage('⏳ 成片已生成，正在烧录字幕（需重编码视频，约1-2分钟）...')
                try {
                  const burned = await burnInSubtitle(coursewareId, options.subtitleId, finalAssetId)
                  finalAssetId = burned.asset_id
                  finalUrl = burned.url
                  withSubtitle = true
                } catch (burnErr) {
                  failNotes.push('字幕烧录失败: ' + (burnErr instanceof Error ? burnErr.message : '未知错误'))
                }
              }

              if (options?.mixNarration && options.subtitleId) {
                setMediaMessage('⏳ 正在合成配音...')
                try {
                  const mix = await mixNarrationCWVideo(coursewareId, finalAssetId, options.subtitleId)
                  finalAssetId = mix.asset_id
                  finalUrl = mix.url
                  withNarration = true
                } catch (mixErr) {
                  failNotes.push('配音合成失败: ' + (mixErr instanceof Error ? mixErr.message : '未知错误'))
                }
              }

              const featTags = [withSubtitle ? '含字幕' : '', withNarration ? '含配音' : ''].filter(Boolean)
              const finalPrompt = '编辑导出' + (featTags.length > 0 ? '(' + featTags.join('+') + ')' : '') + ' ' + clips.length + '个片段'
              const finalMsg = failNotes.length > 0
                ? '⚠️ 成片已导出，但 ' + failNotes.join('；') + '（已保留最近一步可用版本）'
                : '✅ 导出完成：' + finalPrompt
              setMediaMessage(finalMsg)
              setMediaAssets(prev => [makeAsset(coursewareId, {
                id: finalAssetId, oss_url: finalUrl, asset_type: 'video',
                generation_prompt: finalPrompt, mime_type: 'video/mp4',
              }), ...prev])
              setEditorOpen(false)
            } catch (e) {
              setMediaMessage('❌ 导出失败: ' + (e instanceof Error ? e.message : ''))
            } finally { setEditorExporting(false) }
          }}
        />
      )}
    </div>
  )
}

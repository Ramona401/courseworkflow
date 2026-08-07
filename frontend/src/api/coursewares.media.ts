/**
 * 课件工坊 API —— 多媒体层 (coursewares.media.ts)
 *
 * 从原 coursewares.ts 拆出：图片生成/上传/管理、AI 写详细提示词（图片/视频）、
 * 视频生成与编辑（高级拼接/静音/音轨分离/上传/草稿）、字幕轨与 SRT/烧录、
 * TTS 配音、资产上云 OSS、课件离线包下载。
 * 经桶文件 coursewares.ts 透出，对外 import 路径不变。
 *
 * 普通图片积分幂等：
 *   - 每次调用generateCWImage代表一次教师主动生成操作；
 *   - 调用开始时生成一个UUID operation_id；
 *   - 同一个HTTP请求正文始终携带同一个operation_id；
 *   - Axios或网络层对同一请求进行重放时继续复用原请求正文；
 *   - 新一次教师点击会重新调用函数并获得新的operation_id。
 *
 * 视频锚点轮：
 *   generateCWVideo支持可选参数sourceFrameAssetId。
 *   两步流“先出首帧图→确认→生视频”时，把已确认首帧图的资产ID传入，
 *   后端据此写视频资产metadata溯源。
 */
import apiClient from './client'
import { extractData } from './coursewares.types'
import type {
  CoursewareAsset, GenerateImageResponse, UploadImageResponse, ImagePromptSuggestion, VideoStoryboardItem,
  GenerateVideoResponse, VideoStatusResponse, VideoClip, AdvancedConcatResponse,
  MuteVideoResponse, ExtractAudioResponse, VideoDraftItem,
  SubtitleSegment, CoursewareSubtitle, BurnInSubtitleResponse,
  TTSVoice, GenerateTTSRequest, GenerateTTSResponse, UploadToOSSResponse,
  SetStyleAnchorResult,
} from './coursewares.types'

// ==================== 图片生成+上传+管理 ====================

/**
 * 为一次图片生成业务操作创建安全UUID。
 *
 * 生产站点运行在HTTPS环境，现代浏览器均应提供crypto.randomUUID。
 * 不做时间戳或Math.random降级，避免低质量随机值削弱幂等键可靠性。
 */
function createCWImageOperationID(): string {
  const cryptoAPI = globalThis.crypto
  if (!cryptoAPI || typeof cryptoAPI.randomUUID !== 'function') {
    throw new Error('当前浏览器不支持安全UUID生成，请升级浏览器后重试')
  }

  return cryptoAPI.randomUUID()
}

/**
 * 为一次视频生成提交创建安全UUID。
 *
 * 同一次提交发生网络错误时，调用组件继续复用原UUID；
 * 收到asset_id后清空，下一次主动生成创建新UUID。
 */
export function createCWVideoOperationID(): string {
  const cryptoAPI = globalThis.crypto

  if (
    !cryptoAPI ||
    typeof cryptoAPI.randomUUID !== 'function'
  ) {
    throw new Error(
      '当前浏览器不支持安全UUID生成，请升级浏览器后重试',
    )
  }

  return cryptoAPI.randomUUID()
}

/**
 * AI生成普通课件图片。
 *
 * operationId通常不需要调用方传入：
 *   - 普通教师点击调用时，本函数自动创建一个新UUID；
 *   - 需要显式恢复同一业务操作的受控调用方，可传入原operationId；
 *   - 后端会再次按UUID格式校验并以此执行媒体计费幂等。
 */
export async function generateCWImage(
  coursewareId: string,
  pageNumber: number,
  prompt: string,
  placeholderId?: string,
  size?: string,
  refImageUrl?: string,
  operationId?: string,
): Promise<GenerateImageResponse> {
  const stableOperationID = operationId?.trim() || createCWImageOperationID()

  const body: Record<string, string> = {
    prompt,
    placeholder_id: placeholderId || '',
    size: size || '1920x1920',
    operation_id: stableOperationID,
  }

  if (refImageUrl) body.ref_image_url = refImageUrl

  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/generate-image`,
    body,
    { timeout: 60000 },
  )

  return extractData(resp)
}

/** 手动上传图片 */
export async function uploadCWImage(
  coursewareId: string,
  pageNumber: number,
  file: File,
  placeholderId?: string,
): Promise<UploadImageResponse> {
  const formData = new FormData()
  formData.append('file', file)

  if (placeholderId) {
    formData.append('placeholder_id', placeholderId)
  }

  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/upload-image`,
    formData,
    {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      timeout: 30000,
    },
  )

  return extractData(resp)
}

/** 获取页面图片列表 */
export async function listPageAssets(
  coursewareId: string,
  pageNumber: number,
): Promise<{ assets: CoursewareAsset[]; total: number }> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/pages/${pageNumber}/assets`,
  )

  return extractData(resp)
}

/** 获取课件全部媒体资产 */
export async function listCoursewareAssets(
  coursewareId: string,
): Promise<{ assets: CoursewareAsset[]; total: number }> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/assets`,
  )

  return extractData(resp)
}

/** 删除课件资产 */
export async function deleteCWAsset(
  coursewareId: string,
  assetId: string,
): Promise<void> {
  await apiClient.delete(
    `/coursewares/${coursewareId}/assets/${assetId}`,
  )
}

/** 将图片插入到页面HTML */
export async function insertImageToPage(
  coursewareId: string,
  pageNumber: number,
  assetId: string,
): Promise<{
  page_number: number
  html_content: string
  message: string
}> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/insert-image`,
    {
      asset_id: assetId,
    },
  )

  return extractData(resp)
}

// ==================== AI写详细提示词（图片/视频） ====================

/**
 * AI写详细生图提示词。
 *
 * AI读取本页配图需求，自主判断该页需要几张图，
 * 返回一条或多条建议，每条包含caption与prompt。
 */
export async function suggestImagePrompt(
  coursewareId: string,
  pageNumber: number,
): Promise<{ prompts: ImagePromptSuggestion[] }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/suggest-image-prompt`,
    {},
    {
      timeout: 60000,
    },
  )

  return extractData(resp)
}

/** AI写本页视频分镜数组 */
export async function suggestVideoPrompt(
  coursewareId: string,
  pageNumber: number,
): Promise<{ storyboards: VideoStoryboardItem[] }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/suggest-video-prompt`,
    {},
    {
      timeout: 60000,
    },
  )

  return extractData(resp)
}

// ==================== 物料存储 ====================

/**
 * 读取已存生图建议。
 *
 * 本接口不调用AI、不产生模型积分；
 * 数据库没有记录时prompts为空数组。
 */
export async function getStoredImageSuggestions(
  coursewareId: string,
  pageNumber: number,
): Promise<{ prompts: ImagePromptSuggestion[] }> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/pages/${pageNumber}/image-suggestions`,
  )

  return extractData(resp)
}

/**
 * 读取已存视频分镜。
 *
 * 本接口不调用AI；
 * 数据库没有记录时storyboards为空数组。
 */
export async function getStoredVideoStoryboards(
  coursewareId: string,
  pageNumber: number,
): Promise<{ storyboards: VideoStoryboardItem[] }> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/pages/${pageNumber}/video-storyboards`,
  )

  return extractData(resp)
}

/**
 * 保存老师编辑后的视频分镜。
 *
 * 传入空数组表示清除当前页面的已存分镜。
 */
export async function saveVideoStoryboards(
  coursewareId: string,
  pageNumber: number,
  storyboards: VideoStoryboardItem[],
): Promise<{
  message: string
  page_number: number
}> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/video-storyboards`,
    {
      storyboards,
    },
  )

  return extractData(resp)
}

// ==================== 视频生成 ====================

/**
 * AI生成视频。
 *
 * 视频使用异步任务协议，接口先返回asset_id与task_id，
 * 调用方随后使用queryVideoStatus轮询。
 *
 * sourceFrameAssetId非空时：
 *   - 后端重新读取并校验首帧资产属于当前课件且为图片；
 *   - 只使用数据库中的可信图片URL；
 *   - 视频资产metadata记录首帧资产血缘。
 */
export async function generateCWVideo(
  coursewareId: string,
  pageNumber: number,
  prompt: string,
  refImageUrl?: string,
  sourceFrameAssetId?: string,
  operationId?: string,
): Promise<GenerateVideoResponse> {
  const stableOperationID =
    operationId?.trim() ||
    createCWVideoOperationID()

  const body: Record<string, string> = {
    prompt,
    operation_id: stableOperationID,
  }

  if (refImageUrl) {
    body.ref_image_url = refImageUrl
  }

  if (sourceFrameAssetId) {
    body.source_frame_asset_id = sourceFrameAssetId
  }

  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/generate-video`,
    body,
    {
      timeout: 30000,
    },
  )

  return extractData(resp)
}

/** 查询视频生成任务状态 */
export async function queryVideoStatus(
  coursewareId: string,
  assetId: string,
): Promise<VideoStatusResponse> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/assets/${assetId}/video-status`,
  )

  return extractData(resp)
}

// ==================== 视频高级拼接 ====================

/** 高级视频拼接，支持每段独立裁剪和转场效果 */
export async function advancedConcatCWVideos(
  coursewareId: string,
  clips: VideoClip[],
): Promise<AdvancedConcatResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/advanced-concat`,
    {
      clips,
    },
    {
      timeout: 120000,
    },
  )

  return extractData(resp)
}

// ==================== 配音混入成片 ====================

/** 配音混音响应 */
export interface MixNarrationResponse {
  asset_id: string
  url: string
  duration: string
  narration_count: number
  skipped_count: number
  message: string
}

/** 把字幕轨中已生成的TTS配音按时间轴混入指定视频 */
export async function mixNarrationCWVideo(
  coursewareId: string,
  assetId: string,
  subtitleId: string,
  gain?: number,
): Promise<MixNarrationResponse> {
  const body: Record<string, unknown> = {
    asset_id: assetId,
    subtitle_id: subtitleId,
  }

  if (gain && gain !== 1.0) {
    body.gain = gain
  }

  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/mix-narration`,
    body,
    {
      timeout: 180000,
    },
  )

  return extractData(resp)
}

// ==================== 视频静音与音轨分离 ====================

/** 视频静音，生成新的静音视频 */
export async function muteCWVideo(
  coursewareId: string,
  assetId: string,
): Promise<MuteVideoResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/mute`,
    {
      asset_id: assetId,
    },
    {
      timeout: 60000,
    },
  )

  return extractData(resp)
}

/** 从视频提取音频 */
export async function extractCWAudio(
  coursewareId: string,
  assetId: string,
): Promise<ExtractAudioResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/extract-audio`,
    {
      asset_id: assetId,
    },
    {
      timeout: 60000,
    },
  )

  return extractData(resp)
}

// ==================== 视频手动上传 ====================

/** 手动上传视频文件到课件 */
export async function uploadCWVideo(
  coursewareId: string,
  pageNumber: number,
  file: File,
  onProgress?: (pct: number) => void,
): Promise<{
  asset_id: string
  url: string
  file_name: string
  file_size: number
  mime_type: string
}> {
  const formData = new FormData()
  formData.append('file', file)

  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/upload-video`,
    formData,
    {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      timeout: 120000,
      onUploadProgress: event => {
        if (onProgress && event.total) {
          const percentage = Math.round(
            (event.loaded / event.total) * 100,
          )

          onProgress(
            Math.min(100, percentage),
          )
        }
      },
    },
  )

  return extractData(resp)
}

// ==================== 音频手动上传 ====================

/** 手动上传音频文件到课件 */
export async function uploadCWAudio(
  coursewareId: string,
  pageNumber: number,
  file: File,
  onProgress?: (pct: number) => void,
): Promise<{
  asset_id: string
  url: string
  file_name: string
  file_size: number
  mime_type: string
}> {
  const formData = new FormData()
  formData.append('file', file)

  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/upload-audio`,
    formData,
    {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
      timeout: 60000,
      onUploadProgress: event => {
        if (onProgress && event.total) {
          const percentage = Math.round(
            (event.loaded / event.total) * 100,
          )

          onProgress(
            Math.min(100, percentage),
          )
        }
      },
    },
  )

  return extractData(resp)
}

// ==================== 视频编辑器草稿 ====================

export async function listVideoDrafts(
  coursewareId: string,
): Promise<{
  drafts: VideoDraftItem[]
  total: number
}> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/video-drafts`,
  )

  return extractData(resp)
}

export async function saveVideoDraft(
  coursewareId: string,
  data: {
    name: string
    clips_data: any
    clip_count: number
  },
): Promise<{
  id: string
  created_at: string
  message: string
}> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/video-drafts`,
    data,
  )

  return extractData(resp)
}

export async function deleteVideoDraft(
  coursewareId: string,
  draftId: string,
): Promise<void> {
  await apiClient.delete(
    `/coursewares/${coursewareId}/video-drafts/${draftId}`,
  )
}

// ==================== 字幕轨 API ====================

/** 创建或更新字幕轨 */
export async function upsertSubtitle(
  coursewareId: string,
  data: {
    scope_type: string
    scope_id?: string | null
    language: string
    segments: string
    style_config?: string | null
    tts_config?: string | null
  },
): Promise<CoursewareSubtitle> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/subtitles`,
    data,
  )

  return extractData(resp)
}

/** 查询字幕轨列表 */
export async function listSubtitles(
  coursewareId: string,
  scopeType?: string,
  scopeId?: string,
): Promise<CoursewareSubtitle[]> {
  const params: Record<string, string> = {}

  if (scopeType) {
    params.scope_type = scopeType
  }

  if (scopeId) {
    params.scope_id = scopeId
  }

  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/subtitles`,
    {
      params,
    },
  )

  return extractData(resp)
}

/** 删除字幕轨 */
export async function deleteSubtitle(
  coursewareId: string,
  subtitleId: string,
): Promise<void> {
  await apiClient.delete(
    `/coursewares/${coursewareId}/subtitles/${subtitleId}`,
  )
}

/** 前端本地生成SRT文件并触发下载 */
export function exportSubtitleSRTLocal(
  segments: SubtitleSegment[],
  filename?: string,
): void {
  const formatTime = (seconds: number): string => {
    const safeSeconds = seconds < 0 ? 0 : seconds
    const milliseconds = Math.round(safeSeconds * 1000)
    const hours = Math.floor(milliseconds / 3600000)
    const minutes = Math.floor(
      (milliseconds % 3600000) / 60000,
    )
    const secs = Math.floor(
      (milliseconds % 60000) / 1000,
    )
    const millis = milliseconds % 1000

    return [
      String(hours).padStart(2, '0'),
      String(minutes).padStart(2, '0'),
      String(secs).padStart(2, '0'),
    ].join(':') + ',' + String(millis).padStart(3, '0')
  }

  const srt = segments
    .map((segment, index) => (
      `${index + 1}\n` +
      `${formatTime(segment.start_sec)} --> ${formatTime(segment.end_sec)}\n` +
      `${segment.text}\n`
    ))
    .join('\n')

  const blob = new Blob(
    [srt],
    {
      type: 'text/plain;charset=utf-8',
    },
  )

  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')

  anchor.href = url
  anchor.download = filename || 'subtitle.srt'

  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)

  URL.revokeObjectURL(url)
}

/** FFmpeg硬字幕烧录 */
export async function burnInSubtitle(
  coursewareId: string,
  subtitleId: string,
  videoAssetId: string,
): Promise<BurnInSubtitleResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/subtitles/${subtitleId}/burn-in`,
    {
      video_asset_id: videoAssetId,
    },
    {
      timeout: 120000,
    },
  )

  return extractData(resp)
}

// ==================== TTS配音 API ====================

/** 获取可用TTS音色列表 */
export async function listTTSVoices(
  language?: string,
): Promise<{
  voices: TTSVoice[]
  total: number
}> {
  const params: Record<string, string> = {}

  if (language) {
    params.language = language
  }

  const resp = await apiClient.get(
    '/tts-voices',
    {
      params,
    },
  )

  return extractData(resp)
}

/**
 * 为一次字幕TTS批次创建安全UUID。
 *
 * 相同批次发生网络错误或服务端等待恢复时，
 * 调用组件继续复用原UUID。
 */
export function createSubtitleTTSOperationID(): string {
  const cryptoAPI = globalThis.crypto

  if (
    !cryptoAPI ||
    typeof cryptoAPI.randomUUID !== 'function'
  ) {
    throw new Error(
      '当前浏览器不支持安全UUID生成，请升级浏览器后重试',
    )
  }

  return cryptoAPI.randomUUID()
}

/** 批量生成字幕TTS配音 */
export async function generateSubtitleTTS(
  coursewareId: string,
  subtitleId: string,
  voice: string,
  speed?: number,
  segmentIds?: string[],
  operationId?: string,
): Promise<GenerateTTSResponse> {
  const stableOperationID =
    operationId?.trim() ||
    createSubtitleTTSOperationID()

  const body: GenerateTTSRequest = {
    voice,
    operation_id:
      stableOperationID,
  }

  if (
    speed &&
    speed !== 1.0
  ) {
    body.speed = speed
  }

  if (
    segmentIds &&
    segmentIds.length > 0
  ) {
    body.segment_ids =
      segmentIds
  }

  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/subtitles/${subtitleId}/generate-tts`,
    body,
    {
      timeout: 300000,
    },
  )

  return extractData(resp)
}

// ==================== 上传资产到阿里云OSS ====================

/** 将课件资产上传到阿里云OSS */
export async function uploadAssetToOSS(
  coursewareId: string,
  assetId: string,
): Promise<UploadToOSSResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/assets/${assetId}/upload-oss`,
    {},
    {
      timeout: 120000,
    },
  )

  return extractData(resp)
}

// ==================== 风格锚点 ====================

/**
 * 设置课件风格锚点。
 *
 * 后端内部完成：
 *   校验资产归属 → 解析公网URL → 多模态提取VAOCI → 落库。
 */
export async function setStyleAnchor(
  coursewareId: string,
  assetId: string,
): Promise<SetStyleAnchorResult> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/style-anchor`,
    {
      asset_id: assetId,
    },
    {
      timeout: 60000,
    },
  )

  return extractData(resp)
}

/** 清除课件风格锚点 */
export async function clearStyleAnchor(
  coursewareId: string,
): Promise<void> {
  await apiClient.delete(
    `/coursewares/${coursewareId}/style-anchor`,
  )
}

// ==================== 离线打包下载 ====================

/**
 * 下载课件离线包。
 *
 * 使用原生fetch获取二进制流，
 * 避免Axios响应拦截器对非JSON响应进行处理。
 */
export async function downloadCoursewareBundle(
  coursewareId: string,
  title?: string,
): Promise<void> {
  const token = localStorage.getItem('token') || ''

  const response = await fetch(
    `/api/v1/coursewares/${coursewareId}/export-bundle`,
    {
      method: 'GET',
      headers: {
        Authorization: `Bearer ${token}`,
      },
    },
  )

  if (!response.ok) {
    let message = `下载失败(HTTP ${response.status})`

    try {
      const errorBody = await response.json()
      if (errorBody && errorBody.message) {
        message = errorBody.message
      }
    } catch {
      // 非JSON错误响应使用默认文案。
    }

    if (response.status === 401) {
      localStorage.removeItem('token')
      localStorage.removeItem('user')

      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }

    throw new Error(message)
  }

  let filename = title?.trim() || 'courseware'
  filename = filename.replace(
    /[/\\:*?"<>|]/g,
    '_',
  ) + '.zip'

  const contentDisposition =
    response.headers.get('Content-Disposition') || ''

  const filenameMatch = contentDisposition.match(
    /filename\*=UTF-8''([^;]+)/i,
  )

  if (filenameMatch && filenameMatch[1]) {
    try {
      filename = decodeURIComponent(
        filenameMatch[1],
      )
    } catch {
      // 解码失败时继续使用前端构造的文件名。
    }
  }

  const blob = await response.blob()
  const url = URL.createObjectURL(blob)
  const anchor = document.createElement('a')

  anchor.href = url
  anchor.download = filename

  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)

  URL.revokeObjectURL(url)
}

// ==================== 音频裁剪 ====================

/** 音频裁剪响应 */
export interface TrimAudioResponse {
  asset_id: string
  url: string
  duration: string
  file_name: string
  file_size: number
  mime_type: string
  message: string
}

/** 裁剪指定音频资产 */
export async function trimCWAudio(
  coursewareId: string,
  assetId: string,
  startSec: number,
  endSec: number,
): Promise<TrimAudioResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/trim-audio`,
    {
      asset_id: assetId,
      start_sec: startSec,
      end_sec: endSec,
    },
    {
      timeout: 60000,
    },
  )

  return extractData(resp)
}

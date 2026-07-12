/**
 * 课件工坊 API —— 多媒体层 (coursewares.media.ts)
 *
 * 从原 coursewares.ts 拆出：图片生成/上传/管理、AI 写详细提示词（图片/视频）、
 * 视频生成与编辑（高级拼接/静音/音轨分离/上传/草稿）、字幕轨与 SRT/烧录、
 * TTS 配音、资产上云 OSS、课件离线包下载。
 * 经桶文件 coursewares.ts 透出，对外 import 路径不变。
 *
 * 视频锚点轮(本轮)：generateCWVideo 新增可选参数 sourceFrameAssetId——
 *   两步流"先出首帧图→确认→生视频"时，把已确认首帧图的资产ID传入，
 *   后端据此写视频资产 metadata 溯源（{"source_frame_asset_id":"..."}）。
 *   不传则为旧的直接生视频，无溯源。
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

/** v0.42: AI生成图片（调用豆包Seedream API） */
export async function generateCWImage(
  coursewareId: string,
  pageNumber: number,
  prompt: string,
  placeholderId?: string,
  size?: string,
  refImageUrl?: string,
): Promise<GenerateImageResponse> {
  const body: Record<string, string> = {
    prompt,
    placeholder_id: placeholderId || '',
    size: size || '1920x1920',
  }
  if (refImageUrl) body.ref_image_url = refImageUrl
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/generate-image`,
    body,
    { timeout: 60000 },
  )
  return extractData(resp)
}

/** v0.42: 手动上传图片 */
export async function uploadCWImage(
  coursewareId: string,
  pageNumber: number,
  file: File,
  placeholderId?: string,
): Promise<UploadImageResponse> {
  const formData = new FormData()
  formData.append('file', file)
  if (placeholderId) formData.append('placeholder_id', placeholderId)
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/upload-image`,
    formData,
    { headers: { 'Content-Type': 'multipart/form-data' }, timeout: 30000 },
  )
  return extractData(resp)
}

/** v0.42: 获取页面图片列表 */
export async function listPageAssets(
  coursewareId: string,
  pageNumber: number,
): Promise<{ assets: CoursewareAsset[]; total: number }> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/pages/${pageNumber}/assets`,
  )
  return extractData(resp)
}

/** v0.42: 获取课件全部图片 */
export async function listCoursewareAssets(
  coursewareId: string,
): Promise<{ assets: CoursewareAsset[]; total: number }> {
  const resp = await apiClient.get(`/coursewares/${coursewareId}/assets`)
  return extractData(resp)
}

/** v0.42: 删除图片资产 */
export async function deleteCWAsset(
  coursewareId: string,
  assetId: string,
): Promise<void> {
  await apiClient.delete(`/coursewares/${coursewareId}/assets/${assetId}`)
}

/** v0.42: 将图片插入到页面HTML */
export async function insertImageToPage(
  coursewareId: string,
  pageNumber: number,
  assetId: string,
): Promise<{ page_number: number; html_content: string; message: string }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/insert-image`,
    { asset_id: assetId },
  )
  return extractData(resp)
}

// ==================== 批次4c+: AI 写详细提示词（图片/视频）====================

/**
 * 批次4c+/图片多提示词: AI 写详细生图提示词。
 * AI 读本页配图需求自主判断该页要几张图, 返回一条或多条建议(每条含 caption + prompt)。
 * 即使只有一张图, prompts 也是长度为 1 的数组。
 */
export async function suggestImagePrompt(
  coursewareId: string,
  pageNumber: number,
): Promise<{ prompts: ImagePromptSuggestion[] }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/suggest-image-prompt`,
    {},
    { timeout: 60000 },
  )
  return extractData(resp)
}

/** 视频分镜(本轮): AI 写本页视频分镜数组, 每镜含 scene/image_prompt/video_prompt/narration */
export async function suggestVideoPrompt(
  coursewareId: string,
  pageNumber: number,
): Promise<{ storyboards: VideoStoryboardItem[] }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/suggest-video-prompt`,
    {},
    { timeout: 60000 },
  )
  return extractData(resp)
}

// ==================== 物料存储: 读已存建议/分镜 + 保存分镜（省token先读库）====================

/** 物料存储: 读已存生图建议(不调AI、不计token); 库里没有时 prompts 为空数组, 调用方据此回退调 suggestImagePrompt */
export async function getStoredImageSuggestions(
  coursewareId: string,
  pageNumber: number,
): Promise<{ prompts: ImagePromptSuggestion[] }> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/pages/${pageNumber}/image-suggestions`,
  )
  return extractData(resp)
}

/** 物料存储: 读已存视频分镜(不调AI); 库里没有时 storyboards 为空数组 */
export async function getStoredVideoStoryboards(
  coursewareId: string,
  pageNumber: number,
): Promise<{ storyboards: VideoStoryboardItem[] }> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/pages/${pageNumber}/video-storyboards`,
  )
  return extractData(resp)
}

/** 物料存储: 保存视频分镜(老师手动编辑/拆镜结果落库); 传空数组等价清空 */
export async function saveVideoStoryboards(
  coursewareId: string,
  pageNumber: number,
  storyboards: VideoStoryboardItem[],
): Promise<{ message: string; page_number: number }> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/video-storyboards`,
    { storyboards },
  )
  return extractData(resp)
}

// ==================== v0.42.1 视频生成 ====================

/**
 * v0.42.1: AI生成视频（异步提交任务，返回task_id供轮询）
 *
 * 视频锚点轮(本轮)：新增可选参数 sourceFrameAssetId。
 *   - 两步流"先出首帧图→确认→生视频"：把首帧图URL作 refImageUrl(图生视频锁风格人物)，
 *     首帧图资产ID作 sourceFrameAssetId(后端写 metadata 溯源)。
 *   - 旧的直接文字生视频：两参数都不传。
 */
export async function generateCWVideo(
  coursewareId: string,
  pageNumber: number,
  prompt: string,
  refImageUrl?: string,
  sourceFrameAssetId?: string,
): Promise<GenerateVideoResponse> {
  const body: Record<string, string> = { prompt }
  if (refImageUrl) body.ref_image_url = refImageUrl
  if (sourceFrameAssetId) body.source_frame_asset_id = sourceFrameAssetId
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/generate-video`,
    body,
    { timeout: 30000 },
  )
  return extractData(resp)
}

/** v0.42.1: 查询视频生成任务状态（前端轮询直到uploaded或failed） */
export async function queryVideoStatus(
  coursewareId: string,
  assetId: string,
): Promise<VideoStatusResponse> {
  const resp = await apiClient.get(
    `/coursewares/${coursewareId}/assets/${assetId}/video-status`,
  )
  return extractData(resp)
}

// ==================== v0.42.2 视频高级拼接 ====================

/** v0.42.2: 高级视频拼接（支持每段独立裁剪+转场效果） */
export async function advancedConcatCWVideos(
  coursewareId: string,
  clips: VideoClip[],
): Promise<AdvancedConcatResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/advanced-concat`,
    { clips },
    { timeout: 120000 },
  )
  return extractData(resp)
}

// ==================== S-V1 配音混入成片 ====================

/** S-V1: 配音混音响应（后端 MixNarrationResponse） */
export interface MixNarrationResponse {
  asset_id: string
  url: string
  duration: string
  narration_count: number
  skipped_count: number
  message: string
}

/** S-V1: 把字幕轨中已生成的TTS配音按时间轴混入指定视频，产出新视频资产 */
export async function mixNarrationCWVideo(
  coursewareId: string,
  assetId: string,
  subtitleId: string,
  gain?: number,
): Promise<MixNarrationResponse> {
  const body: Record<string, unknown> = { asset_id: assetId, subtitle_id: subtitleId }
  if (gain && gain !== 1.0) body.gain = gain
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/mix-narration`,
    body,
    { timeout: 180000 }, // 混音含ffmpeg处理，给足3分钟
  )
  return extractData(resp)
}

// ==================== v0.42.4 视频静音 + 音轨分离 ====================

/** v0.42.4: 视频静音（去除音轨，生成新的静音视频） */
export async function muteCWVideo(
  coursewareId: string,
  assetId: string,
): Promise<MuteVideoResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/mute`,
    { asset_id: assetId },
    { timeout: 60000 },
  )
  return extractData(resp)
}

/** v0.42.4: 音轨分离（从视频提取音频为MP3） */
export async function extractCWAudio(
  coursewareId: string,
  assetId: string,
): Promise<ExtractAudioResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/extract-audio`,
    { asset_id: assetId },
    { timeout: 60000 },
  )
  return extractData(resp)
}

// ==================== v0.42.5 视频手动上传 ====================

/** v0.42.5: 手动上传视频文件到课件(mp4/webm/mov ≤50MB) */
export async function uploadCWVideo(
  coursewareId: string,
  pageNumber: number,
  file: File,
  onProgress?: (pct: number) => void,
): Promise<{ asset_id: string; url: string; file_name: string; file_size: number; mime_type: string }> {
  const formData = new FormData()
  formData.append('file', file)
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/upload-video`,
    formData,
    {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 120000,
      // v0.42.6+ P1.1: 利用 axios 原生上传进度回调
      onUploadProgress: (e) => {
        if (onProgress && e.total) {
          const pct = Math.round((e.loaded / e.total) * 100)
          onProgress(Math.min(100, pct))
        }
      },
    },
  )
  return extractData(resp)
}

// ==================== 音频手动上传 ====================

/** 手动上传音频文件到课件(mp3/wav/ogg/aac/flac/m4a ≤20MB) */
export async function uploadCWAudio(
  coursewareId: string,
  pageNumber: number,
  file: File,
  onProgress?: (pct: number) => void,
): Promise<{ asset_id: string; url: string; file_name: string; file_size: number; mime_type: string }> {
  const formData = new FormData()
  formData.append('file', file)
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/pages/${pageNumber}/upload-audio`,
    formData,
    {
      headers: { 'Content-Type': 'multipart/form-data' },
      timeout: 60000,
      onUploadProgress: (e) => {
        if (onProgress && e.total) {
          const pct = Math.round((e.loaded / e.total) * 100)
          onProgress(Math.min(100, pct))
        }
      },
    },
  )
  return extractData(resp)
}

// ==================== v0.42.5 视频编辑器草稿(服务器端多版本) ====================

export async function listVideoDrafts(coursewareId: string): Promise<{ drafts: VideoDraftItem[]; total: number }> {
  const resp = await apiClient.get(`/coursewares/${coursewareId}/video-drafts`)
  return extractData(resp)
}

export async function saveVideoDraft(coursewareId: string, data: {
  name: string; clips_data: any; clip_count: number
}): Promise<{ id: string; created_at: string; message: string }> {
  const resp = await apiClient.post(`/coursewares/${coursewareId}/video-drafts`, data)
  return extractData(resp)
}

export async function deleteVideoDraft(coursewareId: string, draftId: string): Promise<void> {
  await apiClient.delete(`/coursewares/${coursewareId}/video-drafts/${draftId}`)
}

// ==================== v0.42.8 字幕轨 API ====================

/** v0.42.8: 创建/更新字幕轨（UPSERT by courseware+scope+language） */
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
  const resp = await apiClient.post(`/coursewares/${coursewareId}/subtitles`, data)
  return extractData(resp)
}

/** v0.42.8: 查询字幕轨列表 */
export async function listSubtitles(
  coursewareId: string,
  scopeType?: string,
  scopeId?: string,
): Promise<CoursewareSubtitle[]> {
  const params: Record<string, string> = {}
  if (scopeType) params.scope_type = scopeType
  if (scopeId) params.scope_id = scopeId
  const resp = await apiClient.get(`/coursewares/${coursewareId}/subtitles`, { params })
  return extractData(resp)
}

/** v0.42.8: 删除字幕轨 */
export async function deleteSubtitle(
  coursewareId: string,
  subtitleId: string,
): Promise<void> {
  await apiClient.delete(`/coursewares/${coursewareId}/subtitles/${subtitleId}`)
}

/** v0.42.8: 前端本地生成 SRT 文件并触发下载（避免 axios 拦截器对纯文本的处理问题） */
export function exportSubtitleSRTLocal(segments: SubtitleSegment[], filename?: string): void {
  // 格式化 SRT 时间码: HH:MM:SS,mmm
  const fmtTime = (sec: number): string => {
    if (sec < 0) sec = 0
    const ms = Math.round(sec * 1000)
    const h = Math.floor(ms / 3600000)
    const m = Math.floor((ms % 3600000) / 60000)
    const s = Math.floor((ms % 60000) / 1000)
    const mill = ms % 1000
    return `${String(h).padStart(2,'0')}:${String(m).padStart(2,'0')}:${String(s).padStart(2,'0')},${String(mill).padStart(3,'0')}`
  }
  const srt = segments.map((seg, i) =>
    `${i + 1}\n${fmtTime(seg.start_sec)} --> ${fmtTime(seg.end_sec)}\n${seg.text}\n`
  ).join('\n')
  const blob = new Blob([srt], { type: 'text/plain;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url; a.download = filename || 'subtitle.srt'
  document.body.appendChild(a); a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

/** v0.42.8: FFmpeg 硬字幕烧录（生成新视频） */
export async function burnInSubtitle(
  coursewareId: string,
  subtitleId: string,
  videoAssetId: string,
): Promise<BurnInSubtitleResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/subtitles/${subtitleId}/burn-in`,
    { video_asset_id: videoAssetId },
    { timeout: 120000 },
  )
  return extractData(resp)
}

// ==================== v0.42.9 TTS 配音 API ====================

/** v0.42.9: 获取可用 TTS 音色列表 */
export async function listTTSVoices(language?: string): Promise<{ voices: TTSVoice[]; total: number }> {
  const params: Record<string, string> = {}
  if (language) params.language = language
  const resp = await apiClient.get('/tts-voices', { params })
  return extractData(resp)
}

/** v0.42.9: 批量生成字幕 TTS 配音 */
export async function generateSubtitleTTS(
  coursewareId: string,
  subtitleId: string,
  voice: string,
  speed?: number,
  segmentIds?: string[],
): Promise<GenerateTTSResponse> {
  const body: GenerateTTSRequest = { voice }
  if (speed && speed !== 1.0) body.speed = speed
  if (segmentIds && segmentIds.length > 0) body.segment_ids = segmentIds
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/subtitles/${subtitleId}/generate-tts`,
    body,
    { timeout: 300000 }, // 5分钟超时（批量TTS可能较慢）
  )
  return extractData(resp)
}

// ==================== v0.42.10 上传资产到阿里云OSS ====================

/** v0.42.10: 将课件资产（图片/视频/音频）上传到阿里云OSS，返回公网URL */
export async function uploadAssetToOSS(
  coursewareId: string,
  assetId: string,
): Promise<UploadToOSSResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/assets/${assetId}/upload-oss`,
    {},
    { timeout: 120000 }, // 大视频上传可能较慢，2分钟超时
  )
  return extractData(resp)
}

// ==================== 风格锚点（VAOCI 课程级风格一致性，轮3）====================

/**
 * 设置课件风格锚点（一步式同步）。
 * 后端内部：校验资产归属 → 取公网URL → 多模态读图提取VAOCI → 落库。
 * 提取VAOCI为多模态调用，耗时数秒到十几秒，故 timeout 给足 60s。
 */
export async function setStyleAnchor(
  coursewareId: string,
  assetId: string,
): Promise<SetStyleAnchorResult> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/style-anchor`,
    { asset_id: assetId },
    { timeout: 60000 },
  )
  return extractData(resp)
}

/** 清除课件风格锚点 */
export async function clearStyleAnchor(coursewareId: string): Promise<void> {
  await apiClient.delete(`/coursewares/${coursewareId}/style-anchor`)
}

// ==================== 离线打包下载 ====================

/**
 * 下载课件离线包(zip)
 * 用原生 fetch 获取二进制流，绕开 axios 响应拦截器对非 JSON(blob) 的处理。
 * 鉴权头与 client.ts 保持一致：Authorization: Bearer <token>
 */
export async function downloadCoursewareBundle(coursewareId: string, title?: string): Promise<void> {
  const token = localStorage.getItem('token') || ''
  const resp = await fetch(`/api/v1/coursewares/${coursewareId}/export-bundle`, {
    method: 'GET',
    headers: { Authorization: `Bearer ${token}` },
  })
  if (!resp.ok) {
    // 尝试解析后端 JSON 错误信息
    let msg = `下载失败(HTTP ${resp.status})`
    try {
      const j = await resp.json()
      if (j && j.message) msg = j.message
    } catch { /* 非 JSON 错误体，忽略 */ }
    if (resp.status === 401) {
      // 登录态失效，与 client.ts 行为保持一致
      localStorage.removeItem('token')
      localStorage.removeItem('user')
      if (window.location.pathname !== '/login') window.location.href = '/login'
    }
    throw new Error(msg)
  }
  // 文件名：优先用后端 Content-Disposition，回退到 title.zip
  let filename = (title ? title.trim() : '') || 'courseware'
  filename = filename.replace(/[/\\:*?"<>|]/g, '_') + '.zip'
  const cd = resp.headers.get('Content-Disposition') || ''
  const m = cd.match(/filename\*=UTF-8''([^;]+)/i)
  if (m && m[1]) {
    try { filename = decodeURIComponent(m[1]) } catch { /* 解码失败用回退名 */ }
  }
  // 触发浏览器下载
  const blob = await resp.blob()
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
}

// ==================== 音频裁剪（课件音频剪辑器专用） ====================

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

/**
 * 音频裁剪：截取指定起止时间段，后端FFmpeg -c copy不重编码裁剪，生成新音频资产
 * 路由: POST /api/v1/coursewares/{id}/videos/trim-audio
 */
export async function trimCWAudio(
  coursewareId: string,
  assetId: string,
  startSec: number,
  endSec: number,
): Promise<TrimAudioResponse> {
  const resp = await apiClient.post(
    `/coursewares/${coursewareId}/videos/trim-audio`,
    { asset_id: assetId, start_sec: startSec, end_sec: endSec },
    { timeout: 60000 },
  )
  return extractData(resp)
}

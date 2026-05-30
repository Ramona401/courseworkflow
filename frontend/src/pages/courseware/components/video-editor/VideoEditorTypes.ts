/**
 * VideoEditorTypes.ts — 视频编辑器类型定义集中管理
 *
 * 从 VideoEditorModal.tsx v6.1 拆分而来(v0.42.7 多轨道音频混音 + 时间轴原地裁剪)
 *
 * 包含:
 *   - EditorClip       片段数据结构(含 v0.42.7 新增 audioMuted/audioVolume 独立混音字段)
 *   - VideoEditorModalProps 主组件 Props 接口
 *   - TrackDef         多轨道定义结构
 *   - TransDef         转场定义结构
 *   - VideoAssetInfo   素材库视频简化结构
 */

// ==================== 片段数据 ====================
export interface EditorClip {
  /** 片段唯一标识(对应 mediaAsset.id) */
  id: string
  /** 视频 URL(可能是本地或公网) */
  url: string
  /** 显示名称(对应 mediaAsset.generation_prompt 或文件名) */
  label: string
  /** 视频原始时长(秒) */
  duration: number
  /** 裁剪入点(秒) */
  trimStart: number
  /** 裁剪出点(秒) */
  trimEnd: number
  /** 转场类型 key */
  transition: string
  /** 转场时长(秒) */
  transDur: number
  /** 是否为分离音轨后的静音视频 */
  muted?: boolean
  /** 原始视频 id(分离前) */
  originalId?: string
  /** 原始视频 url(分离前) */
  originalUrl?: string
  /** blob URL 缓存(优先用于播放,减轻网络压力) */
  blobUrl?: string
  /** 分离后的音频 URL */
  audioUrl?: string
  /** 分离后的音频时长字符串 */
  audioDuration?: string
  /** 分离后的音频文件大小(字节) */
  audioFileSize?: number
  /** v0.42.6 多轨道: 片段所属轨道类型,默认 video 向后兼容旧草稿 */
  trackType?: 'video' | 'audio' | 'subtitle'
  /**
   * v0.42.7 新增: 独立音轨静音状态
   * 默认 false(不静音);为 true 时即使有 audioUrl 也强制 audio.volume=0
   * 与 muted 不同: muted 指视频本身静音状态(分离后的静音 mp4),
   * audioMuted 是用户在编辑器里对【音频轨片段】的独立静音开关
   */
  audioMuted?: boolean
  /**
   * v0.42.7 新增: 独立音轨音量 (0.0 ~ 1.0)
   * 默认 1.0(原始音量);用户可拖滑块降到 0.0 ~ 1.0
   * 仅当 clip.audioUrl 存在时生效,通过旁路 <audio> 元素的 volume 实时控制
   */
  audioVolume?: number
}

// ==================== 主组件 Props ====================
/** 素材库视频项(简化结构,只暴露给 VideoEditorModal 必需字段) */
export interface VideoAssetInfo {
  id: string
  url: string
  label: string
}

export interface VideoEditorModalProps {
  /** 素材库视频列表 */
  videos: VideoAssetInfo[]
  /** 课件 ID(用于音轨分离、上传视频、保存草稿) */
  coursewareId?: string
  /** 关闭编辑器回调 */
  onClose: () => void
  /** 导出最终拼接结果回调 */
  onExport: (clips: { asset_id: string; start_sec: number; end_sec: number; transition: string; trans_dur: number }[]) => void
  /** 是否正在导出(用于禁用导出按钮) */
  exporting?: boolean
  /** 素材库上传视频回调(可选,父组件实现上传 API + 更新 videos 列表) */
  onUploadVideo?: (file: File, onProgress?: (pct: number) => void) => Promise<VideoAssetInfo | null>
}

// ==================== 多轨道定义 ====================
export interface TrackDef {
  type: 'video' | 'audio' | 'subtitle'
  label: string
  emoji: string
  color: string
  height: number
  enabled: boolean
  comingHint?: string
}

// ==================== 转场定义 ====================
export interface TransDef {
  key: string
  label: string
  emoji: string
  color: string
  group: string
}

// ==================== 内部状态辅助类型 ====================
/**
 * v0.42.7: 时间轴原地裁剪拖拽状态
 * null 表示当前无裁剪手柄拖拽中
 */
export interface TrimDragState {
  /** 被拖拽的片段索引 */
  clipIdx: number
  /** 拖拽的是左手柄(改 trimStart)还是右手柄(改 trimEnd) */
  side: 'left' | 'right'
  /** pointerdown 时鼠标的 clientX(用于计算 Δpx) */
  startX: number
  /** pointerdown 时的 trimStart/trimEnd 初值(用于增量叠加) */
  initialValue: number
}

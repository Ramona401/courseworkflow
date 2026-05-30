/**
 * VideoEditorConstants.ts — 视频编辑器常量定义
 *
 * 从 VideoEditorModal.tsx v6.1 拆分而来(v0.42.7)
 *
 * 包含:
 *   - C              颜色调色板(深色主题)
 *   - TRANS          15 种 xfade 转场定义
 *   - TRANS_GROUPS   转场分组(用于属性面板分类展示)
 *   - TRACKS         三轨道定义(视频/音频/字幕)
 *   - TRACK_LABEL_WIDTH 左侧轨道标签栏固定宽度
 *   - TL_GAP         时间轴片段间距
 *   - DRAG_TYPE_ASSET 拖拽 dataTransfer 类型标记
 *   - MIN_DURATION   片段最短时长(秒,沿用 VideoTrimModal)
 *   - TRIM_HANDLE_WIDTH v0.42.7 新增: 原地裁剪手柄宽度
 *   - PX_PER_SEC     v0.42.7 新增: 像素↔秒换算常数(对应 clipPxWidth 的 dur*40 系数)
 */
import type { TransDef, TrackDef } from './VideoEditorTypes'

// ==================== 颜色 ====================
export const C = {
  bg: '#111827',
  surface: '#1F2937',
  surfaceLight: '#374151',
  primary: '#7C3AED',
  primaryLight: 'rgba(124,58,237,0.2)',
  accent: '#F59E0B',
  text: '#F3F4F6',
  textMuted: '#9CA3AF',
  border: 'rgba(255,255,255,0.1)',
  success: '#10B981',
  danger: '#EF4444',
  playing: '#22D3EE',
}

// ==================== 转场 ====================
export const TRANS: TransDef[] = [
  { key: 'none', label: '无', emoji: '—', color: '#6B7280', group: 'basic' },
  { key: 'fade', label: '淡入淡出', emoji: '🌅', color: '#F59E0B', group: 'basic' },
  { key: 'fadeblack', label: '渐黑过渡', emoji: '🌑', color: '#1F2937', group: 'basic' },
  { key: 'fadewhite', label: '渐白过渡', emoji: '⬜', color: '#E5E7EB', group: 'basic' },
  { key: 'dissolve', label: '溶解', emoji: '💫', color: '#3B82F6', group: 'basic' },
  { key: 'wipeleft', label: '左擦除', emoji: '◀️', color: '#10B981', group: 'wipe' },
  { key: 'wiperight', label: '右擦除', emoji: '▶️', color: '#EC4899', group: 'wipe' },
  { key: 'wipeup', label: '上擦除', emoji: '🔼', color: '#14B8A6', group: 'wipe' },
  { key: 'wipedown', label: '下擦除', emoji: '🔽', color: '#F97316', group: 'wipe' },
  { key: 'slideleft', label: '左滑入', emoji: '⬅️', color: '#8B5CF6', group: 'slide' },
  { key: 'slideright', label: '右滑入', emoji: '➡️', color: '#06B6D4', group: 'slide' },
  { key: 'slideup', label: '上滑入', emoji: '⬆️', color: '#84CC16', group: 'slide' },
  { key: 'slidedown', label: '下滑入', emoji: '⬇️', color: '#E11D48', group: 'slide' },
  { key: 'circleopen', label: '圆形展开', emoji: '⭕', color: '#7C3AED', group: 'shape' },
  { key: 'circleclose', label: '圆形关闭', emoji: '🔴', color: '#DC2626', group: 'shape' },
]

export const TRANS_GROUPS = [
  { key: 'basic', label: '基础' },
  { key: 'wipe', label: '擦除' },
  { key: 'slide', label: '滑动' },
  { key: 'shape', label: '图形' },
]

/** 按 key 查找转场色,未匹配返回默认灰色 */
export function getTransColor(k: string): string {
  return TRANS.find(t => t.key === k)?.color || '#6B7280'
}

// ==================== 多轨道 ====================
export const TRACKS: TrackDef[] = [
  { type: 'video',    label: '视频轨', emoji: '🎬', color: '#7C3AED', height: 56, enabled: true },
  { type: 'audio',    label: '音频轨', emoji: '🎵', color: '#10B981', height: 40, enabled: true },
  { type: 'subtitle', label: '字幕轨', emoji: '💬', color: '#F59E0B', height: 40, enabled: true },
]

/** 左侧轨道标签栏宽度(px),不参与水平滚动 */
export const TRACK_LABEL_WIDTH = 88

// ==================== 时间轴尺寸 ====================
/** 片段间距(px) — 用于显示转场圆点 */
export const TL_GAP = 32

/**
 * v0.42.7: 像素↔秒换算常数
 * clipPxWidth = max(80, min(300, duration * PX_PER_SEC))
 * 拖拽手柄时: Δsec = Δpx / PX_PER_SEC
 * 注意片段宽度在 80~300 之间被 clamp,所以拖拽手柄时如果片段是 clamp 边界,
 * 视觉移动会比实际秒数变化更小,但这是合理的体验
 */
export const PX_PER_SEC = 40

// ==================== 拖拽标记 ====================
export const DRAG_TYPE_ASSET = 'application/x-cw-video-asset'

// ==================== 裁剪约束 ====================
/** 片段最短时长(秒) — 与 VideoTrimModal MIN_DURATION 保持一致 */
export const MIN_DURATION = 0.5

/**
 * v0.42.7: 时间轴原地裁剪手柄宽度(px)
 * 6px 在视觉上不抢眼但容易点中(配合 cursor:ew-resize)
 * 仅在片段宽度 >= 80px 时显示(短片段无法挤下手柄)
 */
export const TRIM_HANDLE_WIDTH = 6

// ==================== 字幕语言选项 ====================
/** v0.42.8: 支持的字幕语言列表 */
export const SUBTITLE_LANGUAGES = [
  { code: 'zh-CN', label: '🇨🇳 中文' },
  { code: 'en-US', label: '🇺🇸 英文' },
  { code: 'ja-JP', label: '🇯🇵 日文' },
  { code: 'ko-KR', label: '🇰🇷 韩文' },
]

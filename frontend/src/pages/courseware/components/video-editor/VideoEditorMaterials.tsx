/**
 * VideoEditorMaterials.tsx — 视频编辑器左侧素材库面板
 *
 * 从 VideoEditorModal.tsx v6.1 拆分而来(v0.42.7)
 *
 * 职责:
 *   - 展示素材库视频缩略图(含已添加标记)
 *   - 点击/拖拽添加到时间轴
 *   - 上传新视频(带进度条)
 *
 * 受控渲染: assetDurations 由父组件计算并传入(避免本组件重复 loadDuration)
 */
import type { EditorClip, VideoAssetInfo } from './VideoEditorTypes'
import { C, DRAG_TYPE_ASSET } from './VideoEditorConstants'
import { fmt, withFirstFrame, seekToFirstFrame } from './VideoEditorUtils'

interface VideoEditorMaterialsProps {
  /** 素材库视频列表 */
  videos: VideoAssetInfo[]
  /** 当前时间轴上的片段(用于标记"已添加") */
  clips: EditorClip[]
  /** 各 asset 的时长缓存 */
  assetDurations: Record<string, number>
  /** 上传中标志 */
  uploading: boolean
  /** 上传进度百分比 (0-100) */
  uploadProgress: number
  /** 添加片段到时间轴回调 */
  onAddClip: (video: VideoAssetInfo) => void
  /** 触发上传文件选择回调(父组件实现 <input type="file">) */
  onUploadClick?: () => void
}

export default function VideoEditorMaterials({
  videos, clips, assetDurations, uploading, uploadProgress, onAddClip, onUploadClick,
}: VideoEditorMaterialsProps) {
  return (
    <div style={{ width: 240, borderRight: `1px solid ${C.border}`, display: 'flex', flexDirection: 'column', flexShrink: 0 }}>
      {/* 顶部标题栏 + 上传按钮 */}
      <div style={{ padding: '12px 12px 8px', display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexShrink: 0 }}>
        <span style={{ fontSize: 13, fontWeight: 600, color: C.textMuted }}>
          📂 素材库 <span style={{ fontSize: 11, fontWeight: 400 }}>({videos.length})</span>
        </span>
        {onUploadClick && (
          <button
            onClick={onUploadClick}
            disabled={uploading}
            style={{
              padding: '4px 10px', borderRadius: 6,
              border: `1px solid ${C.accent}`,
              background: uploading ? C.surfaceLight : 'transparent',
              color: C.accent, fontSize: 11, fontWeight: 600,
              cursor: uploading ? 'default' : 'pointer',
            }}
          >
            {uploading ? (uploadProgress > 0 ? `⏳ ${uploadProgress}%` : '⏳ 上传中') : '📤 上传视频'}
          </button>
        )}
      </div>

      {/* 视频卡片列表(可滚动) */}
      <div style={{ flex: 1, overflowY: 'auto', padding: '0 8px 8px' }}>
        {videos.length === 0 && !uploading && (
          <div style={{ fontSize: 12, color: C.textMuted, textAlign: 'center', padding: 20 }}>
            暂无视频素材
          </div>
        )}
        {videos.map(v => {
          // 已添加判定: 原 id 或 originalId(分离音轨后)匹配
          const added = clips.some(c => c.id === v.id || c.originalId === v.id)
          const dur = assetDurations[v.id]
          return (
            <div
              key={v.id}
              draggable={!added}
              onDragStart={e => {
                if (added) { e.preventDefault(); return }
                e.dataTransfer.setData(DRAG_TYPE_ASSET, v.id)
                e.dataTransfer.effectAllowed = 'copy'
              }}
              onClick={() => !added && onAddClip(v)}
              style={{
                marginBottom: 8, borderRadius: 10, overflow: 'hidden',
                cursor: added ? 'default' : 'grab',
                border: `2px solid ${added ? C.primary : 'transparent'}`,
                opacity: added ? 0.5 : 1,
                transition: 'all 150ms',
                background: C.surfaceLight,
              }}
              title={added ? '已添加到时间轴' : '点击或拖拽添加'}
            >
              {/* 视频缩略图(首帧) */}
              <div style={{ position: 'relative', width: '100%', aspectRatio: '16/9', background: '#000', overflow: 'hidden' }}>
                <video
                  src={withFirstFrame(v.url)}
                  preload="metadata"
                  muted
                  playsInline
                  onLoadedMetadata={seekToFirstFrame}
                  style={{ width: '100%', height: '100%', objectFit: 'cover', display: 'block', pointerEvents: 'none' }}
                />
                {/* 右下角时长 */}
                {dur != null && (
                  <div style={{
                    position: 'absolute', bottom: 4, right: 4,
                    padding: '2px 6px', borderRadius: 4,
                    background: 'rgba(0,0,0,0.7)',
                    color: '#fff', fontSize: 10, fontWeight: 600, fontFamily: 'monospace',
                  }}>{fmt(dur)}</div>
                )}
                {/* 已添加遮罩 */}
                {added && (
                  <div style={{
                    position: 'absolute', inset: 0,
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    background: 'rgba(124,58,237,0.3)',
                  }}><span style={{ fontSize: 20 }}>✓</span></div>
                )}
                {/* 未添加: 左上角加号 */}
                {!added && (
                  <div style={{
                    position: 'absolute', top: 4, left: 4,
                    width: 22, height: 22, borderRadius: '50%',
                    background: 'rgba(255,255,255,0.15)', backdropFilter: 'blur(4px)',
                    display: 'flex', alignItems: 'center', justifyContent: 'center',
                    fontSize: 14, color: '#fff', fontWeight: 700,
                  }}>＋</div>
                )}
              </div>
              {/* 文件名 */}
              <div style={{
                padding: '6px 8px', fontSize: 11, fontWeight: 500, color: C.text,
                overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap',
              }}>{v.label}</div>
            </div>
          )
        })}
      </div>

      {/* 底部提示 */}
      <div style={{
        padding: '8px 12px', borderTop: `1px solid ${C.border}`,
        fontSize: 10, color: C.textMuted, textAlign: 'center', flexShrink: 0,
      }}>拖拽到时间轴 或 点击添加</div>
    </div>
  )
}

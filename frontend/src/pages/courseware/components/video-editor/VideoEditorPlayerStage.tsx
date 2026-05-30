/**
 * VideoEditorPlayerStage.tsx — 双播放器舞台区
 *
 * 从 VideoEditorModal.tsx 拆分(B1瘦身)
 *
 * 包含：
 *   - 双 <video> 元素(交叉转场)
 *   - 旁路 <audio> 元素(混音)
 *   - loading 遮罩 + 播放提示徽章
 *   - 软字幕预览叠加
 *   - 空状态占位
 */
import { C } from './VideoEditorConstants'
import { fmtMS, getOutgoingStyle } from './VideoEditorUtils'
import type { SubtitleSegment } from './VideoEditorSubtitleTrack'

interface PlayerStageProps {
  hasClips: boolean
  topPlayer: 'A' | 'B'
  transActive: boolean
  transStyle: string
  transProgress: number
  playMode: 'none' | 'single' | 'all'
  playIdx: number
  clipCount: number
  videoLoading: boolean
  separatingIdx: number
  downloadPct: number
  phTime: number
  totalDur: number
  subtitleSegments: SubtitleSegment[]
  subtitleLanguage: string
  onTogglePlayPause: () => void
}

/** refs 从父组件透传 */
export interface PlayerStageRefs {
  videoARef: React.RefObject<HTMLVideoElement | null>
  videoBRef: React.RefObject<HTMLVideoElement | null>
  audioRef: React.RefObject<HTMLAudioElement | null>
}

export default function VideoEditorPlayerStageComponent({
  hasClips, topPlayer, transActive, transStyle, transProgress,
  playMode, playIdx, clipCount,
  videoLoading, separatingIdx, downloadPct,
  phTime, totalDur,
  subtitleSegments, subtitleLanguage,
  onTogglePlayPause,
  videoARef, videoBRef, audioRef,
}: PlayerStageProps & PlayerStageRefs) {
  const isPlaying = playMode !== 'none'
  const topIsA = topPlayer === 'A'
  const outCSS: React.CSSProperties = transActive ? getOutgoingStyle(transStyle, transProgress) : {}

  let loadingText = '加载视频...'
  if (separatingIdx >= 0) loadingText = downloadPct > 0 ? `缓存视频... ${downloadPct}%` : '分离音轨中...'

  if (!hasClips) {
    return (
      <div style={{
        flex: 1, display: 'flex', flexDirection: 'column',
        alignItems: 'center', justifyContent: 'center', padding: 24, minWidth: 0,
      }}>
        <div style={{ textAlign: 'center', color: C.textMuted }}>
          <div style={{ fontSize: 48, marginBottom: 12 }}>🎬</div>
          <div style={{ fontSize: 15 }}>从左侧素材库拖拽或点击添加视频</div>
          <div style={{ fontSize: 12, marginTop: 6 }}>然后在时间轴上编辑排序和转场</div>
          <audio ref={audioRef} preload="auto" style={{ display: 'none' }} />
        </div>
      </div>
    )
  }

  // 当前时间匹配的字幕
  const currentSub = subtitleSegments
    .filter(s => s.language === subtitleLanguage)
    .find(s => phTime >= s.start_sec && phTime <= s.end_sec)

  return (
    <div style={{
      flex: 1, display: 'flex', flexDirection: 'column',
      alignItems: 'center', justifyContent: 'center', padding: 24, minWidth: 0,
    }}>
      <div onClick={onTogglePlayPause} style={{
        position: 'relative', borderRadius: 12, overflow: 'hidden',
        background: '#000', boxShadow: '0 8px 40px rgba(0,0,0,0.5)',
        cursor: 'pointer', width: '60vw', maxWidth: '100%',
        maxHeight: 'calc(100vh - 350px)', aspectRatio: '16 / 9',
      }}>
        <video ref={videoARef} preload="auto" style={{
          position: 'absolute', inset: 0,
          width: '100%', height: '100%', objectFit: 'contain',
          zIndex: topIsA ? 2 : 1,
          ...(transActive && topIsA ? outCSS : {}),
        }} />
        <video ref={videoBRef} preload="auto" style={{
          position: 'absolute', inset: 0,
          width: '100%', height: '100%', objectFit: 'contain',
          zIndex: topIsA ? 1 : 2,
          ...(transActive && !topIsA ? outCSS : {}),
        }} />
        <audio ref={audioRef} preload="auto" style={{ display: 'none' }} />

        {/* loading 遮罩 */}
        {(videoLoading || separatingIdx >= 0) && (
          <div style={{
            position: 'absolute', inset: 0, zIndex: 12,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'rgba(0,0,0,0.6)', flexDirection: 'column', gap: 8,
          }}>
            <div style={{
              width: 40, height: 40,
              border: '3px solid rgba(255,255,255,0.3)',
              borderTopColor: '#fff', borderRadius: '50%',
              animation: 'cwspin 0.8s linear infinite',
            }} />
            <div style={{ color: '#fff', fontSize: 13, fontWeight: 600 }}>{loadingText}</div>
            <style>{`@keyframes cwspin{to{transform:rotate(360deg)}}`}</style>
          </div>
        )}

        {/* 暂停时播放按钮 */}
        {!isPlaying && !transActive && !videoLoading && separatingIdx < 0 && (
          <div style={{
            position: 'absolute', inset: 0, zIndex: 10,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
            background: 'rgba(0,0,0,0.3)',
          }}>
            <div style={{
              width: 56, height: 56, borderRadius: '50%',
              background: 'rgba(255,255,255,0.2)', backdropFilter: 'blur(8px)',
              display: 'flex', alignItems: 'center', justifyContent: 'center',
              fontSize: 24, color: '#fff',
            }}>▶</div>
          </div>
        )}

        {/* 播放状态徽章 */}
        {(isPlaying || transActive) && (
          <div style={{
            position: 'absolute', top: 10, left: 10, zIndex: 10,
            padding: '4px 10px', borderRadius: 6,
            background: playMode === 'all' ? C.playing + '90' : C.primary + '90',
            color: '#fff', fontSize: 11, fontWeight: 600, backdropFilter: 'blur(4px)',
          }}>{transActive ? '🌀 转场中...' : playMode === 'all' ? `▶ 连贯预览 ${playIdx + 1}/${clipCount}` : '▶ 单段预览'}</div>
        )}

        {/* 时间显示 */}
        <div style={{
          position: 'absolute', bottom: 10, left: 10, zIndex: 10,
          padding: '3px 8px', borderRadius: 4,
          background: 'rgba(0,0,0,0.6)', color: C.accent,
          fontSize: 12, fontWeight: 600, fontFamily: 'monospace',
        }}>{fmtMS(phTime)} / {fmtMS(totalDur)}</div>

        {/* 软字幕预览 */}
        {currentSub && currentSub.text && (
          <div style={{
            position: 'absolute', bottom: 40, left: '50%', transform: 'translateX(-50%)',
            zIndex: 11, maxWidth: '80%', textAlign: 'center',
            padding: '6px 14px', borderRadius: 6,
            background: 'rgba(0,0,0,0.7)', color: '#fff',
            fontSize: 16, fontWeight: 600, lineHeight: 1.4,
            textShadow: '0 1px 3px rgba(0,0,0,0.8)',
            pointerEvents: 'none',
          }}>{currentSub.text}</div>
        )}
      </div>
    </div>
  )
}


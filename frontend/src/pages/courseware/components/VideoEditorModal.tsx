/**
 * VideoEditorModal.tsx — 类剪映多片段视频编辑器主组件 v8.4
 *
 * v8.4模块治理：
 *   - 播放、转场、音轨同步和播放头由useVideoEditorPlayback负责；
 *   - 剪辑、拖拽、音轨分离和上传由useVideoEditorEditing负责；
 *   - 字幕TTS和导出由useVideoEditorSubtitleActions负责；
 *   - 草稿及draft_id字幕绑定由useVideoEditorDraftPersistence负责。
 */

import {
  useCallback,
  useEffect,
  useMemo,
  useState,
} from 'react'
import type {
  EditorClip,
  VideoEditorModalProps,
} from './video-editor/VideoEditorTypes'
import {
  C,
  SUBTITLE_LANGUAGES,
} from './video-editor/VideoEditorConstants'
import VideoEditorMaterials from './video-editor/VideoEditorMaterials'
import VideoEditorPanel from './video-editor/VideoEditorPanel'
import VideoEditorTimeline from './video-editor/VideoEditorTimeline'
import VideoEditorSubtitleModal from './video-editor/VideoEditorSubtitleModal'
import VideoEditorToolbar from './video-editor/VideoEditorToolbar'
import VideoEditorDraftPanel from './video-editor/VideoEditorDraftPanel'
import VideoEditorPlayerStageComponent from './video-editor/VideoEditorPlayerStage'
import VideoEditorExitDialog from './video-editor/VideoEditorExitDialog'
import VideoEditorProgressBar from './video-editor/VideoEditorProgressBar'
import type {
  SubtitleSegment,
} from './video-editor/VideoEditorSubtitleTrack'
import VideoEditorTTSModal from './video-editor/VideoEditorTTSModal'
import VideoEditorExportDialog from './video-editor/VideoEditorExportDialog'
import useVideoEditorDraftPersistence from './video-editor/useVideoEditorDraftPersistence'
import useVideoEditorEditing from './video-editor/useVideoEditorEditing'
import useVideoEditorPlayback from './video-editor/useVideoEditorPlayback'
import useVideoEditorSubtitleActions from './video-editor/useVideoEditorSubtitleActions'

export default function VideoEditorModal({
  videos,
  coursewareId,
  onClose,
  onExport,
  exporting = false,
  onUploadVideo,
}: VideoEditorModalProps) {
  const [clips, setClips] = useState<EditorClip[]>([])
  const [activeIdx, setActiveIdx] = useState(-1)

  const [subtitleSegments, setSubtitleSegments] =
    useState<SubtitleSegment[]>([])
  const [subtitleLanguage, setSubtitleLanguage] = useState('zh-CN')
  const [subtitleDbId, setSubtitleDbId] = useState('')
  const [editingSubtitle, setEditingSubtitle] =
    useState<SubtitleSegment | null>(null)

  const [showExitConfirm, setShowExitConfirm] = useState(false)

  const playback = useVideoEditorPlayback({
    clips,
    activeIdx,
    setActiveIdx,
    subtitleSegments,
    subtitleLanguage,
  })

  const editing = useVideoEditorEditing({
    videos,
    coursewareId,
    onUploadVideo,
    clips,
    setClips,
    activeIdx,
    setActiveIdx,
    stopAll: playback.stopAll,
    queuePlay: playback.queuePlay,
  })

  const draft = useVideoEditorDraftPersistence({
    coursewareId,
    clips,
    setClips,
    subtitleSegments,
    setSubtitleSegments,
    subtitleLanguage,
    setSubtitleLanguage,
    setSubtitleDbId,
  })

  const subtitleActions = useVideoEditorSubtitleActions({
    coursewareId,
    activeDraftId: draft.activeDraftId,
    clips,
    exporting,
    onExport,
    stopAll: playback.stopAll,
    subtitleSegments,
    setSubtitleSegments,
    subtitleLanguage,
    subtitleDbId,
    setSubtitleDbId,
  })

  const doClose = useCallback(() => {
    playback.stopAll()
    setShowExitConfirm(false)
    onClose()
  }, [onClose, playback])

  const handleCloseClick = useCallback(() => {
    if (clips.length > 0) {
      setShowExitConfirm(true)
      return
    }

    doClose()
  }, [clips.length, doClose])

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()

        if (clips.length > 0) {
          setShowExitConfirm(true)
        } else {
          doClose()
        }
      }

      if (event.key === ' ') {
        event.preventDefault()
        playback.togglePlayPause()
      }
    }

    window.addEventListener('keydown', handleKeyDown)

    return () => {
      window.removeEventListener('keydown', handleKeyDown)
    }
  }, [clips.length, doClose, playback])

  const activeClip = activeIdx >= 0 ? clips[activeIdx] : null

  const audioClipCount = useMemo(
    () => clips.filter((clip) => !!clip.audioUrl).length,
    [clips],
  )

  const visibleSubtitles = useMemo(
    () => subtitleSegments.filter(
      (segment) => segment.language === subtitleLanguage,
    ),
    [subtitleLanguage, subtitleSegments],
  )

  return (
    <div
      style={{
        position: 'fixed',
        inset: 0,
        zIndex: 99996,
        background: C.bg,
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <VideoEditorToolbar
        clipCount={clips.length}
        totalDur={playback.totalDur}
        audioClipCount={audioClipCount}
        draftCount={draft.drafts.length}
        draftDismissed={draft.draftDismissed}
        playMode={playback.playMode}
        exporting={exporting}
        onToggleDrafts={() => draft.setDraftDismissed(false)}
        onPlayAll={playback.playAll}
        onStopAll={playback.stopAll}
        onClose={handleCloseClick}
        onExport={subtitleActions.handleExport}
      />

      <VideoEditorDraftPanel
        drafts={draft.drafts}
        visible={draft.drafts.length > 0 && !draft.draftDismissed}
        coursewareId={coursewareId}
        onLoadDraft={async (item) => {
          playback.stopAll()
          setActiveIdx(-1)
          await draft.loadDraft(item)
        }}
        onDeleteDraft={async (draftID) => {
          try {
            await draft.deleteDraftById(draftID)
          } catch {
            alert('删除失败，请重试')
          }
        }}
        onDismiss={() => draft.setDraftDismissed(true)}
      />

      {playback.playMode === 'all' && (
        <VideoEditorProgressBar
          clips={clips}
          playIdx={playback.playIdx}
          playElapsed={playback.playElapsed}
          totalDur={playback.totalDur}
        />
      )}

      <div
        style={{
          flex: 1,
          display: 'flex',
          overflow: 'hidden',
          minHeight: 0,
        }}
      >
        <VideoEditorMaterials
          videos={videos}
          clips={clips}
          assetDurations={editing.assetDurations}
          uploading={editing.uploading}
          uploadProgress={editing.uploadProgress}
          onAddClip={editing.addClip}
          onUploadClick={
            onUploadVideo
              ? editing.handleUploadVideo
              : undefined
          }
        />

        <VideoEditorPlayerStageComponent
          hasClips={clips.length > 0}
          topPlayer={playback.topPlayer}
          transActive={playback.transActive}
          transStyle={playback.transStyle}
          transProgress={playback.transProgress}
          playMode={playback.playMode}
          playIdx={playback.playIdx}
          clipCount={clips.length}
          videoLoading={playback.videoLoading}
          separatingIdx={editing.separatingIdx}
          downloadPct={editing.downloadPct}
          phTime={playback.playheadTime}
          totalDur={playback.totalDur}
          subtitleSegments={subtitleSegments}
          subtitleLanguage={subtitleLanguage}
          onTogglePlayPause={playback.togglePlayPause}
          videoARef={playback.videoARef}
          videoBRef={playback.videoBRef}
          audioRef={playback.audioRef}
        />

        <VideoEditorPanel
          ac={activeClip}
          activeIdx={activeIdx}
          clipsLength={clips.length}
          hasAudio={!!coursewareId}
          separatingIdx={editing.separatingIdx}
          onUpdateClip={editing.updateClip}
          onSeparateAudio={editing.handleSeparateAudio}
          onDeleteAudio={editing.handleDeleteAudio}
          onRestoreOriginal={editing.handleRestoreOriginal}
          onPlaySingle={playback.playSingle}
          onPlayAll={playback.playAll}
        />
      </div>

      <div
        style={{
          borderTop: `1px solid ${C.border}`,
          background: C.surface,
          padding: '10px 24px 14px',
          flexShrink: 0,
        }}
      >
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'center',
            marginBottom: 8,
          }}
        >
          <span style={{ fontSize: 12, color: C.textMuted }}>
            🎞️ 多轨道时间轴 — 三轨堆叠 · 拖手柄裁剪 · 点击标尺跳转 · 空格播放
          </span>

          {clips.length >= 2 && (
            <button
              onClick={
                playback.playMode === 'all'
                  ? playback.stopAll
                  : playback.playAll
              }
              style={{
                padding: '5px 14px',
                borderRadius: 6,
                border: `1px solid ${C.playing}`,
                background: playback.playMode === 'all'
                  ? C.playing + '15'
                  : 'transparent',
                color: C.playing,
                fontSize: 12,
                fontWeight: 600,
                cursor: 'pointer',
              }}
            >
              {playback.playMode === 'all'
                ? '⏹ 停止'
                : '▶ 连贯预览'}
            </button>
          )}
        </div>

        <VideoEditorTimeline
          clips={clips}
          activeIdx={activeIdx}
          playIdx={playback.playIdx}
          playMode={playback.playMode}
          dragIdx={editing.dragIdx}
          dragOverIdx={editing.dragOverIdx}
          timelineDragOver={editing.timelineDragOver}
          phTime={playback.playheadTime}
          draggingPH={playback.draggingPH}
          phDragTime={playback.phDragTime}
          setActiveIdx={setActiveIdx}
          removeClip={editing.removeClip}
          updateClip={editing.updateClip}
          playSingle={playback.playSingle}
          handleDragStart={editing.handleDragStart}
          handleDragOver={editing.handleDragOver}
          handleDrop={editing.handleDrop}
          setDragIdx={editing.setDragIdx}
          setDragOverIdx={editing.setDragOverIdx}
          handleTimelineDragOver={editing.handleTimelineDragOver}
          handleTimelineDragLeave={editing.handleTimelineDragLeave}
          handleTimelineDrop={editing.handleTimelineDrop}
          handleRulerClick={playback.handleRulerClick}
          rulerRef={playback.rulerRef}
          handlePHDown={playback.handlePHDown}
          subtitleSegments={visibleSubtitles}
          subtitleLanguage={subtitleLanguage}
          onSubtitleSegmentsChange={setSubtitleSegments}
          onEditSubtitleSegment={setEditingSubtitle}
          onRequestTTS={subtitleActions.handleRequestTTS}
        />

        <div
          style={{
            marginTop: 6,
            fontSize: 11,
            color: C.textMuted,
            textAlign: 'center',
          }}
        >
          💡 拖手柄裁剪 · 点击标尺跳转 · 拖▼播放头定位 · 空格播放/暂停 · ESC退出
        </div>
      </div>

      <audio
        ref={playback.narrationRef}
        preload="auto"
        style={{ display: 'none' }}
      />

      {subtitleActions.showTTSModal
        && coursewareId
        && subtitleDbId
        && (
          <VideoEditorTTSModal
            coursewareId={coursewareId}
            subtitleId={subtitleDbId}
            segments={visibleSubtitles}
            language={subtitleLanguage}
            onComplete={subtitleActions.handleTTSComplete}
            onClose={() => subtitleActions.setShowTTSModal(false)}
          />
        )}

      {subtitleActions.showExportDialog && (
        <VideoEditorExportDialog
          subtitleCount={subtitleActions.subtitleCount}
          narratedCount={subtitleActions.narratedCount}
          onConfirm={subtitleActions.handleExportConfirm}
          onCancel={() => subtitleActions.setShowExportDialog(false)}
        />
      )}

      {editingSubtitle && (
        <VideoEditorSubtitleModal
          segment={editingSubtitle}
          languages={SUBTITLE_LANGUAGES}
          onSave={(updated) => {
            setSubtitleSegments((current) => (
              updated.text.trim()
                ? current.map((segment) => (
                  segment.id === updated.id ? updated : segment
                ))
                : current.filter((segment) => segment.id !== updated.id)
            ))
            setEditingSubtitle(null)
          }}
          onClose={() => {
            const currentEditing = editingSubtitle

            setSubtitleSegments((current) => current.filter(
              (segment) => !(
                segment.id === currentEditing.id
                && !segment.text.trim()
                && !segment.tts_audio_url
              ),
            ))
            setEditingSubtitle(null)
          }}
        />
      )}

      {showExitConfirm && (
        <VideoEditorExitDialog
          clipCount={clips.length}
          onSaveDraft={async (name) => {
            await draft.saveDraftToServer(name)
            doClose()
          }}
          onDiscard={doClose}
          onCancel={() => setShowExitConfirm(false)}
        />
      )}
    </div>
  )
}

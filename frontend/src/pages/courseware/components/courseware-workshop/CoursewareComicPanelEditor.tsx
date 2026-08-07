/**
 * CoursewareComicPanelEditor.tsx
 *
 * 单格画布编辑器只保留三个层次：
 *   - 当前格标题与必要状态；
 *   - 可直接操作的主画布；
 *   - 自动保存、撤销、重画和同步工具栏。
 *
 * 草稿保护、自动保存、冲突处理和接口调用仍由原Hook负责。
 */

import type {
  CoursewareComicAspectRatio,
  CoursewareComicPanel,
  CoursewareComicProjectStatus,
} from '@/api/coursewares'

import CoursewareComicDirectCanvas from './CoursewareComicDirectCanvas'

import {
  CoursewareComicNotice,
  CoursewareComicPanelFooter,
  CoursewareComicPanelHeader,
  coursewareComicPanelEditorStyles,
} from './CoursewareComicPanelEditorFields'

import {
  useCoursewareComicPanelEditor,
} from './useCoursewareComicPanelEditor'

interface CoursewareComicPanelEditorProps {
  coursewareId: string
  projectId: string
  projectStatus:
    CoursewareComicProjectStatus
  panel: CoursewareComicPanel
  aspectRatio?:
    CoursewareComicAspectRatio
  disabled?: boolean
  regenerating?: boolean
  syncing?: boolean
  onPanelUpdated: (
    panel: CoursewareComicPanel,
  ) => void
  onRegenerate: (
    panel: CoursewareComicPanel,
    regenerationInstruction: string,
  ) => void
  onSync: (
    panel: CoursewareComicPanel,
  ) => void
}

export default function CoursewareComicPanelEditor({
  coursewareId,
  projectId,
  projectStatus,
  panel,
  aspectRatio = 'courseware',
  disabled = false,
  regenerating = false,
  syncing = false,
  onPanelUpdated,
  onRegenerate,
  onSync,
}: CoursewareComicPanelEditorProps) {
  const editor =
    useCoursewareComicPanelEditor({
      coursewareId,
      projectId,
      projectStatus,
      panel,
      disabled,
      regenerating,
      syncing,
      onPanelUpdated,
    })

  return (
    <article
      style={
        coursewareComicPanelEditorStyles
          .container
      }
    >
      <CoursewareComicPanelHeader
        panel={panel}
        overlayDirty={
          editor.overlayDirty
        }
      />

      {editor.notice && (
        <CoursewareComicNotice
          text={editor.notice}
        />
      )}

      <CoursewareComicDirectCanvas
        panel={panel}
        aspectRatio={aspectRatio}
        overlayDocument={
          editor.overlayDocument
        }
        disabled={
          editor.editorDisabled
        }
        selectedElementID={
          editor.selectedElementID
        }
        editingElementID={
          editor.editingElementID
        }
        onSelectElement={
          editor.selectElement
        }
        onBeginEditing={
          editor.beginEditing
        }
        onEndEditing={
          editor.endEditing
        }
        onContentChange={
          editor.handleContentChange
        }
        onQuestionTextChange={
          editor.handleQuestionTextChange
        }
        onQuestionOptionsChange={
          editor.handleQuestionOptionsChange
        }
        onQuestionAnswerChange={
          editor.handleQuestionAnswerChange
        }
        onLayoutChange={
          editor.handleLayoutChange
        }
        onTextStyleChange={
          editor.handleTextStyleChange
        }
        onCycleStyle={
          editor.handleCycleStyle
        }
        onAutoFit={
          editor.handleAutoFit
        }
        onDuplicate={
          editor.handleDuplicate
        }
        onDelete={
          editor.handleDelete
        }
        onKeyDown={
          editor.handleKeyDown
        }
      />

      <CoursewareComicPanelFooter
        version={panel.version}
        overlayDirty={
          editor.overlayDirty
        }
        editorDisabled={
          editor.editorDisabled
        }
        saving={
          editor.saving
        }
        autoSaveState={
          editor.autoSaveState
        }
        autoSaveMessage={
          editor.autoSaveMessage
        }
        overlayVersionConflict={
          editor.overlayVersionConflict
        }
        regenerating={
          regenerating
        }
        syncing={
          syncing
        }
        canRegenerate={
          editor.canRegenerate
        }
        canSync={
          editor.canSync
        }
        canUndo={
          editor.canUndo
        }
        canRedo={
          editor.canRedo
        }
        showSync={
          projectStatus === 'inserted'
        }
        regenerationInstruction={
          editor.regenerationInstruction
        }
        regenerationInstructionLength={
          editor
            .regenerationInstructionLength
        }
        regenerationInstructionMaxLength={
          editor
            .regenerationInstructionMaxLength
        }
        onRegenerationInstructionChange={
          editor
            .handleRegenerationInstructionChange
        }
        onUndo={
          editor.undo
        }
        onRedo={
          editor.redo
        }
        onSave={() => {
          void editor
            .handleSaveOverlay()
        }}
        onRegenerate={() =>
          onRegenerate(
            panel,
            editor
              .normalizedRegenerationInstruction,
          )
        }
        onSync={() =>
          onSync(
            panel,
          )
        }
      />
    </article>
  )
}

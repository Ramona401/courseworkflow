/**
 * CoursewareComicProjectEditorBridge.tsx
 *
 * 隔离五步漫画项目编辑器与父组件回调引用变化。
 */

import {
  memo,
  useCallback,
  useRef,
} from 'react'

import type {
  CoursewareComicProject,
} from '@/api/coursewares'

import CoursewareComicWorkflowProjectEditor from './CoursewareComicWorkflowProjectEditor'

interface CoursewareComicProjectEditorBridgeProps {
  coursewareId: string
  projectId: string
  pageCount: number

  onBack: () => void

  onProjectChanged?: (
    project:
      CoursewareComicProject,
  ) => void

  onPagesChanged?: (
    pageNumber: number,
  ) => void | Promise<void>
}

const StableWorkflowEditor =
  memo(
    CoursewareComicWorkflowProjectEditor,
  )

export default function CoursewareComicProjectEditorBridge({
  coursewareId,
  projectId,
  pageCount,
  onBack,
  onProjectChanged,
  onPagesChanged,
}: CoursewareComicProjectEditorBridgeProps) {
  const onBackRef =
    useRef(onBack)

  const onProjectChangedRef =
    useRef(onProjectChanged)

  const onPagesChangedRef =
    useRef(onPagesChanged)

  onBackRef.current =
    onBack

  onProjectChangedRef.current =
    onProjectChanged

  onPagesChangedRef.current =
    onPagesChanged

  const stableOnBack =
    useCallback(() => {
      onBackRef.current()
    }, [])

  const stableOnProjectChanged =
    useCallback(
      (
        project:
          CoursewareComicProject,
      ) => {
        onProjectChangedRef
          .current?.(
            project,
          )
      },
      [],
    )

  const stableOnPagesChanged =
    useCallback(
      (
        pageNumber: number,
      ) => {
        return onPagesChangedRef
          .current?.(
            pageNumber,
          )
      },
      [],
    )

  return (
    <StableWorkflowEditor
      coursewareId={
        coursewareId
      }
      projectId={
        projectId
      }
      pageCount={
        pageCount
      }
      onBack={
        stableOnBack
      }
      onProjectChanged={
        stableOnProjectChanged
      }
      onPagesChanged={
        stableOnPagesChanged
      }
    />
  )
}

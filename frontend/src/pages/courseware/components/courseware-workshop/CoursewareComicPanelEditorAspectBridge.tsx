/**
 * CoursewareComicPanelEditorAspectBridge.tsx
 *
 * 把项目确认后的真实图片画幅显式传给单格画布编辑器。
 *
 * 不再通过CSS选择器覆盖内部DOM，避免画布结构变化后
 * 4:3、1:1、3:4和9:16项目错误退回16:9。
 */

import type {
  CoursewareComicAspectRatio,
} from '@/api/coursewares'

import CoursewareComicPanelEditor from './CoursewareComicPanelEditor'

interface CoursewareComicPanelEditorAspectBridgeProps
  extends React.ComponentProps<
    typeof CoursewareComicPanelEditor
  > {
  aspectRatio:
    CoursewareComicAspectRatio
}

export default function CoursewareComicPanelEditorAspectBridge({
  aspectRatio,
  ...editorProps
}: CoursewareComicPanelEditorAspectBridgeProps) {
  return (
    <div style={containerStyle}>
      <CoursewareComicPanelEditor
        {...editorProps}
        aspectRatio={aspectRatio}
      />
    </div>
  )
}

const containerStyle:
  React.CSSProperties = {
    minWidth: 0,
  }

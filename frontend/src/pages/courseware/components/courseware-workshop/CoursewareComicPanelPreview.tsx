/**
 * CoursewareComicPanelPreview.tsx
 *
 * 知识点漫画完整分格预览：
 *   - 图片模型只提供无文字视觉底图；
 *   - 说话气泡主体和尾巴使用同一个SVG闭合路径；
 *   - 统一透明度和整体描边，连接处不再露出内部边线；
 *   - 普通预览严格使用教师保存的字号和固定几何，不再隐式缩字；
 *   - 文字、SVG轮廓和尾巴始终共享同一个实际高度。
 */

import type {
  CoursewareComicAspectRatio,
  CoursewareComicOverlayDocument,
  CoursewareComicOverlayElement,
  CoursewareComicPanel,
} from '@/api/coursewares'

import {
  overlayElementDisplayText,
} from './coursewareComicEditorDraft'

import {
  resolveCoursewareComicBubbleTailGeometry,
  resolveCoursewareComicElementVisual,
  resolveCoursewareComicTextContentStyle,
} from './CoursewareComicDirectCanvasElementStyles'

import {
  coursewareComicAspectRatioCSS,
} from './coursewareComicWorkflow'

interface CoursewareComicPanelPreviewProps {
  panel: CoursewareComicPanel
  aspectRatio: CoursewareComicAspectRatio
  overlayDocument?: CoursewareComicOverlayDocument
  emptyLabel?: string
}

const C = {
  textMuted: '#94A3B8',
  border: '#E2E8F0',
  background: '#F8FAFC',
}

export default function CoursewareComicPanelPreview({
  panel,
  aspectRatio,
  overlayDocument,
  emptyLabel = '尚未生成无文字视觉底图',
}: CoursewareComicPanelPreviewProps) {
  const document = overlayDocument || panel.overlay_document

  return (
    <div style={outerStyle}>
      <div
        style={{
          ...previewStyle,
          aspectRatio: coursewareComicAspectRatioCSS(aspectRatio),
        }}
      >
        {panel.current_asset_url ? (
          <img
            src={panel.current_asset_url}
            alt={`第${panel.panel_no}格完整漫画预览`}
            draggable={false}
            style={imageStyle}
          />
        ) : (
          <div style={emptyStyle}>
            <div>
              <div style={emptyIconStyle}>🖼️</div>
              {emptyLabel}
            </div>
          </div>
        )}

        {document.elements.map(element => (
          <OverlayElement
            key={element.id}
            element={element}
          />
        ))}
      </div>
    </div>
  )
}

function OverlayElement({
  element,
}: {
  element: CoursewareComicOverlayElement
}) {
  const visual = resolveCoursewareComicElementVisual(element)
  const textContentStyle = resolveCoursewareComicTextContentStyle(element)
  const bubbleShape = resolveCoursewareComicBubbleTailGeometry(element)

  return (
    <div
      style={{
        ...visual,
        position: 'absolute',
        left: `${element.x * 100}%`,
        top: `${element.y * 100}%`,
        width: `${element.width * 100}%`,
        minHeight: `${element.height * 100}%`,
        height: `${element.height * 100}%`,
        transform: `rotate(${element.rotation || 0}deg)`,
        transformOrigin: 'center',
        zIndex: element.z_index,
        overflow: 'visible',
      }}
    >
      {bubbleShape && (
        <svg
          viewBox="0 0 100 100"
          preserveAspectRatio="none"
          aria-hidden="true"
          style={{
            position: 'absolute',
            inset: 0,
            zIndex: 0,
            width: '100%',
            height: '100%',
            overflow: 'visible',
            pointerEvents: 'none',
            filter: bubbleShape.filter,
          }}
        >
          <path
            d={bubbleShape.shapePath}
            fill={bubbleShape.fill}
            stroke={bubbleShape.stroke}
            strokeWidth={bubbleShape.strokeWidth}
            strokeLinejoin="round"
            strokeLinecap="round"
            vectorEffect="non-scaling-stroke"
          />
        </svg>
      )}

      <div
        style={{
          ...previewTextStyle,
          ...textContentStyle,
        }}
      >
        {overlayElementDisplayText(element)}
      </div>
    </div>
  )
}

const outerStyle: React.CSSProperties = {
  display: 'flex',
  justifyContent: 'center',
  width: '100%',
}

const previewStyle: React.CSSProperties = {
  position: 'relative',
  width: '100%',
  maxWidth: 760,
  maxHeight: 820,
  overflow: 'hidden',
  borderRadius: 12,
  border: `1px solid ${C.border}`,
  background: 'linear-gradient(135deg,#EEF2FF,#F8FAFC)',
  boxShadow: '0 10px 30px rgba(15,23,42,0.08)',
}

const imageStyle: React.CSSProperties = {
  width: '100%',
  height: '100%',
  objectFit: 'cover',
  display: 'block',
}

const previewTextStyle: React.CSSProperties = {
  position: 'relative',
  zIndex: 1,
  width: '100%',
  height: '100%',
  minHeight: 0,
  boxSizing: 'border-box',
  whiteSpace: 'pre-wrap',
  overflow: 'hidden',
  overflowWrap: 'anywhere',
}

const emptyStyle: React.CSSProperties = {
  position: 'absolute',
  inset: 0,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  padding: 20,
  background: C.background,
  color: C.textMuted,
  textAlign: 'center',
  fontSize: 11,
  lineHeight: 1.6,
}

const emptyIconStyle: React.CSSProperties = {
  marginBottom: 6,
  fontSize: 28,
}

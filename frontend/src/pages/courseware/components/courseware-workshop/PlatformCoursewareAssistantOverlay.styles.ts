/**
 * 平台课件预览悬浮教学智能体样式。
 *
 * 本文件只保存视觉和布局规则，不包含会话、鉴权或运行逻辑。
 * 普通预览、全屏预览和原生全屏放映复用同一套视觉协议。
 *
 * 老师端课堂语音增强：
 *   - 全屏和放映默认使用课堂大字模式；
 *   - 嵌入预览可由老师手动打开课堂大字；
 *   - 消息、按钮、状态和备用输入框同步放大；
 *   - 面板仍使用maxWidth约束，避免小屏设备横向溢出。
 */

import type { CSSProperties } from 'react'

export type PlatformAssistantOverlayVariant =
  | 'embedded'
  | 'fullscreen'
  | 'slideshow'

export interface PlatformAssistantOverlayLayout {
  right: number
  bottom: number
  width: number
  maxHeight: number
  zIndex: number
}

export const overlayRootBaseStyle: CSSProperties = {
  position: 'absolute',
  pointerEvents: 'auto',
  fontFamily: '-apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
}

export function panelHeaderStyle(classroomMode: boolean): CSSProperties {
  return {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: classroomMode ? 14 : 10,
    padding: classroomMode ? '15px 17px' : '11px 12px',
    borderBottom: '1px solid #E2E8F0',
    background: 'linear-gradient(135deg, #EEF2FF, #F8FAFC)',
  }
}

export function sessionSummaryStyle(classroomMode: boolean): CSSProperties {
  return {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: 10,
    padding: classroomMode ? '10px 17px' : '7px 12px',
    borderBottom: '1px solid #E2E8F0',
    background: '#F8FAFC',
  }
}

export function conversationStyle(classroomMode: boolean): CSSProperties {
  return {
    flex: 1,
    overflowY: 'auto',
    padding: classroomMode ? 18 : 12,
    background: '#F8FAFC',
  }
}

export function composerStyle(classroomMode: boolean): CSSProperties {
  return {
    display: 'flex',
    flexDirection: 'column',
    gap: classroomMode ? 13 : 9,
    padding: classroomMode ? 16 : 10,
    borderTop: '1px solid #E2E8F0',
    background: '#FFFFFF',
  }
}

export function textareaStyle(classroomMode: boolean): CSSProperties {
  return {
    width: '100%',
    minHeight: classroomMode ? 82 : 62,
    boxSizing: 'border-box',
    resize: 'none',
    padding: classroomMode ? '12px 14px' : '8px 9px',
    borderRadius: classroomMode ? 13 : 9,
    border: '1px solid #CBD5E1',
    color: '#1E293B',
    fontFamily: 'inherit',
    fontSize: classroomMode ? 18 : 10.5,
    lineHeight: classroomMode ? 1.6 : 1.55,
    outline: 'none',
  }
}

export function overlayLayout(
  variant: PlatformAssistantOverlayVariant,
  classroomMode: boolean,
): PlatformAssistantOverlayLayout {
  if (classroomMode) {
    switch (variant) {
    case 'slideshow':
      return {
        right: 28,
        bottom: 82,
        width: 660,
        maxHeight: 760,
        zIndex: 8,
      }

    case 'fullscreen':
      return {
        right: 26,
        bottom: 26,
        width: 640,
        maxHeight: 720,
        zIndex: 8,
      }

    default:
      return {
        right: 18,
        bottom: 18,
        width: 540,
        maxHeight: 650,
        zIndex: 20,
      }
    }
  }

  switch (variant) {
  case 'slideshow':
    return {
      right: 24,
      bottom: 74,
      width: 430,
      maxHeight: 590,
      zIndex: 8,
    }

  case 'fullscreen':
    return {
      right: 22,
      bottom: 22,
      width: 430,
      maxHeight: 570,
      zIndex: 8,
    }

  default:
    return {
      right: 14,
      bottom: 14,
      width: 390,
      maxHeight: 455,
      zIndex: 20,
    }
  }
}

export function floatingButtonStyle(
  variant: PlatformAssistantOverlayVariant,
): CSSProperties {
  const projected = variant !== 'embedded'

  return {
    display: 'inline-flex',
    alignItems: 'center',
    gap: projected ? 10 : 7,
    padding: projected ? '13px 18px' : '9px 12px',
    borderRadius: 999,
    border: '1px solid rgba(79,123,232,0.30)',
    background: 'rgba(255,255,255,0.97)',
    color: '#334155',
    boxShadow: '0 10px 28px rgba(15,23,42,0.22)',
    backdropFilter: 'blur(10px)',
    fontSize: projected ? 15 : 11,
    fontWeight: 800,
    cursor: 'pointer',
  }
}

export function panelStyle(
  layout: PlatformAssistantOverlayLayout,
  classroomMode: boolean,
): CSSProperties {
  return {
    width: layout.width,
    maxWidth: classroomMode
      ? 'calc(100vw - 36px)'
      : 'calc(100vw - 28px)',
    maxHeight: `min(${layout.maxHeight}px, calc(100vh - 32px))`,
    display: 'flex',
    flexDirection: 'column',
    overflow: 'hidden',
    borderRadius: classroomMode ? 20 : 16,
    border: '1px solid rgba(148,163,184,0.42)',
    background: 'rgba(255,255,255,0.98)',
    boxShadow: '0 18px 55px rgba(15,23,42,0.24)',
    backdropFilter: 'blur(12px)',
  }
}

export function headerButtonStyle(
  disabled: boolean,
  classroomMode: boolean,
): CSSProperties {
  return {
    minWidth: classroomMode ? 48 : 30,
    minHeight: classroomMode ? 40 : 29,
    padding: classroomMode ? '8px 12px' : '5px 8px',
    borderRadius: classroomMode ? 10 : 7,
    border: '1px solid #CBD5E1',
    background: '#FFFFFF',
    color: '#64748B',
    fontSize: classroomMode ? 14 : 9,
    fontWeight: 750,
    cursor: disabled ? 'default' : 'pointer',
    opacity: disabled ? 0.45 : 1,
  }
}

export function primaryButtonStyle(
  disabled: boolean,
  classroomMode: boolean,
): CSSProperties {
  return {
    minHeight: classroomMode ? 46 : 32,
    padding: classroomMode ? '10px 20px' : '7px 13px',
    borderRadius: classroomMode ? 12 : 8,
    border: 'none',
    background: '#4F7BE8',
    color: '#FFFFFF',
    fontSize: classroomMode ? 16 : 10,
    fontWeight: 850,
    cursor: disabled ? 'default' : 'pointer',
    opacity: disabled ? 0.45 : 1,
  }
}

export function noticeStyle(
  kind: 'info' | 'success' | 'error',
  classroomMode: boolean,
): CSSProperties {
  const tone = {
    info: {
      background: '#EFF6FF',
      color: '#2563EB',
    },
    success: {
      background: '#ECFDF5',
      color: '#047857',
    },
    error: {
      background: '#FEF2F2',
      color: '#B91C1C',
    },
  }[kind]

  return {
    padding: classroomMode ? '10px 16px' : '7px 11px',
    borderBottom: '1px solid rgba(148,163,184,0.20)',
    background: tone.background,
    color: tone.color,
    fontSize: classroomMode ? 14 : 9.5,
    lineHeight: 1.55,
  }
}

export function messageBubbleStyle(
  assistant: boolean,
  classroomMode: boolean,
): CSSProperties {
  return {
    maxWidth: classroomMode ? '92%' : '86%',
    padding: classroomMode ? '13px 15px' : '8px 9px',
    borderRadius: assistant
      ? classroomMode
        ? '6px 16px 16px 16px'
        : '4px 10px 10px 10px'
      : classroomMode
        ? '16px 6px 16px 16px'
        : '10px 4px 10px 10px',
    border: assistant
      ? '1px solid #E2E8F0'
      : '1px solid rgba(79,123,232,0.26)',
    background: assistant ? '#FFFFFF' : 'rgba(79,123,232,0.10)',
    color: '#1F2937',
    fontSize: classroomMode ? 18 : 10.5,
    lineHeight: classroomMode ? 1.65 : 1.65,
    wordBreak: 'break-word',
  }
}

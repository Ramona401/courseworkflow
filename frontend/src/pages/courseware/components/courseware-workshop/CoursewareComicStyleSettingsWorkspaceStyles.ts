/**
 * CoursewareComicStyleSettingsWorkspaceStyles.ts
 *
 * 第三步视觉设置工作台的纯展示样式。
 * 与业务状态和请求逻辑分离，避免主组件因样式常量再次超过单文件上限。
 */

import type {
  CSSProperties,
} from 'react'

const C = {
  primary: '#7C3AED',
  text: '#1F2937',
  textSecondary: '#64748B',
  textMuted: '#94A3B8',
  border: '#E2E8F0',
  background: '#F8FAFC',
  white: '#FFFFFF',
}

export const containerStyle: CSSProperties = {
  marginBottom: 18,
  padding: 20,
  borderRadius: 16,
  border: `1px solid ${C.border}`,
  background: C.white,
  boxShadow:
    '0 12px 32px rgba(15,23,42,0.05)',
}

export const headerStyle: CSSProperties = {
  display: 'flex',
  alignItems: 'flex-start',
  justifyContent: 'space-between',
  gap: 16,
  marginBottom: 16,
}

export const eyebrowStyle: CSSProperties = {
  marginBottom: 4,
  color: C.primary,
  fontSize: 12,
  fontWeight: 900,
}

export const titleStyle: CSSProperties = {
  margin: 0,
  color: C.text,
  fontSize: 20,
  lineHeight: 1.3,
  fontWeight: 900,
}

export const syncBadgeStyle: CSSProperties = {
  flexShrink: 0,
  padding: '6px 11px',
  borderRadius: 999,
  fontSize: 12,
  fontWeight: 800,
}

export const summaryBarStyle: CSSProperties = {
  display: 'grid',
  gridTemplateColumns:
    'repeat(auto-fit,minmax(130px,1fr))',
  gap: 8,
  marginBottom: 16,
}

export const summaryPillStyle: CSSProperties = {
  minWidth: 0,
  padding: '10px 12px',
  borderRadius: 10,
  background: C.background,
}

export const summaryLabelStyle: CSSProperties = {
  display: 'block',
  marginBottom: 3,
  color: C.textMuted,
  fontSize: 11,
}

export const summaryValueStyle: CSSProperties = {
  display: 'block',
  overflow: 'hidden',
  color: C.text,
  fontSize: 13,
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
}

export const sectionNavStyle: CSSProperties = {
  display: 'grid',
  gridTemplateColumns:
    'repeat(auto-fit,minmax(130px,1fr))',
  gap: 8,
  marginBottom: 16,
}

export const sectionButtonStyle: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  gap: 8,
  minHeight: 44,
  padding: '8px 10px',
  border: '1px solid',
  borderRadius: 10,
  fontFamily: 'inherit',
  fontSize: 13,
  fontWeight: 800,
  cursor: 'pointer',
}

export const sectionNumberStyle: CSSProperties = {
  display: 'inline-flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: 23,
  height: 23,
  borderRadius: 999,
  fontSize: 11,
  fontWeight: 900,
}

export const contentStyle: CSSProperties = {
  minHeight: 250,
  padding: 16,
  borderRadius: 12,
  border: `1px solid ${C.border}`,
  background: '#FCFCFD',
}

export const sectionHeadingStyle: CSSProperties = {
  marginBottom: 14,
}

export const sectionTitleStyle: CSSProperties = {
  margin: 0,
  color: C.text,
  fontSize: 16,
  fontWeight: 900,
}

export const sectionHintStyle: CSSProperties = {
  marginTop: 4,
  color: C.textSecondary,
  fontSize: 13,
  lineHeight: 1.5,
}

export const compactNoticeStyle: CSSProperties = {
  marginTop: 12,
  padding: '11px 13px',
  borderRadius: 10,
  background: '#EFF6FF',
  color: '#334155',
  fontSize: 13,
  lineHeight: 1.55,
}

export const advancedToggleStyle: CSSProperties = {
  marginTop: 10,
  padding: 0,
  border: 'none',
  background: 'transparent',
  color: C.primary,
  fontFamily: 'inherit',
  fontSize: 13,
  fontWeight: 800,
  cursor: 'pointer',
}

export const advancedPanelStyle: CSSProperties = {
  marginTop: 10,
}

export const sourceSummaryStyle: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 12,
  padding: 18,
  borderRadius: 12,
  background: '#EFF6FF',
}

export const sourceIconStyle: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'center',
  width: 36,
  height: 36,
  flexShrink: 0,
  borderRadius: 999,
  background: '#DBEAFE',
  color: '#2563EB',
  fontSize: 18,
  fontWeight: 900,
}

export const sourceTitleStyle: CSSProperties = {
  color: C.text,
  fontSize: 15,
  fontWeight: 900,
}

export const sourceDetailStyle: CSSProperties = {
  marginTop: 4,
  color: C.textSecondary,
  fontSize: 13,
  lineHeight: 1.55,
}

export const formatGridStyle: CSSProperties = {
  display: 'grid',
  gridTemplateColumns:
    'repeat(auto-fit,minmax(260px,1fr))',
  gap: 14,
}

export const sectionFooterStyle: CSSProperties = {
  display: 'flex',
  justifyContent: 'flex-end',
  marginTop: 16,
}

export const nextButtonStyle: CSSProperties = {
  minHeight: 38,
  padding: '8px 13px',
  border: `1px solid ${C.border}`,
  borderRadius: 9,
  background: C.white,
  color: C.primary,
  fontFamily: 'inherit',
  fontSize: 13,
  fontWeight: 800,
  cursor: 'pointer',
}

export const lockedStyle: CSSProperties = {
  marginTop: 12,
  padding: '10px 12px',
  borderRadius: 9,
  background: C.background,
  color: C.textSecondary,
  fontSize: 13,
}

export const warningStyle: CSSProperties = {
  marginTop: 12,
  padding: '10px 12px',
  borderRadius: 9,
  border: '1px solid #FDE68A',
  background: '#FFFBEB',
  color: '#92400E',
  fontSize: 13,
}

export const actionBarStyle: CSSProperties = {
  position: 'sticky',
  bottom: 8,
  zIndex: 2,
  display: 'flex',
  alignItems: 'center',
  justifyContent: 'space-between',
  gap: 12,
  marginTop: 16,
  padding: '12px 14px',
  border: `1px solid ${C.border}`,
  borderRadius: 12,
  background: 'rgba(255,255,255,0.96)',
  boxShadow:
    '0 10px 24px rgba(15,23,42,0.10)',
  backdropFilter: 'blur(10px)',
  flexWrap: 'wrap',
}

export const actionSummaryStyle: CSSProperties = {
  overflow: 'hidden',
  color: C.textSecondary,
  fontSize: 12,
  textOverflow: 'ellipsis',
  whiteSpace: 'nowrap',
}

export const dotStyle: CSSProperties = {
  margin: '0 6px',
  color: C.textMuted,
}

export const actionsStyle: CSSProperties = {
  display: 'flex',
  alignItems: 'center',
  gap: 8,
  flexWrap: 'wrap',
}

export const textButtonStyle: CSSProperties = {
  padding: '9px 6px',
  border: 'none',
  background: 'transparent',
  color: C.textSecondary,
  fontFamily: 'inherit',
  fontSize: 13,
  fontWeight: 700,
  cursor: 'pointer',
}

export const secondaryButtonStyle: CSSProperties = {
  minHeight: 40,
  padding: '9px 14px',
  borderRadius: 9,
  border: `1px solid ${C.border}`,
  background: C.white,
  color: C.text,
  fontFamily: 'inherit',
  fontSize: 13,
  fontWeight: 800,
  cursor: 'pointer',
}

export const primaryButtonStyle: CSSProperties = {
  minHeight: 42,
  padding: '10px 17px',
  borderRadius: 10,
  border: 'none',
  background:
    'linear-gradient(135deg,#7C3AED,#4F46E5)',
  color: '#FFFFFF',
  fontFamily: 'inherit',
  fontSize: 14,
  fontWeight: 900,
  cursor: 'pointer',
}

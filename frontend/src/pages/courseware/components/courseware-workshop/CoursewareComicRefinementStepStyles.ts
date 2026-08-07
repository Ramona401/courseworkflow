/**
 * CoursewareComicRefinementStepStyles.ts
 *
 * 第五步精修工作台的纯样式与画幅辅助。
 *
 * 页面采用“紧凑导航 + 单格画布 + 统一底栏”结构，
 * 不管理业务状态，也不调用接口。
 */

import type {
  CSSProperties,
} from 'react'

import type {
  CoursewareComicWorkflowProject,
} from '@/api/coursewares'

const C = {
  primary: '#7C3AED',
  success: '#059669',
  text: '#1F2937',
  textSecondary: '#64748B',
  textMuted: '#94A3B8',
  border: '#E2E8F0',
  background: '#F8FAFC',
  white: '#FFFFFF',
}

export function thumbnailAspectRatio(
  value:
    CoursewareComicWorkflowProject[
      'workflow'
    ][
      'aspect_ratio'
    ],
): string {
  switch (value) {
  case '4:3':
    return '4 / 3'

  case '1:1':
    return '1 / 1'

  case '3:4':
    return '3 / 4'

  case '9:16':
    return '9 / 16'

  default:
    return '16 / 9'
  }
}

export function clampInteger(
  value: number,
  minimum: number,
  maximum: number,
): number {
  if (!Number.isFinite(value)) {
    return minimum
  }

  return Math.min(
    maximum,
    Math.max(
      minimum,
      Math.trunc(value),
    ),
  )
}

export const refinementColors = {
  primary: C.primary,
  success: C.success,
  border: C.border,
}

export const refinementStyles = {
  compactHeader: {
    display: 'flex',
    alignItems: 'center',
    justifyContent:
      'space-between',
    gap: 16,
    marginBottom: 14,
  },

  title: {
    color: C.text,
    fontSize: 20,
    fontWeight: 900,
    letterSpacing: '-0.02em',
  },

  description: {
    marginTop: 4,
    color: C.textSecondary,
    fontSize: 14,
    lineHeight: 1.55,
  },

  statusBadge: {
    flexShrink: 0,
    padding: '6px 11px',
    borderRadius: 999,
    fontSize: 12,
    fontWeight: 900,
  },

  panelNavigator: {
    display: 'flex',
    alignItems: 'center',
    justifyContent:
      'space-between',
    gap: 10,
    marginBottom: 10,
  },

  panelNavButton: {
    minHeight: 36,
    padding: '7px 12px',
    borderRadius: 9,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    color: C.textSecondary,
    fontSize: 13,
    fontWeight: 800,
    cursor: 'pointer',
  },

  panelCounter: {
    color: C.text,
    fontSize: 14,
    fontWeight: 900,
  },

  panelCounterMuted: {
    color: C.textMuted,
    fontWeight: 700,
  },

  filmstrip: {
    display: 'flex',
    gap: 9,
    marginBottom: 12,
    padding: '3px 2px 7px',
    overflowX: 'auto',
  },

  filmButton: {
    position: 'relative',
    flex: '0 0 108px',
    overflow: 'hidden',
    padding: 0,
    borderRadius: 10,
    border: '2px solid',
    background: C.white,
    cursor: 'pointer',
    transition:
      'border-color 160ms, box-shadow 160ms',
  },

  thumbnail: {
    position: 'relative',
    background: C.background,
  },

  image: {
    width: '100%',
    height: '100%',
    objectFit: 'cover',
    display: 'block',
  },

  empty: {
    height: '100%',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: C.textMuted,
    fontSize: 12,
    fontWeight: 700,
  },

  numberBadge: {
    position: 'absolute',
    left: 6,
    bottom: 6,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    width: 24,
    height: 24,
    borderRadius: 999,
    background:
      'rgba(15,23,42,0.82)',
    color: '#FFFFFF',
    fontSize: 12,
    fontWeight: 900,
  },

  problemBadge: {
    position: 'absolute',
    right: 6,
    top: 6,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    width: 22,
    height: 22,
    borderRadius: 999,
    background: '#DC2626',
    color: '#FFFFFF',
    fontSize: 13,
    fontWeight: 900,
  },

  notice: {
    marginTop: 10,
    padding: '10px 12px',
    borderRadius: 10,
    border: '1px solid',
    fontSize: 13,
    lineHeight: 1.55,
  },

  bottomBar: {
    display: 'flex',
    alignItems: 'center',
    justifyContent:
      'space-between',
    gap: 12,
    marginTop: 12,
    padding: 12,
    borderRadius: 12,
    border:
      `1px solid ${C.border}`,
    background: C.background,
    flexWrap: 'wrap',
  },

  useText: {
    color: C.textSecondary,
    fontSize: 13,
  },

  bottomActions: {
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'flex-end',
    gap: 8,
    flexWrap: 'wrap',
  },

  insertField: {
    display: 'flex',
    alignItems: 'center',
    gap: 5,
    color: C.textSecondary,
    fontSize: 13,
  },

  input: {
    width: 64,
    minHeight: 36,
    padding: '6px 8px',
    borderRadius: 8,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    color: C.text,
    fontSize: 13,
    outline: 'none',
  },

  successButton: {
    minHeight: 38,
    padding: '8px 14px',
    borderRadius: 9,
    border: 'none',
    background:
      'linear-gradient(135deg,#059669,#047857)',
    color: '#FFFFFF',
    fontSize: 13,
    fontWeight: 900,
    cursor: 'pointer',
  },

  unsupported: {
    color: '#92400E',
    fontSize: 13,
  },

  disclosure: {
    position: 'relative',
  },

  disclosureSummary: {
    minHeight: 36,
    display: 'flex',
    alignItems: 'center',
    padding: '7px 12px',
    borderRadius: 9,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    color: C.textSecondary,
    fontSize: 13,
    fontWeight: 800,
    cursor: 'pointer',
    listStyle: 'none',
  },

  disclosureBody: {
    position: 'absolute',
    right: 0,
    bottom: 44,
    zIndex: 20,
    minWidth: 160,
    display: 'grid',
    gap: 7,
    padding: 9,
    borderRadius: 10,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    boxShadow:
      '0 14px 35px rgba(15,23,42,0.16)',
  },

  utilityButton: {
    minHeight: 36,
    padding: '7px 11px',
    borderRadius: 8,
    border:
      `1px solid ${C.border}`,
    background: C.white,
    color: C.textSecondary,
    fontSize: 13,
    fontWeight: 800,
    cursor: 'pointer',
  },
} satisfies Record<
  string,
  CSSProperties
>

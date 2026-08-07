/**
 * coursewareComicExportTextFit.ts
 *
 * 知识点漫画导出专用文字适配：
 *   - 不修改教师保存的气泡位置、宽度、高度或字号；
 *   - 优先保持正式预览字号和内边距；
 *   - 框体空间不足时，先适度收紧内边距，再仅为导出缩小字号；
 *   - 使用浏览器真实DOM排版反复测量，选择能够完整显示的最大字号；
 *   - 不再要求老师把气泡放大到遮挡画面，也不再阻止导出。
 */

import {
  measureCoursewareComicDOMText,
} from './coursewareComicExportDOMMeasure'

import type {
  CoursewareComicDOMTextLayout,
  CoursewareComicDOMTextMeasureOptions,
} from './coursewareComicExportDOMMeasure'

export interface CoursewareComicFittedText {
  options: CoursewareComicDOMTextMeasureOptions
  layout: CoursewareComicDOMTextLayout
  adjusted: boolean
}

interface FitProfile {
  paddingRatio: number
  lineHeightRatio: number
}

const FIT_PROFILES: FitProfile[] = [
  {
    paddingRatio: 1,
    lineHeightRatio: 1,
  },
  {
    paddingRatio: 0.82,
    lineHeightRatio: 1,
  },
  {
    paddingRatio: 0.68,
    lineHeightRatio: 0.94,
  },
  {
    paddingRatio: 0.52,
    lineHeightRatio: 0.88,
  },
]

export function fitCoursewareComicExportText(
  original: CoursewareComicDOMTextMeasureOptions,
): CoursewareComicFittedText {
  const normalized = normalizeOptions(original)
  const originalLayout =
    measureCoursewareComicDOMText(normalized)

  if (!originalLayout.overflow) {
    return {
      options: normalized,
      layout: originalLayout,
      adjusted: false,
    }
  }

  const candidates: CoursewareComicFittedText[] = []

  for (const profile of FIT_PROFILES) {
    const profiled = applyProfile(
      normalized,
      profile,
    )

    const candidate = findLargestFittingFont(profiled)

    if (!candidate.layout.overflow) {
      candidates.push(candidate)
    }
  }

  if (candidates.length > 0) {
    return candidates.sort(compareCandidates)[0]
  }

  /*
   * 极端长文本仍不修改气泡几何。
   * 继续降低导出专用字号直到完整显示；最低值只影响导出成品，
   * 不会写回项目，也不会改变编辑器中的教师字号。
   */
  const emergency = {
    ...applyProfile(
      normalized,
      {
        paddingRatio: 0.38,
        lineHeightRatio: 0.80,
      },
    ),
    fontSize: Math.max(
      2.8,
      normalized.fontSize * 0.16,
    ),
  }

  const emergencyLayout =
    measureCoursewareComicDOMText(emergency)

  return {
    options: emergency,
    layout: emergencyLayout,
    adjusted: true,
  }
}

function findLargestFittingFont(
  profiled: CoursewareComicDOMTextMeasureOptions,
): CoursewareComicFittedText {
  const maximumFont = profiled.fontSize
  const minimumFont = Math.max(
    3.5,
    maximumFont * 0.24,
  )
  const minimumOptions = {
    ...profiled,
    fontSize: minimumFont,
  }
  const minimumLayout =
    measureCoursewareComicDOMText(minimumOptions)

  if (minimumLayout.overflow) {
    return {
      options: minimumOptions,
      layout: minimumLayout,
      adjusted: true,
    }
  }

  let lower = minimumFont
  let upper = maximumFont
  let bestOptions = minimumOptions
  let bestLayout = minimumLayout

  for (let iteration = 0; iteration < 14; iteration += 1) {
    const middle = (lower + upper) / 2
    const options = {
      ...profiled,
      fontSize: middle,
    }
    const layout =
      measureCoursewareComicDOMText(options)

    if (layout.overflow) {
      upper = middle
    } else {
      lower = middle
      bestOptions = options
      bestLayout = layout
    }
  }

  return {
    options: bestOptions,
    layout: bestLayout,
    adjusted:
      Math.abs(bestOptions.fontSize - profiled.fontSize) > 0.1 ||
      bestOptions.paddingHorizontal !== profiled.paddingHorizontal ||
      bestOptions.paddingVertical !== profiled.paddingVertical,
  }
}

function applyProfile(
  original: CoursewareComicDOMTextMeasureOptions,
  profile: FitProfile,
): CoursewareComicDOMTextMeasureOptions {
  return {
    ...original,
    paddingVertical: Math.max(
      1,
      original.paddingVertical * profile.paddingRatio,
    ),
    paddingHorizontal: Math.max(
      1.5,
      original.paddingHorizontal * profile.paddingRatio,
    ),
    lineHeight: Math.max(
      1.02,
      original.lineHeight * profile.lineHeightRatio,
    ),
  }
}

function compareCandidates(
  left: CoursewareComicFittedText,
  right: CoursewareComicFittedText,
): number {
  const fontDifference =
    right.options.fontSize -
    left.options.fontSize

  if (Math.abs(fontDifference) > 0.1) {
    return fontDifference
  }

  const leftPadding =
    left.options.paddingHorizontal +
    left.options.paddingVertical
  const rightPadding =
    right.options.paddingHorizontal +
    right.options.paddingVertical

  return rightPadding - leftPadding
}

function normalizeOptions(
  value: CoursewareComicDOMTextMeasureOptions,
): CoursewareComicDOMTextMeasureOptions {
  return {
    ...value,
    width: finitePositive(value.width, 1),
    height: finitePositive(value.height, 1),
    paddingVertical: Math.max(0, value.paddingVertical),
    paddingHorizontal: Math.max(0, value.paddingHorizontal),
    fontSize: finitePositive(value.fontSize, 10),
    lineHeight: clamp(value.lineHeight, 1, 2.2),
  }
}

function finitePositive(
  value: number,
  fallback: number,
): number {
  return Number.isFinite(value) && value > 0
    ? value
    : fallback
}

function clamp(
  value: number,
  minimum: number,
  maximum: number,
): number {
  if (!Number.isFinite(value)) {
    return minimum
  }

  return Math.min(
    maximum,
    Math.max(minimum, value),
  )
}

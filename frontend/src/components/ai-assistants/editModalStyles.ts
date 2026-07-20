/**
 * editModalStyles.ts — AssistantEditModal 的样式与静态展示常量
 *
 * 文件职责：
 *   1. 提供AI助手编辑弹窗使用的设计颜色；
 *   2. 提供常用emoji快捷选项；
 *   3. 提供Prompt长度上限；
 *   4. 提供表单标签和输入框通用样式。
 *
 * 课程数据职责说明：
 *   - 本文件不再保存任何静态课程或学科清单；
 *   - AssistantEditModal通过useSubjects读取当前用户教育域
 *     和教学组织下的正式课程目录；
 *   - 避免样式文件重新成为业务数据的第二真相源。
 *
 * 抽离动机：
 *   AssistantEditModal包含助手创建、编辑、Prompt设计、
 *   分享策略和存量数据兼容等逻辑。
 *   纯样式和展示常量放在本文件中，可控制主组件体积，
 *   同时保持业务数据由正式Hook统一管理。
 */

import type React from 'react'

/* ==================== 设计颜色 ==================== */

/**
 * AI助手相关弹窗统一使用的设计颜色。
 *
 * 颜色与SaveAssistantModal、AssistantSelector和
 * SharePolicyPicker保持一致。
 */
export const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent: '#F59E0B',
  success: '#10B981',
  danger: '#EF4444',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  bg: '#FAFBFC',
  card: '#FFFFFF',
  border: '#F3F4F6',
  borderMid: '#E5E7EB',
}

/* ==================== 展示选项 ==================== */

/**
 * 常用emoji快捷选择。
 *
 * 用户仍可以在输入框中手动填写其它emoji，
 * 本数组只负责提供常见的快捷入口。
 */
export const QUICK_EMOJIS = [
  '🤖',
  '✨',
  '🎯',
  '📚',
  '🏛️',
  '🏫',
  '👤',
  '🔍',
  '💡',
  '🛠',
  '📝',
  '🧑‍🏫',
]

/* ==================== Prompt限制 ==================== */

/**
 * 助手完整Prompt的存储长度上限。
 *
 * 此值与后端maxAssistantPromptLen保持一致。
 * 运行时实际注入上限由AssistantEditModal单独管理，
 * 两个限制不能混为一谈。
 */
export const MAX_PROMPT_LEN =
  128 * 1024

/* ==================== 通用表单样式 ==================== */

/**
 * 表单标签通用样式。
 */
export const labelStyle:
React.CSSProperties = {
  display: 'block',
  fontSize: '12px',
  fontWeight: 600,
  color: C.textSec,
  marginBottom: '4px',
}

/**
 * input和select通用基础样式。
 *
 * textarea可以在调用处通过展开本对象后，
 * 单独补充resize、lineHeight和最小高度。
 */
export const inputStyle:
React.CSSProperties = {
  padding: '7px 10px',
  borderRadius: '6px',
  border: `1px solid ${C.border}`,
  fontSize: '13px',
  color: C.text,
  outline: 'none',
  boxSizing: 'border-box',
  fontFamily: 'inherit',
  background: '#fff',
}

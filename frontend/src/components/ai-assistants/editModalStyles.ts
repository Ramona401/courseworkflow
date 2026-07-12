/**
 * editModalStyles.ts — AssistantEditModal 的样式与静态数据常量
 *
 * 抽离动机:
 *   AssistantEditModal.tsx 因新增 share_policy 选择器逼近/超过 600 行红线。
 *   把"不含逻辑的纯常量"(颜色、可选学科、快捷 emoji、label/input 样式)外移到本文件,
 *   让主组件回到红线内,且这些常量本就与渲染逻辑无关,外移不影响可读性。
 *
 * 注:颜色 C 与 SaveAssistantModal / SharePolicyPicker 同源(同一套设计 token),
 *    各文件各自持一份最小集,避免引入一个全局 theme 文件造成的牵连改动。
 */
import type React from 'react'
import { DEFAULT_SUBJECTS } from '@/constants/subjects'

/** 设计 token 颜色(与 AssistantSelector / SaveAssistantModal 一致) */
export const C = {
  primary:      '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent:       '#F59E0B',
  success:      '#10B981',
  danger:       '#EF4444',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  bg:           '#FAFBFC',
  card:         '#FFFFFF',
  border:       '#F3F4F6',
  borderMid:    '#E5E7EB',
}

/** 常用 emoji 快捷选择行(用户也可手动输入任意 emoji) */
export const QUICK_EMOJIS = ['🤖', '✨', '🎯', '📚', '🏛️', '🏫', '👤', '🔍', '💡', '🛠', '📝', '🧑‍🏫']

/** 学科可选项(与 workshopConstants 保持一致,避免 import 导致的循环依赖,这里写死) */
export const SUBJECTS = ['', ...DEFAULT_SUBJECTS]  // 空=不限；单一真相源（方案甲，v231）

/** prompt 长度上限(与后端 maxAssistantPromptLen 对齐) */
export const MAX_PROMPT_LEN = 128 * 1024

/** label 通用样式 */
export const labelStyle: React.CSSProperties = {
  display: 'block',
  fontSize: '12px', fontWeight: 600, color: C.textSec,
  marginBottom: '4px',
}

/** input/select 通用样式 */
export const inputStyle: React.CSSProperties = {
  padding: '7px 10px', borderRadius: '6px',
  border: `1px solid ${C.border}`,
  fontSize: '13px', color: C.text,
  outline: 'none', boxSizing: 'border-box',
  fontFamily: 'inherit',
  background: '#fff',
}

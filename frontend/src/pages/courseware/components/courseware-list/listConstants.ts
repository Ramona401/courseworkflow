/**
 * 课件列表页共享常量与类型 — listConstants.ts
 *
 * 从 CoursewareListPage.tsx(原 1140 行)拆出的纯常量层。
 * 列表页主文件 + CWCard / SharedCWCard / PublishPanel / CreateCoursewareModal
 * 五处统一 import,作为单一真相源,避免颜色/学科/状态集合在多文件各写一份。
 *
 * 不含任何运行时请求逻辑或 React 组件,纯数据 + 类型。
 */

import type React from 'react'
import { DEFAULT_SUBJECTS } from '@/constants/subjects'

// ==================== 配色 ====================
// 课件工坊暖色系主色板(橙→红渐变体系),与 CWSidebar / 工坊页一致。
export const C = {
  primary: '#F59E0B', primaryBg: 'rgba(245,158,11,0.08)',
  textPrimary: '#1F2937', textSecondary: '#6B7280', textMuted: '#9CA3AF',
  border: '#E5E7EB',
}

// ==================== 课件来源类型配色 ====================
// 六来源各自的中文标签 + 配色 + emoji,卡片左下角小标签用。
export const SOURCE_CONFIG: Record<string, { label: string; color: string; bg: string; emoji: string }> = {
  lesson_plan:  { label: '教案生成', color: '#2563EB', bg: '#DBEAFE', emoji: '📝' },
  topic_direct: { label: '主题创建', color: '#7C3AED', bg: '#EDE9FE', emoji: '💡' },
  ppt_upload:   { label: 'PPT上传', color: '#D97706', bg: '#FEF3C7', emoji: '📊' },
  doc_upload:   { label: '文档上传', color: '#0891B2', bg: '#CFFAFE', emoji: '📄' },
  html_import:  { label: 'HTML导入', color: '#059669', bg: '#D1FAE5', emoji: '🌐' },
  '3d_single':  { label: '3D互动', color: '#DC2626', bg: '#FEE2E2', emoji: '🎮' },
}

// ==================== 学科列表 ====================
// 与备课工坊 / 配方向导统一为同一份 17 学科清单(v202 三端统一)。
export const SUBJECTS = [...DEFAULT_SUBJECTS]  // 单一真相源（方案甲，v231）

// ==================== 可共享 / 可下载状态集合 ====================
// 已生成、可共享 / 可下载离线包的状态集合(与后端"共享要求 status≥preview"一致)。
// CWCard 的"发布/分享"按钮与"下载离线包"按钮均以此判定显隐。
export const SHAREABLE_STATUSES = ['preview', 'confirmed', 'in_pipeline']

// ==================== 通用按钮基础样式 ====================
export const btnBase: React.CSSProperties = {
  padding: '8px 20px', borderRadius: '8px', fontSize: '14px', cursor: 'pointer',
}

// ==================== 从教案创建用的教案列表项类型 ====================
// 仅"从教案创建"弹窗内部用,字段是教案列表接口返回的子集。
export interface LPItem { id: string; title: string; subject: string; grade: string; status: string }

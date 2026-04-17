/**
 * myPlansConstants.ts — 我的教案页面共用常量和配置
 */
import type { LessonPlanStatus } from '@/api/lesson-plans'

// ==================== 颜色常量 ====================
export const C = {
  primary:      '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  accent:       '#F59E0B',
  success:      '#10B981',
  warning:      '#F97316',
  danger:       '#EF4444',
  text:         '#1F2937',
  textSec:      '#6B7280',
  textMuted:    '#9CA3AF',
  bg:           '#FAFBFC',
  card:         '#FFFFFF',
  border:       '#F3F4F6',
  borderHover:  '#E5E7EB',
}

// ==================== 状态配置 ====================
export interface StatusConfig {
  label: string; color: string; bg: string; dot: string; desc: string
}

export const STATUS_CONFIG: Record<LessonPlanStatus, StatusConfig> = {
  draft:              { label: '草稿',      color: '#6B7280', bg: '#F3F4F6',                dot: '#9CA3AF',  desc: '尚未发布，仅自己可见' },
  published_personal: { label: '已发布',    color: C.primary, bg: C.primaryLight,           dot: C.primary,  desc: '个人发布，可进入课件开发' },
  submitted:          { label: '待评审',    color: C.accent,  bg: 'rgba(245,158,11,0.08)', dot: C.accent,   desc: '已提交教研组，等待评审' },
  revision:           { label: '退回修改',  color: C.warning, bg: 'rgba(249,115,22,0.08)', dot: C.warning,  desc: '评审退回，需修改后重新提交' },
  approved:           { label: '评审通过',  color: C.success, bg: 'rgba(16,185,129,0.08)', dot: C.success,  desc: '已通过教研组评审' },
  published_shared:   { label: '已共享',    color: '#8B5CF6', bg: 'rgba(139,92,246,0.08)', dot: '#8B5CF6',  desc: '已共享到教研组/学校，其他老师可查看' },
  developing:         { label: '课件开发中',color: '#0EA5E9', bg: 'rgba(14,165,233,0.08)', dot: '#0EA5E9',  desc: '已进入课件开发流程，教案已锁定' },
  completed:          { label: '已完成',    color: C.success, bg: 'rgba(16,185,129,0.08)', dot: C.success,  desc: '课件开发完成' },
}

// ==================== 筛选栏配置 ====================
export const STATUS_FILTERS: Array<{
  key: string; label: string; statuses: LessonPlanStatus[] | null
}> = [
  { key: 'all',      label: '全部',   statuses: null },
  { key: 'draft',    label: '草稿',   statuses: ['draft'] },
  { key: 'personal', label: '已发布', statuses: ['published_personal'] },
  { key: 'review',   label: '评审中', statuses: ['submitted', 'revision', 'approved'] },
  { key: 'shared',   label: '已共享', statuses: ['published_shared'] },
  { key: 'dev',      label: '开发中', statuses: ['developing', 'completed'] },
]

export const SUBJECTS = ['全部','人工智能','语文','数学','英语','物理','化学','生物','历史','地理','政治','信息技术']
export const GRADES   = ['全部','七年级','八年级','九年级','高一','高二','高三','小学低段','小学中段','小学高段']

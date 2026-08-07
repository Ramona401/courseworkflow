/**
 * 课件组件管理页面共享UI配置。
 *
 * 这里只保存无状态展示配置，不承担权限判断。
 * 教育域权限始终由后端控制，页面只做对应的交互收敛。
 */
import { CW_COMP_TYPE_CONFIG } from '@/api/coursewares'
import {
  RESOURCE_EDUCATION_DOMAIN_LABELS,
  type ResourceEducationDomain,
} from '@/education-domain/types'

export const CW_COMPONENT_COLORS = {
  primary: '#F59E0B',
  textPrimary: '#1F2937',
  textSecondary: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  bgCard: '#FFFFFF',
  danger: '#EF4444',
}

export const CW_COMPONENT_DOMAIN_OPTIONS = (
  Object.entries(
    RESOURCE_EDUCATION_DOMAIN_LABELS,
  ) as Array<
    [ResourceEducationDomain, string]
  >
).map(([value, label]) => ({
  value,
  label,
}))

export const CW_COMPONENT_TYPE_FILTERS = [
  {
    value: '',
    label: '全部',
  },
  ...Object.entries(
    CW_COMP_TYPE_CONFIG,
  ).map(([value, config]) => ({
    value,
    label: config.label,
  })),
]

export function getCWComponentTypeConfig(
  componentType: string,
) {
  return CW_COMP_TYPE_CONFIG[
    componentType
  ] || {
    label: componentType,
    color: '#6B7280',
    bg: '#F3F4F6',
  }
}

/**
 * adminConstants.ts — Admin页面共用常量、样式变量、工具函数
 *
 * 角色名称与学校体系对齐：
 *   admin              → 系统管理员
 *   region_admin       → 区域管理员（迭代一新增）
 *   district_inspector → 区域教研员（迭代一新增）
 *   senior_operator    → 学校管理员
 *   operator           → 骨干教师
 *   viewer             → 普通教师
 *
 * 归属治理批C（任命唯一事实源）新增：
 *   APPOINTMENT_ONLY_ROLES —— 任命制身份清单（senior_operator/region_admin）。
 *   不变式：这两个身份 ⇔ 组织任命存在。获得只能走「组织架构→🛡️管理员」任命
 *   （自动升身份），失去随末任命移除自动降级。建号弹窗与编辑角色下拉据此过滤；
 *   ROLE_OPTIONS 本体保持全量（用户列表按角色【筛选】仍需要这两项）。
 */

// ==================== 颜色常量 ====================
export const C = {
  primary: '#4F7BE8',
  primaryLight: 'rgba(79,123,232,0.08)',
  success: '#10B981',
  successLight: 'rgba(16,185,129,0.08)',
  danger: '#EF4444',
  dangerLight: 'rgba(239,68,68,0.08)',
  warning: '#F59E0B',
  warningLight: 'rgba(245,158,11,0.08)',
  purple: '#7C3AED',
  purpleLight: 'rgba(124,58,237,0.08)',
  text: '#1F2937',
  textSec: '#6B7280',
  textMuted: '#9CA3AF',
  border: '#E5E7EB',
  bg: '#F9FAFB',
  white: '#FFFFFF',
}

// ==================== 下拉选项常量（角色名与学校体系对齐）====================
// 注意：顺序按权限层级从高到低排列，便于用户列表筛选与下拉理解
export const ROLE_OPTIONS = [
  { value: '',                   label: '全部角色' },
  { value: 'admin',              label: '系统管理员' },
  { value: 'region_admin',       label: '区域管理员' },
  { value: 'district_inspector', label: '区域教研员' },
  { value: 'senior_operator',    label: '学校管理员' },
  { value: 'operator',           label: '骨干教师' },
  { value: 'viewer',             label: '普通教师' },
]

// 批C：任命制身份（不可经建号/编辑直接授予，详见文件头注释）
export const APPOINTMENT_ONLY_ROLES = ['senior_operator', 'region_admin']

export const ACTION_OPTIONS = [
  { value: '', label: '全部操作' },
  { value: 'user.login', label: '用户登录' },
  { value: 'user.logout', label: '用户登出' },
  { value: 'admin.user_create', label: '创建用户' },
  { value: 'admin.user_status', label: '状态变更' },
  { value: 'admin.user_reset_password', label: '重置密码' },
  { value: 'pipeline.confirm_finalize', label: '确认定稿' },
  { value: 'pipeline.verify', label: '触发验收' },
]

// ==================== 工具函数 ====================

/** 相对时间格式化：刚刚 / N分钟前 / N小时前 / N天前 / 月日 */
export function formatRelativeTime(dateStr: string): string {
  const date = new Date(dateStr)
  const diffMin = Math.floor((Date.now() - date.getTime()) / 60000)
  if (diffMin < 1)  return '刚刚'
  if (diffMin < 60) return `${diffMin}分钟前`
  const diffH = Math.floor(diffMin / 60)
  if (diffH < 24)   return `${diffH}小时前`
  const diffD = Math.floor(diffH / 24)
  if (diffD < 7)    return `${diffD}天前`
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

/** 日期时间格式化：去掉T，截取到分钟 */
export function fmt(dateStr: string | null | undefined): string {
  if (!dateStr) return '—'
  return String(dateStr).replace('T', ' ').substring(0, 16)
}

/** 操作日志颜色：user.*蓝 / admin.*紫 / 其他黄 */
export function getActionStyle(action: string) {
  if (action.startsWith('user.'))  return { bg: C.primaryLight, color: C.primary }
  if (action.startsWith('admin.')) return { bg: C.purpleLight,  color: C.purple  }
  return { bg: C.warningLight, color: C.warning }
}

/** 行内小按钮样式生成 */
export function rowBtn(color: string, bgColor: string): React.CSSProperties {
  return {
    padding: '3px 8px', borderRadius: '5px',
    border: `1px solid ${bgColor}`,
    background: bgColor, color, fontSize: '11px',
    cursor: 'pointer', fontWeight: 500,
  }
}

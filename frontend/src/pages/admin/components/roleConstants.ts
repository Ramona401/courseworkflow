/**
 * roleConstants.ts — 角色权限 Tab 专用常量
 *
 * 角色名称与学校体系对齐：
 *   admin           → 系统管理员（全平台）
 *   senior_operator → 学校管理员（管本校）
 *   operator        → 骨干教师（操作Pipeline+审核教案）
 *   viewer          → 普通教师（只做教案，不进Pipeline）
 */

// ==================== 权限分组定义 ====================

/** 单个权限项描述 */
export interface PermissionDef {
  code: string        // permission_code，如 pipeline.create
  resource: string    // 资源，如 pipeline
  action: string      // 动作，如 create
  label: string       // 中文说明
}

/** 权限分组 */
export interface PermissionGroup {
  groupKey: string    // 分组标识
  groupName: string   // 分组中文名
  icon: string        // emoji 图标
  permissions: PermissionDef[]
}

/**
 * 全部权限分组（共18项，与后端 role_permissions 表一致）
 */
export const PERMISSION_GROUPS: PermissionGroup[] = [
  {
    groupKey: 'pipeline',
    groupName: 'Pipeline 课件操作',
    icon: '🎬',
    permissions: [
      { code: 'pipeline.create',        resource: 'pipeline', action: 'create',        label: '创建 Pipeline' },
      { code: 'pipeline.start',         resource: 'pipeline', action: 'start',         label: '启动 Pipeline' },
      { code: 'pipeline.cancel',        resource: 'pipeline', action: 'cancel',        label: '取消 Pipeline' },
      { code: 'pipeline.review',        resource: 'pipeline', action: 'review',        label: '审核页面（决策/定稿）' },
      { code: 'pipeline.finalize',      resource: 'pipeline', action: 'finalize',      label: '确认/退回定稿' },
      { code: 'pipeline.verify',        resource: 'pipeline', action: 'verify',        label: '触发验收' },
      { code: 'pipeline.batch',         resource: 'pipeline', action: 'batch',         label: '批量操作' },
      { code: 'pipeline.force_proceed', resource: 'pipeline', action: 'force_proceed', label: '强制推进（跳过Translator）' },
      { code: 'pipeline.restart',       resource: 'pipeline', action: 'restart',       label: '断点重跑' },
      { code: 'pipeline.delete',        resource: 'pipeline', action: 'delete',        label: '删除 Pipeline' },
    ],
  },
  {
    groupKey: 'system',
    groupName: '系统配置',
    icon: '⚙️',
    permissions: [
      { code: 'user.manage',      resource: 'user',      action: 'manage', label: '用户管理（增删改）' },
      { code: 'ai_config.config', resource: 'ai_config', action: 'config', label: 'AI 配置管理' },
      { code: 'prompt.write',     resource: 'prompt',    action: 'write',  label: '提示词编辑' },
      { code: 'course.manage',    resource: 'course',    action: 'manage', label: '课程管理' },
      { code: 'system.read',      resource: 'system',    action: 'read',   label: '系统只读（仪表盘）' },
      { code: 'system.config',    resource: 'system',    action: 'config', label: '系统配置管理' },
    ],
  },
  {
    groupKey: 'lesson_plan',
    groupName: '教案系统',
    icon: '📚',
    permissions: [
      { code: 'lesson_plan.create', resource: 'lesson_plan', action: 'create', label: '创建教案' },
      { code: 'lesson_plan.review', resource: 'lesson_plan', action: 'review', label: '审核教案' },
    ],
  },
]

/** 所有权限的扁平列表（方便查找） */
export const ALL_PERMISSIONS: PermissionDef[] = PERMISSION_GROUPS.flatMap(g => g.permissions)

// ==================== 系统内置角色静态数据（与学校体系对齐）====================

/** 系统内置角色描述 */
export interface SystemRoleInfo {
  code: string
  displayName: string   // 与学校体系对齐的中文名
  description: string
  permissionCodes: string[]
}

/**
 * 4个系统内置角色完整权限矩阵
 *   admin=18 / senior_operator=12 / operator=10 / viewer=2
 */
export const SYSTEM_ROLES: SystemRoleInfo[] = [
  {
    code: 'admin',
    displayName: '系统管理员',
    description: '平台超级管理员，拥有全部功能权限，包含系统配置、用户管理、AI管理及所有Pipeline操作。',
    permissionCodes: ALL_PERMISSIONS.map(p => p.code),  // 全部18项
  },
  {
    code: 'senior_operator',
    displayName: '学校管理员',
    description: '负责管理本校所有教研组，拥有Pipeline全流程操作权限，含审核、定稿、批量操作、强制推进及验收。',
    permissionCodes: [
      'pipeline.create', 'pipeline.start', 'pipeline.cancel',
      'pipeline.review', 'pipeline.finalize', 'pipeline.verify',
      'pipeline.batch', 'pipeline.force_proceed', 'pipeline.restart',
      'pipeline.delete',
      'system.read',
      'lesson_plan.create',
    ],
  },
  {
    code: 'operator',
    displayName: '骨干教师',
    description: '教研组骨干，可操作Pipeline（创建/审核/定稿/验收），可审核教案、确认萃取，支持断点重跑及强制推进。',
    permissionCodes: [
      'pipeline.create', 'pipeline.start', 'pipeline.cancel',
      'pipeline.review', 'pipeline.finalize', 'pipeline.verify',
      'pipeline.batch', 'pipeline.force_proceed', 'pipeline.restart',
      'system.read',
    ],
  },
  {
    code: 'viewer',
    displayName: '普通教师',
    description: '普通教师，专注教案编写，可创建教案、提交评审，不进入Pipeline课件审核板块。',
    permissionCodes: [
      'system.read',
      'lesson_plan.create',
    ],
  },
]

/** 系统内置角色颜色映射 */
export const SYSTEM_ROLE_COLORS: Record<string, { bg: string; color: string; border: string }> = {
  admin:           { bg: 'rgba(239,68,68,0.06)',   color: '#DC2626', border: 'rgba(239,68,68,0.2)'   },
  senior_operator: { bg: 'rgba(245,158,11,0.06)',  color: '#D97706', border: 'rgba(245,158,11,0.2)'  },
  operator:        { bg: 'rgba(79,123,232,0.06)',  color: '#3B5FC0', border: 'rgba(79,123,232,0.2)'  },
  viewer:          { bg: 'rgba(107,114,128,0.06)', color: '#4B5563', border: 'rgba(107,114,128,0.2)' },
}

/** 基础角色下拉选项（新建角色时选择"继承自"，显示与学校体系对齐的名称）*/
export const BASE_ROLE_OPTIONS = [
  { value: '',                label: '不继承（从零配置）' },
  { value: 'admin',           label: '继承 admin（系统管理员）' },
  { value: 'senior_operator', label: '继承 senior_operator（学校管理员）' },
  { value: 'operator',        label: '继承 operator（骨干教师）' },
  { value: 'viewer',          label: '继承 viewer（普通教师）' },
]

/** 根据 role_code 获取系统内置角色颜色，不存在时返回默认紫色 */
export function getRoleColor(roleCode: string) {
  return SYSTEM_ROLE_COLORS[roleCode] ?? {
    bg: 'rgba(124,58,237,0.06)',
    color: '#6D28D9',
    border: 'rgba(124,58,237,0.2)',
  }
}

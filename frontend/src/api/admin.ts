/**
 * 统一用户管理中心 API 封装
 * 对应后端 /api/v1/admin/* 和 /api/v1/lesson-plans/organizations/* 路由
 * 仅 admin 可调用（路由层保护）
 *
 * 归属治理批B改动（2026-07-04）：
 *   - AdminUserListItem 新增 school_count（校籍数，后端批A起返回）：
 *       0=零归属；>0 且 group_count=0 = 有校籍但未加入任何教研组
 *   - 新增 AdminSchoolMembership 类型（单条校籍：数据源 school_members，
 *     用户归属的唯一事实源；含入校来源与该校组数）
 *   - AdminUserDetail 新增 schools 数组（校籍清单，UserDetailModal"所属学校"区块消费）
 *   - 新增 removeUserFromSchool（R3 移出本校：DELETE /admin/users/{uid}/schools/{sid}，
 *     后端单事务连带退出该校全部教研组+删校籍，返回中文 message）
 *
 * v109改动：
 *   - GroupListItem 新增 lead_user_names（所有组长名称，逗号分隔）
 *   - GroupDetail 新增 lead_user_names
 *   - CreateGroupRequest / UpdateGroupRequest 移除 lead_user_id（改由成员角色管理）
 *
 * Phase6.2改动（区域管理员）：
 *   - AdminStats 新增 region_admin_count（后端 stats 已返回，类型对齐避免 TS 报错）
 *   - 新增组织多管理员接口（organization_admins 的 List/Add/Remove），
 *     对应后端 /api/v1/lesson-plans/organizations/{id}/admins，供第二批多管理员 UI 直接调用
 *
 * 合并重构改动（废弃 school-admin 并轨）：
 *   - 末尾新增批量导入用户 API（batchCreateAdminUsers + 相关类型），
 *     对接后端 POST /api/v1/admin/users/batch，替代旧 /school-admin/users/batch。
 *     类型在此独立定义（不再从 school-admin.ts import），因 school-admin.ts 将被删除。
 *
 * B13改动（任命即同步身份）：
 *   - AddOrgAdminRequest 新增 sync_role（前端默认 true：任命 operator/viewer 时
 *     同步升级其系统身份为区域管理员/学校管理员，根治"有管辖无门票"静默失效）
 *   - 新增 AddOrgAdminResult 类型；addOrgAdmin 返回后端拼好的 message（前端原样
 *     toast）与 role_synced/new_role 标志
 */
import client from './client'

// ==================== 用户相关类型 ====================

export interface AdminUserListItem {
  id: string
  username: string
  display_name: string
  role: string
  role_name: string
  status: string
  login_count: number
  last_login_at: string | null
  created_at: string
  school_name: string
  group_name: string
  group_role: string
  group_count: number
  /** 批B新增：校籍数（school_members 计数）。0=零归属；>0且group_count=0=有校籍但无组 */
  school_count: number
}

export interface AdminUserListResult {
  users: AdminUserListItem[]
  total: number
  page: number
  page_size: number
}

export interface AdminUserListParams {
  page?: number
  page_size?: number
  role?: string
  status?: string
  keyword?: string
  school_id?: string
  group_id?: string
}

export interface AdminCourseAssignment {
  course_code: string
  course_name: string
  assigned_at: string
}

export interface AdminGroupMembership {
  group_id: string
  group_name: string
  school_name: string
  role: string
  role_name: string
  is_lead: boolean
  joined_at: string
}

/**
 * 单条校籍记录（批B新增，数据源 school_members = 用户归属的唯一事实源）
 *   source/source_name : 入校来源（随教研组自动加入/管理员建号入校/批量导入入校…）
 *   group_count        : 该用户在此学校的教研组数（0=有校籍但未加入任何组）
 */
export interface AdminSchoolMembership {
  school_id: string
  school_name: string
  source: string
  source_name: string
  joined_at: string
  group_count: number
  /** 批C2：是否为该校管理员（任命∪主字段双来源），弹窗据此提供"同时移除任命"勾选 */
  is_school_admin: boolean
}

export interface AdminUserDetail extends AdminUserListItem {
  course_assignments: AdminCourseAssignment[]
  teaching_groups: AdminGroupMembership[]
  /** 批B新增：校籍清单（"所属学校"区块与「移出本校」按钮消费） */
  schools: AdminSchoolMembership[]
}

export interface AdminStats {
  total_users: number
  active_users: number
  disabled_users: number
  total_orgs: number
  total_schools: number
  total_groups: number
  total_members: number
  admin_count: number
  // Phase6.2新增：区域管理员人数（后端统计已返回，可能为 0）
  region_admin_count?: number
  senior_operator_count: number
  operator_count: number
  viewer_count: number
}

export interface AuditLogItem {
  id: string
  user_id: string
  username: string
  display_name: string
  action: string
  action_name: string
  detail: string
  ip: string
  created_at: string
}

export interface AuditLogListResult {
  logs: AuditLogItem[]
  total: number
}

export interface AuditLogQueryParams {
  page?: number
  page_size?: number
  user_id?: string
  username?: string
  action?: string
  start_date?: string
  end_date?: string
}

// ==================== 组织相关类型 ====================

/** 组织列表项（后端 OrganizationListItem） */
export interface OrgListItem {
  id: string
  name: string
  type: string           // region / school
  parent_id: string | null
  parent_name: string
  admin_user_id: string | null
  admin_user_name: string
  logo_url: string
  status: string
  group_count: number
  member_count: number
  settings?: string        // v172新增：组织设置JSON（含 portal_modules），编辑弹窗读取用
  created_at: string
}

/** 创建组织请求 */
export interface CreateOrgRequest {
  name: string
  type: string           // region / school
  parent_id?: string | null
  admin_user_id?: string | null
}

/** 更新组织请求 */
export interface UpdateOrgRequest {
  name: string
  admin_user_id?: string | null
  status?: string
  settings?: string
  /** Logo移除链路③：true=清空组织Logo（对应后端 UpdateOrganizationRequest.ClearLogo，缺省不发=不动Logo） */
  clear_logo?: boolean
}

/**
 * 教研组列表项（后端 TeachingGroupListItem）
 * v109改动：新增 lead_user_names（所有组长名称，中文顿号分隔）
 */
export interface GroupListItem {
  id: string
  name: string
  school_id: string
  school_name: string
  subject: string
  grade_range: string
  lead_user_id: string | null    // 兼容保留
  lead_user_name: string         // 兼容保留（第一个组长名称）
  lead_user_names: string        // v109新增：所有组长名称，如"张老师、李老师"
  member_count: number
  status: string
  created_at: string
}

/**
 * 教研组成员列表项（后端 GroupMemberItem）
 * v109改动：role 支持 'lead' / 'backbone' / 'member'
 */
export interface GroupMemberItem {
  id: string
  user_id: string
  username: string
  display_name: string
  role: string           // lead=组长 / backbone=骨干 / member=普通
  joined_at: string | null
}

/**
 * 教研组详情（含成员列表，后端 TeachingGroupDetailResponse）
 * v109改动：新增 lead_user_names
 */
export interface GroupDetail {
  id: string
  name: string
  school_id: string
  school_name: string
  subject: string
  grade_range: string
  lead_user_id: string | null    // 兼容保留
  lead_user_name: string         // 兼容保留
  lead_user_names: string        // v109新增：所有组长名称
  description: string
  settings: string
  status: string
  members: GroupMemberItem[]
  created_at: string
  updated_at: string
}

/**
 * 创建教研组请求
 * v109改动：移除 lead_user_id（改由成员角色管理多组长）
 */
export interface CreateGroupRequest {
  name: string
  school_id: string
  subject: string
  grade_range?: string
  description?: string
}

/**
 * 更新教研组请求
 * v109改动：移除 lead_user_id
 */
export interface UpdateGroupRequest {
  name: string
  subject: string
  grade_range?: string
  description?: string
  status?: string
}

// ==================== 组织多管理员类型（Phase6.2 预埋 + B13 任命同步身份）====================

/**
 * 组织管理员列表项（后端 OrganizationAdminItem）
 * 对应 organization_admins 表，一个组织可有多个管理员。
 * role_type：region_admin（区域管理员，挂在 region 组织）/ school_admin（学校管理员，挂在 school 组织）
 */
export interface OrgAdminItem {
  id: string
  org_id: string
  user_id: string
  username: string
  display_name: string
  role_type: string        // region_admin / school_admin
  created_by: string       // 任命人（后端 COALESCE 为字符串，可能为空串）
  created_at: string
}

/**
 * 任命组织管理员请求
 * B13新增 sync_role：true=任命时同步升级目标的系统身份（仅 operator/viewer 起步，
 *   region 任命→region_admin / school 任命→senior_operator；已具管理身份者不动）。
 *   前端默认勾选 true；不传/false 则行为与 B13 前完全一致（只写组织身份）。
 */
export interface AddOrgAdminRequest {
  user_id: string
  role_type: string        // region_admin / school_admin
  sync_role?: boolean      // B13：是否同步升级系统身份
}

/**
 * 任命组织管理员结果（后端 AddOrgAdmin 响应，B13新增）
 *   message     后端按四种结果拼好的中文文案，前端原样 toast 即可
 *   role_synced true=已同步升级系统身份（目标需重新登录后生效新入口）
 *   new_role    同步后的新身份（仅 role_synced=true 时非空）
 */
export interface AddOrgAdminResult {
  message: string
  role_synced: boolean
  new_role: string
}

// ==================== 统计 API ====================

export async function getAdminStats(): Promise<AdminStats> {
  const res = await client.get<{ code: number; data: AdminStats }>('/admin/stats')
  return res.data.data!
}

// ==================== 用户管理 API ====================

export async function getAdminUsers(params: AdminUserListParams = {}): Promise<AdminUserListResult> {
  const res = await client.get<{ code: number; data: AdminUserListResult }>('/admin/users', { params })
  return res.data.data!
}

export async function getAdminUserDetail(id: string): Promise<AdminUserDetail> {
  const res = await client.get<{ code: number; data: AdminUserDetail }>(`/admin/users/${id}`)
  return res.data.data!
}

export async function createAdminUser(data: {
  username: string; display_name: string; password: string; role: string
}): Promise<AdminUserListItem> {
  const res = await client.post<{ code: number; data: AdminUserListItem }>('/admin/users', data)
  return res.data.data!
}

export async function updateAdminUser(id: string, data: {
  display_name: string; role: string
}): Promise<AdminUserListItem> {
  const res = await client.put<{ code: number; data: AdminUserListItem }>(`/admin/users/${id}`, data)
  return res.data.data!
}

export async function updateAdminUserStatus(id: string, status: 'active' | 'disabled'): Promise<void> {
  await client.put(`/admin/users/${id}/status`, { status })
}

export async function resetAdminUserPassword(id: string, new_password: string): Promise<void> {
  await client.put(`/admin/users/${id}/password`, { new_password })
}

export async function getAdminUserAssignments(id: string): Promise<AdminCourseAssignment[]> {
  const res = await client.get<{ code: number; data: AdminCourseAssignment[] }>(`/admin/users/${id}/assignments`)
  return res.data.data ?? []
}

export async function updateAdminUserAssignments(id: string, course_codes: string[]): Promise<void> {
  await client.put(`/admin/users/${id}/assignments`, { course_codes })
}

// ==================== 用户↔教研组双向分配 API ====================

export async function addUserToGroup(
  userId: string,
  data: { group_id: string; role: string }
): Promise<void> {
  await client.post(`/admin/users/${userId}/groups`, data)
}

export async function removeUserFromGroup(userId: string, groupId: string): Promise<void> {
  await client.delete(`/admin/users/${userId}/groups/${groupId}`)
}

// ==================== 移出本校 API（归属治理批B新增，R3）====================

/**
 * 将用户移出某学校 DELETE /admin/users/{uid}/schools/{sid}
 * 后端单事务：退出该校全部教研组 + 删除校籍行（任一步失败整体回滚）。
 * 返回后端拼好的中文 message（含连带退出的组数），供前端直接 toast。
 * 权限：admin 任意校；senior 仅本校且目标须教师级；region 只读被拦（403 由调用方 catch）。
 */
export async function removeUserFromSchool(userId: string, schoolId: string, removeAdmin?: boolean): Promise<string> {
  // 批C2：removeAdmin=true 时后端先移除其该校管理员任命（走末任命自动降级链路），再移出本校
  const res = await client.delete<{ code: number; data: { message: string } }>(
    `/admin/users/${userId}/schools/${schoolId}${removeAdmin ? '?remove_admin=1' : ''}`
  )
  return res.data.data?.message ?? '移出成功'
}

// ==================== 组织管理 API ====================

export async function getAdminOrgs(params?: {
  type?: string
  parent_id?: string
}): Promise<OrgListItem[]> {
  const res = await client.get<{
    code: number
    data: { organizations: OrgListItem[]; total: number }
  }>('/lesson-plans/organizations', { params })
  return res.data.data?.organizations ?? []
}

export async function getAdminOrg(id: string): Promise<OrgListItem> {
  const res = await client.get<{ code: number; data: OrgListItem }>(
    `/lesson-plans/organizations/${id}`
  )
  return res.data.data!
}

export async function createAdminOrg(data: CreateOrgRequest): Promise<OrgListItem> {
  const res = await client.post<{ code: number; data: OrgListItem }>(
    '/lesson-plans/organizations', data
  )
  return res.data.data!
}

export async function updateAdminOrg(id: string, data: UpdateOrgRequest): Promise<void> {
  await client.put(`/lesson-plans/organizations/${id}`, data)
}

export async function deleteAdminOrg(id: string): Promise<void> {
  await client.delete(`/lesson-plans/organizations/${id}`)
}

/** 上传组织Logo */
export async function uploadOrgLogo(orgId: string, file: File): Promise<{ url: string }> {
  const formData = new FormData()
  formData.append('file', file)
  const res = await client.post<{ code: number; data: { url: string } }>(`/admin/orgs/${orgId}/upload-logo`, formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return res.data.data!
}

// ==================== 组织多管理员 API（Phase6.2 + B13 任命同步身份）====================
// 后端路由：/api/v1/lesson-plans/organizations/{id}/admins
//   GET    列出某组织的全部管理员
//   POST   任命一名管理员（body: {user_id, role_type, sync_role}）
//   DELETE 移除某管理员（/admins/{user_id}）——B13明确：移除永不降级系统身份
// 权限：admin 任意；region_admin 仅其管辖区域本身或辖区学校（后端 service 二次校验）

/** 列出某组织的全部管理员 */
export async function getOrgAdmins(orgId: string): Promise<OrgAdminItem[]> {
  const res = await client.get<{ code: number; data: { admins: OrgAdminItem[]; total: number } }>(
    `/lesson-plans/organizations/${orgId}/admins`
  )
  return res.data.data?.admins ?? []
}

/**
 * 任命一名组织管理员（B13：返回后端拼好的结果供 UI 直接 toast）
 * sync_role=true 时若目标身份为 operator/viewer，任命成功后同步升级其系统身份
 */
export async function addOrgAdmin(orgId: string, data: AddOrgAdminRequest): Promise<AddOrgAdminResult> {
  const res = await client.post<{ code: number; data: AddOrgAdminResult }>(
    `/lesson-plans/organizations/${orgId}/admins`, data
  )
  return res.data.data!
}

/** 移除某组织管理员 */
export async function removeOrgAdmin(orgId: string, userId: string): Promise<void> {
  await client.delete(`/lesson-plans/organizations/${orgId}/admins/${userId}`)
}

// ==================== 教研组管理 API ====================

export async function getAdminGroups(school_id?: string): Promise<GroupListItem[]> {
  const res = await client.get<{
    code: number
    data: { groups: GroupListItem[]; total: number }
  }>('/lesson-plans/teaching-groups', {
    params: school_id ? { school_id } : {}
  })
  return res.data.data?.groups ?? []
}

export async function getAdminGroupDetail(id: string): Promise<GroupDetail> {
  const res = await client.get<{ code: number; data: GroupDetail }>(
    `/lesson-plans/teaching-groups/${id}`
  )
  return res.data.data!
}

export async function createAdminGroup(data: CreateGroupRequest): Promise<GroupListItem> {
  const res = await client.post<{ code: number; data: GroupListItem }>(
    '/lesson-plans/teaching-groups', data
  )
  return res.data.data!
}

export async function updateAdminGroup(id: string, data: UpdateGroupRequest): Promise<void> {
  await client.put(`/lesson-plans/teaching-groups/${id}`, data)
}

export async function deleteAdminGroup(id: string): Promise<void> {
  await client.delete(`/lesson-plans/teaching-groups/${id}`)
}

// ==================== 教研组成员管理 API ====================

export async function getAdminGroupMembers(groupId: string): Promise<GroupMemberItem[]> {
  const res = await client.get<{ code: number; data: GroupMemberItem[] }>(
    `/admin/groups/${groupId}/members`
  )
  return res.data.data ?? []
}

export async function addAdminGroupMember(groupId: string, data: {
  user_id: string; role?: string
}): Promise<void> {
  await client.post(`/admin/groups/${groupId}/members`, data)
}

export async function updateAdminGroupMemberRole(
  groupId: string, userId: string, role: string
): Promise<void> {
  await client.put(`/admin/groups/${groupId}/members/${userId}`, { role })
}

export async function removeAdminGroupMember(groupId: string, userId: string): Promise<void> {
  await client.delete(`/admin/groups/${groupId}/members/${userId}`)
}

// ==================== 操作日志 API ====================

export async function getAdminAuditLogs(
  params: AuditLogQueryParams = {}
): Promise<AuditLogListResult> {
  const res = await client.get<{ code: number; data: AuditLogListResult }>(
    '/admin/audit-logs', { params }
  )
  return res.data.data!
}

// ==================== 批量导入用户 API（合并重构：替代 school-admin 批量端点）====================
// 后端路由：POST /api/v1/admin/users/batch（adminOrSchoolAdmin 中间件）
//   - admin          : 必须传 school_id（目标学校），后端按此入校；为空后端返回 400
//   - senior_operator: 可省 school_id，后端强制本校（忽略前端传入）
//   - 角色白名单：operator / viewer（后端二次强制）
//   - 整批回滚：任一行失败则一个都不创建，返回行号失败明细供前端逐行标红

/** 批量导入的单个用户条目（与后端 services.BatchUserItem 对齐） */
export interface BatchUserItem {
  username: string
  display_name: string
  password: string
}

/** 批量建用户请求（前端只传 role + school_id + users；source 由后端按角色强制） */
export interface BatchCreateAdminUsersRequest {
  role: string                 // operator / viewer
  school_id?: string           // admin 必填（目标学校）；senior 可省（后端强制本校）
  users: BatchUserItem[]
}

/** 单行失败明细（行号 1-based，对齐操作员看到的表格行） */
export interface BatchCreateUserFailure {
  index: number
  username: string
  reason: string
}

/** 批量建用户结果（与后端 services.BatchCreateUsersResult 对齐） */
export interface BatchCreateUsersResult {
  success: boolean
  created_count: number
  total_count: number
  failures: BatchCreateUserFailure[]
}

/**
 * 批量创建用户 POST /admin/users/batch
 * 后端整批回滚 + 行号失败明细：
 *   - 成功 → success=true, created_count=条目数, failures=[]
 *   - 整批校验失败 → success=false, created_count=0, failures 逐行原因（HTTP 仍 200）
 *   - 系统级异常（如空列表、开事务失败）→ 抛错（HTTP 非 200，由调用方 catch）
 */
export async function batchCreateAdminUsers(
  req: BatchCreateAdminUsersRequest
): Promise<BatchCreateUsersResult> {
  const res = await client.post<{ code: number; data: BatchCreateUsersResult }>(
    '/admin/users/batch', req
  )
  return res.data.data!
}


// ==================== 学校境外模型授权策略 API（批二新增，admin专属）====================
// 后端路由：/api/v1/admin/school-model-policies（adminOnly）
//   GET    /school-model-policies            列出全部已登记策略的学校
//   GET    /school-model-policies/{schoolID} 查单校当前策略
//   PUT    /school-model-policies/{schoolID} 授权/取消授权（body: {overseas_enabled, note}）
//   DELETE /school-model-policies/{schoolID} 删除记录（=回到默认境内）
// 业务：默认所有学校只能用境内模型(qwen-max)；仅被授权的学校放行境外模型(claude/gemini等)。

/** 学校授权策略列表项（后端 SchoolModelPolicyItem） */
export interface SchoolModelPolicyItem {
  school_id: string
  school_name: string          // JOIN organizations 填学校名
  overseas_enabled: boolean    // 是否授权境外模型
  note: string                 // 备注
  granted_by_name: string      // 授权人显示名（可空时为空串）
  created_at: string
  updated_at: string
}

/** 单校策略视图（后端 smpPolicyView），无记录时 has_record=false 表默认境内 */
export interface SchoolModelPolicyView {
  school_id: string
  overseas_enabled: boolean
  note: string
  has_record: boolean          // false=该校从未登记策略（=默认境内）
}

/** 列出全部已登记策略的学校 */
export async function getSchoolModelPolicies(): Promise<SchoolModelPolicyItem[]> {
  const res = await client.get<{ code: number; data: { items: SchoolModelPolicyItem[]; total: number } }>(
    '/admin/school-model-policies'
  )
  return res.data.data?.items ?? []
}

/** 查单个学校的当前策略（无记录返回默认境内态 has_record=false） */
export async function getSchoolModelPolicy(schoolId: string): Promise<SchoolModelPolicyView> {
  const res = await client.get<{ code: number; data: SchoolModelPolicyView }>(
    `/admin/school-model-policies/${schoolId}`
  )
  return res.data.data!
}

/** 授权/取消授权某学校走境外模型（overseas_enabled + 可选 note）
 *  分流模块对「学校是否授权」是每次实时查库，无缓存，保存后即时生效 */
export async function setSchoolModelPolicy(
  schoolId: string,
  data: { overseas_enabled: boolean; note?: string }
): Promise<SchoolModelPolicyView> {
  const res = await client.put<{ code: number; data: SchoolModelPolicyView }>(
    `/admin/school-model-policies/${schoolId}`,
    { overseas_enabled: data.overseas_enabled, note: data.note ?? '' }
  )
  return res.data.data!
}

/** 删除某学校策略记录（=回到默认境内） */
export async function deleteSchoolModelPolicy(schoolId: string): Promise<void> {
  await client.delete(`/admin/school-model-policies/${schoolId}`)
}


// ==================== 跨区域多校批量导入 API（跨校批量·新增）====================
// 后端路由（均 adminOnly，仅系统管理员）：
//   GET  /api/v1/admin/region-schools?region_id=xxx        → 取某区域下全部 active 学校(id+name)，生成模板用
//   POST /api/v1/admin/users/batch-multi-school            → 跨校批量建用户(每行自带 school_id)
//
// 与单校批量(batchCreateAdminUsers)的区别：
//   - 单校：整批一个 school_id + 整批回滚；
//   - 跨校：每行各自 school_id + 逐行成败 + 重名自动改名(teacher01→teacher01_2…并回显)。
// 跨校是 admin 专属能力(senior/region_admin 不用)。

/** 区域学校项（生成模板用：学校名 + ID，下拉与"学校名→ID"映射的权威来源） */
export interface RegionSchoolItem {
  id: string
  name: string
}

/**
 * 取某区域下全部 active 学校（id+name）
 * 用途：跨校批量下载 Excel 模板时，admin 选区域后拉该区域学校清单，
 *       供"多选学校 + 模板内学校下拉列 + 学校名→ID 映射"使用。
 */
export async function getRegionSchools(regionId: string): Promise<RegionSchoolItem[]> {
  const res = await client.get<{ code: number; data: { schools: RegionSchoolItem[]; total: number } }>(
    '/admin/region-schools', { params: { region_id: regionId } }
  )
  return res.data.data?.schools ?? []
}

/** 跨校批量的单个用户条目（每行自带 school_id，与后端 services.MultiSchoolUserItem 对齐） */
export interface MultiSchoolUserItem {
  username: string
  display_name: string
  password: string
  school_id: string            // 该老师所属学校(前端由"学校名→ID"反查填入)
}

/** 跨校批量建用户请求（role 批次统一 operator/viewer；每行自带 school_id） */
export interface MultiSchoolBatchRequest {
  role: string
  users: MultiSchoolUserItem[]
}

/** 跨校成功建成的单行明细（含改名回显，admin 据此通知到人） */
export interface MultiSchoolCreatedItem {
  index: number                // 行号(1-based)
  original_username: string    // Excel 里填的原始用户名
  final_username: string       // 实际建成的用户名(改名则为 xxx_N)
  renamed: boolean             // 是否发生了自动改名
  school_id: string            // 入校的学校ID
  display_name: string         // 教师姓名(通知清单展示用)
}

/** 跨校失败的单行明细 */
export interface MultiSchoolFailureItem {
  index: number                // 行号(1-based)
  username: string
  school_id: string
  reason: string               // 失败原因(中文)
}

/** 跨校批量结果（与后端 services.MultiSchoolBatchResult 对齐） */
export interface MultiSchoolBatchResult {
  total_count: number
  created_count: number
  failed_count: number
  created: MultiSchoolCreatedItem[]    // 成功明细(含改名清单)
  failures: MultiSchoolFailureItem[]   // 失败明细
}

/**
 * 跨校批量创建用户 POST /admin/users/batch-multi-school
 * 逐行成败 + 重名自动改名：
 *   - 正常处理 → 返回 result(created 含改名清单 + failures 明细)，HTTP 200；
 *   - 前置异常(行数超 2000 / 角色非法 / 空列表) → 后端 400，由调用方 catch；
 *   - 系统级异常(查库失败) → 后端 500，由调用方 catch。
 * 注意：与单校不同，本接口【没有】整批回滚——能建的已经建好，失败的仅在 failures 列出。
 */
export async function batchCreateMultiSchoolUsers(
  req: MultiSchoolBatchRequest
): Promise<MultiSchoolBatchResult> {
  const res = await client.post<{ code: number; data: MultiSchoolBatchResult }>(
    '/admin/users/batch-multi-school', req
  )
  return res.data.data!
}

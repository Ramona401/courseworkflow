/**
 * 统一用户管理中心 API 封装
 * 对应后端 /api/v1/admin/* 和
 * /api/v1/lesson-plans/organizations/* 路由。
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

export interface AdminSchoolMembership {
  school_id: string
  school_name: string
  source: string
  source_name: string
  joined_at: string
  group_count: number
  is_school_admin: boolean
}

export interface AdminUserDetail extends AdminUserListItem {
  course_assignments: AdminCourseAssignment[]
  teaching_groups: AdminGroupMembership[]
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

/**
 * 学校可使用的三个具体教学教育域。
 * mixed只用于区域或平台管理上下文。
 */
export type TeachingEducationDomain =
  | 'k12'
  | 'vocational'
  | 'adult'

/** 组织自身的合法教育域。 */
export type OrganizationEducationDomain =
  | TeachingEducationDomain
  | 'mixed'

export interface OrgListItem {
  id: string
  name: string
  type: string
  education_domain?: OrganizationEducationDomain
  parent_id: string | null
  parent_name: string
  admin_user_id: string | null
  admin_user_name: string
  logo_url: string
  status: string
  group_count: number
  member_count: number
  settings?: string
  created_at: string
}

/**
 * 创建组织请求。
 *
 * school：
 *   education_domain必填，只允许k12/vocational/adult。
 *
 * region：
 *   不提交education_domain，后端始终强制写mixed。
 */
export interface CreateOrgRequest {
  name: string
  type: string
  education_domain?: TeachingEducationDomain
  parent_id?: string | null
  admin_user_id?: string | null
}

export interface UpdateOrgRequest {
  name: string
  admin_user_id?: string | null
  status?: string
  settings?: string
  clear_logo?: boolean
}

export interface GroupListItem {
  id: string
  name: string
  school_id: string
  school_name: string
  subject: string
  grade_range: string
  lead_user_id: string | null
  lead_user_name: string
  lead_user_names: string
  member_count: number
  status: string
  created_at: string
}

export interface GroupMemberItem {
  id: string
  user_id: string
  username: string
  display_name: string
  role: string
  joined_at: string | null
}

export interface GroupDetail {
  id: string
  name: string
  school_id: string
  school_name: string
  subject: string
  grade_range: string
  lead_user_id: string | null
  lead_user_name: string
  lead_user_names: string
  description: string
  settings: string
  status: string
  members: GroupMemberItem[]
  created_at: string
  updated_at: string
}

export interface CreateGroupRequest {
  name: string
  school_id: string
  subject: string
  grade_range?: string
  description?: string
}

export interface UpdateGroupRequest {
  name: string
  subject: string
  grade_range?: string
  description?: string
  status?: string
}

// ==================== 组织多管理员类型 ====================

export interface OrgAdminItem {
  id?: string
  org_id: string
  user_id: string
  username: string
  display_name: string
  role_type: string
  education_domain: string
  created_by: string
  created_at: string
}

export interface OrgAdminManagementResult {
  admins: OrgAdminItem[]
  total: number
  available_education_domains: TeachingEducationDomain[]
}

export interface AddOrgAdminRequest {
  user_id: string
  role_type: string
  education_domain?: TeachingEducationDomain
  sync_role?: boolean
}

export interface AddOrgAdminResult {
  message: string
  role_synced: boolean
  new_role: string
  education_domain: string
}

// ==================== 统计 API ====================

export async function getAdminStats(): Promise<AdminStats> {
  const res = await client.get<{
    code: number
    data: AdminStats
  }>('/admin/stats')

  return res.data.data!
}

// ==================== 用户管理 API ====================

export async function getAdminUsers(
  params: AdminUserListParams = {},
): Promise<AdminUserListResult> {
  const res = await client.get<{
    code: number
    data: AdminUserListResult
  }>('/admin/users', { params })

  return res.data.data!
}

export async function getAdminUserDetail(
  id: string,
): Promise<AdminUserDetail> {
  const res = await client.get<{
    code: number
    data: AdminUserDetail
  }>(`/admin/users/${id}`)

  return res.data.data!
}

export async function createAdminUser(data: {
  username: string
  display_name: string
  password: string
  role: string
}): Promise<AdminUserListItem> {
  const res = await client.post<{
    code: number
    data: AdminUserListItem
  }>('/admin/users', data)

  return res.data.data!
}

export async function updateAdminUser(
  id: string,
  data: {
    display_name: string
    role: string
  },
): Promise<AdminUserListItem> {
  const res = await client.put<{
    code: number
    data: AdminUserListItem
  }>(`/admin/users/${id}`, data)

  return res.data.data!
}

export async function updateAdminUserStatus(
  id: string,
  status: 'active' | 'disabled',
): Promise<void> {
  await client.put(
    `/admin/users/${id}/status`,
    { status },
  )
}

export async function resetAdminUserPassword(
  id: string,
  new_password: string,
): Promise<void> {
  await client.put(
    `/admin/users/${id}/password`,
    { new_password },
  )
}

export async function getAdminUserAssignments(
  id: string,
): Promise<AdminCourseAssignment[]> {
  const res = await client.get<{
    code: number
    data: AdminCourseAssignment[]
  }>(`/admin/users/${id}/assignments`)

  return res.data.data ?? []
}

export async function updateAdminUserAssignments(
  id: string,
  course_codes: string[],
): Promise<void> {
  await client.put(
    `/admin/users/${id}/assignments`,
    { course_codes },
  )
}

// ==================== 用户↔教研组分配 API ====================

export async function addUserToGroup(
  userId: string,
  data: {
    group_id: string
    role: string
  },
): Promise<void> {
  await client.post(
    `/admin/users/${userId}/groups`,
    data,
  )
}

export async function removeUserFromGroup(
  userId: string,
  groupId: string,
): Promise<void> {
  await client.delete(
    `/admin/users/${userId}/groups/${groupId}`,
  )
}

// ==================== 移出本校 API ====================

export async function removeUserFromSchool(
  userId: string,
  schoolId: string,
  removeAdmin?: boolean,
): Promise<string> {
  const suffix = removeAdmin
    ? '?remove_admin=1'
    : ''

  const res = await client.delete<{
    code: number
    data: {
      message: string
    }
  }>(
    `/admin/users/${userId}/schools/${schoolId}${suffix}`,
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
    data: {
      organizations: OrgListItem[]
      total: number
    }
  }>('/lesson-plans/organizations', { params })

  return res.data.data?.organizations ?? []
}

export async function getAdminOrg(
  id: string,
): Promise<OrgListItem> {
  const res = await client.get<{
    code: number
    data: OrgListItem
  }>(`/lesson-plans/organizations/${id}`)

  return res.data.data!
}

export async function createAdminOrg(
  data: CreateOrgRequest,
): Promise<OrgListItem> {
  const res = await client.post<{
    code: number
    data: OrgListItem
  }>('/lesson-plans/organizations', data)

  return res.data.data!
}

export async function updateAdminOrg(
  id: string,
  data: UpdateOrgRequest,
): Promise<void> {
  await client.put(
    `/lesson-plans/organizations/${id}`,
    data,
  )
}

export async function deleteAdminOrg(
  id: string,
): Promise<void> {
  await client.delete(
    `/lesson-plans/organizations/${id}`,
  )
}

export async function uploadOrgLogo(
  orgId: string,
  file: File,
): Promise<{ url: string }> {
  const formData = new FormData()
  formData.append('file', file)

  const res = await client.post<{
    code: number
    data: {
      url: string
    }
  }>(
    `/admin/orgs/${orgId}/upload-logo`,
    formData,
    {
      headers: {
        'Content-Type': 'multipart/form-data',
      },
    },
  )

  return res.data.data!
}

// ==================== 组织多管理员 API ====================

export async function getOrgAdminManagement(
  orgId: string,
): Promise<OrgAdminManagementResult> {
  const res = await client.get<{
    code: number
    data: OrgAdminManagementResult
  }>(
    `/lesson-plans/organizations/${orgId}/admins`,
  )

  return {
    admins: res.data.data?.admins ?? [],
    total: res.data.data?.total ?? 0,
    available_education_domains:
      res.data.data?.available_education_domains ?? [],
  }
}

export async function getOrgAdmins(
  orgId: string,
): Promise<OrgAdminItem[]> {
  const result = await getOrgAdminManagement(orgId)
  return result.admins
}

export async function addOrgAdmin(
  orgId: string,
  data: AddOrgAdminRequest,
): Promise<AddOrgAdminResult> {
  const res = await client.post<{
    code: number
    data: AddOrgAdminResult
  }>(
    `/lesson-plans/organizations/${orgId}/admins`,
    data,
  )

  return res.data.data!
}

export async function removeOrgAdmin(
  orgId: string,
  userId: string,
): Promise<void> {
  await client.delete(
    `/lesson-plans/organizations/${orgId}/admins/${userId}`,
  )
}

// ==================== 教研组管理 API ====================

export async function getAdminGroups(
  school_id?: string,
): Promise<GroupListItem[]> {
  const res = await client.get<{
    code: number
    data: {
      groups: GroupListItem[]
      total: number
    }
  }>('/lesson-plans/teaching-groups', {
    params: school_id
      ? { school_id }
      : {},
  })

  return res.data.data?.groups ?? []
}

export async function getAdminGroupDetail(
  id: string,
): Promise<GroupDetail> {
  const res = await client.get<{
    code: number
    data: GroupDetail
  }>(`/lesson-plans/teaching-groups/${id}`)

  return res.data.data!
}

export async function createAdminGroup(
  data: CreateGroupRequest,
): Promise<GroupListItem> {
  const res = await client.post<{
    code: number
    data: GroupListItem
  }>('/lesson-plans/teaching-groups', data)

  return res.data.data!
}

export async function updateAdminGroup(
  id: string,
  data: UpdateGroupRequest,
): Promise<void> {
  await client.put(
    `/lesson-plans/teaching-groups/${id}`,
    data,
  )
}

export async function deleteAdminGroup(
  id: string,
): Promise<void> {
  await client.delete(
    `/lesson-plans/teaching-groups/${id}`,
  )
}

// ==================== 教研组成员管理 API ====================

export async function getAdminGroupMembers(
  groupId: string,
): Promise<GroupMemberItem[]> {
  const res = await client.get<{
    code: number
    data: GroupMemberItem[]
  }>(`/admin/groups/${groupId}/members`)

  return res.data.data ?? []
}

export async function addAdminGroupMember(
  groupId: string,
  data: {
    user_id: string
    role?: string
  },
): Promise<void> {
  await client.post(
    `/admin/groups/${groupId}/members`,
    data,
  )
}

export async function updateAdminGroupMemberRole(
  groupId: string,
  userId: string,
  role: string,
): Promise<void> {
  await client.put(
    `/admin/groups/${groupId}/members/${userId}`,
    { role },
  )
}

export async function removeAdminGroupMember(
  groupId: string,
  userId: string,
): Promise<void> {
  await client.delete(
    `/admin/groups/${groupId}/members/${userId}`,
  )
}

// ==================== 操作日志 API ====================

export async function getAdminAuditLogs(
  params: AuditLogQueryParams = {},
): Promise<AuditLogListResult> {
  const res = await client.get<{
    code: number
    data: AuditLogListResult
  }>('/admin/audit-logs', { params })

  return res.data.data!
}

// ==================== 批量导入用户 API ====================

export interface BatchUserItem {
  username: string
  display_name: string
  password: string
}

export interface BatchCreateAdminUsersRequest {
  role: string
  school_id?: string
  users: BatchUserItem[]
}

export interface BatchCreateUserFailure {
  index: number
  username: string
  reason: string
}

export interface BatchCreateUsersResult {
  success: boolean
  created_count: number
  total_count: number
  failures: BatchCreateUserFailure[]
}

export async function batchCreateAdminUsers(
  req: BatchCreateAdminUsersRequest,
): Promise<BatchCreateUsersResult> {
  const res = await client.post<{
    code: number
    data: BatchCreateUsersResult
  }>('/admin/users/batch', req)

  return res.data.data!
}

// ==================== 学校境外模型授权策略 API ====================

export interface SchoolModelPolicyItem {
  school_id: string
  school_name: string
  overseas_enabled: boolean
  note: string
  granted_by_name: string
  created_at: string
  updated_at: string
}

export interface SchoolModelPolicyView {
  school_id: string
  overseas_enabled: boolean
  note: string
  has_record: boolean
}

export async function getSchoolModelPolicies():
Promise<SchoolModelPolicyItem[]> {
  const res = await client.get<{
    code: number
    data: {
      items: SchoolModelPolicyItem[]
      total: number
    }
  }>('/admin/school-model-policies')

  return res.data.data?.items ?? []
}

export async function getSchoolModelPolicy(
  schoolId: string,
): Promise<SchoolModelPolicyView> {
  const res = await client.get<{
    code: number
    data: SchoolModelPolicyView
  }>(`/admin/school-model-policies/${schoolId}`)

  return res.data.data!
}

export async function setSchoolModelPolicy(
  schoolId: string,
  data: {
    overseas_enabled: boolean
    note?: string
  },
): Promise<SchoolModelPolicyView> {
  const res = await client.put<{
    code: number
    data: SchoolModelPolicyView
  }>(
    `/admin/school-model-policies/${schoolId}`,
    {
      overseas_enabled: data.overseas_enabled,
      note: data.note ?? '',
    },
  )

  return res.data.data!
}

export async function deleteSchoolModelPolicy(
  schoolId: string,
): Promise<void> {
  await client.delete(
    `/admin/school-model-policies/${schoolId}`,
  )
}

// ==================== 跨区域多校批量导入 API ====================

export interface RegionSchoolItem {
  id: string
  name: string
}

export async function getRegionSchools(
  regionId: string,
): Promise<RegionSchoolItem[]> {
  const res = await client.get<{
    code: number
    data: {
      schools: RegionSchoolItem[]
      total: number
    }
  }>('/admin/region-schools', {
    params: {
      region_id: regionId,
    },
  })

  return res.data.data?.schools ?? []
}

export interface MultiSchoolUserItem {
  username: string
  display_name: string
  password: string
  school_id: string
}

export interface MultiSchoolBatchRequest {
  role: string
  users: MultiSchoolUserItem[]
}

export interface MultiSchoolCreatedItem {
  index: number
  original_username: string
  final_username: string
  renamed: boolean
  school_id: string
  display_name: string
}

export interface MultiSchoolFailureItem {
  index: number
  username: string
  school_id: string
  reason: string
}

export interface MultiSchoolBatchResult {
  total_count: number
  created_count: number
  failed_count: number
  created: MultiSchoolCreatedItem[]
  failures: MultiSchoolFailureItem[]
}

export async function batchCreateMultiSchoolUsers(
  req: MultiSchoolBatchRequest,
): Promise<MultiSchoolBatchResult> {
  const res = await client.post<{
    code: number
    data: MultiSchoolBatchResult
  }>('/admin/users/batch-multi-school', req)

  return res.data.data!
}

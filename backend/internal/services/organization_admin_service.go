package services

// organization_admin_service.go — 组织多管理员业务逻辑（迭代一 Phase 5）
//
// 职责：在 repository 的多管理员 CRUD 之上，封装"任命/移除/列出组织管理员"的业务规则：
//   1. 权限分层：
//      - admin            ：可对任何组织任命/移除（region 组织任 region_admin / school 组织任 school_admin）
//      - region_admin     ：仅可对"自己辖区内的学校"任命/移除 school_admin（辖区由 organization_admins 权威+递归学校确定）
//      - 其它角色          ：一律拒绝
//   2. 类型匹配：role_type 必须与组织 type 对应（region→region_admin / school→school_admin），否则拒绝。
//   3. 单字段回填（保证现有 GetSchoolByAdminUserID 单字段判定链不失效）：
//      - 任命 school_admin：若该校 admin_user_id 为空，则回填为新任命者（首个管理员占主字段）；已有则不覆盖。
//      - 移除 school_admin：若被移除者正是主管理员(admin_user_id 指向他)，从剩余 school_admin 挑首个补位；
//        无剩余则把 admin_user_id 置空。
//      - region_admin 任命/移除不涉及单字段（区域管辖完全走 organization_admins 表）。
//
// 独立成文件而非塞进 organization_service.go：后者已较大，多管理员逻辑聚合于此便于维护。
// OrganizationService 的方法接收者在 organization_service.go 已定义，本文件为其补充方法。

import (
        "context"
        "errors"

        "tedna/internal/models"
        "tedna/internal/repository"
)

// ==================== 多管理员相关错误常量 ====================

var (
        ErrOrgAdminUserRequired     = errors.New("管理员用户ID不能为空")
        ErrOrgAdminRoleTypeInvalid  = errors.New("无效的管理员类型，可选值：region_admin/school_admin")
        ErrOrgAdminRoleTypeMismatch = errors.New("管理员类型与组织类型不匹配（区域只能任命区域管理员，学校只能任命学校管理员）")
        ErrOrgAdminNoPermission     = errors.New("您没有权限管理该组织的管理员")
        ErrOrgAdminTargetUserNF     = errors.New("目标用户不存在")
)

// ListOrgAdmins 列出某组织的全部管理员
//
// 权限：admin 可看任何组织；region_admin 可看自己辖区内的组织（含自己管辖的区域本身及其下学校）；
//   senior_operator 可看自己所辖学校；其它角色拒绝。
func (s *OrganizationService) ListOrgAdmins(ctx context.Context, orgID string, callerRole string, callerID string) ([]*models.OrganizationAdminItem, error) {
        org, err := repository.GetOrganizationByID(ctx, orgID)
        if err != nil {
                if errors.Is(err, repository.ErrOrgNotFound) {
                        return nil, ErrOrgNotFound
                }
                return nil, err
        }

        // 权限校验：能否查看该组织的管理员列表
        if err := s.canManageOrgAdmins(ctx, org, callerRole, callerID); err != nil {
                return nil, err
        }

        items, err := repository.ListOrgAdmins(ctx, orgID)
        if err != nil {
                return nil, err
        }
        if items == nil {
                items = []*models.OrganizationAdminItem{}
        }
        return items, nil
}

// AddOrgAdmin 任命某用户为某组织的管理员
//
// 流程：
//   1. 校验组织存在、目标用户存在、role_type 合法且与组织类型匹配。
//   2. 权限校验（admin 任何组织 / region_admin 仅辖区学校 / 其它拒绝）。
//   3. 写入 organization_admins（幂等）。
//   4. 若任命的是 school_admin 且该校 admin_user_id 为空 → 回填主管理员单字段。
func (s *OrganizationService) AddOrgAdmin(ctx context.Context, orgID string, targetUserID string, roleType string, callerRole string, callerID string) error {
        if targetUserID == "" {
                return ErrOrgAdminUserRequired
        }
        if !models.IsValidOrgAdminRoleType(roleType) {
                return ErrOrgAdminRoleTypeInvalid
        }

        org, err := repository.GetOrganizationByID(ctx, orgID)
        if err != nil {
                if errors.Is(err, repository.ErrOrgNotFound) {
                        return ErrOrgNotFound
                }
                return err
        }

        // 类型匹配：region 组织只能任 region_admin，school 组织只能任 school_admin
        if !orgTypeMatchesAdminRole(org.Type, roleType) {
                return ErrOrgAdminRoleTypeMismatch
        }

        // 目标用户必须存在
        if _, uErr := repository.FindUserByID(ctx, targetUserID); uErr != nil {
                return ErrOrgAdminTargetUserNF
        }

        // 权限校验
        if err := s.canManageOrgAdmins(ctx, org, callerRole, callerID); err != nil {
                return err
        }

        // 写入多管理员表（幂等）
        if err := repository.AddOrgAdmin(ctx, orgID, targetUserID, roleType, callerID); err != nil {
                orgLog.Error("任命组织管理员失败", "org_id", orgID, "target", targetUserID, "role_type", roleType, "error", err)
                return err
        }

        // 回填主管理员单字段：仅 school_admin，且该校当前无主管理员时
        if roleType == models.OrgAdminRoleSchool {
                if org.AdminUserID == nil || *org.AdminUserID == "" {
                        if fErr := repository.UpdateOrganizationAdminUserID(ctx, orgID, &targetUserID); fErr != nil {
                                // 回填失败不回滚多管理员表（多管理员记录已是权威），仅记日志
                                orgLog.Error("回填主管理员单字段失败（多管理员已任命，不影响）", "org_id", orgID, "target", targetUserID, "error", fErr)
                        } else {
                                orgLog.Info("任命首个学校管理员并回填主字段", "org_id", orgID, "target", targetUserID)
                        }
                }
        }

        orgLog.Info("任命组织管理员成功", "org_id", orgID, "target", targetUserID, "role_type", roleType, "by", callerID)
        return nil
}

// RemoveOrgAdmin 移除某组织的某管理员
//
// 流程：
//   1. 校验组织存在 + 权限。
//   2. 移除前先记下该组织当前主管理员是谁（用于判断被移除者是否就是主管理员）。
//   3. 从 organization_admins 移除。
//   4. 若移除的是 school 组织的主管理员 → 从剩余 school_admin 挑首个补位回填；无剩余则置空。
func (s *OrganizationService) RemoveOrgAdmin(ctx context.Context, orgID string, targetUserID string, callerRole string, callerID string) error {
        if targetUserID == "" {
                return ErrOrgAdminUserRequired
        }

        org, err := repository.GetOrganizationByID(ctx, orgID)
        if err != nil {
                if errors.Is(err, repository.ErrOrgNotFound) {
                        return ErrOrgNotFound
                }
                return err
        }

        // 权限校验
        if err := s.canManageOrgAdmins(ctx, org, callerRole, callerID); err != nil {
                return err
        }

        // 记下移除前的主管理员指针
        wasPrimaryAdmin := org.AdminUserID != nil && *org.AdminUserID == targetUserID

        // 从多管理员表移除
        if err := repository.RemoveOrgAdmin(ctx, orgID, targetUserID); err != nil {
                if errors.Is(err, repository.ErrMemberNotFound) {
                        return ErrMemberNotFound
                }
                orgLog.Error("移除组织管理员失败", "org_id", orgID, "target", targetUserID, "error", err)
                return err
        }

        // 若移除的是 school 组织的主管理员，需要补位/置空主字段
        if org.Type == models.OrgTypeSchool && wasPrimaryAdmin {
                remaining, lErr := repository.ListSchoolAdminUserIDs(ctx, orgID)
                if lErr != nil {
                        orgLog.Error("查询剩余学校管理员失败（移除后未能补位主字段）", "org_id", orgID, "error", lErr)
                } else if len(remaining) > 0 {
                        // 挑首个补位（ListSchoolAdminUserIDs 按 created_at 升序）
                        newPrimary := remaining[0]
                        if fErr := repository.UpdateOrganizationAdminUserID(ctx, orgID, &newPrimary); fErr != nil {
                                orgLog.Error("补位主管理员单字段失败", "org_id", orgID, "new_primary", newPrimary, "error", fErr)
                        } else {
                                orgLog.Info("移除主管理员后已自动补位", "org_id", orgID, "new_primary", newPrimary)
                        }
                } else {
                        // 无剩余 school_admin → 主字段置空
                        if fErr := repository.UpdateOrganizationAdminUserID(ctx, orgID, nil); fErr != nil {
                                orgLog.Error("清空主管理员单字段失败", "org_id", orgID, "error", fErr)
                        } else {
                                orgLog.Info("移除最后一名学校管理员，主字段已置空", "org_id", orgID)
                        }
                }
        }

        orgLog.Info("移除组织管理员成功", "org_id", orgID, "target", targetUserID, "by", callerID)
        return nil
}

// ==================== 内部辅助 ====================

// orgTypeMatchesAdminRole 校验组织类型与管理员类型是否匹配
//   region 组织 ↔ region_admin；school 组织 ↔ school_admin
func orgTypeMatchesAdminRole(orgType string, roleType string) bool {
        switch orgType {
        case models.OrgTypeRegion:
                return roleType == models.OrgAdminRoleRegion
        case models.OrgTypeSchool:
                return roleType == models.OrgAdminRoleSchool
        default:
                return false
        }
}

// canManageOrgAdmins 判断 caller 是否有权管理(查看/任命/移除) 某组织的管理员
//
//   - admin            ：放行任何组织
//   - region_admin     ：仅放行"自己辖区内"的组织——
//        * 目标组织是自己直接管辖的区域本身；或
//        * 目标组织是 school 且其 parent_id 在自己管辖的区域集合内（用 ListDescendantSchoolIDs 校验更稳，但
//          当前两级结构下 school.parent_id ∈ 管辖区域集合 即等价；这里用辖区学校集合判断，兼容多级）
//   - 其它角色（含 senior_operator/operator/viewer/district_inspector）：拒绝
//
// 注：senior_operator 不在此放行——学校管理员管理"本校教师"，但"任命学校管理员"是上级（区域/系统）的职权，
//   学校管理员不能自己任命自己的同级或上级，避免权限自举。
func (s *OrganizationService) canManageOrgAdmins(ctx context.Context, org *models.Organization, callerRole string, callerID string) error {
        if callerRole == models.RoleAdmin {
                return nil
        }

        if callerRole == models.RoleRegionAdmin {
                // 取 caller 管辖的所有区域
                regionIDs, rErr := repository.ListRegionIDsByAdmin(ctx, callerID)
                if rErr != nil {
                        orgLog.Error("查询区域管理员辖区失败", "caller", callerID, "error", rErr)
                        return ErrOrgAdminNoPermission
                }
                if len(regionIDs) == 0 {
                        return ErrOrgAdminNoPermission
                }

                // 情况1：目标组织本身就是 caller 管辖的某个区域
                for _, rid := range regionIDs {
                        if rid == org.ID {
                                return nil
                        }
                }

                // 情况2：目标组织是 school，且落在 caller 辖区树下的学校集合内
                if org.Type == models.OrgTypeSchool {
                        for _, rid := range regionIDs {
                                schoolIDs, sErr := repository.ListDescendantSchoolIDs(ctx, rid)
                                if sErr != nil {
                                        orgLog.Error("递归查询辖区学校失败", "region", rid, "error", sErr)
                                        return ErrOrgAdminNoPermission
                                }
                                for _, sid := range schoolIDs {
                                        if sid == org.ID {
                                                return nil
                                        }
                                }
                        }
                }

                return ErrOrgAdminNoPermission
        }

        // 其它角色一律拒绝
        return ErrOrgAdminNoPermission
}

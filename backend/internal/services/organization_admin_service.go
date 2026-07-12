package services

// organization_admin_service.go — 组织多管理员业务逻辑（迭代一 Phase 5 + B13 任命即同步身份 + 批C 末任命自动降级）
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
// B13——任命即同步身份（根治"有管辖无门票"静默失效）：
//   背景：平台权限由两套规则 AND 组合——users.role(系统身份)是"门票"，决定前端卡片/
//   路由守卫/后端 RequireRole/ResolveDataScope 分支；organization_admins(组织身份)是
//   "管辖范围"，决定进门后能看到哪些数据。此前任命只写组织身份不动系统身份，
//   目标用户若还是 operator/viewer，登录后连用户管理入口都没有（B2 实测证实）。
//   规则（AddOrgAdmin 的 syncRole=true 时）：
//     - 期望身份：region 任命→region_admin；school 任命→senior_operator
//     - 仅当目标当前身份 ∈ {operator, viewer} 时才执行升级（白名单起步）
//     - admin / senior_operator / region_admin / district_inspector 一律不动——
//       senior 兼区域管辖已被 data_scope.buildUnionScope 双重身份并集支持，不自动改其身份
//     - 失败策略：任命是主操作先落库，身份同步失败不回滚任命，结果结构体标记
//       SyncFailed 供 handler 明示"请到用户管理手动修改"——最坏退化为 B13 之前的现状
//
// 批C（2026-07-04）——末任命自动降级，补齐 B13 的反向半条同步，闭合不变式：
//   「users.role=senior_operator/region_admin ⇔ 存在对应任命」
//   规则（RemoveOrgAdmin 移除成功之后，best-effort）：
//     - 目标身份为任命制身份(models.IsAppointmentOnlyRole)时，统计其剩余管辖绑定
//       (CountUserAdminBindings：organization_admins ∪ organizations.admin_user_id 双来源)；
//     - 归零 → 自动降级为骨干教师(operator)；仍持有任何任命/主字段指向 → 不降；
//     - admin/operator/viewer/district_inspector 永不受降级影响；
//     - 降级失败不回滚移除（移除是主操作），结果结构体标记 DowngradeFailed
//       供 handler 明示手工处理；成功由 handler 写 admin.org_admin_role_downgrade 审计。
//   顺序保证：主字段补位/置空发生在计数之前——被移除者若曾是主管理员，
//   admin_user_id 已不再指向他，计数不会被残留指针撑住。
//   （B13 时的"移除永不降级"担忧是误伤多任命用户；按"归零才降"即无此问题。）
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

// ==================== B13：任命结果结构体 ====================

// OrgAdminAddResult 任命组织管理员的结果（供 handler 拼装响应文案与审计）
//
// 四种结果形态与 handler 文案的对应关系：
//   RoleSynced=true                    → "任命成功，已同步身份为X"（写 role_sync 审计）
//   请求同步但 TargetRole 不在白名单     → RoleSynced=false, SyncFailed=false
//                                        → "任命成功；该用户现有身份为X，未变更"
//   请求同步但 UPDATE 失败              → SyncFailed=true
//                                        → "任命成功，但身份同步失败，请到用户管理手动修改"
//   未请求同步                          → 三标志均零值 → "任命成功"
type OrgAdminAddResult struct {
	RoleSynced bool   // 是否完成了系统身份同步升级
	NewRole    string // 同步后的新身份（仅 RoleSynced=true 时非空）
	SyncFailed bool   // 请求了同步但 users.role 更新失败（任命本身已成功）
	TargetRole string // 目标用户任命前的系统身份（供 handler 拼"现有身份为X"文案）
}

// ==================== 批C：移除结果结构体 ====================

// OrgAdminRemoveResult 移除组织管理员的结果（供 handler 拼装响应文案与审计）
//
// 三种结果形态：
//   RoleDowngraded=true → 目标任命归零，身份已自动降级为骨干教师
//                         （handler 写 admin.org_admin_role_downgrade 审计并在响应文案明示）
//   DowngradeFailed=true→ 应降级但 users.role 更新失败（移除本身已成功），明示手工处理
//   两标志均 false      → 无需降级（目标非任命制身份，或仍持有其他任命）
type OrgAdminRemoveResult struct {
	RoleDowngraded  bool   // 已自动降级
	FromRole        string // 降级前身份（仅降级流程触发时非空）
	NewRole         string // 降级后身份（恒为 operator，仅 RoleDowngraded=true 时非空）
	DowngradeFailed bool   // 应降级但更新失败
}

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

// AddOrgAdmin 任命某用户为某组织的管理员（B13：可选同步升级系统身份）
//
// 流程：
//   1. 校验组织存在、目标用户存在、role_type 合法且与组织类型匹配。
//   2. 权限校验（admin 任何组织 / region_admin 仅辖区学校 / 其它拒绝）。
//   3. 写入 organization_admins（幂等）。
//   4. 若任命的是 school_admin 且该校 admin_user_id 为空 → 回填主管理员单字段。
//   5. B13：syncRole=true 时按白名单规则同步升级 users.role（详见文件头注释）。
//
// 返回：(*OrgAdminAddResult, error)。error 非 nil 表示任命本身失败（未落库）；
//   error 为 nil 时任命必已成功，同步结果看 result 三标志。
func (s *OrganizationService) AddOrgAdmin(ctx context.Context, orgID string, targetUserID string, roleType string, syncRole bool, callerRole string, callerID string) (*OrgAdminAddResult, error) {
	if targetUserID == "" {
		return nil, ErrOrgAdminUserRequired
	}
	if !models.IsValidOrgAdminRoleType(roleType) {
		return nil, ErrOrgAdminRoleTypeInvalid
	}

	org, err := repository.GetOrganizationByID(ctx, orgID)
	if err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}

	// 类型匹配：region 组织只能任 region_admin，school 组织只能任 school_admin
	if !orgTypeMatchesAdminRole(org.Type, roleType) {
		return nil, ErrOrgAdminRoleTypeMismatch
	}

	// 目标用户必须存在（B13：保留完整对象，其当前 Role 供同步白名单判定与结果文案）
	targetUser, uErr := repository.FindUserByID(ctx, targetUserID)
	if uErr != nil {
		return nil, ErrOrgAdminTargetUserNF
	}

	// 权限校验
	if err := s.canManageOrgAdmins(ctx, org, callerRole, callerID); err != nil {
		return nil, err
	}

	// 写入多管理员表（幂等）
	if err := repository.AddOrgAdmin(ctx, orgID, targetUserID, roleType, callerID); err != nil {
		orgLog.Error("任命组织管理员失败", "org_id", orgID, "target", targetUserID, "role_type", roleType, "error", err)
		return nil, err
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

	// ==================== B13：可选同步升级系统身份 ====================
	result := &OrgAdminAddResult{TargetRole: targetUser.Role}
	if syncRole {
		// 期望身份由任命类型决定：区域任命→区域管理员，学校任命→学校管理员
		desiredRole := models.RoleSeniorOperator
		if roleType == models.OrgAdminRoleRegion {
			desiredRole = models.RoleRegionAdmin
		}

		switch targetUser.Role {
		case models.RoleOperator, models.RoleViewer:
			// 白名单内：执行升级。UpdateUserRole 只动 role 单列，不碰显示名等其它字段
			if sErr := repository.UpdateUserRole(ctx, targetUserID, desiredRole); sErr != nil {
				// 任命已成功，同步失败不回滚——标记 SyncFailed 供 handler 明示手工修复路径
				result.SyncFailed = true
				orgLog.Error("任命成功但身份同步失败", "org_id", orgID, "target", targetUserID,
					"from_role", targetUser.Role, "to_role", desiredRole, "error", sErr)
			} else {
				result.RoleSynced = true
				result.NewRole = desiredRole
				orgLog.Info("任命并同步身份成功", "org_id", orgID, "target", targetUserID,
					"from_role", targetUser.Role, "to_role", desiredRole, "by", callerID)
			}
		default:
			// 白名单外（admin/senior_operator/region_admin/district_inspector）一律不动：
			// 已具备管理身份者任命只授予管辖范围；senior 兼区域管辖走 buildUnionScope 并集。
			// 三标志保持零值，handler 据 TargetRole 拼"现有身份为X，未变更"文案。
			orgLog.Info("任命成功，目标身份不在升级白名单，未同步",
				"org_id", orgID, "target", targetUserID, "current_role", targetUser.Role)
		}
	}

	orgLog.Info("任命组织管理员成功", "org_id", orgID, "target", targetUserID, "role_type", roleType, "by", callerID)
	return result, nil
}

// RemoveOrgAdmin 移除某组织的某管理员（批C：末任命自动降级，闭合身份↔任命不变式）
//
// 流程：
//   1. 校验组织存在 + 权限。
//   2. 移除前先记下该组织当前主管理员是谁（用于判断被移除者是否就是主管理员）。
//   3. 从 organization_admins 移除。
//   4. 若移除的是 school 组织的主管理员 → 从剩余 school_admin 挑首个补位回填；无剩余则置空。
//   5. 批C：目标身份为任命制身份且管辖绑定归零 → 自动降级为骨干教师（best-effort，
//      失败不回滚移除，标记 DowngradeFailed）。计数在补位/置空之后执行，
//      保证被移除者的残留主字段指针不会撑住计数。
func (s *OrganizationService) RemoveOrgAdmin(ctx context.Context, orgID string, targetUserID string, callerRole string, callerID string) (*OrgAdminRemoveResult, error) {
	if targetUserID == "" {
		return nil, ErrOrgAdminUserRequired
	}

	org, err := repository.GetOrganizationByID(ctx, orgID)
	if err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}

	// 权限校验
	if err := s.canManageOrgAdmins(ctx, org, callerRole, callerID); err != nil {
		return nil, err
	}

	// 记下移除前的主管理员指针
	wasPrimaryAdmin := org.AdminUserID != nil && *org.AdminUserID == targetUserID

	// 从多管理员表移除
	if err := repository.RemoveOrgAdmin(ctx, orgID, targetUserID); err != nil {
		if errors.Is(err, repository.ErrMemberNotFound) {
			return nil, ErrMemberNotFound
		}
		orgLog.Error("移除组织管理员失败", "org_id", orgID, "target", targetUserID, "error", err)
		return nil, err
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

	// ==================== 批C：末任命自动降级（best-effort）====================
	result := &OrgAdminRemoveResult{}
	target, uErr := repository.FindUserByID(ctx, targetUserID)
	if uErr != nil {
		// 目标用户查不到（并发被删等）：移除已成功，跳过降级判定
		orgLog.Error("移除后查询目标用户失败(跳过降级判定)", "target", targetUserID, "error", uErr)
	} else if models.IsAppointmentOnlyRole(target.Role) {
		bindings, cErr := repository.CountUserAdminBindings(ctx, targetUserID)
		if cErr != nil {
			orgLog.Error("统计剩余管辖绑定失败(跳过自动降级)", "target", targetUserID, "error", cErr)
		} else if bindings == 0 {
			result.FromRole = target.Role
			if dErr := repository.UpdateUserRole(ctx, targetUserID, models.RoleOperator); dErr != nil {
				result.DowngradeFailed = true
				orgLog.Error("末任命移除后身份自动降级失败", "target", targetUserID,
					"from_role", target.Role, "error", dErr)
			} else {
				result.RoleDowngraded = true
				result.NewRole = models.RoleOperator
				orgLog.Info("末任命移除，身份自动降级为骨干教师", "target", targetUserID,
					"from_role", target.Role, "by", callerID)
			}
		} else {
			orgLog.Info("移除任命后目标仍持有其他管辖绑定，不降级",
				"target", targetUserID, "remaining_bindings", bindings)
		}
	}

	orgLog.Info("移除组织管理员成功", "org_id", orgID, "target", targetUserID, "by", callerID)
	return result, nil
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

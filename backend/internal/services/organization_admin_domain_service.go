package services

// organization_admin_domain_service.go
//
// 本文件实现区域管理员固定教育域的业务规则。
//
// 核心规则：
//   1. region_admin必须明确选择k12、vocational、adult之一；
//   2. 只能选择当前区域下实际存在的有效学校类型；
//   3. 同一用户的所有有效区域任命必须属于同一教育域；
//   4. 用户存在其它未配置教育域的区域任命时拒绝新增；
//   5. school_admin不单独保存教育域，直接继承学校；
//   6. 任命后的系统身份同步、学校主管理员回填继续沿用原有规则。
//
// 测试设计：
//   输入标准化和跨区域状态判断被拆成不访问数据库的纯函数，正式方法仍按
//   “权限→区域学校类型→其它任命状态→写入”的原顺序执行。拆分仅用于防回归测试，
//   不改变错误类型、校验顺序、数据库操作或现有授权边界。

import (
	"context"
	"errors"
	"strings"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// 区域管理员教育域任命错误。
var (
	ErrOrgAdminEducationDomainRequired = errors.New(
		"任命区域管理员时必须选择负责教育类型",
	)
	ErrOrgAdminEducationDomainInvalid = errors.New(
		"负责教育类型只能是k12、vocational或adult",
	)
	ErrOrgAdminEducationDomainUnavailable = errors.New(
		"所选教育类型在该区域下没有有效学校，不能任命",
	)
	ErrOrgAdminEducationDomainConflict = errors.New(
		"该用户已负责其它教育域，不能新增跨域任命",
	)
	ErrOrgAdminEducationDomainUnconfigured = errors.New(
		"该用户存在未配置教育域的区域管理员任命，请先修复原任命",
	)
)

// OrgAdminEducationDomainAddResult 区域固定教育域任命结果。
//
// OrgAdminAddResult继续承载原有身份同步结果；
// EducationDomain返回本次实际保存的标准化教育域。
// 学校管理员任命时EducationDomain为空字符串。
type OrgAdminEducationDomainAddResult struct {
	*OrgAdminAddResult
	EducationDomain string
}

// ListOrgAdminsWithEducationDomain 列出管理员并返回教育域。
//
// 权限规则与旧ListOrgAdmins完全一致：
//   - admin可查看任意组织；
//   - region_admin可查看自己管辖的区域及辖区学校；
//   - 其它角色拒绝。
func (s *OrganizationService) ListOrgAdminsWithEducationDomain(
	ctx context.Context,
	orgID string,
	callerRole string,
	callerID string,
) ([]*models.OrganizationAdminItem, error) {
	org, err := repository.GetOrganizationByID(ctx, orgID)
	if err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}

	if err := s.canManageOrgAdmins(
		ctx,
		org,
		callerRole,
		callerID,
	); err != nil {
		return nil, err
	}

	items, err := repository.ListOrgAdminsWithEducationDomain(
		ctx,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []*models.OrganizationAdminItem{}
	}

	return items, nil
}

// ListOrgAdminEducationDomains 返回任命面板允许选择的教育域。
//
// 区域组织：
//
//	只返回该区域递归子树下实际存在且有效的学校教育域。
//
// 学校组织：
//
//	返回空切片，因为学校管理员教育域直接继承学校。
//
// 不存在任何合法学校类型时返回空切片，绝不默认K12。
func (s *OrganizationService) ListOrgAdminEducationDomains(
	ctx context.Context,
	orgID string,
	callerRole string,
	callerID string,
) ([]string, error) {
	org, err := repository.GetOrganizationByID(ctx, orgID)
	if err != nil {
		if errors.Is(err, repository.ErrOrgNotFound) {
			return nil, ErrOrgNotFound
		}
		return nil, err
	}

	if err := s.canManageOrgAdmins(
		ctx,
		org,
		callerRole,
		callerID,
	); err != nil {
		return nil, err
	}

	if org.Type != models.OrgTypeRegion {
		return []string{}, nil
	}

	domains, err := repository.ListRegionSchoolEducationDomains(
		ctx,
		orgID,
	)
	if err != nil {
		return nil, err
	}
	if domains == nil {
		domains = []string{}
	}

	return domains, nil
}

// AddOrgAdminWithEducationDomain 严格任命组织管理员。
//
// 与旧AddOrgAdmin相比，本方法增加educationDomain参数并执行固定教育域校验。
func (s *OrganizationService) AddOrgAdminWithEducationDomain(
	ctx context.Context,
	orgID string,
	targetUserID string,
	roleType string,
	educationDomain string,
	syncRole bool,
	callerRole string,
	callerID string,
) (*OrgAdminEducationDomainAddResult, error) {
	if strings.TrimSpace(targetUserID) == "" {
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

	// 区域只能任命region_admin，学校只能任命school_admin。
	if !orgTypeMatchesAdminRole(org.Type, roleType) {
		return nil, ErrOrgAdminRoleTypeMismatch
	}

	// 目标用户必须真实存在。
	// 完整对象同时用于后续可选的系统身份同步。
	targetUser, userErr := repository.FindUserByID(
		ctx,
		targetUserID,
	)
	if userErr != nil {
		return nil, ErrOrgAdminTargetUserNF
	}

	// 权限校验必须发生在区域学校类型查询之前，
	// 防止无权调用者借错误差异探测组织数据。
	if err := s.canManageOrgAdmins(
		ctx,
		org,
		callerRole,
		callerID,
	); err != nil {
		return nil, err
	}

	normalizedDomain := ""

	if roleType == models.OrgAdminRoleRegion {
		normalizedDomain, err =
			normalizeRequiredRegionAdminEducationDomain(
				educationDomain,
			)
		if err != nil {
			return nil, err
		}

		// 只能选择该区域实际存在的学校教育类型。
		availableDomains, domainErr :=
			repository.ListRegionSchoolEducationDomains(
				ctx,
				orgID,
			)
		if domainErr != nil {
			return nil, domainErr
		}

		// 检查用户在其它区域的有效任命。
		//
		// 排除当前orgID，使管理员可以通过重新任命当前区域的方式，
		// 给该区域存量空值记录补齐教育域；其它区域仍严格检查。
		state, stateErr :=
			repository.GetUserRegionAdminEducationDomainState(
				ctx,
				targetUserID,
				orgID,
			)
		if stateErr != nil {
			return nil, stateErr
		}

		if err := validateRegionAdminEducationDomainAssignment(
			normalizedDomain,
			availableDomains,
			state,
		); err != nil {
			return nil, err
		}
	}

	// 显式写入教育域。
	// 同组织同用户已有记录时UPSERT，用于补齐存量区域任命。
	if err := repository.AddOrgAdminWithEducationDomain(
		ctx,
		orgID,
		targetUserID,
		roleType,
		normalizedDomain,
		callerID,
	); err != nil {
		orgLog.Error(
			"任命组织管理员并写入教育域失败",
			"org_id", orgID,
			"target", targetUserID,
			"role_type", roleType,
			"education_domain", normalizedDomain,
			"error", err,
		)
		return nil, err
	}

	// 学校管理员继续维护organizations.admin_user_id兼容字段。
	// 区域负责人不通过本方法写区域主管理员单字段。
	if roleType == models.OrgAdminRoleSchool {
		if org.AdminUserID == nil || *org.AdminUserID == "" {
			fillErr := repository.UpdateOrganizationAdminUserID(
				ctx,
				orgID,
				&targetUserID,
			)
			if fillErr != nil {
				// 多管理员表已经成功写入，是权威记录。
				// 兼容字段失败只记录日志，不回滚主操作。
				orgLog.Error(
					"回填学校主管理员单字段失败",
					"org_id", orgID,
					"target", targetUserID,
					"error", fillErr,
				)
			} else {
				orgLog.Info(
					"任命首个学校管理员并回填主字段",
					"org_id", orgID,
					"target", targetUserID,
				)
			}
		}
	}

	baseResult := &OrgAdminAddResult{
		TargetRole: targetUser.Role,
	}
	result := &OrgAdminEducationDomainAddResult{
		OrgAdminAddResult: baseResult,
		EducationDomain:   normalizedDomain,
	}

	// 保留原有“任命时可选同步账户身份”规则。
	if syncRole {
		desiredRole := models.RoleSeniorOperator
		if roleType == models.OrgAdminRoleRegion {
			desiredRole = models.RoleRegionAdmin
		}

		switch targetUser.Role {
		case models.RoleOperator, models.RoleViewer:
			syncErr := repository.UpdateUserRole(
				ctx,
				targetUserID,
				desiredRole,
			)
			if syncErr != nil {
				baseResult.SyncFailed = true
				orgLog.Error(
					"任命成功但身份同步失败",
					"org_id", orgID,
					"target", targetUserID,
					"from_role", targetUser.Role,
					"to_role", desiredRole,
					"error", syncErr,
				)
			} else {
				baseResult.RoleSynced = true
				baseResult.NewRole = desiredRole
				orgLog.Info(
					"任命并同步身份成功",
					"org_id", orgID,
					"target", targetUserID,
					"from_role", targetUser.Role,
					"to_role", desiredRole,
					"education_domain", normalizedDomain,
					"by", callerID,
				)
			}

		default:
			// admin、已有学校管理员、已有区域管理员和区域教研员
			// 均不自动改变账户身份，只增加本次组织管辖。
			orgLog.Info(
				"任命成功，目标身份不在升级白名单",
				"org_id", orgID,
				"target", targetUserID,
				"current_role", targetUser.Role,
			)
		}
	}

	orgLog.Info(
		"固定教育域任命成功",
		"org_id", orgID,
		"target", targetUserID,
		"role_type", roleType,
		"education_domain", normalizedDomain,
		"by", callerID,
	)

	return result, nil
}

// normalizeRequiredRegionAdminEducationDomain 校验并标准化区域管理员教育域。
//
// 本函数明确区分“未填写”和“填写了非法值”两类错误，不允许调用
// NormalizeEducationDomain，因为后者会把非法值回退为K12，不适合授权判断。
func normalizeRequiredRegionAdminEducationDomain(
	educationDomain string,
) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(educationDomain))

	if normalized == "" {
		return "", ErrOrgAdminEducationDomainRequired
	}
	if !models.IsTeachingEducationDomain(normalized) {
		return "", ErrOrgAdminEducationDomainInvalid
	}

	return normalized, nil
}

// validateRegionAdminEducationDomainAssignment 校验区域和用户已有任命状态。
//
// 入参均为Repository查询结果的只读快照，因此本函数不访问数据库，便于对
// 三个合法教育域、同域多区域、跨域冲突、历史空值和区域无对应学校类型进行
// 确定性防回归测试。
//
// fail-closed规则：
//   - state为nil时按未配置异常处理；
//   - state.Domains中出现非法值时按未配置异常处理；
//   - 任何已有合法域与目标域不同均按跨域冲突处理。
func validateRegionAdminEducationDomainAssignment(
	educationDomain string,
	availableDomains []string,
	state *repository.UserRegionAdminEducationDomainState,
) error {
	normalizedTarget, err :=
		normalizeRequiredRegionAdminEducationDomain(
			educationDomain,
		)
	if err != nil {
		return err
	}

	if !containsRegionAdminEducationDomain(
		availableDomains,
		normalizedTarget,
	) {
		return ErrOrgAdminEducationDomainUnavailable
	}

	if state == nil || state.HasUnconfigured {
		return ErrOrgAdminEducationDomainUnconfigured
	}

	for _, existingDomain := range state.Domains {
		normalizedExisting := strings.ToLower(
			strings.TrimSpace(existingDomain),
		)

		if !models.IsTeachingEducationDomain(normalizedExisting) {
			return ErrOrgAdminEducationDomainUnconfigured
		}
		if normalizedExisting != normalizedTarget {
			return ErrOrgAdminEducationDomainConflict
		}
	}

	return nil
}

// containsRegionAdminEducationDomain 判断目标教育域是否存在于区域实际学校类型中。
//
// 权限判断不得调用NormalizeEducationDomain，避免非法值静默回退K12。
func containsRegionAdminEducationDomain(
	domains []string,
	target string,
) bool {
	target = strings.ToLower(strings.TrimSpace(target))

	for _, domain := range domains {
		if strings.ToLower(strings.TrimSpace(domain)) == target {
			return true
		}
	}

	return false
}

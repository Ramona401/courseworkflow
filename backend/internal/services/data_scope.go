package services

// data_scope.go — 全系统唯一的数据范围解析器
//
// 设计目标：
//   建立“用户能看到哪些数据行”的唯一权威来源。所有列表查询通过本解析器获得
//   组织、学校和用户白名单，再据白名单过滤，杜绝各 Handler 分散判断造成的遗漏。
//
// fail-closed 三态白名单语义：
//   - nil          → 不过滤，仅 admin 可以使用；
//   - 非 nil 空切片 → 匹配空集；
//   - 非空切片      → 只允许白名单中的记录。
//
// 上下文 5：区域管理员学校范围同域过滤
//
//   region_admin 的学校范围现在严格为：
//     管辖区域树下学校
//     AND 学校 status='active'
//     AND 学校 education_domain=管理员固定教育域
//
//   固定教育域通过 ResolveRegionAdminEducationScope 严格解析：
//     - 只允许 k12、vocational、adult；
//     - 多个区域任命必须全部同域；
//     - 空值、mixed、common、非法值和数据库错误全部 fail-closed；
//     - 不调用 NormalizeEducationDomain，不回退 K12。
//
//   区域管理员兼任的本校不再被无条件并入。只有本校本身位于管辖区域树下、
//   状态为 active 且教育域一致时，才会由同域递归查询自然进入白名单。
//
// senior_operator 的历史双重身份行为本上下文保持不变：
//   本校是主范围；若兼任区域管理员，区域范围仍作为 best-effort 加料。
//   本上下文只收口正式 role=region_admin 的学校范围，不顺带修改其它角色业务规则。

import (
	"context"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// dataScopeLog 模块级结构化日志器。
var dataScopeLog = logger.WithModule("services.data_scope")

// DataScope 描述当前请求者对业务数据的可见范围。
//
// 白名单字段三态语义：
//   - nil          → 不过滤，仅 admin；
//   - 非 nil 空切片 → 匹配空集；
//   - 非空切片      → 只允许名单内记录。
//
// 各维度用途：
//   - OrgIDs：区域和学校组织行；
//   - SchoolIDs：学校范围；
//   - UserIDs：学校成员、作者和个人数据范围。
type DataScope struct {
	Role          string   // 请求者角色
	IsAdmin       bool     // 是否系统管理员
	OrgIDs        []string // 可见组织 ID
	SchoolIDs     []string // 可见学校 ID
	UserIDs       []string // 可见用户 ID
	Blocked       bool     // 是否因异常被收窄为空集
	BlockedReason string   // 收窄原因
}

// newBlockedScope 构造统一的 fail-closed 空范围。
func newBlockedScope(role string, reason string) DataScope {
	return DataScope{
		Role:          role,
		IsAdmin:       false,
		OrgIDs:        []string{},
		SchoolIDs:     []string{},
		UserIDs:       []string{},
		Blocked:       true,
		BlockedReason: reason,
	}
}

// buildScopeFromResolvedSchools 使用已经完成授权过滤的区域和学校 ID 构造 DataScope。
//
// 本函数不会自行扩大学校范围：
//   - regionIDs 只用于组织列表显示区域节点；
//   - schoolIDs 必须由调用方事先完成权限过滤；
//   - 用户白名单只从这些 schoolIDs 的 school_members 汇总。
//
// 学校列表为空是合法结果：
//   - OrgIDs 仍包含管理员管辖区域；
//   - SchoolIDs 和 UserIDs 返回非 nil 空切片；
//   - Blocked=false，表示“配置正确但当前域没有学校”，而不是系统异常。
//
// 任一学校成员查询失败则收窄为 Blocked 空集，不能返回部分成员范围。
func buildScopeFromResolvedSchools(
	ctx context.Context,
	role string,
	regionIDs []string,
	schoolIDs []string,
) DataScope {
	normalizedRegionIDs :=
		normalizeRegionEducationScopeIDs(regionIDs)
	normalizedSchoolIDs :=
		normalizeRegionEducationScopeIDs(schoolIDs)

	userIDSet := make(map[string]struct{})
	for _, schoolID := range normalizedSchoolIDs {
		memberIDs, err := repository.ListSchoolMemberIDs(
			ctx,
			schoolID,
		)
		if err != nil {
			dataScopeLog.Warn(
				"查询学校成员失败，数据范围收窄为空集",
				"role",
				role,
				"school_id",
				schoolID,
				"error",
				err,
			)
			return newBlockedScope(
				role,
				"查询学校成员失败",
			)
		}

		for _, userID := range memberIDs {
			if userID != "" {
				userIDSet[userID] = struct{}{}
			}
		}
	}

	userIDs := make([]string, 0, len(userIDSet))
	for userID := range userIDSet {
		userIDs = append(userIDs, userID)
	}
	userIDs = normalizeRegionEducationScopeIDs(userIDs)

	orgIDs := make(
		[]string,
		0,
		len(normalizedRegionIDs)+len(normalizedSchoolIDs),
	)
	orgIDs = append(orgIDs, normalizedRegionIDs...)
	orgIDs = append(orgIDs, normalizedSchoolIDs...)

	return DataScope{
		Role:      role,
		IsAdmin:   false,
		OrgIDs:    orgIDs,
		SchoolIDs: normalizedSchoolIDs,
		UserIDs:   userIDs,
	}
}

// resolveRegionIDsBestEffort 查询学校管理员兼任管理的区域。
//
// 本函数只供 senior_operator 历史“双重身份并集”使用。
// 查询失败时跳过区域加料，范围只会更小，不会放大。
func resolveRegionIDsBestEffort(
	ctx context.Context,
	userID string,
) []string {
	regionIDs, err := repository.ListRegionIDsByAdmin(
		ctx,
		userID,
	)
	if err != nil {
		dataScopeLog.Warn(
			"双重身份加料：查询管辖区域失败",
			"user_id",
			userID,
			"error",
			err,
		)
		return []string{}
	}
	return regionIDs
}

// buildUnionScope 构造 senior_operator 的历史双重身份范围。
//
// 输入：
//   - regionIDs：兼任管理的区域；
//   - ownSchoolID：学校管理员主身份对应的本校。
//
// 流程：
//  1. 递归取得区域树下全部 active 学校；
//  2. 并入本校；
//  3. 调用 buildScopeFromResolvedSchools 汇总成员。
//
// 本函数不用于正式 role=region_admin。
// role=region_admin 已改用 ResolveRegionAdminEducationScope 执行同域过滤。
func buildUnionScope(
	ctx context.Context,
	role string,
	regionIDs []string,
	ownSchoolID string,
) DataScope {
	schoolIDSet := make(map[string]struct{})

	for _, regionID := range regionIDs {
		schoolIDs, err := repository.ListDescendantSchoolIDs(
			ctx,
			regionID,
		)
		if err != nil {
			dataScopeLog.Warn(
				"递归查询区域树下学校失败，收窄为空集",
				"region_id",
				regionID,
				"error",
				err,
			)
			return newBlockedScope(
				role,
				"查询区域树下学校失败",
			)
		}

		for _, schoolID := range schoolIDs {
			if schoolID != "" {
				schoolIDSet[schoolID] = struct{}{}
			}
		}
	}

	if ownSchoolID != "" {
		schoolIDSet[ownSchoolID] = struct{}{}
	}

	schoolIDs := make([]string, 0, len(schoolIDSet))
	for schoolID := range schoolIDSet {
		schoolIDs = append(schoolIDs, schoolID)
	}

	return buildScopeFromResolvedSchools(
		ctx,
		role,
		regionIDs,
		schoolIDs,
	)
}

// ResolveDataScope 根据角色与用户 ID 解析数据可见范围。
//
// 角色规则：
//   - admin：全系统范围，白名单为 nil；
//   - region_admin：管辖区域树下同域 active 学校及其成员；
//   - senior_operator：本校及历史兼任区域范围；
//   - operator/viewer：仅本人；
//   - 空身份或未知身份：fail-closed 空集。
func ResolveDataScope(
	ctx context.Context,
	role string,
	userID string,
) DataScope {
	if role == "" || userID == "" {
		return newBlockedScope(role, "未认证")
	}

	switch role {
	case models.RoleAdmin:
		return DataScope{
			Role:      role,
			IsAdmin:   true,
			OrgIDs:    nil,
			SchoolIDs: nil,
			UserIDs:   nil,
		}

	case models.RoleRegionAdmin:
		// 上下文 5 的唯一正式入口。
		//
		// 返回范围已经满足：
		//   管辖区域 AND active 学校 AND 固定教育域。
		resolvedScope, err :=
			ResolveRegionAdminEducationScope(
				ctx,
				userID,
			)
		if err != nil {
			dataScopeLog.Warn(
				"解析区域管理员同域学校范围失败，收窄为空集",
				"user_id",
				userID,
				"error",
				err,
			)
			return newBlockedScope(
				role,
				"教育域或管辖范围尚未正确配置",
			)
		}

		return buildScopeFromResolvedSchools(
			ctx,
			role,
			resolvedScope.RegionIDs,
			resolvedScope.SchoolIDs,
		)

	case models.RoleSeniorOperator:
		// senior_operator 的主来源仍为本校。
		// 本上下文不改变其历史业务行为。
		school, err := repository.GetSchoolByAdminUserID(
			ctx,
			userID,
		)
		if err != nil ||
			school == nil ||
			school.ID == "" {
			return newBlockedScope(
				role,
				"您尚未绑定学校，请联系系统管理员",
			)
		}

		regionIDs := resolveRegionIDsBestEffort(
			ctx,
			userID,
		)
		return buildUnionScope(
			ctx,
			role,
			regionIDs,
			school.ID,
		)

	default:
		// 普通教学角色只取得自己的用户范围。
		//
		// SchoolIDs 不在此处推导，本校共享等其它业务规则由具体查询层处理。
		return DataScope{
			Role:      role,
			IsAdmin:   false,
			OrgIDs:    []string{},
			SchoolIDs: []string{},
			UserIDs:   []string{userID},
		}
	}
}

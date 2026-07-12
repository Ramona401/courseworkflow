package services

// data_scope.go — 全系统唯一的数据范围解析器（迭代一·组织与权限重铸）
//
// 设计目标（对应规划"核心洞察"）：
//   建立"用户能看到哪些数据行"的唯一权威来源。所有列表查询（教案/审核/积分等）
//   都过这一个解析器拿到白名单，再据白名单过滤，杜绝散落各 handler 的 scope 判断遗漏。
//
// 设计完全对齐既有 ResolveTokenScope（token_service.go）的成熟模式：
//   - 只收 (ctx, role, userID)，不依赖 middleware，避免循环依赖。
//   - fail-closed 三态白名单语义（与 repository 层一致）：
//       nil        → 不过滤（看全部，仅 admin）
//       非nil空切片 → 匹配空集（看不到任何数据；如 senior 未绑校、孤儿账号）
//       非空        → 仅匹配名单内
//   - 任何不确定（查询失败/未绑校）一律收窄为空集 + Blocked，绝不放大为全量。
//
// 与 ResolveTokenScope 的关系（Phase 4 收口）：
//   Phase 4 会让 ResolveTokenScope 改为委托 ResolveDataScope，使两套权限统一。
//   本文件的 DataScope 字段命名（IsAdmin/UserIDs/Blocked/BlockedReason）刻意对齐
//   TokenScope，减少 Phase 4 委托时的转换摩擦。
//
// ★ B2 修复（账户与权限·第一批）★
//   1. region_admin 管辖来源升级：repository.ListRegionIDsByAdmin 已改为双来源并集
//      （organization_admins ∪ organizations.admin_user_id），"编辑区域"弹窗单字段任命的
//      区域管理员不再落空为 Blocked → 组织架构能正常显示辖区与辖下学校。
//   2. 双重身份（同时是区域管理员和学校管理员）显示优先级定为【并集】：
//        - users.role=region_admin    ：区域管辖 ∪ 本校（GetSchoolByAdminUserID 加料）
//        - users.role=senior_operator ：本校 ∪ 区域管辖（ListRegionIDsByAdmin 加料）
//      两分支共用 buildUnionScope 构造。fail-closed 纪律不变：
//        - 主来源（region 分支的区域 / senior 分支的本校）查询失败或落空 → Blocked（与历史一致）；
//        - 加料来源查询失败 → 仅 Warn 跳过（并集少一块，范围只会更小不会放大，方向仍是收窄）。
//
// 阶段状态：
//   admin / region_admin / senior_operator / operator / viewer 五分支均已完整实现。

import (
	"context"
	"errors"

	"tedna/internal/logger"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// dataScopeLog 模块级结构化日志器
var dataScopeLog = logger.WithModule("services.data_scope")

// DataScope 描述当前请求者对业务数据的可见范围（教案/审核/积分等列表查询的唯一白名单来源）
//
// 白名单字段三态语义（与 repository 层、TokenScope 完全一致）：
//   - nil        → 不过滤（看全部，仅 admin）
//   - 非nil空切片 → 匹配空集（fail-closed，看不到任何数据）
//   - 非空        → 仅匹配名单内
//
// 各白名单维度用途：
//   - SchoolIDs：按学校过滤（如教案 school_id、学校账户）
//   - UserIDs ：按用户过滤（如教案 author_id、消费流水 user_id、个人账户 owner）
//   - OrgIDs  ：按组织过滤（区域+学校组织行本身；区域级资源用）
type DataScope struct {
	Role          string   // 请求者角色（调试/日志用）
	IsAdmin       bool     // 是否系统管理员（看全部，各白名单为 nil）
	OrgIDs        []string // 可见的组织ID白名单（区域+学校组织行本身）
	SchoolIDs     []string // 可见的学校ID白名单
	UserIDs       []string // 可见的用户ID白名单
	Blocked       bool     // 是否被收窄为空集（孤儿账号/未绑校/查询失败）
	BlockedReason string   // 收窄原因（供上层提示）
}

// newBlockedScope 构造一个"收窄为空集"的 DataScope（fail-closed 统一出口）
//
// 所有白名单均为"非nil空切片"，对应 repository 层"匹配空集"语义——看不到任何数据。
func newBlockedScope(role, reason string) DataScope {
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

// resolveOwnSchoolIDBestEffort 加料方向查询"本人作为学校管理员管理的本校"（best-effort）
//
// 供 region_admin 分支做双重身份并集使用：查到返回学校ID；查不到/查询失败返回空串。
//   - ErrOrgNotFound = 不是任何学校的管理员（正常情况，不打日志）
//   - 其它 DB 错误  = 仅 Warn 跳过（加料失败范围只会更小，不违背 fail-closed）
//
// GetSchoolByAdminUserID 本身已是 B1 后的两级查找（organizations.admin_user_id 单字段
// + organization_admins 多管理员表兜底），第二学校管理员在此也能命中。
func resolveOwnSchoolIDBestEffort(ctx context.Context, userID string) string {
	school, err := repository.GetSchoolByAdminUserID(ctx, userID)
	if err != nil {
		if !errors.Is(err, repository.ErrOrgNotFound) {
			dataScopeLog.Warn("双重身份加料：查询本校失败（跳过本校并集）", "user", userID, "error", err)
		}
		return ""
	}
	if school == nil || school.ID == "" {
		return ""
	}
	return school.ID
}

// resolveRegionIDsBestEffort 加料方向查询"本人管辖的区域"（best-effort）
//
// 供 senior_operator 分支做双重身份并集使用：查到返回区域ID列表；查询失败返回空切片。
//   - 查询失败仅 Warn 跳过（加料失败范围只会更小，不违背 fail-closed）
//   - 返回空切片 = 不是任何区域的管理员（正常情况，senior 只见本校，行为与历史完全一致）
func resolveRegionIDsBestEffort(ctx context.Context, userID string) []string {
	regionIDs, err := repository.ListRegionIDsByAdmin(ctx, userID)
	if err != nil {
		dataScopeLog.Warn("双重身份加料：查询管辖区域失败（跳过区域并集）", "user", userID, "error", err)
		return []string{}
	}
	return regionIDs
}

// buildUnionScope 按"管辖并集"构造管理类角色（region_admin / senior_operator）的 DataScope
//
// 输入（两者可任一为空，但不可同时为空——调用方保证主来源非空才进入本函数）：
//   - regionIDs   ：管辖的区域组织ID集合（并集的区域部分，可为空）
//   - ownSchoolID ：本人作为学校管理员管理的本校ID（并集的本校部分，可为空串）
//
// 构造流程（严格 fail-closed）：
//  1. 逐区域 ListDescendantSchoolIDs 递归取辖下学校 → 任一步失败即 Blocked（主链路不容错）；
//  2. 本校ID（非空且未被辖区覆盖时）并入学校集合；
//  3. 逐学校 ListSchoolMemberIDs 汇总成员 user_id 去重 → 任一步失败即 Blocked；
//  4. OrgIDs = 区域ID + 全部学校ID；SchoolIDs = 全部学校ID；UserIDs = 全部成员。
//
// 说明：本函数把原 region_admin 分支的第 2~5 步抽出复用，SQL 行为与失败语义逐字保持，
// 仅新增"本校ID并入"一个可选输入，供双重身份并集使用。
func buildUnionScope(ctx context.Context, role string, regionIDs []string, ownSchoolID string) DataScope {
	// 1. 学校集合去重容器（辖区学校 ∪ 本校）
	schoolIDSet := make(map[string]struct{})
	for _, regionID := range regionIDs {
		schoolIDs, sErr := repository.ListDescendantSchoolIDs(ctx, regionID)
		if sErr != nil {
			dataScopeLog.Warn("递归查询区域树下学校失败，收窄为空集", "region", regionID, "error", sErr)
			return newBlockedScope(role, "查询区域树下学校失败")
		}
		for _, sid := range schoolIDs {
			schoolIDSet[sid] = struct{}{}
		}
	}
	// 2. 本校并入（双重身份：区域管辖 ∪ 本校；本校已在辖区内时 map 天然去重）
	if ownSchoolID != "" {
		schoolIDSet[ownSchoolID] = struct{}{}
	}

	// 3. 汇总所有学校的成员 user_id（去重）
	userIDSet := make(map[string]struct{})
	schoolIDs := make([]string, 0, len(schoolIDSet))
	for sid := range schoolIDSet {
		schoolIDs = append(schoolIDs, sid)
		memberIDs, mErr := repository.ListSchoolMemberIDs(ctx, sid)
		if mErr != nil {
			dataScopeLog.Warn("查询学校成员失败，收窄为空集", "school", sid, "error", mErr)
			return newBlockedScope(role, "查询学校成员失败")
		}
		for _, uid := range memberIDs {
			userIDSet[uid] = struct{}{}
		}
	}

	// 4. 组织白名单 = 区域ID + 学校ID（区域级资源 + 学校级资源都可见）
	orgIDs := make([]string, 0, len(regionIDs)+len(schoolIDs))
	orgIDs = append(orgIDs, regionIDs...)
	orgIDs = append(orgIDs, schoolIDs...)

	// 5. 用户白名单 = 所有学校成员
	userIDs := make([]string, 0, len(userIDSet))
	for uid := range userIDSet {
		userIDs = append(userIDs, uid)
	}

	return DataScope{
		Role:      role,
		IsAdmin:   false,
		OrgIDs:    orgIDs,
		SchoolIDs: schoolIDs,
		UserIDs:   userIDs,
	}
}

// ResolveDataScope 根据角色与用户ID解析数据可见范围（全系统唯一数据范围解析点，fail-closed）
//
// 各角色范围：
//   - admin            → IsAdmin=true，所有白名单 nil（不过滤）
//   - region_admin     → 管辖区域树下所有学校 ∪ 本人本校（双重身份并集）+ 这些学校成员
//   - senior_operator  → 本校 ∪ 本人管辖区域树下学校（双重身份并集）+ 这些学校成员
//                        （未绑校/查询失败 → Blocked 空集，与历史一致）
//   - operator/viewer  → 仅本人（SchoolIDs 不限定，由共享可见逻辑在查询层处理）
//   - 其它/空           → Blocked 空集
func ResolveDataScope(ctx context.Context, role string, userID string) DataScope {
	// 入参不全 → 直接收窄（未认证）
	if role == "" || userID == "" {
		return newBlockedScope(role, "未认证")
	}

	switch role {

	// ---------- admin：全系统，不过滤 ----------
	case models.RoleAdmin:
		return DataScope{
			Role:      role,
			IsAdmin:   true,
			OrgIDs:    nil,
			SchoolIDs: nil,
			UserIDs:   nil,
		}

	// ---------- region_admin：区域管理员，管辖区域树 ∪ 本校（双重身份并集）----------
	case models.RoleRegionAdmin:
		// 主来源：查该用户管辖的所有区域
		// （B2 后 ListRegionIDsByAdmin 为双来源并集：organization_admins ∪ organizations.admin_user_id）
		regionIDs, rErr := repository.ListRegionIDsByAdmin(ctx, userID)
		if rErr != nil {
			dataScopeLog.Warn("查询区域管理员管辖区域失败，收窄为空集", "user", userID, "error", rErr)
			return newBlockedScope(role, "查询管辖区域失败")
		}
		if len(regionIDs) == 0 {
			// 不是任何区域的管理员 → 收窄空集（fail-closed，与历史一致）
			return newBlockedScope(role, "您尚未被任命为任何区域的管理员")
		}
		// 加料来源（best-effort）：本人若同时是某学校管理员，并入本校
		ownSchoolID := resolveOwnSchoolIDBestEffort(ctx, userID)
		return buildUnionScope(ctx, role, regionIDs, ownSchoolID)

	// ---------- senior_operator：学校管理员，本校 ∪ 管辖区域树（双重身份并集）----------
	case models.RoleSeniorOperator:
		// 主来源：本校（未绑校 → Blocked，与历史一致）
		school, err := repository.GetSchoolByAdminUserID(ctx, userID)
		if err != nil || school == nil || school.ID == "" {
			return newBlockedScope(role, "您尚未绑定学校，请联系系统管理员")
		}
		// 加料来源（best-effort）：本人若同时被任命为区域管理员，并入辖区
		regionIDs := resolveRegionIDsBestEffort(ctx, userID)
		return buildUnionScope(ctx, role, regionIDs, school.ID)

	// ---------- operator / viewer：普通/骨干教师，仅本人 ----------
	default:
		// 仅本人可见。SchoolIDs 不在此限定（教案"本校共享可见"等逻辑由具体查询层处理），
		// 这里只给出 UserIDs=[自己]，与 ResolveTokenScope 的 operator/viewer 分支一致。
		return DataScope{
			Role:      role,
			IsAdmin:   false,
			OrgIDs:    []string{},
			SchoolIDs: []string{},
			UserIDs:   []string{userID},
		}
	}
}

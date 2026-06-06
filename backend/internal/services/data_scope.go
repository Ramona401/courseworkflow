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
// 阶段状态（迭代一）：
//   admin / region_admin / senior_operator / operator / viewer 五分支均已完整实现。
//   region_admin 分支依赖 Phase 2 新建的 ListRegionIDsByAdmin + ListDescendantSchoolIDs
//   （以 organization_admins 表为管辖权威），已于 Phase 2 完成后接入。

import (
	"context"

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

// ResolveDataScope 根据角色与用户ID解析数据可见范围（全系统唯一数据范围解析点，fail-closed）
//
// 各角色范围：
//   - admin            → IsAdmin=true，所有白名单 nil（不过滤）
//   - region_admin     → 本区域树下所有学校 + 这些学校成员（以 organization_admins 为管辖权威）
//   - senior_operator  → 本校 + 本校成员（未绑校/查询失败 → Blocked 空集）
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

	// ---------- region_admin：区域管理员，本区域树下所有学校 ----------
	case models.RoleRegionAdmin:
		// 1. 查该用户管辖的所有区域（以 organization_admins.role_type='region_admin' 为权威）
		regionIDs, rErr := repository.ListRegionIDsByAdmin(ctx, userID)
		if rErr != nil {
			dataScopeLog.Warn("查询区域管理员管辖区域失败，收窄为空集", "user", userID, "error", rErr)
			return newBlockedScope(role, "查询管辖区域失败")
		}
		if len(regionIDs) == 0 {
			// 不是任何区域的管理员 → 收窄空集（fail-closed）
			return newBlockedScope(role, "您尚未被任命为任何区域的管理员")
		}

		// 2. 对每个区域递归查出树下所有学校，汇总去重
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

		// 3. 汇总所有学校的成员 user_id（去重）
		userIDSet := make(map[string]struct{})
		schoolIDs := make([]string, 0, len(schoolIDSet))
		for sid := range schoolIDSet {
			schoolIDs = append(schoolIDs, sid)
			memberIDs, mErr := repository.ListSchoolMemberIDs(ctx, sid)
			if mErr != nil {
				dataScopeLog.Warn("查询学校成员失败，收窄为空集", "school", sid, "error", mErr)
				return newBlockedScope(role, "查询区域树下学校成员失败")
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

	// ---------- senior_operator：学校管理员，仅本校 ----------
	case models.RoleSeniorOperator:
		school, err := repository.GetSchoolByAdminUserID(ctx, userID)
		if err != nil || school == nil || school.ID == "" {
			// 未绑定学校 → 收窄空集 + 提示（与 ResolveTokenScope 行为一致）
			return newBlockedScope(role, "您尚未绑定学校，请联系系统管理员")
		}
		memberIDs, mErr := repository.ListSchoolMemberIDs(ctx, school.ID)
		if mErr != nil {
			dataScopeLog.Warn("查询本校成员失败，收窄为空集", "school", school.ID, "error", mErr)
			return newBlockedScope(role, "查询本校成员失败")
		}
		// 学校ID白名单 = 本校；用户ID白名单 = 本校全体成员
		userIDs := make([]string, 0, len(memberIDs))
		userIDs = append(userIDs, memberIDs...)
		return DataScope{
			Role:      role,
			IsAdmin:   false,
			OrgIDs:    []string{school.ID},
			SchoolIDs: []string{school.ID},
			UserIDs:   userIDs,
		}

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

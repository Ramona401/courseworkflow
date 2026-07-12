package services

// token_scope.go — Token积分系统数据权限范围解析（TokenScope + ResolveTokenScope）
//
// 从 token_service.go 拆出（守 600 行红线），并新增「双重身份并集」能力。
//
// 双重身份并集 batch（修复 lubin/lichao 类账号的区域积分盲区）：
//   背景：任命制身份下，一个 users.role=senior_operator 的学校管理员可以同时被任命为
//   某区域的 region_admin（organization_admins.role_type='region_admin'）。任命服务
//   刻意不改其 users.role（避免破坏其学校管理员身份），设计意图是由数据范围解析层
//   做「本校 ∪ 辖区」并集（data_scope.go 的 buildUnionScope 已如此实现，用户管理/
//   组织架构/课件审核等链路均已验证）。但积分系统的 ResolveTokenScope 此前只按
//   users.role 单一分支解析：
//     - senior_operator 分支只解析本校 → 兼任区域管理员的校管在积分中心看不到辖区
//       其他学校账户、没有区域分配入口（AllowedRegionOwnerIDs 恒空）；
//     - region_admin 分支只解析辖区 → 反向兼任校管的区域管理员管不了本校个人账户。
//   本次把 data_scope 的并集哲学镜像到积分系统：
//     - senior_operator：主链路=本校（失败→Blocked，与旧行为逐字一致）；
//       加料=区域管辖（best-effort：ListRegionIDsByAdmin 查任命，命中则并入
//       辖区学校 owner_id + 辖区成员 user_id + AllowedRegionOwnerIDs=辖区区域ID；
//       加料任一步失败仅 Warn 跳过——并集只会更小，不违 fail-closed）。
//     - region_admin：主链路=辖区（失败→Blocked，与旧行为逐字一致）；
//       加料=本校（best-effort：GetSchoolByAdminUserID 查其是否兼任校管，命中则
//       按 senior 语义并入本校成员个人账户，使其能做本校 school→personal 分配）。
//   纯身份账号（只当校管 / 只当区域管理员）加料查询天然落空，行为与改造前完全一致。
//
// 白名单三态语义（与 repository 层完全一致）：
//   - nil        → 不过滤（看全部，仅 admin）
//   - 非nil空切片 → 匹配空集（fail-closed，看不到任何数据）
//   - 非空        → 仅匹配名单内
//
// 查询维度复用白名单：
//   - OwnerIDs：账户列表 / 概览统计 / 分配记录 / 采购记录（按 token_accounts.owner_id）
//   - UserIDs ：消费流水（按 token_consumption_logs.user_id）
//   - AllowedRegionOwnerIDs：可作分配来源的区域账户 owner_id（不参与任何列表/统计SQL，
//     专供 token_handler.tokenSourceAllowed 与 my-region-accounts 入口，
//     完整保留 v172.1 "非admin列表排除region账户"的跨级泄漏防线）

import (
	"context"

	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== TokenScope 结构体 ====================

// TokenScope 描述当前请求者对积分数据的可见范围（积分系统唯一权限决策点）
type TokenScope struct {
	Role                  string   // 请求者角色
	IsAdmin               bool     // 是否系统管理员（看全部）
	OwnerIDs              []string // 账户/概览/分配/采购 owner_id 白名单
	UserIDs               []string // 消费流水 user_id 白名单
	AllowedRegionOwnerIDs []string // 可作分配来源的区域账户 owner_id 白名单（有区域管辖任命者非空）
	Blocked               bool     // 是否被收窄为空集（如 senior 未绑定学校）
	BlockedReason         string   // 收窄原因（供上层提示）
}

// ==================== 解析入口 ====================

// ResolveTokenScope 根据角色与用户ID解析数据可见范围（fail-closed，任何不确定收窄为空集）
func (s *TokenService) ResolveTokenScope(ctx context.Context, role string, userID string) *TokenScope {
	// 未认证：直接空集
	if role == "" || userID == "" {
		return tokenBlockedScope(role, "未认证")
	}

	switch role {
	case models.RoleAdmin:
		// admin 看全部：三白名单皆 nil（不过滤）
		return &TokenScope{Role: role, IsAdmin: true, OwnerIDs: nil, UserIDs: nil}

	case models.RoleRegionAdmin:
		return s.resolveTokenRegionAdminScope(ctx, userID)

	case models.RoleSeniorOperator:
		return s.resolveTokenSeniorScope(ctx, userID)

	default:
		// operator/viewer：仅本人（账户 owner=自己 / 消费 user=自己）
		return &TokenScope{
			Role:     role,
			IsAdmin:  false,
			OwnerIDs: []string{userID},
			UserIDs:  []string{userID},
		}
	}
}

// ==================== region_admin 分支（主链路=辖区 + 加料=兼任本校）====================

// resolveTokenRegionAdminScope 解析 users.role=region_admin 的可见范围
func (s *TokenService) resolveTokenRegionAdminScope(ctx context.Context, userID string) *TokenScope {
	role := models.RoleRegionAdmin

	// ---- 主链路①：查管辖区域（organization_admins ∪ organizations.admin_user_id 双来源权威）----
	regionIDs, rErr := repository.ListRegionIDsByAdmin(ctx, userID)
	if rErr != nil {
		tokenLog.Warn("查询区域管理员管辖区域失败，收窄为空集", "user", userID, "error", rErr)
		return tokenBlockedScope(role, "查询管辖区域失败")
	}
	if len(regionIDs) == 0 {
		// 不是任何区域的管理员 → 收窄空集（fail-closed）
		return tokenBlockedScope(role, "您尚未被任命为任何区域的管理员")
	}

	// ---- 主链路②：逐区域递归树下学校（任一失败→Blocked，与 v172 旧行为一致）----
	schoolIDSet := make(map[string]struct{})
	for _, regionID := range regionIDs {
		schoolIDs, sErr := repository.ListDescendantSchoolIDs(ctx, regionID)
		if sErr != nil {
			tokenLog.Warn("递归查询区域树下学校失败，收窄为空集", "region", regionID, "error", sErr)
			return tokenBlockedScope(role, "查询区域树下学校失败")
		}
		for _, sid := range schoolIDs {
			schoolIDSet[sid] = struct{}{}
		}
	}

	// ---- 消费明细：辖区学校成员 user_id（单校失败仅跳过该校，不整体瘫痪）----
	memberIDSet := make(map[string]struct{})
	tokenCollectSchoolMembersBestEffort(ctx, schoolIDSet, memberIDSet)

	// ---- 加料（best-effort）：兼任校管则按 senior 语义并入本校 ----
	// GetSchoolByAdminUserID 未命中（不是任何学校的管理员）是常态，静默跳过不告警。
	// 命中则：本校ID + 本校成员个人账户并入 OwnerIDs（使其能管理本校个人账户、
	// 做 school→personal 分配），本校成员并入 UserIDs（消费明细）。
	ownMemberSet := make(map[string]struct{})
	ownSchoolID := ""
	if school, err := repository.GetSchoolByAdminUserID(ctx, userID); err == nil && school != nil && school.ID != "" {
		ownSchoolID = school.ID
		if members, mErr := repository.ListSchoolMemberIDs(ctx, school.ID); mErr == nil {
			for _, uid := range members {
				ownMemberSet[uid] = struct{}{}
			}
		} else {
			tokenLog.Warn("查询兼任学校成员失败，本校个人账户本轮不并入（仅辖区范围）",
				"school", school.ID, "error", mErr)
		}
	}

	// ---- 特权剔除：一次查询覆盖两个成员集合（失败仅 Warn 不剔除，同 v172 选项甲）----
	tokenRemovePrivilegedFromSets(ctx, memberIDSet, ownMemberSet)

	// ---- 组装 ----
	// OwnerIDs = 辖区学校 owner_id ∪ 兼任本校ID ∪ 兼任本校成员 user_id
	//（辖区其他学校仍只到学校一层不含成员——Q1 决策：区域只分配到学校，不越级到别校个人）
	ownerSet := make(map[string]struct{}, len(schoolIDSet)+len(ownMemberSet)+1)
	for sid := range schoolIDSet {
		ownerSet[sid] = struct{}{}
	}
	if ownSchoolID != "" {
		ownerSet[ownSchoolID] = struct{}{}
	}
	for uid := range ownMemberSet {
		ownerSet[uid] = struct{}{}
	}
	// UserIDs = 辖区成员 ∪ 兼任本校成员（去重）
	for uid := range ownMemberSet {
		memberIDSet[uid] = struct{}{}
	}

	// AllowedRegionOwnerIDs = 管辖的区域账户 owner_id（区域组织ID）
	allowed := make([]string, 0, len(regionIDs))
	allowed = append(allowed, regionIDs...)

	return &TokenScope{
		Role:                  role,
		IsAdmin:               false,
		OwnerIDs:              tokenSetToSlice(ownerSet),
		UserIDs:               tokenSetToSlice(memberIDSet),
		AllowedRegionOwnerIDs: allowed,
	}
}

// ==================== senior_operator 分支（主链路=本校 + 加料=区域管辖）====================

// resolveTokenSeniorScope 解析 users.role=senior_operator 的可见范围
func (s *TokenService) resolveTokenSeniorScope(ctx context.Context, userID string) *TokenScope {
	role := models.RoleSeniorOperator

	// ---- 主链路①：本校（失败→Blocked，与 v172 旧行为逐字一致）----
	school, err := repository.GetSchoolByAdminUserID(ctx, userID)
	if err != nil || school == nil || school.ID == "" {
		return tokenBlockedScope(role, "您尚未绑定学校，请联系系统管理员")
	}

	// ---- 主链路②：本校成员（失败→Blocked）----
	memberIDs, mErr := repository.ListSchoolMemberIDs(ctx, school.ID)
	if mErr != nil {
		tokenLog.Warn("查询本校成员失败，收窄为空集", "school", school.ID, "error", mErr)
		return tokenBlockedScope(role, "查询本校成员失败")
	}
	ownMemberSet := make(map[string]struct{}, len(memberIDs))
	for _, uid := range memberIDs {
		ownMemberSet[uid] = struct{}{}
	}

	// ---- 加料（best-effort）：兼任区域管理员则并入辖区 region 视角 ----
	// ListRegionIDsByAdmin 查询失败仅 Warn 跳过（保住本校主范围，不瘫痪）；
	// 无任命（空列表）是普通校管常态，零额外行为。
	var regionIDs []string
	if rIDs, rErr := repository.ListRegionIDsByAdmin(ctx, userID); rErr != nil {
		tokenLog.Warn("查询区域管辖任命失败，本轮不并入辖区（仅本校范围）", "user", userID, "error", rErr)
	} else {
		regionIDs = rIDs
	}

	regionSchoolSet := make(map[string]struct{})
	regionMemberSet := make(map[string]struct{})
	if len(regionIDs) > 0 {
		// 逐区域递归学校（加料语义：单区域失败仅 Warn 跳过该区域，不阻断）
		for _, regionID := range regionIDs {
			schoolIDs, sErr := repository.ListDescendantSchoolIDs(ctx, regionID)
			if sErr != nil {
				tokenLog.Warn("递归查询辖区学校失败，该区域本轮跳过", "region", regionID, "error", sErr)
				continue
			}
			for _, sid := range schoolIDs {
				regionSchoolSet[sid] = struct{}{}
			}
		}
		// 辖区成员 user_id（消费明细维度，单校失败跳过）
		tokenCollectSchoolMembersBestEffort(ctx, regionSchoolSet, regionMemberSet)
	}

	// ---- 特权剔除：一次查询覆盖两个成员集合 ----
	tokenRemovePrivilegedFromSets(ctx, ownMemberSet, regionMemberSet)

	// ---- 组装 ----
	// OwnerIDs = 本校成员 user_id ∪ 本校ID ∪ 辖区学校 owner_id
	//（辖区学校只到学校一层，不含其成员个人账户——与 region_admin 的 Q1 决策同口径）
	ownerSet := make(map[string]struct{}, len(ownMemberSet)+len(regionSchoolSet)+1)
	for uid := range ownMemberSet {
		ownerSet[uid] = struct{}{}
	}
	ownerSet[school.ID] = struct{}{}
	for sid := range regionSchoolSet {
		ownerSet[sid] = struct{}{}
	}
	// UserIDs = 本校成员 ∪ 辖区成员（去重）
	userSet := make(map[string]struct{}, len(ownMemberSet)+len(regionMemberSet))
	for uid := range ownMemberSet {
		userSet[uid] = struct{}{}
	}
	for uid := range regionMemberSet {
		userSet[uid] = struct{}{}
	}

	// AllowedRegionOwnerIDs：有区域任命才非空（普通校管保持 nil——语义不变，
	// tokenSourceAllowed 对 region 来源账户 fail-closed，my-region-accounts 返空列表）
	var allowed []string
	if len(regionIDs) > 0 {
		allowed = make([]string, 0, len(regionIDs))
		allowed = append(allowed, regionIDs...)
	}

	return &TokenScope{
		Role:                  role,
		IsAdmin:               false,
		OwnerIDs:              tokenSetToSlice(ownerSet),
		UserIDs:               tokenSetToSlice(userSet),
		AllowedRegionOwnerIDs: allowed,
	}
}

// ==================== 私有辅助 ====================

// tokenBlockedScope 构造 fail-closed 空集范围（统一出口，空切片=匹配空集绝非不过滤）
func tokenBlockedScope(role string, reason string) *TokenScope {
	return &TokenScope{
		Role:          role,
		IsAdmin:       false,
		OwnerIDs:      []string{},
		UserIDs:       []string{},
		Blocked:       true,
		BlockedReason: reason,
	}
}

// tokenCollectSchoolMembersBestEffort 逐校汇总成员 user_id 到 into 集合
//（单校查询失败仅 Warn 跳过该校，不整体瘫痪——与 v172 region 分支既有策略一致）
func tokenCollectSchoolMembersBestEffort(ctx context.Context, schoolIDs map[string]struct{}, into map[string]struct{}) {
	for sid := range schoolIDs {
		memberIDs, err := repository.ListSchoolMemberIDs(ctx, sid)
		if err != nil {
			tokenLog.Warn("查询学校成员失败，该校成员本轮跳过（消费明细不含该校）",
				"school", sid, "error", err)
			continue
		}
		for _, uid := range memberIDs {
			into[uid] = struct{}{}
		}
	}
}

// tokenRemovePrivilegedFromSets 从若干成员集合中剔除特权账户(admin/region_admin)
//
// 根因回顾（积分越权修复）：admin 因 migration/group_member 被登记进 school_members，
// 若不剔除，admin 的个人账户会落进 senior/region 白名单导致越权可见与分配。
// fail-closed 策略同 v172（选项甲）：查特权名单失败时记 Warn 但不剔除，
// 不让查询整体瘫痪（账户列表层另有 account_type<>'region' 等防线兜底）。
func tokenRemovePrivilegedFromSets(ctx context.Context, sets ...map[string]struct{}) {
	privilegedIDs, err := repository.ListPrivilegedUserIDs(ctx)
	if err != nil {
		tokenLog.Warn("查询特权用户列表失败，本轮不剔除（继续用原成员列表）", "error", err)
		return
	}
	for _, set := range sets {
		for _, id := range privilegedIDs {
			delete(set, id)
		}
	}
}

// tokenSetToSlice 集合转切片（恒返回非nil切片，保证空集语义为"匹配空集"而非"不过滤"）
func tokenSetToSlice(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	return out
}

package services

// lesson_plan_shared_scope.go — 教案库"共享可见作者白名单"解析器（共享库数据隔离收窄）
//
// 背景（2026-07-03 修复）：
//   教案列表可见性 OR 子句的 B 分支（lp.status IN ('published_shared','approved')）
//   历史上没有任何作者范围限制——任何登录用户（含学校管理员）都能在教案库
//   （教研组库/学校库/区域库三个 Tab）看到全平台所有已共享/已通过的教案，属数据泄漏。
//
// 修复方案：
//   给 B 分支追加"共享可见作者白名单"——非 admin 用户只能看到白名单内作者的共享教案。
//   白名单 = 同教研组成员 ∪ 同校成员 ∪ 所在区域成员 ∪ 管辖范围成员 ∪ 自己：
//     1) 管辖范围成员：senior_operator/region_admin 的 DataScope.UserIDs
//        （本校成员 ∪ 管辖区域树下全部学校成员，由 ResolveDataScope 解析）。
//        普通老师（operator/viewer）的 UserIDs 只含自己，并入无副作用；
//     2) 同校成员：GetSchoolIDByUserID 解析本人学校 → ListSchoolMemberIDs 取全体成员；
//     3) 所在区域成员：本校组织的 parent_id（父区域）→ ListDescendantSchoolIDs 递归
//        取父区域树下全部 active 学校 → 逐校 ListSchoolMemberIDs 汇总（"区域库"语义落点）；
//     4) 同教研组成员：GetUserTeachingGroups 取本人全部教研组 → 逐组
//        ListTeachingGroupMemberIDs 汇总（教研组可能跨校协作，独立于同校层并入）。
//
// 设计范式：
//   镜像 courseware_share_service.go 的 resolveSameOrgUserIDs（课件共享库同款可见性口径），
//   在其"同校 ∪ 同组"基础上按需求增加"同区域"与"管辖并集"两层。
//   fail-closed 纪律：任一子查询失败仅记 Warn 跳过该层，白名单永远至少含本人——
//   最坏情况用户只看得到自己的共享教案，绝不放大可见范围。
//
// 性能说明：
//   每次教案列表请求解析一次（多次小查询）。当前平台规模下开销可忽略；
//   若未来区域规模扩大，可在本函数内加进程内短 TTL 缓存（按 userID 键），调用方零改动。
//   admin 与"我的教案页"两条路径在 service 层已跳过本解析
//   （见 lesson_plan_service.ListLessonPlans），不产生额外查询。
//
// 消费方：lesson_plan_service.ListLessonPlans（唯一调用点），
//         解析结果作为 sharedAuthorIDs 传 repository.ListLessonPlans 拼入 B 分支 SQL。

import (
	"context"

	"tedna/internal/logger"
	"tedna/internal/repository"
)

// lpSharedScopeLog 模块级结构化日志器
var lpSharedScopeLog = logger.WithModule("services.lp_shared_scope")

// resolveLPSharedVisibleAuthorIDs 解析当前用户在教案库中"共享可见"的作者白名单
//
// 入参：
//   - userID：当前登录用户ID（handler 保证非空；极端空串时白名单可能为空集=fail-closed）
//   - scope ：ResolveDataScope 解析出的数据范围（可为 nil；nil 时跳过管辖并集层）
//
// 返回：
//   - 非 nil 切片，去重后的作者ID白名单；正常情况至少含 userID 本人。
//
// 注意：本函数【不】处理 admin——admin 在 repo 层不拼可见性子句（scopeIsAdmin=true），
// 调用方（lesson_plan_service.ListLessonPlans）已在 admin 路径上跳过本函数。
func resolveLPSharedVisibleAuthorIDs(ctx context.Context, userID string, scope *DataScope) []string {
	// 白名单容器（map 去重）。永远先放入本人：任何一层解析失败，
	// 用户至少能看到自己共享的教案（与课件共享库 resolveSameOrgUserIDs 兜底口径一致）。
	idSet := make(map[string]struct{})
	if userID != "" {
		idSet[userID] = struct{}{}
	}

	// ---------- 第 1 层：管理类角色的管辖并集 ----------
	// senior_operator 的 scope.UserIDs = 本校成员 ∪ 管辖区域成员；
	// region_admin    的 scope.UserIDs = 辖区全部学校成员；
	// operator/viewer 的 scope.UserIDs = [自己]（并入等价于无操作）。
	// scope.Blocked（senior 未绑校等 fail-closed 场景）时 UserIDs 为空切片，天然跳过——
	// 此时靠下方第 2~4 层按成员归属（school_members/教研组）解析，未绑校管的 senior
	// 若本人在 school_members 里仍能看到本校范围，符合"成员归属"语义。
	if scope != nil && !scope.IsAdmin {
		for _, uid := range scope.UserIDs {
			idSet[uid] = struct{}{}
		}
	}

	// ---------- 第 2 层：同校成员 + 第 3 层：所在区域成员 ----------
	// GetSchoolIDByUserID：school_members 权威查 + 教研组兜底反查，
	// 查不到（未绑任何学校）返回空串非错误，此时同校/同区域两层静默跳过。
	schoolID, sErr := repository.GetSchoolIDByUserID(ctx, userID)
	if sErr != nil {
		lpSharedScopeLog.Warn("共享白名单：解析本人学校失败（跳过同校与同区域层）", "user", userID, "error", sErr)
	}
	if schoolID != "" {
		// 第 2 层：同校全体成员
		if memberIDs, mErr := repository.ListSchoolMemberIDs(ctx, schoolID); mErr == nil {
			for _, uid := range memberIDs {
				idSet[uid] = struct{}{}
			}
		} else {
			lpSharedScopeLog.Warn("共享白名单：查询同校成员失败（跳过该层）", "school", schoolID, "error", mErr)
		}

		// 第 3 层：所在区域成员——本校组织的 parent_id 即父区域，
		// 沿 ListDescendantSchoolIDs（WITH RECURSIVE）取父区域树下全部 active 学校，
		// 逐校汇总成员。本校已在第 2 层并入，此处 continue 省一次查询（map 本身也去重）。
		// GetOrganizationByID 失败/本校无父区域（ParentID 为 nil 或空）→ 同区域层静默跳过。
		if org, oErr := repository.GetOrganizationByID(ctx, schoolID); oErr == nil && org != nil && org.ParentID != nil && *org.ParentID != "" {
			if regionSchoolIDs, rErr := repository.ListDescendantSchoolIDs(ctx, *org.ParentID); rErr == nil {
				for _, sid := range regionSchoolIDs {
					if sid == schoolID {
						continue // 本校成员已并入
					}
					if mids, mErr2 := repository.ListSchoolMemberIDs(ctx, sid); mErr2 == nil {
						for _, uid := range mids {
							idSet[uid] = struct{}{}
						}
					} else {
						lpSharedScopeLog.Warn("共享白名单：查询区域内学校成员失败（跳过该校）", "school", sid, "error", mErr2)
					}
				}
			} else {
				lpSharedScopeLog.Warn("共享白名单：递归查询父区域学校失败（跳过同区域层）", "region", *org.ParentID, "error", rErr)
			}
		}
	}

	// ---------- 第 4 层：同教研组成员（教研组可能跨校协作，独立于同校层并入） ----------
	if groups, gErr := repository.GetUserTeachingGroups(ctx, userID); gErr == nil {
		for _, g := range groups {
			if members, mErr := repository.ListTeachingGroupMemberIDs(ctx, g.ID); mErr == nil {
				for _, uid := range members {
					idSet[uid] = struct{}{}
				}
			} else {
				lpSharedScopeLog.Warn("共享白名单：查询教研组成员失败（跳过该组）", "group", g.ID, "error", mErr)
			}
		}
	} else {
		lpSharedScopeLog.Warn("共享白名单：查询本人教研组失败（跳过同组层）", "user", userID, "error", gErr)
	}

	// 收拢为切片返回（非 nil 保证；极端空 userID 时可能为空切片=匹配空集，fail-closed）
	out := make([]string, 0, len(idSet))
	for uid := range idSet {
		out = append(out, uid)
	}
	return out
}

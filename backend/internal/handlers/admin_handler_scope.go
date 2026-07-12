package handlers

// admin_handler_scope.go — 用户管理中心·区域管理员(region_admin)只读视图的数据范围解析
//
// 背景:
//   前端 AdminPage 已为 region_admin 提供"用户 Tab"收窄只读视图(users+orgs 两 Tab、
//   隐藏新建/批量按钮),但后端此前对 region_admin 全拦(中间件不放行 + scope 解析无分支),
//   导致区域管理员打开用户 Tab 直接 403。本文件补齐"读"这一侧的数据范围解析。
//
// 设计边界(与 admin_handler.go 的既有口径协同):
//   - region_admin 在用户管理中心【只读】:
//       读  → 用户列表按"辖区学校白名单"过滤(本文件 resolveAdminUserListScope);
//             用户详情/课程分配读取经 ensureUserInScope 的 region 分支
//             (repository.IsUserInSchools, 与列表 SQL 同口径)。
//       写  → 路由层 regionReadOnlyGate(GET-only 门, routes_admin.go)第一道拦截,
//             主文件各写端点 ensureRegionAdminReadOnly 双保险,一律 403。
//   - resolveSchoolScope(admin_handler.go)保持不加 region 分支:
//     它是"单校写语义"(批量导入/建号),region 走它落 default"权限不足"正是预期。
//
// 辖区学校白名单口径(与 services/data_scope.go 的 B2 并集决策一致):
//   区域主来源 repository.ListRegionIDsByAdmin(organization_admins ∪ organizations.admin_user_id
//   双来源 UNION) → 逐区域 ListDescendantSchoolIDs 递归取辖下学校 → 并入"本人本校"加料
//   (双重身份 best-effort)。主链路任一步失败即返回错误(fail-closed),加料失败仅静默跳过
//   (并集只会更小不会放大,不违背 fail-closed)。

import (
	"context"
	"fmt"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/repository"
)

// ==================== 数据范围结构 ====================

// adminUserListScope 用户列表查询的数据范围(两字段互斥,由 resolveAdminUserListScope 保证):
//   - SchoolID  : 单校精确过滤(admin 传参可空=全系统 / senior 强制本校),沿用既有口径;
//   - SchoolIDs : 学校白名单(仅 region_admin 非 nil):
//       非空       → 仅名单内学校的成员可见;
//       非nil空切片 → 匹配空集(辖区无学校,fail-closed 看不到任何用户,非错误)。
type adminUserListScope struct {
	SchoolID  string
	SchoolIDs []string
}

// ==================== 只读写保护 ====================

// ensureRegionAdminReadOnly 写操作准入守卫: region_admin 在用户管理中心为只读,一律拒绝。
//
// 第一道拦截在路由层 regionReadOnlyGate(routes_admin.go, 非 GET 即 403);本函数是
// handler 内的双保险,防将来路由重排/中间件配置疏漏导致第一道门失效。
// 仅按角色判断,调用方传入 claims.Role(各写端点均已先取 claims,免重复解析 ctx)。
func ensureRegionAdminReadOnly(role string) error {
	if role == models.RoleRegionAdmin {
		return fmt.Errorf("区域管理员在用户管理中心为只读,如需调整用户请联系系统管理员")
	}
	return nil
}

// ==================== 辖区学校白名单解析 ====================

// resolveRegionScopeSchoolIDs 解析区域管理员的"辖区学校ID白名单"
//
// 流程(严格 fail-closed,镜像 services/data_scope.go 的 buildUnionScope 主链路语义):
//  1. 主来源 ListRegionIDsByAdmin 取管辖区域(双来源 UNION);查询失败/无任何区域 → 返回错误;
//  2. 逐区域 ListDescendantSchoolIDs 递归取辖下学校,任一区域查询失败 → 返回错误(主链路不容错);
//  3. 加料(best-effort): 本人若同时是某学校的管理员(GetSchoolByAdminUserID 两级查找),
//     并入本校;查不到/查询失败仅跳过(范围只会更小)。
//
// 返回的切片恒为非 nil:辖区区域存在但无任何学校时返回空切片(=匹配空集,列表为空,非错误)。
func resolveRegionScopeSchoolIDs(ctx context.Context, userID string) ([]string, error) {
	// 1. 主来源: 管辖区域(双来源 UNION,与 data_scope/token_service 的 region 分支同源)
	regionIDs, err := repository.ListRegionIDsByAdmin(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("查询管辖区域失败")
	}
	if len(regionIDs) == 0 {
		return nil, fmt.Errorf("您尚未被任命为任何区域的管理员")
	}

	// 2. 逐区域递归取辖下学校(map 去重;主链路任一步失败即整体失败,fail-closed)
	schoolIDSet := make(map[string]struct{})
	for _, regionID := range regionIDs {
		schoolIDs, sErr := repository.ListDescendantSchoolIDs(ctx, regionID)
		if sErr != nil {
			return nil, fmt.Errorf("查询辖区学校失败")
		}
		for _, sid := range schoolIDs {
			schoolIDSet[sid] = struct{}{}
		}
	}

	// 3. 双重身份加料(best-effort): 本人若同时是某学校管理员,并入本校。
	//    与组织架构 Tab(经 ResolveDataScope 的并集)口径一致,避免两个 Tab 可见范围不一致。
	//    ErrOrgNotFound=不是任何学校的管理员属正常;任何错误一律静默跳过(范围只会更小)。
	if school, sErr := repository.GetSchoolByAdminUserID(ctx, userID); sErr == nil && school != nil && school.ID != "" {
		schoolIDSet[school.ID] = struct{}{}
	}

	out := make([]string, 0, len(schoolIDSet))
	for sid := range schoolIDSet {
		out = append(out, sid)
	}
	return out, nil
}

// ==================== 用户列表范围解析 ====================

// resolveAdminUserListScope 用户列表(GET /admin/users)的数据范围解析(读语义)
//
//   - admin           : 沿用 resolveSchoolScope —— SchoolID=前端传参(可空=全系统);
//   - senior_operator : 沿用 resolveSchoolScope —— SchoolID=强制本校;
//   - region_admin    : SchoolIDs=辖区学校白名单;若前端另传了 school_id 筛选,
//     必须落在白名单内(收窄为该单校),否则 403"该学校不在您的管辖范围内",
//     绝不允许借筛选参数越出辖区;
//   - 其他角色        : 走 resolveSchoolScope 的 default → "权限不足"。
func resolveAdminUserListScope(ctx context.Context, requestedSchoolID string) (adminUserListScope, error) {
	claims, ok := middleware.GetClaims(ctx)
	if !ok {
		return adminUserListScope{}, fmt.Errorf("未登录")
	}

	// region_admin: 辖区学校白名单口径
	if claims.Role == models.RoleRegionAdmin {
		schoolIDs, err := resolveRegionScopeSchoolIDs(ctx, claims.UserID)
		if err != nil {
			return adminUserListScope{}, err
		}
		// 前端传了 school_id 筛选 → 必须在白名单内,收窄为该单校(仍走白名单字段防口径分叉)
		if requestedSchoolID != "" {
			found := false
			for _, sid := range schoolIDs {
				if sid == requestedSchoolID {
					found = true
					break
				}
			}
			if !found {
				return adminUserListScope{}, fmt.Errorf("该学校不在您的管辖范围内")
			}
			return adminUserListScope{SchoolIDs: []string{requestedSchoolID}}, nil
		}
		return adminUserListScope{SchoolIDs: schoolIDs}, nil
	}

	// admin / senior_operator / 其他: 沿用既有单校口径(行为逐字不变)
	schoolID, _, err := resolveSchoolScope(ctx, requestedSchoolID)
	if err != nil {
		return adminUserListScope{}, err
	}
	return adminUserListScope{SchoolID: schoolID}, nil
}

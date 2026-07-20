package handlers

// admin_handler_scope.go
//
// 用户管理中心区域管理员只读视图的数据范围解析。
//
// 设计边界：
//   - region_admin 在用户管理中心仅允许读取；
//   - 写操作由 routes_admin.go 的 regionReadOnlyGate 和
//     ensureRegionAdminReadOnly 两道保护统一拒绝；
//   - 用户列表、用户详情和课程分配读取必须使用相同学校白名单。
//
// 上下文 5：区域管理员学校范围同域过滤
//
//   本文件不再自行递归组织树，也不再把“本人兼任学校管理员的本校”无条件并入。
//   学校白名单统一委托 services.ResolveRegionAdminEducationScope，确保与：
//     - services.ResolveDataScope；
//     - 组织架构列表；
//     - 教研组学校范围；
//     - 课程和作者依赖的用户白名单；
//   使用完全相同的授权边界。
//
// 统一范围为：
//   管辖区域树下学校
//   AND 学校 status='active'
//   AND 学校 education_domain=区域管理员固定教育域
//
// fail-closed：
//   教育域未配置、非法、多域冲突、无区域任命或数据库错误时，返回错误，
//   不回退 K12，不回退全局学校，也不返回部分范围。

import (
	"context"
	"fmt"

	"tedna/internal/middleware"
	"tedna/internal/models"
	"tedna/internal/services"
)

// adminUserListScope 用户列表查询的数据范围。
//
// 两字段互斥：
//   - SchoolID：admin 或 senior_operator 的既有单校筛选；
//   - SchoolIDs：region_admin 的学校白名单。
//
// SchoolIDs 三态：
//   - nil：不使用学校白名单字段；
//   - 非 nil 空切片：匹配空集；
//   - 非空：只允许名单中的学校成员。
type adminUserListScope struct {
	SchoolID  string
	SchoolIDs []string
}

// ensureRegionAdminReadOnly 是区域管理员用户管理写操作的 Handler 层双保险。
func ensureRegionAdminReadOnly(role string) error {
	if role == models.RoleRegionAdmin {
		return fmt.Errorf(
			"区域管理员在用户管理中心为只读，如需调整用户请联系系统管理员",
		)
	}
	return nil
}

// resolveRegionScopeSchoolIDs 解析区域管理员统一同域学校白名单。
//
// 该函数仅做 Handler 层适配，不再保存独立的范围业务规则。
// 真正的范围解析由 services.ResolveRegionAdminEducationScope 统一完成。
//
// 返回值恒为非 nil：
//   - 配置正确但当前域没有学校时返回空切片；
//   - 配置或数据库异常时返回错误。
func resolveRegionScopeSchoolIDs(
	ctx context.Context,
	userID string,
) ([]string, error) {
	resolvedScope, err :=
		services.ResolveRegionAdminEducationScope(
			ctx,
			userID,
		)
	if err != nil {
		return nil, err
	}

	schoolIDs := make(
		[]string,
		len(resolvedScope.SchoolIDs),
	)
	copy(schoolIDs, resolvedScope.SchoolIDs)

	return schoolIDs, nil
}

// resolveAdminUserListScope 解析 GET /admin/users 的读取范围。
//
// 规则：
//   - admin：沿用 resolveSchoolScope，school_id 可空；
//   - senior_operator：沿用 resolveSchoolScope，强制本校；
//   - region_admin：使用统一同域学校白名单；
//   - 其它角色：由 resolveSchoolScope 拒绝。
//
// region_admin 传入 requestedSchoolID 时：
//   - 该学校必须位于统一白名单中；
//   - 合法时收窄成单校白名单；
//   - 非法时返回“该学校不在您的管辖范围内”；
//   - 不允许通过伪造 school_id 扩大权限。
func resolveAdminUserListScope(
	ctx context.Context,
	requestedSchoolID string,
) (adminUserListScope, error) {
	claims, ok := middleware.GetClaims(ctx)
	if !ok {
		return adminUserListScope{}, fmt.Errorf("未登录")
	}

	if claims.Role == models.RoleRegionAdmin {
		schoolIDs, err := resolveRegionScopeSchoolIDs(
			ctx,
			claims.UserID,
		)
		if err != nil {
			return adminUserListScope{}, err
		}

		if requestedSchoolID != "" {
			found := false
			for _, schoolID := range schoolIDs {
				if schoolID == requestedSchoolID {
					found = true
					break
				}
			}

			if !found {
				return adminUserListScope{}, fmt.Errorf(
					"该学校不在您的管辖范围内",
				)
			}

			return adminUserListScope{
				SchoolIDs: []string{requestedSchoolID},
			}, nil
		}

		return adminUserListScope{
			SchoolIDs: schoolIDs,
		}, nil
	}

	// admin、senior_operator 和其它角色沿用既有单校解析规则。
	schoolID, _, err := resolveSchoolScope(
		ctx,
		requestedSchoolID,
	)
	if err != nil {
		return adminUserListScope{}, err
	}

	return adminUserListScope{
		SchoolID: schoolID,
	}, nil
}

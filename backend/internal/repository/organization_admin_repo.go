package repository

// organization_admin_repo.go — 组织多管理员数据访问层（迭代一 Phase 5 从 organization_repo.go 抽出）
//
// 抽出动机：
//   organization_repo.go 已达 1069 行，超过项目 600 行/文件红线。
//   本文件承载"组织全部管理员"(organization_admins 表)相关的数据访问，
//   既给 organization_repo.go 瘦身开个头，也让多管理员逻辑聚合在一处便于维护。
//
// organization_admins 是"组织全部管理员"的权威来源（迭代一新增）。
// 与 organizations.admin_user_id 单字段并存：旧字段保留作"主管理员"兼容，本表作"全部管理员"权威。
// role_type：region_admin（org 须为 region）/ school_admin（org 须为 school），由 service/handler 层保证类型匹配。
//
// 本文件函数：
//   - AddOrgAdmin / RemoveOrgAdmin / ListOrgAdmins / CountOrgAdminsByUser / ListRegionIDsByAdmin
//     （从 organization_repo.go 逐字搬入，SQL 与语义完全不变；ListRegionIDsByAdmin 于 B2 升级双来源，见函数注释）
//   - UpdateOrganizationAdminUserID（Phase 5 新增）：只更新 organizations.admin_user_id 单字段，
//     供 service 层在任命/移除 school_admin 时回填"主管理员"指针，保证现有 GetSchoolByAdminUserID
//     单字段判定链路（getCurrentSchoolAdminContext / ResolveDataScope senior 分支等）不失效。

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// AddOrgAdmin 任命某用户为某组织的管理员
// - 幂等：ON CONFLICT (org_id, user_id) 不报错（同组织同用户只一条）
// - createdBy 任命人（审计用，可空）
func AddOrgAdmin(ctx context.Context, orgID string, userID string, roleType string, createdBy string) error {
	if orgID == "" || userID == "" {
		return fmt.Errorf("orgID 或 userID 为空")
	}
	if roleType != "region_admin" && roleType != "school_admin" {
		return fmt.Errorf("无效的管理员类型: %s", roleType)
	}
	var createdByArg interface{}
	if createdBy == "" {
		createdByArg = nil
	} else {
		createdByArg = createdBy
	}
	_, err := database.DB.Exec(ctx, `
		INSERT INTO organization_admins (org_id, user_id, role_type, created_by)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (org_id, user_id) DO NOTHING
	`, orgID, userID, roleType, createdByArg)
	if err != nil {
		return fmt.Errorf("任命组织管理员失败: %w", err)
	}
	return nil
}

// RemoveOrgAdmin 移除某组织的某管理员
// - 找不到记录返回 ErrMemberNotFound（供上层判断）
func RemoveOrgAdmin(ctx context.Context, orgID string, userID string) error {
	result, err := database.DB.Exec(ctx,
		`DELETE FROM organization_admins WHERE org_id = $1 AND user_id = $2`,
		orgID, userID,
	)
	if err != nil {
		return fmt.Errorf("移除组织管理员失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

// ListOrgAdmins 列出某组织的全部管理员（含用户名、显示名、类型、任命时间）
func ListOrgAdmins(ctx context.Context, orgID string) ([]*models.OrganizationAdminItem, error) {
	query := `
		SELECT oa.org_id, oa.user_id, u.username, u.display_name, oa.role_type,
		       COALESCE(oa.created_by::text, ''), oa.created_at
		FROM organization_admins oa
		JOIN users u ON u.id = oa.user_id
		WHERE oa.org_id = $1
		ORDER BY oa.role_type, oa.created_at
	`
	rows, err := database.DB.Query(ctx, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("查询组织管理员列表失败: %w", err)
	}
	defer rows.Close()

	items := []*models.OrganizationAdminItem{}
	for rows.Next() {
		item := &models.OrganizationAdminItem{}
		if err := rows.Scan(
			&item.OrgID, &item.UserID, &item.Username, &item.DisplayName,
			&item.RoleType, &item.CreatedBy, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描组织管理员行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// CountOrgAdminsByUser 统计某用户当前还担任多少个组织的管理员
// 用途：移除某组织管理员后，判断该用户是否还是任何组织的管理员（若否，service 层可据此决定是否降级其角色）
func CountOrgAdminsByUser(ctx context.Context, userID string) (int, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM organization_admins WHERE user_id = $1`,
		userID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计用户组织管理员身份失败: %w", err)
	}
	return count, nil
}

// ListRegionIDsByAdmin 查询某用户管辖的所有区域组织ID（B2 修复：双来源并集）
//
// 用途：ResolveDataScope（组织/教案可见范围）与 ResolveTokenScope（积分范围）的
//
//	region_admin 分支——确定该用户管辖哪些区域。改本函数一处，两条消费链同时修复。
//
// ⚠ B2 修复（"区域管理员组织架构不显示辖下学校"根治）：
//
//	历史实现只查 organization_admins(role_type='region_admin') 单一来源。而当前 UI 里
//	任命区域管理员的唯一入口是"编辑区域"弹窗（OrgFormModal）写 organizations.admin_user_id
//	单字段——区域卡片没有多管理员任命面板（OrgAdminsPanel 只挂在学校卡片上）。
//	于是单字段任命的区域管理员在本函数落空 → ResolveDataScope 收窄为 Blocked 空集 →
//	组织架构/教案范围全空。这也是 B11"区域只能添加 1 个区域管理员"的同根现象。
//
//	修法与 B1（GetSchoolByAdminUserID 两级查找）同哲学、方向互补：
//	  来源① organization_admins WHERE role_type='region_admin'（多管理员表，Phase 6.4 任命路径）
//	         —— 本轮补 JOIN organizations 校验目标确为 active 的 region（防御脏数据/已禁用区域）
//	  来源② organizations WHERE admin_user_id=$1 AND type='region' AND status='active'（单字段任命路径）
//	  两来源 UNION 天然去重。
//
//	行为变化说明：来源①新增了"区域须 active"过滤——被禁用区域的管理员不再获得该区域管辖
//	（与 ListDescendantSchoolIDs 只返回 active 学校的口径一致，属修正而非回归）。
//
// 返回空切片（非 nil）表示该用户不是任何区域的管理员。
func ListRegionIDsByAdmin(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return []string{}, nil
	}
	query := `
		SELECT oa.org_id AS id
		FROM organization_admins oa
		JOIN organizations o ON o.id = oa.org_id AND o.type = 'region' AND o.status = 'active'
		WHERE oa.user_id = $1 AND oa.role_type = 'region_admin'
		UNION
		SELECT id
		FROM organizations
		WHERE admin_user_id = $1 AND type = 'region' AND status = 'active'
	`
	rows, err := database.DB.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户管辖区域失败: %w", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历用户管辖区域结果失败: %w", err)
	}
	return ids, nil
}

// ListSchoolAdminUserIDs 列出某组织全部 role_type='school_admin' 的管理员 user_id
//
// Phase 5 新增，供 service 层在"移除主管理员后挑选补位者"时使用：
//
//	移除某 school_admin 后，若该用户正是 organizations.admin_user_id 指向的主管理员，
//	service 需从本表剩余的 school_admin 里挑一个回填单字段。本函数返回候选列表。
//
// 返回空切片（非 nil）表示该组织已无任何 school_admin。
func ListSchoolAdminUserIDs(ctx context.Context, orgID string) ([]string, error) {
	if orgID == "" {
		return []string{}, nil
	}
	rows, err := database.DB.Query(ctx,
		`SELECT user_id FROM organization_admins
		 WHERE org_id = $1 AND role_type = 'school_admin'
		 ORDER BY created_at`,
		orgID,
	)
	if err != nil {
		return nil, fmt.Errorf("查询学校管理员候选列表失败: %w", err)
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil {
			ids = append(ids, id)
		}
	}
	return ids, nil
}

// UpdateOrganizationAdminUserID 只更新 organizations.admin_user_id 单字段（Phase 5 新增）
//
// 为什么需要独立函数：
//
//	现有 UpdateOrganization 是全字段更新（要传完整 UpdateOrganizationRequest，会一并改 name/settings/status），
//	不适合在"任命/移除管理员"这种只想动 admin_user_id 一个字段的场景使用。本函数专做单字段更新。
//
// 参数 adminUserID 用 *string：
//   - 非 nil → 回填为该用户（任命主管理员）
//   - nil    → 置空（移除主管理员且无补位者时，单字段清空）
//
// 注意：本函数只动 admin_user_id 单字段，不碰 organization_admins 多管理员表；
//
//	两者的一致性由 service 层编排（先写多管理员表，再据规则回填本单字段）。
func UpdateOrganizationAdminUserID(ctx context.Context, orgID string, adminUserID *string) error {
	if orgID == "" {
		return fmt.Errorf("orgID 为空")
	}
	result, err := database.DB.Exec(ctx,
		`UPDATE organizations SET admin_user_id = $1, updated_at = now() WHERE id = $2`,
		adminUserID, orgID,
	)
	if err != nil {
		return fmt.Errorf("更新组织主管理员失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrOrgNotFound
	}
	return nil
}

// ListRegionAdminUserIDsBySchool 由学校组织ID正向解析出该校所在区域的全部区域管理员 user_id
//
// 用途（Token 积分自动分配 2026-07-04 新增）：
//
//	学校积分池不足时，需通知该校所在区域的区域管理员前来充值。本函数完成
//	"学校 → 其 parent_id 区域 → 该区域全部 region_admin" 的正向解析（现有函数都是反向）。
//
// 双来源并集（与 ListRegionIDsByAdmin 反向口径对称，两处口径一致避免遗漏）：
//
//	来源① organization_admins WHERE org_id=区域 AND role_type='region_admin'（多管理员表）
//	来源② organizations WHERE id=区域 AND admin_user_id 非空（单字段任命路径）
//	两来源 UNION 天然去重。均要求区域 status='active'。
//
// 链路：
//  1. 取学校组织，读其 parent_id（=区域ID）；学校无父区域 → 返回空切片（非 nil）。
//  2. 用区域ID双来源查出全部区域管理员 user_id。
//
// fail-safe：schoolOrgID 为空、学校不存在、学校无父区域、区域无管理员 → 均返回空切片（非 nil），
//
//	不返回错误（调用方据"空=无人可通知"处理，不阻断主流程）；仅真正的 DB 异常上抛。
func ListRegionAdminUserIDsBySchool(ctx context.Context, schoolOrgID string) ([]string, error) {
	if schoolOrgID == "" {
		return []string{}, nil
	}

	// 1. 取学校的 parent_id（区域ID）
	var parentID *string
	err := database.DB.QueryRow(ctx,
		`SELECT parent_id FROM organizations WHERE id = $1 AND type = 'school'`,
		schoolOrgID,
	).Scan(&parentID)
	if err != nil {
		// 学校不存在等 → 无人可通知，返回空切片不报错（best-effort 语义）
		return []string{}, nil
	}
	if parentID == nil || *parentID == "" {
		// 学校无父区域（游离学校）→ 无区域管理员可通知
		return []string{}, nil
	}
	regionID := *parentID

	// 2. 双来源查该区域全部区域管理员 user_id
	query := `
SELECT oa.user_id
FROM organization_admins oa
JOIN organizations o ON o.id = oa.org_id AND o.type = 'region' AND o.status = 'active'
WHERE oa.org_id = $1 AND oa.role_type = 'region_admin'
UNION
SELECT admin_user_id
FROM organizations
WHERE id = $1 AND type = 'region' AND status = 'active' AND admin_user_id IS NOT NULL
`
	rows, err := database.DB.Query(ctx, query, regionID)
	if err != nil {
		return nil, fmt.Errorf("查询区域管理员列表失败: %w", err)
	}
	defer rows.Close()

	ids := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err == nil && id != "" {
			ids = append(ids, id)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历区域管理员结果失败: %w", err)
	}
	return ids, nil
}

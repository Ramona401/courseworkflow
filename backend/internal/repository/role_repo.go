package repository

/*
 * role_repo.go — 角色权限数据访问层
 *
 * 对应数据库表：roles / role_permissions
 *
 * 核心函数：
 *   ListRoles             — 列表（含权限数+用户数统计）
 *   GetRoleByID           — 详情（含权限明细）
 *   GetRoleByCode         — 按 role_code 查询（用于唯一性校验）
 *   CreateRole            — 新建角色
 *   UpdateRole            — 编辑显示名/描述
 *   UpdateRoleStatus      — 启用/禁用
 *   DeleteRole            — 删除（业务层已校验 is_system 和 user_count）
 *   CountUsersByRoleCode  — 统计使用该角色码的用户数（删除前检查）
 *   CopyPermissions       — 从内置角色复制权限到新角色（事务内）
 *   GetRolePermissions    — 获取角色权限列表
 *   UpdateRolePermissions — 全量替换权限（事务：先删后插）
 */

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 角色列表 ====================

// ListRoles 查询所有角色，含权限数量和用户使用数量统计
func ListRoles(ctx context.Context) (*models.RoleListResponse, error) {
	// LEFT JOIN role_permissions 统计权限数
	// LEFT JOIN users 统计使用人数（users.role = roles.role_code）
	query := `
		SELECT
			r.id,
			r.role_code,
			r.display_name,
			COALESCE(r.description, '') AS description,
			r.base_role,
			r.is_system,
			r.status,
			TO_CHAR(r.created_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD HH24:MI:SS') AS created_at,
			TO_CHAR(r.updated_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD HH24:MI:SS') AS updated_at,
			COUNT(DISTINCT rp.id)::int AS permission_count,
			COUNT(DISTINCT u.id)::int  AS user_count
		FROM roles r
		LEFT JOIN role_permissions rp ON rp.role_id = r.id
		LEFT JOIN users u ON u.role = r.role_code AND u.status = 'active'
		GROUP BY r.id
		ORDER BY r.is_system DESC, r.created_at ASC
	`

	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询角色列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.RoleListItem
	for rows.Next() {
		item := &models.RoleListItem{}
		err := rows.Scan(
			&item.ID,
			&item.RoleCode,
			&item.DisplayName,
			&item.Description,
			&item.BaseRole,
			&item.IsSystem,
			&item.Status,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.PermissionCount,
			&item.UserCount,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描角色列表行失败: %w", err)
		}
		items = append(items, item)
	}
	if items == nil {
		items = []*models.RoleListItem{}
	}

	return &models.RoleListResponse{
		Roles: items,
		Total: len(items),
	}, nil
}

// ==================== 角色详情 ====================

// GetRoleByID 查询角色详情（含权限明细和用户数）
func GetRoleByID(ctx context.Context, roleID string) (*models.RoleDetailResponse, error) {
	// 查角色基础信息 + 用户数
	baseQuery := `
		SELECT
			r.id,
			r.role_code,
			r.display_name,
			COALESCE(r.description, '') AS description,
			r.base_role,
			r.is_system,
			r.status,
			r.created_by,
			TO_CHAR(r.created_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD HH24:MI:SS') AS created_at,
			TO_CHAR(r.updated_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD HH24:MI:SS') AS updated_at,
			COUNT(DISTINCT u.id)::int AS user_count
		FROM roles r
		LEFT JOIN users u ON u.role = r.role_code AND u.status = 'active'
		WHERE r.id = $1
		GROUP BY r.id
	`

	detail := &models.RoleDetailResponse{}
	err := database.DB.QueryRow(ctx, baseQuery, roleID).Scan(
		&detail.ID,
		&detail.RoleCode,
		&detail.DisplayName,
		&detail.Description,
		&detail.BaseRole,
		&detail.IsSystem,
		&detail.Status,
		&detail.CreatedBy,
		&detail.CreatedAt,
		&detail.UpdatedAt,
		&detail.UserCount,
	)
	if err != nil {
		return nil, fmt.Errorf("查询角色详情失败: %w", err)
	}

	// 查权限列表
	perms, err := GetRolePermissions(ctx, roleID)
	if err != nil {
		return nil, err
	}
	detail.Permissions = perms

	return detail, nil
}

// GetRoleByCode 按 role_code 查询角色（用于唯一性校验）
func GetRoleByCode(ctx context.Context, roleCode string) (*models.Role, error) {
	query := `
		SELECT id, role_code, display_name, COALESCE(description,''),
		       base_role, is_system, status
		FROM roles
		WHERE role_code = $1
	`
	role := &models.Role{}
	err := database.DB.QueryRow(ctx, query, roleCode).Scan(
		&role.ID,
		&role.RoleCode,
		&role.DisplayName,
		&role.Description,
		&role.BaseRole,
		&role.IsSystem,
		&role.Status,
	)
	if err != nil {
		return nil, err // pgx.ErrNoRows 由调用方判断
	}
	return role, nil
}

// ==================== 创建角色 ====================

// CreateRole 插入新角色，返回完整角色信息
func CreateRole(ctx context.Context, req *models.CreateRoleRequest, createdBy string) (*models.Role, error) {
	query := `
		INSERT INTO roles (role_code, display_name, description, base_role, is_system, status, created_by)
		VALUES ($1, $2, $3, $4, false, 'active', $5)
		RETURNING id, role_code, display_name, COALESCE(description,''),
		          base_role, is_system, status,
		          TO_CHAR(created_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD HH24:MI:SS'),
		          TO_CHAR(updated_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD HH24:MI:SS')
	`
	role := &models.Role{}
	err := database.DB.QueryRow(ctx, query,
		req.RoleCode,
		req.DisplayName,
		req.Description,
		req.BaseRole,
		createdBy,
	).Scan(
		&role.ID,
		&role.RoleCode,
		&role.DisplayName,
		&role.Description,
		&role.BaseRole,
		&role.IsSystem,
		&role.Status,
		&role.CreatedAt,
		&role.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("创建角色失败: %w", err)
	}
	return role, nil
}

// ==================== 编辑角色 ====================

// UpdateRole 更新角色显示名和描述（is_system 由业务层拦截）
func UpdateRole(ctx context.Context, roleID, displayName, description string) error {
	query := `
		UPDATE roles
		SET display_name = $1,
		    description  = $2,
		    updated_at   = NOW()
		WHERE id = $3 AND is_system = false
	`
	ct, err := database.DB.Exec(ctx, query, displayName, description, roleID)
	if err != nil {
		return fmt.Errorf("更新角色失败: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("角色不存在或系统内置角色不可编辑")
	}
	return nil
}

// ==================== 状态管理 ====================

// UpdateRoleStatus 启用/禁用角色（is_system 由业务层拦截）
func UpdateRoleStatus(ctx context.Context, roleID, status string) error {
	query := `
		UPDATE roles
		SET status     = $1,
		    updated_at = NOW()
		WHERE id = $2 AND is_system = false
	`
	ct, err := database.DB.Exec(ctx, query, status, roleID)
	if err != nil {
		return fmt.Errorf("更新角色状态失败: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("角色不存在或系统内置角色不可修改状态")
	}
	return nil
}

// ==================== 删除角色 ====================

// DeleteRole 删除角色（is_system 和 user_count 由业务层提前校验）
func DeleteRole(ctx context.Context, roleID string) error {
	// role_permissions 设有 ON DELETE CASCADE，无需手动删除
	query := `DELETE FROM roles WHERE id = $1 AND is_system = false`
	ct, err := database.DB.Exec(ctx, query, roleID)
	if err != nil {
		return fmt.Errorf("删除角色失败: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return fmt.Errorf("角色不存在或系统内置角色不可删除")
	}
	return nil
}

// CountUsersByRoleCode 统计使用指定角色码的用户数（用于删除前检查）
func CountUsersByRoleCode(ctx context.Context, roleCode string) (int, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE role = $1`,
		roleCode,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("统计角色用户数失败: %w", err)
	}
	return count, nil
}

// ==================== 权限列表 ====================

// GetRolePermissions 获取指定角色的所有权限
func GetRolePermissions(ctx context.Context, roleID string) ([]*models.RolePermission, error) {
	query := `
		SELECT
			id,
			role_id,
			permission_code,
			resource,
			action,
			TO_CHAR(created_at AT TIME ZONE 'Asia/Shanghai', 'YYYY-MM-DD HH24:MI:SS') AS created_at
		FROM role_permissions
		WHERE role_id = $1
		ORDER BY resource, action
	`
	rows, err := database.DB.Query(ctx, query, roleID)
	if err != nil {
		return nil, fmt.Errorf("查询角色权限失败: %w", err)
	}
	defer rows.Close()

	var perms []*models.RolePermission
	for rows.Next() {
		p := &models.RolePermission{}
		if err := rows.Scan(&p.ID, &p.RoleID, &p.PermissionCode, &p.Resource, &p.Action, &p.CreatedAt); err != nil {
			return nil, fmt.Errorf("扫描权限行失败: %w", err)
		}
		perms = append(perms, p)
	}
	if perms == nil {
		perms = []*models.RolePermission{}
	}
	return perms, nil
}

// ==================== 权限复制（新建自定义角色时使用）====================

// CopyPermissionsFromBaseRole 将内置角色的权限复制到新角色（事务内执行）
// baseRoleCode: 内置角色码（admin/senior_operator/operator/viewer）
// newRoleID:    新建角色的 UUID
func CopyPermissionsFromBaseRole(ctx context.Context, baseRoleCode, newRoleID string) error {
	// 先找到内置角色的 ID
	var baseRoleID string
	err := database.DB.QueryRow(ctx,
		`SELECT id FROM roles WHERE role_code = $1 AND is_system = true`,
		baseRoleCode,
	).Scan(&baseRoleID)
	if err != nil {
		return fmt.Errorf("找不到内置角色 %s: %w", baseRoleCode, err)
	}

	// 批量复制权限（permission_code 唯一约束冲突则跳过）
	insertQuery := `
		INSERT INTO role_permissions (role_id, permission_code, resource, action)
		SELECT $1, permission_code, resource, action
		FROM role_permissions
		WHERE role_id = $2
		ON CONFLICT (role_id, permission_code) DO NOTHING
	`
	_, err = database.DB.Exec(ctx, insertQuery, newRoleID, baseRoleID)
	if err != nil {
		return fmt.Errorf("复制权限失败: %w", err)
	}
	return nil
}

// ==================== 权限全量替换 ====================

// UpdateRolePermissions 全量替换角色权限（事务：先删除旧权限，再插入新权限）
// is_system 由业务层拦截，此处不再重复判断
func UpdateRolePermissions(ctx context.Context, roleID string, perms []models.PermissionItem) error {
	tx, err := database.DB.Begin(ctx)
	if err != nil {
		return fmt.Errorf("开启事务失败: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// 删除旧权限
	if _, err := tx.Exec(ctx, `DELETE FROM role_permissions WHERE role_id = $1`, roleID); err != nil {
		return fmt.Errorf("清除旧权限失败: %w", err)
	}

	// 插入新权限
	for _, p := range perms {
		_, err := tx.Exec(ctx,
			`INSERT INTO role_permissions (role_id, permission_code, resource, action)
			 VALUES ($1, $2, $3, $4)
			 ON CONFLICT (role_id, permission_code) DO NOTHING`,
			roleID, p.PermissionCode, p.Resource, p.Action,
		)
		if err != nil {
			return fmt.Errorf("插入权限 %s 失败: %w", p.PermissionCode, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("提交权限更新事务失败: %w", err)
	}
	return nil
}

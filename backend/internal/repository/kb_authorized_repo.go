package repository

// kb_authorized_repo.go — 知识库压缩子系统访问白名单数据访问层
//
// 对应表：kb_authorized_users（user_id 主键 + granted_by + note + created_at）
// 守卫语义：admin 角色恒通过（在 service/middleware 层判定）；
//          本表内的 user_id 通过；其余拒。本 repo 只负责名单的增删查与"是否在名单内"判定。
//
// 风格对齐 organization_admin_repo.go：database.DB、ctx、LEFT JOIN users 填用户名、
// COALESCE 处理可空 granted_by。

import (
	"context"
	"fmt"

	"tedna/internal/database"
	"tedna/internal/models"
)

// IsKBAuthorized 判断某用户是否在知识库压缩白名单内（admin 恒通过的逻辑在 service 层，本函数只查表）
func IsKBAuthorized(ctx context.Context, userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	var exists bool
	err := database.DB.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM kb_authorized_users WHERE user_id = $1)`,
		userID,
	).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("查询知识库白名单失败: %w", err)
	}
	return exists, nil
}

// AddKBAuthorized 添加白名单成员（幂等：ON CONFLICT 主键不报错）
// grantedBy 授权操作人（可空）；note 备注（可空）
func AddKBAuthorized(ctx context.Context, userID string, grantedBy string, note string) error {
	if userID == "" {
		return fmt.Errorf("userID 为空")
	}
	var grantedByArg interface{}
	if grantedBy == "" {
		grantedByArg = nil
	} else {
		grantedByArg = grantedBy
	}
	_, err := database.DB.Exec(ctx, `
		INSERT INTO kb_authorized_users (user_id, granted_by, note)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id) DO UPDATE SET note = EXCLUDED.note
	`, userID, grantedByArg, note)
	if err != nil {
		return fmt.Errorf("添加知识库白名单成员失败: %w", err)
	}
	return nil
}

// RemoveKBAuthorized 移除白名单成员（找不到返回 ErrMemberNotFound）
func RemoveKBAuthorized(ctx context.Context, userID string) error {
	result, err := database.DB.Exec(ctx,
		`DELETE FROM kb_authorized_users WHERE user_id = $1`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("移除知识库白名单成员失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

// ListKBAuthorized 列出全部白名单成员（JOIN users 填用户名/显示名/角色，再 JOIN 授权人显示名）
func ListKBAuthorized(ctx context.Context) ([]*models.KBAuthorizedUserItem, error) {
	query := `
		SELECT k.user_id, u.username, COALESCE(u.display_name,''), u.role,
		       COALESCE(g.display_name, COALESCE(g.username,''), ''),
		       COALESCE(k.note,''), k.created_at
		FROM kb_authorized_users k
		JOIN users u ON u.id = k.user_id
		LEFT JOIN users g ON g.id = k.granted_by
		ORDER BY k.created_at DESC
	`
	rows, err := database.DB.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("查询知识库白名单列表失败: %w", err)
	}
	defer rows.Close()

	items := []*models.KBAuthorizedUserItem{}
	for rows.Next() {
		item := &models.KBAuthorizedUserItem{}
		if err := rows.Scan(
			&item.UserID, &item.Username, &item.DisplayName, &item.Role,
			&item.GrantedBy, &item.Note, &item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("扫描知识库白名单行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

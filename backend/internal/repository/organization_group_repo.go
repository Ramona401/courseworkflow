package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"tedna/internal/database"
	"tedna/internal/models"
)

// ==================== 教研组数据访问 ====================
//
// 本文件从 organization_repo.go 拆出（超 600 行红线拆分），承载教研组(teaching_groups)
// 与教研组成员(teaching_group_members)的全部数据访问，含组长/骨干权限判定。
// 与 organization_repo.go 同 package，错误常量(ErrGroupNotFound 等)仍定义在 organization_repo.go，
// 本文件直接引用无需重复定义。纯位置搬迁，SQL 与逻辑逐字保留零变更。

// ==================== 教研组 CRUD ====================

func CreateTeachingGroup(ctx context.Context, tg *models.TeachingGroup) error {
	query := `
		INSERT INTO teaching_groups (name, school_id, subject, grade_range, lead_user_id, description, settings, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`
	settings := tg.Settings
	if settings == "" {
		settings = "{}"
	}
	err := database.DB.QueryRow(ctx, query,
		tg.Name, tg.SchoolID, tg.Subject, tg.GradeRange,
		tg.LeadUserID, tg.Description, settings, "active",
	).Scan(&tg.ID, &tg.CreatedAt, &tg.UpdatedAt)
	if err != nil {
		return fmt.Errorf("创建教研组失败: %w", err)
	}
	return nil
}

func GetTeachingGroupByID(ctx context.Context, id string) (*models.TeachingGroup, error) {
	tg := &models.TeachingGroup{}
	query := `
		SELECT id, name, school_id, subject, grade_range, lead_user_id,
		       description, settings, status, created_at, updated_at
		FROM teaching_groups WHERE id = $1
	`
	err := database.DB.QueryRow(ctx, query, id).Scan(
		&tg.ID, &tg.Name, &tg.SchoolID, &tg.Subject, &tg.GradeRange,
		&tg.LeadUserID, &tg.Description, &tg.Settings, &tg.Status,
		&tg.CreatedAt, &tg.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrGroupNotFound
		}
		return nil, fmt.Errorf("查询教研组失败: %w", err)
	}
	return tg, nil
}

// ListTeachingGroups 获取教研组列表
// v109改动：lead_user_names 从成员角色表聚合所有 role='lead' 的成员名称（逗号分隔）
func ListTeachingGroups(ctx context.Context, schoolID string) ([]*models.TeachingGroupListItem, error) {
	query := `
		SELECT tg.id, tg.name, tg.school_id, tg.subject, tg.grade_range,
		       tg.lead_user_id, tg.status, tg.created_at,
		       COALESCE(o.name, '') AS school_name,
		       COALESCE(u.display_name, '') AS lead_user_name,
		       (SELECT COUNT(*) FROM teaching_group_members tgm WHERE tgm.group_id = tg.id) AS member_count,
		       COALESCE(
		         (SELECT string_agg(u2.display_name, '、' ORDER BY tgm2.joined_at)
		          FROM teaching_group_members tgm2
		          JOIN users u2 ON u2.id = tgm2.user_id
		          WHERE tgm2.group_id = tg.id AND tgm2.role = 'lead'),
		         ''
		       ) AS lead_user_names
		FROM teaching_groups tg
		LEFT JOIN organizations o ON o.id = tg.school_id
		LEFT JOIN users u ON u.id = tg.lead_user_id
		WHERE 1=1
	`
	args := []interface{}{}
	argIdx := 1

	if schoolID != "" {
		query += fmt.Sprintf(" AND tg.school_id = $%d", argIdx)
		args = append(args, schoolID)
	}
	query += " ORDER BY tg.name ASC"

	rows, err := database.DB.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询教研组列表失败: %w", err)
	}
	defer rows.Close()

	var items []*models.TeachingGroupListItem
	for rows.Next() {
		item := &models.TeachingGroupListItem{}
		err := rows.Scan(
			&item.ID, &item.Name, &item.SchoolID, &item.Subject, &item.GradeRange,
			&item.LeadUserID, &item.Status, &item.CreatedAt,
			&item.SchoolName, &item.LeadUserName, &item.MemberCount,
			&item.LeadUserNames,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描教研组行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func UpdateTeachingGroup(ctx context.Context, id string, req *models.UpdateTeachingGroupRequest) error {
	query := `
		UPDATE teaching_groups
		SET name = $1, subject = $2, grade_range = $3,
		    description = $4, settings = $5, status = $6, updated_at = $7
		WHERE id = $8
	`
	settings := req.Settings
	if settings == "" {
		settings = "{}"
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	now := time.Now()
	result, err := database.DB.Exec(ctx, query,
		req.Name, req.Subject, req.GradeRange,
		req.Description, settings, status, now, id,
	)
	if err != nil {
		return fmt.Errorf("更新教研组失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

func DeleteTeachingGroup(ctx context.Context, id string) error {
	result, err := database.DB.Exec(ctx, `DELETE FROM teaching_groups WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("删除教研组失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrGroupNotFound
	}
	return nil
}

func CheckGroupNameExists(ctx context.Context, schoolID string, name string, excludeID string) (bool, error) {
	query := `SELECT COUNT(*) FROM teaching_groups WHERE school_id = $1 AND name = $2`
	args := []interface{}{schoolID, name}
	if excludeID != "" {
		query += " AND id != $3"
		args = append(args, excludeID)
	}
	var count int
	err := database.DB.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查教研组名称失败: %w", err)
	}
	return count > 0, nil
}

// ==================== 教研组成员 CRUD ====================

func AddGroupMember(ctx context.Context, member *models.TeachingGroupMember) error {
	query := `
		INSERT INTO teaching_group_members (group_id, user_id, role)
		VALUES ($1, $2, $3)
		RETURNING id, joined_at
	`
	role := member.Role
	if role == "" {
		role = "member"
	}
	err := database.DB.QueryRow(ctx, query,
		member.GroupID, member.UserID, role,
	).Scan(&member.ID, &member.JoinedAt)
	if err != nil {
		return fmt.Errorf("添加教研组成员失败: %w", err)
	}
	return nil
}

func RemoveGroupMember(ctx context.Context, groupID string, userID string) error {
	result, err := database.DB.Exec(ctx,
		`DELETE FROM teaching_group_members WHERE group_id = $1 AND user_id = $2`,
		groupID, userID,
	)
	if err != nil {
		return fmt.Errorf("移除教研组成员失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func ListGroupMembers(ctx context.Context, groupID string) ([]*models.GroupMemberItem, error) {
	query := `
		SELECT tgm.id, tgm.user_id, u.username, u.display_name, tgm.role, tgm.joined_at
		FROM teaching_group_members tgm
		JOIN users u ON u.id = tgm.user_id
		WHERE tgm.group_id = $1
		ORDER BY
		  CASE tgm.role WHEN 'lead' THEN 0 WHEN 'backbone' THEN 1 ELSE 2 END,
		  tgm.joined_at ASC
	`
	rows, err := database.DB.Query(ctx, query, groupID)
	if err != nil {
		return nil, fmt.Errorf("查询教研组成员失败: %w", err)
	}
	defer rows.Close()

	var items []*models.GroupMemberItem
	for rows.Next() {
		item := &models.GroupMemberItem{}
		err := rows.Scan(
			&item.ID, &item.UserID, &item.Username, &item.DisplayName,
			&item.Role, &item.JoinedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描成员行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

func UpdateGroupMemberRole(ctx context.Context, groupID string, userID string, role string) error {
	result, err := database.DB.Exec(ctx,
		`UPDATE teaching_group_members SET role = $1 WHERE group_id = $2 AND user_id = $3`,
		role, groupID, userID,
	)
	if err != nil {
		return fmt.Errorf("更新成员角色失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrMemberNotFound
	}
	return nil
}

func CheckMemberExists(ctx context.Context, groupID string, userID string) (bool, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM teaching_group_members WHERE group_id = $1 AND user_id = $2`,
		groupID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查成员存在性失败: %w", err)
	}
	return count > 0, nil
}

func GetUserTeachingGroups(ctx context.Context, userID string) ([]*models.TeachingGroupListItem, error) {
	query := `
		SELECT tg.id, tg.name, tg.school_id, tg.subject, tg.grade_range,
		       tg.lead_user_id, tg.status, tg.created_at,
		       COALESCE(o.name, '') AS school_name,
		       COALESCE(u.display_name, '') AS lead_user_name,
		       (SELECT COUNT(*) FROM teaching_group_members tgm2 WHERE tgm2.group_id = tg.id) AS member_count,
		       COALESCE(
		         (SELECT string_agg(u2.display_name, '、' ORDER BY tgm3.joined_at)
		          FROM teaching_group_members tgm3
		          JOIN users u2 ON u2.id = tgm3.user_id
		          WHERE tgm3.group_id = tg.id AND tgm3.role = 'lead'),
		         ''
		       ) AS lead_user_names
		FROM teaching_group_members tgm
		JOIN teaching_groups tg ON tg.id = tgm.group_id
		LEFT JOIN organizations o ON o.id = tg.school_id
		LEFT JOIN users u ON u.id = tg.lead_user_id
		WHERE tgm.user_id = $1 AND tg.status = 'active'
		ORDER BY tg.name ASC
	`
	rows, err := database.DB.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户教研组失败: %w", err)
	}
	defer rows.Close()

	var items []*models.TeachingGroupListItem
	for rows.Next() {
		item := &models.TeachingGroupListItem{}
		err := rows.Scan(
			&item.ID, &item.Name, &item.SchoolID, &item.Subject, &item.GradeRange,
			&item.LeadUserID, &item.Status, &item.CreatedAt,
			&item.SchoolName, &item.LeadUserName, &item.MemberCount,
			&item.LeadUserNames,
		)
		if err != nil {
			return nil, fmt.Errorf("扫描教研组行失败: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// ==================== 教研组组长/骨干权限判定 ====================

// IsGroupLead 检查用户是否是某教研组的组长
// v109改动：从成员角色表查 role='lead'（支持多组长），不再只查 lead_user_id 字段
func IsGroupLead(ctx context.Context, groupID string, userID string) (bool, error) {
	var count int
	err := database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM teaching_group_members
		 WHERE group_id = $1 AND user_id = $2 AND role = 'lead'`,
		groupID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查组长权限失败: %w", err)
	}
	// 兼容旧数据：同时检查 teaching_groups.lead_user_id
	if count == 0 {
		err = database.DB.QueryRow(ctx,
			`SELECT COUNT(*) FROM teaching_groups WHERE id = $1 AND lead_user_id = $2`,
			groupID, userID,
		).Scan(&count)
		if err != nil {
			return false, fmt.Errorf("检查组长权限(兼容)失败: %w", err)
		}
	}
	return count > 0, nil
}

// IsGroupLeadOrBackbone 检查用户是否有评审权限（组长或骨干）
func IsGroupLeadOrBackbone(ctx context.Context, groupID string, userID string) (bool, error) {
	isLead, err := IsGroupLead(ctx, groupID, userID)
	if err != nil {
		return false, err
	}
	if isLead {
		return true, nil
	}
	var count int
	err = database.DB.QueryRow(ctx,
		`SELECT COUNT(*) FROM teaching_group_members
		 WHERE group_id = $1 AND user_id = $2 AND role = 'backbone'`,
		groupID, userID,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查骨干权限失败: %w", err)
	}
	return count > 0, nil
}

// GetGroupLeadNames 获取教研组所有组长的名称列表（逗号分隔）
func GetGroupLeadNames(ctx context.Context, groupID string) (string, error) {
	var names []string
	rows, err := database.DB.Query(ctx,
		`SELECT u.display_name FROM teaching_group_members tgm
		 JOIN users u ON u.id = tgm.user_id
		 WHERE tgm.group_id = $1 AND tgm.role = 'lead'
		 ORDER BY tgm.joined_at`,
		groupID,
	)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			names = append(names, name)
		}
	}
	return strings.Join(names, "、"), nil
}

// ==================== v139.1：查询用户管理的教研组 ====================

// ListMyLeadOrBackboneGroups 查询当前用户在其中担任 lead 或 backbone 的所有教研组
//
// 用途:发布模板时,让用户从"自己管理的教研组"下拉中选择,而不是手填 UUID
// 返回:每个教研组的 ID/名称/所属学校名/当前用户在此组的角色
// 排序:lead 排前,然后按 joined_at 升序
//
// 复用 organizations + teaching_groups + teaching_group_members 三表 JOIN
func ListMyLeadOrBackboneGroups(ctx context.Context, userID string) ([]models.PublishTargetGroup, error) {
	query := `
		SELECT tg.id, tg.name, COALESCE(o.name, ''), tgm.role
		FROM teaching_group_members tgm
		JOIN teaching_groups tg ON tg.id = tgm.group_id
		LEFT JOIN organizations o ON o.id = tg.school_id
		WHERE tgm.user_id = $1
		  AND tgm.role IN ('lead', 'backbone')
		  AND tg.status = 'active'
		ORDER BY
		  CASE tgm.role WHEN 'lead' THEN 0 ELSE 1 END,
		  tgm.joined_at ASC
	`
	rows, err := database.DB.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("查询用户管理的教研组失败: %w", err)
	}
	defer rows.Close()

	var groups []models.PublishTargetGroup
	for rows.Next() {
		g := models.PublishTargetGroup{}
		if err := rows.Scan(&g.ID, &g.Name, &g.SchoolName, &g.Role); err != nil {
			return nil, fmt.Errorf("扫描教研组行失败: %w", err)
		}
		groups = append(groups, g)
	}
	if groups == nil {
		groups = []models.PublishTargetGroup{}
	}
	return groups, nil
}

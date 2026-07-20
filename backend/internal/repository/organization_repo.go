package repository

// organization_repo.go
//
// 本文件只承载：
//   1. 组织与教研组共享错误常量；
//   2. 门户板块配置；
//   3. 学校直接成员；
//   4. 组织Logo与用户组织品牌解析。
//
// 组织CRUD、组织列表、区域学校递归查询已拆至
// organization_crud_repo.go，避免本文件继续超过600行。

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// 共享错误常量。
//
// ErrGroupNotFound、ErrMemberExists、ErrMemberNotFound同时供
// organization_group_repo.go 使用，因此继续保留在本文件。
var (
	ErrOrgNotFound     = errors.New("组织不存在")
	ErrOrgNameExists   = errors.New("同类型下组织名称已存在")
	ErrGroupNotFound   = errors.New("教研组不存在")
	ErrGroupNameExists = errors.New("该学校下教研组名称已存在")
	ErrMemberExists    = errors.New("该用户已是教研组成员")
	ErrMemberNotFound  = errors.New("教研组成员不存在")
)

// ==================== 门户板块配置 ====================

// parsePortalModulesFromSettings 从组织settings解析portal_modules。
//
// 容错规则：
//   - settings为空、非法JSON或无portal_modules时，全部板块开启；
//   - 缺失的板块键按开启处理；
//   - 只有显式false的板块会被关闭。
func parsePortalModulesFromSettings(
	settings string,
) map[string]bool {
	result := models.DefaultPortalModules()

	settings = strings.TrimSpace(settings)
	if settings == "" || settings == "{}" {
		return result
	}

	var raw struct {
		PortalModules map[string]bool `json:"portal_modules"`
	}

	if err := json.Unmarshal([]byte(settings), &raw); err != nil {
		return result
	}
	if raw.PortalModules == nil {
		return result
	}

	for _, key := range models.AllPortalModules {
		if value, exists := raw.PortalModules[key]; exists {
			result[key] = value
		}
	}

	return result
}

// GetUserPortalModules 获取用户所属学校配置的门户板块。
//
// 查找顺序：
//  1. school_members直接校籍；
//  2. teaching_group_members反查学校；
//  3. 无学校时返回全部开启。
func GetUserPortalModules(
	ctx context.Context,
	userID string,
) map[string]bool {
	var settings string

	err := database.DB.QueryRow(ctx, `
		SELECT COALESCE(o.settings, '{}')
		FROM school_members sm
		JOIN organizations o
		  ON o.id = sm.school_id
		WHERE sm.user_id = $1
		  AND o.status = 'active'
		LIMIT 1
	`, userID).Scan(&settings)
	if err != nil {
		err = database.DB.QueryRow(ctx, `
			SELECT COALESCE(o.settings, '{}')
			FROM teaching_group_members tgm
			JOIN teaching_groups tg
			  ON tg.id = tgm.group_id
			JOIN organizations o
			  ON o.id = tg.school_id
			WHERE tgm.user_id = $1
			  AND o.status = 'active'
			LIMIT 1
		`, userID).Scan(&settings)
		if err != nil {
			return models.DefaultPortalModules()
		}
	}

	return parsePortalModulesFromSettings(settings)
}

// ==================== 学校直接成员 ====================

// AddSchoolMember 将用户加入学校直接成员名单。
//
// 使用ON CONFLICT保证幂等。
func AddSchoolMember(
	ctx context.Context,
	schoolID string,
	userID string,
	source string,
) error {
	if schoolID == "" || userID == "" {
		return fmt.Errorf("schoolID 或 userID 为空")
	}
	if source == "" {
		source = "manual"
	}

	_, err := database.DB.Exec(ctx, `
		INSERT INTO school_members (
			school_id,
			user_id,
			joined_at,
			source
		)
		VALUES ($1, $2, now(), $3)
		ON CONFLICT (school_id, user_id) DO NOTHING
	`, schoolID, userID, source)
	if err != nil {
		return fmt.Errorf("加入学校成员失败: %w", err)
	}

	return nil
}

// RemoveSchoolMember 从学校直接成员名单移除用户。
func RemoveSchoolMember(
	ctx context.Context,
	schoolID string,
	userID string,
) error {
	_, err := database.DB.Exec(
		ctx,
		`DELETE FROM school_members
		 WHERE school_id = $1
		   AND user_id = $2`,
		schoolID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("移除学校成员失败: %w", err)
	}

	return nil
}

// IsUserInSchool 宽松检查用户是否属于指定学校。
//
// 先查school_members，再以教研组成员身份兜底。
// 本函数适合管理操作放行，不适合严格数据隔离。
func IsUserInSchool(
	ctx context.Context,
	userID string,
	schoolID string,
) (bool, error) {
	var count int

	err := database.DB.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM school_members
		WHERE user_id = $1
		  AND school_id = $2
	`, userID, schoolID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查学校直接成员失败: %w", err)
	}
	if count > 0 {
		return true, nil
	}

	err = database.DB.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM teaching_group_members tgm
		JOIN teaching_groups tg
		  ON tg.id = tgm.group_id
		WHERE tgm.user_id = $1
		  AND tg.school_id = $2
	`, userID, schoolID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查用户学校归属(教研组兜底)失败: %w", err)
	}

	return count > 0, nil
}

// IsUserInSchoolStrict 严格检查用户是否属于指定学校。
//
// 只认school_members，不使用教研组兜底，用于数据隔离。
func IsUserInSchoolStrict(
	ctx context.Context,
	userID string,
	schoolID string,
) (bool, error) {
	if userID == "" || schoolID == "" {
		return false, nil
	}

	var count int
	err := database.DB.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM school_members
		WHERE user_id = $1
		  AND school_id = $2
	`, userID, schoolID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("严格检查学校直接成员失败: %w", err)
	}

	return count > 0, nil
}

// ListSchoolMemberIDs 返回某学校全部直接成员ID。
func ListSchoolMemberIDs(
	ctx context.Context,
	schoolID string,
) ([]string, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT user_id
		FROM school_members
		WHERE school_id = $1
	`, schoolID)
	if err != nil {
		return nil, fmt.Errorf("查询学校成员ID列表失败: %w", err)
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("扫描学校成员ID失败: %w", err)
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历学校成员ID失败: %w", err)
	}

	return ids, nil
}

// IsUserInSchoolByGroup 仅通过教研组检查学校归属。
//
// Deprecated: 新代码优先使用IsUserInSchool或IsUserInSchoolStrict。
func IsUserInSchoolByGroup(
	ctx context.Context,
	userID string,
	schoolID string,
) (bool, error) {
	var count int

	err := database.DB.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM teaching_group_members tgm
		JOIN teaching_groups tg
		  ON tg.id = tgm.group_id
		WHERE tgm.user_id = $1
		  AND tg.school_id = $2
	`, userID, schoolID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("检查用户学校归属失败: %w", err)
	}

	return count > 0, nil
}

// ==================== 组织Logo ====================

// UpdateOrganizationLogo 更新组织Logo URL。
func UpdateOrganizationLogo(
	ctx context.Context,
	id string,
	logoURL string,
) error {
	result, err := database.DB.Exec(ctx, `
		UPDATE organizations
		SET
			logo_url = $1,
			updated_at = $2
		WHERE id = $3
	`, logoURL, time.Now(), id)
	if err != nil {
		return fmt.Errorf("更新组织Logo失败: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrOrgNotFound
	}

	return nil
}

// GetUserOrgLogo 获取用户所属学校Logo和名称。
//
// 查找顺序：
//  1. school_members直接校籍；
//  2. 教研组成员反查学校；
//  3. 学校无Logo时继承上级区域Logo。
func GetUserOrgLogo(
	ctx context.Context,
	userID string,
) (string, string) {
	var schoolID string
	var schoolName string
	var schoolLogoURL string
	var parentID *string

	err := database.DB.QueryRow(ctx, `
		SELECT
			o.id,
			o.name,
			COALESCE(o.logo_url, ''),
			o.parent_id
		FROM school_members sm
		JOIN organizations o
		  ON o.id = sm.school_id
		WHERE sm.user_id = $1
		  AND o.status = 'active'
		LIMIT 1
	`, userID).Scan(
		&schoolID,
		&schoolName,
		&schoolLogoURL,
		&parentID,
	)
	if err != nil {
		err = database.DB.QueryRow(ctx, `
			SELECT
				o.id,
				o.name,
				COALESCE(o.logo_url, ''),
				o.parent_id
			FROM teaching_group_members tgm
			JOIN teaching_groups tg
			  ON tg.id = tgm.group_id
			JOIN organizations o
			  ON o.id = tg.school_id
			WHERE tgm.user_id = $1
			  AND o.status = 'active'
			LIMIT 1
		`, userID).Scan(
			&schoolID,
			&schoolName,
			&schoolLogoURL,
			&parentID,
		)
		if err != nil {
			return "", ""
		}
	}

	if schoolLogoURL != "" {
		return schoolLogoURL, schoolName
	}

	if parentID != nil && *parentID != "" {
		var regionLogoURL string
		var regionName string

		err = database.DB.QueryRow(ctx, `
			SELECT
				COALESCE(logo_url, ''),
				name
			FROM organizations
			WHERE id = $1
			  AND status = 'active'
		`, *parentID).Scan(
			&regionLogoURL,
			&regionName,
		)
		if err == nil && regionLogoURL != "" {
			return regionLogoURL, schoolName
		}
	}

	return "", schoolName
}

package repository

// organization_admin_domain_repo.go
//
// 本文件专门负责区域管理员固定教育域的数据访问，避免继续扩大
// organization_admin_repo.go。现有组织管理员函数暂时保留不动，
// 固定教育域任命入口使用本文件提供的严格版本。
//
// 主要职责：
//   1. 任命区域管理员时显式写入education_domain；
//   2. 学校管理员任命始终把education_domain写为NULL；
//   3. 列出管理员时返回education_domain；
//   4. 查询某区域实际存在的学校教育域；
//   5. 查询同一用户其它区域任命的教育域状态，供Service阻止跨域任命。
//
// 测试设计：
//   数据库写入前的“角色类型→教育域参数”转换被提取为纯函数，
//   使三个合法教学域、非法教育域和学校管理员NULL规则可以在不连接数据库的
//   情况下进行稳定的自动化测试。该提取不改变正式SQL和业务语义。

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"tedna/internal/database"
	"tedna/internal/models"
)

// UserRegionAdminEducationDomainState 描述一个用户其它有效区域任命的教育域状态。
//
// Domains只包含已经正确配置的具体教学教育域，并完成去重和稳定排序。
// HasUnconfigured表示存在空值、非法值，或者存在只有organizations.admin_user_id
// 单字段、但没有organization_admins教育域记录的历史任命。
type UserRegionAdminEducationDomainState struct {
	Domains         []string
	HasUnconfigured bool
}

// normalizeOrgAdminEducationDomainForWrite 将管理员类型和请求教育域转换为数据库写入参数。
//
// 返回规则：
//   - region_admin：返回标准化后的k12、vocational或adult字符串；
//   - school_admin：返回nil，确保数据库字段写为NULL；
//   - 非法角色或非法区域教育域：返回错误。
//
// 本函数只做确定性的写入参数规范化，不查询数据库、不判断跨区域冲突。
// 跨区域冲突、历史空值和区域学校类型检查仍由Service统一完成。
func normalizeOrgAdminEducationDomainForWrite(
	roleType string,
	educationDomain string,
) (interface{}, error) {
	switch roleType {
	case models.OrgAdminRoleRegion:
		normalized := strings.ToLower(strings.TrimSpace(educationDomain))
		if !models.IsTeachingEducationDomain(normalized) {
			return nil, fmt.Errorf(
				"区域管理员教育域无效: %s",
				educationDomain,
			)
		}
		return normalized, nil

	case models.OrgAdminRoleSchool:
		// 学校管理员的教育域由学校组织继承，
		// 任命记录本身不得重复保存任何教育域值。
		return nil, nil

	default:
		return nil, fmt.Errorf("无效的管理员类型: %s", roleType)
	}
}

// AddOrgAdminWithEducationDomain 任命组织管理员并显式写入教育域。
//
// 规则：
//   - region_admin必须传入k12/vocational/adult之一；
//   - school_admin不保存本字段，始终写NULL，教育域由学校继承；
//   - 同组织同用户已存在时执行UPSERT，允许给存量区域任命补齐教育域；
//   - 用户跨域冲突不在Repository判断，由Service结合全部任命统一校验。
func AddOrgAdminWithEducationDomain(
	ctx context.Context,
	orgID string,
	userID string,
	roleType string,
	educationDomain string,
	createdBy string,
) error {
	if strings.TrimSpace(orgID) == "" ||
		strings.TrimSpace(userID) == "" {
		return fmt.Errorf("orgID或userID为空")
	}

	domainArg, err := normalizeOrgAdminEducationDomainForWrite(
		roleType,
		educationDomain,
	)
	if err != nil {
		return err
	}

	var createdByArg interface{}
	if strings.TrimSpace(createdBy) == "" {
		createdByArg = nil
	} else {
		createdByArg = createdBy
	}

	_, err = database.DB.Exec(ctx, `
		INSERT INTO organization_admins (
			org_id,
			user_id,
			role_type,
			education_domain,
			created_by
		)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (org_id, user_id) DO UPDATE
		SET role_type = EXCLUDED.role_type,
		    education_domain = EXCLUDED.education_domain,
		    created_by = EXCLUDED.created_by
	`, orgID, userID, roleType, domainArg, createdByArg)
	if err != nil {
		return fmt.Errorf(
			"任命组织管理员并写入教育域失败: %w",
			err,
		)
	}

	return nil
}

// ListOrgAdminsWithEducationDomain 列出某组织全部管理员，并返回教育域。
//
// 学校管理员和存量未配置区域任命返回空字符串。
// Service和前端据role_type区分这两种语义。
func ListOrgAdminsWithEducationDomain(
	ctx context.Context,
	orgID string,
) ([]*models.OrganizationAdminItem, error) {
	rows, err := database.DB.Query(ctx, `
		SELECT
			oa.org_id,
			oa.user_id,
			u.username,
			u.display_name,
			oa.role_type,
			COALESCE(oa.education_domain, ''),
			COALESCE(oa.created_by::text, ''),
			oa.created_at
		FROM organization_admins oa
		JOIN users u ON u.id = oa.user_id
		WHERE oa.org_id = $1
		ORDER BY
			oa.role_type,
			oa.education_domain NULLS LAST,
			oa.created_at
	`, orgID)
	if err != nil {
		return nil, fmt.Errorf(
			"查询组织管理员教育域列表失败: %w",
			err,
		)
	}
	defer rows.Close()

	items := []*models.OrganizationAdminItem{}
	for rows.Next() {
		item := &models.OrganizationAdminItem{}
		if err := rows.Scan(
			&item.OrgID,
			&item.UserID,
			&item.Username,
			&item.DisplayName,
			&item.RoleType,
			&item.EducationDomain,
			&item.CreatedBy,
			&item.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描组织管理员教育域列表失败: %w",
				err,
			)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历组织管理员教育域列表失败: %w",
			err,
		)
	}

	return items, nil
}

// ListRegionSchoolEducationDomains 返回区域树下实际存在的有效学校教育域。
//
// 仅统计：
//   - 指定区域的递归下级；
//   - active学校；
//   - education_domain为k12/vocational/adult。
//
// 返回结果去重并稳定排序。没有合法学校时返回空切片，绝不默认K12。
func ListRegionSchoolEducationDomains(
	ctx context.Context,
	regionID string,
) ([]string, error) {
	if strings.TrimSpace(regionID) == "" {
		return []string{}, nil
	}

	rows, err := database.DB.Query(ctx, `
		WITH RECURSIVE org_tree AS (
			SELECT
				id,
				type,
				parent_id,
				status,
				education_domain,
				0 AS depth
			FROM organizations
			WHERE id = $1
			  AND type = 'region'
			  AND status = 'active'

			UNION ALL

			SELECT
				o.id,
				o.type,
				o.parent_id,
				o.status,
				o.education_domain,
				tree.depth + 1
			FROM organizations o
			JOIN org_tree tree ON o.parent_id = tree.id
		)
		SELECT DISTINCT LOWER(BTRIM(education_domain))
		FROM org_tree
		WHERE depth > 0
		  AND type = 'school'
		  AND status = 'active'
		  AND LOWER(BTRIM(education_domain))
		      IN ('k12', 'vocational', 'adult')
		ORDER BY 1
	`, regionID)
	if err != nil {
		return nil, fmt.Errorf(
			"查询区域实际学校教育域失败: %w",
			err,
		)
	}
	defer rows.Close()

	domains := []string{}
	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, fmt.Errorf(
				"扫描区域学校教育域失败: %w",
				err,
			)
		}
		domains = append(domains, domain)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历区域学校教育域失败: %w",
			err,
		)
	}

	return domains, nil
}

// GetUserRegionAdminEducationDomainState 查询用户除指定区域外的其它有效区域任命。
//
// 同时覆盖两种历史来源：
//  1. organization_admins多管理员任命；
//  2. organizations.admin_user_id区域主管理员单字段。
//
// excludeOrgID通常传本次正在任命的区域，使该区域的存量记录可以通过
// UPSERT补齐教育域，同时仍检查用户在其它区域的教育域是否冲突。
func GetUserRegionAdminEducationDomainState(
	ctx context.Context,
	userID string,
	excludeOrgID string,
) (*UserRegionAdminEducationDomainState, error) {
	state := &UserRegionAdminEducationDomainState{
		Domains: []string{},
	}

	if strings.TrimSpace(userID) == "" {
		return state, nil
	}

	rows, err := database.DB.Query(ctx, `
		WITH effective_appointments AS (
			SELECT
				oa.org_id::text AS org_id,
				COALESCE(
					NULLIF(
						LOWER(BTRIM(oa.education_domain)),
						''
					),
					''
				) AS education_domain
			FROM organization_admins oa
			JOIN organizations o
			  ON o.id = oa.org_id
			 AND o.type = 'region'
			 AND o.status = 'active'
			WHERE oa.user_id = $1
			  AND oa.role_type = 'region_admin'
			  AND ($2 = '' OR oa.org_id::text <> $2)

			UNION ALL

			SELECT
				o.id::text AS org_id,
				'' AS education_domain
			FROM organizations o
			WHERE o.admin_user_id = $1
			  AND o.type = 'region'
			  AND o.status = 'active'
			  AND ($2 = '' OR o.id::text <> $2)
			  AND NOT EXISTS (
				SELECT 1
				FROM organization_admins oa
				WHERE oa.org_id = o.id
				  AND oa.user_id = $1
				  AND oa.role_type = 'region_admin'
			  )
		)
		SELECT education_domain
		FROM effective_appointments
	`, userID, excludeOrgID)
	if err != nil {
		return nil, fmt.Errorf(
			"查询用户区域任命教育域状态失败: %w",
			err,
		)
	}
	defer rows.Close()

	domainSet := make(map[string]struct{})

	for rows.Next() {
		var domain string
		if err := rows.Scan(&domain); err != nil {
			return nil, fmt.Errorf(
				"扫描用户区域任命教育域状态失败: %w",
				err,
			)
		}

		domain = strings.ToLower(strings.TrimSpace(domain))
		if !models.IsTeachingEducationDomain(domain) {
			state.HasUnconfigured = true
			continue
		}

		domainSet[domain] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历用户区域任命教育域状态失败: %w",
			err,
		)
	}

	for domain := range domainSet {
		state.Domains = append(state.Domains, domain)
	}
	sort.Strings(state.Domains)

	return state, nil
}

package repository

// region_admin_education_context_repo.go
//
// 本文件专门解析区域管理员登录后的固定教育域。
//
// 为什么不能直接使用organizations.education_domain：
//   区域组织本身固定为mixed，表示该组织可以下辖不同类型的学校；
//   区域管理员个人负责的固定教育域保存在
//   organization_admins.education_domain中，两者语义完全不同。
//
// 解析规则：
//   1. 只读取active区域的region_admin任命；
//   2. 多个区域任命可以存在，但必须全部为同一具体教学教育域；
//   3. organizations.admin_user_id遗留单字段任命如果没有对应的
//      organization_admins记录，视为未配置教育域；
//   4. 无任命、空值、非法值、多域冲突和数据库错误全部fail-closed；
//   5. 本文件不做K12默认回退，也不调用NormalizeEducationDomain。
//
// 业务异常通过ErrRegionAdminEducationDomainNotReady返回。
// AuthService会允许身份认证完成，但向前端下发
// education_domain_ready=false和统一的配置异常提示。

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// ErrRegionAdminEducationDomainNotReady 表示区域管理员无法解析出唯一具体教育域。
//
// 该错误可以由以下情况触发：
//   - 没有任何有效区域任命；
//   - 任命教育域为空；
//   - 任命教育域不是k12、vocational或adult；
//   - 多个区域任命属于不同教育域。
var ErrRegionAdminEducationDomainNotReady = errors.New(
	"区域管理员教育域尚未正确配置",
)

// regionAdminEducationDomainCandidate 描述一条有效区域任命候选。
//
// JoinedAt用于数据库稳定排序；纯函数测试可以使用零值时间。
type regionAdminEducationDomainCandidate struct {
	OrganizationID   string
	OrganizationName string
	OrganizationLogo string
	EducationDomain  string
	JoinedAt         time.Time
}

// resolveRegionAdminEducationContextFromCandidates 将任命候选解析为唯一教育域。
//
// 本函数不访问数据库，便于稳定覆盖以下自动化测试：
//   - K12、职业教育、成人教育单域任命；
//   - 多个同域区域任命；
//   - 空值、非法值和多域冲突；
//   - 没有任何任命。
func resolveRegionAdminEducationContextFromCandidates(
	candidates []regionAdminEducationDomainCandidate,
) (*models.UserEducationContext, error) {
	if len(candidates) == 0 {
		return nil, fmt.Errorf(
			"%w: 没有有效区域任命",
			ErrRegionAdminEducationDomainNotReady,
		)
	}

	// 数据库已经按任命时间、组织名称和组织ID稳定排序，
	// 第一条只用于组织名称与Logo展示，不代表“取第一条教育域”。
	selectedBranding := candidates[0]
	domainSet := make(map[string]struct{})

	for _, candidate := range candidates {
		domain := strings.ToLower(
			strings.TrimSpace(candidate.EducationDomain),
		)

		// 空值、mixed、common及其它非法值全部拒绝，
		// 不允许通过NormalizeEducationDomain回退到K12。
		if !models.IsTeachingEducationDomain(domain) {
			return nil, fmt.Errorf(
				"%w: 组织%s的任命教育域为空或非法",
				ErrRegionAdminEducationDomainNotReady,
				candidate.OrganizationID,
			)
		}

		domainSet[domain] = struct{}{}
	}

	if len(domainSet) != 1 {
		return nil, fmt.Errorf(
			"%w: 多个区域任命属于不同教育域",
			ErrRegionAdminEducationDomainNotReady,
		)
	}

	resolvedDomain := ""
	for domain := range domainSet {
		resolvedDomain = domain
	}

	return &models.UserEducationContext{
		OrganizationID:   selectedBranding.OrganizationID,
		OrganizationName: selectedBranding.OrganizationName,
		OrganizationLogo: selectedBranding.OrganizationLogo,
		EducationDomain:  resolvedDomain,
		PortalModules:    models.DefaultPortalModules(),
		DomainConflict:   false,
	}, nil
}

// resolveRegionAdminEducationContext 从数据库解析区域管理员固定教育域。
//
// 数据来源同时覆盖：
//   1. organization_admins正式多管理员任命；
//   2. organizations.admin_user_id遗留主管理员单字段。
//
// 遗留单字段只有在同区域不存在对应正式任命时才补入候选，且其教育域为空，
// 因而会被纯函数明确判定为“尚未配置”，绝不会被猜测为K12。
func resolveRegionAdminEducationContext(
	ctx context.Context,
	userID string,
) (*models.UserEducationContext, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, fmt.Errorf(
			"%w: 用户ID为空",
			ErrRegionAdminEducationDomainNotReady,
		)
	}

	rows, err := database.DB.Query(ctx, `
		WITH effective_appointments AS (
			-- 正式区域管理员任命：教育域来自任命记录本身。
			SELECT
				o.id::text AS organization_id,
				o.name AS organization_name,
				COALESCE(
					NULLIF(o.logo_url, ''),
					NULLIF(parent.logo_url, ''),
					''
				) AS effective_logo,
				COALESCE(
					oa.education_domain,
					''
				) AS appointment_education_domain,
				COALESCE(
					oa.created_at,
					o.created_at,
					now()
				) AS linked_at
			FROM organization_admins oa
			JOIN organizations o
			  ON o.id = oa.org_id
			 AND o.type = 'region'
			 AND o.status = 'active'
			LEFT JOIN organizations parent
			  ON parent.id = o.parent_id
			WHERE oa.user_id = $1
			  AND oa.role_type = 'region_admin'

			UNION ALL

			-- 遗留主管理员单字段：没有正式任命记录时视为未配置。
			SELECT
				o.id::text,
				o.name,
				COALESCE(
					NULLIF(o.logo_url, ''),
					NULLIF(parent.logo_url, ''),
					''
				),
				'' AS appointment_education_domain,
				COALESCE(
					o.created_at,
					now()
				)
			FROM organizations o
			LEFT JOIN organizations parent
			  ON parent.id = o.parent_id
			WHERE o.admin_user_id = $1
			  AND o.type = 'region'
			  AND o.status = 'active'
			  AND NOT EXISTS (
				SELECT 1
				FROM organization_admins oa
				WHERE oa.org_id = o.id
				  AND oa.user_id = $1
				  AND oa.role_type = 'region_admin'
			  )
		)
		SELECT
			organization_id,
			organization_name,
			effective_logo,
			appointment_education_domain,
			linked_at
		FROM effective_appointments
		ORDER BY
			linked_at,
			organization_name,
			organization_id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf(
			"查询区域管理员固定教育域失败: %w",
			err,
		)
	}
	defer rows.Close()

	candidates := make(
		[]regionAdminEducationDomainCandidate,
		0,
	)

	for rows.Next() {
		var candidate regionAdminEducationDomainCandidate

		if err := rows.Scan(
			&candidate.OrganizationID,
			&candidate.OrganizationName,
			&candidate.OrganizationLogo,
			&candidate.EducationDomain,
			&candidate.JoinedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描区域管理员固定教育域失败: %w",
				err,
			)
		}

		candidates = append(candidates, candidate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历区域管理员固定教育域失败: %w",
			err,
		)
	}

	return resolveRegionAdminEducationContextFromCandidates(
		candidates,
	)
}

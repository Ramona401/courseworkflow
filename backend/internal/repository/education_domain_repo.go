package repository

// education_domain_repo.go — 用户教育域解析
//
// 普通用户教育域解析来源，按确定性优先级排序：
//   1. organizations.admin_user_id / organization_admins正式管理员任命；
//   2. school_members权威校籍；
//   3. teaching_group_members历史归属兜底。
//
// 区域管理员是独立规则：
//   - 区域组织本身的education_domain固定为mixed；
//   - 区域管理员个人固定域来自organization_admins.education_domain；
//   - 由region_admin_education_context_repo.go独立解析；
//   - 无任命、空值、非法值、多域冲突或查询失败全部fail-closed。
//
// 不使用users.school_id：
//   当前users表不存在该字段；平台正式学校归属已经由school_members承担，
//   并支持一个用户属于多所学校。新增单一school_id会造成两个互相冲突的真相源。
//
// 身份规则：
//   - admin、district_inspector固定为mixed跨域管理上下文；
//   - region_admin使用任命表中的唯一具体教育域；
//   - senior_operator优先采用正式学校管理员任命对应的学校；
//   - 普通教师忽略mixed教育局或管理组织，优先选择具体教学学校；
//   - 多个具体学校属于同一教育域时允许存在；
//   - 同时属于不同具体教育域时DomainConflict=true，并按确定性顺序选择首个。
//
// 分域课程目录查询已经拆至education_domain_subject_repo.go，
// 避免本文件超过项目600行红线。

import (
	"context"
	"fmt"
	"time"

	"tedna/internal/database"
	"tedna/internal/models"
)

// educationOrgCandidate 用户可能关联的一个组织候选。
//
// SourcePriority越小优先级越高；候选在数据库查询中已按
// SourcePriority、关联时间、组织名和组织ID稳定排序。
type educationOrgCandidate struct {
	ID              string
	Name            string
	Logo            string
	Settings        string
	EducationDomain string
	SourcePriority  int
	JoinedAt        time.Time
}

// isCrossDomainManagementRole 判断是否属于固定跨域管理身份。
//
// admin负责全平台管理，district_inspector负责跨学校抽查，二者固定为mixed。
// region_admin不在此列：其登录教育域必须从任命表解析为唯一具体教学域。
func isCrossDomainManagementRole(role string) bool {
	return role == models.RoleAdmin ||
		role == models.RoleDistrictInspector
}

// defaultEducationDomainForRole 返回没有任何普通组织候选时的兼容教育域。
//
// region_admin刻意返回空字符串，防止任何意外调用路径把无任命状态默认成K12。
// 正式入口会在ResolveUserEducationContext函数开头进入独立严格解析器。
func defaultEducationDomainForRole(role string) string {
	if isCrossDomainManagementRole(role) {
		return models.EducationDomainMixed
	}

	if role == models.RoleRegionAdmin {
		return ""
	}

	return models.EducationDomainK12
}

// defaultEducationContextForRole 构造无组织情况下的兼容上下文。
func defaultEducationContextForRole(
	role string,
) *models.UserEducationContext {
	domain := defaultEducationDomainForRole(role)

	return &models.UserEducationContext{
		EducationDomain: domain,
		PortalModules:   models.DefaultPortalModules(),
	}
}

// selectBrandingCandidate 为跨域管理身份选择稳定的组织名称和Logo来源。
//
// mixed组织优先用于管理身份品牌展示；没有mixed候选时使用第一条候选。
// 该选择不改变教育域，admin和district_inspector始终返回mixed。
func selectBrandingCandidate(
	candidates []educationOrgCandidate,
) *educationOrgCandidate {
	if len(candidates) == 0 {
		return nil
	}

	for i := range candidates {
		if models.NormalizeEducationDomain(
			candidates[i].EducationDomain,
		) == models.EducationDomainMixed {
			return &candidates[i]
		}
	}

	return &candidates[0]
}

// resolveEducationContextFromCandidates 将已排序候选解析为最终教育上下文。
//
// 本函数用于admin、district_inspector、学校管理员和普通教师。
// region_admin由独立严格解析器处理，不进入本函数的正式数据库调用路径。
func resolveEducationContextFromCandidates(
	role string,
	candidates []educationOrgCandidate,
) *models.UserEducationContext {
	if len(candidates) == 0 {
		return defaultEducationContextForRole(role)
	}

	// 平台管理员和区域教研员固定使用mixed上下文。
	if isCrossDomainManagementRole(role) {
		selected := selectBrandingCandidate(candidates)
		if selected == nil {
			return defaultEducationContextForRole(role)
		}

		return &models.UserEducationContext{
			OrganizationID:   selected.ID,
			OrganizationName: selected.Name,
			OrganizationLogo: selected.Logo,
			EducationDomain:  models.EducationDomainMixed,
			PortalModules:    models.DefaultPortalModules(),
		}
	}

	concreteDomains := make(map[string]struct{})

	var selectedConcrete *educationOrgCandidate
	var selectedMixed *educationOrgCandidate
	var selectedFallback *educationOrgCandidate

	for i := range candidates {
		item := &candidates[i]
		domain := models.NormalizeEducationDomain(
			item.EducationDomain,
		)

		if selectedFallback == nil {
			selectedFallback = item
		}

		if models.IsTeachingEducationDomain(domain) {
			concreteDomains[domain] = struct{}{}

			if selectedConcrete == nil {
				selectedConcrete = item
			}
			continue
		}

		if domain == models.EducationDomainMixed &&
			selectedMixed == nil {
			selectedMixed = item
		}
	}

	// 普通教学身份优先具体教学学校；
	// 只有没有具体学校时才使用mixed或兼容候选。
	selected := selectedConcrete
	if selected == nil {
		selected = selectedMixed
	}
	if selected == nil {
		selected = selectedFallback
	}
	if selected == nil {
		return defaultEducationContextForRole(role)
	}

	resolvedDomain := models.NormalizeEducationDomain(
		selected.EducationDomain,
	)

	return &models.UserEducationContext{
		OrganizationID:   selected.ID,
		OrganizationName: selected.Name,
		OrganizationLogo: selected.Logo,
		EducationDomain:  resolvedDomain,
		PortalModules: parsePortalModulesFromSettings(
			selected.Settings,
		),
		DomainConflict: len(concreteDomains) > 1,
	}
}

// ResolveUserEducationContext 解析当前用户确定性的教育域和教学组织。
func ResolveUserEducationContext(
	ctx context.Context,
	userID string,
	role string,
) (*models.UserEducationContext, error) {
	// 区域管理员必须优先进入专用解析器。
	// 不能读取区域组织自身的mixed域，也不能走后续K12兼容默认。
	if role == models.RoleRegionAdmin {
		return resolveRegionAdminEducationContext(
			ctx,
			userID,
		)
	}

	if userID == "" {
		return defaultEducationContextForRole(role), nil
	}

	rows, err := database.DB.Query(ctx, `
		-- 1A. organizations.admin_user_id：历史主管理员字段
		SELECT
			o.id::text AS organization_id,
			o.name AS organization_name,
			COALESCE(
				NULLIF(o.logo_url, ''),
				NULLIF(parent.logo_url, ''),
				''
			) AS effective_logo,
			COALESCE(
				o.settings::text,
				'{}'
			) AS settings_json,
			COALESCE(
				o.education_domain,
				''
			) AS education_domain,
			1 AS source_priority,
			COALESCE(
				o.created_at,
				now()
			) AS linked_at
		FROM organizations o
		LEFT JOIN organizations parent
		  ON parent.id = o.parent_id
		WHERE o.admin_user_id = $1
		  AND o.status = 'active'

		UNION ALL

		-- 1B. organization_admins：多管理员正式任命
		SELECT
			o.id::text,
			o.name,
			COALESCE(
				NULLIF(o.logo_url, ''),
				NULLIF(parent.logo_url, ''),
				''
			),
			COALESCE(
				o.settings::text,
				'{}'
			),
			COALESCE(
				o.education_domain,
				''
			),
			1,
			COALESCE(
				oa.created_at,
				o.created_at,
				now()
			)
		FROM organization_admins oa
		JOIN organizations o
		  ON o.id = oa.org_id
		LEFT JOIN organizations parent
		  ON parent.id = o.parent_id
		WHERE oa.user_id = $1
		  AND o.status = 'active'

		UNION ALL

		-- 2. school_members：学校直接校籍权威来源
		SELECT
			o.id::text,
			o.name,
			COALESCE(
				NULLIF(o.logo_url, ''),
				NULLIF(parent.logo_url, ''),
				''
			),
			COALESCE(
				o.settings::text,
				'{}'
			),
			COALESCE(
				o.education_domain,
				''
			),
			2,
			COALESCE(
				sm.joined_at,
				o.created_at,
				now()
			)
		FROM school_members sm
		JOIN organizations o
		  ON o.id = sm.school_id
		LEFT JOIN organizations parent
		  ON parent.id = o.parent_id
		WHERE sm.user_id = $1
		  AND o.status = 'active'

		UNION ALL

		-- 3. teaching_group_members：历史教研组归属兜底
		SELECT
			o.id::text,
			o.name,
			COALESCE(
				NULLIF(o.logo_url, ''),
				NULLIF(parent.logo_url, ''),
				''
			),
			COALESCE(
				o.settings::text,
				'{}'
			),
			COALESCE(
				o.education_domain,
				''
			),
			3,
			COALESCE(
				tgm.joined_at,
				tg.created_at,
				o.created_at,
				now()
			)
		FROM teaching_group_members tgm
		JOIN teaching_groups tg
		  ON tg.id = tgm.group_id
		JOIN organizations o
		  ON o.id = tg.school_id
		LEFT JOIN organizations parent
		  ON parent.id = o.parent_id
		WHERE tgm.user_id = $1
		  AND o.status = 'active'

		ORDER BY
			source_priority,
			linked_at,
			organization_name,
			organization_id
	`, userID)
	if err != nil {
		return nil, fmt.Errorf(
			"解析用户教育域失败: %w",
			err,
		)
	}
	defer rows.Close()

	candidates := make(
		[]educationOrgCandidate,
		0,
	)
	seenOrganizations := make(
		map[string]struct{},
	)

	for rows.Next() {
		var item educationOrgCandidate

		if err := rows.Scan(
			&item.ID,
			&item.Name,
			&item.Logo,
			&item.Settings,
			&item.EducationDomain,
			&item.SourcePriority,
			&item.JoinedAt,
		); err != nil {
			return nil, fmt.Errorf(
				"扫描用户教育域组织失败: %w",
				err,
			)
		}

		// 同一组织可能同时由管理员任命、校籍和教研组命中。
		// 查询顺序已保证更高优先级来源先出现，此处保留首条即可。
		if _, exists := seenOrganizations[item.ID]; exists {
			continue
		}

		seenOrganizations[item.ID] = struct{}{}
		candidates = append(candidates, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(
			"遍历用户教育域组织失败: %w",
			err,
		)
	}

	return resolveEducationContextFromCandidates(
		role,
		candidates,
	), nil
}
